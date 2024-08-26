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
	"fmt"
	"slices"
	"testing"
)

func TestOpcode_String(t *testing.T) {
	for i := 0; i < 256; i++ {
		op := OpCode(i)
		want, found := op_to_string[op]
		got := op.String()
		if !found {
			want = fmt.Sprintf("0x%04x", byte(op))
		}
		if got != want {
			t.Errorf("expected %s, got %s", want, got)
		}
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
