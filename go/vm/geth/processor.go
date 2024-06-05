package geth

import (
	"errors"
	"fmt"
	"math"
	"math/big"

	"github.com/Fantom-foundation/Tosca/go/vm"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	geth "github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/params"
	"github.com/holiman/uint256"
	"golang.org/x/crypto/sha3"
)

var (
	// ErrNonceTooLow is returned if the nonce of a transaction is lower than the
	// one present in the local chain.
	ErrNonceTooLow = errors.New("nonce too low")

	// ErrNonceTooHigh is returned if the nonce of a transaction is higher than the
	// next one expected based on the local chain.
	ErrNonceTooHigh = errors.New("nonce too high")

	// ErrInsufficientFunds is returned if the total cost of executing a transaction
	// is higher than the balance of the user's account.
	ErrInsufficientFunds = errors.New("insufficient funds for gas * price + value")

	// ErrGasLimitReached
	// ErrGasUintOverflow is returned when calculating gas usage.
	ErrGasUintOverflow = errors.New("gas uint64 overflow")

	// ErrIntrinsicGas is returned if the transaction is specified to use less gas
	// than required to start the invocation.
	ErrIntrinsicGas = errors.New("intrinsic gas too low")

	// ErrSenderNoEOA is returned if the sender of a transaction is a contract.
	ErrSenderNoEOA = errors.New("sender not an eoa")
)

type processor struct{}

var _ vm.Processor = (*processor)(nil)

// TODO: remove and use a registry instead
func NewProcessor() vm.Processor {
	return &processor{}
}

func (*processor) Run(
	revision vm.Revision,
	transaction vm.Transaction,
	blockContext vm.TransactionContext,
	transactionContext vm.TxContext,
) (vm.Receipt, error) {

	// --- setup ---

	// Hashing function used in the context for BLOCKHASH instruction
	getHash := func(num uint64) common.Hash {
		return common.Hash(transactionContext.GetBlockHash(int64(num)))
	}

	// Intercept the transfer function to conduct the transfer on the actual state.
	transferFunc := func(_ geth.StateDB, from common.Address, to common.Address, amount *uint256.Int) {
		a := vm.Address(from)
		b := vm.Address(to)
		d := vm.Uint256ToValue(amount)
		curA := transactionContext.GetBalance(a)
		curB := transactionContext.GetBalance(b)
		transactionContext.SetBalance(a, curA.Sub(d))
		transactionContext.SetBalance(b, curB.Add(d))
	}

	// Create empty block context based on block number
	// TODO: this is a copy of geth.go; try to refactor this
	blockCtx := geth.BlockContext{
		BlockNumber: big.NewInt(int64(blockContext.BlockNumber)),
		Time:        uint64(blockContext.Timestamp),
		Difficulty:  big.NewInt(1), // < TODO: check this
		GasLimit:    uint64(blockContext.GasLimit),
		GetHash:     getHash,
		BaseFee:     new(big.Int).SetBytes(blockContext.BaseFee[:]),
		Transfer:    transferFunc,
		CanTransfer: canTransferFunc,
	}

	// Create empty tx context
	txCtx := geth.TxContext{
		GasPrice: new(big.Int).SetBytes(blockContext.GasPrice[:]),
	}
	// Set interpreter variant for this VM
	config := geth.Config{}

	// Set hard forks for chainconfig
	chainConfig :=
		makeChainConfig(*params.AllEthashProtocolChanges,
			new(big.Int).SetBytes(blockContext.ChainID[:]),
			vmRevisionToCt(revision))

	stateDb := &stateDbAdapter{context: transactionContext}
	evm := geth.NewEVM(blockCtx, txCtx, stateDb, &chainConfig, config)

	// -- start of execution --

	//snapshot := stateDb.Snapshot()

	// This function is required to mimic the behavior of Sonic's
	// evmcore transaction handling function. For reference, see:
	// https://github.com/Fantom-foundation/Sonic/blob/1819a05c9dc1081d24a71f93ec140eb674618967/evmcore/state_transition.go#L255

	// First check this message satisfies all consensus rules before
	// applying the message. The rules include these clauses
	//
	// 1. the nonce of the message caller is correct
	// 2. caller has enough balance to cover transaction fee(gaslimit * gasprice)
	// 3. the amount of gas required is available in the block
	// 4. the purchased gas is enough to cover intrinsic usage
	// 5. there is no overflow when calculating intrinsic gas

	// Note: insufficient balance for **topmost** call isn't a consensus error in Opera, unlike Ethereum
	// Such transaction will revert and consume sender's gas

	gas := transaction.GasLimit

	// Check clauses 1-3, buy gas if everything is correct
	if err := preCheck(transaction, transactionContext); err != nil {
		return vm.Receipt{}, err
	}
	// Check clauses 4-5, subtract intrinsic gas if everything is correct
	intrinsicGasCosts, err := IntrinsicGas(transaction)
	if err != nil {
		return vm.Receipt{}, err
	}
	if gas < intrinsicGasCosts {
		return vm.Receipt{}, fmt.Errorf("%w: have %d, want %d", ErrIntrinsicGas, transaction.GasLimit, intrinsicGasCosts)
	}
	gas -= intrinsicGasCosts

	sender := geth.AccountRef(transaction.Sender)
	contractCreation := transaction.Recipient == nil

	// Set up the initial access list.
	if revision >= vm.R09_Berlin {
		var dest *common.Address
		if transaction.Recipient != nil {
			dest = &common.Address{}
			*dest = common.Address(*transaction.Recipient)
		}

		precompiledContracts := []common.Address{} // TODO: list precompiled contracts

		var accessList types.AccessList
		for _, tuple := range transaction.AccessList {
			keys := make([]common.Hash, len(tuple.Keys))
			for i, key := range tuple.Keys {
				keys[i] = common.Hash(key)
			}
			accessList = append(accessList, types.AccessTuple{
				Address:     common.Address(tuple.Address),
				StorageKeys: keys,
			})
		}

		stateDb.PrepareAccessList(
			common.Address(transaction.Sender),
			dest,
			precompiledContracts,
			accessList,
		)
	}

	var (
		gasLeft         uint64
		output          []byte
		vmError         error
		createdContract *vm.Address
	)
	if contractCreation {
		var created common.Address
		output, created, gasLeft, vmError = evm.Create(sender, transaction.Input, uint64(transaction.GasLimit), transaction.Value.ToU256())
		createdContract = &vm.Address{}
		*createdContract = vm.Address(created)
	} else {
		// Increment the nonce to avoid double execution
		stateDb.SetNonce(common.Address(transaction.Sender), stateDb.GetNonce(common.Address(transaction.Sender))+1)
		output, gasLeft, vmError = evm.Call(sender, common.Address(*transaction.Recipient), transaction.Input, uint64(transaction.GasLimit), transaction.Value.ToU256())
	}

	// For whatever reason, 10% of remaining gas is charged for non-internal transactions.
	if !isInternal(transaction) {
		gasLeft = gasLeft - gasLeft/10
	}

	// TODO: handle refund gas

	return vm.Receipt{
		Success:         vmError == nil,
		ContractAddress: createdContract,
		Output:          output,
	}, nil

	//evm.Call()
	/*
		// prepare tx
		gasPool.AddGas(inputEnv.GetGasLimit())

		db.Prepare(txHash, tx)
		blockCtx := prepareBlockCtx(inputEnv, &hashError)
		txCtx := evmcore.NewEVMTxContext(msg)
		evm := vm.NewEVM(*blockCtx, txCtx, db, s.chainCfg, s.vmCfg)
		snapshot := db.Snapshot()

		// apply
		msgResult, err := evmcore.ApplyMessage(evm, msg, gasPool)
		if err != nil {
			// if transaction fails, revert to the first snapshot.
			db.RevertToSnapshot(snapshot)
			finalError = errors.Join(fmt.Errorf("block: %v transaction: %v", block, tx), err)
		}

		// inform about failing transaction
		if msgResult != nil && msgResult.Failed() {
			s.log.Debugf("Block: %v\nTransaction %v\n Status: Failed", block, tx)
		}

		// check whether getHash func produced an error
		if hashError != nil {
			finalError = errors.Join(finalError, hashError)
		}

		// if no prior error, create result and pass it to the data.
		blockHash := common.HexToHash(fmt.Sprintf("0x%016d", block))
		res = newTransactionResult(db.GetLogs(txHash, blockHash), msg, msgResult, err, evm.TxContext.Origin)
		return
	*/

	panic("not implemented")
}

var emptyCodeHash = keccak(nil)

func keccak(data []byte) vm.Hash {
	res := vm.Hash{}
	hasher := sha3.NewLegacyKeccak256()
	hasher.Write(data)
	hasher.Sum(res[0:0])
	return res
}

func preCheck(transaction vm.Transaction, context vm.TxContext) error {
	// Only check transactions that are not fake
	// TODO: add support for non-checked transactions

	// Make sure this transaction's nonce is correct.
	stNonce := context.GetNonce(transaction.Sender)
	if msgNonce := transaction.Nonce; stNonce < msgNonce {
		//skippedTxsNonceTooHighMeter.Mark(1)
		return fmt.Errorf("%w: address %v, tx: %d state: %d", ErrNonceTooHigh,
			transaction.Sender, msgNonce, stNonce)
	} else if stNonce > msgNonce {
		//skippedTxsNonceTooLowMeter.Mark(1)
		return fmt.Errorf("%w: address %v, tx: %d state: %d", ErrNonceTooLow,
			transaction.Sender, msgNonce, stNonce)
	}
	// Make sure the sender is an EOA (Externally Owned Account)
	if codeHash := context.GetCodeHash(transaction.Sender); codeHash != emptyCodeHash && codeHash != (vm.Hash{}) {
		return fmt.Errorf("%w: address %v, codehash: %s", ErrSenderNoEOA,
			transaction.Sender, codeHash)
	}

	// Note: Opera doesn't need to check gasFeeCap >= BaseFee, because it's already checked by epochcheck
	return buyGas(transaction, context)
}

func buyGas(tx vm.Transaction, context vm.TxContext) error {
	// TODO: support arithmetic operations with Value type
	gasPrice := context.GetTransactionContext().GasPrice.ToU256()
	mgval := uint256.NewInt(uint64(tx.GasLimit))
	mgval = mgval.Mul(mgval, gasPrice)
	// Note: Opera doesn't need to check against gasFeeCap instead of gasPrice, as it's too aggressive in the asynchronous environment
	balance := context.GetBalance(tx.Sender)
	if have, want := balance.ToU256(), mgval; have.Cmp(want) < 0 {
		//skippedTxsNoBalanceMeter.Mark(1)
		return fmt.Errorf("%w: address %v have %v want %v", ErrInsufficientFunds, tx.Sender, have, want)
	}
	// TODO: track block-wide gas usage
	/*
		if err := st.gp.SubGas(st.msg.Gas()); err != nil {
			return err
		}
	*/
	/*
		st.gas += st.msg.Gas()

		st.initialGas = st.msg.Gas()
	*/
	balance = balance.Sub(vm.Uint256ToValue(mgval))
	context.SetBalance(tx.Sender, balance)
	return nil
}

// IntrinsicGas computes the 'intrinsic gas' for a message with the given data.
func IntrinsicGas(transaction vm.Transaction) (vm.Gas, error) {
	// Set the starting gas for the raw transaction
	var gas uint64
	if transaction.Recipient == nil {
		gas = params.TxGasContractCreation
	} else {
		gas = params.TxGas
	}
	// Bump the required gas by the amount of transactional data
	if len(transaction.Input) > 0 {
		// Zero and non-zero bytes are priced differently
		var nz uint64
		for _, byt := range transaction.Input {
			if byt != 0 {
				nz++
			}
		}
		// Make sure we don't exceed uint64 for all data combinations
		if (math.MaxUint64-gas)/params.TxDataNonZeroGasEIP2028 < nz {
			return 0, geth.ErrOutOfGas
		}
		gas += nz * params.TxDataNonZeroGasEIP2028

		z := uint64(len(transaction.Input)) - nz
		if (math.MaxUint64-gas)/params.TxDataZeroGas < z {
			return 0, ErrGasUintOverflow
		}
		gas += z * params.TxDataZeroGas
	}
	accessList := transaction.AccessList
	if accessList != nil {
		gas += uint64(len(accessList)) * params.TxAccessListAddressGas
		for _, tuple := range accessList {
			gas += uint64(len(tuple.Keys)) * params.TxAccessListStorageKeyGas
		}
	}
	return vm.Gas(gas), nil
}

func isInternal(transaction vm.Transaction) bool {
	return transaction.Sender == vm.Address{}
}
