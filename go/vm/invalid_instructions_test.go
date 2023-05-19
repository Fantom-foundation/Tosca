package vm

import (
	"fmt"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/core/vm"
)

func TestInterpreterDetectsInvalidInstruction(t *testing.T) {
	for _, rev := range revisions {
		evm := newTestEVM(rev)
		for _, variant := range variants {
			// LFVM currently does not support detection of invalid codes!
			// TODO: fix this
			if strings.Contains(variant, "lfvm") {
				continue
			}
			interpreter := vm.NewInterpreter(variant, evm, vm.Config{})
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
					if err := runCode(interpreter, code, input); !isInvalidOpCodeError(err) {
						t.Errorf("failed to identify invalid OpCode %v as invalid instruction, got %v", op, err)
					}
				})
			}
		}
	}
}

func isInvalidOpCodeError(err error) bool {
	_, ok := err.(*vm.ErrInvalidOpCode)
	return ok
}
