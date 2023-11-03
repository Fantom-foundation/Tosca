package gen

import (
	"errors"
	"testing"

	"pgregory.net/rand"

	. "github.com/Fantom-foundation/Tosca/go/ct/common"
)

func TestCodeGenerator_UnconstrainedGeneratorCanProduceCode(t *testing.T) {
	rnd := rand.New(0)
	generator := NewCodeGenerator()
	if _, err := generator.Generate(rnd); err != nil {
		t.Fatalf("unexpected error during build: %v", err)
	}
}

func TestCodeGenerator_SetCodeSizeIsEnforced(t *testing.T) {
	sizes := []int{0, 1, 2, 1 << 20, 1 << 23}

	rnd := rand.New(0)
	for _, size := range sizes {
		generator := NewCodeGenerator()
		generator.SetSize(size)
		code, err := generator.Generate(rnd)
		if err != nil {
			t.Fatalf("unexpected error during build: %v", err)
		}
		if want, got := size, code.Length(); want != got {
			t.Errorf("unexpected code length, wanted %d, got %d", want, got)
		}
	}
}

func TestCodeGenerator_ConflictingSizesAreDetected(t *testing.T) {
	generator := NewCodeGenerator()
	generator.SetSize(12)
	generator.SetSize(14)
	rnd := rand.New(0)
	if _, err := generator.Generate(rnd); !errors.Is(err, ErrUnsatisfiable) {
		t.Errorf("unsatisfiable constraint not detected, got %v", err)
	}
}

func TestCodeGenerator_NegativeCodeSizesAreDetected(t *testing.T) {
	generator := NewCodeGenerator()
	generator.SetSize(-12)
	rnd := rand.New(0)
	if _, err := generator.Generate(rnd); !errors.Is(err, ErrUnsatisfiable) {
		t.Errorf("unsatisfiable constraint not detected, got %v", err)
	}
}

func TestCodeGenerator_NonConflictingSizesAreAccepted(t *testing.T) {
	generator := NewCodeGenerator()
	generator.SetSize(12)
	generator.SetSize(12)
	rnd := rand.New(0)
	if _, err := generator.Generate(rnd); err != nil {
		t.Errorf("generation failed: %v", err)
	}
}

func TestCodeGenerator_OperationConstraintsAreEnforced(t *testing.T) {
	tests := map[string][]struct {
		pos int
		op  OpCode
	}{
		"empty":            {},
		"single":           {{4, STOP}},
		"multiple-no-data": {{4, STOP}, {6, ADD}, {2, INVALID}},
		"pair":             {{4, PUSH1}, {7, PUSH32}},
		"tight":            {{0, PUSH1}, {2, PUSH1}, {4, PUSH1}},
		"wide":             {{2, PUSH1}, {20000, PUSH1}},
	}

	rnd := rand.New(0)
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			generator := NewCodeGenerator()

			for _, cur := range test {
				generator.SetOperation(cur.pos, cur.op)
			}

			code, err := generator.Generate(rnd)
			if err != nil {
				t.Fatalf("unexpected error during build: %v", err)
			}

			for _, cur := range test {
				if !code.IsCode(cur.pos) {
					t.Fatalf("position %d is not code", cur.pos)
				}
				if op, err := code.GetOperation(cur.pos); err != nil || op != cur.op {
					t.Errorf("failed to satisfy operator constraint for position %v, wanted %v, got %v, err %v", cur.pos, cur.op, op, err)
				}
			}
		})
	}
}

func TestCodeGenerator_ImpossibleConstraintsAreDetected(t *testing.T) {
	type op struct {
		pos int
		op  OpCode
	}
	tests := map[string]struct {
		size int
		ops  []op
	}{
		"too_small_code":                            {size: 2, ops: []op{{4, STOP}}},
		"just_too_small":                            {size: 4, ops: []op{{4, STOP}}},
		"conflicting_ops":                           {size: 4, ops: []op{{2, STOP}, {2, INVALID}}},
		"operation_in_short_data_begin":             {size: 4, ops: []op{{0, PUSH2}, {1, STOP}}},
		"operation_in_short_data_end":               {size: 4, ops: []op{{0, PUSH2}, {2, STOP}}},
		"operation_in_long_data_begin":              {size: 40, ops: []op{{0, PUSH32}, {1, STOP}}},
		"operation_in_long_data_mid":                {size: 40, ops: []op{{0, PUSH32}, {16, PUSH1}}},
		"operation_in_long_data_end":                {size: 40, ops: []op{{0, PUSH32}, {32, PUSH32}}},
		"add_operation_making_other_operation_data": {size: 40, ops: []op{{16, PUSH32}, {0, PUSH32}}},
	}

	rnd := rand.New(0)
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			generator := NewCodeGenerator()

			generator.SetSize(test.size)

			for _, cur := range test.ops {
				generator.SetOperation(cur.pos, cur.op)
			}

			if _, err := generator.Generate(rnd); !errors.Is(err, ErrUnsatisfiable) {
				t.Fatalf("expected error indicating unsatisfiability, but got %v", err)
			}
		})
	}
}

func TestCodeGenerator_CloneCopiesGeneratorState(t *testing.T) {
	original := NewCodeGenerator()
	original.SetSize(12)
	original.SetOperation(4, PUSH2)
	original.SetOperation(7, STOP)

	clone := original.Clone()

	if want, got := original.String(), clone.String(); want != got {
		t.Errorf("invalid clone, wanted %s, got %s", want, got)
	}
}

func TestCodeGenerator_ClonesAreIndependent(t *testing.T) {
	base := NewCodeGenerator()
	base.SetOperation(4, STOP)

	clone1 := base.Clone()
	clone1.SetSize(12)
	clone1.SetOperation(7, INVALID)

	clone2 := base.Clone()
	clone2.SetSize(14)
	clone2.SetOperation(7, PUSH2)

	want := "{size=12,op[4]=STOP,op[7]=INVALID}"
	if got := clone1.String(); got != want {
		t.Errorf("invalid clone, wanted %s, got %s", want, got)
	}

	want = "{size=14,op[4]=STOP,op[7]=PUSH2}"
	if got := clone2.String(); got != want {
		t.Errorf("invalid clone, wanted %s, got %s", want, got)
	}
}

func TestCodeGenerator_ClonesCanBeUsedToResetGenerator(t *testing.T) {
	generator := NewCodeGenerator()
	generator.SetOperation(4, STOP)

	backup := generator.Clone()

	generator.SetSize(12)
	generator.SetOperation(7, INVALID)
	want := "{size=12,op[4]=STOP,op[7]=INVALID}"
	if got := generator.String(); got != want {
		t.Errorf("unexpected generator state, wanted %s, got %s", want, got)
	}

	generator.Restore(backup)
	want = "{op[4]=STOP}"
	if got := generator.String(); got != want {
		t.Errorf("invalid clone, wanted %s, got %s", want, got)
	}
}
