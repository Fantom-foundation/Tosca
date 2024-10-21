// Copyright (c) 2024 Fantom Foundation
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at fantom.foundation/bsl11.
//
// Change Date: 2028-4-16
//
// On the date above, in accordance with the Business Source License, use of
// this software will be governed by the GNU Lesser General Public License v3.

package evmc

/*
#cgo CFLAGS: -I${SRCDIR}/../../../third_party/evmc/include -Wall -Wextra
#cgo !windows LDFLAGS: -ldl

#include <evmc/evmc.h>
*/
import "C"

import (
	"fmt"

	"github.com/Fantom-foundation/Tosca/go/tosca"
	"github.com/ethereum/evmc/v11/bindings/go/evmc"
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

func (e *EvmcInterpreter) Run(params tosca.Parameters) (tosca.Result, error) {
	host_ctx := hostContext{
		params:  params,
		context: params.Context,
	}

	host_ctx.evmcBlobHashes = make([]evmc.Hash, 0, len(params.BlobHashes))
	for _, hash := range params.BlobHashes {
		host_ctx.evmcBlobHashes = append(host_ctx.evmcBlobHashes, evmc.Hash(hash))
	}

	revision, err := toEvmcRevision(params.Revision)
	if err != nil {
		return tosca.Result{}, err
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
	res := tosca.Result{
		Success:   true,
		Output:    result.Output,
		GasLeft:   tosca.Gas(result.GasLeft),
		GasRefund: tosca.Gas(result.GasRefund),
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
		return tosca.Result{Success: false}, nil
	default:
		return tosca.Result{}, fmt.Errorf("unexpected EVMC execution error: %w", err)
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

func toEvmcRevision(revision tosca.Revision) (evmc.Revision, error) {
	// Pick proper EVM revision based on block height.
	switch revision {
	case tosca.R07_Istanbul:
		return evmc.Istanbul, nil
	case tosca.R09_Berlin:
		return evmc.Berlin, nil
	case tosca.R10_London:
		return evmc.London, nil
	case tosca.R11_Paris:
		return evmc.Paris, nil
	case tosca.R12_Shanghai:
		return evmc.Shanghai, nil
	case tosca.R13_Cancun:
		return evmc.Cancun, nil
	default:
		return 0, fmt.Errorf("unsupported revision: %v", revision)
	}
}

// hostContext allows a non-Go Interpreter implementation to access transaction
// context and chain state information external to the the interpreter.
// It implements the host interface of evmc's Go bindings.
type hostContext struct {
	params         tosca.Parameters
	context        tosca.RunContext
	evmcBlobHashes []evmc.Hash
}

func (ctx *hostContext) AccountExists(addr evmc.Address) bool {
	// Although the EVMC function name asks for the existence of an account,
	// it is actually referring to the emptiness of an account. The concept
	// of an existing or non-existing account is a DB concept that is not
	// exposed to any interpreter implementation.
	return !ctx.isEmpty(addr)
}

func (ctx *hostContext) isEmpty(addr evmc.Address) bool {
	return (ctx.context.GetNonce(tosca.Address(addr)) == 0 &&
		ctx.context.GetBalance(tosca.Address(addr)) == tosca.Value{} &&
		ctx.context.GetCodeSize(tosca.Address(addr)) == 0)
}

func (ctx *hostContext) GetStorage(addr evmc.Address, key evmc.Hash) evmc.Hash {
	return evmc.Hash(ctx.context.GetStorage(tosca.Address(addr), tosca.Key(key)))
}

func (ctx *hostContext) SetStorage(addr evmc.Address, key evmc.Hash, value evmc.Hash) evmc.StorageStatus {
	status := ctx.context.SetStorage(tosca.Address(addr), tosca.Key(key), tosca.Word(value))
	switch status {
	case tosca.StorageAssigned:
		return evmc.StorageAssigned
	case tosca.StorageAdded:
		return evmc.StorageAdded
	case tosca.StorageDeleted:
		return evmc.StorageDeleted
	case tosca.StorageModified:
		return evmc.StorageModified
	case tosca.StorageDeletedAdded:
		return evmc.StorageDeletedAdded
	case tosca.StorageModifiedDeleted:
		return evmc.StorageModifiedDeleted
	case tosca.StorageDeletedRestored:
		return evmc.StorageDeletedRestored
	case tosca.StorageAddedDeleted:
		return evmc.StorageAddedDeleted
	case tosca.StorageModifiedRestored:
		return evmc.StorageModifiedRestored
	default:
		panic(fmt.Sprintf("unsupported storage state: %v", status))
	}
}

func (ctx *hostContext) GetTransientStorage(addr evmc.Address, key evmc.Hash) evmc.Hash {
	return evmc.Hash(ctx.context.GetTransientStorage(tosca.Address(addr), tosca.Key(key)))
}

func (ctx *hostContext) SetTransientStorage(addr evmc.Address, key evmc.Hash, value evmc.Hash) {
	ctx.context.SetTransientStorage(tosca.Address(addr), tosca.Key(key), tosca.Word(value))
}

func (ctx *hostContext) GetBalance(addr evmc.Address) evmc.Hash {
	return evmc.Hash(ctx.context.GetBalance(tosca.Address(addr)))
}

func (ctx *hostContext) GetCodeSize(addr evmc.Address) int {
	return ctx.context.GetCodeSize(tosca.Address(addr))
}

func (ctx *hostContext) GetCodeHash(addr evmc.Address) evmc.Hash {
	if ctx.isEmpty(addr) {
		return evmc.Hash{}
	}
	return evmc.Hash(ctx.context.GetCodeHash(tosca.Address(addr)))
}

func (ctx *hostContext) GetCode(addr evmc.Address) []byte {
	return ctx.context.GetCode(tosca.Address(addr))
}

func (ctx *hostContext) Selfdestruct(addr evmc.Address, beneficiary evmc.Address) bool {
	return ctx.context.SelfDestruct(tosca.Address(addr), tosca.Address(beneficiary))
}

func (ctx *hostContext) GetTxContext() evmc.TxContext {
	params := ctx.params
	return evmc.TxContext{
		GasPrice:    evmc.Hash(params.GasPrice),
		Origin:      evmc.Address(params.Origin),
		Coinbase:    evmc.Address(params.Coinbase),
		Number:      params.BlockNumber,
		Timestamp:   params.Timestamp,
		GasLimit:    int64(params.GasLimit),
		PrevRandao:  evmc.Hash(params.PrevRandao),
		ChainID:     evmc.Hash(params.ChainID),
		BaseFee:     evmc.Hash(params.BaseFee),
		BlobBaseFee: evmc.Hash(params.BlobBaseFee),
		BlobHashes:  ctx.evmcBlobHashes,
	}
}

func (ctx *hostContext) GetBlockHash(number int64) evmc.Hash {
	return evmc.Hash(ctx.context.GetBlockHash(number))
}

func (ctx *hostContext) EmitLog(addr evmc.Address, topics_in []evmc.Hash, data []byte) {
	topics := make([]tosca.Hash, len(topics_in))
	for i := range topics {
		topics[i] = tosca.Hash(topics_in[i])
	}
	ctx.context.EmitLog(tosca.Log{
		Address: tosca.Address(addr),
		Topics:  topics,
		Data:    data,
	})
}

func (ctx *hostContext) Call(kind evmc.CallKind, recipient evmc.Address, sender evmc.Address, value evmc.Hash, input []byte, gas int64, depth int, static bool, salt evmc.Hash, codeAddress evmc.Address) (output []byte, gasLeft int64, gasRefund int64, createAddr evmc.Address, err error) {

	var callKind tosca.CallKind
	switch kind {
	case evmc.Create:
		callKind = tosca.Create
	case evmc.Create2:
		callKind = tosca.Create2
	case evmc.Call:
		callKind = tosca.Call
		if static {
			callKind = tosca.StaticCall
		}
	case evmc.CallCode:
		callKind = tosca.CallCode
	case evmc.DelegateCall:
		callKind = tosca.DelegateCall
	default:
		panic(fmt.Sprintf("unsupported call kind: %v", kind))
	}

	params := tosca.CallParameters{
		Sender:      tosca.Address(sender),
		Recipient:   tosca.Address(recipient),
		Value:       tosca.Value(value),
		Input:       input,
		Gas:         tosca.Gas(gas),
		Salt:        tosca.Hash(salt),
		CodeAddress: tosca.Address(codeAddress),
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
	if ctx.context.AccessAccount(tosca.Address(addr)) == tosca.WarmAccess {
		return evmc.WarmAccess
	}
	return evmc.ColdAccess
}

func (ctx *hostContext) AccessStorage(addr evmc.Address, key evmc.Hash) evmc.AccessStatus {
	if ctx.context.AccessStorage(tosca.Address(addr), tosca.Key(key)) == tosca.WarmAccess {
		return evmc.WarmAccess
	}
	return evmc.ColdAccess
}
