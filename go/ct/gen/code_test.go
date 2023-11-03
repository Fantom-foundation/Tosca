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
		ops []op
	}{
		"conflicting_ops":                           {ops: []op{{2, STOP}, {2, INVALID}}},
		"operation_in_short_data_begin":             {ops: []op{{0, PUSH2}, {1, STOP}}},
		"operation_in_short_data_end":               {ops: []op{{0, PUSH2}, {2, STOP}}},
		"operation_in_long_data_begin":              {ops: []op{{0, PUSH32}, {1, STOP}}},
		"operation_in_long_data_mid":                {ops: []op{{0, PUSH32}, {16, PUSH1}}},
		"operation_in_long_data_end":                {ops: []op{{0, PUSH32}, {32, PUSH32}}},
		"add_operation_making_other_operation_data": {ops: []op{{16, PUSH32}, {0, PUSH32}}},
	}

	rnd := rand.New(0)
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			generator := NewCodeGenerator()

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
