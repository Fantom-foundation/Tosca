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
	"bytes"
	"errors"
	"math"
	"testing"

	"github.com/Fantom-foundation/Tosca/go/tosca"
)

func TestExpansionCosts(t *testing.T) {

	tests := []struct {
		size uint64
		cost tosca.Gas
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
		cost := m.getExpansionCosts(test.size)
		if cost != test.cost {
			t.Errorf("ExpansionCosts(%d) = %d, want %d", test.size, cost, test.cost)
		}
	}
}

func TestMemory_expandMemoryWithoutCharging(t *testing.T) {

	test := map[string]struct {
		size        uint64
		initialMem  []byte
		expectedMem []byte
	}{
		"empty-memory-increases-to-desired-size": {
			size:        32,
			initialMem:  []byte{},
			expectedMem: []byte{31: 0x0},
		},
		"memory-bigger-than-size-changes-nothing": {
			size:        32,
			initialMem:  []byte{63: 0x0},
			expectedMem: []byte{63: 0x0},
		},
		"size-zero-changes-nothing": {
			size:        0,
			initialMem:  []byte{},
			expectedMem: []byte{},
		},
	}

	for name, test := range test {
		t.Run(name, func(t *testing.T) {
			m := NewMemory()
			fee := m.getExpansionCosts(test.size)
			m.store = test.initialMem
			m.expandMemoryWithoutCharging(test.size, fee)
			if !bytes.Equal(m.store, test.expectedMem) {
				t.Errorf("unexpected memory value, want: %x, got: %x", test.expectedMem, m.store)
			}
			if m.total_memory_cost != fee {
				t.Errorf("unexpected total memory cost, want: %d, got: %d", fee, m.total_memory_cost)
			}
		})
	}

}

func TestMemory_expandMemory_ErrorCases(t *testing.T) {

	tests := map[string]struct {
		size     uint64
		offset   uint64
		gas      tosca.Gas
		expected error
	}{
		"not enough gas": {
			size:     32,
			offset:   0,
			gas:      0,
			expected: errOutOfGas,
		},
		"offset overflow": {
			size:     1,
			offset:   math.MaxUint64,
			gas:      100,
			expected: errGasUintOverflow,
		},
		"size overflow": {
			size:     math.MaxUint64,
			offset:   1,
			gas:      100,
			expected: errGasUintOverflow,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			ctxt := getEmptyContext()
			m := NewMemory()
			ctxt.gas = test.gas

			err := m.expandMemory(test.offset, test.size, &ctxt)
			if !errors.Is(err, test.expected) {
				t.Errorf("unexpected error, want: %v, got: %v", test.expected, err)
			}
			if m.length() != 0 {
				t.Errorf("should have remained empty, got length: %d", m.length())
			}
		})
	}
}

func TestMemory_expandMemory(t *testing.T) {

	tests := map[string]struct {
		size          uint64
		offset        uint64
		initialMemory []byte
	}{
		"empty-memory-with-zero-offset-and-size-does-not-expand": {},
		"size-zero-with-offset-does-not-expand": {
			size:   0,
			offset: 32,
		},
		"expand-memory": {
			size:   32,
			offset: 0,
		},
		"memory-bigger-than-size+offset-does-not-expand": {
			size:          32,
			offset:        64,
			initialMemory: []byte{127: 0x0},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			ctxt := getEmptyContext()
			m := NewMemory()
			ctxt.gas = 100

			err := m.expandMemory(test.offset, test.size, &ctxt)
			memSize := m.length()
			if err != nil {
				t.Fatalf("unexpected error, want: %v, got: %v", nil, err)
			}
			if test.size == 0 {
				if m.length() != uint64(len(test.initialMemory)) {
					t.Errorf("memory size should not have changed, want: %d, got: %d", len(test.initialMemory), memSize)
				}
			} else {
				if want := test.size + test.offset; memSize < want {
					t.Errorf("memory size should be bigger than offset+size, want: %d, got: %d", want, memSize)
				}
			}
		})
	}
}
