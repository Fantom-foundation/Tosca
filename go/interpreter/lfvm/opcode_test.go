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
