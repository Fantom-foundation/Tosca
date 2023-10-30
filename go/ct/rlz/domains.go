package rlz

import (
	"math"

	"github.com/Fantom-foundation/Tosca/go/ct"
	"github.com/Fantom-foundation/Tosca/go/ct/st"
)

// Domain represents the domain of values for a given type.
type Domain[T any] interface {
	Equal(T, T) bool
	Less(T, T) bool
	Predecessor(T) T
	Successor(T) T
	SomethingNotEqual(T) T
	Samples(T) []T
	SamplesForAll([]T) []T
}

////////////////////////////////////////////////////////////
// Bool

type boolDomain struct{}

func (boolDomain) Equal(a bool, b bool) bool { return a == b }
func (boolDomain) Less(a bool, b bool) bool  { panic("not useful") }
func (boolDomain) Predecessor(a bool) bool   { panic("not useful") }
func (boolDomain) Successor(a bool) bool     { panic("not useful") }

func (boolDomain) SomethingNotEqual(a bool) bool {
	return !a
}

func (boolDomain) Samples(bool) []bool {
	return []bool{false, true}
}

func (boolDomain) SamplesForAll(_ []bool) []bool {
	return []bool{false, true}
}

////////////////////////////////////////////////////////////
// uint16

type uint16Domain struct{}

func (uint16Domain) Equal(a uint16, b uint16) bool     { return a == b }
func (uint16Domain) Less(a uint16, b uint16) bool      { return a < b }
func (uint16Domain) Predecessor(a uint16) uint16       { return a - 1 }
func (uint16Domain) Successor(a uint16) uint16         { return a + 1 }
func (uint16Domain) SomethingNotEqual(a uint16) uint16 { return a + 1 }

func (d uint16Domain) Samples(a uint16) []uint16 {
	return d.SamplesForAll([]uint16{a})
}

func (uint16Domain) SamplesForAll(as []uint16) []uint16 {
	res := []uint16{0, ^uint16(0)}

	// Test every element off by one.
	for _, a := range as {
		res = append(res, a-1)
		res = append(res, a)
		res = append(res, a+1)
	}

	// Add all powers of 2.
	for i := 0; i < 16; i++ {
		res = append(res, uint16(1<<i))
	}

	// TODO: consider removing duplicates.

	return res
}

////////////////////////////////////////////////////////////
// uint64

type uint64Domain struct{}

func (uint64Domain) Equal(a uint64, b uint64) bool     { return a == b }
func (uint64Domain) Less(a uint64, b uint64) bool      { return a < b }
func (uint64Domain) Predecessor(a uint64) uint64       { return a - 1 }
func (uint64Domain) Successor(a uint64) uint64         { return a + 1 }
func (uint64Domain) SomethingNotEqual(a uint64) uint64 { return a + 1 }

func (d uint64Domain) Samples(a uint64) []uint64 {
	return d.SamplesForAll([]uint64{a})
}

func (uint64Domain) SamplesForAll(as []uint64) []uint64 {
	res := []uint64{0, ^uint64(0)}

	// Test every element off by one.
	for _, a := range as {
		res = append(res, a-1)
		res = append(res, a)
		res = append(res, a+1)
	}

	// Add all powers of 2.
	for i := 0; i < 64; i++ {
		res = append(res, uint64(1<<i))
	}

	// TODO: consider removing duplicates.

	return res
}

////////////////////////////////////////////////////////////
// U256

type u256Domain struct{}

func (u256Domain) Equal(a ct.U256, b ct.U256) bool { return a.Eq(b) }
func (u256Domain) Less(a ct.U256, b ct.U256) bool  { return a.Lt(b) }
func (u256Domain) Predecessor(a ct.U256) ct.U256   { return a.Sub(ct.NewU256(1)) }
func (u256Domain) Successor(a ct.U256) ct.U256     { return a.Add(ct.NewU256(1)) }

func (u256Domain) SomethingNotEqual(a ct.U256) ct.U256 {
	return a.Add(ct.NewU256(1))
}

func (d u256Domain) Samples(a ct.U256) []ct.U256 {
	return d.SamplesForAll([]ct.U256{a})
}

func (d u256Domain) SamplesForAll(as []ct.U256) []ct.U256 {
	res := []ct.U256{}

	// Test every element off by one.
	for _, a := range as {
		res = append(res, d.Predecessor(a))
		res = append(res, a)
		res = append(res, d.Successor(a))
	}

	// Add more interesting values.
	res = append(res, NumericParameter{}.SampleValues()...)

	// TODO: consider removing duplicates.

	return res
}

////////////////////////////////////////////////////////////
// st.Status

type statusCodeDomain struct{}

func (statusCodeDomain) Equal(a st.StatusCode, b st.StatusCode) bool { return a == b }
func (statusCodeDomain) Less(a st.StatusCode, b st.StatusCode) bool  { panic("not useful") }
func (statusCodeDomain) Predecessor(a st.StatusCode) st.StatusCode   { panic("not useful") }
func (statusCodeDomain) Successor(a st.StatusCode) st.StatusCode     { panic("not useful") }

func (statusCodeDomain) SomethingNotEqual(a st.StatusCode) st.StatusCode {
	if a == st.Running {
		return st.Stopped
	}
	return st.Running
}

func (statusCodeDomain) Samples(a st.StatusCode) []st.StatusCode {
	return []st.StatusCode{st.Running, st.Stopped, st.Returned, st.Reverted, st.Failed}
}

func (statusCodeDomain) SamplesForAll(a []st.StatusCode) []st.StatusCode {
	return []st.StatusCode{st.Running, st.Stopped, st.Returned, st.Reverted, st.Failed}
}

////////////////////////////////////////////////////////////
// Program Counter

type pcDomain struct{}

func (pcDomain) Equal(a ct.U256, b ct.U256) bool     { return a.Eq(b) }
func (pcDomain) Less(a ct.U256, b ct.U256) bool      { return a.Lt(b) }
func (pcDomain) Predecessor(a ct.U256) ct.U256       { return a.Sub(ct.NewU256(1)) }
func (pcDomain) Successor(a ct.U256) ct.U256         { return a.Add(ct.NewU256(1)) }
func (pcDomain) SomethingNotEqual(a ct.U256) ct.U256 { return a.Add(ct.NewU256(1)) }

func (d pcDomain) Samples(a ct.U256) []ct.U256 {
	return d.SamplesForAll([]ct.U256{a})
}

func (pcDomain) SamplesForAll(as []ct.U256) []ct.U256 {
	pcs := []uint16{}
	for _, a := range as {
		if a.IsUint64() && a.Uint64() <= uint64(math.MaxUint16) {
			pcs = append(pcs, uint16(a.Uint64()))
		}
	}

	pcs = uint16Domain{}.SamplesForAll(pcs)

	res := make([]ct.U256, 0, len(pcs))
	for _, cur := range pcs {
		res = append(res, ct.NewU256(uint64(cur)))
	}
	return res
}
