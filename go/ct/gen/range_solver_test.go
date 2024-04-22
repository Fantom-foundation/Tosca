//
// Copyright (c) 2024 Fantom Foundation
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at fantom.foundation/bsl11.
//
// Change Date: 2028-4-16
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by the GNU Lesser General Public Licence v3
//

package gen

import (
	"errors"
	"fmt"
	"math"
	"testing"

	"pgregory.net/rand"
)

func TestRangeSolver_ProducesValueInRange(t *testing.T) {
	tests := []struct {
		min, max int
	}{
		{math.MinInt, 12},          // without effective lower boundary
		{5, math.MaxInt},           // without effective upper boundary
		{5, 12},                    // lower and upper boundary
		{6, 6},                     // a fixed value
		{math.MinInt, math.MaxInt}, // the maximum range
	}
	for _, test := range tests {
		t.Run(fmt.Sprintf("[%d,%d]", test.min, test.max), func(t *testing.T) {
			rnd := rand.New()
			solver := NewRangeSolver(test.min, test.max)
			for i := 0; i < 10; i++ {
				sample, err := solver.Generate(rnd)
				if err != nil {
					t.Fatalf("failed to sample value from range, got %v", err)
				}
				if sample < test.min {
					t.Errorf("generated element out of bounds, got %d which should be ≥ %d", sample, test.min)
				}
				if sample > test.max {
					t.Errorf("generated element out of bounds, got %d which should be ≤ %d", sample, test.max)
				}
			}
		})
	}
}

func TestRangeSolver_EmptyRegionsFailToProduceAValue(t *testing.T) {
	rnd := rand.New()
	solver := NewRangeSolver[uint8](4, 2)
	sample, err := solver.Generate(rnd)
	if !errors.Is(err, ErrUnsatisfiable) {
		t.Errorf("solver should have failed to produce a value in unsatisfiable range, got %d with error %v", sample, err)
	}
}

func TestRangeSolver_RangesCanBeIncrementallyConstraint(t *testing.T) {
	// initial boundaries are considered
	solver := NewRangeSolver[int32](1, 200)
	if want, got := "1≤X≤200", solver.String(); want != got {
		t.Errorf("unexpected range restriction, wanted %s, got %s", want, got)
	}

	// boundaries can be further restricted
	solver.AddLowerBoundary(5)
	if want, got := "5≤X≤200", solver.String(); want != got {
		t.Errorf("unexpected range restriction, wanted %s, got %s", want, got)
	}
	solver.AddUpperBoundary(64)
	if want, got := "5≤X≤64", solver.String(); want != got {
		t.Errorf("unexpected range restriction, wanted %s, got %s", want, got)
	}

	// weaker boundaries are ignored
	solver.AddLowerBoundary(3)
	if want, got := "5≤X≤64", solver.String(); want != got {
		t.Errorf("unexpected range restriction, wanted %s, got %s", want, got)
	}
	solver.AddUpperBoundary(100)
	if want, got := "5≤X≤64", solver.String(); want != got {
		t.Errorf("unexpected range restriction, wanted %s, got %s", want, got)
	}
	if !solver.IsSatisfiable() {
		t.Errorf("interval should be solvable: %s", solver.String())
	}

	// equality constraints are supported
	solver.AddEqualityConstraint(7)
	if want, got := "X=7", solver.String(); want != got {
		t.Errorf("unexpected range restriction, wanted %s, got %s", want, got)
	}
	if !solver.IsSatisfiable() {
		t.Errorf("interval should be solvable: %s", solver.String())
	}

	// conflicts lead to unsatisfiable solution
	solver.AddLowerBoundary(8)
	if want, got := "unsatisfiable(X)", solver.String(); want != got {
		t.Errorf("unexpected range restriction, wanted %s, got %s", want, got)
	}
	if solver.IsSatisfiable() {
		t.Errorf("interval should not be solvable: %s", solver.String())
	}
}

func TestRangeSolver_CanHandleMaximumRangeInt64(t *testing.T) {
	rnd := rand.New()
	solver := NewRangeSolver[int64](math.MinInt64, math.MaxInt64)
	_, err := solver.Generate(rnd)
	if err != nil {
		t.Fatalf("failed to produce a sample: %v", err)
	}
}

func TestRangeSolver_CanHandleMaximumRangeUint64(t *testing.T) {
	rnd := rand.New(0)
	solver := NewRangeSolver[uint64](0, math.MaxUint64)
	_, err := solver.Generate(rnd)
	if err != nil {
		t.Fatalf("failed to produce a sample: %v", err)
	}
}
