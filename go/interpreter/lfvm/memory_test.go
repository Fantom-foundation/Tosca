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

	"github.com/Fantom-foundation/Tosca/go/tosca"
	"github.com/holiman/uint256"
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
		cost := m.ExpansionCosts(test.size)
		if cost != test.cost {
			t.Errorf("ExpansionCosts(%d) = %d, want %d", test.size, cost, test.cost)
		}
	}
}

func TestCopyWord(t *testing.T) {

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

func TestSetByte(t *testing.T) {

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

// func TestSet(t *testing.T) {

// 	tests := map[string]struct {
// 		size     uint64
// 		offset   uint64
// 		expected error
// 	}{
// 		"regular":         {size: 8, offset: 0, expected: nil},
// 		"size overflow":   {size: math.MaxUint64, offset: 1, expected: errGasUintOverflow},
// 		"offset overflow": {size: 32, offset: math.MaxUint64, expected: errGasUintOverflow},
// 		"not enough memory": {size: 32, offset: 32,
// 			expected: fmt.Errorf("memory too small, size %d, attempted to write %d bytes at %d", 8, 32, 32)},
// 	}

// 	for name, test := range tests {
// 		t.Run(name, func(t *testing.T) {
// 			ctxt := getEmptyContext()
// 			m := NewMemory()
// 			m.store = make([]byte, 8)
// 			ctxt.gas = 100

// 			err := m.Set(test.offset, test.size, []byte{0x12, 0x34, 0x56, 0x78, 0x90, 0xab, 0xcd, 0xef})
// 			if !strings.Contains(err.Error(), test.expected.Error()) {
// 				t.Errorf("unexpected error, want: %v, got: %v", test.expected, err)
// 			}
// 		})
// 	}

// }
