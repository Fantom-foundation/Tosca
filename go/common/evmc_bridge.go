package common

/*
#cgo CFLAGS: -I${SRCDIR}/../../third_party/evmc/include -Wall -Wextra
#cgo !windows LDFLAGS: -ldl

#include <evmc/evmc.h>
*/
import "C"

import (
	"fmt"
	"math"
	"math/big"

	"github.com/ethereum/evmc/v10/bindings/go/evmc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/params"
	"github.com/holiman/uint256"
)

// EvmcVM is a EVM implementation accessible through the EVMC library.
type EvmcVM struct {
	vm *evmc.VM
}

type EvmcVMSteppable struct {
	vm *evmc.VMSteppable
}

type EvmcSteppableInterpreter struct {
	evmc        *EvmcVMSteppable
	interpreter *EvmcInterpreter
}

// NewEvmcSteppableInterpreter instantiates an interpreter with the given evm and config.
func NewEvmcSteppableInterpreter(vm *EvmcVMSteppable, evm *vm.EVM, cfg vm.Config) *EvmcSteppableInterpreter {
	return &EvmcSteppableInterpreter{
		evmc: vm,
		interpreter: &EvmcInterpreter{
			evmc: (*EvmcVM)(vm.vm.GetBaseHandle()),
			evm:  evm,
			cfg:  cfg,
		},
	}
}

func (e *EvmcSteppableInterpreter) StepN(contract *vm.Contract, revision evmc.Revision, readOnly bool, gasRefund uint64, input []byte, stepStatus evmc.StepStatus, pc uint64, stack []byte, memory []byte, numSteps int) (evmc.StepResult, error) {
	host_ctx := HostContext{
		evm:         e.interpreter.evm,
		interpreter: e.interpreter,
		contract:    contract,
	}

	// The EVMC binding uses int64 to represent gas values while Geth utilizes
	// uint64. Thus, everything larger than math.MaxInt64 will lead to negative
	// values after the conversion. However, in practice, gas limits should be
	// way below MaxInt64, which would by 2^63-1 gas units -- an equivalent of
	// 10 days processing if 10.000 gas/ns would get burned. It would also cost
	// more than 10 Billion FTM (assuming 1 Gwei/gas, which is usally >100) to
	// run this contract, which is > 3x more than there is in existence.
	// The assumption is that gas endowments > MaxInt64 are test cases.
	gasBefore := int64(contract.Gas)
	if contract.Gas > math.MaxInt64 {
		gasBefore = math.MaxInt64
	}
	gasRefundBefore := int64(gasRefund)
	if gasRefund > math.MaxInt64 {
		gasRefundBefore = math.MaxInt64
	}

	value, err := bigIntToHash(contract.Value())
	if err != nil {
		panic(fmt.Sprintf("Could not convert value: %v", err))
	}

	codeHash := evmc.Hash(contract.CodeHash)

	params := evmc.StepParameters{
		Context:        &host_ctx,
		Revision:       revision,
		Kind:           evmc.Call,
		Static:         readOnly,
		Depth:          e.interpreter.evm.Depth - 1,
		Gas:            gasBefore,
		GasRefund:      gasRefundBefore,
		Recipient:      evmc.Address(contract.Address()),
		Sender:         evmc.Address(contract.Caller()),
		Input:          input,
		Value:          value,
		CodeHash:       &codeHash,
		Code:           contract.Code,
		StepStatusCode: stepStatus,
		Pc:             pc,
		Stack:          stack,
		Memory:         memory,
		NumSteps:       numSteps,
	}

	return e.evmc.vm.StepN(params)
}

func (e *EvmcSteppableInterpreter) Run(contract *vm.Contract, input []byte, readOnly bool) (ret []byte, err error) {
	return e.interpreter.Run(contract, input, readOnly)
}

// LoadEvmcVM attempts to load an EVM implementation from a given library.
// The `library` parameter should name the library file, while the actual
// path to the library should be enforced using an rpath (see evmone
// implementation for an example).
func LoadEvmcVM(library string) (*EvmcVM, error) {
	vm, err := evmc.Load(library)
	if err != nil {
		return nil, err
	}
	return &EvmcVM{vm: vm}, nil
}

// LoadEvmcVMSteppable attempts to load an EVM implementation from a given library.
// The `library` parameter should name the library file, while the actual
// path to the library should be enforced using an rpath (see evmone
// implementation for an example).
func LoadEvmcVMSteppable(library string) (*EvmcVMSteppable, error) {
	vm, err := evmc.LoadSteppable(library)
	if err != nil {
		return nil, err
	}
	return &EvmcVMSteppable{vm: vm}, nil
}

// SetOption enables the configuration of implementation specific options.
func (e *EvmcVM) SetOption(property string, value string) error {
	return e.vm.SetOption(property, value)
}

// Destroy releases resources bound by this VM instance.
func (e *EvmcVM) Destroy() {
	if e.vm != nil {
		e.vm.Destroy()
	}
	e.vm = nil
}

// GetEvmcVM provides direct access to the VM connected through the EVMC library.
func (e *EvmcVM) GetEvmcVM() *evmc.VM {
	return e.vm
}

// NewEvmcInterpreter instantiates an interpreter with the given evm and config.
func NewEvmcInterpreter(vm *EvmcVM, evm *vm.EVM, cfg vm.Config) *EvmcInterpreter {
	return &EvmcInterpreter{
		evmc: vm,
		evm:  evm,
		cfg:  cfg,
	}
}

type EvmcInterpreter struct {
	evmc          *EvmcVM
	evm           *vm.EVM
	cfg           vm.Config
	readOnly      bool
	lastGasRefund int64
}

func (e *EvmcInterpreter) Run(contract *vm.Contract, input []byte, readOnly bool) (ret []byte, err error) {

	// Track the recursive call depth of this Call within a transaction.
	// A maximum limit of params.CallCreateDepth must be enforced.
	if e.evm.Depth > int(params.CallCreateDepth) {
		return nil, vm.ErrDepth
	}
	e.evm.Depth++
	defer func() { e.evm.Depth-- }()

	host_ctx := HostContext{
		evm:         e.evm,
		interpreter: e,
		contract:    contract,
	}

	// Pick proper EVM revision based on block height.
	revision := evmc.Istanbul
	if chainConfig := e.evm.ChainConfig(); chainConfig != nil {
		// Note: configurations need to be checked in reverse order since
		// later revisions implicitly include earlier revisions.
		if chainConfig.IsLondon(e.evm.Context.BlockNumber) {
			revision = evmc.London
		} else if chainConfig.IsBerlin(e.evm.Context.BlockNumber) {
			revision = evmc.Berlin
		}
	}

	// Make sure the readOnly is only set if we aren't in readOnly yet.
	// This also makes sure that the readOnly flag isn't removed for child calls.
	if readOnly && !e.readOnly {
		e.readOnly = true
		defer func() { e.readOnly = false }()
	}

	// The EVMC binding uses int64 to represent gas values while Geth utilizes
	// uint64. Thus, everything larger than math.MaxInt64 will lead to negative
	// values after the conversion. However, in practice, gas limits should be
	// way below MaxInt64, which would by 2^63-1 gas units -- an equivalent of
	// 10 days processing if 10.000 gas/ns would get burned. It would also cost
	// more than 10 Billion FTM (assuming 1 Gwei/gas, which is usally >100) to
	// run this contract, which is > 3x more than there is in existence.
	// The assumption is that gas endowments > MaxInt64 are test cases.
	gasBefore := int64(contract.Gas)
	if contract.Gas > math.MaxInt64 {
		gasBefore = math.MaxInt64
	}

	value, err := bigIntToHash(contract.Value())
	if err != nil {
		panic(fmt.Sprintf("Could not convert value: %v", err))
	}

	codeHash := evmc.Hash(contract.CodeHash)

	// Forward the execution call to the underlying EVM implementation.
	result, err := e.evmc.vm.Execute(evmc.Parameters{
		Context:   &host_ctx,
		Revision:  revision,
		Kind:      evmc.Call,
		Static:    e.readOnly,
		Depth:     e.evm.Depth - 1,
		Gas:       gasBefore,
		Recipient: evmc.Address(contract.Address()),
		Sender:    evmc.Address(contract.Caller()),
		Input:     input,
		Value:     value,
		CodeHash:  &codeHash,
		Code:      contract.Code,
	})

	// update remaining gas
	gasUsed := gasBefore - result.GasLeft
	contract.Gas -= uint64(gasUsed)

	if err != nil {
		// translate error codes to vm errors
		switch err {
		case evmc.Revert:
			err = vm.ErrExecutionReverted
			return result.Output, err
		case evmc.Failure:
			err = vm.ErrInvalidCode
		case evmc.Error(C.EVMC_OUT_OF_GAS):
			err = vm.ErrOutOfGas
		case evmc.Error(C.EVMC_INVALID_INSTRUCTION):
			err = &vm.ErrInvalidOpCode{}
		case evmc.Error(C.EVMC_UNDEFINED_INSTRUCTION):
			err = &vm.ErrInvalidOpCode{}
		case evmc.Error(C.EVMC_BAD_JUMP_DESTINATION):
			err = vm.ErrInvalidJump
		case evmc.Error(C.EVMC_INVALID_MEMORY_ACCESS):
			// Technically not every EVMC_INVALID_MEMORY_ACCESS is an
			// ErrReturnDataOutOfBounds, but there is currently no dedicated
			// error defined in EVMC that lets us distinguish between the
			// different cases.
			//
			// Similarly, the evm module does not define a generic
			// ErrInvalidMemoryAccess that we could use here.
			err = vm.ErrReturnDataOutOfBounds
		case evmc.Error(C.EVMC_STATIC_MODE_VIOLATION):
			err = vm.ErrWriteProtection
		case evmc.Error(C.EVMC_STACK_OVERFLOW):
			err = &vm.ErrStackOverflow{}
		case evmc.Error(C.EVMC_STACK_UNDERFLOW):
			err = &vm.ErrStackUnderflow{}
		}
		return nil, err
	}

	// update the amount of refund gas in the state DB
	if e.evm.Depth == 1 {
		state := e.evm.StateDB
		if state != nil {
			if result.GasRefund != 0 {
				if result.GasRefund > 0 {
					state.AddRefund(uint64(result.GasRefund))
				} else {
					state.SubRefund(uint64(result.GasRefund * -1))
				}
			}
		}
	} else {
		e.lastGasRefund = result.GasRefund
	}

	return result.Output, err
}

// The HostContext allows a non-Go EVM implementation to access the StateDB and
// other systems external to the interpreter. This implementation leverages
// evmc's Go bindings.
type HostContext struct {
	evm         *vm.EVM
	interpreter *EvmcInterpreter
	contract    *vm.Contract
}

func (ctx *HostContext) AccountExists(addr evmc.Address) bool {
	return ctx.interpreter.evm.StateDB.Exist((common.Address)(addr))
}

func (ctx *HostContext) GetStorage(addr evmc.Address, key evmc.Hash) evmc.Hash {
	return evmc.Hash(ctx.interpreter.evm.StateDB.GetState((common.Address)(addr), (common.Hash)(key)))
}

func (ctx *HostContext) SetStorage(evmcAddr evmc.Address, evmcKey evmc.Hash, evmcValue evmc.Hash) evmc.StorageStatus {
	var zero = common.Hash{}

	// See t.ly/b5HPf for the definition of the return status.
	addr := (common.Address)(evmcAddr)
	key := (common.Hash)(evmcKey)
	newValue := (common.Hash)(evmcValue)

	stateDB := ctx.interpreter.evm.StateDB
	currentValue := stateDB.GetState(addr, key)
	if currentValue == newValue {
		return evmc.StorageAssigned
	}
	stateDB.SetState(addr, key, newValue)

	originalValue := stateDB.GetCommittedState(addr, key)

	// 0 -> 0 -> Z
	if originalValue == zero && currentValue == zero && newValue != zero {
		return evmc.StorageAdded
	}

	// X -> X -> 0
	if originalValue != zero && currentValue == originalValue && newValue == zero {
		return evmc.StorageDeleted
	}

	// X -> X -> Z
	if originalValue != zero && currentValue == originalValue && newValue != zero && newValue != originalValue {
		return evmc.StorageModified
	}

	// X -> 0 -> Z
	if originalValue != zero && currentValue == zero && newValue != originalValue && newValue != zero {
		return evmc.StorageDeletedAdded
	}

	// X -> Y -> 0
	if originalValue != zero && currentValue != originalValue && currentValue != zero && newValue == zero {
		return evmc.StorageModifiedDeleted
	}

	// X -> 0 -> X
	if originalValue != zero && currentValue == zero && newValue == originalValue {
		return evmc.StorageDeletedRestored
	}

	// 0 -> Y -> 0
	if originalValue == zero && currentValue != zero && newValue == zero {
		return evmc.StorageAddedDeleted
	}

	// X -> Y -> X
	if originalValue != zero && currentValue != originalValue && currentValue != zero && newValue == originalValue {
		return evmc.StorageModifiedRestored
	}

	// Default
	return evmc.StorageAssigned
}

func (ctx *HostContext) GetBalance(addr evmc.Address) evmc.Hash {
	balance := ctx.interpreter.evm.StateDB.GetBalance((common.Address)(addr))
	result, err := bigIntToHash(balance)
	if err != nil {
		panic(fmt.Sprintf("Could not convert balance: %v", err))
	}
	return result
}

func (ctx *HostContext) GetCodeSize(addr evmc.Address) int {
	return ctx.interpreter.evm.StateDB.GetCodeSize((common.Address)(addr))
}

func (ctx *HostContext) GetCodeHash(addr evmc.Address) evmc.Hash {
	if ctx.interpreter.evm.StateDB.Empty((common.Address)(addr)) {
		return evmc.Hash{0}
	} else {
		return evmc.Hash(ctx.interpreter.evm.StateDB.GetCodeHash((common.Address)(addr)))
	}
}

func (ctx *HostContext) GetCode(addr evmc.Address) []byte {
	return ctx.interpreter.evm.StateDB.GetCode((common.Address)(addr))
}

func (ctx *HostContext) Selfdestruct(addr evmc.Address, beneficiary evmc.Address) bool {
	if ctx.interpreter.evm.StateDB.HasSuicided((common.Address)(addr)) {
		return false
	}
	balance := ctx.interpreter.evm.StateDB.GetBalance(ctx.contract.Address())
	ctx.interpreter.evm.StateDB.AddBalance(common.Address(beneficiary), balance)
	ctx.interpreter.evm.StateDB.Suicide((common.Address)(addr))
	return true
}

func (ctx *HostContext) GetTxContext() evmc.TxContext {
	gasPrice, err := bigIntToHash(ctx.interpreter.evm.TxContext.GasPrice)
	if err != nil {
		panic(fmt.Sprintf("Could not convert gas price: %v", err))
	}

	chainId, err := bigIntToHash(ctx.interpreter.evm.ChainConfig().ChainID)
	if err != nil {
		panic(fmt.Sprintf("Could not convert chain Id: %v", err))
	}

	// BaseFee can be assumed zero unless set.
	var baseFee evmc.Hash
	if ctx.interpreter.evm.Context.BaseFee != nil {
		baseFee, err = bigIntToHash(ctx.interpreter.evm.Context.BaseFee)
		if err != nil {
			panic(fmt.Sprintf("Could not convert base fee: %v", err))
		}
	}

	var difficulty evmc.Hash
	if ctx.interpreter.evm.Context.Difficulty != nil {
		difficulty, err = bigIntToHash(ctx.interpreter.evm.Context.Difficulty)
		if err != nil {
			panic(fmt.Sprintf("Could not convert difficulty: %v", err))
		}
	}

	return evmc.TxContext{
		GasPrice:   gasPrice,
		Origin:     evmc.Address(ctx.interpreter.evm.Origin),
		Coinbase:   evmc.Address(ctx.interpreter.evm.Context.Coinbase),
		Number:     ctx.interpreter.evm.Context.BlockNumber.Int64(),
		Timestamp:  ctx.interpreter.evm.Context.Time.Int64(),
		GasLimit:   int64(ctx.interpreter.evm.Context.GasLimit),
		PrevRandao: difficulty,
		ChainID:    chainId,
		BaseFee:    baseFee,
	}
}

func (ctx *HostContext) GetBlockHash(number int64) evmc.Hash {
	return evmc.Hash(ctx.interpreter.evm.Context.GetHash(uint64(number)))
}

func (ctx *HostContext) EmitLog(addr evmc.Address, topics_in []evmc.Hash, data []byte) {
	topics := make([]common.Hash, len(topics_in))
	for i := range topics {
		topics[i] = (common.Hash)(topics_in[i])
	}

	ctx.interpreter.evm.StateDB.AddLog(&types.Log{
		Address: (common.Address)(addr),
		Topics:  ([]common.Hash)(topics),
		Data:    data,
		// This is a non-consensus field, but assigned here because
		// core/state doesn't know the current block number.
		BlockNumber: ctx.interpreter.evm.Context.BlockNumber.Uint64(),
	})
}

func (ctx *HostContext) Call(kind evmc.CallKind, recipient evmc.Address, sender evmc.Address, value evmc.Hash, input []byte, gas int64, depth int, static bool, salt evmc.Hash, codeAddress evmc.Address) (output []byte, gasLeft int64, gasRefund int64, createAddr evmc.Address, err error) {
	// Documentation of the parameters can be found here: t.ly/yhxC
	toAddr := common.Address(codeAddress)

	var returnGas uint64
	switch kind {
	case evmc.Call:
		if static {
			output, returnGas, err = ctx.evm.StaticCall(ctx.contract, toAddr, input, uint64(gas))
		} else {
			output, returnGas, err = ctx.evm.Call(ctx.contract, toAddr, input, uint64(gas), hashToBigInt(&value))
		}
	case evmc.DelegateCall:
		output, returnGas, err = ctx.evm.DelegateCall(ctx.contract, toAddr, input, uint64(gas))
	case evmc.CallCode:
		output, returnGas, err = ctx.evm.CallCode(ctx.contract, toAddr, input, uint64(gas), hashToBigInt(&value))
	case evmc.Create:
		var newAddr common.Address
		output, newAddr, returnGas, err = ctx.evm.Create(ctx.contract, input, uint64(gas), hashToBigInt(&value))
		createAddr = evmc.Address(newAddr)
	case evmc.Create2:
		var newAddr common.Address
		vmSalt := &uint256.Int{}
		vmSalt.SetBytes(salt[:])
		output, newAddr, returnGas, err = ctx.evm.Create2(ctx.contract, input, uint64(gas), hashToBigInt(&value), vmSalt)
		createAddr = evmc.Address(newAddr)
	default:
		panic(fmt.Sprintf("unsupported call kind: %v", kind))
	}
	gasLeft = int64(returnGas)

	gasRefund = ctx.interpreter.lastGasRefund
	ctx.interpreter.lastGasRefund = 0

	if err != nil {
		// translate vm errors to evmc errors
		switch err {
		case vm.ErrExecutionReverted:
			err = evmc.Revert
		case vm.ErrInvalidCode:
			err = evmc.Failure
		case vm.ErrOutOfGas:
			err = evmc.Error(C.EVMC_OUT_OF_GAS)
		default:
			err = evmc.Failure
		}
	}

	return
}

func (ctx *HostContext) AccessAccount(evmcAddr evmc.Address) evmc.AccessStatus {
	addr := (common.Address)(evmcAddr)
	stateDB := ctx.interpreter.evm.StateDB
	if stateDB.AddressInAccessList(addr) {
		return evmc.WarmAccess
	} else {
		stateDB.AddAddressToAccessList(addr)
		return evmc.ColdAccess
	}
}

func (ctx *HostContext) AccessStorage(evmcAddr evmc.Address, evmcKey evmc.Hash) evmc.AccessStatus {
	addr := (common.Address)(evmcAddr)
	key := (common.Hash)(evmcKey)
	stateDB := ctx.interpreter.evm.StateDB
	_, slotOk := stateDB.SlotInAccessList((common.Address)(addr), (common.Hash)(key))
	if slotOk {
		return evmc.WarmAccess
	} else {
		stateDB.AddSlotToAccessList(addr, key)
		return evmc.ColdAccess
	}
}

func bigIntToHash(value *big.Int) (result evmc.Hash, err error) {
	if value == nil {
		return result, fmt.Errorf("unable to convert nil to Hash")
	}
	if value.Sign() < 0 {
		return result, fmt.Errorf("cannot convert a negative number to a Hash, got %v", value)
	}
	if len(value.Bytes()) > 32 {
		return result, fmt.Errorf("value exceeds maximum value for Hash, %v of 32 bytes max", len(value.Bytes()))
	}
	value.FillBytes(result[:])
	return result, nil
}

func hashToBigInt(hash *evmc.Hash) *big.Int {
	res := &big.Int{}
	res.SetBytes(hash[:])
	return res
}
