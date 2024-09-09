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

func TestMemory_ExpansionCosts_ComputesCorrectCosts(t *testing.T) {

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
		cost := m.expansionCosts(test.size)
		if cost != test.cost {
			t.Errorf("ExpansionCosts(%d) = %d, want %d", test.size, cost, test.cost)
		}
	}
}

func TestMemory_GetWord_CopiesData(t *testing.T) {

	valueSmall := uint256.NewInt(0x1223457890abcdef)
	valueMiddle := uint256.NewInt(0).Lsh(valueSmall, 64)
	valueBig := uint256.NewInt(0).Lsh(valueSmall, 256-16)
	memorySize := uint64(32)

	tests := map[string]struct {
		offset       uint64
		expectedData *uint256.Int
	}{
		"regular": {
			offset:       0,
			expectedData: valueSmall,
		},
		"small offset": {
			offset:       memorySize / 4,
			expectedData: valueMiddle,
		},
		"big offset crops value": {
			offset:       memorySize - 2,
			expectedData: valueBig,
		},
		"big offset returns zero": {
			offset:       memorySize,
			expectedData: uint256.NewInt(0),
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			ctxt := getEmptyContext()
			m := NewMemory()
			target := uint256.NewInt(1)
			m.store = make([]byte, memorySize)
			copy(m.store[24:], valueSmall.Bytes())
			ctxt.gas = 100

			err := m.getWord(test.offset, target, &ctxt)
			if err != nil {
				t.Fatalf("unexpected error, want: %v, got: %v", nil, err)
			}
			if target.Cmp(test.expectedData) != 0 {
				t.Errorf("unexpected target value, want: %x, got: %x", *test.expectedData, *target)
			}
		})
	}
}

func TestMemory_GetWord_ReturnsError(t *testing.T) {

	valueSmall := uint256.NewInt(0x1223457890abcdef)
	baseTargetValue := uint256.NewInt(1)
	memorySize := uint64(32)
	enoughGas := tosca.Gas(100)

	tests := map[string]struct {
		gas           tosca.Gas
		offset        uint64
		expectedError error
	}{
		"not enough gas": {
			gas:           6,
			offset:        uint64(memorySize * 2),
			expectedError: errOutOfGas,
		},
		"offset overflow": {
			gas:           enoughGas,
			offset:        math.MaxUint64,
			expectedError: errGasUintOverflow,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			ctxt := getEmptyContext()
			m := NewMemory()
			target := baseTargetValue
			m.store = make([]byte, memorySize)
			copy(m.store[24:], valueSmall.Bytes())
			ctxt.gas = test.gas

			err := m.getWord(test.offset, target, &ctxt)
			if !errors.Is(err, test.expectedError) {
				t.Fatalf("unexpected error, want: %v, got: %v", test.expectedError, err)
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
		"regular empty memory": {},
		"regular with memory": {
			memory: baseData,
			offset: 0,
		},
		"regular with memory and offset": {
			memory: baseData,
			offset: 4,
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
			m.store = test.memory
			ctxt.gas = tosca.Gas(100)
			value := byte(0x12)

			err := m.setByte(test.offset, value, &ctxt)
			if err != nil {
				t.Errorf("unexpected error, want: %v, got: %v", nil, err)
			}
			if m.len() < test.offset {
				t.Errorf("unexpected memory size, want: %d, got: %d", test.offset, m.len())
			}
			if m.store[test.offset] != value {
				t.Errorf("unexpected value, want: %v, got: %v", value, m.store[test.offset])
			}
		})
	}
}

func TestMemory_SetByte_ErrorCases(t *testing.T) {

	tests := map[string]struct {
		offset   uint64
		gas      tosca.Gas
		expected error
	}{
		"not enough gas": {
			offset:   64,
			gas:      0,
			expected: errOutOfGas,
		},
		"offset overflow": {
			offset:   math.MaxUint64,
			gas:      100,
			expected: errGasUintOverflow,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			ctxt := getEmptyContext()
			m := NewMemory()
			m.store = []byte{0x12, 0x34, 0x56, 0x78, 0x90, 0xab, 0xcd, 0xef}
			ctxt.gas = test.gas

			err := m.setByte(test.offset, 0x12, &ctxt)
			if !errors.Is(err, test.expected) {
				t.Errorf("unexpected error, want: %v, got: %v", test.expected, err)
			}
		})
	}
}

func TestMemory_SetWord_ReportsNotEnoughGas(t *testing.T) {

	testValue := []byte{
		0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07,
		0x08, 0x09, 0x0A, 0x0B, 0x0C, 0x0D, 0x0E, 0x0F,
		0x10, 0x11, 0x12, 0x13, 0x14, 0x15, 0x16, 0x17,
		0x18, 0x19, 0x1A, 0x1B, 0x1C, 0x1D, 0x1E, 0x1F}

	tests := map[string]struct {
		memory        []byte
		gas           tosca.Gas
		offset        uint64
		expectedData  []byte
		expectedError error
	}{
		"regular": {
			gas:          3,
			expectedData: testValue,
		},
		"not enough gas": {
			gas:           1,
			offset:        32,
			expectedError: errOutOfGas,
		},
		"offset fits in memory": {
			gas:    6,
			offset: 24,
			expectedData: append(
				append([]byte{23: 0x0}, testValue...),
				[]byte{7: 0x0}...)},
		"offset same as memory size": {
			memory:       testValue,
			gas:          6,
			offset:       32,
			expectedData: append(testValue, testValue...),
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {

			ctxt := getEmptyContext()
			m := NewMemory()
			if test.memory != nil {
				m.store = test.memory
			}
			ctxt.gas = test.gas
			err := m.setWord(test.offset, new(uint256.Int).SetBytes(testValue), &ctxt)
			if !errors.Is(err, test.expectedError) {
				t.Errorf("unexpected error, want: %v, got: %v", test.expectedError, err)
			}
			if err == nil {
				if !bytes.Equal(m.store, test.expectedData) {
					t.Errorf("unexpected value, want: %v, got: %v", test.expectedData, m.store)
				}
			}

		})
	}
}

func TestMemory_Set_SuccessfulCases(t *testing.T) {

	size := uint64(8)
	offset := uint64(0)

	ctxt := getEmptyContext()
	m := NewMemory()
	m.store = make([]byte, 8)
	ctxt.gas = 100

	data := []byte{0x12, 0x34, 0x56, 0x78, 0x90, 0xab, 0xcd, 0xef}
	err := m.set(offset, size, data)

	if err != nil {
		t.Fatalf("unexpected error, want: %v, got: %v", nil, err)
	}

	if m.len() != size+offset {
		t.Errorf("unexpected memory size, want: %d, got: %d", size+offset, m.len())
	}
	if !bytes.Equal(m.store[offset:], data) {
		t.Errorf("unexpected memory value, want: %x, got: %x", data, m.store[offset:])
	}

}

func TestMemory_Set_ErrorCases(t *testing.T) {

	tests := map[string]struct {
		size     uint64
		offset   uint64
		expected error
	}{
		"size overflow": {
			size:     math.MaxUint64,
			offset:   1,
			expected: errGasUintOverflow,
		},
		"offset overflow": {
			size:     32,
			offset:   math.MaxUint64,
			expected: errGasUintOverflow,
		},
		"not enough memory": {
			size:     32,
			offset:   32,
			expected: makeInsufficientMemoryError(8, 32, 32),
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			ctxt := getEmptyContext()
			m := NewMemory()
			m.store = make([]byte, 8)
			ctxt.gas = 100

			data := []byte{0x12, 0x34, 0x56, 0x78, 0x90, 0xab, 0xcd, 0xef}
			err := m.set(test.offset, test.size, data)

			if !errors.Is(err, test.expected) {
				t.Errorf("unexpected error, want: %v, got: %v", test.expected, err)
			}

		})
	}
}

func TestMemory_GetData(t *testing.T) {

	baseStore := []byte{0x12, 0x34, 0x56, 0x78, 0x90, 0xab, 0xcd, 0xef}
	baseTarget := []byte{0xfe, 0xdc, 0xba, 0x98, 0x76, 0x54, 0x32}

	tests := map[string]struct {
		target   []byte
		store    []byte
		offset   uint64
		expected []byte
	}{
		"regular": {
			target:   bytes.Clone(baseTarget),
			store:    baseStore,
			expected: baseStore[:len(baseTarget)],
		},
		"offset bigger than memory": {
			target:   bytes.Clone(baseTarget),
			store:    baseStore,
			offset:   9,
			expected: bytes.Repeat([]byte{0}, len(baseTarget)),
		},
		"padding needed": {
			target:   bytes.Clone(baseTarget),
			store:    baseStore,
			offset:   4,
			expected: []byte{0x90, 0xab, 0xcd, 0xef, 0, 0, 0},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			m := NewMemory()
			m.store = test.store
			target := test.target

			m.getData(test.offset, target)
			if !bytes.Equal(target, test.expected) {
				t.Errorf("unexpected target value, want: %x, got: %x", test.expected, target)
			}
		})
	}

}
