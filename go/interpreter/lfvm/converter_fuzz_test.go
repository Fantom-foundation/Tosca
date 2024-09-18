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
	"math"
	"testing"

	"github.com/Fantom-foundation/Tosca/go/tosca/vm"
)

// To run this fuzzer use the following command:
// go test ./interpreter/lfvm -run none -fuzz LfvmConverter --fuzztime 10m

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
		maxCodeSize := math.MaxUint16
		if len(toscaCode) > maxCodeSize {
			t.Skip()
		}

		mapping := make([]int, len(toscaCode))
		lfvmCode := convertWithObserver(toscaCode, ConversionConfig{}, func(evm, lfvm int) {
			mapping[evm] = lfvm
		})

		// Check that no super-instructions have been used.
		for _, op := range lfvmCode {
			if op.opcode.isSuperInstruction() {
				t.Errorf("Super-instruction %v used", op.opcode)
			}
		}

		// Check that all operations are mapped to matching operations.
		for i := 0; i < len(toscaCode); i++ {
			originalPos := i
			lfvmPos := mapping[originalPos]

			toscaOpCode := vm.OpCode(toscaCode[originalPos])
			lfvmOpCode := lfvmCode[lfvmPos].opcode

			if !lfvmOpCode.isBaseInstruction() {
				t.Errorf("Expected base instructions only, got %v", lfvmOpCode)
			}

			if vm.OpCode(lfvmOpCode) != toscaOpCode {
				t.Errorf("Invalid conversion from %v to %v", toscaOpCode, lfvmOpCode)
			}

			// Check that the position of JUMPDEST ops are preserved.
			if toscaOpCode == vm.JUMPDEST {
				if originalPos != lfvmPos {
					t.Errorf("Expected JUMPDEST at %d, got %d", originalPos, lfvmPos)
				}
			}

			// Check that PC instructions point to the correct target.
			if toscaOpCode == vm.PC {
				target := int(lfvmCode[lfvmPos].arg)
				if target != originalPos {
					t.Errorf("Invalid PC target, wanted %d, got %d", originalPos, target)
				}
			}

			// Skip the data section of PUSH instructions.
			if vm.PUSH1 <= toscaOpCode && toscaOpCode <= vm.PUSH32 {
				i += int(toscaOpCode-vm.PUSH1) + 1
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
