package gen

import (
	"math"
	"testing"

	"pgregory.net/rand"
)

func TestIntervalSolver_CanFormulateConstraints(t *testing.T) {
	tests := map[string]struct {
		setup func(*IntervalSolver[int32])
		want  string
	}{
		"default": {
			setup: func(s *IntervalSolver[int32]) {},
			want:  "X ∈ [200..400]",
		},
		"remove-something-before": {
			setup: func(s *IntervalSolver[int32]) {
				s.Exclude(100, 150)
			},
			want: "X ∈ [200..400]",
		},
		"remove-something-after": {
			setup: func(s *IntervalSolver[int32]) {
				s.Exclude(500, 550)
			},
			want: "X ∈ [200..400]",
		},
		"remove-equal": {
			setup: func(s *IntervalSolver[int32]) {
				s.Exclude(200, 400)
			},
			want: "false",
		},
		"remove-subset": {
			setup: func(s *IntervalSolver[int32]) {
				s.Exclude(250, 340)
			},
			want: "X ∈ [200..249] ∪ [341..400]",
		},
		"remove-superset": {
			setup: func(s *IntervalSolver[int32]) {
				s.Exclude(150, 440)
			},
			want: "false",
		},
		"remove-touches-on-the-left": {
			setup: func(s *IntervalSolver[int32]) {
				s.Exclude(150, 250)
			},
			want: "X ∈ [251..400]",
		},
		"remove-touches-on-the-right": {
			setup: func(s *IntervalSolver[int32]) {
				s.Exclude(350, 450)
			},
			want: "X ∈ [200..349]",
		},
		"fragment-range": {
			setup: func(s *IntervalSolver[int32]) {
				s.Exclude(250, 300)
				s.Exclude(320, 380)
			},
			want: "X ∈ [200..249] ∪ [301..319] ∪ [381..400]",
		},
		"fragment-range-with-overlap": {
			setup: func(s *IntervalSolver[int32]) {
				s.Exclude(250, 300)
				s.Exclude(320, 380)
				s.Exclude(220, 310)
			},
			want: "X ∈ [200..219] ∪ [311..319] ∪ [381..400]",
		},
		"remove-empty-interval": {
			setup: func(s *IntervalSolver[int32]) {
				s.Exclude(300, 250) // < considered empty
			},
			want: "X ∈ [200..400]",
		},
		"equality-constraint": {
			setup: func(s *IntervalSolver[int32]) {
				s.AddEqualityConstraint(300)
			},
			want: "X ∈ [300..300]",
		},
		"lower-boundary-constraint": {
			setup: func(s *IntervalSolver[int32]) {
				s.AddLowerBoundary(300)
			},
			want: "X ∈ [300..400]",
		},
		"upper-boundary-constraint": {
			setup: func(s *IntervalSolver[int32]) {
				s.AddUpperBoundary(300)
			},
			want: "X ∈ [200..300]",
		},
		// add test for upper boundary constraint but with value off-range (lower than low, and higher than up)
		"upper-boundary-constraint-off-range-high": {
			setup: func(s *IntervalSolver[int32]) {
				s.AddUpperBoundary(500)
			},
			want: "X ∈ [200..400]",
		},
		"upper-boundary-constraint-off-range-low": {
			setup: func(s *IntervalSolver[int32]) {
				s.AddUpperBoundary(100)
			},
			want: "false",
		},
		// add test for lower boundary constraint but with value off-range (lower than low, and higher than up)
		"lower-boundary-constraint-off-range-high": {
			setup: func(s *IntervalSolver[int32]) {
				s.AddLowerBoundary(500)
			},
			want: "false",
		},
		"lower-boundary-constraint-off-range-low": {
			setup: func(s *IntervalSolver[int32]) {
				s.AddLowerBoundary(100)
			},
			want: "X ∈ [200..400]",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			solver := NewIntervalSolver[int32](200, 400)
			test.setup(solver)
			got := solver.String()
			if got != test.want {
				t.Errorf("unexpected result, got %v, want %v", got, test.want)
			}
		})
	}
}

func TestIntervalSolver_Exclude(t *testing.T) {
	const N = 20
	// remove interval [c..d] from [a..b]
	for a := 0; a < N; a++ {
		for b := 0; b < N; b++ {
			for c := 0; c < N; c++ {
				for d := 0; d < N; d++ {
					solver := NewIntervalSolver(uint32(a), uint32(b))
					solver.Exclude(uint32(c), uint32(d))
					for i := 0; i < N; i++ {
						inAtoB := a <= i && i <= b
						inCtoD := c <= i && i <= d
						want := inAtoB && !inCtoD
						got := solver.Contains(uint32(i))
						if got != want {
							t.Fatalf("unexpected result for %d ∈ [%d..%d] \\ [%d..%d]: got %v, want %v", i, a, b, c, d, got, want)
						}
					}
				}
			}
		}
	}
}

func TestIntervalSolver_EmptyInterval(t *testing.T) {
	solver := NewIntervalSolver[int32](200, 100)
	_, err := solver.Generate(rand.New())
	if err != ErrUnsatisfiable {
		t.Errorf("empty interval should not be solvable")
	}
}

func TestIntervalSolver_ExcludeEmptyIntervalHasNoEffect(t *testing.T) {
	solver := NewIntervalSolver[int32](200, 400)
	clone := solver.Clone()
	clone.Exclude(300, 250)
	if !solver.Equals(clone) {
		t.Errorf("excluding empty interval should have no effect")
	}
}

func TestIntervalSolver_TestIntervals(t *testing.T) {
	solver := NewIntervalSolver[int32](200, 400)
	solver.Exclude(250, 300)

	rnd := rand.New()
	for i := 0; i < 100; i++ {
		res, err := solver.Generate(rnd)
		if err != nil {
			t.Fatalf("error solving intervals: %v", err)
		}
		if res < 200 || res > 400 || (res >= 250 && res <= 300) {
			t.Fatalf("produced unexpected result for condition %v: %d", solver, res)
		}
	}
}

func TestIntervalSolver_IsSatisfiable(t *testing.T) {
	solver := NewIntervalSolver[int32](200, 400)
	if !solver.IsSatisfiable() {
		t.Fatalf("interval should be solvable: %s", solver.String())
	}
	solver.intervals = nil
	if solver.IsSatisfiable() {
		t.Fatalf("empty interval should be unsolvable")
	}
}

func TestIntervalSolver_AddBoundariesInEmptySolverHasNoEffect(t *testing.T) {

	tests := map[string]func(*IntervalSolver[int32]){
		"lower":    func(s *IntervalSolver[int32]) { s.AddLowerBoundary(300) },
		"upper":    func(s *IntervalSolver[int32]) { s.AddUpperBoundary(300) },
		"equality": func(s *IntervalSolver[int32]) { s.AddEqualityConstraint(300) },
	}

	for name, setup := range tests {
		t.Run(name, func(t *testing.T) {

			solver := NewIntervalSolver[int32](400, 200)
			clone := solver.Clone()
			setup(clone)
			if !solver.Equals(clone) {
				t.Fatalf("empty interval solver should not be modified by adding boundaries")
			}
		})
	}
}

func TestIntervalSolver_Clone(t *testing.T) {
	solver := NewIntervalSolver[int32](200, 400)
	clone := solver.Clone()
	if !solver.Equals(clone) {
		t.Errorf("cloned should be same as original")
	}
	clone.AddLowerBoundary(300)
	if solver.Equals(clone) {
		t.Fatalf("cloned solver should be a distinct instance")
	}
}

func TestIntervalSolver_Equals(t *testing.T) {
	solver1 := NewIntervalSolver[int32](200, 400)
	solver2 := NewIntervalSolver[int32](100, 400)
	if !solver1.Equals(solver1) {
		t.Errorf("equals fails comparison with same instance")
	}
	if solver1.Equals(solver2) {
		t.Errorf("equals fails to report different instances")
	}
}

func TestInterval_isEmpty(t *testing.T) {
	tests := map[string]struct {
		low  int32
		high int32
		want bool
	}{
		"empty":     {low: 1, high: 0, want: true},
		"non-empty": {low: 0, high: 1, want: false},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			got := (&interval[int32]{low: test.low, high: test.high}).isEmpty()
			if got != test.want {
				t.Errorf("unexpected result, got %v, want %v", got, test.want)
			}
		})
	}
}

func TestIntervalSolver_uint64fullrangeAndEdges(t *testing.T) {

	tests := map[string]struct {
		setup func(*IntervalSolver[uint64])
		check func(uint64) bool
	}{
		"full-range": {
			setup: func(s *IntervalSolver[uint64]) {},
			check: func(v uint64) bool { return v >= 0 && v <= math.MaxUint64 }},
		"remove-zero": {
			setup: func(s *IntervalSolver[uint64]) { s.Exclude(0, 0) },
			check: func(v uint64) bool { return v > 0 && v < math.MaxUint64 }},
		"remove-max": {
			setup: func(s *IntervalSolver[uint64]) { s.Exclude(math.MaxUint64, math.MaxUint64) },
			check: func(v uint64) bool { return v >= 0 && v < math.MaxUint64-1 }},
	}

	rnd := rand.New()

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			solver := NewIntervalSolver[uint64](0, math.MaxUint64)
			test.setup(solver)
			for i := 0; i < 10000; i++ {
				res, err := solver.Generate(rnd)
				if err != nil {
					t.Fatalf("error solving intervals: %v at step %v", err, i)
				}
				if !test.check(res) {
					t.Fatalf("produced unexpected result for condition %v: %d - at step %v", solver, res, i)
				}
			}
		})
	}
}

func TestIntervalSolver_int64fullrangeAndEdges(t *testing.T) {

	tests := map[string]struct {
		setup func(*IntervalSolver[int64])
		check func(int64) bool
	}{
		"full-range": {
			setup: func(s *IntervalSolver[int64]) {},
			check: func(v int64) bool { return v >= math.MinInt64 && v <= math.MaxInt64 }},
		"remove-min": {
			setup: func(s *IntervalSolver[int64]) { s.Exclude(math.MinInt64, math.MinInt64) },
			check: func(v int64) bool { return v != math.MaxInt64 }},
		"remove-max": {
			setup: func(s *IntervalSolver[int64]) { s.Exclude(math.MaxInt64, math.MaxInt64) },
			check: func(v int64) bool { return v != math.MaxInt64 }},
		"remove-all-positive": {
			setup: func(s *IntervalSolver[int64]) { s.Exclude(0, math.MaxInt64) },
			check: func(v int64) bool { return v < 0 }},
		"remove-all-negative": {
			setup: func(s *IntervalSolver[int64]) { s.Exclude(math.MinInt64, 0) },
			check: func(v int64) bool { return v > 0 }},
		"remove-zero": {
			setup: func(s *IntervalSolver[int64]) { s.Exclude(0, 0) },
			check: func(v int64) bool { return v != 0 }},
	}

	rnd := rand.New(0)

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			solver := NewIntervalSolver[int64](math.MinInt64, math.MaxInt64)
			test.setup(solver)
			for i := 0; i < 10000; i++ {
				res, err := solver.Generate(rnd)
				if err != nil {
					t.Fatalf("error solving intervals: %v at step %v", err, i)
				}
				if !test.check(res) {
					t.Fatalf("produced unexpected result for condition %v: %d - at step %v", solver, res, i)
				}
			}
		})
	}
}

func TestIntervalSolver_int64Edges(t *testing.T) {

	tests := map[string]struct {
		setup func(*IntervalSolver[int64])
	}{
		"add-lower": {
			setup: func(s *IntervalSolver[int64]) { s.AddLowerBoundary(math.MinInt64) }},
		"add-upper": {
			setup: func(s *IntervalSolver[int64]) { s.AddUpperBoundary(math.MaxInt64) }},
	}

	rnd := rand.New(0)

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			solver := NewIntervalSolver[int64](math.MinInt64, math.MaxInt64)
			clone := solver.Clone()
			test.setup(clone)
			_, err := solver.Generate(rnd)
			if err != nil {
				t.Fatalf("error solving intervals: %v ", err)
			}
			if !solver.Equals(clone) {
				t.Fatalf("restricting beyond the current boundaries should not have any effect. want %v. got %v", solver, clone)
			}
		})
	}
}

func TestIntervalSolver_int64SmallEdges(t *testing.T) {

	tests := map[string]struct {
		setup func(*IntervalSolver[int64])
	}{
		"add-lower": {
			setup: func(s *IntervalSolver[int64]) { s.AddLowerBoundary(100) }},
		"add-upper": {
			setup: func(s *IntervalSolver[int64]) { s.AddUpperBoundary(500) }},
	}

	rnd := rand.New(0)

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			solver := NewIntervalSolver[int64](200, 400)
			clone := solver.Clone()
			test.setup(clone)
			_, err := solver.Generate(rnd)
			if err != nil {
				t.Fatalf("error solving intervals: %v ", err)
			}
			if !solver.Equals(clone) {
				t.Fatalf("restricting beyond the current boundaries should not have any effect. want %v. got %v", solver, clone)
			}
		})
	}
}

func TestIntervalSolver_allValuesAreGenerated(t *testing.T) {

	tests := map[string]struct {
		solver *IntervalSolver[int32]
		check  func(map[int32]int) bool
	}{
		"full-range": {
			solver: NewIntervalSolver[int32](20, 40),
			check: func(seen map[int32]int) bool {
				for i := 20; i <= 40; i++ {
					if seen[int32(i)] == 0 {
						return false
					}
				}
				return true
			},
		},
		"fragmented-range": {
			solver: func() *IntervalSolver[int32] {
				solver := NewIntervalSolver[int32](20, 40)
				solver.Exclude(25, 35)
				return solver
			}(),
			check: func(seen map[int32]int) bool {
				for i := 20; i <= 40; i++ {
					if i >= 25 && i <= 35 {
						if seen[int32(i)] != 0 {
							return false
						}
					} else {
						if seen[int32(i)] == 0 {

							return false
						}
					}
				}
				return true
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			seen := map[int32]int{}
			rnd := rand.New()
			for i := 0; i <= 100; i++ {
				res, err := test.solver.Generate(rnd)
				if err != nil {
					t.Fatalf("error solving intervals: %v", err)
				}
				seen[res] += 1
			}
			if !test.check(seen) {
				t.Fatalf("not all values were generated")
			}
		})
	}
}
