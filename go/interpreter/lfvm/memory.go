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
// size, it returns an error.
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
// Returns an error in case of not enough gas or offset+32 overflow.
func (m *Memory) readWord(offset uint64, target *uint256.Int, c *context) error {
	data, err := m.getSlice(offset, 32, c)
	if err != nil {
		return err
	}
	target.SetBytes32(data)
	return nil
}

// set sets the given value at the given offset, expands memory as needed and charges for it.
// Returns error if insufficient gas or offset+size overflows.
func (m *Memory) set(offset, size uint64, value []byte, c *context) error {
	data, err := m.getSlice(offset, size, c)
	if err != nil {
		return err
	}
	copy(data, value)
	return nil
}

// setByte sets a byte at the given offset, expanding memory as needed and charging for it.
// Returns error if insufficient gas or offset+1 overflows.
func (m *Memory) setByte(offset uint64, value byte, c *context) error {
	return m.set(offset, 1, []byte{value}, c)
}

// setWord sets a 32-byte word at the given offset, expanding memory as needed and charging for it.
func (m *Memory) setWord(offset uint64, value *uint256.Int, c *context) error {
	data := value.Bytes32()
	return m.set(offset, 32, data[:], c)
}
