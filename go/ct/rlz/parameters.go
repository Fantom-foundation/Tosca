// Copyright (c) 2024 Fantom Foundation
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at fantom.foundation/bsl11.
//
// Change Date: 2028-4-16
//
// On the date above, in accordance with the Business Source License, use of
// this software will be governed by the GNU Lesser General Public License v3.

package rlz

import (
	"math"

	. "github.com/Fantom-foundation/Tosca/go/ct/common"
)

type Parameter interface {
	// Samples returns a list of test values for an operation parameter.
	// For efficiency reasons, the resulting slice may be shared among
	// multiple calls. Thus, neither the slice itself nor its elements must
	// be modified by users of this function.
	Samples() []U256
}

type NumericParameter struct{}

var numericParameterSamples = []U256{
	NewU256(0),
	NewU256(1),
	NewU256(1 << 8),
	NewU256(1 << 16),
	NewU256(1 << 32),
	NewU256(1 << 48),
	NewU256(1).Shl(NewU256(64)),
	NewU256(1).Shl(NewU256(128)),
	NewU256(1).Shl(NewU256(192)),
	NewU256(1).Shl(NewU256(255)),
	NewU256(0).Not(),
	NewU256(1, 1),
}

func (NumericParameter) Samples() []U256 {
	return numericParameterSamples
}

type JumpTargetParameter struct{}

var jumpTargetParameterSamples = []U256{
	NewU256(0),
	NewU256(1),
	NewU256(1 << 8),
	NewU256(math.MaxInt32 + 1),
	NewU256(1, 1),
}

func (JumpTargetParameter) Samples() []U256 {
	return jumpTargetParameterSamples
}

type StorageAccessKeyParameter = NumericParameter

type MemoryOffsetParameter struct{}

var memoryOffsetParameterSamples = []U256{
	NewU256(0),
	NewU256(1),
	NewU256(31),
	NewU256(32),
	NewU256(1, 0),
}

func (MemoryOffsetParameter) Samples() []U256 {
	return memoryOffsetParameterSamples
}

type MemoryOffsetForCopyParameter struct{}

var memoryOffsetForCopyParameter = []U256{
	NewU256(0),
	NewU256(1),
	NewU256(32),
	NewU256(2 * 24576),
	NewU256(0x1FFFFFFFE1), // st.maxMemoryExpansionSize
	NewU256(1, 0),
}

func (MemoryOffsetForCopyParameter) Samples() []U256 {
	return memoryOffsetForCopyParameter
}

type MemorySizeParameter struct{}

var memorySizeParameterSamples = []U256{
	NewU256(0),
	NewU256(1),
	NewU256(31),
	NewU256(32),
	NewU256(1, 0),
	// Samples stressing the max init code size introduced with Shanghai
	NewU256(2*24576 - 1),
	NewU256(2 * 24576),
	NewU256(2*24576 + 1),
	NewU256(0x1FFFFFFFE1), // st.maxMemoryExpansionSize
}

func (MemorySizeParameter) Samples() []U256 {
	return memorySizeParameterSamples
}

type TopicParameter struct{}

var topicParameterSamples = []U256{
	// Two samples to ensure topic order is correct. Adding more samples
	// here will create significant more test cases for LOG instructions.
	NewU256(101),
	NewU256(102),
}

func (TopicParameter) Samples() []U256 {
	return topicParameterSamples
}

type AddressParameter struct{}

var addressParameterSamples = []U256{
	// Adding more samples here will create significantly more test cases for EXTCODECOPY.
	// TODO: evaluate code coverage
	NewU256(0),
	//NewU256(1),
	//NewU256(1).Shl(NewU256(20*8 - 1)), // < first bit of 20-byte address set
	//NewU256(3).Shl(NewU256(20*8 - 1)), // < first bit beyond 20-byte address set as well (should be the same address as above)
	NewU256(0).Not(),
}

func (AddressParameter) Samples() []U256 {
	return addressParameterSamples
}

type DataOffsetParameter = MemoryOffsetParameter
type DataSizeParameter = MemorySizeParameter

type GasParameter struct{}

var gasParameterSamples = []U256{
	NewU256(0),
	NewU256(1),
	NewU256(1 << 10),
	// NewU256(1 << 20),
	// NewU256(1 << 62),
	// MaxU256(),
}

func (GasParameter) Samples() []U256 {
	return gasParameterSamples
}

type ValueParameter struct{}

var valueParameterSamples = []U256{
	NewU256(0),
	NewU256(1),
	MaxU256(),
}

func (ValueParameter) Samples() []U256 {
	return valueParameterSamples
}
