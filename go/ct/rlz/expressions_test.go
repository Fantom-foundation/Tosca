package rlz

import (
	"testing"

	"pgregory.net/rand"

	. "github.com/Fantom-foundation/Tosca/go/ct/common"
	"github.com/Fantom-foundation/Tosca/go/ct/gen"
	"github.com/Fantom-foundation/Tosca/go/ct/st"
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

func TestExpression_GasRefundEval(t *testing.T) {
	state := st.NewState(st.NewCode([]byte{}))
	state.GasRefund = 42
	if gas, err := GasRefund().Eval(state); err != nil || gas != 42 {
		t.Fail()
	}
}

func TestExpression_GasRefundRestrict(t *testing.T) {
	generator := gen.NewStateGenerator()
	GasRefund().Restrict(RestrictEqual, 42, generator)

	state, err := generator.Generate(rand.New(0))
	if err != nil {
		t.Errorf("State generation failed %v", err)
	}
	if state.GasRefund != 42 {
		t.Errorf("Generator was not restricted by expression")
	}
}

func TestExpression_ReadOnlyEval(t *testing.T) {
	state := st.NewState(st.NewCode([]byte{}))
	state.ReadOnly = true
	if readOnly, err := ReadOnly().Eval(state); err != nil || readOnly != true {
		t.Fail()
	}
}

func TestExpression_ReadOnlyRestrict(t *testing.T) {
	generator := gen.NewStateGenerator()
	ReadOnly().Restrict(RestrictEqual, true, generator)

	state, err := generator.Generate(rand.New(0))
	if err != nil {
		t.Errorf("State generation failed %v", err)
	}
	if state.ReadOnly != true {
		t.Errorf("Generator was not restricted by expression")
	}
}

func TestExpression_OpEval(t *testing.T) {
	state := st.NewState(st.NewCode([]byte{byte(STOP), byte(STOP), byte(ADD)}))
	state.Pc = 2
	if op, err := Op(Pc()).Eval(state); err != nil || op != ADD {
		t.Fail()
	}
}

func TestExpression_OpRestrict(t *testing.T) {
	generator := gen.NewStateGenerator()
	Op(Pc()).Restrict(RestrictEqual, ADD, generator)

	state, err := generator.Generate(rand.New(0))
	if err != nil {
		t.Errorf("State generation failed %v", err)
	}
	if op, err := state.Code.GetOperation(int(state.Pc)); err != nil || op != ADD {
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
