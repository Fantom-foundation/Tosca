package rlz

import (
	. "github.com/Fantom-foundation/Tosca/go/ct/common"
)

type Parameter interface {
	Samples() []U256
}

type NumericParameter struct{}

func (NumericParameter) Samples() []U256 {
	return []U256{
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
	}
}

type MemoryOffsetParameter struct{}

func (MemoryOffsetParameter) Samples() []U256 {
	return []U256{
		NewU256(0),
		NewU256(1),
		NewU256(31),
		NewU256(32),
		NewU256(1, 0),
	}
}

type MemorySizeParameter struct{}

func (MemorySizeParameter) Samples() []U256 {
	return []U256{
		NewU256(0),
		NewU256(1),
		NewU256(31),
		NewU256(32),
		NewU256(1, 0),
	}
}

type TopicParameter struct{}

func (TopicParameter) Samples() []U256 {
	return []U256{
		// Two samples to ensure topic order is correct. Adding more samples
		// here will create significant more test cases for LOG instructions.
		NewU256(101),
		NewU256(102),
	}
}

type AddressParameter struct{}

func (AddressParameter) Samples() []U256 {
	return []U256{
		// Adding more samples here will create significant more test cases for EXTCODECOPY.
		// TODO: evaluate code coverage
		NewU256(0),
		//NewU256(1),
		//NewU256(1).Shl(NewU256(20*8 - 1)), // < first bit of 20-byte address set
		//NewU256(3).Shl(NewU256(20*8 - 1)), // < first bit beyond 20-byte address set as well (should be the same address as above)
		NewU256(0).Not(),
	}
}

type DataOffsetParameter = MemoryOffsetParameter
type DataSizeParameter = MemorySizeParameter
