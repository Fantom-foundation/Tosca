package vm_test

import (
	"fmt"
	"testing"

	// This is only imported to get the EVM opcode definitions.
	// TODO: write up our own op-code definition and remove this dependency.
	vm "github.com/ethereum/go-ethereum/core/vm"
)

func TestEmptyCodeShouldBeIgnored(t *testing.T) {
	for _, variant := range Variants {
		evm := GetCleanEVM(Istanbul, variant, nil)
		t.Run(variant, func(t *testing.T) {
			code := []byte{}
			input := []byte{}
			if _, err := evm.Run(code, input); err != nil {
				t.Errorf("failed to accept empty code, got %v", err)
			}
		})
	}
}

func TestPushWithMissingDataIsIgnored(t *testing.T) {
	for _, variant := range Variants {
		evm := GetCleanEVM(Istanbul, variant, nil)
		for i := 1; i <= 32; i++ {
			op := vm.OpCode(int(vm.PUSH1) - 1 + i)
			t.Run(fmt.Sprintf("%s-%s", variant, op), func(t *testing.T) {
				input := []byte{}
				for j := 0; j < i; j++ {
					code := make([]byte, 1+j)
					code[0] = byte(op)
					if _, err := evm.Run(code, input); err != nil {
						t.Errorf("failed to accept missing data, got %v", err)
					}
				}
			})
		}
	}
}

func TestDetectsJumpOutOfCode(t *testing.T) {
	for _, variant := range Variants {
		evm := GetCleanEVM(Istanbul, variant, nil)
		t.Run(variant, func(t *testing.T) {
			code := []byte{
				byte(vm.PUSH1), 200,
				byte(vm.JUMP),
			}
			input := []byte{}

			result, err := evm.Run(code, input)
			if err != nil {
				t.Fatalf("unexpected failure in VM execution: %v", err)
			}
			if result.Success {
				t.Errorf("expected VM to fail, got %v", result)
			}
		})
	}
}

func TestDetectsJumpToNonJumpDestTarget(t *testing.T) {
	for _, variant := range Variants {
		evm := GetCleanEVM(Istanbul, variant, nil)
		t.Run(variant, func(t *testing.T) {
			code := []byte{
				byte(vm.PUSH1), 3,
				byte(vm.JUMP),
				byte(vm.STOP),
			}
			input := []byte{}

			result, err := evm.Run(code, input)
			if err != nil {
				t.Fatalf("unexpected failure in VM execution: %v", err)
			}
			if result.Success {
				t.Errorf("expected VM to fail, got %v", result)
			}
		})
	}
}
