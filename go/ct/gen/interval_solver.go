package gen

import (
	"fmt"
	"strings"

	"golang.org/x/exp/constraints"
	"pgregory.net/rand"
)

type IntervalSolver[T constraints.Integer] struct {
	intervals []interval[T]
}

func NewIntervalSolver[T constraints.Integer](min, max T) *IntervalSolver[T] {
	return &IntervalSolver[T]{
		intervals: []interval[T]{{min, max}},
	}
}

func (s *IntervalSolver[T]) Exclude(min, max T) {
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
	s.AddLowerBoundary(value)
	s.AddUpperBoundary(value)
}

func (s *IntervalSolver[T]) AddLowerBoundary(value T) {
	if len(s.intervals) == 0 {
		return
	}
	min := s.intervals[0].low
	s.Exclude(min, value-1)
}

func (s *IntervalSolver[T]) AddUpperBoundary(value T) {
	if len(s.intervals) == 0 {
		return
	}
	max := s.intervals[len(s.intervals)-1].high
	s.Exclude(value+1, max)
}

func (s *IntervalSolver[T]) IsSatisfiable() bool {
	return len(s.intervals) > 0
}

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

// [a..b] ... pick one element in the range
// [a..b][c..d][e..f] ... pick one element in any of those ranges

type interval[T constraints.Integer] struct {
	low, high T
}

func (i *interval[T]) contains(x T) bool {
	return i.low <= x && x <= i.high
}

type relation int

const (
	isEqual             relation = iota // [1..3] isEqual [1..3]
	isBefore                            // [1..3] is before [4..6]
	isAfter                             // [4..6] is after [1..3]
	isSubset                            // [1..6] isContained in [2..5]
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
