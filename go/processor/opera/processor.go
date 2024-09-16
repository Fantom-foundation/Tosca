// Copyright (c) 2024 Fantom Foundation
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at fantom.foundation/bsl11.
//
// Change Date: 2028-4-16
//
// On the date above, in accordance with the Business Source License, use of
// this software will be governed by the GNU Lesser General Public License v3.

package geth

// The processor implementation in this file is a rough copy of the processor
// code that is used by Aida to run transactions using the Opera/Sonic
// implementation of the Ethereum Virtual Machine (EVM). The code is copied
// here to provide a reference implementation for the Tosca EVM implementation.

import (
	"bytes"
	"errors"
	"fmt"
	"math"
	"math/big"
	"strings"

	"github.com/Fantom-foundation/Tosca/go/geth_adapter"
	geth_interpreter "github.com/Fantom-foundation/Tosca/go/interpreter/geth"
	"github.com/Fantom-foundation/Tosca/go/tosca"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/types"
	geth "github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/params"
	"github.com/holiman/uint256"
	"golang.org/x/crypto/sha3"
)

func init() {
	tosca.RegisterProcessorFactory("geth", newProcessor)
	tosca.RegisterProcessorFactory("opera", newProcessor)
}

// newProcessor is a factory function for the geth/opera processor implemented in this file.
// By including this package, it gets registered in the global processor registry.
func newProcessor(interpreter tosca.Interpreter) tosca.Processor {
	return &processor{
		interpreter: geth_adapter.NewGethInterpreterFactory(interpreter),
	}
}

var (
	// errNonceTooLow is returned if the nonce of a transaction is lower than the
	// one present in the local chain.
	errNonceTooLow = errors.New("nonce too low")

	// errNonceTooHigh is returned if the nonce of a transaction is higher than the
	// next one expected based on the local chain.
	errNonceTooHigh = errors.New("nonce too high")

	// errInsufficientFunds is returned if the total cost of executing a transaction
	// is higher than the balance of the user's account.
	errInsufficientFunds = errors.New("insufficient funds for gas * price + value")

	// ErrGasLimitReached
	// errGasUintOverflow is returned when calculating gas usage.
	errGasUintOverflow = errors.New("gas uint64 overflow")

	// errIntrinsicGas is returned if the transaction is specified to use less gas
	// than required to start the invocation.
	errIntrinsicGas = errors.New("intrinsic gas too low")

	// errSenderNoEOA is returned if the sender of a transaction is a contract.
	errSenderNoEOA = errors.New("sender not an eoa")
)

type processor struct {
	interpreter geth.InterpreterFactory
}

func (p *processor) Run(
	blockParams tosca.BlockParameters,
	transaction tosca.Transaction,
	context tosca.TransactionContext,
) (tosca.Receipt, error) {

	// --- setup ---

	// Hashing function used in the context for BLOCKHASH instruction
	getHash := func(num uint64) common.Hash {
		return common.Hash(context.GetBlockHash(int64(num)))
	}

	// Intercept the transfer function to conduct the transfer on the actual state.
	transferFunc := func(_ geth.StateDB, from common.Address, to common.Address, amount *uint256.Int) {
		if amount.Sign() != 1 || from == to {
			return
		}
		a := tosca.Address(from)
		b := tosca.Address(to)
		d := tosca.ValueFromUint256(amount)
		curA := context.GetBalance(a)
		curB := context.GetBalance(b)
		context.SetBalance(a, tosca.Sub(curA, d))
		context.SetBalance(b, tosca.Add(curB, d))
	}

	// Create empty block context based on block number
	// TODO: this is a copy of geth.go; try to refactor this
	blockCtx := geth.BlockContext{
		BlockNumber: big.NewInt(int64(blockParams.BlockNumber)),
		Time:        uint64(blockParams.Timestamp),
		Difficulty:  big.NewInt(1), // < TODO: check this
		GasLimit:    uint64(blockParams.GasLimit),
		GetHash:     getHash,
		BaseFee:     new(big.Int).SetBytes(blockParams.BaseFee[:]),
		Transfer:    transferFunc,
		CanTransfer: canTransferFunc,
	}

	// Create empty tx context
	txCtx := geth.TxContext{
		Origin:   common.Address(transaction.Sender),
		GasPrice: new(big.Int).SetBytes(transaction.GasPrice[:]),
	}

	// Create a configuration for the geth EVM.
	config := geth.Config{
		Interpreter: p.interpreter,
		StatePrecompiles: map[common.Address]geth.PrecompiledStateContract{
			stateContractAddress: preCompiledStateContract{},
		},
	}

	// Set hard forks for chainconfig
	chainConfig :=
		geth_interpreter.MakeChainConfig(*params.AllEthashProtocolChanges,
			new(big.Int).SetBytes(blockParams.ChainID[:]),
			blockParams.Revision)

	// Fix block boundaries to match required revisions
	chainConfig.IstanbulBlock = big.NewInt(0)
	chainConfig.BerlinBlock = big.NewInt(0)
	chainConfig.LondonBlock = big.NewInt(0)

	if blockParams.Revision < tosca.R10_London {
		chainConfig.LondonBlock = big.NewInt(blockParams.BlockNumber + 1)
	}
	if blockParams.Revision < tosca.R09_Berlin {
		chainConfig.BerlinBlock = big.NewInt(blockParams.BlockNumber + 1)
	}
	if blockParams.Revision < tosca.R07_Istanbul {
		chainConfig.IstanbulBlock = big.NewInt(blockParams.BlockNumber + 1)
	}

	stateDb := geth_interpreter.NewStateDbAdapter(context)
	evm := geth.NewEVM(blockCtx, txCtx, stateDb, &chainConfig, config)

	// -- start of execution --

	// This code is required to mimic the behavior of Sonic's
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
	if err := preCheck(transaction, context); err != nil {
		return tosca.Receipt{}, err
	}
	// Check clauses 4-5, subtract intrinsic gas if everything is correct
	intrinsicGasCosts, err := IntrinsicGas(transaction)
	if err != nil {
		return tosca.Receipt{}, err
	}
	if gas < intrinsicGasCosts {
		return tosca.Receipt{GasUsed: transaction.GasLimit}, fmt.Errorf("%w: have %d, want %d", errIntrinsicGas, transaction.GasLimit, intrinsicGasCosts)
	}
	gas -= intrinsicGasCosts

	sender := geth.AccountRef(transaction.Sender)
	contractCreation := transaction.Recipient == nil

	// Set up the initial access list.
	if blockParams.Revision >= tosca.R09_Berlin {
		var dest *common.Address
		if transaction.Recipient != nil {
			dest = &common.Address{}
			*dest = common.Address(*transaction.Recipient)
		}

		// London uses the same list as Berlin, Cancun extends it.
		precompiledContracts := geth.PrecompiledAddressesBerlin

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
		createdContract *tosca.Address
	)
	if contractCreation {
		var created common.Address
		output, created, gasLeft, vmError = evm.Create(sender, transaction.Input, uint64(gas), transaction.Value.ToUint256())
		createdContract = &tosca.Address{}
		*createdContract = tosca.Address(created)
	} else {
		// Increment the nonce to avoid double execution
		stateDb.SetNonce(common.Address(transaction.Sender), stateDb.GetNonce(common.Address(transaction.Sender))+1)
		output, gasLeft, vmError = evm.Call(sender, common.Address(*transaction.Recipient), transaction.Input, uint64(gas), transaction.Value.ToUint256())
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
		if blockParams.Revision < tosca.R10_London {
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

	// refund remaining gas
	refundGas(transaction, tosca.Gas(gasLeft), context)

	// Extract log messages.
	logs := make([]tosca.Log, 0)
	for _, log := range stateDb.GetLogs() {
		topics := make([]tosca.Hash, len(log.Topics))
		for i, topic := range log.Topics {
			topics[i] = tosca.Hash(topic)
		}
		logs = append(logs, tosca.Log{
			Address: tosca.Address(log.Address),
			Topics:  topics,
			Data:    log.Data,
		})
	}

	return tosca.Receipt{
		Success:         vmError == nil,
		GasUsed:         transaction.GasLimit - tosca.Gas(gasLeft),
		ContractAddress: createdContract,
		Output:          output,
		Logs:            logs,
	}, nil
}

var emptyCodeHash = keccak(nil)

func keccak(data []byte) tosca.Hash {
	res := tosca.Hash{}
	hasher := sha3.NewLegacyKeccak256()
	hasher.Write(data)
	hasher.Sum(res[0:0])
	return res
}

func preCheck(transaction tosca.Transaction, state tosca.WorldState) error {
	// Only check transactions that are not fake
	// TODO: add support for non-checked transactions

	// Make sure this transaction's nonce is correct.
	stNonce := state.GetNonce(transaction.Sender)
	if msgNonce := transaction.Nonce; stNonce < msgNonce {
		//skippedTxsNonceTooHighMeter.Mark(1)
		return fmt.Errorf("%w: address %v, tx: %d state: %d", errNonceTooHigh,
			transaction.Sender, msgNonce, stNonce)
	} else if stNonce > msgNonce {
		//skippedTxsNonceTooLowMeter.Mark(1)
		return fmt.Errorf("%w: address %v, tx: %d state: %d", errNonceTooLow,
			transaction.Sender, msgNonce, stNonce)
	}
	// Make sure the sender is an EOA (Externally Owned Account)
	if codeHash := state.GetCodeHash(transaction.Sender); codeHash != emptyCodeHash && codeHash != (tosca.Hash{}) {
		return fmt.Errorf("%w: address %v, codehash: %s", errSenderNoEOA,
			transaction.Sender, codeHash)
	}

	// Note: Opera doesn't need to check gasFeeCap >= BaseFee, because it's already checked by epochcheck
	return buyGas(transaction, state)
}

func buyGas(tx tosca.Transaction, state tosca.WorldState) error {
	// TODO: support arithmetic operations with Value type
	gasPrice := tx.GasPrice.ToUint256()
	mgval := uint256.NewInt(uint64(tx.GasLimit))
	mgval = mgval.Mul(mgval, gasPrice)
	// Note: Opera doesn't need to check against gasFeeCap instead of gasPrice, as it's too aggressive in the asynchronous environment
	balance := state.GetBalance(tx.Sender)
	if have, want := balance.ToUint256(), mgval; have.Cmp(want) < 0 {
		//skippedTxsNoBalanceMeter.Mark(1)
		return fmt.Errorf("%w: address %v have %v want %v", errInsufficientFunds, tx.Sender, have, want)
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
	balance = tosca.Sub(balance, tosca.ValueFromUint256(mgval))
	state.SetBalance(tx.Sender, balance)
	return nil
}

func refundGas(tx tosca.Transaction, gasLeft tosca.Gas, state tosca.WorldState) {

	// Return wei for remaining gas, exchanged at the original rate.
	refund := new(uint256.Int).Mul(new(uint256.Int).SetUint64(uint64(gasLeft)), tx.GasPrice.ToUint256())

	cur := state.GetBalance(tx.Sender)
	updated := new(uint256.Int).Add(cur.ToUint256(), refund)
	state.SetBalance(tx.Sender, tosca.ValueFromUint256(updated))

	/*
		// TODO: track block-wide gas usage
		// Also return remaining gas to the block gas counter so it is
		// available for the next transaction.
		st.gp.AddGas(st.gas)
	*/
}

// IntrinsicGas computes the 'intrinsic gas' for a message with the given data.
func IntrinsicGas(transaction tosca.Transaction) (tosca.Gas, error) {
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
			return transaction.GasLimit, geth.ErrOutOfGas
		}
		gas += nz * params.TxDataNonZeroGasEIP2028

		z := uint64(len(transaction.Input)) - nz
		if (math.MaxUint64-gas)/params.TxDataZeroGas < z {
			return transaction.GasLimit, errGasUintOverflow
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
	return tosca.Gas(gas), nil
}

func isInternal(transaction tosca.Transaction) bool {
	return transaction.Sender == tosca.Address{}
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

// preCompiledStateContract is a Fantom specific pre-compiled contract that enables
// arbitrary state manipulation for book-keeping and testing purposes.
// It is copied here to avoid a dependency to the Sonic project, which would risk
// substantial dependency issues in down-stream projects.
// Source: https://github.com/Fantom-foundation/Sonic/blob/34b607b882eca12fe25cfc28cbcfa869def6d3f3/opera/contracts/evmwriter/evm_writer.go#L54
type preCompiledStateContract struct{}

func (preCompiledStateContract) Run(
	stateDB geth.StateDB,
	_ geth.BlockContext,
	txCtx geth.TxContext,
	caller common.Address,
	input []byte,
	suppliedGas uint64,
) ([]byte, uint64, error) {
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
		value := new(uint256.Int).SetBytes(input[:32])

		if acc == txCtx.Origin {
			// Origin balance shouldn't decrease during his transaction
			return nil, 0, geth.ErrExecutionReverted
		}

		balance := stateDB.GetBalance(acc)
		if balance.Cmp(value) >= 0 {
			diff := new(uint256.Int).Sub(balance, value)
			stateDB.SubBalance(acc, diff, tracing.BalanceChangeUnspecified)
		} else {
			diff := new(uint256.Int).Sub(value, balance)
			stateDB.AddBalance(acc, diff, tracing.BalanceChangeUnspecified)
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

// canTransferFunc is the signature of a transfer function
func canTransferFunc(stateDB geth.StateDB, callerAddress common.Address, value *uint256.Int) bool {
	return stateDB.GetBalance(callerAddress).Cmp(value) >= 0
}
