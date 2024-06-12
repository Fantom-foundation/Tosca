package geth

import (
	"bytes"
	"errors"
	"fmt"
	"math"
	"math/big"
	"strings"

	ct "github.com/Fantom-foundation/Tosca/go/ct/common"
	"github.com/Fantom-foundation/Tosca/go/vm"
	"github.com/ethereum/go-ethereum/accounts/abi"
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

type processor struct {
	interpreterImplementation string
}

var _ vm.Processor = (*processor)(nil)

// TODO: remove and use a registry instead
func NewProcessor() vm.Processor {
	return NewProcessorWithVm("geth")
}

func NewProcessorWithVm(impl string) vm.Processor {
	return &processor{impl}
}

func (p *processor) Run(
	blockInfo vm.BlockInfo,
	transaction vm.Transaction,
	state vm.WorldState,
) (vm.Receipt, error) {

	// --- setup ---

	// Hashing function used in the context for BLOCKHASH instruction
	getHash := func(num uint64) common.Hash {
		return common.Hash(state.GetBlockHash(int64(num)))
	}

	// Intercept the transfer function to conduct the transfer on the actual state.
	transferFunc := func(_ geth.StateDB, from common.Address, to common.Address, amount *big.Int) {
		if amount.Sign() != 1 || from == to {
			return
		}
		a := vm.Address(from)
		b := vm.Address(to)
		//d := vm.Uint256ToValue(amount)
		var tmp vm.Value
		amount.FillBytes(tmp[:])
		d := tmp
		curA := state.GetBalance(a)
		curB := state.GetBalance(b)
		state.SetBalance(a, curA.Sub(d))
		state.SetBalance(b, curB.Add(d))
	}

	// Create empty block context based on block number
	// TODO: this is a copy of geth.go; try to refactor this
	blockCtx := geth.BlockContext{
		BlockNumber: big.NewInt(int64(blockInfo.BlockNumber)),
		Time:        big.NewInt(int64(blockInfo.Timestamp)),
		Difficulty:  big.NewInt(1), // < TODO: check this
		GasLimit:    uint64(blockInfo.GasLimit),
		GetHash:     getHash,
		BaseFee:     new(big.Int).SetBytes(blockInfo.BaseFee[:]),
		Transfer:    transferFunc,
		CanTransfer: canTransferFunc,
	}

	// Create empty tx context
	txCtx := geth.TxContext{
		Origin:   common.Address(transaction.Sender),
		GasPrice: new(big.Int).SetBytes(blockInfo.GasPrice[:]),
	}

	// Create a configuration for the geth EVM.
	config := geth.Config{
		InterpreterImpl: p.interpreterImplementation,
		StatePrecompiles: map[common.Address]geth.PrecompiledStateContract{
			stateContractAddress: PreCompiledContract{},
		},
	}

	// Set hard forks for chainconfig
	chainConfig :=
		makeChainConfig(*params.AllEthashProtocolChanges,
			new(big.Int).SetBytes(blockInfo.ChainID[:]),
			vmRevisionToCt(blockInfo.Revision))

	// Fix block boundaries to match required revisions
	chainConfig.IstanbulBlock = big.NewInt(0)
	chainConfig.BerlinBlock = big.NewInt(0)
	chainConfig.LondonBlock = big.NewInt(0)

	if blockInfo.Revision < vm.R10_London {
		chainConfig.LondonBlock = big.NewInt(blockInfo.BlockNumber + 1)
	}
	if blockInfo.Revision < vm.R09_Berlin {
		chainConfig.BerlinBlock = big.NewInt(blockInfo.BlockNumber + 1)
	}
	if blockInfo.Revision < vm.R07_Istanbul {
		chainConfig.IstanbulBlock = big.NewInt(blockInfo.BlockNumber + 1)
	}

	stateDb := &stateDbAdapter{context: state}
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
	if err := preCheck(transaction, state); err != nil {
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
	if blockInfo.Revision >= vm.R09_Berlin {
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
		output, created, gasLeft, vmError = evm.Create(sender, transaction.Input, uint64(gas), transaction.Value.ToBig())
		createdContract = &vm.Address{}
		*createdContract = vm.Address(created)
	} else {
		// Increment the nonce to avoid double execution
		stateDb.SetNonce(common.Address(transaction.Sender), stateDb.GetNonce(common.Address(transaction.Sender))+1)
		output, gasLeft, vmError = evm.Call(sender, common.Address(*transaction.Recipient), transaction.Input, uint64(gas), transaction.Value.ToBig())
	}

	// For whatever reason, 10% of remaining gas is charged for non-internal transactions.
	if !isInternal(transaction) {
		gasLeft = gasLeft - gasLeft/10
	}

	// Add refund to the remaining gas.
	if vmError == nil {
		refund := stateDb.GetRefund()

		maxRefund := uint64(0)
		gasUsed := uint64(transaction.GasLimit) - gasLeft
		if blockInfo.Revision < vm.R10_London {
			// Before EIP-3529: refunds were capped to gasUsed / 2
			maxRefund = gasUsed / 2
		} else {
			// After EIP-3529: refunds are capped to gasUsed / 5
			maxRefund = gasUsed / 5
		}

		if refund > maxRefund {
			refund = maxRefund
		}

		gasLeft += refund
	}

	// Extract log messages.
	logs := make([]vm.Log, 0)
	for _, log := range stateDb.GetLogs() {
		topics := make([]vm.Hash, len(log.Topics))
		for i, topic := range log.Topics {
			topics[i] = vm.Hash(topic)
		}
		logs = append(logs, vm.Log{
			Address: vm.Address(log.Address),
			Topics:  topics,
			Data:    log.Data,
		})
	}

	return vm.Receipt{
		Success:         vmError == nil,
		GasUsed:         transaction.GasLimit - vm.Gas(gasLeft),
		ContractAddress: createdContract,
		Output:          output,
		Logs:            logs,
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

func preCheck(transaction vm.Transaction, state vm.WorldState) error {
	// Only check transactions that are not fake
	// TODO: add support for non-checked transactions

	// Make sure this transaction's nonce is correct.
	stNonce := state.GetNonce(transaction.Sender)
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
	if codeHash := state.GetCodeHash(transaction.Sender); codeHash != emptyCodeHash && codeHash != (vm.Hash{}) {
		return fmt.Errorf("%w: address %v, codehash: %s", ErrSenderNoEOA,
			transaction.Sender, codeHash)
	}

	// Note: Opera doesn't need to check gasFeeCap >= BaseFee, because it's already checked by epochcheck
	return buyGas(transaction, state)
}

func buyGas(tx vm.Transaction, state vm.WorldState) error {
	// TODO: support arithmetic operations with Value type
	gasPrice := state.GetTransactionContext().GasPrice.ToU256()
	mgval := uint256.NewInt(uint64(tx.GasLimit))
	mgval = mgval.Mul(mgval, gasPrice)
	// Note: Opera doesn't need to check against gasFeeCap instead of gasPrice, as it's too aggressive in the asynchronous environment
	balance := state.GetBalance(tx.Sender)
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
	state.SetBalance(tx.Sender, balance)
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

// makeChainConfig returns a chain config for the given chain ID and target revision.
// The baseline config is used as a starting point, so that any prefilled configuration from go-ethereum:params/config.go can be used.
// chainId needs to be prefilled as it may be accessed with the opcode CHAINID.
// the fork-blocks and the fork-times are set to the respective values for the given revision.
func makeChainConfig(baseline params.ChainConfig, chainId *big.Int, targetRevision ct.Revision) params.ChainConfig {
	istanbulBlock, err := ct.GetForkBlock(ct.R07_Istanbul)
	if err != nil {
		panic(fmt.Sprintf("Failed to get Istanbul fork block: %v", err))
	}
	berlinBlock, err := ct.GetForkBlock(ct.R09_Berlin)
	if err != nil {
		panic(fmt.Sprintf("Failed to get Berlin fork block: %v", err))
	}
	londonBlock, err := ct.GetForkBlock(ct.R10_London)
	if err != nil {
		panic(fmt.Sprintf("Failed to get London fork block: %v", err))
	}
	/*
		parisBlock, err := ct.GetForkBlock(ct.R11_Paris)
		if err != nil {
			panic(fmt.Sprintf("Failed to get Paris fork block: %v", err))
		}
		shanghaiTime := ct.GetForkTime(ct.R12_Shanghai)
		cancunTime := ct.GetForkTime(ct.R13_Cancun)
	*/

	chainConfig := baseline
	chainConfig.ChainID = chainId
	chainConfig.ByzantiumBlock = big.NewInt(0)
	chainConfig.IstanbulBlock = big.NewInt(0).SetUint64(istanbulBlock)
	chainConfig.BerlinBlock = big.NewInt(0).SetUint64(berlinBlock)
	chainConfig.LondonBlock = big.NewInt(0).SetUint64(londonBlock)

	/*
		if targetRevision >= ct.R11_Paris {
			chainConfig.MergeNetsplitBlock = big.NewInt(0).SetUint64(parisBlock)
		}
		if targetRevision >= ct.R12_Shanghai {
			chainConfig.ShanghaiTime = &shanghaiTime
		}
		if targetRevision >= ct.R13_Cancun {
			chainConfig.CancunTime = &cancunTime
		}
	*/

	return chainConfig
}

// TODO: remove once there is only one Revision definition
func vmRevisionToCt(revision vm.Revision) ct.Revision {
	switch revision {
	case vm.R07_Istanbul:
		return ct.R07_Istanbul
	case vm.R09_Berlin:
		return ct.R09_Berlin
	case vm.R10_London:
		return ct.R10_London
		/*
			case vm.R11_Paris:
				return ct.R11_Paris
			case vm.R12_Shanghai:
				return ct.R12_Shanghai
			case vm.R13_Cancun:
				return ct.R13_Cancun
		*/
	}
	panic(fmt.Sprintf("Unknown revision: %v", revision))
}

// Source: https://github.com/Fantom-foundation/Sonic/blob/main/opera/contracts/evmwriter/evm_writer.go#L24

// driverAddress is the NodeDriver contract address
var driverAddress = common.HexToAddress("0xd100a01e00000000000000000000000000000000")

// stateContractAddress is the EvmWriter pre-compiled contract address
var stateContractAddress = common.HexToAddress("0xd100ec0000000000000000000000000000000000")

// stateContractABI is the input ABI used to generate the binding from
var stateContractABI string = "[{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"num\",\"type\":\"uint256\"}],\"name\":\"AdvanceEpochs\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"bytes\",\"name\":\"diff\",\"type\":\"bytes\"}],\"name\":\"UpdateNetworkRules\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"version\",\"type\":\"uint256\"}],\"name\":\"UpdateNetworkVersion\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"uint256\",\"name\":\"validatorID\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"bytes\",\"name\":\"pubkey\",\"type\":\"bytes\"}],\"name\":\"UpdateValidatorPubkey\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"uint256\",\"name\":\"validatorID\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"weight\",\"type\":\"uint256\"}],\"name\":\"UpdateValidatorWeight\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"backend\",\"type\":\"address\"}],\"name\":\"UpdatedBackend\",\"type\":\"event\"},{\"constant\":false,\"inputs\":[{\"internalType\":\"address\",\"name\":\"_backend\",\"type\":\"address\"}],\"name\":\"setBackend\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"internalType\":\"address\",\"name\":\"_backend\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"_evmWriterAddress\",\"type\":\"address\"}],\"name\":\"initialize\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"internalType\":\"address\",\"name\":\"acc\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"value\",\"type\":\"uint256\"}],\"name\":\"setBalance\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"internalType\":\"address\",\"name\":\"acc\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"from\",\"type\":\"address\"}],\"name\":\"copyCode\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"internalType\":\"address\",\"name\":\"acc\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"with\",\"type\":\"address\"}],\"name\":\"swapCode\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"internalType\":\"address\",\"name\":\"acc\",\"type\":\"address\"},{\"internalType\":\"bytes32\",\"name\":\"key\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"value\",\"type\":\"bytes32\"}],\"name\":\"setStorage\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"internalType\":\"address\",\"name\":\"acc\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"diff\",\"type\":\"uint256\"}],\"name\":\"incNonce\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"internalType\":\"bytes\",\"name\":\"diff\",\"type\":\"bytes\"}],\"name\":\"updateNetworkRules\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"version\",\"type\":\"uint256\"}],\"name\":\"updateNetworkVersion\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"num\",\"type\":\"uint256\"}],\"name\":\"advanceEpochs\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"validatorID\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"value\",\"type\":\"uint256\"}],\"name\":\"updateValidatorWeight\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"validatorID\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"pubkey\",\"type\":\"bytes\"}],\"name\":\"updateValidatorPubkey\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"internalType\":\"address\",\"name\":\"_auth\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"validatorID\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"pubkey\",\"type\":\"bytes\"},{\"internalType\":\"uint256\",\"name\":\"status\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"createdEpoch\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"createdTime\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"deactivatedEpoch\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"deactivatedTime\",\"type\":\"uint256\"}],\"name\":\"setGenesisValidator\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"internalType\":\"address\",\"name\":\"delegator\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"toValidatorID\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"stake\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"lockedStake\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"lockupFromEpoch\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"lockupEndTime\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"lockupDuration\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"earlyUnlockPenalty\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"rewards\",\"type\":\"uint256\"}],\"name\":\"setGenesisDelegation\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"validatorID\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"status\",\"type\":\"uint256\"}],\"name\":\"deactivateValidator\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"internalType\":\"uint256[]\",\"name\":\"nextValidatorIDs\",\"type\":\"uint256[]\"}],\"name\":\"sealEpochValidators\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"internalType\":\"uint256[]\",\"name\":\"offlineTimes\",\"type\":\"uint256[]\"},{\"internalType\":\"uint256[]\",\"name\":\"offlineBlocks\",\"type\":\"uint256[]\"},{\"internalType\":\"uint256[]\",\"name\":\"uptimes\",\"type\":\"uint256[]\"},{\"internalType\":\"uint256[]\",\"name\":\"originatedTxsFee\",\"type\":\"uint256[]\"}],\"name\":\"sealEpoch\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"internalType\":\"uint256[]\",\"name\":\"offlineTimes\",\"type\":\"uint256[]\"},{\"internalType\":\"uint256[]\",\"name\":\"offlineBlocks\",\"type\":\"uint256[]\"},{\"internalType\":\"uint256[]\",\"name\":\"uptimes\",\"type\":\"uint256[]\"},{\"internalType\":\"uint256[]\",\"name\":\"originatedTxsFee\",\"type\":\"uint256[]\"},{\"internalType\":\"uint256\",\"name\":\"usedGas\",\"type\":\"uint256\"}],\"name\":\"sealEpochV1\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"}]"

var (
	setBalanceMethodID []byte
	copyCodeMethodID   []byte
	swapCodeMethodID   []byte
	setStorageMethodID []byte
	incNonceMethodID   []byte
)

func init() {
	abi, err := abi.JSON(strings.NewReader(stateContractABI))
	if err != nil {
		panic(err)
	}

	for name, constID := range map[string]*[]byte{
		"setBalance": &setBalanceMethodID,
		"copyCode":   &copyCodeMethodID,
		"swapCode":   &swapCodeMethodID,
		"setStorage": &setStorageMethodID,
		"incNonce":   &incNonceMethodID,
	} {
		method, exist := abi.Methods[name]
		if !exist {
			panic("unknown EvmWriter method")
		}

		*constID = make([]byte, len(method.ID))
		copy(*constID, method.ID)
	}
}

type PreCompiledContract struct{}

func (_ PreCompiledContract) Run(stateDB geth.StateDB, _ geth.BlockContext, txCtx geth.TxContext, caller common.Address, input []byte, suppliedGas uint64) ([]byte, uint64, error) {
	if caller != driverAddress {
		return nil, 0, geth.ErrExecutionReverted
	}
	if len(input) < 4 {
		return nil, 0, geth.ErrExecutionReverted
	}
	if bytes.Equal(input[:4], setBalanceMethodID) {
		input = input[4:]
		// setBalance
		if suppliedGas < params.CallValueTransferGas {
			return nil, 0, geth.ErrOutOfGas
		}
		suppliedGas -= params.CallValueTransferGas
		if len(input) != 64 {
			return nil, 0, geth.ErrExecutionReverted
		}

		acc := common.BytesToAddress(input[12:32])
		input = input[32:]
		value := new(big.Int).SetBytes(input[:32])

		if acc == txCtx.Origin {
			// Origin balance shouldn't decrease during his transaction
			return nil, 0, geth.ErrExecutionReverted
		}

		balance := stateDB.GetBalance(acc)
		if balance.Cmp(value) >= 0 {
			diff := new(big.Int).Sub(balance, value)
			stateDB.SubBalance(acc, diff /*, tracing.BalanceChangeUnspecified*/)
		} else {
			diff := new(big.Int).Sub(value, balance)
			stateDB.AddBalance(acc, diff /*, tracing.BalanceChangeUnspecified*/)
		}
	} else if bytes.Equal(input[:4], copyCodeMethodID) {
		input = input[4:]
		// copyCode
		if suppliedGas < params.CreateGas {
			return nil, 0, geth.ErrOutOfGas
		}
		suppliedGas -= params.CreateGas
		if len(input) != 64 {
			return nil, 0, geth.ErrExecutionReverted
		}

		accTo := common.BytesToAddress(input[12:32])
		input = input[32:]
		accFrom := common.BytesToAddress(input[12:32])

		code := stateDB.GetCode(accFrom)
		if code == nil {
			code = []byte{}
		}
		cost := uint64(len(code)) * (params.CreateDataGas + params.MemoryGas)
		if suppliedGas < cost {
			return nil, 0, geth.ErrOutOfGas
		}
		suppliedGas -= cost
		if accTo != accFrom {
			stateDB.SetCode(accTo, code)
		}
	} else if bytes.Equal(input[:4], swapCodeMethodID) {
		input = input[4:]
		// swapCode
		cost := 2 * params.CreateGas
		if suppliedGas < cost {
			return nil, 0, geth.ErrOutOfGas
		}
		suppliedGas -= cost
		if len(input) != 64 {
			return nil, 0, geth.ErrExecutionReverted
		}

		acc0 := common.BytesToAddress(input[12:32])
		input = input[32:]
		acc1 := common.BytesToAddress(input[12:32])
		code0 := stateDB.GetCode(acc0)
		if code0 == nil {
			code0 = []byte{}
		}
		code1 := stateDB.GetCode(acc1)
		if code1 == nil {
			code1 = []byte{}
		}
		cost0 := uint64(len(code0)) * (params.CreateDataGas + params.MemoryGas)
		cost1 := uint64(len(code1)) * (params.CreateDataGas + params.MemoryGas)
		cost = (cost0 + cost1) / 2 // 50% discount because trie size won't increase after pruning
		if suppliedGas < cost {
			return nil, 0, geth.ErrOutOfGas
		}
		suppliedGas -= cost
		if acc0 != acc1 {
			stateDB.SetCode(acc0, code1)
			stateDB.SetCode(acc1, code0)
		}
	} else if bytes.Equal(input[:4], setStorageMethodID) {
		input = input[4:]
		// setStorage
		if suppliedGas < params.SstoreSetGasEIP2200 {
			return nil, 0, geth.ErrOutOfGas
		}
		suppliedGas -= params.SstoreSetGasEIP2200
		if len(input) != 96 {
			return nil, 0, geth.ErrExecutionReverted
		}
		acc := common.BytesToAddress(input[12:32])
		input = input[32:]
		key := common.BytesToHash(input[:32])
		input = input[32:]
		value := common.BytesToHash(input[:32])

		stateDB.SetState(acc, key, value)
	} else if bytes.Equal(input[:4], incNonceMethodID) {
		input = input[4:]
		// incNonce
		if suppliedGas < params.CallValueTransferGas {
			return nil, 0, geth.ErrOutOfGas
		}
		suppliedGas -= params.CallValueTransferGas
		if len(input) != 64 {
			return nil, 0, geth.ErrExecutionReverted
		}

		acc := common.BytesToAddress(input[12:32])
		input = input[32:]
		value := new(big.Int).SetBytes(input[:32])

		if acc == txCtx.Origin {
			// Origin nonce shouldn't change during his transaction
			return nil, 0, geth.ErrExecutionReverted
		}

		if value.Cmp(common.Big256) >= 0 {
			// Don't allow large nonce increasing to prevent a nonce overflow
			return nil, 0, geth.ErrExecutionReverted
		}
		if value.Sign() <= 0 {
			return nil, 0, geth.ErrExecutionReverted
		}

		stateDB.SetNonce(acc, stateDB.GetNonce(acc)+value.Uint64())
	} else {
		return nil, 0, geth.ErrExecutionReverted
	}
	return nil, suppliedGas, nil
}
