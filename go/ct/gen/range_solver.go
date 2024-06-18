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
	"fmt"
	"math"

	"golang.org/x/exp/constraints"
	"pgregory.net/rand"
)

// RangeSolver is a generic utility to solve constraints of the form
//
//	constraint ::= true | constraint ∧ clause
//	clause     ::= X op C
//	op         ::= < | ≤ | = | ≥ | >
//
// where X is a numeric property to be solved for and C are constants.
// Type T is the the domain for X and C.
type RangeSolver[T constraints.Integer] struct {
	min, max T // < inclusive boundaries
}

func NewRangeSolver[T constraints.Integer](min, max T) *RangeSolver[T] {
	return &RangeSolver[T]{min: min, max: max}
}

func (s *RangeSolver[T]) GetMin() T {
	return s.min
}

func (s *RangeSolver[T]) AddLowerBoundary(min T) {
	if min > s.min {
		s.min = min
	}
}

func (s *RangeSolver[T]) GetMax() T {
	return s.max
}

func (s *RangeSolver[T]) AddUpperBoundary(max T) {
	if max < s.max {
		s.max = max
	}
}

func (s *RangeSolver[T]) AddEqualityConstraint(value T) {
	s.AddLowerBoundary(value)
	s.AddUpperBoundary(value)
}

func (s *RangeSolver[T]) IsSatisfiable() bool {
	return !(s.min > s.max)
}

func (s *RangeSolver[T]) Clone() *RangeSolver[T] {
	return &RangeSolver[T]{min: s.min, max: s.max}
}

func (s *RangeSolver[T]) Restore(backup *RangeSolver[T]) {
	*s = *backup
}

func (s *RangeSolver[T]) Generate(rnd *rand.Rand) (T, error) {
	if s.min > s.max {
		return 0, ErrUnsatisfiable
	}
	diff := s.max - s.min
	if uint64(diff) == math.MaxUint64 {
		return T(rnd.Uint64()), nil
	}
	return T(rnd.Uint64n(uint64(diff)+1)) + s.min, nil
}

func (s *RangeSolver[T]) String() string {
	return s.Print("X")
}

func (s *RangeSolver[T]) Print(value string) string {
	if s.min == s.max {
		return fmt.Sprintf("%s=%v", value, s.min)
	}
	if s.min > s.max {
		return fmt.Sprintf("unsatisfiable(%s)", value)
	}
	return fmt.Sprintf("%v≤%s≤%v", s.min, value, s.max)
}
