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

func TestMemory_set_SuccessfulCase(t *testing.T) {
	const (
		memoryOriginalSize = uint64(8)
		offset             = uint64(1)
		sizeToExpand       = 32 - memoryOriginalSize - offset
	)
	// data of size (memoryOriginalSize - offset), to ensure it would fit in memory
	data := make([]byte, memoryOriginalSize-offset)
	for i := range data {
		// add non zero values.
		data[i] = byte(i + 1)
	}
	size := uint64(len(data))
	c := context{gas: 3}
	m := NewMemory()
	m.store = []byte{0xff}

	err := m.set(offset, size, data, &c)
	if err != nil {
		t.Fatalf("unexpected error, want: %v, got: %v", nil, err)
	}
	got, err := toValidMemorySize(memoryOriginalSize)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if m.length() != got {
		t.Errorf("set should not change memory size, want: %d, got: %d", got, m.length())
	}
	want := append([]byte{0xff}, data...)
	want = append(want, []byte{sizeToExpand: 0x0}...)
	if !bytes.Equal(m.store, want) {
		t.Errorf("unexpected memory value, want: %x, got: %x", want, m.store)
	}
}

func TestMemory_set_ErrorCases(t *testing.T) {
	c := context{gas: 0}
	m := NewMemory()

	err := m.set(0, 1, []byte{}, &c)
	if !errors.Is(err, errOutOfGas) {
		t.Errorf("unexpected error, want: %v, got: %v", errOutOfGas, err)
	}
	err = m.set(math.MaxUint64, 1, []byte{}, &c)
	if !errors.Is(err, errOverflow) {
		t.Errorf("unexpected error, want: %v, got: %v", errOverflow, err)
	}
	err = m.set(1, math.MaxUint64, []byte{}, &c)
	if !errors.Is(err, errOverflow) {
		t.Errorf("unexpected error, want: %v, got: %v", errOverflow, err)
	}
}

func TestMemory_set_sizeZeroDoesNotProduceExpansion(t *testing.T) {
	for offset := uint64(0); offset < 128; offset++ {
		c := context{gas: 0}
		m := NewMemory()
		m.store = []byte{0x1}
		err := m.set(offset, 0, []byte{}, &c)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if m.length() != 1 {
			t.Errorf("memory should not have been expanded, instead is: %d", m.length())
		}
	}
}

func TestMemory_setByte_SuccessfulCase(t *testing.T) {
	memory := []byte{0x12, 0x34, 0x56, 0x78}
	offset := uint64(8)
	ctxt := context{gas: 3}
	m := NewMemory()
	m.store = bytes.Clone(memory)
	value := byte(0xff)

	err := m.setByte(offset, value, &ctxt)
	if err != nil {
		t.Errorf("unexpected error, want: %v, got: %v", nil, err)
	}
	if m.length() <= offset {
		t.Errorf("unexpected memory size, want: %d, got: %d", offset, m.length())
	}
	for i, b := range memory {
		if i == int(offset) {
			if m.store[i] != value {
				t.Errorf("unexpected value, want: %v, got: %v", value, m.store[i])
			}
		} else {
			if m.store[i] != b {
				t.Errorf("unexpected value at position %v, want: %v, got: %v", i, b, m.store[i])
			}
		}
	}

}

func TestMemory_SetByte_ErrorCases(t *testing.T) {
	ctxt := context{gas: 0}
	m := NewMemory()
	err := m.setByte(math.MaxUint64, 0x12, &ctxt)
	if !errors.Is(err, errOverflow) {
		t.Errorf("unexpected error, want: %v, got: %v", errOverflow, err)
	}
	err = m.setByte(0, 0x12, &ctxt)
	if !errors.Is(err, errOutOfGas) {
		t.Errorf("unexpected error, want: %v, got: %v", errOutOfGas, err)
	}
}

func TestMemory_setWord_successfulCase(t *testing.T) {
	c := context{gas: 6 + 1}
	m := NewMemory()
	offset := uint64(1)
	value := new(uint256.Int).SetBytes([]byte{0xff, 0xee})
	err := m.setWord(offset, value, &c)
	if err != nil {
		t.Fatalf("unexpected error, want: %v, got: %v", nil, err)
	}
	if m.length() != 64 {
		t.Errorf("memory size should be 64 bytes instead got: %d", m.length())
	}
	val32 := value.Bytes32()
	if !bytes.Equal(m.store[offset:offset+32], val32[:]) {
		t.Errorf("unexpected memory value, want: %x, got: %x", value.Bytes32(), m.store[offset:offset+32])
	}
	if c.gas != 1 {
		t.Errorf("unexpected gas value, want: %d, got: %d", 1, c.gas)
	}
}

func TestMemory_SetWord_ErrorCases(t *testing.T) {
	ctxt := context{gas: 0}
	// SetWord (and EnsureCapacity) check for offset overflow before checking gas
	// hence it is fine to check both cases with gas = 0
	m := NewMemory()
	if err := m.setWord(math.MaxUint64-31, new(uint256.Int), &ctxt); !errors.Is(err, errOverflow) {
		t.Fatalf("expected error, wanted overflow error but got %v", err)
	}
	if err := m.setWord(0, new(uint256.Int), &ctxt); !errors.Is(err, errOutOfGas) {
		t.Fatalf("expected error, wanted out of gas error but got %v", err)
	}
}
