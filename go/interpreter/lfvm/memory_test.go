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
	"strings"
	"testing"

	"github.com/Fantom-foundation/Tosca/go/tosca"
)

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

func TestMemory_SetByte_SuccessfulCases(t *testing.T) {

	baseData := []byte{0x12, 0x34, 0x56, 0x78, 0x90, 0xab, 0xcd, 0xef}

	tests := map[string]struct {
		memory []byte
		offset uint64
	}{
		"write to empty memory": {},
		"write to first element of memory": {
			memory: baseData,
			offset: 0,
		},
		"write with offset within memory": {
			memory: baseData,
			offset: 4,
		},
		"offset trigger expansion of empty memory": {
			offset: 2,
		},
		"offset trigger memory expansion": {
			memory: baseData[:4],
			offset: 8,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			ctxt := getEmptyContext()
			m := NewMemory()
			m.store = bytes.Clone(test.memory)
			ctxt.gas = tosca.Gas(3)
			value := byte(0xff)

			err := m.setByte(test.offset, value, &ctxt)
			if err != nil {
				t.Errorf("unexpected error, want: %v, got: %v", nil, err)
			}
			// if memory expansion is needed, it expands up to offset+1
			// if no expansion is needed, is because memory is already large enough
			if m.length() <= test.offset {
				t.Errorf("unexpected memory size, want: %d, got: %d", test.offset, m.length())
			}
			if test.memory != nil {
				for i, b := range test.memory {
					if i == int(test.offset) {
						if m.store[i] != value {
							t.Errorf("unexpected value, want: %v, got: %v", value, m.store[i])
						}
					} else {
						if m.store[i] != b {
							t.Errorf("unexpected value at position %v, want: %v, got: %v", i, b, m.store[i])
						}
					}
				}
			} else {
				// for empty memories expansion we only want to check the offset byte.
				if m.store[test.offset] != value {
					t.Errorf("unexpected value, want: %v, got: %v", value, m.store[test.offset])
				}
			}
		})
	}
}

func TestMemory_SetByte_ErrorCases(t *testing.T) {
	ctxt := getEmptyContext()
	m := NewMemory()
	ctxt.gas = 0
	err := m.setByte(math.MaxUint64, 0x12, &ctxt)
	if !errors.Is(err, errGasUintOverflow) {
		t.Errorf("unexpected error, want: %v, got: %v", errGasUintOverflow, err)
	}
	err = m.setByte(0, 0x12, &ctxt)
	if !errors.Is(err, errOutOfGas) {
		t.Errorf("unexpected error, want: %v, got: %v", errGasUintOverflow, err)
	}
}

func TestMemory_Set_Successful(t *testing.T) {
	c := getEmptyContext()
	c.gas = 4
	m := NewMemory()
	offset := uint64(0)
	value := []byte{0xff, 0xee}
	size := uint64(len(value))
	err := m.set(offset, size, value, &c)
	if err != nil {
		t.Fatalf("unexpected error, want: %v, got: %v", nil, err)
	}
	if m.length() != 32 {
		t.Errorf("set should not change memory size, want: %d, got: %d", 32, m.length())
	}
	if !bytes.Equal(m.store[offset:offset+size], value) {
		t.Errorf("unexpected memory value, want: %x, got: %x", value, m.store[offset:offset+size])
	}
	if c.gas != 1 {
		t.Errorf("unexpected gas value, want: %d, got: %d", 1, c.gas)
	}
}

func TestMemory_Set_ErrorCases(t *testing.T) {
	c := getEmptyContext()
	c.gas = 0
	m := NewMemory()

	err := m.set(0, 1, []byte{}, &c)
	if !errors.Is(err, errOutOfGas) {
		t.Errorf("unexpected error, want: %v, got: %v", errOutOfGas, err)
	}
	err = m.set(math.MaxUint64, 1, []byte{}, &c)
	if !errors.Is(err, errGasUintOverflow) {
		t.Errorf("unexpected error, want: %v, got: %v", errGasUintOverflow, err)
	}
	err = m.set(1, math.MaxUint64, []byte{}, &c)
	if !errors.Is(err, errGasUintOverflow) {
		t.Errorf("unexpected error, want: %v, got: %v", errGasUintOverflow, err)
	}
}

func TestMemory_TrySet_SuccessfulCases(t *testing.T) {

	memoryOriginalSize := uint64(8)
	offset := uint64(1)
	// data of size (memoryOriginalSize - offset), to ensure it would fit in memory
	data := make([]byte, memoryOriginalSize-offset)
	for i := range data {
		// add non zero values.
		data[i] = byte(i + 1)
	}
	size := uint64(len(data))

	m := NewMemory()
	m.store = make([]byte, memoryOriginalSize)

	err := m.trySet(offset, size, data)
	if err != nil {
		t.Fatalf("unexpected error, want: %v, got: %v", nil, err)
	}

	if m.length() != memoryOriginalSize {
		t.Errorf("set should not change memory size, want: %d, got: %d", memoryOriginalSize, m.length())
	}
	if want := append([]byte{0x0}, data...); !bytes.Equal(m.store, want) {
		t.Errorf("unexpected memory value, want: %x, got: %x", want, m.store)
	}

}

func TestMemory_TrySet_ErrorCases(t *testing.T) {
	m := NewMemory()
	m.store = make([]byte, 8)
	// since we are only testing for failed cases, data is not relevant because
	// the internal checks are done with offset and size parameters.

	// size overflow
	err := m.trySet(1, math.MaxUint64, []byte{})
	if !errors.Is(err, errGasUintOverflow) {
		t.Errorf("unexpected error, want: %v, got: %v", errGasUintOverflow, err)
	}
	// offset overflow
	err = m.trySet(math.MaxUint64, 1, []byte{})
	if !errors.Is(err, errGasUintOverflow) {
		t.Errorf("unexpected error, want: %v, got: %v", errGasUintOverflow, err)
	}
	// not enough memory
	err = m.trySet(32, 32, []byte{})
	if !strings.Contains(err.Error(), "memory too small") {
		t.Errorf("unexpected error, want: memory too small, got: %v", err)
	}
}
