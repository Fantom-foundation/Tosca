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
