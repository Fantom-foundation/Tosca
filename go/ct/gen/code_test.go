package gen

import (
	"errors"
	"testing"

	"github.com/Fantom-foundation/Tosca/go/ct/st"
	"pgregory.net/rand"
)

func TestCodeGenerator_UnconstrainedGeneratorCanProduceCode(t *testing.T) {
	rnd := rand.New(0)
	generator := NewCodeGenerator()
	if _, err := generator.Generate(nil, rnd); err != nil {
		t.Fatalf("unexpected error during build: %v", err)
	}
}

func TestCodeGenerator_SetCodeSizeIsEnforced(t *testing.T) {
	sizes := []int{0, 1, 2, 1 << 20, 1 << 23}

	rnd := rand.New(0)
	for _, size := range sizes {
		generator := NewCodeGenerator()
		generator.SetSize(size)
		code, err := generator.Generate(nil, rnd)
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
	if _, err := generator.Generate(nil, rnd); !errors.Is(err, ErrUnsatisfiable) {
		t.Errorf("unsatisfiable constraint not detected, got %v", err)
	}
}

func TestCodeGenerator_NegativeCodeSizesAreDetected(t *testing.T) {
	generator := NewCodeGenerator()
	generator.SetSize(-12)
	rnd := rand.New(0)
	if _, err := generator.Generate(nil, rnd); !errors.Is(err, ErrUnsatisfiable) {
		t.Errorf("unsatisfiable constraint not detected, got %v", err)
	}
}

func TestCodeGenerator_NonConflictingSizesAreAccepted(t *testing.T) {
	generator := NewCodeGenerator()
	generator.SetSize(12)
	generator.SetSize(12)
	rnd := rand.New(0)
	if _, err := generator.Generate(nil, rnd); err != nil {
		t.Errorf("generation failed: %v", err)
	}
}

func TestCodeGenerator_ConflictingOperationsAreDetected(t *testing.T) {
	generator := NewCodeGenerator()
	generator.SetOperation(12, st.ADD)
	generator.SetOperation(12, st.JUMP)
	rnd := rand.New(0)
	if _, err := generator.Generate(nil, rnd); !errors.Is(err, ErrUnsatisfiable) {
		t.Errorf("unsatisfiable constraint not detected, got %v", err)
	}
}

func TestCodeGenerator_VariablesAreSupported(t *testing.T) {
	constraints := []struct {
		variable  Variable
		operation st.OpCode
	}{
		{Variable("A"), st.ADD},
		{Variable("B"), st.JUMP},
		{Variable("C"), st.PUSH2},
	}

	generator := NewCodeGenerator()
	generator.SetSize(10)
	for _, cur := range constraints {
		generator.AddOperation(cur.variable, cur.operation)
	}

	assignment := Assignment{}
	rnd := rand.New(0)
	code, err := generator.Generate(assignment, rnd)
	if err != nil {
		t.Fatalf("generation failed: %v", err)
	}

	for _, cur := range constraints {
		pos, found := assignment[cur.variable]
		if !found {
			t.Fatalf("free variable %v not bound by generator", cur.variable)
		}
		if !pos.IsUint64() || pos.Uint64() > uint64(code.Length()) {
			t.Fatalf("invalid value for code position: %v, code size is %d", pos, code.Length())
		}
		if op, err := code.GetOperation(int(pos.Uint64())); err != nil || op != cur.operation {
			t.Errorf("unsatisfied constraint, wanted %v, got %v, err %v", cur.operation, op, err)
		}
	}
}

func TestCodeGenerator_ConflictingVariablesAreDetected(t *testing.T) {
	generator := NewCodeGenerator()
	generator.AddOperation(Variable("X"), st.ADD)
	generator.AddOperation(Variable("X"), st.JUMP)
	rnd := rand.New(0)
	if _, err := generator.Generate(nil, rnd); !errors.Is(err, ErrUnsatisfiable) {
		t.Errorf("unsatisfiable constraint not detected, got %v", err)
	}
}

func TestCodeGenerator_OperationConstraintsAreEnforced(t *testing.T) {
	tests := map[string][]struct {
		p  int
		v  string
		op st.OpCode
	}{
		"empty":            {},
		"single":           {{p: 4, op: st.STOP}},
		"multiple-no-data": {{p: 4, op: st.STOP}, {p: 6, op: st.ADD}, {p: 2, op: st.INVALID}},
		"pair":             {{p: 4, op: st.PUSH1}, {p: 7, op: st.PUSH32}},
		"tight":            {{p: 0, op: st.PUSH1}, {p: 2, op: st.PUSH1}, {p: 4, op: st.PUSH1}},
		"wide":             {{p: 2, op: st.PUSH1}, {p: 20000, op: st.PUSH1}},
		"single-var":       {{v: "A", op: st.STOP}},
		"multi-var":        {{v: "A", op: st.STOP}, {v: "B", op: st.ADD}},
		"const-var-mix":    {{p: 5, op: st.STOP}, {v: "A", op: st.ADD}},
	}

	rnd := rand.New(0)
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			generator := NewCodeGenerator()

			for _, cur := range test {
				if len(cur.v) == 0 {
					generator.SetOperation(cur.p, cur.op)
				} else {
					generator.AddOperation(Variable(cur.v), cur.op)
				}
			}

			assignment := Assignment{}
			code, err := generator.Generate(assignment, rnd)
			if err != nil {
				t.Fatalf("unexpected error during build: %v", err)
			}

			for _, cur := range test {
				pos := cur.p
				if len(cur.v) > 0 {
					selectedPosition, found := assignment[Variable(cur.v)]
					if !found || !selectedPosition.IsUint64() {
						t.Fatalf("failed to bind variable %v to valid value: %v, found %t", cur.v, selectedPosition, found)
					}
					pos = int(selectedPosition.Uint64())
				}
				if !code.IsCode(pos) {
					t.Fatalf("position %d is not code", pos)
				}
				if op, err := code.GetOperation(pos); err != nil || op != cur.op {
					t.Errorf("failed to satisfy operator constraint for position %v, wanted %v, got %v, err %v", pos, cur.op, op, err)
				}
			}
		})
	}
}

func TestCodeGenerator_ImpossibleConstraintsAreDetected(t *testing.T) {
	type op struct {
		p  int
		v  string
		op st.OpCode
	}
	tests := map[string]struct {
		size int
		ops  []op
	}{
		"too_small_code":                            {size: 2, ops: []op{{p: 4, op: st.STOP}}},
		"just_too_small":                            {size: 4, ops: []op{{p: 4, op: st.STOP}}},
		"conflicting_ops":                           {size: 4, ops: []op{{p: 2, op: st.STOP}, {p: 2, op: st.INVALID}}},
		"operation_in_short_data_begin":             {size: 4, ops: []op{{p: 0, op: st.PUSH2}, {p: 1, op: st.STOP}}},
		"operation_in_short_data_end":               {size: 4, ops: []op{{p: 0, op: st.PUSH2}, {p: 2, op: st.STOP}}},
		"operation_in_long_data_begin":              {size: 40, ops: []op{{p: 0, op: st.PUSH32}, {p: 1, op: st.STOP}}},
		"operation_in_long_data_mid":                {size: 40, ops: []op{{p: 0, op: st.PUSH32}, {p: 16, op: st.PUSH1}}},
		"operation_in_long_data_end":                {size: 40, ops: []op{{p: 0, op: st.PUSH32}, {p: 32, op: st.PUSH32}}},
		"add_operation_making_other_operation_data": {size: 40, ops: []op{{p: 16, op: st.PUSH32}, {p: 0, op: st.PUSH32}}},
		"too_small_code_with_variables":             {size: 2, ops: []op{{v: "A", op: st.STOP}, {v: "B", op: st.ADD}, {v: "C", op: st.JUMP}}},
		"too_fragmented":                            {size: 15, ops: []op{{p: 5, op: st.PUSH32}, {v: "A", op: st.PUSH32}}},
	}

	rnd := rand.New(0)
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			generator := NewCodeGenerator()

			generator.SetSize(test.size)

			for _, cur := range test.ops {
				if len(cur.v) > 0 {
					generator.AddOperation(Variable(cur.v), cur.op)
				} else {
					generator.SetOperation(cur.p, cur.op)
				}
			}

			if _, err := generator.Generate(nil, rnd); !errors.Is(err, ErrUnsatisfiable) {
				t.Fatalf("expected error indicating unsatisfiability, but got %v", err)
			}
		})
	}
}

func TestCodeGenerator_CloneCopiesGeneratorState(t *testing.T) {
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

func TestCodeGenerator_ClonesCanBeUsedToResetGenerator(t *testing.T) {
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
