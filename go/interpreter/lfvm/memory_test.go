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
	"io"
	"math"
	"os"
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
		cost := m.ExpansionCosts(test.size)
		if cost != test.cost {
			t.Errorf("ExpansionCosts(%d) = %d, want %d", test.size, cost, test.cost)
		}
	}
}

func TestMemory_CopyWord_CopiesDataOrReturnsError(t *testing.T) {

	testValue := []byte{0x12, 0x34, 0x56, 0x78, 0x90, 0xab, 0xcd, 0xef}
	testValue2 := append(testValue, bytes.Repeat([]byte{0x00}, 24)...)
	testValue3 := append(
		append(bytes.Repeat([]byte{0x00}, 16), testValue...),
		bytes.Repeat([]byte{0x00}, 8)...)

	tests := map[string]struct {
		gas           tosca.Gas
		offset        uint64
		expectedData  uint256.Int
		expectedError error
	}{
		"regular": {
			gas:           100,
			offset:        0,
			expectedData:  *uint256.NewInt(1).SetBytes8(testValue),
			expectedError: nil},
		"not enough gas": {
			gas:           6,
			offset:        64,
			expectedData:  *uint256.NewInt(1),
			expectedError: errOutOfGas},
		"offset fits in memory": {
			gas:           100,
			offset:        24,
			expectedData:  *uint256.NewInt(1).SetBytes32(testValue2),
			expectedError: nil},
		"offset same as memory size": {
			gas:           100,
			offset:        8,
			expectedData:  *uint256.NewInt(1).SetBytes32(testValue3),
			expectedError: nil},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			ctxt := getEmptyContext()
			m := NewMemory()
			target := uint256.NewInt(1)
			m.store = make([]byte, 32)
			copy(m.store[24:], testValue)
			ctxt.gas = test.gas

			err := m.CopyWord(test.offset, target, &ctxt)
			if !errors.Is(err, test.expectedError) {
				t.Fatalf("unexpected error, want: %v, got: %v", test.expectedError, err)
			}
			if target.Cmp(&test.expectedData) != 0 {
				t.Errorf("unexpected target value, want: %x, got: %x", test.expectedData, *target)
			}
		})
	}
}

func TestMemory_SetByte(t *testing.T) {

	tests := map[string]struct {
		memory   []byte
		value    byte
		offset   uint64
		gas      tosca.Gas
		expected error
	}{
		"regular empty memory": {value: 0x12, offset: 0, gas: 100, expected: nil},
		"not enough gas":       {value: 0x12, offset: 64, gas: 0, expected: errOutOfGas},
		"offset overflow": {value: 0x12, offset: math.MaxUint64, gas: 100,
			expected: errGasUintOverflow},
		"regular with memory": {memory: []byte{0x12, 0x34, 0x56, 0x78, 0x90, 0xab, 0xcd, 0xef},
			value: 0x12, offset: 0, gas: 100, expected: nil},
		"regular with memory and offset": {memory: []byte{0x12, 0x34, 0x56, 0x78, 0x90, 0xab, 0xcd, 0xef},
			value: 0x12, offset: 4, gas: 100, expected: nil},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			ctxt := getEmptyContext()
			m := NewMemory()
			m.store = test.memory
			ctxt.gas = test.gas

			err := m.SetByte(test.offset, test.value, &ctxt)
			if !errors.Is(err, test.expected) {
				t.Errorf("unexpected error, want: %v, got: %v", test.expected, err)
			}
			if err == nil {
				if m.store[test.offset] != test.value {
					t.Errorf("unexpected value, want: %v, got: %v", test.value, m.store[test.offset])
				}
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
			expectedData: testValue},
		"not enough gas": {
			gas:           1,
			offset:        32,
			expectedError: errOutOfGas},
		"offset fits in memory": {
			gas:          6,
			offset:       24,
			expectedData: testValue},
		"offset same as memory size": {
			memory:       append(testValue, testValue...),
			gas:          3,
			expectedData: testValue,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {

			ctxt := getEmptyContext()
			m := NewMemory()
			ctxt.gas = test.gas
			err := m.SetWord(test.offset, uint256.NewInt(1).SetBytes(test.expectedData), &ctxt)
			if !errors.Is(err, test.expectedError) {
				t.Errorf("unexpected error, want: %v, got: %v", test.expectedError, err)
			}
			if err == nil {
				if !bytes.Equal(m.store[test.offset:test.offset+32], test.expectedData) {
					t.Errorf("unexpected value, want: %v, got: %v", test.expectedData, m.store[test.offset])
				}
			}

		})
	}
}

func TestMemory_Set(t *testing.T) {

	tests := map[string]struct {
		size     uint64
		offset   uint64
		expected error
	}{
		"regular":         {size: 8, offset: 0, expected: nil},
		"size overflow":   {size: math.MaxUint64, offset: 1, expected: errGasUintOverflow},
		"offset overflow": {size: 32, offset: math.MaxUint64, expected: errGasUintOverflow},
		"not enough memory": {size: 32, offset: 32,
			expected: errSetMemTooSmall(8, 32, 32)},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			ctxt := getEmptyContext()
			m := NewMemory()
			m.store = make([]byte, 8)
			ctxt.gas = 100

			data := []byte{0x12, 0x34, 0x56, 0x78, 0x90, 0xab, 0xcd, 0xef}
			err := m.Set(test.offset, test.size, data)

			if err == nil {
				if test.expected != nil {
					t.Errorf("expected error %v, got nil", test.expected)
				}
				if m.Len() != test.size+test.offset {
					t.Errorf("unexpected memory size, want: %d, got: %d", test.size+test.offset, m.Len())
				}
				if !bytes.Equal(m.store[test.offset:], data) {
					t.Errorf("unexpected memory value, want: %x, got: %x", data, m.store[test.offset:])
				}
			} else if !errors.Is(err, test.expected) {
				t.Errorf("unexpected error, want: %v, got: %v", test.expected, err)
			}

		})
	}
}

func TestMemory_CopyData(t *testing.T) {

	baseStore := []byte{0x12, 0x34, 0x56, 0x78, 0x90, 0xab, 0xcd, 0xef}
	baseTarget := []byte{0xfe, 0xdc, 0xba, 0x98, 0x76, 0x54, 0x32}

	tests := map[string]struct {
		target   []byte
		store    []byte
		offset   uint64
		expected []byte
	}{
		"regular": {target: baseTarget, store: baseStore, expected: baseStore[:len(baseTarget)]},
		"offset bigger than memory": {target: baseTarget, store: baseStore, offset: 9,
			expected: bytes.Repeat([]byte{0}, len(baseTarget))},
		"padding needed": {store: baseStore,
			offset: 4, target: baseTarget,
			expected: []byte{0x90, 0xab, 0xcd, 0xef, 0, 0, 0}},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			m := NewMemory()
			m.store = test.store
			target := test.target

			m.CopyData(test.offset, target)
			if !bytes.Equal(target, test.expected) {
				t.Errorf("unexpected target value, want: %x, got: %x", test.expected, target)
			}
		})
	}

}

func TestMemory_GetSliceWithCapacity(t *testing.T) {

	baseStore := []byte{0x12, 0x34, 0x56, 0x78, 0x90, 0xab, 0xcd, 0xef}

	tests := map[string]struct {
		store    []byte
		offset   uint64
		size     uint64
		expected []byte
	}{
		"regular":                 {store: baseStore, offset: 0, size: 8, expected: baseStore},
		"regular in empty memory": {store: []byte{}, offset: 0, size: 8, expected: bytes.Repeat([]byte{0}, 8)},
		"size zero":               {store: baseStore, offset: 0, size: 0, expected: nil},
		"size+offset bigger than memory": {store: baseStore, offset: 4, size: 8,
			expected: nil},
		"expand memory": {store: baseStore, offset: 0, size: 12,
			expected: append(baseStore, bytes.Repeat([]byte{0}, 4)...)},
		"offset bigger than memory": {store: baseStore, offset: uint64(len(baseStore) + 2), size: 4,
			expected: nil},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			m := NewMemory()
			m.store = test.store

			result := m.GetSliceWithCapacity(test.offset, test.size)
			if !bytes.Equal(result, test.expected) {
				t.Errorf("unexpected result, want: %x, got: %x", test.expected, result)
			}
		})
	}
}

func TestMemory_Print(t *testing.T) {

	tests := map[string]struct {
		store    []byte
		expected string
	}{
		"empty": {store: []byte{},
			expected: "### mem 0 bytes ###\n-- empty --\n####################\n"},
		"8 bytes": {store: []byte{0x12, 0x34, 0x56, 0x78, 0x90, 0xab, 0xcd, 0xef},
			expected: "### mem 8 bytes ###\n000: 12 34 56 78 90 ab cd ef\n####################\n"},
		"32 bytes": {store: []byte{0x12, 0x34, 0x56, 0x78, 0x90, 0xab, 0xcd, 0xef,
			0x12, 0x34, 0x56, 0x78, 0x90, 0xab, 0xcd, 0xef,
			0x12, 0x34, 0x56, 0x78, 0x90, 0xab, 0xcd, 0xef,
			0x12, 0x34, 0x56, 0x78, 0x90, 0xab, 0xcd, 0xef},
			expected: "### mem 32 bytes ###\n000: 12 34 56 78 90 ab cd ef 12 34 56 78 90 ab cd ef 12 34 56 78 90 ab cd ef 12 34 56 78 90 ab cd ef\n####################\n"},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			m := NewMemory()
			m.store = test.store

			// redirect stdout
			old := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			// Run testing code
			m.Print()
			// read the output
			w.Close()
			out, _ := io.ReadAll(r)
			os.Stdout = old

			if string(out) != test.expected {
				t.Errorf("unexpected output, want: %v, got: %v", test.expected, string(out))
			}

		})
	}

}
