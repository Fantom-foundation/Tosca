package rlz

import (
	"math"

	. "github.com/Fantom-foundation/Tosca/go/ct/common"
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

func (u256Domain) Equal(a U256, b U256) bool { return a.Eq(b) }
func (u256Domain) Less(a U256, b U256) bool  { return a.Lt(b) }
func (u256Domain) Predecessor(a U256) U256   { return a.Sub(NewU256(1)) }
func (u256Domain) Successor(a U256) U256     { return a.Add(NewU256(1)) }

func (u256Domain) SomethingNotEqual(a U256) U256 {
	return a.Add(NewU256(1))
}

func (d u256Domain) Samples(a U256) []U256 {
	return d.SamplesForAll([]U256{a})
}

func (d u256Domain) SamplesForAll(as []U256) []U256 {
	res := []U256{}

	// Test every element off by one.
	for _, a := range as {
		res = append(res, d.Predecessor(a))
		res = append(res, a)
		res = append(res, d.Successor(a))
	}

	// Add more interesting values.
	res = append(res, NumericParameter{}.Samples()...)

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

func (pcDomain) Equal(a U256, b U256) bool     { return a.Eq(b) }
func (pcDomain) Less(a U256, b U256) bool      { return a.Lt(b) }
func (pcDomain) Predecessor(a U256) U256       { return a.Sub(NewU256(1)) }
func (pcDomain) Successor(a U256) U256         { return a.Add(NewU256(1)) }
func (pcDomain) SomethingNotEqual(a U256) U256 { return a.Add(NewU256(1)) }

func (d pcDomain) Samples(a U256) []U256 {
	return d.SamplesForAll([]U256{a})
}

func (pcDomain) SamplesForAll(as []U256) []U256 {
	pcs := []uint16{}
	for _, a := range as {
		if a.IsUint64() && a.Uint64() <= uint64(math.MaxUint16) {
			pcs = append(pcs, uint16(a.Uint64()))
		}
	}

	pcs = uint16Domain{}.SamplesForAll(pcs)

	res := make([]U256, 0, len(pcs))
	for _, cur := range pcs {
		res = append(res, NewU256(uint64(cur)))
	}
	return res
}

////////////////////////////////////////////////////////////
// Op Codes

type opCodeDomain struct{}

func (opCodeDomain) Equal(a OpCode, b OpCode) bool     { return a == b }
func (opCodeDomain) Less(a OpCode, b OpCode) bool      { panic("not useful") }
func (opCodeDomain) Predecessor(a OpCode) OpCode       { panic("not useful") }
func (opCodeDomain) Successor(a OpCode) OpCode         { panic("not useful") }
func (opCodeDomain) SomethingNotEqual(a OpCode) OpCode { return a + 1 }
func (opCodeDomain) Samples(a OpCode) []OpCode         { return []OpCode{a, a + 1} }

func (opCodeDomain) SamplesForAll([]OpCode) []OpCode {
	res := make([]OpCode, 0, 256)
	for i := 0; i < 256; i++ {
		res = append(res, OpCode(i))
	}
	return res
}

////////////////////////////////////////////////////////////
// Stack Size

type stackSizeDomain struct{}

func (stackSizeDomain) Equal(a int, b int) bool     { return a == b }
func (stackSizeDomain) Less(a int, b int) bool      { return a < b }
func (stackSizeDomain) Predecessor(a int) int       { return a - 1 }
func (stackSizeDomain) Successor(a int) int         { return a + 1 }
func (stackSizeDomain) SomethingNotEqual(a int) int { return (a + 1) % st.MaxStackSize }

func (d stackSizeDomain) Samples(a int) []int {
	return d.SamplesForAll([]int{a})
}
func (stackSizeDomain) SamplesForAll(as []int) []int {
	res := []int{0, st.MaxStackSize / 2, st.MaxStackSize} // interesting values

	// Test every element off by one.
	for _, a := range as {
		if 0 <= a && a <= st.MaxStackSize {
			if a != 0 {
				res = append(res, a-1)
			}
			res = append(res, a)
			if a != st.MaxStackSize {
				res = append(res, a+1)
			}
		}
	}

	// TODO: consider removing duplicates.

	return res
}

////////////////////////////////////////////////////////////
// Address

type addressDomain struct{}

func (addressDomain) Equal(a, b Address) bool {
	return a == b
}

func (addressDomain) Less(Address, Address) bool  { panic("not implemented") }
func (addressDomain) Predecessor(Address) Address { panic("not implemented") }
func (addressDomain) Successor(Address) Address   { panic("not implemented") }

func (addressDomain) SomethingNotEqual(a Address) Address {
	return Address{a[0] + 1}
}

func (ad addressDomain) Samples(a Address) []Address {
	return ad.SamplesForAll([]Address{a})
}

func (addressDomain) SamplesForAll(as []Address) []Address {
	ret := []Address{}
	ret = append(ret, as...)

	zero := Address{}
	ffs := Address{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff}

	ret = append(ret, zero)
	ret = append(ret, ffs)

	return ret
}
