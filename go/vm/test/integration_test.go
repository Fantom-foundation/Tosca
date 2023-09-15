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

func TestMaxCallDepth(t *testing.T) {
	var mockCtrl *gomock.Controller
	var mockStateDB *vm_mock.MockStateDB

	// For every variant of interpreter
	for _, variant := range Variants {

		for _, revision := range revisions {

			t.Run(fmt.Sprintf("%s/%s", variant, revision), func(t *testing.T) {

				mockCtrl = gomock.NewController(t)
				mockStateDB = vm_mock.NewMockStateDB(mockCtrl)

				zeroVal := big.NewInt(0)
				account := common.Address{byte(0)}
				// big value to cover all recursive calls
				gasSentWithCall := 1000000000000

				// return and input data size is 32bytes, memory offset is 0 for all
				callStackValues := []*big.Int{big.NewInt(32), zeroVal, big.NewInt(32), zeroVal, zeroVal, account.Hash().Big(), big.NewInt(int64(gasSentWithCall))}
				pushCode, _ := addValuesToStack(callStackValues, 0)

				// put 32byte input value with 0 offset from memory to stack, add 1 to it and put it back to memory with 0 offset
				code := []byte{byte(vm.PUSH1), byte(0), byte(vm.CALLDATALOAD), byte(vm.PUSH1), byte(1), byte(vm.ADD), byte(vm.PUSH1), byte(0), byte(vm.MSTORE)}

				// add stack values for call instruction
				code = append(code, pushCode...)

				// make inner call and return 32byte value with 0 offset from memory
				codeReturn := []byte{byte(vm.CALL), byte(vm.PUSH1), byte(32), byte(vm.PUSH1), byte(0), byte(vm.RETURN)}
				code = append(code, codeReturn...)

				// mock state calls for call instruction
				mockStateDB.EXPECT().GetRefund().AnyTimes().Return(uint64(0))
				mockStateDB.EXPECT().SubRefund(gomock.Any()).AnyTimes()
				mockStateDB.EXPECT().AddRefund(gomock.Any()).AnyTimes()
				mockStateDB.EXPECT().GetCodeHash(account).AnyTimes().Return(common.Hash{byte(0)})
				mockStateDB.EXPECT().Snapshot().AnyTimes().Return(0)
				mockStateDB.EXPECT().Exist(account).AnyTimes().Return(true)
				mockStateDB.EXPECT().GetCode(account).AnyTimes().Return(code)
				mockStateDB.EXPECT().AddressInAccessList(account).AnyTimes().Return(true)
				mockStateDB.EXPECT().RevertToSnapshot(gomock.Any()).AnyTimes()

				evm := GetCleanEVM(revision, variant, mockStateDB)

				// Run an interpreter
				result, err := evm.Run(code, []byte{})

				// Check the result.
				if err != nil {
					t.Errorf("execution failed and should not fail, error is: %v", err)
				} else {
					expectedDepth := 1025
					depth := big.NewInt(0).SetBytes(result.Output).Uint64()
					if depth != uint64(expectedDepth) {
						t.Errorf("expected call depth is %v, got %v", expectedDepth, depth)
					}
				}
			})
		}
	}
}

func TestInvalidJumpOverflow(t *testing.T) {

	// For every variant of interpreter
	for _, variant := range Variants {

		for _, revision := range revisions {

			testInstructions := []vm.OpCode{vm.JUMP, vm.JUMPI}

			for _, instruction := range testInstructions {

				t.Run(fmt.Sprintf("%s/%s/%s", variant, revision, instruction.String()), func(t *testing.T) {

					evm := GetCleanEVM(revision, variant, nil)

					// condition for JUMPI instruction
					condition := big.NewInt(1)
					// destination number bigger then uint64 with uint64 part containing relevant jump destination
					dst, _ := big.NewInt(0).SetString("0x1000000000000000d", 0)
					code, _ := addValuesToStack([]*big.Int{condition, dst}, 0)
					codeJump := []byte{byte(instruction), byte(vm.JUMPDEST), byte(vm.PUSH1), byte(0), byte(vm.STOP)}
					code = append(code, codeJump...)

					// Run an interpreter
					_, err := evm.Run(code, []byte{})

					// Check the result.
					if err == nil {
						t.Errorf("execution should fail with error : %v", vm.ErrInvalidJump)
					} else {
						if err != vm.ErrInvalidJump {
							t.Errorf("execution should fail with error : %v but got: %v", vm.ErrInvalidJump, err)
						}
					}
				})
			}
		}
	}
}
