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

import "testing"

func TestInstruction_String(t *testing.T) {

	tests := []struct {
		instruction Instruction
		want        string
	}{
		{Instruction{opcode: STOP, arg: 0x0000}, "STOP"},
		{Instruction{opcode: PUSH1, arg: 0x0001}, "PUSH1 0x0001"},
		{Instruction{opcode: PUSH2, arg: 0x0002}, "PUSH2 0x0002"},
		{Instruction{opcode: DATA, arg: 0x0002}, "DATA 0x0002"},
		{Instruction{opcode: JUMP_TO, arg: 0x0002}, "JUMP_TO 0x0002"},
		{Instruction{opcode: PUSH2_JUMP, arg: 0x0002}, "PUSH2_JUMP 0x0002"},
		{Instruction{opcode: PUSH2_JUMPI, arg: 0x0002}, "PUSH2_JUMPI 0x0002"},
		{Instruction{opcode: PUSH1_PUSH4_DUP3, arg: 0x0002}, "PUSH1_PUSH4_DUP3 0x0002"},
	}

	for _, tt := range tests {
		if got := tt.instruction.String(); got != tt.want {
			t.Errorf("Instruction.String() = %q, want %q", got, tt.want)
		}
	}

}

func TestCode_String(t *testing.T) {
	code := Code{
		Instruction{opcode: STOP, arg: 0x0000},
		Instruction{opcode: PUSH1, arg: 0x0001},
	}
	if got, want := code.String(), "0x0000: STOP\n0x0001: PUSH1 0x0001\n"; got != want {
		t.Errorf("Code.String() = %q, want %q", got, want)
	}
}
