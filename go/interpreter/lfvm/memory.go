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

func toValidMemorySize(size uint64) (uint64, error) {
	fullWordsSize := tosca.SizeInWords(size) * 32
	if size != 0 && fullWordsSize < size {
		return 0, errOverflow
	}
	return fullWordsSize, nil
}

const (
	// Maximum memory size allowed
	// This magic number comes from 'core/vm/gas_table.go' 'memoryGasCost' in geth
	maxMemoryExpansionSize = 0x1FFFFFFFE0
)

// getExpansionCosts returns the gas cost of expanding the memory to the given size.
func (m *Memory) getExpansionCosts(size uint64) (tosca.Gas, error) {

	// static assert
	const (
		// Memory expansion cost is done using unsigned arithmetic,
		// check for the maximum memory expansion size, not overflowing int64 after computing costs
		maxInWords uint64 = (uint64(maxMemoryExpansionSize) + 31) / 32
		_                 = int64(maxInWords*maxInWords/512 + 3*maxInWords)
	)

	if m.length() >= size {
		return 0, nil
	}

	size, err := toValidMemorySize(size)
	if err != nil {
		return 0, err
	}

	if size > maxMemoryExpansionSize {
		return 0, errMaxMemoryExpansionSize
	}

	words := tosca.SizeInWords(size)
	new_costs := tosca.Gas((words*words)/512 + (3 * words))
	fee := new_costs - m.currentMemoryCost
	return fee, nil
}

// expandMemory tries to expand memory to the given size.
// If the memory is already large enough or size is 0, it does nothing.
// If there is not enough gas in the context or an overflow occurs when adding offset and
// size, it returns an error. Caller should check the error and handle it.
func (m *Memory) expandMemory(offset, size uint64, c *context) error {
	if size == 0 {
		return nil
	}
	needed := offset + size
	// check overflow
	if needed < offset {
		return errOverflow
	}
	if m.length() < needed {
		fee, err := m.getExpansionCosts(needed)
		if err != nil {
			return err
		}
		if err := c.useGas(fee); err != nil {
			return err
		}
		if err := m.expandMemoryWithoutCharging(needed); err != nil {
			return err
		}
	}

	return nil
}

// expandMemoryWithoutCharging expands the memory to the given size without charging gas.
func (m *Memory) expandMemoryWithoutCharging(needed uint64) error {
	needed, err := toValidMemorySize(needed)
	if err != nil {
		return err
	}
	size := m.length()
	if size < needed {
		fee, err := m.getExpansionCosts(needed)
		if err != nil {
			return err
		}
		m.currentMemoryCost += fee
		m.store = append(m.store, make([]byte, needed-size)...)
	}
	return nil
}

func (m *Memory) length() uint64 {
	return uint64(len(m.store))
}

func (m *Memory) SetByte(offset uint64, value byte, c *context) error {
	err := m.expandMemory(offset, 1, c)
	if err != nil {
		return err
	}

	if m.length() < offset+1 {
		return fmt.Errorf("memory too small, size %d, attempted to write at position %d", m.length(), offset)
	}
	m.store[offset] = value
	return nil
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

func (m *Memory) Set(offset, size uint64, value []byte) error {
	if size > 0 {
		if offset+size < offset {
			return errOverflow
		}
		if offset+size > m.length() {
			return fmt.Errorf("memory too small, size %d, attempted to write %d bytes at %d", m.length(), size, offset)
		}
		copy(m.store[offset:offset+size], value)
	}
	return nil
}

func (m *Memory) SetWithCapacityAndGasCheck(offset, size uint64, value []byte, c *context) error {
	err := m.expandMemory(offset, size, c)
	if err != nil {
		return err
	}
	err = m.Set(offset, size, value)
	if err != nil {
		return err
	}
	return nil
}

// getSlice obtains a slice of size bytes from the memory at the given offset.
// The returned slice is backed by the memory's internal data. Updates to the
// slice will thus effect the memory states. This connection is invalidated by any
// subsequent memory operation that may change the size of the memory.
func (m *Memory) getSlice(offset, size uint64, c *context) ([]byte, error) {
	err := m.expandMemory(offset, size, c)
	if err != nil {
		return nil, err
	}
	// since memory does not expand on size 0 independently of the offset,
	// we need to prevent out of bounds access
	if size == 0 {
		return nil, nil
	}
	return m.store[offset : offset+size], nil
}

// readWord reads a Word (32 byte) from the memory at the given offset and stores
// that word in the provided target.
// Expands memory as needed and charges for it.
// Returns error in case of not enough gas or offset+32 overflow.
func (m *Memory) readWord(offset uint64, target *uint256.Int, c *context) error {
	data, err := m.getSlice(offset, 32, c)
	if err != nil {
		return err
	}
	target.SetBytes32(data)
	return nil
}
