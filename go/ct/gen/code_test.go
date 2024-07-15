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
	"errors"
	"fmt"
	"testing"

	"pgregory.net/rand"

	. "github.com/Fantom-foundation/Tosca/go/ct/common"
	"github.com/Fantom-foundation/Tosca/go/tosca/vm"
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
	generator.SetOperation(12, vm.ADD)
	generator.SetOperation(12, vm.JUMP)
	rnd := rand.New(0)
	if _, err := generator.Generate(nil, rnd); !errors.Is(err, ErrUnsatisfiable) {
		t.Errorf("unsatisfiable constraint not detected, got %v", err)
	}
}

func TestCodeGenerator_VariablesAreSupported(t *testing.T) {
	constraints := []struct {
		variable  Variable
		operation vm.OpCode
	}{
		{Variable("A"), vm.ADD},
		{Variable("B"), vm.JUMP},
		{Variable("C"), vm.PUSH2},
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

func TestCodeGenerator_PreDefinedVariablesAreAccepted(t *testing.T) {
	constraints := []struct {
		variable  Variable
		operation vm.OpCode
	}{
		{Variable("A"), vm.ADD},
		{Variable("B"), vm.JUMP},
		{Variable("C"), vm.PUSH2},
	}

	generator := NewCodeGenerator()
	for _, cur := range constraints {
		generator.AddOperation(cur.variable, cur.operation)
	}

	assignment := Assignment{}
	for i, cur := range constraints {
		assignment[cur.variable] = NewU256(uint64(i))
	}

	rnd := rand.New(0)
	code, err := generator.Generate(assignment, rnd)
	if err != nil {
		t.Fatalf("generation failed: %v", err)
	}

	for i, cur := range constraints {
		pos, found := assignment[cur.variable]
		if !found {
			t.Fatalf("pre-bound variable freed by generator: %v", cur.variable)
		}
		if want, got := NewU256(uint64(i)), pos; want != got {
			t.Errorf("variable bound to wrong value, wanted %d, got %d", want, got)
		}
		if op, err := code.GetOperation(int(i)); err != nil || op != cur.operation {
			t.Errorf("unsatisfied constraint, wanted %v, got %v, err %v", cur.operation, op, err)
		}
	}
}

func TestCodeGenerator_ConflictInPredefinedVariablesIsDetected(t *testing.T) {
	opCode := vm.PUSH4 // < the op-code to be used for all variables
	tests := map[string]map[Variable]uint64{
		"position_collision": {
			Variable("A"): 12,
			Variable("B"): 12,
		},
		"in_the_data_of_another_instruction": {
			Variable("A"): 12,
			Variable("B"): 14,
		},
		"at_the_end_of_the_data_of_another_instruction": {
			Variable("A"): 12,
			Variable("B"): 16,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {

			assignment := Assignment{}
			for v, p := range test {
				assignment[v] = NewU256(p)
			}

			generator := NewCodeGenerator()
			for v := range test {
				generator.AddOperation(v, opCode)
			}

			rnd := rand.New(0)
			_, err := generator.Generate(assignment, rnd)
			if !errors.Is(err, ErrUnsatisfiable) {
				t.Errorf("failed to detect unsatisfiability in %v: %v", generator, err)
			}
		})
	}
}

func TestCodeGenerator_ConflictingVariablesAreDetected(t *testing.T) {
	generator := NewCodeGenerator()
	generator.AddOperation(Variable("X"), vm.ADD)
	generator.AddOperation(Variable("X"), vm.JUMP)
	rnd := rand.New(0)
	if _, err := generator.Generate(nil, rnd); !errors.Is(err, ErrUnsatisfiable) {
		t.Errorf("unsatisfiable constraint not detected, got %v", err)
	}
}

func TestCodeGenerator_IsCodeConstraint(t *testing.T) {
	variable := Variable("X")

	generator := NewCodeGenerator()
	generator.AddIsCode(variable)

	assignment := Assignment{}

	state, err := generator.Generate(assignment, rand.New(0))
	if err != nil {
		t.Error(err)
	}

	pos, found := assignment[variable]
	if !found {
		t.Errorf("free variable %v not bound by generator", variable)
	}

	if !state.IsCode(int(pos.Uint64())) {
		t.Error("IsCode constraint not satisfied")
	}
}

func TestCodeGenerator_IsDataConstraint(t *testing.T) {
	variable := Variable("X")

	generator := NewCodeGenerator()
	generator.AddIsData(variable)

	assignment := Assignment{}

	state, err := generator.Generate(assignment, rand.New(0))
	if err != nil {
		t.Error(err)
	}

	pos, found := assignment[variable]
	if !found {
		t.Errorf("free variable %v not bound by generator", variable)
	}

	if !state.IsData(int(pos.Uint64())) {
		t.Error("IsCode constraint not satisfied")
	}
}

func TestCodeGenerator_OperationConstraintsAreEnforced(t *testing.T) {
	tests := map[string][]struct {
		p  int
		v  string
		op vm.OpCode
	}{
		"single":           {{p: 4, op: vm.STOP}},
		"multiple-no-data": {{p: 4, op: vm.STOP}, {p: 6, op: vm.ADD}, {p: 2, op: vm.INVALID}},
		"pair":             {{p: 4, op: vm.PUSH1}, {p: 7, op: vm.PUSH32}},
		"tight":            {{p: 0, op: vm.PUSH1}, {p: 2, op: vm.PUSH1}, {p: 4, op: vm.PUSH1}},
		"wide":             {{p: 2, op: vm.PUSH1}, {p: 20000, op: vm.PUSH1}},
		"single-var":       {{v: "A", op: vm.STOP}},
		"multi-var":        {{v: "A", op: vm.STOP}, {v: "B", op: vm.ADD}},
		"const-var-mix":    {{p: 5, op: vm.STOP}, {v: "A", op: vm.ADD}},
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
		op vm.OpCode
	}
	tests := map[string]struct {
		ops []op
	}{
		"conflicting_ops":                           {ops: []op{{p: 2, op: vm.STOP}, {p: 2, op: vm.INVALID}}},
		"operation_in_short_data_begin":             {ops: []op{{p: 0, op: vm.PUSH2}, {p: 1, op: vm.STOP}}},
		"operation_in_short_data_end":               {ops: []op{{p: 0, op: vm.PUSH2}, {p: 2, op: vm.STOP}}},
		"operation_in_long_data_begin":              {ops: []op{{p: 0, op: vm.PUSH32}, {p: 1, op: vm.STOP}}},
		"operation_in_long_data_mid":                {ops: []op{{p: 0, op: vm.PUSH32}, {p: 16, op: vm.PUSH1}}},
		"operation_in_long_data_end":                {ops: []op{{p: 0, op: vm.PUSH32}, {p: 32, op: vm.PUSH32}}},
		"add_operation_making_other_operation_data": {ops: []op{{p: 16, op: vm.PUSH32}, {p: 0, op: vm.PUSH32}}},
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
	original.SetOperation(4, vm.PUSH2)
	original.SetOperation(7, vm.STOP)
	original.AddIsCode(Variable("X"))
	original.AddIsData(Variable("Y"))

	clone := original.Clone()

	if want, got := original.String(), clone.String(); want != got {
		t.Errorf("invalid clone, wanted %s, got %s", want, got)
	}
}

func TestCodeGenerator_ClonesAreIndependent(t *testing.T) {
	base := NewCodeGenerator()
	base.SetOperation(4, vm.STOP)

	clone1 := base.Clone()
	clone1.SetOperation(7, vm.INVALID)
	clone1.AddIsCode(Variable("X"))

	clone2 := base.Clone()
	clone2.SetOperation(7, vm.PUSH2)
	clone2.AddIsData(Variable("Y"))

	want := "{op[4]=STOP,op[7]=INVALID,isCode[$X]}"
	if got := clone1.String(); got != want {
		t.Errorf("invalid clone, wanted %s, got %s", want, got)
	}

	want = "{op[4]=STOP,op[7]=PUSH2,isData[$Y]}"
	if got := clone2.String(); got != want {
		t.Errorf("invalid clone, wanted %s, got %s", want, got)
	}
}

func TestCodeGenerator_ClonesCanBeUsedToResetGenerator(t *testing.T) {
	generator := NewCodeGenerator()
	generator.SetOperation(4, vm.STOP)

	backup := generator.Clone()

	generator.SetOperation(7, vm.INVALID)
	generator.AddIsCode(Variable("X"))
	want := "{op[4]=STOP,op[7]=INVALID,isCode[$X]}"
	if got := generator.String(); got != want {
		t.Errorf("unexpected generator state, wanted %s, got %s", want, got)
	}

	generator.Restore(backup)
	want = "{op[4]=STOP}"
	if got := generator.String(); got != want {
		t.Errorf("invalid clone, wanted %s, got %s", want, got)
	}
}

func TestCodeGenerator_TooSmallCodeSizeLeadsToUnsatisfiableResult(t *testing.T) {
	fixSize := func(g *CodeGenerator, size int) {
		g.codeSize = new(int)
		*g.codeSize = size
	}

	tests := map[string]func(*CodeGenerator){
		"empty code with variable constraint": func(g *CodeGenerator) {
			fixSize(g, 0)
			g.AddOperation(Variable("X"), vm.STOP)
		},
		"must contain code with size 0": func(g *CodeGenerator) {
			fixSize(g, 0)
			g.AddIsCode(Variable("X"))
		},
		"two variable ops with size of 1": func(g *CodeGenerator) {
			fixSize(g, 1)
			g.AddOperation(Variable("X"), vm.STOP)
			g.AddOperation(Variable("Y"), vm.ADD)
		},
		"two constant ops with size 1 ": func(g *CodeGenerator) {
			fixSize(g, 1)
			g.SetOperation(1, vm.STOP)
			g.SetOperation(2, vm.ADD)
		},
		"two mix ops with size 1 ": func(g *CodeGenerator) {
			fixSize(g, 1)
			g.SetOperation(1, vm.STOP)
			g.AddOperation(Variable("Y"), vm.ADD)
		},
	}

	for name, setup := range tests {
		t.Run(name, func(t *testing.T) {
			generator := NewCodeGenerator()
			setup(generator)
			if _, err := generator.Generate(nil, rand.New(0)); !errors.Is(err, ErrUnsatisfiable) {
				t.Errorf("expected unsatisfiable result, but got %v", err)
			}
		})
	}
}

func TestCodeGenerator_CodeIsLargeEnoughForAllConditionOps(t *testing.T) {

	// constantOp is an location which has a pre-assigned Op
	type constantOp struct {
		location int
		op       vm.OpCode
	}

	// variableOp is a solver variable bound to an Op
	type variableOp struct {
		variable string
		op       vm.OpCode
	}

	tests := map[string]struct {
		size         int
		constantOps  []constantOp
		variableOps  []variableOp
		containsCode bool
	}{
		"Empty code": {
			size: 0,
		},
		"Code size 1": {
			size: 1,
		},
		"Code must contain one instruction": {
			size:         1,
			containsCode: true,
		},
		"Single constantOp": {
			size: 1,
			constantOps: []constantOp{
				{location: 0, op: vm.STOP},
			},
		},
		"Multiple constantOps": {
			size: 3,
			constantOps: []constantOp{
				{location: 0, op: vm.STOP},
				{location: 1, op: vm.ADDMOD},
				{location: 2, op: vm.BALANCE},
			},
		},
		"Multiple constantOps with gaps": {
			size: 9,
			constantOps: []constantOp{
				{location: 0, op: vm.STOP},
				{location: 4, op: vm.ADDMOD},
				{location: 8, op: vm.BALANCE},
			},
		},
		"Multiple constantOps with containsCode": {
			size: 3,
			constantOps: []constantOp{
				{location: 0, op: vm.STOP},
				{location: 1, op: vm.ADDMOD},
				{location: 2, op: vm.BALANCE},
			},
			containsCode: true,
		},
		"Multiple variableOps": {
			size: 3,
			variableOps: []variableOp{
				{variable: "a", op: vm.STOP},
				{variable: "b", op: vm.ADDMOD},
				{variable: "c", op: vm.BALANCE},
			},
		},
		"Multiple variableOps with identical operations": {
			size: 3, // < this could be 2, but the solver fails on that (which it should not)
			variableOps: []variableOp{
				{variable: "a", op: vm.STOP},
				{variable: "b", op: vm.ADDMOD},
				{variable: "c", op: vm.STOP},
			},
		},
		"Multiple constantOps and variableOps": {
			size: 6,
			constantOps: []constantOp{
				{location: 0, op: vm.STOP},
				{location: 1, op: vm.ADDMOD},
				{location: 2, op: vm.BALANCE},
			},
			variableOps: []variableOp{
				{variable: "a", op: vm.ADD},
				{variable: "b", op: vm.BASEFEE},
				{variable: "c", op: vm.BYTE},
			},
		},
		"Multiple constantOps and variableOps with containsCode": {
			size: 6,
			constantOps: []constantOp{
				{location: 0, op: vm.STOP},
				{location: 1, op: vm.ADDMOD},
				{location: 2, op: vm.BALANCE},
			},
			variableOps: []variableOp{
				{variable: "a", op: vm.ADD},
				{variable: "b", op: vm.BASEFEE},
				{variable: "c", op: vm.BYTE},
			},
			containsCode: true,
		},
		// This is a challenging one, right now not supported.
		/*
			"Multiple constantOps and variableOps with overlaps": {
				size: 4,
				constantOps: []constantOp{
					{p: 0, op: vm.STOP},
					{p: 1, op: vm.ADDMOD},
					{p: 2, op: vm.BALANCE},
				},
				variableOps: []varOp{
					{v: "a", op: vm.STOP},
					{v: "b", op: vm.ADDMOD},
					{v: "c", op: vm.BYTE},
				},
			},
		*/
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			generator := NewCodeGenerator()
			generator.codeSize = &test.size
			if test.containsCode {
				generator.AddIsCode(Variable("X"))
			}
			for _, op := range test.constantOps {
				generator.SetOperation(op.location, op.op)
			}
			for _, op := range test.variableOps {
				generator.AddOperation(Variable(op.variable), op.op)
			}

			code, err := generator.Generate(Assignment{}, rand.New(0))

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if code.Length() != test.size {
				t.Errorf("unexpected code length: wanted %d, got %d", test.size, code.Length())
			}
		})
	}
}

func TestVarCodeConstraintSolver_fitsOnEmpty(t *testing.T) {
	tests := []struct {
		pos  int
		op   vm.OpCode
		fits bool
	}{
		{0, vm.JUMP, true},
		{1, vm.JUMP, true},
		{2, vm.JUMP, true},
		{3, vm.JUMP, false},
		{4, vm.JUMP, false},
		{0, vm.PUSH1, true},
		{1, vm.PUSH1, true},
		{2, vm.PUSH1, false},
		{0, vm.PUSH2, true},
		{1, vm.PUSH2, false},
		{2, vm.PUSH2, false},
		{0, vm.PUSH3, false},
		{1, vm.PUSH3, false},
	}
	for _, test := range tests {
		solver := newVarCodeConstraintSolver(3, nil, nil, nil)
		if want, got := test.fits, solver.fits(test.pos, test.op); want != got {
			t.Fatalf("incorrect fit want %v, got %v", want, got)
		}
	}
}

func TestVarCodeConstraintSolver_fitsOnUsed(t *testing.T) {
	tests := []struct {
		pos  int
		op   vm.OpCode
		fits bool
	}{
		{0, vm.JUMP, true},
		{1, vm.JUMP, true},
		{2, vm.JUMP, true},
		{3, vm.JUMP, false},
		{4, vm.JUMP, false},
		{0, vm.PUSH1, true},
		{1, vm.PUSH1, true},
		{2, vm.PUSH1, false},
		{0, vm.PUSH2, true},
		{1, vm.PUSH2, false},
		{2, vm.PUSH2, false},
		{0, vm.PUSH3, false},
		{1, vm.PUSH3, false},
	}
	for _, test := range tests {
		solver := newVarCodeConstraintSolver(4, nil, nil, nil)
		solver.markUsed(3, vm.JUMPDEST)
		if want, got := test.fits, solver.fits(test.pos, test.op); want != got {
			t.Fatalf("incorrect fit want %v, got %v", want, got)
		}
	}
}

func TestCodeGenerator_IsDataConstraintSmallSize(t *testing.T) {
	variable := Variable("X")

	tests := []int{0, 1, 2}

	for size := range tests {
		t.Run(fmt.Sprintf("size: %v", size), func(t *testing.T) {

			generator := NewCodeGenerator()
			generator.AddIsData(variable)
			generator.codeSize = new(int)
			*generator.codeSize = size

			assignment := Assignment{}

			state, err := generator.Generate(assignment, rand.New(0))
			if size < 2 {
				if !errors.Is(err, ErrUnsatisfiable) {
					t.Errorf("expected %v, but got %v", ErrUnsatisfiable, err)
				}
				return
			}
			if err != nil {
				t.Error(err)
			}

			pos, found := assignment[variable]
			if !found {
				t.Errorf("free variable %v not bound by generator", variable)
			}

			if !state.IsData(int(pos.Uint64())) {
				t.Error("IsCode constraint not satisfied")
			}

		})
	}
}

func TestSolver_largestFit(t *testing.T) {
	s := varCodeConstraintSolver{}
	s.codeSize = 34
	s.usedPositions = make(map[int]Used, s.codeSize)
	for i := 0; i < s.codeSize; i++ {
		s.usedPositions[i] = isUnused
	}
	if s.largestFit(0) > 33 {
		t.Error("largestFit should not return more than 33.")
	}
}
