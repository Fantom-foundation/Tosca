package gen

import (
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
					if a > b || c > d {
						// we skip empty intervals
						continue
					}
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
		t.Errorf("failed to generate with empty interval: %v", err)
	}
}

func TestIntervalSolver_AddEmptyIntervalHasNoEffect(t *testing.T) {
	solver := NewIntervalSolver[int32](200, 400)
	clone := solver.Clone()
	clone.Exclude(300, 250)
	if !solver.Equals(clone) {
		t.Fatalf("empty interval should not be modified by adding empty interval")
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

			solver := NewIntervalSolver[int32](200, 400)
			solver.intervals = nil
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
		interval interval[int32]
		want     bool
	}{
		"empty":     {interval: interval[int32]{high: 0, low: 1}, want: true},
		"non-empty": {interval: interval[int32]{high: 1, low: 0}, want: false},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			got := test.interval.isEmpty()
			if got != test.want {
				t.Errorf("unexpected result, got %v, want %v", got, test.want)
			}
		})
	}
}
