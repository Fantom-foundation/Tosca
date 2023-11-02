package rlz

import (
	"testing"

	"pgregory.net/rand"

	"github.com/Fantom-foundation/Tosca/go/ct"
	"github.com/Fantom-foundation/Tosca/go/ct/gen"
	"github.com/Fantom-foundation/Tosca/go/ct/st"
)

func TestExpression_StatusEval(t *testing.T) {
	state := st.NewState(st.NewCode([]byte{}))
	state.Status = st.Reverted
	if Status().Eval(state) != st.Reverted {
		t.Fail()
	}
}

func TestExpression_StatusRestrict(t *testing.T) {
	generator := gen.NewStateGenerator()
	Status().Restrict(st.Reverted, generator)

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
	if Pc().Eval(state) != ct.NewU256(42) {
		t.Fail()
	}
}

func TestExpression_PcRestrict(t *testing.T) {
	generator := gen.NewStateGenerator()
	Pc().Restrict(ct.NewU256(42), generator)

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
	if Gas().Eval(state) != 42 {
		t.Fail()
	}
}

func TestExpression_GasRestrict(t *testing.T) {
	generator := gen.NewStateGenerator()
	Gas().Restrict(42, generator)

	state, err := generator.Generate(rand.New(0))
	if err != nil {
		t.Errorf("State generation failed %v", err)
	}
	if state.Gas != 42 {
		t.Errorf("Generator was not restricted by expression")
	}
}

func TestExpression_StackSizeEval(t *testing.T) {
	state := st.NewState(st.NewCode([]byte{}))
	state.Stack.Push(ct.NewU256(1))
	state.Stack.Push(ct.NewU256(2))
	state.Stack.Push(ct.NewU256(4))
	if StackSize().Eval(state) != 3 {
		t.Fail()
	}
}

func TestExpression_StackSizeRestrict(t *testing.T) {
	generator := gen.NewStateGenerator()
	StackSize().Restrict(4, generator)

	state, err := generator.Generate(rand.New(0))
	if err != nil {
		t.Errorf("State generation failed %v", err)
	}
	if state.Stack.Size() != 4 {
		t.Errorf("Generator was not restricted by expression")
	}
}
