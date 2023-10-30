package rlz

import (
	"github.com/Fantom-foundation/Tosca/go/ct"
)

type Parameter interface {
	Samples(example ct.U256) []ct.U256
}

type NumericParameter struct{}

func (n NumericParameter) Samples(ct.U256) []ct.U256 {
	return n.SampleValues()
}

func (NumericParameter) SampleValues() []ct.U256 {
	return []ct.U256{
		ct.NewU256(0),
		ct.NewU256(1),
		ct.NewU256(1 << 8),
		ct.NewU256(1 << 16),
		ct.NewU256(1 << 32),
		ct.NewU256(1 << 48),
		ct.NewU256(1).Shl(ct.NewU256(64)),
		ct.NewU256(1).Shl(ct.NewU256(128)),
		ct.NewU256(1).Shl(ct.NewU256(192)),
		ct.NewU256(1).Shl(ct.NewU256(255)),
		ct.NewU256(0).Not(),
	}
}
