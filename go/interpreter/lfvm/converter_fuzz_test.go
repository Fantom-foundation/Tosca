// Copyright (c) 2024 Fantom Foundation
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at fantom.foundation/bsl11.
//
// Change Date: 2028-4-16
//
// On the date above, in accordance with the Business Source License, use of
// this software will be governed by the GNU Lesser General Public License v3.

package lfvm

import (
	"testing"

	"github.com/Fantom-foundation/Tosca/go/tosca/vm"
)

func FuzzLfvmConverter(f *testing.F) {

	// Add empty code
	f.Add([]byte{})

	f.Fuzz(func(t *testing.T, toscaCode []byte) {

		// EIP-170 stablish maximum code size to 0x6000 bytes, ~22 KB
		// (see https://eips.ethereum.org/EIPS/eip-170)
		// EIP-3860 stablish maximum init code size to 49_152 bytes
		// (see https://eips.ethereum.org/EIPS/eip-3860)
		// Before EIP-3860, any size was allowed, but in the Fantom
		// network anything larger than 2^16 was observed, which is
		// also the limit for the conversion code (due to a 16-bit PC
		// counter representation in the PC instruction). Thus, we test
		// the code up to this level.
		maxCodeSize := 1<<16 - 1
		if len(toscaCode) > maxCodeSize {
			t.Skip()
		}

		type pair struct {
			originalPos, lfvmPos int
		}
		var pairs []pair
		lfvmCode := convertWithObserver(toscaCode, ConversionConfig{}, func(evm, lfvm int) {
			pairs = append(pairs, pair{evm, lfvm})
		})

		// Check that no super-instructions have been used.
		for _, op := range lfvmCode {
			if op.opcode.isSuperInstruction() {
				t.Errorf("Super-instruction %v used", op.opcode)
			}
		}

		// Check that all operations are mapped to matching operations.
		for _, p := range pairs {

			toscaOpCode := vm.OpCode(toscaCode[p.originalPos])
			lfvmOpCode := lfvmCode[p.lfvmPos].opcode

			if !vm.IsValid(toscaOpCode) && lfvmOpCode != INVALID {
				t.Errorf("Expected INVALID, got %v", lfvmOpCode.String())
			}

			if vm.IsValid(toscaOpCode) {
				if got, want := lfvmOpCode, OpCode(toscaOpCode); got != want {
					t.Errorf("Expected %v, got %v", want, got)
				}
			}
		}

		// Check that the position of JUMPDEST ops are preserved.
		for _, p := range pairs {
			if vm.OpCode(toscaCode[p.originalPos]) == vm.JUMPDEST {
				if p.originalPos != p.lfvmPos {
					t.Errorf("Expected JUMPDEST at %d, got %d", p.originalPos, p.lfvmPos)
				}
			}
		}

		// Check that JUMP_TO instructions point to their immediately succeeding JUMPDEST.
		for i := 0; i < len(lfvmCode); i++ {
			if lfvmCode[i].opcode == JUMP_TO {
				trg := int(lfvmCode[i].arg)
				if trg < i {
					t.Errorf("invalid JUMP_TO target from %d to %d", i, trg)
				}
				if trg >= len(lfvmCode) || lfvmCode[trg].opcode != JUMPDEST {
					t.Fatalf("JUMP_TO target %d is not a JUMPDEST", trg)
				}
				for j := i + 1; j < trg; j++ {
					cur := lfvmCode[j].opcode
					if cur != NOOP {
						t.Errorf("found %v between JUMP_TO and JUMPDEST at %d", cur, j)
					}
				}
			}
		}
	})
}
