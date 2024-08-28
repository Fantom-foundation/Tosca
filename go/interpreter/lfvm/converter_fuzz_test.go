package lfvm

import (
	"testing"

	"github.com/Fantom-foundation/Tosca/go/ct/gen"
	"github.com/Fantom-foundation/Tosca/go/tosca/vm"

	"pgregory.net/rand"
)

func FuzzLfvmConverter(f *testing.F) {

	// Add empty code
	f.Add([]byte{})

	// Use CT code generator to generate one contract starting with each
	// opcode
	rnd := rand.New(1) // deterministic to preserve initial corpus coherence
	generator := gen.NewCodeGenerator()
	empty := generator.Clone()
	for i := 0; i <= 0xFF; i++ {
		op := vm.OpCode(i)
		if !vm.IsValid(op) {
			continue
		}
		generator.Restore(empty)
		generator.SetOperation(0, op)
		code, err := generator.Generate(gen.Assignment{}, rnd)
		if err != nil {
			f.Errorf("Error generating code for opCode %v", op)
		}
		f.Add(code.Copy())
	}

	f.Fuzz(func(t *testing.T, code []byte) {

		// EIP-170 stablish maximum code size
		// (see https://eips.ethereum.org/EIPS/eip-170)
		maxCodeSize := 24_576
		if len(code) > maxCodeSize {
			t.Skip()
		}

		type pair struct {
			originalPos, lfvmPos int
		}
		var pairs []pair
		res := convertWithObserver(code, ConversionConfig{}, func(evm, lfvm int) {
			pairs = append(pairs, pair{evm, lfvm})
		})

		// Check that all operations are mapped to matching operations.
		for _, p := range pairs {
			var want OpCode
			if vm.OpCode(code[p.originalPos]) >= vm.PUSH1 && vm.OpCode(code[p.originalPos]) <= vm.PUSH32 {
				want = PUSH1 + OpCode(res[p.lfvmPos].opcode-PUSH1)
			} else {
				want = op_2_op[vm.OpCode(code[p.originalPos])]
			}

			got := res[p.lfvmPos].opcode
			if want != got {
				t.Logf("Code: %v", code)
				t.Logf("Res : %v", res)
				t.Errorf("Expected %v, got %v", want, got)
			}
		}

		// Check that the position of JUMPDEST ops are preserved.
		for _, p := range pairs {
			if vm.OpCode(code[p.originalPos]) == vm.JUMPDEST {
				if p.originalPos != p.lfvmPos {
					t.Errorf("Expected JUMPDEST at %d, got %d", p.originalPos, p.lfvmPos)
				}
			}
		}
	})
}
