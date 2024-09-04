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
	"regexp"
	"slices"
	"testing"
)

func TestOpcode_UnnamedOpcodes(t *testing.T) {
	validName := regexp.MustCompile(`^0x00[0-9A-Fa-f]{2}$`)
	for i := 0; i < math.MaxUint16; i++ {
		op := OpCode(i)
		if !validName.MatchString(op.String()) && op > NUM_OPCODES {
			t.Errorf("Invalid print for op %v (%d)", op, i)
		}
	}
}

func TestOpcode_NamedOpcodes(t *testing.T) {
	validName := regexp.MustCompile(`^0x00[0-9A-Fa-f]{2}$`)
	for i := 0; i < math.MaxUint16; i++ {
		op := OpCode(i)
		// NUM_EXECUTABLE_OPCODES is a special case, it does not have a string name
		if validName.MatchString(op.String()) && op < NUM_OPCODES && op != NUM_EXECUTABLE_OPCODES {
			t.Errorf("Invalid print for op %v (%d)", op, i)
		}
	}
}

func TestOpcode_HasArgument(t *testing.T) {

	haveArgument := []OpCode{PC, DATA, JUMP_TO, PUSH2_JUMP, PUSH2_JUMPI, PUSH1_PUSH4_DUP3}
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
