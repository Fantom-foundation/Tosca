// Copyright (c) 2024 Fantom Foundation
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at fantom.foundation/bsl11.
//
// Change Date: 2028-4-16
//
// On the date above, in accordance with the Business Source License, use of
// this software will be governed by the GNU Lesser General Public License v3.

package interpreter_test

import (
	_ "github.com/Fantom-foundation/Tosca/go/interpreter/evmone"
	_ "github.com/Fantom-foundation/Tosca/go/interpreter/evmzero"
	_ "github.com/Fantom-foundation/Tosca/go/interpreter/lfvm"
	"github.com/Fantom-foundation/Tosca/go/tosca"
)

//go:generate mockgen -source test_evm.go -destination test_evm_mock.go -package interpreter_test

const InitialTestGas tosca.Gas = 1 << 44

var TestEvmCreatedAccountAddress = tosca.Address{42}

// TestEVM is a minimal EVM implementation wrapping an Interpreter into an EVM
// instance capable of processing recursive calls. It is only intended to be
// utilized for integration tests in this package, and thus misses almost all
// features of a fully functional EVM.
type TestEVM struct {
	interpreter tosca.Interpreter
	revision    tosca.Revision
	state       StateDB
	depth       int
	readOnly    bool
}

func GetCleanEVM(revision Revision, interpreter string, stateDB StateDB) TestEVM {
	rev := tosca.R07_Istanbul
	switch revision {
	case Istanbul:
		rev = tosca.R07_Istanbul
	case Berlin:
		rev = tosca.R09_Berlin
	case London:
		rev = tosca.R10_London
	}

	instance, err := tosca.NewInterpreter(interpreter)
	if err != nil {
		panic(err)
	}

	return TestEVM{
		interpreter: instance,
		revision:    rev,
		state:       stateDB,
	}
}

// StateDB is a TestEVM interface that is mocked by tests to formulate
// expectations on chain-state side-effects of interpreter operations.
type StateDB interface {
	AccountExists(tosca.Address) bool
	GetStorage(tosca.Address, tosca.Key) tosca.Word
	SetStorage(tosca.Address, tosca.Key, tosca.Word)
	GetTransientStorage(tosca.Address, tosca.Key) tosca.Word
	SetTransientStorage(tosca.Address, tosca.Key, tosca.Word)
	GetBalance(tosca.Address) tosca.Value
	SetBalance(tosca.Address, tosca.Value)
	GetNonce(tosca.Address) uint64
	SetNonce(tosca.Address, uint64)
	GetCodeSize(tosca.Address) int
	GetCodeHash(tosca.Address) tosca.Hash
	GetCode(tosca.Address) tosca.Code
	SetCode(tosca.Address, tosca.Code)
	GetBlockHash(int64) tosca.Hash
	EmitLog(tosca.Log)
	AccessAccount(tosca.Address) tosca.AccessStatus
	AccessStorage(tosca.Address, tosca.Key) tosca.AccessStatus
	GetCommittedStorage(tosca.Address, tosca.Key) tosca.Word
	IsAddressInAccessList(tosca.Address) bool
	IsSlotInAccessList(tosca.Address, tosca.Key) (addressPresent, slotPresent bool)
	HasSelfDestructed(tosca.Address) bool
}
type RunResult struct {
	Output  []byte
	GasUsed tosca.Gas
	Success bool
}

func (e *TestEVM) Run(code []byte, input []byte) (RunResult, error) {
	return e.RunWithGas(code, input, InitialTestGas)
}

func (e *TestEVM) RunWithGas(code []byte, input []byte, initialGas tosca.Gas) (RunResult, error) {
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

func (e *TestEVM) runInternal(code []byte, input []byte, gas tosca.Gas, readOnly bool) (tosca.Result, error) {

	params := tosca.Parameters{
		BlockParameters: tosca.BlockParameters{
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

// runContextAdapter is an internal implementation of the tosca.RunContext mapping operations
// to the TestEVM and its StateDB interface to be implemented by tests, mostly through mocks.
type runContextAdapter struct {
	StateDB
	evm *TestEVM
}

func (a *runContextAdapter) SetStorage(addr tosca.Address, key tosca.Key, newValue tosca.Word) tosca.StorageStatus {
	stateDB := a.StateDB
	currentValue := stateDB.GetStorage(addr, key)
	if currentValue == newValue {
		return tosca.StorageAssigned
	}
	stateDB.SetStorage(addr, key, newValue)

	originalValue := stateDB.GetCommittedStorage(addr, key)
	return tosca.GetStorageStatus(originalValue, currentValue, newValue)
}

func (a *runContextAdapter) GetTransactionContext() tosca.TransactionParameters {
	return tosca.TransactionParameters{}
}

func (a *runContextAdapter) Call(kind tosca.CallKind, parameter tosca.CallParameters) (tosca.CallResult, error) {
	// This is a simple implementation of an EVM handling recursive calls for tests.
	// A full implementation would need to consider additional side-effects of calls
	// like the transfer of values, StateDB snapshots, and precompiled contracts.

	// Check the maximum nesting depth, tracked by the EVM, not the interpreter.
	if a.evm.depth >= 1024 {
		return tosca.CallResult{
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
	case tosca.Create, tosca.Create2:
		code = parameter.Input
	case tosca.Call, tosca.StaticCall:
		code = a.GetCode(parameter.Recipient)
	case tosca.CallCode, tosca.DelegateCall:
		code = a.GetCode(parameter.CodeAddress)
	default:
		panic("not implemented")
	}

	// Switch to read-only mode if this call is a static call.
	// Also this is tracked outside the interpreter implementation.
	if kind == tosca.StaticCall && !a.evm.readOnly {
		a.evm.readOnly = true
		defer func() {
			a.evm.readOnly = false
		}()
	}

	result, err := a.evm.runInternal(code, parameter.Input, parameter.Gas, a.evm.readOnly)
	if err != nil {
		return tosca.CallResult{}, err
	}

	// Charge extra costs for creating the contract -- 200 gas per byte.
	if (kind == tosca.Create || kind == tosca.Create2) && result.Success {
		initCodeCost := tosca.Gas(200 * len(result.Output))
		if result.GasLeft < initCodeCost {
			return tosca.CallResult{Success: false}, nil
		}
		result.GasLeft -= initCodeCost
	}

	return tosca.CallResult{
		Output:         result.Output,
		GasLeft:        result.GasLeft,
		GasRefund:      result.GasRefund,
		CreatedAddress: TestEvmCreatedAccountAddress,
		Success:        result.Success,
	}, err
}

func (a *runContextAdapter) SelfDestruct(address tosca.Address, beneficiary tosca.Address) bool {
	beneficiaryEmpty := a.GetBalance(beneficiary) == (tosca.Value{}) &&
		a.GetNonce(beneficiary) == 0 &&
		a.GetCodeSize(beneficiary) == 0
	if beneficiaryEmpty {
		return false
	}
	balance := a.GetBalance(address)
	return balance != (tosca.Value{})
}

func (a *runContextAdapter) CreateAccount(tosca.Address, tosca.Code) bool {
	panic("should not be needed for interpreter tests")
}

func (a *runContextAdapter) CreateSnapshot() tosca.Snapshot {
	// ignored in interpreter tests
	return 0
}

func (a *runContextAdapter) RestoreSnapshot(tosca.Snapshot) {
	// ignored in interpreter tests
}

func (a *runContextAdapter) GetLogs() []tosca.Log {
	panic("should not be needed for interpreter tests")
}
