package vm_test

import (
	"fmt"
	"testing"

	"github.com/Fantom-foundation/Tosca/go/vm/lfvm"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/params"
)

func TestStackMaxBoundry(t *testing.T) {
	// For every variant of interpreter
	for _, variant := range Variants {
		for _, revision := range revisions {
			for _, op := range getFullStackFailOpCodes(revision) {
				t.Run(fmt.Sprintf("%s/%s/%s", variant, revision, op), func(t *testing.T) {
					var stateDB vm.StateDB
					evm := GetCleanEVM(revision, variant, stateDB)

					// Fill stack with PUSH1 instructions.
					size := int(params.StackLimit)
					code := make([]byte, 0, size*2+1)
					for i := 0; i < size; i++ {
						code = append(code, []byte{byte(vm.PUSH1), 1}...)
					}

					// Set a tested instruction as the last one.
					code = append(code, byte(op))

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
			for _, op := range getEmptyStackFailOpCodes(revision) {
				t.Run(fmt.Sprintf("%s/%s/%s", variant, revision, op), func(t *testing.T) {
					var stateDB *lfvm.MockStateDB

					evm := GetCleanEVM(revision, variant, stateDB)

					// Execute only solo instruction with empty stack
					code := []byte{byte(op)}

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

func getEmptyStackFailOpCodes(revision Revision) []vm.OpCode {
	var result []vm.OpCode
	for op, info := range getInstructions(revision) {
		if info.stack.popped > 0 {
			result = append(result, op)
		}
	}
	return result
}

func getFullStackFailOpCodes(revision Revision) []vm.OpCode {
	var result []vm.OpCode
	for op, info := range getInstructions(revision) {
		if info.stack.popped < info.stack.pushed {
			result = append(result, op)
		}
	}
	return result
}
