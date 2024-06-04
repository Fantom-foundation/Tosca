package gen

import (
	"fmt"
	"slices"
	"strings"

	"golang.org/x/exp/constraints"
	"pgregory.net/rand"
)

// IntervalSolver is a generic utility to solve constraints of the form
//
//	X ∈ [a1..b1] ∪ [a2..b2] ∪ ... ∪ [an..bn]
//
// The solver maintains a list of inclusive intervals [a..b] that represent the
// domain of the variable x. The solver can exclude ranges from the domain and
// check if the domain is empty, making the constraints unsatisfiable.
type IntervalSolver[T constraints.Integer] struct {
	intervals []interval[T]
}

// NewIntervalSolver creates a new IntervalSolver with the domain [min..max].
// If min > max, the solver is initialized with an empty domain.
func NewIntervalSolver[T constraints.Integer](min, max T) *IntervalSolver[T] {
	initialInterval := interval[T]{low: min, high: max}
	if initialInterval.isEmpty() {
		return &IntervalSolver[T]{}
	}
	return &IntervalSolver[T]{
		intervals: []interval[T]{initialInterval},
	}
}

// Exclude removes the range [min..max] from the domain of the solver.
// If min > max, the interval would be empty and so is ignored.
func (s *IntervalSolver[T]) Exclude(min, max T) {
	// this insures an empty interval will never be added.
	if max < min {
		return
	}
	res := []interval[T]{}
	for _, current := range s.intervals {
		newLeft := interval[T]{current.low, min - 1}
		newRight := interval[T]{max + 1, current.high}

		switch current.getRelationTo(&interval[T]{min, max}) {
		case isEqual:
			continue // we remove this interval
		case isSubset:
			res = append(res, newLeft, newRight)
		case isSuperset:
			continue // the current interval is dropped
		case intersectsWithStart:
			res = append(res, newRight)
		case intersectsWithEnd:
			res = append(res, newLeft)
		default:
			res = append(res, current)
		}

	}
	s.intervals = res
}

func (s *IntervalSolver[T]) Contains(value T) bool {
	for _, interval := range s.intervals {
		if interval.contains(value) {
			return true
		}
	}
	return false
}

func (s *IntervalSolver[T]) AddEqualityConstraint(value T) {
	s.AddUpperBoundary(value)
	s.AddLowerBoundary(value)
}

func (s *IntervalSolver[T]) AddLowerBoundary(value T) {
	if len(s.intervals) == 0 {
		return
	}
	min := s.intervals[0].low
	if value <= min {
		return
	}
	s.Exclude(min, value-1)
}

func (s *IntervalSolver[T]) AddUpperBoundary(value T) {
	if len(s.intervals) == 0 {
		return
	}
	max := s.intervals[len(s.intervals)-1].high
	if value >= max {
		return
	}
	s.Exclude(value+1, max)
}

func (s *IntervalSolver[T]) IsSatisfiable() bool {
	return len(s.intervals) > 0
}

// Generate returns a random value from the domain of the solver.
func (s *IntervalSolver[T]) Generate(rnd *rand.Rand) (T, error) {
	if len(s.intervals) == 0 {
		return 0, ErrUnsatisfiable
	}

	domainSize := uint64(0)
	for _, interval := range s.intervals {
		domainSize += uint64(interval.high - interval.low + 1)
	}

	if domainSize == 0 { // the domain is the full uint64 domain
		return T(rnd.Uint64()), nil
	}

	sample := rnd.Uint64n(domainSize)
	for _, interval := range s.intervals {
		if uint64(interval.high-interval.low+1) > sample {
			return interval.low + T(sample), nil
		}
		sample -= uint64(interval.high - interval.low + 1)
	}
	return 0, fmt.Errorf("internal error")
}

func (s *IntervalSolver[T]) String() string {
	if len(s.intervals) == 0 {
		return "false"
	}
	clauses := []string{}
	for _, interval := range s.intervals {
		clauses = append(clauses, fmt.Sprintf("[%d..%d]", interval.low, interval.high))
	}
	return fmt.Sprintf("X ∈ %s", strings.Join(clauses, " ∪ "))
}

func (s *IntervalSolver[T]) Clone() *IntervalSolver[T] {
	return &IntervalSolver[T]{intervals: slices.Clone(s.intervals)}
}

func (s *IntervalSolver[T]) Equals(other *IntervalSolver[T]) bool {
	return slices.Equal(s.intervals, other.intervals)
}

// interval represents an interval [a..b] of type T.
type interval[T constraints.Integer] struct {
	low, high T
}

func (i *interval[T]) contains(x T) bool {
	return i.low <= x && x <= i.high
}

func (i *interval[T]) isEmpty() bool {
	return i.low > i.high
}

type relation int

const (
	isEqual             relation = iota // [1..3] isEqual [1..3]
	isBefore                            // [1..3] is before [4..6]
	isAfter                             // [4..6] is after [1..3]
	isSubset                            // [1..6] contains [2..5]
	isSuperset                          // [2..5] isEnclosed in [1..6]
	intersectsWithStart                 // [1..3] intersectsWithStart [2..6]
	intersectsWithEnd                   // [2..6] intersectsWithEnd [1..5]
)

func (i *interval[T]) getRelationTo(other *interval[T]) relation {
	if *i == *other {
		return isEqual
	}
	if i.high < other.low {
		return isBefore
	}
	if i.low > other.high {
		return isAfter
	}
	if i.low < other.low && other.high < i.high {
		return isSubset
	}
	if other.low <= i.low && i.high <= other.high {
		return isSuperset
	}
	if other.low <= i.high && i.high <= other.high {
		return intersectsWithEnd
	}
	return intersectsWithStart
}
