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
	if _, err := generator.Generate(nil, rnd); err != nil {
		t.Fatalf("unexpected error during build: %v", err)
	}
}

func TestCodeGenerator_ConflictingOperationsAreDetected(t *testing.T) {
	generator := NewCodeGenerator()
	generator.SetOperation(12, ADD)
	generator.SetOperation(12, JUMP)
	rnd := rand.New(0)
	if _, err := generator.Generate(nil, rnd); !errors.Is(err, ErrUnsatisfiable) {
		t.Errorf("unsatisfiable constraint not detected, got %v", err)
	}
}

func TestCodeGenerator_VariablesAreSupported(t *testing.T) {
	constraints := []struct {
		variable  Variable
		operation OpCode
	}{
		{Variable("A"), ADD},
		{Variable("B"), JUMP},
		{Variable("C"), PUSH2},
	}

	generator := NewCodeGenerator()
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
	generator.AddOperation(Variable("X"), ADD)
	generator.AddOperation(Variable("X"), JUMP)
	rnd := rand.New(0)
	if _, err := generator.Generate(nil, rnd); !errors.Is(err, ErrUnsatisfiable) {
		t.Errorf("unsatisfiable constraint not detected, got %v", err)
	}
}

func TestCodeGenerator_OperationConstraintsAreEnforced(t *testing.T) {
	tests := map[string][]struct {
		p  int
		v  string
		op OpCode
	}{
		"single":           {{p: 4, op: STOP}},
		"multiple-no-data": {{p: 4, op: STOP}, {p: 6, op: ADD}, {p: 2, op: INVALID}},
		"pair":             {{p: 4, op: PUSH1}, {p: 7, op: PUSH32}},
		"tight":            {{p: 0, op: PUSH1}, {p: 2, op: PUSH1}, {p: 4, op: PUSH1}},
		// "wide":             {{p: 2, op: PUSH1}, {p: 20000, op: PUSH1}}, // TODO re-enable when max code size is restored
		"single-var":    {{v: "A", op: STOP}},
		"multi-var":     {{v: "A", op: STOP}, {v: "B", op: ADD}},
		"const-var-mix": {{p: 5, op: STOP}, {v: "A", op: ADD}},
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
		op OpCode
	}
	tests := map[string]struct {
		ops []op
	}{
		"conflicting_ops":                           {ops: []op{{p: 2, op: STOP}, {p: 2, op: INVALID}}},
		"operation_in_short_data_begin":             {ops: []op{{p: 0, op: PUSH2}, {p: 1, op: STOP}}},
		"operation_in_short_data_end":               {ops: []op{{p: 0, op: PUSH2}, {p: 2, op: STOP}}},
		"operation_in_long_data_begin":              {ops: []op{{p: 0, op: PUSH32}, {p: 1, op: STOP}}},
		"operation_in_long_data_mid":                {ops: []op{{p: 0, op: PUSH32}, {p: 16, op: PUSH1}}},
		"operation_in_long_data_end":                {ops: []op{{p: 0, op: PUSH32}, {p: 32, op: PUSH32}}},
		"add_operation_making_other_operation_data": {ops: []op{{p: 16, op: PUSH32}, {p: 0, op: PUSH32}}},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			generator := NewCodeGenerator()

			for _, cur := range test.ops {
				generator.SetOperation(cur.p, cur.op)
			}

			if _, err := generator.Generate(nil, rand.New(0)); !errors.Is(err, ErrUnsatisfiable) {
				t.Fatalf("expected error indicating unsatisfiability, but got %v", err)
			}
		})
	}
}

func TestCodeGenerator_CloneCopiesGeneratorState(t *testing.T) {
	original := NewCodeGenerator()
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
	clone1.SetOperation(7, INVALID)

	clone2 := base.Clone()
	clone2.SetOperation(7, PUSH2)

	want := "{op[4]=STOP,op[7]=INVALID}"
	if got := clone1.String(); got != want {
		t.Errorf("invalid clone, wanted %s, got %s", want, got)
	}

	want = "{op[4]=STOP,op[7]=PUSH2}"
	if got := clone2.String(); got != want {
		t.Errorf("invalid clone, wanted %s, got %s", want, got)
	}
}

func TestCodeGenerator_ClonesCanBeUsedToResetGenerator(t *testing.T) {
	generator := NewCodeGenerator()
	generator.SetOperation(4, STOP)

	backup := generator.Clone()

	generator.SetOperation(7, INVALID)
	want := "{op[4]=STOP,op[7]=INVALID}"
	if got := generator.String(); got != want {
		t.Errorf("unexpected generator state, wanted %s, got %s", want, got)
	}

	generator.Restore(backup)
	want = "{op[4]=STOP}"
	if got := generator.String(); got != want {
		t.Errorf("invalid clone, wanted %s, got %s", want, got)
	}
}
