package vm_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/core/vm"
)

func TestInterpreterDetectsInvalidInstruction(t *testing.T) {
	for _, rev := range revisions {
		for _, variant := range Variants {
			evm := GetCleanEVM(rev, variant, nil)
			// LFVM currently does not support detection of invalid codes!
			// TODO: fix this
			if strings.Contains(variant, "lfvm") {
				continue
			}
			instructions := getInstructions(rev)
			for i := 0; i < 256; i++ {
				op := vm.OpCode(i)
				_, exits := instructions[op]
				if exits {
					continue
				}
				t.Run(fmt.Sprintf("%s-%s-%s", variant, rev, op), func(t *testing.T) {
					code := []byte{byte(op), byte(vm.STOP)}
					input := []byte{}

					result, err := evm.Run(code, input)
					if err != nil {
						t.Fatalf("unexpected failure of EVM execution: %v", err)
					}
					if result.Success {
						t.Errorf("expected execution to fail, but got %v", result)
					}
				})
			}
		}
	}
}
