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

	"github.com/Fantom-foundation/Tosca/go/tosca/vm"
)

func TestOpcode_String(t *testing.T) {

	special_op_str := map[OpCode]string{
		JUMP_TO: "JUMP_TO",

		SWAP2_SWAP1_POP_JUMP:  "SWAP2_SWAP1_POP_JUMP",
		SWAP1_POP_SWAP2_SWAP1: "SWAP1_POP_SWAP2_SWAP1",
		POP_SWAP2_SWAP1_POP:   "POP_SWAP2_SWAP1_POP",
		PUSH2_JUMP:            "PUSH2_JUMP",
		PUSH2_JUMPI:           "PUSH2_JUMPI",
		DUP2_MSTORE:           "DUP2_MSTORE",
		DUP2_LT:               "DUP2_LT",

		SWAP1_POP:   "SWAP1_POP",
		POP_JUMP:    "POP_JUMP",
		SWAP2_SWAP1: "SWAP2_SWAP1",
		SWAP2_POP:   "SWAP2_POP",
		PUSH1_PUSH1: "PUSH1_PUSH1",
		PUSH1_ADD:   "PUSH1_ADD",
		PUSH1_DUP1:  "PUSH1_DUP1",
		POP_POP:     "POP_POP",
		PUSH1_SHL:   "PUSH1_SHL",

		ISZERO_PUSH2_JUMPI:        "ISZERO_PUSH2_JUMPI",
		PUSH1_PUSH4_DUP3:          "PUSH1_PUSH4_DUP3",
		AND_SWAP1_POP_SWAP2_SWAP1: "AND_SWAP1_POP_SWAP2_SWAP1",
		PUSH1_PUSH1_PUSH1_SHL_SUB: "PUSH1_PUSH1_PUSH1_SHL_SUB",

		DATA:    "DATA",
		NOOP:    "NOOP",
		INVALID: "INVALID",
	}

	for i := 0; i < 256; i++ {
		op := OpCode(i)
		want := ""
		found := false
		toscaOp := findIndex(op_2_op, op)
		if PUSH1 <= op && op <= PUSH32 {
			toscaOp = int(vm.PUSH1) + int(op) - 2
		}
		if toscaOp < 256 && vm.IsValid(vm.OpCode(toscaOp)) {
			want = vm.OpCode(toscaOp).String()
		} else {
			want, found = special_op_str[op]
			if !found {
				want = fmt.Sprintf("0x%04x", byte(op))
			}
		}
		got := op.String()
		if got != want {
			t.Errorf("expected %s, got %s", want, got)
		}
	}
}

func findIndex(slice []OpCode, element OpCode) int {
	for i, v := range slice {
		if v == element {
			return i
		}
	}
	return 256
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
