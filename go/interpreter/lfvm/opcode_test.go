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
)

func TestOpCode_String(t *testing.T) {
	tests := []struct {
		op   OpCode
		want string
	}{
		{STOP, "STOP"},
		{SWAP2_SWAP1_POP_JUMP, "SWAP2_SWAP1_POP_JUMP"},
		{0x0c, "op(0x0C)"},
		{0x0200, "op(0x0200)"},
	}
	for _, tt := range tests {
		if got := tt.op.String(); got != tt.want {
			t.Errorf("got = %q, want %q", got, tt.want)
		}
	}
}

func TestOpCode_SuperInstructionsAreDecomposedToBasicOpCodes(t *testing.T) {
	for _, op := range allOpCodesWhere(OpCode.isSuperInstruction) {
		baseOps := op.decompose()
		for _, baseOp := range baseOps {
			if baseOp.isSuperInstruction() {
				t.Errorf("decomposition of %v contains super instruction %v", op, baseOp)
			}
		}
	}
}

func TestOpCode_AllOpCodesAreSmallerThanTheOpCodeCapacity(t *testing.T) {
	if want, get := numOpCodes, opCodeMask+1; want != get {
		t.Errorf("opCodeMask+1 = %d, want %d", get, want)
	}
	if _highestOpCode >= numOpCodes {
		t.Errorf(
			"highest op code %d exceeds the current OpCode type capacity of %d",
			_highestOpCode,
			numOpCodes,
		)
	}
}

func TestOpcodeProperty_DoesNotOverflow(t *testing.T) {
	identity := newOpCodeProperty(func(op OpCode) OpCode { return op })
	for i := OpCode(0); i < OpCode(math.MaxInt16); i++ {
		if got, want := identity.get(i), i%numOpCodes; got != want {
			t.Errorf("got %d, want %d", got, want)
		}
	}
}

func allOpCodesWhere(predicate func(op OpCode) bool) []OpCode {
	res := []OpCode{}
	for op := OpCode(0); op < numOpCodes; op++ {
		if predicate(op) {
			res = append(res, op)
		}
	}
	return res
}

func allOpCodes() []OpCode {
	return allOpCodesWhere(func(op OpCode) bool { return true })
}
