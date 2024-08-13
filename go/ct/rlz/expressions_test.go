// Copyright (c) 2024 Fantom Foundation
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at fantom.foundation/bsl11.
//
// Change Date: 2028-4-16
//
// On the date above, in accordance with the Business Source License, use of
// this software will be governed by the GNU Lesser General Public License v3.

package rlz

import (
	"errors"
	"fmt"
	"testing"

	"pgregory.net/rand"

	. "github.com/Fantom-foundation/Tosca/go/ct/common"
	"github.com/Fantom-foundation/Tosca/go/ct/gen"
	"github.com/Fantom-foundation/Tosca/go/ct/st"
	"github.com/Fantom-foundation/Tosca/go/tosca"
	"github.com/Fantom-foundation/Tosca/go/tosca/vm"
)

func TestExpression_StatusEval(t *testing.T) {
	state := st.NewState(st.NewCode([]byte{}))
	state.Status = st.Reverted
	if s, err := Status().Eval(state); err != nil || s != st.Reverted {
		t.Fail()
	}
}

func TestExpression_StatusRestrict(t *testing.T) {
	generator := gen.NewStateGenerator()
	Status().Restrict(RestrictEqual, st.Reverted, generator)

	state, err := generator.Generate(rand.New(0))
	if err != nil {
		t.Errorf("State generation failed %v", err)
	}
	if state.Status != st.Reverted {
		t.Errorf("Generator was not restricted by expression")
	}
}

func TestExpression_ReadOnlyEval(t *testing.T) {
	for _, readOnlyWant := range []bool{false, true} {
		state := st.NewState(st.NewCode([]byte{}))
		state.ReadOnly = readOnlyWant
		if readOnlyGet, err := ReadOnly().Eval(state); err != nil || readOnlyWant != readOnlyGet {
			t.Fail()
		}
	}
}

func TestExpression_ReadOnlyRestrict(t *testing.T) {
	for _, readOnly := range []bool{false, true} {
		generator := gen.NewStateGenerator()
		ReadOnly().Restrict(RestrictEqual, readOnly, generator)

		state, err := generator.Generate(rand.New(0))
		if err != nil {
			t.Errorf("State generation failed %v", err)
		}
		if state.ReadOnly != readOnly {
			t.Errorf("Generator was not restricted by expression")
		}
	}
}

func TestExpression_PcEval(t *testing.T) {
	state := st.NewState(st.NewCode([]byte{}))
	state.Pc = 42
	if pc, err := Pc().Eval(state); err != nil || pc != NewU256(42) {
		t.Fail()
	}
}

func TestExpression_PcRestrict(t *testing.T) {
	generator := gen.NewStateGenerator()
	Pc().Restrict(RestrictEqual, NewU256(42), generator)

	state, err := generator.Generate(rand.New(0))
	if err != nil {
		t.Errorf("State generation failed %v", err)
	}
	if state.Pc != 42 {
		t.Errorf("Generator was not restricted by expression")
	}
}

func TestExpression_GasEval(t *testing.T) {
	state := st.NewState(st.NewCode([]byte{}))
	state.Gas = 42
	if gas, err := Gas().Eval(state); err != nil || gas != 42 {
		t.Fail()
	}
}

func TestExpression_GasRestrict(t *testing.T) {
	generator := gen.NewStateGenerator()
	Gas().Restrict(RestrictEqual, 42, generator)

	state, err := generator.Generate(rand.New(0))
	if err != nil {
		t.Errorf("State generation failed %v", err)
	}
	if state.Gas != 42 {
		t.Errorf("Generator was not restricted by expression")
	}
}

func TestExpression_GasConstraints(t *testing.T) {
	tests := []struct {
		condition     Condition
		valid         bool // < if the condition holds for every gas value
		unsatisfiable bool // < if there is no gas value satisfying the condition
	}{
		// Equality
		{condition: Eq(Gas(), tosca.Gas(0))},
		{condition: Eq(Gas(), tosca.Gas(1))},
		{condition: Eq(Gas(), tosca.Gas(5))},
		{condition: Eq(Gas(), st.MaxGasUsedByCt)},

		// Not Equal
		{condition: Ne(Gas(), tosca.Gas(0))},
		{condition: Ne(Gas(), tosca.Gas(1))},
		{condition: Ne(Gas(), tosca.Gas(5))},
		{condition: Ne(Gas(), st.MaxGasUsedByCt)},

		// Less
		{condition: Lt(Gas(), tosca.Gas(0)), unsatisfiable: true},
		{condition: Lt(Gas(), tosca.Gas(1))},
		{condition: Lt(Gas(), tosca.Gas(5))},
		{condition: Lt(Gas(), st.MaxGasUsedByCt)},

		// Less or equal
		{condition: Le(Gas(), tosca.Gas(0))},
		{condition: Le(Gas(), tosca.Gas(1))},
		{condition: Le(Gas(), tosca.Gas(5))},
		{condition: Le(Gas(), st.MaxGasUsedByCt), valid: true},

		// Greater or equal
		{condition: Ge(Gas(), tosca.Gas(0)), valid: true},
		{condition: Ge(Gas(), tosca.Gas(1))},
		{condition: Ge(Gas(), tosca.Gas(5))},
		{condition: Ge(Gas(), st.MaxGasUsedByCt)},

		// Greater
		{condition: Gt(Gas(), tosca.Gas(0))},
		{condition: Gt(Gas(), tosca.Gas(1))},
		{condition: Gt(Gas(), tosca.Gas(5))},
		{condition: Gt(Gas(), st.MaxGasUsedByCt), unsatisfiable: true},

		// Ranges
		{condition: And(Ge(Gas(), tosca.Gas(4)), Le(Gas(), tosca.Gas(10)))},
		{condition: And(Ge(Gas(), tosca.Gas(4)), Le(Gas(), tosca.Gas(4)))},
		{condition: And(Gt(Gas(), tosca.Gas(4)), Le(Gas(), tosca.Gas(5)))},
		{condition: And(Ge(Gas(), tosca.Gas(4)), Lt(Gas(), tosca.Gas(5)))},

		{condition: And(Ge(Gas(), tosca.Gas(0)), Le(Gas(), st.MaxGasUsedByCt)), valid: true},
		{condition: And(Ge(Gas(), tosca.Gas(10)), Le(Gas(), tosca.Gas(4))), unsatisfiable: true},
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("%v", test.condition), func(t *testing.T) {
			random := rand.New()
			hits := 0
			misses := 0
			enumerateTestCases(test.condition, gen.NewStateGenerator(), func(g *gen.StateGenerator) ConsumerResult {
				state, err := g.Generate(random)
				if errors.Is(err, gen.ErrUnsatisfiable) {
					return ConsumeContinue // ignored
				}
				if err != nil {
					t.Fatalf("failed to generate test case: %v", err)
				}
				match, err := test.condition.Check(state)
				if err != nil {
					t.Fatalf("failed to check condition: %v", err)
				}
				if match {
					hits++
				} else {
					misses++
				}
				if hits > 0 && misses > 0 {
					return ConsumeAbort
				}
				return ConsumeContinue
			})
			if hits == 0 && !test.unsatisfiable {
				t.Errorf("failed to generate matching test case, got %d hits and %d misses", hits, misses)
			}
			if misses == 0 && !test.valid {
				t.Errorf("failed to generate non-matching test case, got %d hits and %d misses", hits, misses)
			}
		})
	}
}

func TestExpression_OpEval(t *testing.T) {
	state := st.NewState(st.NewCode([]byte{byte(vm.STOP), byte(vm.STOP), byte(vm.ADD)}))
	state.Pc = 2
	if op, err := Op(Pc()).Eval(state); err != nil || op != vm.ADD {
		t.Fail()
	}
}

func TestExpression_OpRestrict(t *testing.T) {
	generator := gen.NewStateGenerator()
	Op(Pc()).Restrict(RestrictEqual, vm.ADD, generator)

	state, err := generator.Generate(rand.New(0))
	if err != nil {
		t.Errorf("State generation failed %v", err)
	}
	if op, err := state.Code.GetOperation(int(state.Pc)); err != nil || op != vm.ADD {
		t.Errorf("Generator was not restricted by expression")
	}
}

func TestExpression_StackSizeEval(t *testing.T) {
	state := st.NewState(st.NewCode([]byte{}))
	state.Stack.Push(NewU256(1))
	state.Stack.Push(NewU256(2))
	state.Stack.Push(NewU256(4))
	if size, err := StackSize().Eval(state); err != nil || size != 3 {
		t.Fail()
	}
}

func TestExpression_StackSizeRestrict(t *testing.T) {
	generator := gen.NewStateGenerator()
	StackSize().Restrict(RestrictEqual, 4, generator)

	state, err := generator.Generate(rand.New(0))
	if err != nil {
		t.Errorf("State generation failed %v", err)
	}
	if state.Stack.Size() != 4 {
		t.Errorf("Generator was not restricted by expression")
	}
}

func TestConstant_HumanFriendlyPrinting(t *testing.T) {
	tests := []struct {
		expression BindableExpression[U256]
		print      string
	}{
		{Constant(NewU256(0)), "0"},
		{Constant(NewU256(1)), "1"},
		{Constant(NewU256(256)), "256"},
		{Constant(NewU256(123456)), "123456"},
		{Constant(NewU256(1, 2, 3, 4)), "0000000000000001 0000000000000002 0000000000000003 0000000000000004"},
	}

	for _, test := range tests {
		t.Run(test.print, func(t *testing.T) {
			if want, got := test.print, test.expression.String(); want != got {
				t.Errorf("unexpected print, wanted %s, got %s", want, got)
			}
			if want, got := "$constant_"+test.print, test.expression.GetVariable().String(); want != got {
				t.Errorf("unexpected print, wanted %s, got %s", want, got)
			}
		})
	}
}

func TestConstant_EvalReturnsValue(t *testing.T) {
	tests := []U256{
		NewU256(0),
		NewU256(1),
		NewU256(256),
		NewU256(123456),
		NewU256(1, 2, 3, 4),
	}

	for _, test := range tests {
		t.Run(test.String(), func(t *testing.T) {
			c := Constant(test)
			value, err := c.Eval(nil)
			if err != nil {
				t.Fatalf("failed to evaluate constant: %v", err)
			}
			if value != test {
				t.Errorf("unexpected value for constant, wanted %v, got %v", test, value)
			}
		})
	}
}

func TestExpression_RestrictPanicsWhenCalledWithCertainKind(t *testing.T) {

	tests := map[string]struct {
		expression interface{}
		kind       RestrictionKind
		value      interface{}
	}{
		"Status":       {Status(), RestrictGreater, st.Reverted},
		"Pc":           {Pc(), RestrictGreater, NewU256(42)},
		"Pc-notUint64": {Pc(), RestrictEqual, NewU256(42, 42)},
		"readOnly":     {ReadOnly(), RestrictGreater, true},
		"op":           {Op(Pc()), RestrictGreater, vm.ADD},
		"param":        {Param(0), RestrictGreater, NewU256(42)},
		"constant":     {Constant(NewU256(42)), RestrictGreater, NewU256(42)},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			defer func() {
				if r := recover(); r == nil {
					t.Errorf("expected panic, but none occurred")
				}
			}()
			switch v := test.expression.(type) {
			case status:
				v.Restrict(test.kind, test.value.(st.StatusCode), gen.NewStateGenerator())
			case pc:
				v.Restrict(test.kind, test.value.(U256), gen.NewStateGenerator())
			case readOnly:
				v.Restrict(test.kind, test.value.(bool), gen.NewStateGenerator())
			case op:
				v.Restrict(test.kind, test.value.(vm.OpCode), gen.NewStateGenerator())
			case param:
				v.Restrict(test.kind, test.value.(U256), gen.NewStateGenerator())
			case constant:
				v.Restrict(test.kind, test.value.(U256), gen.NewStateGenerator())
			default:
				t.Fatalf("unsupported expression type %T", test.expression)
			}
		})
	}
}

func TestExpression_Property(t *testing.T) {

	tests := []struct {
		got  Property
		want string
	}{
		{ReadOnly().Property(), "readOnly"},
		{Param(0).Property(), "param[0]"},
		{Constant(NewU256(42)).Property(), "constant(42)"},
		{Constant(NewU256(1, 42)).Property(), "constant(18446744073709551658)"},
	}

	for _, test := range tests {
		if string(test.got) != test.want {
			t.Errorf("unexpected property, wanted %s, got %s", test.want, test.got)
		}
	}
}

func TestExpressions_StackSizeRestrict(t *testing.T) {

	tests := map[string]struct {
		kind  RestrictionKind
		size  int
		check func(int) bool
	}{
		"Less":         {RestrictLess, 5, func(size int) bool { return size < 5 }},
		"LessEqual":    {RestrictLessEqual, 5, func(size int) bool { return size <= 5 }},
		"Equal":        {RestrictEqual, 0, func(size int) bool { return size == 0 }},
		"GreaterEqual": {RestrictGreaterEqual, 5, func(size int) bool { return size >= 5 }},
		"Greater":      {RestrictGreater, 5, func(size int) bool { return size > 5 }},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			generator := gen.NewStateGenerator()
			StackSize().Restrict(test.kind, test.size, generator)

			state, err := generator.Generate(rand.New(0))
			if err != nil {
				t.Errorf("State generation failed %v", err)
			}
			if !test.check(state.Stack.Size()) {
				t.Errorf("Generator was not restricted by expression. got: %d", state.Stack.Size())
			}
		})
	}

}
