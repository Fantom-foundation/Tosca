// Copyright (c) 2024 Fantom Foundation
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at fantom.foundation/bsl11.
//
// Change Date: 2028-4-16
//
// On the date above, in accordance with the Business Source License, use of
// this software will be governed by the GNU Lesser General Public License v3.

package vm_test

import (
	"github.com/Fantom-foundation/Tosca/go/vm"
	_ "github.com/Fantom-foundation/Tosca/go/vm/evmone"
	_ "github.com/Fantom-foundation/Tosca/go/vm/evmzero"
	_ "github.com/Fantom-foundation/Tosca/go/vm/lfvm"
)

//go:generate mockgen -source test_evm.go -destination test_evm_mock.go -package vm_test

const InitialTestGas vm.Gas = 1 << 44

var TestEvmCreatedAccountAddress = vm.Address{42}

// TestEVM is a minimal EVM implementation wrapping an Interpreter into an EVM
// instance capable of processing recursive calls. It is only intended to be
// utilized for integration tests in this package, and thus misses almost all
// features of a fully functional EVM.
type TestEVM struct {
	interpreter vm.Interpreter
	revision    vm.Revision
	state       StateDB
	depth       int
	readOnly    bool
}

func GetCleanEVM(revision Revision, interpreter string, stateDB StateDB) TestEVM {
	rev := vm.R07_Istanbul
	switch revision {
	case Istanbul:
		rev = vm.R07_Istanbul
	case Berlin:
		rev = vm.R09_Berlin
	case London:
		rev = vm.R10_London
	}

	return TestEVM{
		interpreter: vm.GetInterpreter(interpreter),
		revision:    rev,
		state:       stateDB,
	}
}

// StateDB is a TestEVM interface that is mocked by tests to formulate
// expectations on chain-state side-effects of interpreter operations.
type StateDB interface {
	AccountExists(vm.Address) bool
	GetStorage(vm.Address, vm.Key) vm.Word
	SetStorage(vm.Address, vm.Key, vm.Word)
	GetTransientStorage(vm.Address, vm.Key) vm.Word
	SetTransientStorage(vm.Address, vm.Key, vm.Word)
	GetBalance(vm.Address) vm.Value
	SetBalance(vm.Address, vm.Value)
	GetNonce(vm.Address) uint64
	SetNonce(vm.Address, uint64)
	GetCodeSize(vm.Address) int
	GetCodeHash(vm.Address) vm.Hash
	GetCode(vm.Address) vm.Code
	SetCode(vm.Address, vm.Code)
	GetBlockHash(int64) vm.Hash
	EmitLog(vm.Log)
	AccessAccount(vm.Address) vm.AccessStatus
	AccessStorage(vm.Address, vm.Key) vm.AccessStatus
	GetCommittedStorage(vm.Address, vm.Key) vm.Word
	IsAddressInAccessList(vm.Address) bool
	IsSlotInAccessList(vm.Address, vm.Key) (addressPresent, slotPresent bool)
	HasSelfDestructed(vm.Address) bool
}
type RunResult struct {
	Output  []byte
	GasUsed vm.Gas
	Success bool
}

func (e *TestEVM) Run(code []byte, input []byte) (RunResult, error) {
	return e.RunWithGas(code, input, InitialTestGas)
}

func (e *TestEVM) RunWithGas(code []byte, input []byte, initialGas vm.Gas) (RunResult, error) {
	result, err := e.runInternal(code, input, initialGas, false)
	if err != nil {
		return RunResult{}, err
	}

	return RunResult{
		Output:  result.Output,
		GasUsed: InitialTestGas - result.GasLeft,
		Success: result.Success,
	}, nil
}

func (e *TestEVM) runInternal(code []byte, input []byte, gas vm.Gas, readOnly bool) (vm.Result, error) {

	params := vm.Parameters{
		BlockParameters: vm.BlockParameters{
			Revision: e.revision,
		},
		Context: &runContextAdapter{
			StateDB: e.state,
			evm:     e,
		},
		Code:   code,
		Input:  input,
		Gas:    gas,
		Depth:  0,
		Static: readOnly,
	}

	return e.interpreter.Run(params)
}

// --- adapter ---

// runContextAdapter is an internal implementation of the vm.RunContext mapping operations
// to the TestEVM and its StateDB interface to be implemented by tests, mostly through mocks.
type runContextAdapter struct {
	StateDB
	evm *TestEVM
}

func (a *runContextAdapter) SetStorage(addr vm.Address, key vm.Key, newValue vm.Word) vm.StorageStatus {
	stateDB := a.StateDB
	currentValue := stateDB.GetStorage(addr, key)
	if currentValue == newValue {
		return vm.StorageAssigned
	}
	stateDB.SetStorage(addr, key, newValue)

	originalValue := stateDB.GetCommittedStorage(addr, key)
	return vm.GetStorageStatus(originalValue, currentValue, newValue)
}

func (a *runContextAdapter) GetTransactionContext() vm.TransactionParameters {
	return vm.TransactionParameters{}
}

func (a *runContextAdapter) Call(kind vm.CallKind, parameter vm.CallParameters) (vm.CallResult, error) {
	// This is a simple implementation of an EVM handling recursive calls for tests.
	// A full implementation would need to consider additional side-effects of calls
	// like the transfer of values, StateDB snapshots, and precompiled contracts.

	// Check the maximum nesting depth, tracked by the EVM, not the interpreter.
	if a.evm.depth >= 1024 {
		return vm.CallResult{
			Success: false,
		}, nil
	}
	a.evm.depth++
	defer func() {
		a.evm.depth--
	}()

	// Get code to be executed.
	var code []byte
	switch kind {
	case vm.Create, vm.Create2:
		code = parameter.Input
	case vm.Call, vm.StaticCall:
		code = a.GetCode(parameter.Recipient)
	case vm.CallCode, vm.DelegateCall:
		code = a.GetCode(parameter.CodeAddress)
	default:
		panic("not implemented")
	}

	// Switch to read-only mode if this call is a static call.
	// Also this is tracked outside the interpreter implementation.
	if kind == vm.StaticCall && !a.evm.readOnly {
		a.evm.readOnly = true
		defer func() {
			a.evm.readOnly = false
		}()
	}

	result, err := a.evm.runInternal(code, parameter.Input, parameter.Gas, a.evm.readOnly)
	if err != nil {
		return vm.CallResult{}, err
	}

	// Charge extra costs for creating the contract -- 200 gas per byte.
	if (kind == vm.Create || kind == vm.Create2) && result.Success {
		initCodeCost := vm.Gas(200 * len(result.Output))
		if result.GasLeft < initCodeCost {
			return vm.CallResult{Success: false}, nil
		}
		result.GasLeft -= initCodeCost
	}

	return vm.CallResult{
		Output:         result.Output,
		GasLeft:        result.GasLeft,
		GasRefund:      result.GasRefund,
		CreatedAddress: TestEvmCreatedAccountAddress,
		Success:        result.Success,
	}, err
}

func (a *runContextAdapter) SelfDestruct(address vm.Address, beneficiary vm.Address) bool {
	if a.AccountExists(beneficiary) {
		return false
	}
	balance := a.GetBalance(address)
	return balance != (vm.Value{})
}

func (a *runContextAdapter) CreateAccount(vm.Address, vm.Code) bool {
	panic("should not be needed for interpreter tests")
}

func (a *runContextAdapter) CreateSnapshot() vm.Snapshot {
	// ignored in interpreter tests
	return 0
}

func (a *runContextAdapter) RestoreSnapshot(vm.Snapshot) {
	// ignored in interpreter tests
}

func (a *runContextAdapter) GetLogs() []vm.Log {
	panic("should not be needed for interpreter tests")
}
