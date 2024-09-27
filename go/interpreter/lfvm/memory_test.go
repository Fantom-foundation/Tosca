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
	"crypto/rand"
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
		err  error
	}{
		{0, 0, nil},
		{1, 3, nil},
		{32, 3, nil},
		{33, 6, nil},
		{64, 6, nil},
		{65, 9, nil},
		{22 * 32, 3 * 22, nil},             // last word size without square cost
		{23 * 32, (23*23)/512 + 3*23, nil}, // fist word size with square cost
		{maxMemoryExpansionSize - 33, 36028809870311418, nil},
		{maxMemoryExpansionSize - 1, 36028809887088637, nil},
		{maxMemoryExpansionSize, 36028809887088637, nil}, // magic number, max cost
		{maxMemoryExpansionSize + 1, 0, errMaxMemoryExpansionSize},
		{math.MaxInt64, 0, errMaxMemoryExpansionSize},
	}

	for _, test := range tests {

		m := NewMemory()
		cost, err := m.getExpansionCosts(test.size)
		if !errors.Is(err, test.err) {
			t.Errorf("unexpected error: want: %v but got: %v", test.err, err)
		}
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
			fee, err := m.getExpansionCosts(test.size)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			err = m.expandMemoryWithoutCharging(test.size)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
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
			expected: errOverflow,
		},
		"size overflow": {
			size:     math.MaxUint64,
			offset:   1,
			gas:      100,
			expected: errOverflow,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			ctxt := context{gas: test.gas}
			m := NewMemory()

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
			ctxt := context{gas: 3}
			m := NewMemory()
			m.store = make([]byte, test.initialMemorySize)

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
	c := context{gas: 0}
	m := NewMemory()
	_, err := m.getSlice(0, 1, &c)
	if !errors.Is(err, errOutOfGas) {
		t.Errorf("error should be errOutOfGas, instead is: %v", err)
	}
	_, err = m.getSlice(math.MaxUint64-31, 32, &c)
	if !errors.Is(err, errOverflow) {
		t.Errorf("error should be errOverflow, instead is: %v", err)
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
			c := context{gas: 3}
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
	for memSize := uint64(0); memSize < 128; memSize += 32 {
		for offset := 0; offset < 128; offset++ {
			for size := 1; size < 32; size++ {
				c := &context{gas: 15}
				m := NewMemory()
				m.store = make([]byte, memSize)
				_, err := m.getSlice(uint64(offset), uint64(size), c)
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				want, err := toValidMemorySize(uint64(offset + size))
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if want < memSize {
					want = memSize
				}
				if got, want := m.length(), want; got != want {
					t.Errorf("unexpected memory length: %d, want: %d", got, want)
				}
			}
		}
	}
}

func TestMemory_getSlice_SizeOfZeroIsNotGrowingTheMemory(t *testing.T) {
	for memSize := 0; memSize < 128; memSize += 32 {
		for offset := 0; offset < 128; offset++ {
			c := &context{gas: 1}
			m := NewMemory()
			m.store = make([]byte, memSize)
			_, err := m.getSlice(uint64(offset), 0, c)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if c.gas != 1 {
				t.Error("no gas should have been consumed when size is zero.")
			}
			if got, want := m.length(), uint64(memSize); got != want {
				t.Errorf("unexpected memory length: %d, want: %d", got, want)
			}
		}
	}
}

func TestMemory_getSlice_MemoryExpansionDoesNotOverwriteExistingMemory(t *testing.T) {
	c := context{gas: 6}
	m := NewMemory()
	m.store = []byte{0x0, 0x01, 0x02, 0x03, 0x04}
	_, err := m.getSlice(4, 29, &c)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if m.length() != 64 {
		t.Errorf("memory should have been expanded to 64 bytes, instead is: %d", m.length())
	}
	if !bytes.Equal(m.store[:5], []byte{0x0, 0x01, 0x02, 0x03, 0x04}) {
		t.Errorf("unexpected memory value: %x", m.store)
	}
}

func TestMemory_getSlice_ExpandsWithZeros(t *testing.T) {
	c := context{gas: 6}
	m := NewMemory()
	baseMemory := []byte{0x0, 0x01, 0x02, 0x03, 0x04}
	m.store = bytes.Clone(baseMemory)
	_, err := m.getSlice(28, 8, &c)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if m.length() != 64 {
		t.Errorf("memory should have been expanded to 64 bytes, instead is: %d", m.length())
	}
	if !bytes.Equal(m.store, append(baseMemory, []byte{58: 0x0}...)) {
		t.Errorf("unexpected memory value: %x", m.store)
	}
}

func TestMemory_readWord_ErrorCases(t *testing.T) {
	c := context{gas: 0}
	m := NewMemory()
	originalTarget := uint256.NewInt(1)
	target := originalTarget.Clone()
	err := m.readWord(math.MaxUint64-31, target, &c)
	if !errors.Is(err, errOverflow) {
		t.Errorf("error should be errOverflow, instead is: %v", err)
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

func TestToValidMemorySize_RoundsUpToNextMultipleOf32(t *testing.T) {
	for i := uint64(0); i < 128; i++ {
		got, err := toValidMemorySize(i)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if got%32 != 0 {
			t.Errorf("result should be a multiple of 32, got: %d", got)
		}
		if got < i {
			t.Errorf("result should be greater or equal to input, got: %d", got)
		}
		if got-i >= 32 {
			t.Errorf("result should be less than 32 bytes greater than input, got: %d", got)
		}
	}
}

func TestToValidateMemorySize_ReturnsOverflowError(t *testing.T) {
	_, err := toValidMemorySize(math.MaxUint64 - 30)
	if !errors.Is(err, errOverflow) {
		t.Errorf("error should be errOverflow, instead is: %v", err)
	}
}

func TestMemory_set_UpdatesDataInMemoryAtGivenOffset(t *testing.T) {
	before := generateRandomBytes(128)
	for offset := uint64(0); offset < 128; offset++ {
		for size := 0; size < int(128-offset); size++ {
			data := generateRandomBytes(size)

			// test the memory update
			m := &Memory{store: bytes.Clone(before)}
			if err := m.set(offset, data, nil); err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			// check if only the data at the given offset has changed
			want := bytes.Clone(before)
			copy(want[offset:], data)
			if !bytes.Equal(m.store, want) {
				t.Errorf("unexpected memory value after set, want: %x, got: %x", want, m.store)
			}
		}
	}
}

func TestMemory_set_PreservesMemoryWhenGrowingAndPadsWithZeros(t *testing.T) {
	before := generateRandomBytes(32)
	m := &Memory{store: bytes.Clone(before)}
	if err := m.set(64, []byte{0x1, 0x2}, &context{gas: 100}); err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	want := bytes.Clone(before)              // < preserved data
	want = append(want, make([]byte, 32)...) // < zero-padding before the new data
	want = append(want, []byte{0x1, 0x2}...) // < the new data
	want = append(want, make([]byte, 30)...) // < zero-padding after the new data

	if !bytes.Equal(m.store, want) {
		t.Errorf("unexpected memory value after set, want: %x, got: %x", want, m.store)
	}
}

func TestMemory_set_IgnoresEmptyData(t *testing.T) {
	before := generateRandomBytes(32)
	for _, offset := range []uint64{0, 32, 64} {
		for _, data := range [][]byte{nil, {}} {
			m := &Memory{store: bytes.Clone(before)}
			if err := m.set(offset, data, nil); err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if !bytes.Equal(m.store, before) {
				t.Errorf("unexpected no change in data, want: %x, got: %x", before, m.store)
			}
		}
	}
}

func TestMemory_set_FailsIfThereIsNotEnoughGasToGrow(t *testing.T) {
	c := &context{gas: 2}
	m := &Memory{store: make([]byte, 32)}
	if err := m.set(64, []byte{0x1}, c); !errors.Is(err, errOutOfGas) {
		t.Errorf("unexpected error %v, got %v", errOutOfGas, err)
	}
	if len(m.store) != 32 {
		t.Errorf("memory should not have been expanded, instead is: %d", len(m.store))
	}
}

func TestMemory_set_FailsIfOffsetLeadsToOverflow(t *testing.T) {
	m := &Memory{store: make([]byte, 32)}
	data := []byte{0x1, 0x2, 0x3}
	offset := math.MaxUint64 - uint64(len(data)) + 1
	if err := m.set(offset, data, nil); !errors.Is(err, errOverflow) {
		t.Errorf("unexpected error %v, got %v", errOutOfGas, err)
	}
	if len(m.store) != 32 {
		t.Errorf("memory should not have been expanded, instead is: %d", len(m.store))
	}
}

func generateRandomBytes(size int) []byte {
	data := make([]byte, size)
	_, _ = rand.Read(data)
	return data
}
