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
	"github.com/Fantom-foundation/Tosca/go/ct/st"
	"github.com/Fantom-foundation/Tosca/go/vm"
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
func (boolDomain) Less(a bool, b bool) bool  { return !a && b }
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

	res = removeDuplicatesGeneric[uint16](res)

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

	res = removeDuplicatesGeneric[uint64](res)

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

	res = removeDuplicatesGeneric[U256](res)

	return res
}

// valueDomain is a domain value parameters of call expressions.
type valueDomain struct {
	u256Domain
}

func (d valueDomain) Samples(a U256) []U256 {
	return d.SamplesForAll([]U256{a})
}

func (d valueDomain) SamplesForAll(as []U256) []U256 {
	res := []U256{}

	// Test every element off by one.
	for _, a := range as {
		res = append(res, d.Predecessor(a))
		res = append(res, a)
		res = append(res, d.Successor(a))
	}

	return res
}

////////////////////////////////////////////////////////////
// Revision

type revisionDomain struct{}

func (revisionDomain) Equal(a Revision, b Revision) bool { return a == b }
func (revisionDomain) Less(a Revision, b Revision) bool  { return a < b }
func (revisionDomain) Predecessor(a Revision) Revision {
	numRevisions := R99_UnknownNextRevision + 1
	return (a + numRevisions - 1) % numRevisions
}

func (revisionDomain) Successor(a Revision) Revision {
	numRevisions := R99_UnknownNextRevision + 1
	return (a + 1) % numRevisions
}

func (d revisionDomain) SomethingNotEqual(a Revision) Revision {
	return d.Successor(a)
}

func (d revisionDomain) Samples(a Revision) []Revision {
	return d.SamplesForAll(nil)
}

func (revisionDomain) SamplesForAll(a []Revision) []Revision {
	res := []Revision{}
	for r := R07_Istanbul; r <= R99_UnknownNextRevision; r++ {
		res = append(res, r)
	}
	return res
}

////////////////////////////////////////////////////////////
// st.Status

type statusCodeDomain struct{}

func (statusCodeDomain) Equal(a st.StatusCode, b st.StatusCode) bool { return a == b }
func (statusCodeDomain) Less(a st.StatusCode, b st.StatusCode) bool  { return a < b }
func (statusCodeDomain) Predecessor(a st.StatusCode) st.StatusCode   { panic("not useful") }
func (statusCodeDomain) Successor(a st.StatusCode) st.StatusCode     { panic("not useful") }

func (statusCodeDomain) SomethingNotEqual(a st.StatusCode) st.StatusCode {
	if a == st.Running {
		return st.Stopped
	}
	return st.Running
}

func (statusCodeDomain) Samples(a st.StatusCode) []st.StatusCode {
	return []st.StatusCode{st.Running, st.Stopped, st.Reverted, st.Failed}
}

func (statusCodeDomain) SamplesForAll(a []st.StatusCode) []st.StatusCode {
	return []st.StatusCode{st.Running, st.Stopped, st.Reverted, st.Failed}
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
func (opCodeDomain) Less(a OpCode, b OpCode) bool      { return a < b }
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
	res := []int{0, st.MaxStackSize} // extreme values

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

	res = removeDuplicatesGeneric[int](res)

	return res
}

////////////////////////////////////////////////////////////
// Address

type addressDomain struct{}

func (addressDomain) Equal(a, b vm.Address) bool {
	return a == b
}

func (addressDomain) Less(vm.Address, vm.Address) bool  { panic("not implemented") }
func (addressDomain) Predecessor(vm.Address) vm.Address { panic("not implemented") }
func (addressDomain) Successor(vm.Address) vm.Address   { panic("not implemented") }

func (addressDomain) SomethingNotEqual(a vm.Address) vm.Address {
	return vm.Address{a[0] + 1}
}

func (ad addressDomain) Samples(a vm.Address) []vm.Address {
	return ad.SamplesForAll([]vm.Address{a})
}

func (addressDomain) SamplesForAll(as []vm.Address) []vm.Address {
	ret := []vm.Address{}
	ret = append(ret, as...)

	zero := vm.Address{}
	ffs := vm.Address{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff}

	ret = append(ret, zero)
	ret = append(ret, ffs)

	return ret
}

////////////////////////////////////////////////////////////
// Gas with upper bound

type gasDomain struct{}

func (gasDomain) Equal(a vm.Gas, b vm.Gas) bool     { return a == b }
func (gasDomain) Less(a vm.Gas, b vm.Gas) bool      { return a < b }
func (gasDomain) Predecessor(a vm.Gas) vm.Gas       { return a - 1 }
func (gasDomain) Successor(a vm.Gas) vm.Gas         { return a + 1 }
func (gasDomain) SomethingNotEqual(a vm.Gas) vm.Gas { return a + 1 }

func (d gasDomain) Samples(a vm.Gas) []vm.Gas {
	return d.SamplesForAll([]vm.Gas{a})
}

func (gasDomain) SamplesForAll(as []vm.Gas) []vm.Gas {
	res := []vm.Gas{0, st.MaxGas}

	// Test every element off by one.
	for _, a := range as {
		res = append(res, a-1)
		res = append(res, a)
		res = append(res, a+1)
	}

	res = removeDuplicatesGeneric[vm.Gas](res)

	return res
}

////////////////////////////////////////////////////////////
// BlockNumber Range Domain

type BlockNumberOffsetDomain struct{}

func (BlockNumberOffsetDomain) Equal(a int64, b int64) bool { return a == b }
func (BlockNumberOffsetDomain) Less(a int64, b int64) bool  { return a < b }
func (BlockNumberOffsetDomain) Predecessor(a int64) int64   { return a - 1 }
func (BlockNumberOffsetDomain) Successor(a int64) int64     { return a + 1 }
func (BlockNumberOffsetDomain) SomethingNotEqual(a int64) int64 {
	return a + 2
}

func (d BlockNumberOffsetDomain) Samples(a int64) []int64 {
	return d.SamplesForAll([]int64{a})
}

func (BlockNumberOffsetDomain) SamplesForAll(as []int64) []int64 {
	res := []int64{math.MinInt64, -1, 0, 1, 255, 256, 257, math.MaxInt64}

	// Test every element off by one.
	for _, a := range as {
		res = append(res, a-1)
		res = append(res, a)
		res = append(res, a+1)
	}

	res = removeDuplicatesGeneric[int64](res)

	return res
}

func removeDuplicatesGeneric[T comparable](slice []T) []T {
	seen := make(map[T]bool)
	result := []T{}

	for _, value := range slice {
		if _, ok := seen[value]; !ok {
			seen[value] = true
			result = append(result, value)
		}
	}

	return result
}
