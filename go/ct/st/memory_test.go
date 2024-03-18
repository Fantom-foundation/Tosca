package st

import (
	"math"
	"testing"

	"golang.org/x/exp/slices"

	. "github.com/Fantom-foundation/Tosca/go/ct/common"
	"github.com/Fantom-foundation/Tosca/go/vm"
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

func TestMemory_ExpansionCosts(t *testing.T) {
	mem := NewMemory()

	cost, offset, size := mem.ExpansionCosts(NewU256(128), NewU256(32))
	if want, got := vm.Gas(15), cost; want != got {
		t.Errorf("Expansion cost calculation wrong, want %v got %v", want, got)
	}

	if want, got := uint64(128), offset; want != got {
		t.Errorf("Offset conversion wrong, want %v got %v", want, got)
	}
	if want, got := uint64(32), size; want != got {
		t.Errorf("Size conversion wrong, want %v got %v", want, got)
	}
}

func TestMemory_ExpansionCostsSizeTooBig(t *testing.T) {
	mem := NewMemory()

	cost, offset, size := mem.ExpansionCosts(NewU256(128), NewU256(1, 0))
	if want, got := vm.Gas(math.MaxInt64), cost; want != got {
		t.Errorf("Expansion cost calculation wrong, want %v got %v", want, got)
	}

	if want, got := uint64(0), offset; want != got {
		t.Errorf("Offset conversion wrong, want %v got %v", want, got)
	}
	if want, got := uint64(0), size; want != got {
		t.Errorf("Size conversion wrong, want %v got %v", want, got)
	}
}

func TestMemory_ExpansionCostsOffsetTooBig(t *testing.T) {
	mem := NewMemory()

	cost, offset, size := mem.ExpansionCosts(NewU256(1, 0), NewU256(32))
	if want, got := vm.Gas(math.MaxInt64), cost; want != got {
		t.Errorf("Expansion cost calculation wrong, want %v got %v", want, got)
	}

	if want, got := uint64(0), offset; want != got {
		t.Errorf("Offset conversion wrong, want %v got %v", want, got)
	}
	if want, got := uint64(0), size; want != got {
		t.Errorf("Size conversion wrong, want %v got %v", want, got)
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
