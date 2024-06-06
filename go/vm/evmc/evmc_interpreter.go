//
// Copyright (c) 2024 Fantom Foundation
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at fantom.foundation/bsl11.
//
// Change Date: 2028-4-16
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by the GNU Lesser General Public Licence v3
//

package evmc

/*
#cgo CFLAGS: -I${SRCDIR}/../../../third_party/evmc/include -Wall -Wextra
#cgo !windows LDFLAGS: -ldl

#include <evmc/evmc.h>
*/
import "C"

import (
	"fmt"

	"github.com/Fantom-foundation/Tosca/go/vm"
	"github.com/ethereum/evmc/v10/bindings/go/evmc"
)

// LoadEvmcInterpreter attempts to load an Interpreter implementation from a
// given library. The `library` parameter should name the library file, while
// the actual path to the library should be enforced using an rpath (see evmone
// implementation for an example).
func LoadEvmcInterpreter(library string) (*EvmcInterpreter, error) {
	vm, err := evmc.Load(library)
	if err != nil {
		return nil, err
	}
	return &EvmcInterpreter{vm: vm}, nil
}

// EvmcInterpreter is an Interpreter implementation accessible through the EVMC library.
type EvmcInterpreter struct {
	vm *evmc.VM
}

// SetOption enables the configuration of implementation specific options.
func (e *EvmcInterpreter) SetOption(property string, value string) error {
	return e.vm.SetOption(property, value)
}

func (e *EvmcInterpreter) Run(params vm.Parameters) (vm.Result, error) {
	host_ctx := hostContext{
		params:  params,
		context: params.Context,
	}

	revision, err := toEvmcRevision(params.Revision)
	if err != nil {
		return vm.Result{}, err
	}

	var codeHash *evmc.Hash
	if params.CodeHash != nil {
		codeHash = new(evmc.Hash)
		*codeHash = evmc.Hash(*params.CodeHash)
	}

	// Forward the execution call to the underlying EVM implementation.
	result, err := e.vm.Execute(
		&host_ctx,
		revision,
		evmc.Call,
		params.Static,
		params.Depth,
		int64(params.Gas),
		evmc.Address(params.Recipient),
		evmc.Address(params.Sender),
		params.Input,
		evmc.Hash(params.Value),
		codeHash,
		params.Code,
	)

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
		// The result is thus marked as not successful, and all gas is
		// removed. Also, all refunds are removed and no data is returned.
		return vm.Result{Success: false}, nil
	default:
		return vm.Result{}, fmt.Errorf("unexpected EVMC execution error: %w", err)
	}
}

// GetEvmcVM provides direct access to the Evmc VM connected through the EVMC library.
func (e *EvmcInterpreter) GetEvmcVM() *evmc.VM {
	return e.vm
}

// Destroy releases resources bound by this VM instance.
func (e *EvmcInterpreter) Destroy() {
	if e.vm != nil {
		e.vm.Destroy()
	}
	e.vm = nil
}

func toEvmcRevision(revision vm.Revision) (evmc.Revision, error) {
	// Pick proper EVM revision based on block height.
	switch revision {
	case vm.R07_Istanbul:
		return evmc.Istanbul, nil
	case vm.R09_Berlin:
		return evmc.Berlin, nil
	case vm.R10_London:
		return evmc.London, nil
	case vm.R11_Paris:
		return evmc.Paris, nil
	case vm.R12_Shanghai:
		return evmc.Shanghai, nil
	default:
		return 0, fmt.Errorf("unsupported revision: %v", revision)
	}
}

// hostContext allows a non-Go Interpreter implementation to access transaction
// context and chain state information external to the the interpreter.
// It implements the host interface of evmc's Go bindings.
type hostContext struct {
	params  vm.Parameters
	context vm.RunContext
}

func (ctx *hostContext) AccountExists(addr evmc.Address) bool {
	return ctx.context.AccountExists(vm.Address(addr))
}

func (ctx *hostContext) GetStorage(addr evmc.Address, key evmc.Hash) evmc.Hash {
	return evmc.Hash(ctx.context.GetStorage(vm.Address(addr), vm.Key(key)))
}

func (ctx *hostContext) SetStorage(addr evmc.Address, key evmc.Hash, value evmc.Hash) evmc.StorageStatus {
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

func (ctx *hostContext) GetTransientStorage(addr evmc.Address, key evmc.Hash) evmc.Hash {
	return evmc.Hash(ctx.context.GetTransientStorage(vm.Address(addr), vm.Key(key)))
}

func (ctx *hostContext) SetTransientStorage(addr evmc.Address, key evmc.Hash, value evmc.Hash) {
	ctx.context.SetTransientStorage(vm.Address(addr), vm.Key(key), vm.Word(value))
}

func (ctx *hostContext) GetBalance(addr evmc.Address) evmc.Hash {
	return evmc.Hash(ctx.context.GetBalance(vm.Address(addr)))
}

func (ctx *hostContext) GetCodeSize(addr evmc.Address) int {
	return ctx.context.GetCodeSize(vm.Address(addr))
}

func (ctx *hostContext) GetCodeHash(addr evmc.Address) evmc.Hash {
	return evmc.Hash(ctx.context.GetCodeHash(vm.Address(addr)))
}

func (ctx *hostContext) GetCode(addr evmc.Address) []byte {
	return ctx.context.GetCode(vm.Address(addr))
}

func (ctx *hostContext) Selfdestruct(addr evmc.Address, beneficiary evmc.Address) bool {
	return ctx.context.SelfDestruct(vm.Address(addr), vm.Address(beneficiary))
}

func (ctx *hostContext) GetTxContext() evmc.TxContext {
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

func (ctx *hostContext) GetBlockHash(number int64) evmc.Hash {
	return evmc.Hash(ctx.context.GetBlockHash(number))
}

func (ctx *hostContext) EmitLog(addr evmc.Address, topics_in []evmc.Hash, data []byte) {
	topics := make([]vm.Hash, len(topics_in))
	for i := range topics {
		topics[i] = vm.Hash(topics_in[i])
	}
	ctx.context.EmitLog(vm.Address(addr), topics, data)
}

func (ctx *hostContext) Call(kind evmc.CallKind, recipient evmc.Address, sender evmc.Address, value evmc.Hash, input []byte, gas int64, depth int, static bool, salt evmc.Hash, codeAddress evmc.Address) (output []byte, gasLeft int64, gasRefund int64, createAddr evmc.Address, err error) {

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
	if !result.Success {
		err = evmc.Revert
	}
	return result.Output,
		int64(result.GasLeft),
		int64(result.GasRefund),
		evmc.Address(result.CreatedAddress),
		err
}

func (ctx *hostContext) AccessAccount(addr evmc.Address) evmc.AccessStatus {
	if ctx.context.AccessAccount(vm.Address(addr)) == vm.WarmAccess {
		return evmc.WarmAccess
	}
	return evmc.ColdAccess
}

func (ctx *hostContext) AccessStorage(addr evmc.Address, key evmc.Hash) evmc.AccessStatus {
	if ctx.context.AccessStorage(vm.Address(addr), vm.Key(key)) == vm.WarmAccess {
		return evmc.WarmAccess
	}
	return evmc.ColdAccess
}
