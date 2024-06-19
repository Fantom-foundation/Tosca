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

	"github.com/Fantom-foundation/Tosca/go/vm"
)

func TestMagicNumber(t *testing.T) {

	// Call me paranoid, but I am repeating the test for the magic number:
	// The difference between geth and lfvm is that they use uint64 as gas
	// type, while lfvm uses vm.Gas which is int64. This is why this test
	// checks that after the calculation, the result can still fit in a signed
	// 64-bit integer.

	v := maxMemoryExpansionSize
	w := sizeInWords(uint64(v))
	square := w * w

	if square/512 < 3*w {
		t.Errorf("square/512 < 3*w")
	}

	if square/512+3*w > math.MaxInt64 {
		t.Errorf("square/512 + 3*w  > math.MaxInt64")
	}
}

func TestExpansionCosts(t *testing.T) {

	tests := []struct {
		size uint64
		cost vm.Gas
	}{
		{0, 0},
		{1, 3},
		{32, 3},
		{33, 6},
		{64, 6},
		{65, 9},
		{22 * 32, 3 * 22},             // last word size without square cost
		{23 * 32, (23*23)/512 + 3*23}, // fist word size with square cost
		{maxMemoryExpansionSize - 33, 36028809870311418},
		{maxMemoryExpansionSize - 1, 36028809887088637},
		{maxMemoryExpansionSize, 36028809887088637}, // magic number, max cost
		{maxMemoryExpansionSize + 1, math.MaxInt64},
		{math.MaxInt64, math.MaxInt64},
	}

	for _, test := range tests {

		m := NewMemory()
		cost := m.ExpansionCosts(test.size)
		if cost != test.cost {
			t.Errorf("ExpansionCosts(%d) = %d, want %d", test.size, cost, test.cost)
		}
	}
}
