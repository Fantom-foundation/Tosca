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
		cost := m.getExpansionCosts(test.size)
		if cost != test.cost {
			t.Errorf("ExpansionCosts(%d) = %d, want %d", test.size, cost, test.cost)
		}
	}
}

func TestMemory_ReadWord_ReadsDataWithByteAddressingAndZeroPadding(t *testing.T) {
	pattern := make([]byte, 32)
	for i := 0; i < 32; i++ {
		pattern[i] = byte(i + 1)
	}

	for i := 0; i <= 32; i++ {
		ctxt := getEmptyContext()
		ctxt.gas = 100

		m := NewMemory()
		m.expandMemory(0, 32, &ctxt)
		copy(m.store, pattern)

		var target uint256.Int
		err := m.readWord(uint64(i), &target, &ctxt)
		if err != nil {
			t.Fatalf("unexpected error, want: %v, got: %v", nil, err)
		}

		// Check that the memory pattern is shifted by i bytes to the left.
		want := [32]byte{}
		copy(want[:], pattern[i:])
		if got := target.Bytes32(); want != got {
			t.Errorf("unexpected value, want: %x, got: %x", want, got)
		}
	}
}

func TestMemory_ReadWord_ReturnsError(t *testing.T) {

	baseTarget := new(uint256.Int)
	memorySize := uint64(32)

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
			gas:           100,
			offset:        math.MaxUint64,
			expectedError: errGasUintOverflow,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			ctxt := getEmptyContext()
			m := NewMemory()
			target := baseTarget
			m.store = make([]byte, memorySize)
			ctxt.gas = test.gas

			err := m.readWord(test.offset, target, &ctxt)
			if !errors.Is(err, test.expectedError) {
				t.Fatalf("unexpected error, want: %v, got: %v", test.expectedError, err)
			}
			if baseTarget.Cmp(target) != 0 {
				t.Errorf("target should remain unmodified, want: %v, got: %v", baseTarget, target)
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
		"offset 0 in empty memory": {},
		"offset 0 in non-empty memory": {
			memory: baseData,
			offset: 0,
		},
		"non-zero offset in non-empty memory": {
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
			if m.length() < test.offset {
				t.Errorf("unexpected memory size, want: %d, got: %d", test.offset, m.length())
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
			ctxt.gas = test.gas

			err := m.setByte(test.offset, 0x12, &ctxt)
			if !errors.Is(err, test.expected) {
				t.Errorf("unexpected error, want: %v, got: %v", test.expected, err)
			}
			if m.length() != 0 {
				t.Errorf("memory should remain unmodified, want: %d, got: %d", []byte{}, m.store)
			}
		})
	}
}

func TestMemory_SetWord(t *testing.T) {

	mem32byte := []byte{
		0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07,
		0x08, 0x09, 0x0A, 0x0B, 0x0C, 0x0D, 0x0E, 0x0F,
		0x10, 0x11, 0x12, 0x13, 0x14, 0x15, 0x16, 0x17,
		0x18, 0x19, 0x1A, 0x1B, 0x1C, 0x1D, 0x1E, 0x1F}

	tests := map[string]struct {
		memory       []byte
		offset       uint64
		expectedData []byte
	}{
		"write word at offset 0": {
			expectedData: mem32byte,
		},
		"expand memory and pad": {
			memory: []byte{31: 0x0},
			offset: 24,
			expectedData: append(
				append([]byte{23: 0x0}, mem32byte...),
				[]byte{7: 0x0}...)}, // 64 bytes mem.
		"offset larger than memory": {
			memory: mem32byte,
			offset: 33,
			expectedData: append(
				append(
					append(mem32byte, 0x00),
					mem32byte...),
				[]byte{30: 0x0}...),
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {

			ctxt := getEmptyContext()
			m := NewMemory()
			if test.memory != nil {
				m.store = test.memory
			}
			ctxt.gas = 9
			err := m.setWord(test.offset, new(uint256.Int).SetBytes(mem32byte), &ctxt)
			if err != nil {
				t.Errorf("unexpected error, want: %v, got: %v", nil, err)
			}
			if !bytes.Equal(m.store, test.expectedData) {
				t.Errorf("unexpected memory, want: %v, got: %v", test.expectedData, m.store)
			}
		})
	}
}

func TestMemory_SetWord_ReportsOutOfGas(t *testing.T) {

	testValue := []byte{
		0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07,
		0x08, 0x09, 0x0A, 0x0B, 0x0C, 0x0D, 0x0E, 0x0F,
		0x10, 0x11, 0x12, 0x13, 0x14, 0x15, 0x16, 0x17,
		0x18, 0x19, 0x1A, 0x1B, 0x1C, 0x1D, 0x1E, 0x1F}

	ctxt := getEmptyContext()
	m := NewMemory()
	m.store = make([]byte, 8)
	ctxt.gas = 1
	err := m.setWord(32, new(uint256.Int).SetBytes(testValue), &ctxt)
	if !errors.Is(err, errOutOfGas) {
		t.Errorf("unexpected error, want: %v, got: %v", errOutOfGas, err)
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

	if m.length() != size+offset {
		t.Errorf("unexpected memory size, want: %d, got: %d", size+offset, m.length())
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

func TestMemory_ReadData(t *testing.T) {

	baseStore := []byte{0x12, 0x34, 0x56, 0x78, 0x90, 0xab, 0xcd, 0xef}
	baseTarget := []byte{0xfe, 0xdc, 0xba, 0x98, 0x76, 0x54, 0x32}

	tests := map[string]struct {
		offset   uint64
		expected []byte
	}{
		"regular": {
			expected: baseStore[:len(baseTarget)],
		},
		"offset bigger than memory": {
			offset:   9,
			expected: bytes.Repeat([]byte{0}, len(baseTarget)),
		},
		"padding needed": {
			offset:   4,
			expected: []byte{0x90, 0xab, 0xcd, 0xef, 0, 0, 0},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			m := NewMemory()
			m.store = bytes.Clone(baseStore)
			target := bytes.Clone(baseTarget)

			m.readData(test.offset, target)
			if !bytes.Equal(target, test.expected) {
				t.Errorf("unexpected target value, want: %x, got: %x", test.expected, target)
			}
			if !bytes.Equal(m.store, baseStore) {
				t.Errorf("read must not modify the memory, want: %x, got: %x", baseStore, m.store)
			}
		})
	}

}

func TestMemory_expandMemoryWithoutCharging(t *testing.T) {

	test := map[string]struct {
		initialMem  []byte
		expectedMem []byte
	}{
		"empty-memory-increases": {
			initialMem:  []byte{},
			expectedMem: []byte{31: 0x0},
		},
		"memory-bigger-than-size": {
			initialMem:  []byte{63: 0x0},
			expectedMem: []byte{63: 0x0},
		},
	}

	for name, test := range test {
		t.Run(name, func(t *testing.T) {
			m := NewMemory()
			size := uint64(32)
			fee := m.getExpansionCosts(size)
			m.store = test.initialMem
			m.expandMemoryWithoutCharging(size, fee)
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
			size:     2,
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
				t.Errorf("unexpected memory size, want: %d, got: %d", 0, m.length())
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
		"empty-memory": {},
		"memory-bigger-than-size": {
			size:          32,
			offset:        64,
			initialMemory: []byte{127: 0x0},
		},
		"size-zero": {
			size:   0,
			offset: 32,
		},
		"expand-memory": {
			size:   32,
			offset: 0,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			ctxt := getEmptyContext()
			m := NewMemory()
			ctxt.gas = 100

			err := m.expandMemory(test.offset, test.size, &ctxt)
			if err != nil {
				t.Fatalf("unexpected error, want: %v, got: %v", nil, err)
			}
			if test.size != 0 && m.length() < test.size+test.offset {
				t.Errorf("memory size should have increased, want: %d, got: %d", test.size+test.offset, m.length())
			} else if test.size == 0 && m.length() != uint64(len(test.initialMemory)) {
				t.Errorf("memory size should not have changed, want: %d, got: %d", len(test.initialMemory), m.length())
			}
		})
	}
}
