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
	var mockCtrl *gomock.Controller
	var mockStateDB *vm_mock.MockStateDB
	mockCtrl = gomock.NewController(t)
	mockStateDB = vm_mock.NewMockStateDB(mockCtrl)

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
			for op, info := range getInstructions(revision) {
				if info.gas.dynamic != nil {
					for _, testCase := range info.gas.dynamic() {
						t.Run(fmt.Sprintf("%s/%s/%s/%s", variant, revision, op, testCase.testName), func(t *testing.T) {
							// Initialize EVM clean instance
							evm := GetCleanEVM(revision, variant, mockStateDB)
							var wantGas uint64 = 0

							// Put needed values on stack with PUSH instructions.
							stackValues := testCase.stackValues
							stackValuesCount := len(stackValues)
							code := make([]byte, 0)
							for i := 0; i < stackValuesCount; i++ {
								valueBytes := stackValues[i].Bytes()
								if len(valueBytes) == 0 {
									valueBytes = []byte{0}
								}
								push := vm.PUSH1 + vm.OpCode(len(valueBytes)-1)
								code = append(code, byte(push))
								for j := 0; j < len(valueBytes); j++ {
									code = append(code, valueBytes[j])
								}
								wantGas += pushGas
							}

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
							if result.GasUsed != uint64(wantGas) {
								t.Errorf("execution failed %v use wrong amount of gas: was %v, want %v", op, result.GasUsed, wantGas)
							}
						})
					}
				}
			}
		}
	}
}
