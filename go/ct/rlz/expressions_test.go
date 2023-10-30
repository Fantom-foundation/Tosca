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
