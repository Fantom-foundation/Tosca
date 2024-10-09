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

const (
	// Maximum memory size allowed
	// This magic number comes from 'core/vm/gas_table.go' 'memoryGasCost' in geth
	maxMemoryExpansionSize = 0x1FFFFFFFE0
)

// getExpansionCostsAndSize returns the gas cost and the new memory size after
// the expansion.
// The function returns an error if the new size is greater than the maximum
// memory size allowed, or an overflow happens when computing the costs.
func (m *Memory) getExpansionCostsAndSize(size uint64) (tosca.Gas, uint64, error) {

	// static assert
	const (
		// Memory expansion cost is done using unsigned arithmetic,
		// check for the maximum memory expansion size, not overflowing int64 after computing costs
		maxInWords uint64 = (uint64(maxMemoryExpansionSize) + 31) / 32
		_                 = int64(maxInWords*maxInWords/512 + 3*maxInWords)
	)

	if m.length() >= size {
		return 0, m.length(), nil
	}

	words := tosca.SizeInWords(size)
	validSize := words * 32
	if validSize < size {
		return 0, 0, errOverflow
	}
	if validSize > maxMemoryExpansionSize {
		return 0, 0, errMaxMemoryExpansionSize
	}

	new_costs := tosca.Gas((words*words)/512 + (3 * words))
	fee := new_costs - m.currentMemoryCost
	return fee, validSize, nil
}

// expandMemory tries to expand memory to the given size.
// If the memory is already large enough or size is 0, it does nothing.
// If there is not enough gas in the context or an overflow occurs when adding
// offset and size, it returns an error.
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
		fee, expandedSize, err := m.getExpansionCostsAndSize(needed)
		if err != nil {
			return err
		}
		if err := c.useGas(fee); err != nil {
			return err
		}

		currentSize := m.length()
		m.currentMemoryCost += fee
		m.store = append(m.store, make([]byte, expandedSize-currentSize)...)
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
func (m *Memory) getSlice(offset, size *uint256.Int, c *context) ([]byte, error) {
	if size.IsZero() {
		return nil, nil
	}
	if !offset.IsUint64() || !size.IsUint64() {
		return nil, errOverflow
	}

	offset64 := offset.Uint64()
	size64 := size.Uint64()

	err := m.expandMemory(offset64, size64, c)
	if err != nil {
		return nil, err
	}
	return m.store[offset64 : offset64+size64], nil
}

// readWord reads a Word (32 byte) from the memory at the given offset and stores
// that word in the provided target.
// Expands memory as needed and charges for it.
// Returns an error in case of not enough gas or offset+32 overflow.
func (m *Memory) readWord(offset *uint256.Int, target *uint256.Int, c *context) error {
	data, err := m.getSlice(offset, uint256.NewInt(32), c)
	if err != nil {
		return err
	}
	target.SetBytes32(data)
	return nil
}

// set copies the given value into memory at the given offset.
// Expands the memory size as needed and charges for it.
// Returns an error if there is not enough gas or offset+len(value) overflows.
func (m *Memory) set(offset *uint256.Int, value []byte, c *context) error {
	data, err := m.getSlice(offset, uint256.NewInt(uint64(len(value))), c)
	if err != nil {
		return err
	}
	copy(data, value)
	return nil
}
