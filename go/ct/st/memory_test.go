// Copyright (c) 2024 Fantom Foundation
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at fantom.foundation/bsl11.
//
// Change Date: 2028-4-16
//
// On the date above, in accordance with the Business Source License, use of
// this software will be governed by the GNU Lesser General Public License v3.

package st

import (
	"math"
	"testing"

	"golang.org/x/exp/slices"

	. "github.com/Fantom-foundation/Tosca/go/ct/common"
	"github.com/Fantom-foundation/Tosca/go/tosca"
)

func TestMemory_NewMemory(t *testing.T) {
	mem := NewMemory()
	if want, got := 0, mem.Size(); want != got {
		t.Errorf("unexpected memory size, want %v, got %v", want, got)
	}

	mem = NewMemory(1, 2, 3)
	if want, got := 3, mem.Size(); want != got {
		t.Errorf("unexpected memory size, want %v, got %v", want, got)
	}
}

func TestMemory_Clone(t *testing.T) {
	mem := NewMemory(1, 2, 3)
	clone := mem.Clone()

	if mem.Size() != clone.Size() {
		t.Error("Clone does not have the same size")
	}

	mem.Write([]byte{4, 5}, 0)
	if !slices.Equal(clone.Read(0, 2), []byte{1, 2}) {
		t.Error("Clone is not independent from original")
	}
}

func TestMemory_Set(t *testing.T) {
	mem := NewMemory(1, 2, 3)

	slice := []byte{4, 5, 6}
	mem.Set(slice)

	if !slices.Equal(mem.Read(0, 3), []byte{4, 5, 6}) {
		t.Error("Set is broken")
	}

	slice[0] = 7
	if !slices.Equal(mem.Read(0, 3), []byte{4, 5, 6}) {
		t.Error("Set does not copy the given slice")
	}
}

func TestMemory_Append(t *testing.T) {
	mem := NewMemory(1, 2)

	slice := []byte{4, 5}
	mem.Append(slice)

	if !slices.Equal(mem.Read(0, 4), []byte{1, 2, 4, 5}) {
		t.Error("Append is broken")
	}

	slice[0] = 7
	if !slices.Equal(mem.Read(0, 4), []byte{1, 2, 4, 5}) {
		t.Error("Append does not copy the given slice")
	}
}

func TestMemory_Read(t *testing.T) {
	mem := NewMemory(1, 2, 3)

	if !slices.Equal(mem.Read(1, 2), []byte{2, 3}) {
		t.Error("Read is broken")
	}

	if !slices.Equal(mem.Read(2, 4), []byte{3, 0, 0, 0}) {
		t.Error("Read did not zero-initialize out-of-bounds values")
	}

	if mem.Size() != 32 {
		t.Error("Out-of-bounds read did not grow memory correctly")
	}
}

func TestMemory_ReadSize0(t *testing.T) {
	mem := NewMemory(1, 2, 3)

	if !slices.Equal(mem.Read(10, 0), []byte{}) {
		t.Error("Read is broken")
	}

	if mem.Size() != 3 {
		t.Error("Out-of-bounds read with size 0 should not grow memory")
	}
}

func TestMemory_Write(t *testing.T) {
	mem := NewMemory(1, 2)

	mem.Write([]byte{4, 5, 6}, 3)

	if !slices.Equal(mem.Read(0, 6), []byte{1, 2, 0, 4, 5, 6}) {
		t.Error("Write is broken")
	}

	if mem.Size() != 32 {
		t.Error("Out-of-bounds write did not grow memory correctly")
	}
}

func TestMemory_WriteSize0(t *testing.T) {
	mem := NewMemory(1, 2, 3)

	mem.Write([]byte{}, 6)

	if mem.Size() != 3 {
		t.Error("Out-of-bounds write with size 0 should not grow memory")
	}
}

func TestMemory_MemoryExpansionCosts(t *testing.T) {
	tests := map[string]struct {
		offset U256
		size   U256

		wantCost   tosca.Gas
		wantOffset uint64
		wantSize   uint64
	}{
		"large size no overflow":            {NewU256(0), NewU256(math.MaxUint64), math.MaxInt64, 0, math.MaxUint64},
		"large offset no overflow":          {NewU256(math.MaxUint64 - 1), NewU256(1), math.MaxInt64, math.MaxUint64 - 1, 1},
		"large offset and size no overflow": {NewU256(math.MaxUint64 / 2), NewU256(math.MaxUint64 / 2), math.MaxInt64, math.MaxUint64 / 2, math.MaxUint64 / 2},
		"size overflow":                     {NewU256(0), NewU256(1, 0), tosca.Gas(math.MaxInt64), 0, 0},
		"offset overflow":                   {NewU256(1, 0), NewU256(1), tosca.Gas(math.MaxInt64), 0, 0},
		"large offset and size overflow":    {NewU256(math.MaxUint64/2 + 1), NewU256(math.MaxUint64/2 + 1), tosca.Gas(math.MaxInt64), math.MaxUint64/2 + 1, math.MaxUint64/2 + 1},
		"zero size":                         {NewU256(0), NewU256(0), tosca.Gas(0), 0, 0},
		"zero size offset":                  {NewU256(1024), NewU256(0), tosca.Gas(0), 1024, 0},
		"zero size offset overflow":         {NewU256(1, 0), NewU256(0), tosca.Gas(0), 0, 0},
		"max memory size allowed":           {NewU256(0), NewU256(MaxMemoryExpansionSize + 1), math.MaxInt64, 0, MaxMemoryExpansionSize + 1},
		"acceptable size":                   {NewU256(0), NewU256(MaxMemoryExpansionSize), tosca.Gas(36028809887088637), 0, MaxMemoryExpansionSize},
		"acceptable offset":                 {NewU256(MaxMemoryExpansionSize - 1), NewU256(1), tosca.Gas(36028809887088637), MaxMemoryExpansionSize - 1, 1},
		"size not multiple of 32":           {NewU256(0), NewU256(31), tosca.Gas(3), 0, 31},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			mem := NewMemory()

			cost, offset, size := mem.ExpansionCosts(test.offset, test.size)

			if want, got := test.wantCost, cost; want != got {
				t.Errorf("Expansion cost calculation wrong, want %v got %v", want, got)
			}

			if want, got := test.wantOffset, offset; want != got {
				t.Errorf("Offset conversion wrong, want %v got %v", want, got)
			}

			if want, got := test.wantSize, size; want != got {
				t.Errorf("Size conversion wrong, want %v got %v", want, got)
			}
		})
	}
}

func TestMemory_Eq(t *testing.T) {
	mem1 := NewMemory(1, 2, 3)
	mem2 := mem1.Clone()

	if !mem1.Eq(mem1) {
		t.Error("Self-comparison is broken")
	}

	if !mem1.Eq(mem2) {
		t.Error("Clones are not equal")
	}

	mem2 = NewMemory(1, 2)
	if mem1.Eq(mem2) {
		t.Error("Equality does not consider all elements")
	}

	mem2 = NewMemory(4, 2, 3)
	if mem1.Eq(mem2) {
		t.Error("Equality does not consider all elements")
	}

	mem2 = NewMemory(1, 2, 3)
	if !mem1.Eq(mem2) {
		t.Error("Equality does not support separate instances")
	}
}
