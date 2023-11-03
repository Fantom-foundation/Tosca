package rlz

import (
	. "github.com/Fantom-foundation/Tosca/go/ct/common"
)

type Parameter interface {
	Samples(example U256) []U256
}

type NumericParameter struct{}

func (n NumericParameter) Samples(U256) []U256 {
	return n.SampleValues()
}

func (NumericParameter) SampleValues() []U256 {
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
