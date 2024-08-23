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
	"slices"
	"testing"
)

func TestOpcode_String(t *testing.T) {
	tests := []struct {
		name   string
		opcode OpCode
	}{
		{"PUSH1", PUSH1},
		{"PUSH32", PUSH32},
		{"INVALID", INVALID},
		{"0x00ad", OpCode(0xAD)},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := test.opcode.String(); got != test.name {
				t.Errorf("expected %s, got %s", test.name, got)
			}
		})
	}
}

func TestOpcode_HasArgument(t *testing.T) {

	haveArgument := []OpCode{DATA, JUMP_TO, PUSH2_JUMP, PUSH2_JUMPI, PUSH1_PUSH4_DUP3}
	for i := PUSH1; i <= PUSH32; i++ {
		haveArgument = append(haveArgument, i)
	}

	for i := 0; i < 256; i++ {
		op := OpCode(i)
		if op.HasArgument() != slices.Contains(haveArgument, op) {
			t.Errorf("failed to recognized arguments for opcode %v", op)
		}
	}

}
