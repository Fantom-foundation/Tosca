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
	for _, op := range allOpCodes() {
		if !op.isSuperInstruction() {
			continue
		}

		baseOps := op.decompose()
		for _, baseOp := range baseOps {
			if baseOp.isSuperInstruction() {
				t.Errorf("decomposition of %v contains super instruction %v", op, baseOp)
			}
		}
	}
}

func TestOpCode_AllOpCodesAreSmallerThan512(t *testing.T) {
	// Some lookup tables are sized to 512 and use mask 0x1FF to index them.
	// If any opcode that violates this property is introduced, undefined behavior
	// may be difficult to detect.
	for _, op := range allOpCodes() {
		if op > 512 {
			t.Errorf("op code %v is greater than 512", op)
		}
	}
}

func allOpCodesWhere(predicate func(op OpCode) bool) []OpCode {
	res := []OpCode{}
	for op := OpCode(0); op < NUM_OPCODES; op++ {
		if op.isValid() && predicate(op) {
			res = append(res, op)
		}
	}
	return res
}

func allOpCodes() []OpCode {
	return allOpCodesWhere(func(op OpCode) bool { return true })
}

func allExecutableOpCodes() []OpCode {
	return allOpCodesWhere(func(op OpCode) bool { return op.isExecutable() })
}
