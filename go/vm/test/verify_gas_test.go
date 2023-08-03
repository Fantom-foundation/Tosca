package vm_test

import (
	"fmt"
	"math/big"
	"testing"

	vm_mock "github.com/Fantom-foundation/Tosca/go/vm/test/mocks"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/golang/mock/gomock"
)

func TestStaticGas(t *testing.T) {
	var mockCtrl *gomock.Controller
	var mockStateDB *vm_mock.MockStateDB
	mockCtrl = gomock.NewController(t)
	mockStateDB = vm_mock.NewMockStateDB(mockCtrl)
	mockStateDB.EXPECT().GetState(gomock.Any(), gomock.Any()).AnyTimes().Return(common.Hash{0})
	mockStateDB.EXPECT().GetBalance(gomock.Any()).AnyTimes().Return(big.NewInt(0))
	mockStateDB.EXPECT().GetCodeSize(gomock.Any()).AnyTimes().Return(0)
	mockStateDB.EXPECT().Empty(gomock.Any()).AnyTimes().Return(true)
	// evmone needs following in addition to geth and lfvm
	mockStateDB.EXPECT().GetRefund().AnyTimes().Return(uint64(0))
	mockStateDB.EXPECT().SubRefund(gomock.Any()).AnyTimes()
	mockStateDB.EXPECT().AddRefund(gomock.Any()).AnyTimes()
	mockStateDB.EXPECT().GetCodeHash(gomock.Any()).AnyTimes().Return(common.Hash{0})

	// For every variant of interpreter
	for _, variant := range Variants {
		for _, revision := range revisions {
			// Get staic gas for frequently used instructions
			pushGas := getInstructions(revision)[vm.PUSH1].gas.static
			jumpdestGas := getInstructions(revision)[vm.JUMPDEST].gas.static

			for op, info := range getInstructions(revision) {
				if info.gas.dynamic == nil {
					t.Run(fmt.Sprintf("%s/%s/%s", variant, revision, op), func(t *testing.T) {
						evm := GetCleanEVM(revision, variant, mockStateDB)
						var wantGas uint64 = 0
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
						if result.GasUsed != uint64(wantGas) {
							t.Errorf("execution failed %v use wrong amount of gas: was %v, want %v", op, result.GasUsed, wantGas)
						}
					})
				}
			}
		}
	}
}

func TestDynamicGas(t *testing.T) {
	account := common.Address{0}
	accountBalance := big.NewInt(1000)
	var mockCtrl *gomock.Controller
	var mockStateDB *vm_mock.MockStateDB

	// For every variant of interpreter
	for _, variant := range Variants {
		for _, revision := range revisions {
			// Get static gas for frequently used instructions
			pushGas := getInstructions(revision)[vm.PUSH1].gas.static
			for op, info := range getInstructions(revision) {

				if info.gas.dynamic == nil {
					continue
				}

				for _, testCase := range info.gas.dynamic(revision) {

					t.Run(fmt.Sprintf("%s/%s/%s/%s", variant, revision, op, testCase.testName), func(t *testing.T) {

						// Need new mock for every testcase
						mockCtrl = gomock.NewController(t)
						mockStateDB = vm_mock.NewMockStateDB(mockCtrl)

						// evmone needs following in addition to geth and lfvm
						mockStateDB.EXPECT().GetRefund().AnyTimes().Return(uint64(0))
						mockStateDB.EXPECT().SubRefund(uint64(0)).AnyTimes()

						// SELFDESTRUCT gas computation is dependent on an account balance
						if op != vm.SELFDESTRUCT {
							mockStateDB.EXPECT().AddRefund(uint64(0)).AnyTimes()
							mockStateDB.EXPECT().GetBalance(account).AnyTimes().Return(accountBalance)
						}

						// Init stateDB mock calls from test function
						if testCase.mockCalls != nil {
							testCase.mockCalls(mockStateDB)
						}

						// Initialize EVM clean instance
						evm := GetCleanEVM(revision, variant, mockStateDB)
						var wantGas uint64 = 0
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
						if err != nil && op != vm.REVERT {
							t.Errorf("execution failed %v should not fail: error is %v", op, err)
						} else if op == vm.REVERT && err != vm.ErrExecutionReverted {
							t.Errorf("execution of %v should fail with ErrExecutionReverted: error is %v", op, err)
						}

						// Check the result.
						if result.GasUsed != uint64(wantGas) {
							t.Errorf("execution failed %v use wrong amount of gas: was %v, want %v", op, result.GasUsed, wantGas)
						}
					})
				}
			}
		}
	}
}

func TestOutOfGas(t *testing.T) {
	var mockCtrl *gomock.Controller
	var mockStateDB *vm_mock.MockStateDB

	// For every variant of interpreter
	for _, variant := range Variants {
		for _, revision := range revisions {
			// Get static gas for frequently used instructions
			pushGas := getInstructions(revision)[vm.PUSH1].gas.static

			for _, testCase := range getOutOfGasTests(revision) {

				t.Run(fmt.Sprintf("%s/%s/%s", variant, revision, testCase.testName), func(t *testing.T) {

					// Need new mock for every testcase
					mockCtrl = gomock.NewController(t)
					mockStateDB = vm_mock.NewMockStateDB(mockCtrl)

					// evmone needs following in addition to geth and lfvm
					mockStateDB.EXPECT().GetRefund().AnyTimes().Return(uint64(0))
					mockStateDB.EXPECT().SubRefund(uint64(0)).AnyTimes()

					// Init stateDB mock calls from test function
					if testCase.mockCalls != nil {
						testCase.mockCalls(mockStateDB)
					}

					// Initialize EVM clean instance
					evm := GetCleanEVM(revision, variant, mockStateDB)
					code := make([]byte, 0)

					// Put needed values on stack with PUSH instructions.
					pushCode, _ := addValuesToStack(testCase.stackValues, pushGas)
					code = append(code, pushCode...)

					// Set a tested instruction as the last one.
					code = append(code, byte(testCase.instruction))

					// Run an interpreter
					_, err := evm.RunWithGas(code, []byte{}, testCase.initialGas)

					// Check the result.
					if err != nil && err != testCase.expectedError {
						t.Errorf("execution failed %v should fail with %v but got error: %v", testCase.testName, testCase.expectedError, err)
					} else if err == nil {
						t.Errorf("execution of %v should fail with ErrOutOfGas but there is no error", testCase.testName)
					}
				})
			}
		}
	}
}

func TestOutOfStaticGasOnly(t *testing.T) {
	// For every variant of interpreter
	for _, variant := range Variants {
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
					_, err := evm.RunWithGas(code, []byte{}, info.gas.static+needGas-1)

					// Check the result.
					if err != nil && err != vm.ErrOutOfGas {
						t.Errorf("execution should fail with %v but got error: %v", vm.ErrOutOfGas, err)
					} else if err == nil {
						t.Errorf("execution should fail with ErrOutOfGas but there is no error")
					}
				})
			}
		}
	}
}

func addValuesToStack(stackValues []*big.Int, pushGas uint64) ([]byte, uint64) {
	stackValuesCount := len(stackValues)

	var (
		code    []byte
		wantGas uint64
	)

	for i := 0; i < stackValuesCount; i++ {
		code, wantGas = addBytesTostack(stackValues[i].Bytes(), code, wantGas, pushGas)
	}
	return code, wantGas
}

func addMemToStack(stackValues []*big.Int, pushGas uint64) ([]byte, uint64) {
	stackValuesCount := len(stackValues)

	var (
		code    []byte
		wantGas uint64
	)

	for i := 0; i < stackValuesCount; i += 2 {
		code, wantGas = addBytesTostack(stackValues[i].Bytes(), code, wantGas, pushGas)
		code, wantGas = addBytesTostack(stackValues[i+1].Bytes(), code, wantGas, pushGas)
		code = append(code, byte(vm.MSTORE))
		wantGas += memoryExpansionGasCost(32)
	}
	return code, wantGas
}

func addBytesTostack(valueBytes []byte, code []byte, wantGas uint64, pushGas uint64) ([]byte, uint64) {
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
func getCallInstructionGas(t *testing.T, revision Revision, callCode []byte) uint64 {
	accountNumber := 99
	account := common.Address{byte(accountNumber)}
	code := make([]byte, 0)
	zeroVal := big.NewInt(0)
	gasSentWithCall := big.NewInt(100000)
	if len(callCode) == 0 {
		callCode = []byte{byte(vm.STOP)}
	}
	var mockCtrl *gomock.Controller
	var mockStateDB *vm_mock.MockStateDB
	mockCtrl = gomock.NewController(t)
	mockStateDB = vm_mock.NewMockStateDB(mockCtrl)

	// evmone needs following in addition to geth and lfvm
	mockStateDB.EXPECT().GetRefund().AnyTimes().Return(uint64(0))
	mockStateDB.EXPECT().SubRefund(gomock.Any()).AnyTimes()
	mockStateDB.EXPECT().AddRefund(gomock.Any()).AnyTimes()
	mockStateDB.EXPECT().GetCodeHash(account).Return(common.Hash{byte(accountNumber)})
	mockStateDB.EXPECT().Snapshot().AnyTimes().Return(0)
	mockStateDB.EXPECT().Exist(account).AnyTimes().Return(true)
	mockStateDB.EXPECT().GetCode(account).AnyTimes().Return(callCode)
	mockStateDB.EXPECT().AddressInAccessList(account).AnyTimes().Return(true)

	// Minimum stack values to execute CALL instruction
	stackValues := []*big.Int{zeroVal, zeroVal, zeroVal, zeroVal, zeroVal, account.Hash().Big(), gasSentWithCall}

	evm := GetCleanEVM(revision, Variants[0], mockStateDB)

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

// Processes call which ends with a return value, so it is put into the memmory of the EVM
func putCallReturnValue(t *testing.T, revision Revision, code []byte, mockStateDB *vm_mock.MockStateDB) (gas uint64, returnCode []byte) {
	accountNumber := 100
	account := common.Address{byte(accountNumber)}
	gasSentWithCall := big.NewInt(100000)

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
	mockStateDB.EXPECT().Snapshot().AnyTimes().Return(0)
	mockStateDB.EXPECT().Exist(account).AnyTimes().Return(true)
	mockStateDB.EXPECT().GetCode(account).AnyTimes().Return(codeWithReturnValue)
	mockStateDB.EXPECT().GetCodeHash(account).AnyTimes().Return(common.Hash{byte(accountNumber)})
	mockStateDB.EXPECT().AddressInAccessList(account).AnyTimes().Return(true)
	// Get needed gas from a CALL execution for this code and revision
	gas = getCallInstructionGas(t, revision, codeWithReturnValue)

	zeroVal := big.NewInt(0)
	stackCallValues := []*big.Int{zeroVal, zeroVal, zeroVal, zeroVal, zeroVal, account.Hash().Big(), gasSentWithCall}

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
