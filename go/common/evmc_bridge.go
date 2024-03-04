package common

/*
#cgo CFLAGS: -I${SRCDIR}/../../third_party/evmc/include -Wall -Wextra
#cgo !windows LDFLAGS: -ldl

#include <evmc/evmc.h>
*/
import "C"

import (
	"fmt"

	"github.com/Fantom-foundation/Tosca/go/vm"
	"github.com/ethereum/evmc/v10/bindings/go/evmc"
)

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

/*
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
*/

// EvmcVM is a VirtualMachine implementation accessible through the EVMC library.
type EvmcVM struct {
	vm *evmc.VM
}

// SetOption enables the configuration of implementation specific options.
func (e *EvmcVM) SetOption(property string, value string) error {
	return e.vm.SetOption(property, value)
}

func (e *EvmcVM) Run(params vm.Parameters) (vm.Result, error) {
	host_ctx := HostContext{
		params:  params,
		context: params.Context,
	}

	// Pick proper EVM revision based on block height.
	var revision evmc.Revision
	switch params.Revision {
	case vm.R07_Istanbul:
		revision = evmc.Istanbul
	case vm.R09_Berlin:
		revision = evmc.Berlin
	case vm.R10_London:
		revision = evmc.London
	default:
		return vm.Result{}, fmt.Errorf("unsupported revision: %v", params.Revision)
	}

	var codeHash *evmc.Hash
	if params.CodeHash != nil {
		codeHash = new(evmc.Hash)
		*codeHash = evmc.Hash(*params.CodeHash)
	}

	// Forward the execution call to the underlying EVM implementation.
	result, err := e.vm.Execute(evmc.Parameters{
		Context:   &host_ctx,
		Revision:  revision,
		Kind:      evmc.Call,
		Static:    params.Static,
		Depth:     params.Depth,
		Gas:       int64(params.Gas),
		Recipient: evmc.Address(params.Recipient),
		Sender:    evmc.Address(params.Sender),
		Input:     params.Input,
		Value:     evmc.Hash(params.Value),
		CodeHash:  codeHash,
		Code:      params.Code,
	})

	// Build result struct.
	res := vm.Result{
		Success:   true,
		Output:    result.Output,
		GasLeft:   vm.Gas(result.GasLeft),
		GasRefund: vm.Gas(result.GasRefund),
	}

	// If no error was reported, the processing stopped with a STOP,
	// RETURN, or SELF-DESTRUCT instruction.
	if err == nil {
		return res, nil
	}

	// translate error codes to vm errors
	switch err {
	case evmc.Revert:
		// This is not really an error, but actually a revert.
		// This is to be processed as a successful execution.
		res.Success = false // < signal that execution reverted
		return res, nil
	case evmc.Error(C.EVMC_OUT_OF_GAS),
		evmc.Error(C.EVMC_INVALID_INSTRUCTION),
		evmc.Error(C.EVMC_UNDEFINED_INSTRUCTION),
		evmc.Error(C.EVMC_BAD_JUMP_DESTINATION),
		evmc.Error(C.EVMC_INVALID_MEMORY_ACCESS),
		evmc.Error(C.EVMC_STATIC_MODE_VIOLATION),
		evmc.Error(C.EVMC_STACK_OVERFLOW),
		evmc.Error(C.EVMC_STACK_UNDERFLOW):
		// These are errors in the executed contract, but not VM errors.
		// The result is thus marked as not successfully, and all gas is
		// removed. Also, all refunds are removed and no data is returned.
		return vm.Result{Success: false}, nil
	default:
		return vm.Result{}, fmt.Errorf("unexpected EVMC execution error: %w", err)
	}
}

// GetEvmcVM provides direct access to the VM connected through the EVMC library.
func (e *EvmcVM) GetEvmcVM() *evmc.VM {
	return e.vm
}

// Destroy releases resources bound by this VM instance.
func (e *EvmcVM) Destroy() {
	if e.vm != nil {
		e.vm.Destroy()
	}
	e.vm = nil
}

// The HostContext allows a non-Go EVM implementation to access the StateDB and
// other systems external to the interpreter. This implementation leverages
// evmc's Go bindings.
type HostContext struct {
	params  vm.Parameters
	context vm.RunContext
}

func (ctx *HostContext) AccountExists(addr evmc.Address) bool {
	return ctx.context.AccountExists(vm.Address(addr))
}

func (ctx *HostContext) GetStorage(addr evmc.Address, key evmc.Hash) evmc.Hash {
	return evmc.Hash(ctx.context.GetStorage(vm.Address(addr), vm.Key(key)))
}

func (ctx *HostContext) SetStorage(addr evmc.Address, key evmc.Hash, value evmc.Hash) evmc.StorageStatus {
	status := ctx.context.SetStorage(vm.Address(addr), vm.Key(key), vm.Word(value))
	switch status {
	case vm.StorageAssigned:
		return evmc.StorageAssigned
	case vm.StorageAdded:
		return evmc.StorageAdded
	case vm.StorageDeleted:
		return evmc.StorageDeleted
	case vm.StorageModified:
		return evmc.StorageModified
	case vm.StorageDeletedAdded:
		return evmc.StorageDeletedAdded
	case vm.StorageModifiedDeleted:
		return evmc.StorageModifiedDeleted
	case vm.StorageDeletedRestored:
		return evmc.StorageDeletedRestored
	case vm.StorageAddedDeleted:
		return evmc.StorageAddedDeleted
	case vm.StorageModifiedRestored:
		return evmc.StorageModifiedRestored
	default:
		panic(fmt.Sprintf("unsupported storage state: %v", status))
	}
}

func (ctx *HostContext) GetBalance(addr evmc.Address) evmc.Hash {
	return evmc.Hash(ctx.context.GetBalance(vm.Address(addr)))
}

func (ctx *HostContext) GetCodeSize(addr evmc.Address) int {
	return ctx.context.GetCodeSize(vm.Address(addr))
}

func (ctx *HostContext) GetCodeHash(addr evmc.Address) evmc.Hash {
	return evmc.Hash(ctx.context.GetCodeHash(vm.Address(addr)))
}

func (ctx *HostContext) GetCode(addr evmc.Address) []byte {
	return ctx.context.GetCode(vm.Address(addr))
}

func (ctx *HostContext) Selfdestruct(addr evmc.Address, beneficiary evmc.Address) bool {
	return ctx.context.SelfDestruct(vm.Address(addr), vm.Address(beneficiary))
}

func (ctx *HostContext) GetTxContext() evmc.TxContext {
	ctxt := ctx.context.GetTransactionContext()
	return evmc.TxContext{
		GasPrice:   evmc.Hash(ctxt.GasPrice),
		Origin:     evmc.Address(ctxt.Origin),
		Coinbase:   evmc.Address(ctxt.Coinbase),
		Number:     ctxt.BlockNumber,
		Timestamp:  ctxt.Timestamp,
		GasLimit:   int64(ctxt.GasLimit),
		PrevRandao: evmc.Hash(ctxt.PrevRandao),
		ChainID:    evmc.Hash(ctxt.ChainID),
		BaseFee:    evmc.Hash(ctxt.BaseFee),
	}
}

func (ctx *HostContext) GetBlockHash(number int64) evmc.Hash {
	return evmc.Hash(ctx.context.GetBlockHash(number))
}

func (ctx *HostContext) EmitLog(addr evmc.Address, topics_in []evmc.Hash, data []byte) {
	topics := make([]vm.Hash, len(topics_in))
	for i := range topics {
		topics[i] = vm.Hash(topics_in[i])
	}
	ctx.context.EmitLog(vm.Address(addr), topics, data)
}

func (ctx *HostContext) Call(kind evmc.CallKind, recipient evmc.Address, sender evmc.Address, value evmc.Hash, input []byte, gas int64, depth int, static bool, salt evmc.Hash, codeAddress evmc.Address) (output []byte, gasLeft int64, gasRefund int64, createAddr evmc.Address, err error) {

	var callKind vm.CallKind
	switch kind {
	case evmc.Create:
		callKind = vm.Create
	case evmc.Create2:
		callKind = vm.Create2
	case evmc.Call:
		callKind = vm.Call
		if static {
			callKind = vm.StaticCall
		}
	case evmc.CallCode:
		callKind = vm.CallCode
	case evmc.DelegateCall:
		callKind = vm.DelegateCall
	default:
		panic(fmt.Sprintf("unsupported call kind: %v", kind))
	}

	params := vm.CallParameter{
		Sender:      vm.Address(sender),
		Recipient:   vm.Address(recipient),
		Value:       vm.Value(value),
		Input:       input,
		Gas:         vm.Gas(gas),
		Salt:        vm.Hash(salt),
		CodeAddress: vm.Address(codeAddress),
	}

	result, err := ctx.context.Call(callKind, params)
	if err != nil {
		return nil, 0, 0, evmc.Address{}, err
	}
	if !result.Reverted {
		return nil, 0, 0, evmc.Address{}, evmc.Revert
	}

	return result.Output,
		int64(result.GasLeft),
		int64(result.GasRefund),
		evmc.Address(result.CreatedAddress),
		nil
}

func (ctx *HostContext) AccessAccount(addr evmc.Address) evmc.AccessStatus {
	if ctx.context.AccessAccount(vm.Address(addr)) == vm.WarmAccess {
		return evmc.WarmAccess
	}
	return evmc.ColdAccess
}

func (ctx *HostContext) AccessStorage(addr evmc.Address, key evmc.Hash) evmc.AccessStatus {
	if ctx.context.AccessStorage(vm.Address(addr), vm.Key(key)) == vm.WarmAccess {
		return evmc.WarmAccess
	}
	return evmc.ColdAccess
}
