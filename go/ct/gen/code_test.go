package gen

import (
	"errors"
	"testing"

	"github.com/Fantom-foundation/Tosca/go/ct/st"
	"pgregory.net/rand"
)

func TestCodeGenerator_UnconstrainedBuilderCanProduceCode(t *testing.T) {
	rnd := rand.New()
	generator := NewCodeGenerator()
	if _, err := generator.Generate(rnd); err != nil {
		t.Fatalf("unexpected error during build: %v", err)
	}
}

func TestCodeGenerator_SetCodeSizeIsEnforced(t *testing.T) {
	sizes := []int{0, 1, 2, 1 << 20, 1 << 23}

	rnd := rand.New()
	for _, size := range sizes {
		generator := NewCodeGenerator()
		if err := generator.SetSize(size); err != nil {
			t.Fatalf("unexpected error setting size: %v", err)
		}
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
	if err := generator.SetSize(12); err != nil {
		t.Fatalf("error setting code size: %v", err)
	}
	if err := generator.SetSize(14); err != nil {
		t.Fatalf("error setting code size: %v", err)
	}
	rnd := rand.New()
	if _, err := generator.Generate(rnd); !errors.Is(err, ErrUnsatisfiable) {
		t.Errorf("unsatisfiable constraint not detected, got %v", err)
	}
}

func TestCodeGenerator_NegativeCodeSizesAreDetected(t *testing.T) {
	generator := NewCodeGenerator()
	if err := generator.SetSize(-12); err != nil {
		t.Fatalf("error setting code size: %v", err)
	}
	rnd := rand.New()
	if _, err := generator.Generate(rnd); !errors.Is(err, ErrUnsatisfiable) {
		t.Errorf("unsatisfiable constraint not detected, got %v", err)
	}
}

func TestCodeGenerator_NonConflictingSizesAreAccepted(t *testing.T) {
	generator := NewCodeGenerator()
	if err := generator.SetSize(12); err != nil {
		t.Fatalf("error setting code size: %v", err)
	}
	if err := generator.SetSize(12); err != nil {
		t.Fatalf("error setting code size: %v", err)
	}
	rnd := rand.New()
	if _, err := generator.Generate(rnd); err != nil {
		t.Errorf("generation failed: %v", err)
	}
}

func TestCodeGenerator_OperationConstraintsAreEnforced(t *testing.T) {
	tests := map[string][]struct {
		pos int
		op  st.OpCode
	}{
		"empty":            {},
		"single":           {{4, st.STOP}},
		"multiple-no-data": {{4, st.STOP}, {6, st.ADD}, {2, st.INVALID}},
		"pair":             {{4, st.PUSH1}, {7, st.PUSH32}},
		"tight":            {{0, st.PUSH1}, {2, st.PUSH1}, {4, st.PUSH1}},
		"wide":             {{2, st.PUSH1}, {20000, st.PUSH1}},
	}

	rnd := rand.New()
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			generator := NewCodeGenerator()

			for _, cur := range test {
				if err := generator.SetOperation(cur.pos, cur.op); err != nil {
					t.Fatalf("failed to add operation constraint: %v", err)
				}
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
		op  st.OpCode
	}
	tests := map[string]struct {
		size int
		ops  []op
	}{
		"too_small_code":                            {size: 2, ops: []op{{4, st.STOP}}},
		"just_too_small":                            {size: 4, ops: []op{{4, st.STOP}}},
		"conflicting_ops":                           {size: 4, ops: []op{{2, st.STOP}, {2, st.INVALID}}},
		"operation_in_short_data_begin":             {size: 4, ops: []op{{0, st.PUSH2}, {1, st.STOP}}},
		"operation_in_short_data_end":               {size: 4, ops: []op{{0, st.PUSH2}, {2, st.STOP}}},
		"operation_in_long_data_begin":              {size: 40, ops: []op{{0, st.PUSH32}, {1, st.STOP}}},
		"operation_in_long_data_mid":                {size: 40, ops: []op{{0, st.PUSH32}, {16, st.PUSH1}}},
		"operation_in_long_data_end":                {size: 40, ops: []op{{0, st.PUSH32}, {32, st.PUSH32}}},
		"add_operation_making_other_operation_data": {size: 40, ops: []op{{16, st.PUSH32}, {0, st.PUSH32}}},
	}

	rnd := rand.New()
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			generator := NewCodeGenerator()

			if err := generator.SetSize(test.size); err != nil {
				t.Fatalf("unexpected error setting code size: %v", err)
			}

			for _, cur := range test.ops {
				if err := generator.SetOperation(cur.pos, cur.op); err != nil {
					t.Fatalf("unexpected error setting operation constraint: %v", err)
				}
			}

			if _, err := generator.Generate(rnd); !errors.Is(err, ErrUnsatisfiable) {
				t.Fatalf("expected error indicating unsatisfiability, but got %v", err)
			}
		})
	}
}

func TestCodeGenerator_CloneCopiesBuilderState(t *testing.T) {
	original := NewCodeGenerator()
	original.SetSize(12)
	original.SetOperation(4, st.PUSH2)
	original.SetOperation(7, st.STOP)

	clone := original.Clone()

	if want, got := original.String(), clone.String(); want != got {
		t.Errorf("invalid clone, wanted %s, got %s", want, got)
	}
}

func TestCodeGenerator_ClonesAreIndependent(t *testing.T) {
	base := NewCodeGenerator()
	base.SetOperation(4, st.STOP)

	clone1 := base.Clone()
	clone1.SetSize(12)
	clone1.SetOperation(7, st.INVALID)

	clone2 := base.Clone()
	clone2.SetSize(14)
	clone2.SetOperation(7, st.PUSH2)

	want := "{size=12,op[4]=STOP,op[7]=INVALID}"
	if got := clone1.String(); got != want {
		t.Errorf("invalid clone, wanted %s, got %s", want, got)
	}

	want = "{size=14,op[4]=STOP,op[7]=PUSH2}"
	if got := clone2.String(); got != want {
		t.Errorf("invalid clone, wanted %s, got %s", want, got)
	}
}

func TestCodeGenerator_ClonesCanBeUsedToResetBuilder(t *testing.T) {
	generator := NewCodeGenerator()
	generator.SetOperation(4, st.STOP)

	backup := generator.Clone()

	generator.SetSize(12)
	generator.SetOperation(7, st.INVALID)
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
