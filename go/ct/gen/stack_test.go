// Copyright (c) 2024 Fantom Foundation
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at fantom.foundation/bsl11.
//
// Change Date: 2028-4-16
//
// On the date above, in accordance with the Business Source License, use of
// this software will be governed by the GNU Lesser General Public License v3.

package gen

import (
	"errors"
	"testing"

	"pgregory.net/rand"

	. "github.com/Fantom-foundation/Tosca/go/ct/common"
)

func TestStackGenerator_UnconstrainedGeneratorCanProduceStack(t *testing.T) {
	rnd := rand.New(0)
	generator := NewStackGenerator()
	if _, err := generator.Generate(nil, rnd); err != nil {
		t.Fatalf("unexpected error during build: %v", err)
	}
}

func TestStackGenerator_SetSizeIsEnforced(t *testing.T) {
	sizes := []int{0, 1, 2, 42}

	rnd := rand.New(0)
	for _, size := range sizes {
		generator := NewStackGenerator()
		generator.SetSize(size)
		stack, err := generator.Generate(nil, rnd)
		if err != nil {
			t.Fatalf("unexpected error during build: %v", err)
		}
		if want, got := size, stack.Size(); want != got {
			t.Errorf("unexpected stack size, wanted %d, got %d", want, got)
		}
	}
}

func TestStackGenerator_SizeRangesAreEnforced(t *testing.T) {
	sizes := []struct {
		min, max int
	}{
		{0, 0},
		{0, 1},
		{0, 2},
		{0, 3},

		{1, 1},
		{1, 2},
		{1, 3},

		{2, 2},
		{2, 3},
	}

	rnd := rand.New(0)
	for _, size := range sizes {
		generator := NewStackGenerator()
		generator.AddMinSize(size.min)
		generator.AddMaxSize(size.max)
		stack, err := generator.Generate(nil, rnd)
		if err != nil {
			t.Fatalf("unexpected error during build: %v", err)
		}
		if got := stack.Size(); got < size.min || got > size.max {
			t.Errorf("unexpected stack size, wanted size in range [%d,%d], got %d", size.min, size.max, got)
		}
	}
}
func TestStackGenerator_NonConflictingSizesAreAccepted(t *testing.T) {
	generator := NewStackGenerator()
	generator.SetSize(12)
	generator.SetSize(12)
	rnd := rand.New(0)
	if _, err := generator.Generate(nil, rnd); err != nil {
		t.Errorf("generation failed: %v", err)
	}
}

func TestStackGenerator_ConflictingSizesAreDetected(t *testing.T) {
	generator := NewStackGenerator()
	generator.SetSize(12)
	generator.SetSize(14)
	rnd := rand.New(0)
	if _, err := generator.Generate(nil, rnd); !errors.Is(err, ErrUnsatisfiable) {
		t.Errorf("unsatisfiable constraint not detected, got %v", err)
	}
}

func TestStackGenerator_EmptySizeIntervalIsDetected(t *testing.T) {
	generator := NewStackGenerator()
	generator.AddMinSize(14)
	generator.AddMaxSize(12)
	rnd := rand.New(0)
	if _, err := generator.Generate(nil, rnd); !errors.Is(err, ErrUnsatisfiable) {
		t.Errorf("unsatisfiable constraint not detected, got %v", err)
	}
}

func TestStackGenerator_SetValueIsEnforced(t *testing.T) {
	type value struct {
		pos   int
		value U256
	}
	tests := [][]value{
		{{0, NewU256(1)}},
		{{3, NewU256(6)}},
		{{42, NewU256(21)}},
	}

	rnd := rand.New(0)
	for _, test := range tests {
		generator := NewStackGenerator()

		for _, v := range test {
			generator.SetValue(v.pos, v.value)
		}

		stack, err := generator.Generate(nil, rnd)
		if err != nil {
			t.Fatalf("unexpected error during build: %v", err)
		}

		for _, v := range test {
			if want, got := v.value, stack.Get(v.pos); want != got {
				t.Errorf("invalid value at %d, wanted %s, got %s", v.pos, want, got)
			}
		}
	}
}

func TestStackGenerator_BindValueIsEnforced(t *testing.T) {
	generator := NewStackGenerator()

	v := Variable("v")
	generator.BindValue(4, v)

	assignment := Assignment{}
	assignment[v] = NewU256(42)

	stack, err := generator.Generate(assignment, rand.New(0))
	if err != nil {
		t.Fatalf("unexpected error during build: %v", err)
	}
	if want, got := NewU256(42), stack.Get(4); want != got {
		t.Errorf("invalid value, wanted %s, got %s", want, got)
	}
}

func TestStackGenerator_UnboundVariablesAreDetected(t *testing.T) {
	generator := NewStackGenerator()

	v := Variable("v")
	generator.BindValue(3, v)

	_, err := generator.Generate(nil, rand.New(0))
	if !errors.Is(err, ErrUnboundVariable) {
		t.Fatalf("unexpected error during build: %v", err)
	}
}

func TestStackGenerator_ConflictingVariablesAreDetected(t *testing.T) {
	generator := NewStackGenerator()

	v1 := Variable("v1")
	generator.BindValue(3, v1)

	v2 := Variable("v2")
	generator.BindValue(3, v2)

	assignment := Assignment{}
	assignment[v1] = NewU256(42)
	assignment[v2] = NewU256(21)

	_, err := generator.Generate(assignment, rand.New(0))
	if !errors.Is(err, ErrUnsatisfiable) {
		t.Fatalf("unexpected error during build: %v", err)
	}
}

func TestStackGenerator_ConflictingValuesWithVariablesAreDetected(t *testing.T) {
	generator := NewStackGenerator()
	generator.SetValue(3, NewU256(16))

	v := Variable("v")
	generator.BindValue(3, v)

	assignment := Assignment{}
	assignment[v] = NewU256(42)

	_, err := generator.Generate(assignment, rand.New(0))
	if !errors.Is(err, ErrUnsatisfiable) {
		t.Fatalf("unexpected error during build: %v", err)
	}
}

func TestStackGenerator_NegativeValuePositionsAreDetected(t *testing.T) {
	generator := NewStackGenerator()
	generator.SetValue(-1, NewU256(42))
	rnd := rand.New(0)
	if _, err := generator.Generate(nil, rnd); !errors.Is(err, ErrUnsatisfiable) {
		t.Errorf("unsatisfiable constraint not detected, got %v", err)
	}
}

func TestStackGenerator_NegativeVariablePositionsAreDetected(t *testing.T) {
	generator := NewStackGenerator()
	v := Variable("v")
	generator.BindValue(-1, v)

	assignment := Assignment{}
	assignment[v] = NewU256(42)

	if _, err := generator.Generate(assignment, rand.New(0)); !errors.Is(err, ErrUnsatisfiable) {
		t.Errorf("unsatisfiable constraint not detected, got %v", err)
	}
}

func TestStackGenerator_NonConflictingValuesAreAccepted(t *testing.T) {
	generator := NewStackGenerator()
	generator.SetValue(0, NewU256(42))
	generator.SetValue(0, NewU256(42))
	rnd := rand.New(0)
	if _, err := generator.Generate(nil, rnd); err != nil {
		t.Errorf("generation failed: %v", err)
	}
}

func TestStackGenerator_NonConflictingValuePositionsWithRangeSizesAreAccepted(t *testing.T) {
	tests := []struct {
		min, max, pos int
	}{
		{0, 10, 0},
		{0, 10, 1},
		{0, 10, 8},
		{0, 10, 9},

		{5, 10, 0},
		{5, 10, 1},
		{5, 10, 2},
		{5, 10, 7},
		{5, 10, 8},
		{5, 10, 9},
	}

	for _, test := range tests {
		generator := NewStackGenerator()
		generator.AddMinSize(test.min)
		generator.AddMaxSize(test.max)
		generator.SetValue(test.pos, NewU256(21))
		rnd := rand.New(0)
		stack, err := generator.Generate(nil, rnd)
		if err != nil {
			t.Fatalf("failed to generate state for %v: %v", generator, err)
		}

		if size := stack.Size(); size < test.min {
			t.Errorf("invalid size, wanted something >= %d, got %d", test.min, size)
		}
		if size := stack.Size(); size > test.max {
			t.Errorf("invalid size, wanted something <= %d, got %d", test.max, size)
		}
		if got, want := stack.Get(test.pos), NewU256(21); got != want {
			t.Errorf("wrong value at position %d, wanted %v, got %v", test.pos, got, want)
		}
	}
}

func TestStackGenerator_ConflictingValuesAreDetected(t *testing.T) {
	generator := NewStackGenerator()
	generator.SetValue(0, NewU256(42))
	generator.SetValue(0, NewU256(21))
	rnd := rand.New(0)
	if _, err := generator.Generate(nil, rnd); !errors.Is(err, ErrUnsatisfiable) {
		t.Errorf("unsatisfiable constraint not detected, got %v", err)
	}
}

func TestStackGenerator_ConflictingValuePositionsWithSizeRangeAreDetected(t *testing.T) {
	tests := []struct {
		min, max, pos int
	}{
		// Position exceeding upper bound.
		{0, 10, 10},
		{0, 10, 11},
		{0, 10, 12},

		// Position hitting both bounds.
		{10, 10, 10},
	}

	for _, test := range tests {
		generator := NewStackGenerator()
		generator.AddMinSize(test.min)
		generator.AddMaxSize(test.max)
		generator.SetValue(test.pos, NewU256(21))
		rnd := rand.New(0)
		if _, err := generator.Generate(nil, rnd); !errors.Is(err, ErrUnsatisfiable) {
			t.Errorf("unsatisfiable constraint not detected, got %v", err)
		}
	}
}

func TestStackGenerator_ConflictingVariablePositionsWithSizesAreDetected(t *testing.T) {
	generator := NewStackGenerator()
	generator.SetSize(10)

	v := Variable("v")
	generator.BindValue(10, v)

	assignment := Assignment{}
	assignment[v] = NewU256(42)

	if _, err := generator.Generate(assignment, rand.New(0)); !errors.Is(err, ErrUnsatisfiable) {
		t.Errorf("unsatisfiable constraint not detected, got %v", err)
	}
}

func TestStackGenerator_CloneCopiesGeneratorState(t *testing.T) {
	original := NewStackGenerator()
	original.SetSize(5)
	original.SetValue(0, NewU256(42))
	original.SetValue(0, NewU256(43))

	v := Variable("v")
	original.BindValue(1, v)

	clone := original.Clone()

	if want, got := original.String(), clone.String(); want != got {
		t.Errorf("invalid clone, wanted %s, got %s", want, got)
	}
}

func TestStackGenerator_ClonesAreIndependent(t *testing.T) {
	v := Variable("v")

	base := NewStackGenerator()
	base.SetSize(5)

	clone1 := base.Clone()
	clone1.SetValue(0, NewU256(16))
	clone1.BindValue(2, v)

	clone2 := base.Clone()
	clone2.SetValue(0, NewU256(17))
	clone2.BindValue(1, v)

	want := "{size=5,value[0]=0000000000000000 0000000000000000 0000000000000000 0000000000000010,value[2]=$v}"
	if got := clone1.String(); got != want {
		t.Errorf("invalid clone, wanted %s, got %s", want, got)
	}

	want = "{size=5,value[0]=0000000000000000 0000000000000000 0000000000000000 0000000000000011,value[1]=$v}"
	if got := clone2.String(); got != want {
		t.Errorf("invalid clone, wanted %s, got %s", want, got)
	}
}

func TestStackGenerator_ClonesCanBeUsedToResetGenerator(t *testing.T) {
	v := Variable("v")

	generator := NewStackGenerator()
	generator.SetValue(0, NewU256(42))
	generator.BindValue(2, v)

	backup := generator.Clone()

	generator.SetSize(5)
	generator.SetValue(1, NewU256(16))
	generator.BindValue(3, v)

	want := "{size=5,value[0]=0000000000000000 0000000000000000 0000000000000000 000000000000002a,value[1]=0000000000000000 0000000000000000 0000000000000000 0000000000000010,value[2]=$v,value[3]=$v}"
	if got := generator.String(); got != want {
		t.Errorf("unexpected generator state, wanted %s, got %s", want, got)
	}

	generator.Restore(backup)
	want = "{0≤size≤1024,value[0]=0000000000000000 0000000000000000 0000000000000000 000000000000002a,value[2]=$v}"
	if got := generator.String(); got != want {
		t.Errorf("unexpected generator state, wanted %s, got %s", want, got)
	}
}
