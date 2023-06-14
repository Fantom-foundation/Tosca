package vm_test

import (
	"fmt"
	"testing"

	vm_mock "github.com/Fantom-foundation/Tosca/go/vm/test/mocks"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/params"
)

func TestStackMaxBoundry(t *testing.T) {
	// For every variant of interpreter
	for _, variant := range Variants {
		for _, revision := range revisions {
			for op, info := range getInstructions(revision) {
				if info.stack.popped >= info.stack.pushed {
					continue
				}
				t.Run(fmt.Sprintf("%s/%s/%s", variant, revision, op), func(t *testing.T) {
					var stateDB vm.StateDB
					evm := GetCleanEVM(revision, variant, stateDB)

					// push size
					size := info.stack.pushed - info.stack.popped
					// needed stack size
					size = int(params.StackLimit) - size + 1

					code := getCode(size, op)

					// Run an interpreter
					_, err := evm.Run(code, []byte{})

					// Check the result.
					if _, isOverflow := err.(*vm.ErrStackOverflow); !isOverflow {
						t.Errorf("execution failed %v should end with stack overflow: status is %v", op, err)
					}
					// Note: the amount of consumed gas is not relevant, since
					// in case of a stack overflow all remaining gas is consumed.
				})
			}
		}
	}
}

func TestStackMinBoundry(t *testing.T) {
	// For every variant of interpreter
	for _, variant := range Variants {
		for _, revision := range revisions {
			for op, info := range getInstructions(revision) {
				if info.stack.popped <= 0 {
					continue
				}
				t.Run(fmt.Sprintf("%s/%s/%s", variant, revision, op), func(t *testing.T) {
					var stateDB *vm_mock.MockStateDB

					evm := GetCleanEVM(revision, variant, stateDB)
					code := getCode(info.stack.popped-1, op)

					// Run an interpreter
					_, err := evm.Run(code, []byte{})

					// Check the result.
					if _, isUnderflow := err.(*vm.ErrStackUnderflow); !isUnderflow {
						t.Errorf("execution failed %v should end with stack underflow: status is %v", op, err)
					}
					// Note: the amount of consumed gas is not relevant, since
					// in case of a stack underflow all remaining gas is consumed.
				})
			}
		}
	}
}

func getCode(stackLength int, op vm.OpCode) []byte {
	code := make([]byte, 0, stackLength*2+1)

	// Add to stack PUSH1 instructions
	for i := 0; i < stackLength; i++ {
		code = append(code, []byte{byte(vm.PUSH1), byte(0)}...)
	}

	// Set a tested instruction as the last one.
	code = append(code, byte(op))
	return code
}
