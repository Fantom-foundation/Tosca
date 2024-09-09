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
	"testing"
)

func TestComputeStackUsage_ProducesValidResultsForSingleOps(t *testing.T) {
	tests := []struct {
		op    OpCode
		usage stackUsage
	}{
		{STOP, stackUsage{from: 0, to: 0, delta: 0}},
		{ADD, stackUsage{from: -2, to: 0, delta: -1}},
		{POP, stackUsage{from: -1, to: 0, delta: -1}},
		{PUSH5, stackUsage{from: 0, to: 1, delta: 1}},
		{SWAP1, stackUsage{from: -2, to: 0, delta: 0}},
		{SWAP10, stackUsage{from: -11, to: 0, delta: 0}},
		{DUP1, stackUsage{from: -1, to: 1, delta: 1}},
		{DUP12, stackUsage{from: -12, to: 1, delta: 1}},
		{LOG3, stackUsage{from: -5, to: 0, delta: -5}},
	}

	for _, test := range tests {
		t.Run(test.op.String(), func(t *testing.T) {
			usage, err := computeStackUsage(test.op)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got, want := usage, test.usage; got != want {
				t.Errorf("unexpected result: want %v, got %v", want, got)
			}
		})
	}
}

func TestComputeStackUsage_ReportsAnErrorForInvalidOperations(t *testing.T) {
	ops := []OpCode{
		INVALID, // defined invalid
		NOOP,    // extended set of instructions
		DATA,    // extended set of instructions
		0xffff,  // out of range
		0x0c,    //< some code that is not an opcode but within the range
	}

	for _, op := range ops {
		_, err := computeStackUsage(op)
		if err == nil {
			t.Errorf("expected error for opcode %v", op)
		}
	}
}

func TestCombineStackUsage(t *testing.T) {
	tests := []struct {
		ops   []OpCode
		usage stackUsage
	}{
		{
			[]OpCode{},
			stackUsage{from: 0, to: 0, delta: 0},
		},
		{
			[]OpCode{PUSH1},
			stackUsage{from: 0, to: 1, delta: 1},
		},
		{
			[]OpCode{POP},
			stackUsage{from: -1, to: 0, delta: -1},
		},
		{
			[]OpCode{PUSH1, PUSH1},
			stackUsage{from: 0, to: 2, delta: 2},
		},
		{
			[]OpCode{PUSH1, POP},
			stackUsage{from: 0, to: 1, delta: 0},
		},
		{
			[]OpCode{POP, PUSH1},
			stackUsage{from: -1, to: 0, delta: 0},
		},
		{
			[]OpCode{POP, POP},
			stackUsage{from: -2, to: 0, delta: -2},
		},
		{
			[]OpCode{PUSH1, PUSH1, POP, POP},
			stackUsage{from: 0, to: 2, delta: 0},
		},
		{
			[]OpCode{PUSH1, PUSH1, POP, POP, POP, PUSH1, PUSH1},
			stackUsage{from: -1, to: 2, delta: 1},
		},
		{
			[]OpCode{PUSH1, LOG4, PUSH1},
			stackUsage{from: -5, to: 1, delta: -4},
		},
		{
			[]OpCode{PUSH1_ADD, ISZERO_PUSH2_JUMPI},
			stackUsage{from: -1, to: 1, delta: -1},
		},
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("%v", test.ops), func(t *testing.T) {
			usages := []stackUsage{}
			for _, op := range test.ops {
				usage, err := computeStackUsage(op)
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				usages = append(usages, usage)
			}

			res := combineStackUsage(usages...)
			if res != test.usage {
				t.Errorf("unexpected result: want %v, got %v", test.usage, res)
			}
		})
	}
}
