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
	"github.com/holiman/uint256"
)

func TestMemory_NewMemoryIsEmpty(t *testing.T) {
	m := NewMemory()
	if m.length() != 0 {
		t.Errorf("memory should be empty, instead has length: %d", m.length())
	}
}

func TestGetExpansionCosts(t *testing.T) {

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
			t.Errorf("getExpansionCosts(%d) = %d, want %d", test.size, cost, test.cost)
		}
	}
}

func TestMemory_expandMemoryWithoutCharging(t *testing.T) {

	test := map[string]struct {
		size        uint64
		initialMem  []byte
		expectedMem []byte
	}{
		"empty memory increases to desired size": {
			size:        32,
			initialMem:  []byte{},
			expectedMem: []byte{31: 0x0},
		},
		"memory bigger than size changes nothing": {
			size:        32,
			initialMem:  []byte{63: 0x0},
			expectedMem: []byte{63: 0x0},
		},
		"size zero changes nothing": {
			size:        0,
			initialMem:  []byte{},
			expectedMem: []byte{},
		},
		"check memory increases by 32": {
			size:        41,
			initialMem:  []byte{},
			expectedMem: []byte{63: 0x0},
		},
	}

	for name, test := range test {
		t.Run(name, func(t *testing.T) {
			m := NewMemory()
			m.store = test.initialMem
			fee := m.getExpansionCosts(test.size)
			m.expandMemoryWithoutCharging(test.size)
			if !bytes.Equal(m.store, test.expectedMem) {
				t.Errorf("unexpected memory value, want: %x, got: %x", test.expectedMem, m.store)
			}
			if m.currentMemoryCost != fee {
				t.Errorf("unexpected total memory cost, want: %d, got: %d", fee, m.currentMemoryCost)
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

func TestMemory_expandMemory_expandsMemoryOnlyWhenNeeded(t *testing.T) {

	tests := map[string]struct {
		size               uint64
		offset             uint64
		initialMemorySize  uint64
		expectedMemorySize uint64
	}{
		"empty memory with zero offset and size does not expand": {},
		"size zero with offset does not expand": {
			size:               0,
			offset:             32,
			expectedMemorySize: 0,
		},
		"expand memory in words length": {
			size:               13,
			offset:             0,
			expectedMemorySize: 32,
		},
		"memory bigger than size+offset does not expand": {
			size:               41,
			offset:             41,
			initialMemorySize:  128,
			expectedMemorySize: 128,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			ctxt := getEmptyContext()
			m := NewMemory()
			m.store = make([]byte, test.initialMemorySize)
			ctxt.gas = 3

			err := m.expandMemory(test.offset, test.size, &ctxt)
			if err != nil {
				t.Fatalf("unexpected error, want: %v, got: %v", nil, err)
			}
			if m.length() != test.expectedMemorySize {
				t.Errorf("unexpected memory size, want: %d, got: %d", test.expectedMemorySize, m.length())
			}
		})
	}
}

func TestMemory_getSlice_ErrorCases(t *testing.T) {
	c := getEmptyContext()
	c.gas = 0
	m := NewMemory()
	_, err := m.getSlice(0, 1, &c)
	if !errors.Is(err, errOutOfGas) {
		t.Errorf("error should be errOutOfGas, instead is: %v", err)
	}
	_, err = m.getSlice(math.MaxUint64-31, 32, &c)
	if !errors.Is(err, errGasUintOverflow) {
		t.Errorf("error should be errGasUintOverflow, instead is: %v", err)
	}
}

func TestMemory_getSlice_properlyHandlesMemoryExpansionsAndReturnsExpectedMemory(t *testing.T) {

	tests := map[string]struct {
		offset   uint64
		size     uint64
		expected []byte
	}{
		"size zero returns empty slice": {
			offset:   64,
			size:     0,
			expected: []byte{},
		},
		"memory does not expand when not needed": {
			offset:   0,
			size:     4,
			expected: []byte{0x0, 0x01, 0x02, 0x03},
		},
		"memory expands when needed": {
			offset:   4,
			size:     5,
			expected: []byte{0x04, 4: 0x0},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			c := getEmptyContext()
			m := NewMemory()
			m.store = []byte{0x0, 0x01, 0x02, 0x03, 0x04}
			slice, err := m.getSlice(test.offset, test.size, &c)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if !bytes.Equal(slice, test.expected) {
				t.Errorf("unexpected slice: %x, want: %x", slice, test.expected)
			}
		})
	}
}

func TestMemory_getSlice_ExpandsMemoryIn32ByteChunks(t *testing.T) {
	for increment := 0; increment < 10; increment++ {
		for size := 0; size < 32; size++ {
			for offset := 0; offset < 32; offset++ {
				c := getEmptyContext()
				originalGas := c.gas
				m := NewMemory()
				offsetIncremented := uint64(offset + 32*increment)
				_, err := m.getSlice(offsetIncremented, uint64(size), &c)
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				want := uint64(32 * (increment + 1))
				if size == 0 {
					want = 0
					if c.gas != originalGas {
						t.Error("no gas should have been consumed when size is zero.")
					}
				}
				if size+offset > 32 {
					want = uint64(32 * (increment + 2))
				}
				if m.length() != want {
					t.Errorf("unexpected slice length: %d, want: %d", m.length(), 32*increment)
				}
			}
		}
	}
}

func TestMemory_getSlice_MemoryExpansionDoesNotOverwriteExistingMemory(t *testing.T) {
	c := getEmptyContext()
	m := NewMemory()
	m.store = []byte{0x0, 0x01, 0x02, 0x03, 0x04}
	_, err := m.getSlice(4, 29, &c)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if !bytes.Equal(m.store[:5], []byte{0x0, 0x01, 0x02, 0x03, 0x04}) {
		t.Errorf("unexpected memory value: %x", m.store)
	}
}

func TestMemory_readWord_ReadAndPadWithZeros(t *testing.T) {
	c := getEmptyContext()
	m := NewMemory()
	baseMemory := []byte{0x0, 0x01, 0x02, 0x03, 0x04}
	m.store = bytes.Clone(baseMemory)
	target := new(uint256.Int)
	err := m.readWord(1, target, &c)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if m.length() != 64 {
		t.Errorf("memory should have been expanded to 64 bytes, instead is: %d", m.length())
	}
	if !target.Eq(new(uint256.Int).SetBytes(append(baseMemory[1:], []byte{27: 0x0}...))) {
		t.Errorf("unexpected value: %v", target)
	}
}

func TestMemory_readWord_ErrorCases(t *testing.T) {
	c := getEmptyContext()
	c.gas = 0
	m := NewMemory()
	originalTarget := new(uint256.Int)
	target := originalTarget.Clone()
	err := m.readWord(math.MaxUint64-31, target, &c)
	if !errors.Is(err, errGasUintOverflow) {
		t.Errorf("error should be errGasUintOverflow, instead is: %v", err)
	}
	if target.Cmp(originalTarget) != 0 {
		t.Errorf("target should not have been modified, want %v but got %v", originalTarget, target)
	}
	err = m.readWord(0, target, &c)
	if !errors.Is(err, errOutOfGas) {
		t.Errorf("error should be errOutOfGas, instead is: %v", err)
	}
	if target.Cmp(originalTarget) != 0 {
		t.Errorf("target should not have been modified, want %v but got %v", originalTarget, target)
	}
}
