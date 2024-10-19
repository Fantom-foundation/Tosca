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
	"fmt"
	"math/big"
	"testing"

	"github.com/Fantom-foundation/Tosca/go/tosca"
	"github.com/Fantom-foundation/Tosca/go/tosca/vm"
	"go.uber.org/mock/gomock"
)

func TestStaticGas(t *testing.T) {
	// For every variant of interpreter
	for _, variant := range getAllInterpreterVariantsForTests() {
		for _, revision := range revisions {
			// Get static gas for frequently used instructions
			pushGas := getInstructions(revision)[vm.PUSH1].gas.static
			jumpdestGas := getInstructions(revision)[vm.JUMPDEST].gas.static

			for op, info := range getInstructions(revision) {
				if info.gas.dynamic == nil {
					t.Run(fmt.Sprintf("%s/%s/%s", variant, revision, op), func(t *testing.T) {
						ctrl := gomock.NewController(t)
						mockStateDB := NewMockStateDB(ctrl)
						mockStateDB.EXPECT().GetStorage(gomock.Any(), gomock.Any()).AnyTimes().Return(tosca.Word{})
						mockStateDB.EXPECT().GetBalance(gomock.Any()).AnyTimes().Return(tosca.Value{})
						mockStateDB.EXPECT().GetNonce(gomock.Any()).AnyTimes().Return(uint64(0))
						mockStateDB.EXPECT().GetCodeSize(gomock.Any()).AnyTimes().Return(0)
						mockStateDB.EXPECT().AccountExists(gomock.Any()).AnyTimes().Return(true)
						mockStateDB.EXPECT().GetCodeHash(gomock.Any()).AnyTimes().Return(tosca.Hash{})
						mockStateDB.EXPECT().GetBlockHash(gomock.Any()).AnyTimes().Return(tosca.Hash{})

						evm := GetCleanEVM(revision, variant, mockStateDB)
						var wantGas tosca.Gas = 0
						var code []byte
						if op == vm.JUMP {
							code = []byte{
								byte(vm.PUSH1),
								byte(3),
								byte(op),
								byte(vm.JUMPDEST),
							}
							wantGas = pushGas + info.gas.static + jumpdestGas
						} else {
							// Fill stack with PUSH1 instructions.
							codeLen := info.stack.popped*2 + 1
							code = make([]byte, 0, codeLen)
							for i := 0; i < info.stack.popped; i++ {
								code = append(code, []byte{byte(vm.PUSH1), 0}...)
								wantGas += pushGas
							}

							// Set a tested instruction as the last one.
							code = append(code, byte(op))
							wantGas += info.gas.static
						}

						// Run an interpreter
						result, err := evm.Run(code, []byte{})

						// Check the result.
						if err != nil {
							t.Errorf("execution failed %v should not fail: error is %v", op, err)
						}

						// Check the result.
						if result.GasUsed != wantGas {
							t.Errorf("execution failed %v use wrong amount of gas: was %v, want %v", op, result.GasUsed, wantGas)
						}
					})
				}
			}
		}
	}
}

func TestDynamicGas(t *testing.T) {
	accountBalance := tosca.Value{100}

	// For every variant of interpreter
	for _, variant := range getAllInterpreterVariantsForTests() {
		for _, revision := range revisions {
			// Get static gas for frequently used instructions
			pushGas := getInstructions(revision)[vm.PUSH1].gas.static
			for op, info := range getInstructions(revision) {

				if info.gas.dynamic == nil {
					continue
				}

				for _, testCase := range info.gas.dynamic(revision) {
					t.Run(fmt.Sprintf("%s/%s/%s/%s", variant, revision, op, testCase.testName), func(t *testing.T) {

						mockCtrl := gomock.NewController(t)
						mockStateDB := NewMockStateDB(mockCtrl)

						// World state interactions triggered by the EVM.
						recipient := tosca.Address{}
						mockStateDB.EXPECT().GetNonce(recipient).AnyTimes()
						mockStateDB.EXPECT().GetCodeSize(recipient).AnyTimes()
						mockStateDB.EXPECT().SetNonce(recipient, gomock.Any()).AnyTimes()
						mockStateDB.EXPECT().SetCode(recipient, gomock.Any()).AnyTimes()
						mockStateDB.EXPECT().SetBalance(gomock.Any(), gomock.Any()).AnyTimes()

						// SELFDESTRUCT gas computation is dependent on an account balance and sets its own expectations
						if op != vm.SELFDESTRUCT {
							mockStateDB.EXPECT().GetBalance(recipient).AnyTimes().Return(accountBalance)
						}

						if op == vm.CREATE || op == vm.CREATE2 {
							// Create calls check that the target account is indeed empty.
							mockStateDB.EXPECT().AccountExists(gomock.Any()).AnyTimes().Return(false)
							mockStateDB.EXPECT().GetNonce(gomock.Any()).AnyTimes()
							mockStateDB.EXPECT().GetBalance(gomock.Any()).AnyTimes()
							mockStateDB.EXPECT().GetCodeSize(gomock.Any()).AnyTimes()
							// Also, they may create the account.
							mockStateDB.EXPECT().SetNonce(gomock.Any(), uint64(1)).AnyTimes()
							mockStateDB.EXPECT().SetCode(gomock.Any(), gomock.Any()).AnyTimes()
						}

						// For EXTCODEHASH the targeted account must exist.
						if op == vm.EXTCODEHASH {
							mockStateDB.EXPECT().GetNonce(gomock.Any()).AnyTimes().Return(uint64(1))
						}

						// Init stateDB mock calls from test function
						if testCase.mockCalls != nil {
							testCase.mockCalls(mockStateDB)
						}

						// Initialize EVM clean instance
						evm := GetCleanEVM(revision, variant, mockStateDB)
						var wantGas tosca.Gas = 0
						code := make([]byte, 0)

						// When test need return value from inner call operation
						if op == vm.RETURNDATACOPY {
							gas, returnCode := putCallReturnValue(t, revision, code, mockStateDB)
							wantGas += gas
							code = append(code, returnCode...)
						}

						// If test needs to put values into memory
						memCode, gas := addMemToStack(testCase.memValues, pushGas)
						code = append(code, memCode...)
						wantGas += gas

						// Put needed values on stack with PUSH instructions.
						pushCode, gas := addValuesToStack(testCase.stackValues, pushGas)
						code = append(code, pushCode...)
						wantGas += gas

						// Set a tested instruction as the last one.
						code = append(code, byte(op))
						// Add expected static and dynamic gas for test case
						wantGas += info.gas.static + testCase.expectedGas

						// Run an interpreter
						result, err := evm.Run(code, []byte{})

						// Check the result.
						if err != nil {
							t.Errorf("execution failed %v should not fail: error is %v", op, err)
						}

						// Check the result.
						if result.GasUsed != wantGas {
							t.Errorf("execution failed %v use wrong amount of gas: was %v, want %v", op, result.GasUsed, wantGas)
						}
					})
				}
			}
		}
	}
}

func TestOutOfDynamicGas(t *testing.T) {
	// For every variant of interpreter
	for _, variant := range getAllInterpreterVariantsForTests() {
		for _, revision := range revisions {
			// Get static gas for frequently used instructions
			pushGas := getInstructions(revision)[vm.PUSH1].gas.static

			for _, testCase := range getOutOfDynamicGasTests(revision) {
				t.Run(fmt.Sprintf("%s/%s/%s", variant, revision, testCase.testName), func(t *testing.T) {

					// Need new mock for every testcase
					mockCtrl := gomock.NewController(t)
					mockStateDB := NewMockStateDB(mockCtrl)

					// Init stateDB mock calls from test function
					if testCase.mockCalls != nil {
						testCase.mockCalls(mockStateDB)
					}

					// Initialize EVM clean instance
					evm := GetCleanEVM(revision, variant, mockStateDB)
					code := make([]byte, 0)

					// Put needed values on stack with PUSH instructions.
					pushCode, pushGasAdded := addValuesToStack(testCase.stackValues, pushGas)
					code = append(code, pushCode...)

					// Set a tested instruction as the last one.
					code = append(code, byte(testCase.instruction))

					// Run an interpreter
					res, err := evm.RunWithGas(code, []byte{}, testCase.initialGas+pushGasAdded)

					// Check the result.
					if err != nil {
						t.Fatalf("failed to run test code: %v", err)
					}
					if res.Success {
						t.Errorf("execution should have failed due to too little gas, got %v", res)
					}
				})
			}
		}
	}
}

func TestOutOfStaticGasOnly(t *testing.T) {
	// For every variant of interpreter
	for _, variant := range getAllInterpreterVariantsForTests() {
		for _, revision := range revisions {
			// Get static gas for frequently used instructions
			pushGas := getInstructions(revision)[vm.PUSH1].gas.static
			for op, info := range getInstructions(revision) {

				if info.gas.static == 0 || info.gas.dynamic != nil {
					continue
				}

				t.Run(fmt.Sprintf("%s/%s/%s", variant, revision, op), func(t *testing.T) {

					// Initialize EVM clean instance
					evm := GetCleanEVM(revision, variant, nil)
					code := make([]byte, 0)

					// Put needed values on stack with PUSH instructions.
					stackValues := make([]*big.Int, 0)
					for i := 0; i < info.stack.popped; i++ {
						stackValues = append(stackValues, big.NewInt(1))
					}
					pushCode, needGas := addValuesToStack(stackValues, pushGas)
					code = append(code, pushCode...)

					// Set a tested instruction as the last one.
					code = append(code, byte(op))

					// Run an interpreter with gas set to fail
					res, err := evm.RunWithGas(code, []byte{}, info.gas.static+needGas-1)

					// Check the result.
					if err != nil {
						t.Fatalf("failed to run test code: %v", err)
					}
					if res.Success {
						t.Errorf("execution should have failed due to too little gas, got %v", res)
					}
				})
			}
		}
	}
}

func addValuesToStack(stackValues []*big.Int, pushGas tosca.Gas) ([]byte, tosca.Gas) {
	stackValuesCount := len(stackValues)

	var (
		code    []byte
		wantGas tosca.Gas
	)

	for i := 0; i < stackValuesCount; i++ {
		code, wantGas = addBytesToStack(stackValues[i].Bytes(), code, wantGas, pushGas)
	}
	return code, wantGas
}

func addMemToStack(stackValues []*big.Int, pushGas tosca.Gas) ([]byte, tosca.Gas) {
	stackValuesCount := len(stackValues)

	var (
		code    []byte
		wantGas tosca.Gas
	)

	for i := 0; i < stackValuesCount; i += 2 {
		code, wantGas = addBytesToStack(stackValues[i].Bytes(), code, wantGas, pushGas)
		code, wantGas = addBytesToStack(stackValues[i+1].Bytes(), code, wantGas, pushGas)
		code = append(code, byte(vm.MSTORE))
		wantGas += memoryExpansionGasCost(32)
	}
	return code, wantGas
}

func addBytesToStack(valueBytes []byte, code []byte, wantGas tosca.Gas, pushGas tosca.Gas) ([]byte, tosca.Gas) {
	if len(valueBytes) == 0 {
		valueBytes = []byte{0}
	}
	push := vm.PUSH1 + vm.OpCode(len(valueBytes)-1)
	code = append(code, byte(push))
	for j := 0; j < len(valueBytes); j++ {
		code = append(code, valueBytes[j])
	}
	wantGas += pushGas
	return code, wantGas
}

// Returns computed gas for calling passed callCode with a Call instruction
func getCallInstructionGas(t *testing.T, revision Revision, callCode []byte) tosca.Gas {
	accountNumber := 99
	account := tosca.Address{byte(accountNumber)}
	code := make([]byte, 0)
	zeroVal := big.NewInt(0)
	gasSentWithCall := big.NewInt(100000)
	if len(callCode) == 0 {
		callCode = []byte{byte(vm.STOP)}
	}
	mockCtrl := gomock.NewController(t)
	mockStateDB := NewMockStateDB(mockCtrl)
	mockStateDB.EXPECT().AccountExists(account).AnyTimes().Return(true)
	mockStateDB.EXPECT().GetBalance(gomock.Any()).AnyTimes().Return(tosca.Value{})
	mockStateDB.EXPECT().SetBalance(gomock.Any(), gomock.Any()).AnyTimes().Return()
	mockStateDB.EXPECT().GetCode(account).AnyTimes().Return(callCode)
	mockStateDB.EXPECT().GetCodeHash(account).AnyTimes()
	mockStateDB.EXPECT().IsAddressInAccessList(account).AnyTimes().Return(true)

	// Minimum stack values to execute CALL instruction
	stackValues := []*big.Int{zeroVal, zeroVal, zeroVal, zeroVal, zeroVal, addressToBigInt(account), gasSentWithCall}

	evm := GetCleanEVM(revision, "geth", mockStateDB)

	for i := 0; i < len(stackValues); i++ {
		valueBytes := stackValues[i].Bytes()
		if len(valueBytes) == 0 {
			valueBytes = []byte{0}
		}
		push := vm.PUSH1 + vm.OpCode(len(valueBytes)-1)
		code = append(code, byte(push))
		for j := 0; j < len(valueBytes); j++ {
			code = append(code, valueBytes[j])
		}
	}

	// Set a CALL instruction as the last one.
	code = append(code, byte(vm.CALL))

	result, err := evm.Run(code, []byte{})
	if err != nil {
		return 0
	}

	return result.GasUsed
}

// Processes call which ends with a return value, so it is put into the memory of the EVM
func putCallReturnValue(t *testing.T, revision Revision, code []byte, mockStateDB *MockStateDB) (gas tosca.Gas, returnCode []byte) {
	accountNumber := 100
	account := tosca.Address{byte(accountNumber)}

	// Code processed inside inner call
	codeWithReturnValue := []byte{
		byte(vm.PUSH1),
		byte(0),
		byte(vm.PUSH1),
		byte(1),
		byte(vm.MSTORE),
		byte(vm.PUSH2),
		byte(255),
		byte(255),
		byte(vm.PUSH1),
		byte(0),
		byte(vm.RETURN)}
	mockStateDB.EXPECT().AccountExists(account).AnyTimes().Return(true)
	mockStateDB.EXPECT().GetBalance(gomock.Any()).AnyTimes().Return(tosca.Value{})
	mockStateDB.EXPECT().GetCode(account).AnyTimes().Return(codeWithReturnValue)
	mockStateDB.EXPECT().GetCodeHash(account).AnyTimes().Return(tosca.Hash{byte(accountNumber)})
	mockStateDB.EXPECT().IsAddressInAccessList(account).AnyTimes().Return(true)
	mockStateDB.EXPECT().AccessAccount(account).AnyTimes().Return(tosca.WarmAccess)
	// Get needed gas from a CALL execution for this code and revision
	gas = getCallInstructionGas(t, revision, codeWithReturnValue)

	zeroVal := big.NewInt(0)
	stackCallValues := []*big.Int{zeroVal, zeroVal, zeroVal, zeroVal, zeroVal, addressToBigInt(account), big.NewInt(int64(gas))}

	for i := 0; i < len(stackCallValues); i++ {
		valueBytes := stackCallValues[i].Bytes()
		if len(valueBytes) == 0 {
			valueBytes = []byte{0}
		}
		push := vm.PUSH1 + vm.OpCode(len(valueBytes)-1)
		code = append(code, byte(push))
		for j := 0; j < len(valueBytes); j++ {
			code = append(code, valueBytes[j])
		}
	}
	returnCode = append(code, byte(vm.CALL))
	return gas, returnCode
}
