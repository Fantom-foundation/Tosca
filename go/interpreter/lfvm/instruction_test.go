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

func TestInstruction_NewInstruction(t *testing.T) {

	instruction := NewInstruction(STOP, 0x0000)
	if got, want := instruction.opcode, STOP; got != want {
		t.Errorf("unexpected opcode, want %v, got = %v", want, got)
	}
	if got, want := instruction.arg, uint16(0x0000); got != want {
		t.Errorf("unexpected arg, want %v, got %v", want, got)
	}
}

func TestCode_IsIndexOp(t *testing.T) {
	code := Code{
		Instruction{opcode: STOP, arg: 0x0000},
		Instruction{opcode: PUSH1, arg: 0x0001},
	}
	if want, got := STOP, code[0].opcode; !code.IsIndexOp(0, STOP) {
		t.Errorf("unexpected code at position 0, want %v got %v", want, got)
	}
	if want, got := PUSH1, code[1]; code.IsIndexOp(1, STOP) {
		t.Errorf("unexpected code at position 1, want %v got %v", want, got)
	}
}

func TestCode_GetArgOf(t *testing.T) {
	code := Code{
		Instruction{opcode: STOP, arg: 0x0002},
		Instruction{opcode: PUSH1, arg: 0x0001},
	}
	if want, got := uint16(0x0002), code.GetArgOf(0); got != want {
		t.Errorf("unexpected arg at position 0, want %v, got %v", want, got)
	}
	if want, got := uint16(0x0001), code.GetArgOf(1); got != want {
		t.Errorf("unexpected arg at position 1, want %v, got %v", want, got)
	}
}

func TestInstruction_String(t *testing.T) {
	instruction := Instruction{opcode: STOP, arg: 0x0000}
	if got, want := instruction.String(), "STOP"; got != want {
		t.Errorf("Instruction.String() = %q, want %q", got, want)
	}

	instruction = Instruction{opcode: PUSH1, arg: 0x0001}
	if got, want := instruction.String(), "PUSH1 0x0001"; got != want {
		t.Errorf("Instruction.String() = %q, want %q", got, want)
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
