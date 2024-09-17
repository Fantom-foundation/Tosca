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
	"fmt"
	"math"

	"github.com/Fantom-foundation/Tosca/go/tosca"
	"github.com/holiman/uint256"
)

type Memory struct {
	store             []byte
	currentMemoryCost tosca.Gas
}

func NewMemory() *Memory {
	return &Memory{}
}

func toValidMemorySize(size uint64) uint64 {
	fullWordsSize := tosca.SizeInWords(size) * 32
	if size != 0 && fullWordsSize < size {
		// TODO: this is a compromised solution, reconsider this with issue #524
		// Geth handles this by adding an overflow boolean to every mem operation: core/vm/common.go (calcMemSize64WithUint)
		return math.MaxUint64
	}
	return fullWordsSize
}

const (
	// Maximum memory size allowed
	// This magic number comes from 'core/vm/gas_table.go' 'memoryGasCost' in geth
	maxMemoryExpansionSize = 0x1FFFFFFFE0
)

func (m *Memory) getExpansionCosts(size uint64) tosca.Gas {

	// static assert
	const (
		// Memory expansion cost is done using unsigned arithmetic,
		// check for the maximum memory expansion size, not overflowing int64 after computing costs
		maxInWords uint64 = (uint64(maxMemoryExpansionSize) + 31) / 32
		_                 = int64(maxInWords*maxInWords/512 + 3*maxInWords)
	)

	if m.length() >= size {
		return 0
	}
	size = toValidMemorySize(size)

	if size > maxMemoryExpansionSize {
		return tosca.Gas(math.MaxInt64)
	}

	words := tosca.SizeInWords(size)
	new_costs := tosca.Gas((words*words)/512 + (3 * words))
	fee := new_costs - m.currentMemoryCost
	return fee
}

// expandMemory tries to expand memory to the given size.
// If the memory is already large enough or size is 0, it does nothing.
// If there is not enough gas in the context or an overflow occurs when adding offset and
// size, it returns an error.
func (m *Memory) expandMemory(offset, size uint64, c *context) error {
	if size == 0 {
		return nil
	}
	needed := offset + size
	// check overflow
	if needed < offset {
		c.signalError()
		return errGasUintOverflow
	}
	if m.length() < needed {
		fee := m.getExpansionCosts(needed)
		if !c.useGas(fee) {
			c.status = statusOutOfGas
			return errOutOfGas
		}
		m.expandMemoryWithoutCharging(needed)
	}

	return nil
}

// expandMemoryWithoutCharging expands the memory to the given size without charging gas.
func (m *Memory) expandMemoryWithoutCharging(needed uint64) {
	needed = toValidMemorySize(needed)
	size := m.length()
	if size < needed {
		m.currentMemoryCost += m.getExpansionCosts(needed)
		m.store = append(m.store, make([]byte, needed-size)...)
	}
}

func (m *Memory) length() uint64 {
	return uint64(len(m.store))
}

// setByte sets a byte at the given offset, expanding memory as needed and charging for it.
// Returns error if insufficient gas or offset+1 overflows.
func (m *Memory) setByte(offset uint64, value byte, c *context) error {
	return m.set(offset, 1, []byte{value}, c)
}

func (m *Memory) SetWord(offset uint64, value *uint256.Int, c *context) error {
	err := m.expandMemory(offset, 32, c)
	if err != nil {
		return err
	}

	if m.length() < offset+32 {
		return fmt.Errorf("memory too small, size %d, attempted to write 32 byte at position %d", m.length(), offset)
	}

	// Inlining and unrolling value.WriteToSlice(..) lead to a 7x speedup
	dest := m.store[offset : offset+32]
	dest[31] = byte(value[0])
	dest[30] = byte(value[0] >> 8)
	dest[29] = byte(value[0] >> 16)
	dest[28] = byte(value[0] >> 24)
	dest[27] = byte(value[0] >> 32)
	dest[26] = byte(value[0] >> 40)
	dest[25] = byte(value[0] >> 48)
	dest[24] = byte(value[0] >> 56)

	dest[23] = byte(value[1])
	dest[22] = byte(value[1] >> 8)
	dest[21] = byte(value[1] >> 16)
	dest[20] = byte(value[1] >> 24)
	dest[19] = byte(value[1] >> 32)
	dest[18] = byte(value[1] >> 40)
	dest[17] = byte(value[1] >> 48)
	dest[16] = byte(value[1] >> 56)

	dest[15] = byte(value[2])
	dest[14] = byte(value[2] >> 8)
	dest[13] = byte(value[2] >> 16)
	dest[12] = byte(value[2] >> 24)
	dest[11] = byte(value[2] >> 32)
	dest[10] = byte(value[2] >> 40)
	dest[9] = byte(value[2] >> 48)
	dest[8] = byte(value[2] >> 56)

	dest[7] = byte(value[3])
	dest[6] = byte(value[3] >> 8)
	dest[5] = byte(value[3] >> 16)
	dest[4] = byte(value[3] >> 24)
	dest[3] = byte(value[3] >> 32)
	dest[2] = byte(value[3] >> 40)
	dest[1] = byte(value[3] >> 48)
	dest[0] = byte(value[3] >> 56)
	return nil
}

// trySet sets the given value at the given offset.
// Returns error if insufficient memory or offset+size overflows.
func (m *Memory) trySet(offset, size uint64, value []byte) error {
	if size > 0 {
		if offset+size < offset {
			return errGasUintOverflow
		}
		if offset+size > m.length() {
			return makeInsufficientMemoryError(m.length(), size, offset)
		}
		copy(m.store[offset:offset+size], value)
	}
	return nil
}

func makeInsufficientMemoryError(memSize, size, offset uint64) error {
	return tosca.ConstError(fmt.Sprintf("memory too small, size %d, attempted to write %d bytes at %d", memSize, size, offset))
}

// set sets the given value at the given offset, expands memory as needed and charges for it.
// Returns error if insufficient gas or offset+size overflows.
func (m *Memory) set(offset, size uint64, value []byte, c *context) error {
	err := m.expandMemory(offset, size, c)
	if err != nil {
		return err
	}
	return m.trySet(offset, size, value)
}

func (m *Memory) CopyWord(offset uint64, trg *uint256.Int, c *context) error {
	err := m.expandMemory(offset, 32, c)
	if err != nil {
		return err
	}
	if m.length() < offset+32 {
		return fmt.Errorf("memory too small, size %d, attempted to read 32 byte at position %d", m.length(), offset)
	}
	trg.SetBytes32(m.store[offset : offset+32])
	return nil
}

// Copies data from the memory to the given slice.
func (m *Memory) CopyData(offset uint64, trg []byte) {
	if m.length() < offset {
		copy(trg, make([]byte, len(trg)))
		return
	}

	// Copy what is available.
	covered := copy(trg, m.store[offset:])

	// Pad the rest
	if covered < len(trg) {
		copy(trg[covered:], make([]byte, len(trg)-covered))
	}
}

func (m *Memory) GetSlice(offset, size uint64) []byte {
	if size == 0 {
		return nil
	}

	if m.length() >= offset+size {
		return m.store[offset : offset+size]
	}

	return nil
}

func (m *Memory) GetSliceWithCapacityAndGas(offset, size uint64, c *context) ([]byte, error) {
	err := m.expandMemory(offset, size, c)
	if err != nil {
		return nil, err
	}
	return m.GetSlice(offset, size), nil
}
