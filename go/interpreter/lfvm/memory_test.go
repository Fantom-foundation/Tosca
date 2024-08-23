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
	"fmt"
	"io"
	"math"
	"os"
	"slices"
	"strings"
	"testing"

	"github.com/Fantom-foundation/Tosca/go/tosca"
	"github.com/holiman/uint256"
)

func TestMemory_ExpansionCosts(t *testing.T) {

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

func TestMemory_CopyWord(t *testing.T) {

	testValue := []byte{0x12, 0x34, 0x56, 0x78, 0x90, 0xab, 0xcd, 0xef}

	tests := map[string]struct {
		setup          func(*Memory, *context)
		offset         uint64
		expectedTarget uint256.Int
		expectedError  error
	}{
		"regular": {
			setup: func(m *Memory, ctxt *context) {
				m.store = make([]byte, 32)
				copy(m.store[24:], testValue)
				ctxt.gas = 100
			},
			offset:         0,
			expectedTarget: *uint256.NewInt(0).SetBytes8(testValue),
			expectedError:  nil},
		"not enough gas": {
			setup: func(m *Memory, ctxt *context) {
				m.store = make([]byte, 32)
				copy(m.store[24:], testValue)
				ctxt.gas = 6
			},
			offset:         64,
			expectedTarget: *uint256.NewInt(0),
			expectedError:  errOutOfGas},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			ctxt := getEmptyContext()
			m := NewMemory()
			target := uint256.NewInt(0)
			test.setup(m, &ctxt)

			err := m.CopyWord(test.offset, target, &ctxt)
			if err != test.expectedError {
				t.Errorf("unexpected error, want: %v, got: %v", test.expectedError, err)
			}
			if target.Cmp(&test.expectedTarget) != 0 {
				t.Errorf("unexpected target value, want: %v, got: %v", test.expectedTarget, target)
			}
		})
	}
}

func TestMemory_SetByte(t *testing.T) {

	tests := map[string]struct {
		value    byte
		offset   uint64
		gas      tosca.Gas
		expected error
	}{
		"regular":        {value: 0x12, offset: 0, gas: 100, expected: nil},
		"not enough gas": {value: 0x12, offset: 64, gas: 0, expected: errOutOfGas},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			ctxt := getEmptyContext()
			m := NewMemory()
			ctxt.gas = test.gas

			err := m.SetByte(test.offset, test.value, &ctxt)
			if err != test.expected {
				t.Errorf("unexpected error, want: %v, got: %v", test.expected, err)
			}
			if err == nil && m.store[test.offset] != test.value {
				t.Errorf("unexpected value, want: %v, got: %v", test.value, m.store[test.offset])
			}
		})
	}
}

func TestSetWordProperlyReportsNotEnoughGas(t *testing.T) {
	ctxt := getEmptyContext()
	m := NewMemory()
	ctxt.gas = 0
	err := m.SetWord(0, uint256.NewInt(0), &ctxt)
	if err != errOutOfGas {
		t.Errorf("unexpected error, want: %v, got: %v", errOutOfGas, err)
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
			expected: fmt.Errorf("memory too small, size %d, attempted to write %d bytes at %d", 8, 32, 32)},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			ctxt := getEmptyContext()
			m := NewMemory()
			m.store = make([]byte, 8)
			ctxt.gas = 100

			err := m.Set(test.offset, test.size, []byte{0x12, 0x34, 0x56, 0x78, 0x90, 0xab, 0xcd, 0xef})

			if err == nil {
				if test.expected != nil {
					t.Errorf("expected error %v, got nil", test.expected)
				}
			} else if !strings.Contains(err.Error(), test.expected.Error()) {
				t.Errorf("unexpected error, want: %v, got: %v", test.expected, err)
			}
		})
	}
}

func TestMemory_CopyData(t *testing.T) {

	baseStore := []byte{0x12, 0x34, 0x56, 0x78, 0x90, 0xab, 0xcd, 0xef}

	tests := map[string]struct {
		store    []byte
		offset   uint64
		target   []byte
		expected []byte
	}{
		"regular": {store: baseStore, offset: 0, target: make([]byte, 8),
			expected: baseStore},
		"offset bigger than memory": {store: baseStore, offset: 9, target: make([]byte, 8),
			expected: []byte{0, 0, 0, 0, 0, 0, 0, 0}},
		"padding needed": {store: baseStore,
			offset: 4, target: make([]byte, 8),
			expected: []byte{0x90, 0xab, 0xcd, 0xef, 0, 0, 0, 0}},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			m := NewMemory()
			m.store = test.store
			target := make([]byte, 8)

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
		"regular":   {store: baseStore, offset: 0, size: 8, expected: baseStore},
		"size zero": {store: baseStore, offset: 0, size: 0, expected: nil},
		"size+offset bigger than memory": {store: baseStore, offset: 4, size: 8,
			expected: nil},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			m := NewMemory()
			m.store = test.store

			result := m.GetSliceWithCapacity(test.offset, test.size)
			if !slices.Equal(result, test.expected) {
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
