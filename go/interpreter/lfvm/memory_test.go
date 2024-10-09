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

func TestGetExpansionCostsAndSize(t *testing.T) {
	type testDef struct {
		size uint64
		cost tosca.Gas
	}

	tests := map[string]struct {
		tests []testDef
		err   error
	}{
		"zero size does not expand": {
			tests: []testDef{
				{size: 0},
			},
		},
		"in-rage size can be expanded": {
			tests: []testDef{
				{size: 1, cost: 3},
				{size: 32, cost: 3},
				{size: 33, cost: 6},
				{size: 64, cost: 6},
				{size: 65, cost: 9},
				{size: 22 * 32, cost: 3 * 22},
				{size: 23 * 32, cost: (23*23)/512 + 3*23},
				{size: maxMemoryExpansionSize - 33, cost: 36028809870311418},
				{size: maxMemoryExpansionSize - 1, cost: 36028809887088637},
				{size: maxMemoryExpansionSize, cost: 36028809887088637},
			},
		},
		"larger size than memory size limit yields an error": {
			tests: []testDef{
				{size: maxMemoryExpansionSize + 1},
				{size: math.MaxInt64},
			},
			err: errMaxMemoryExpansionSize,
		},
		"size overflowing word count computation yields an error": {
			tests: []testDef{
				{size: math.MaxUint64},
			},
			err: errOverflow,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			expectedError := test.err

			for _, test := range test.tests {

				m := NewMemory()
				cost, size, err := m.getExpansionCostsAndSize(test.size)
				if !errors.Is(err, expectedError) {
					t.Errorf("unexpected error: want: %v but got: %v", expectedError, err)
				}

				if err != nil {
					continue
				}

				// all expansions must be done in 32-byte chunks
				expectedSize := (test.size + 31) / 32 * 32
				if want, got := expectedSize, size; want != got {
					t.Errorf("unexpected size: want: %d but got: %d", want, got)
				}

				// cost must be calculated by the formula
				words := tosca.SizeInWords(test.size)
				expectedCost := tosca.Gas((words*words)/512 + 3*words)
				if want, got := expectedCost, cost; want != got {
					t.Errorf("unexpected cost: want: %d but got: %d", want, got)
				}
			}
		})
	}
}

func TestMemory_ExpansionsAndCostsAreIncremental(t *testing.T) {
	for a := uint64(0); a < 128; a += 32 {
		for b := uint64(0); b < 128; b += 32 {

			m := NewMemory()

			// Compute costs from 0 to target size.
			costA, sizeA, errA := m.getExpansionCostsAndSize(a)
			costB, sizeB, errB := m.getExpansionCostsAndSize(b)
			if errA != nil || errB != nil {
				t.Fatalf("unexpected error: %v, %v", errA, errB)
			}

			// Compute costs for increasing from size a to size b.
			ctxt := &context{gas: math.MaxInt}
			if err := m.expandMemory(0, a, ctxt); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			delta, sizeAB, err := m.getExpansionCostsAndSize(b)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			max := sizeA
			if sizeB > sizeA {
				max = sizeB
			}
			if sizeAB != max {
				t.Fatalf("size must increase monotonically, got: %d, %d, %d", sizeA, sizeB, sizeAB)
			}

			wantDelta := tosca.Gas(0)
			if a <= b {
				wantDelta = costB - costA
			}

			if wantDelta != delta {
				t.Fatalf("unexpected delta for expansions to %d and %d: want: %d, got: %d", a, b, wantDelta, delta)
			}
		}
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
	_, err := m.getSlice(uint256.NewInt(0), uint256.NewInt(1), &c)
	if !errors.Is(err, errOutOfGas) {
		t.Errorf("error should be errOutOfGas, instead is: %v", err)
	}

	_, err = m.getSlice(uint256.NewInt(math.MaxUint64-31), uint256.NewInt(32), &c)
	if !errors.Is(err, errOverflow) {
		t.Errorf("error should be errOverflow, instead is: %v", err)
	}
}

func TestMemory_getSlice_ReturnsSliceOfRequestedSize(t *testing.T) {

	tests := map[string]struct {
		offset   *uint256.Int
		size     *uint256.Int
		expected []byte
	}{
		"size zero returns empty slice": {
			offset:   uint256.NewInt(64),
			size:     uint256.NewInt(0),
			expected: []byte{},
		},
		"memory does not expand when not needed": {
			offset:   uint256.NewInt(0),
			size:     uint256.NewInt(4),
			expected: []byte{0x0, 0x01, 0x02, 0x03},
		},
		"memory expands when needed": {
			offset:   uint256.NewInt(4),
			size:     uint256.NewInt(5),
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
		for offset := uint64(0); offset < 128; offset++ {
			for size := uint64(1); size < 32; size++ {
				offset256 := uint256.NewInt(offset)
				size256 := uint256.NewInt(size)
				c := &context{gas: 15}
				m := NewMemory()
				m.store = make([]byte, memSize)
				_, err := m.getSlice(offset256, size256, c)
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}

				want := (offset + size + 31) / 32 * 32
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

func TestMemory_getSlice_DoesNotExpandWithSizeZero(t *testing.T) {
	for memSize := 0; memSize < 128; memSize += 32 {
		for offset := uint64(0); offset < 128; offset++ {
			c := &context{gas: 1}
			m := NewMemory()
			m.store = make([]byte, memSize)
			_, err := m.getSlice(uint256.NewInt(offset), uint256.NewInt(0), c)
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
	_, err := m.getSlice(uint256.NewInt(4), uint256.NewInt(29), &c)
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
	_, err := m.getSlice(uint256.NewInt(28), uint256.NewInt(8), &c)
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
	err := m.readWord(uint256.NewInt(math.MaxUint64-31), target, &c)
	if !errors.Is(err, errOverflow) {
		t.Errorf("error should be errOverflow, instead is: %v", err)
	}
	if target.Cmp(originalTarget) != 0 {
		t.Errorf("target should not have been modified, want %v but got %v", originalTarget, target)
	}
	err = m.readWord(uint256.NewInt(0), target, &c)
	if !errors.Is(err, errOutOfGas) {
		t.Errorf("error should be errOutOfGas, instead is: %v", err)
	}
	if target.Cmp(originalTarget) != 0 {
		t.Errorf("target should not have been modified, want %v but got %v", originalTarget, target)
	}
}

func TestMemory_set_UpdatesDataInMemoryAtGivenOffset(t *testing.T) {
	before := generateRandomBytes(128)
	for offset := uint64(0); offset < 128; offset++ {
		for size := 0; size < int(128-offset); size++ {
			data := generateRandomBytes(size)

			// test the memory update
			m := &Memory{store: bytes.Clone(before)}
			if err := m.set(uint256.NewInt(offset), data, nil); err != nil {
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

func TestMemory_set_ExpansionPreservesMemoryContentAndPadsWithZeroesToTheRight(t *testing.T) {
	before := generateRandomBytes(32)
	m := &Memory{store: bytes.Clone(before)}
	if err := m.set(uint256.NewInt(64), []byte{0x1, 0x2}, &context{gas: 100}); err != nil {
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
			if err := m.set(uint256.NewInt(offset), data, nil); err != nil {
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
	if err := m.set(uint256.NewInt(64), []byte{0x1}, c); !errors.Is(err, errOutOfGas) {
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
	if err := m.set(uint256.NewInt(offset), data, nil); !errors.Is(err, errOverflow) {
		t.Errorf("unexpected error %v, got %v", errOutOfGas, err)
	}
	if len(m.store) != 32 {
		t.Errorf("memory should not have been expanded, instead is: %d", len(m.store))
	}
}

////////////////////////////////////////////////////////////////////////////////
// Helper functions

func generateRandomBytes(size int) []byte {
	data := make([]byte, size)
	_, _ = rand.Read(data)
	return data
}
