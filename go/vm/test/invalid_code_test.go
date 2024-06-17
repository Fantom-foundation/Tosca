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
