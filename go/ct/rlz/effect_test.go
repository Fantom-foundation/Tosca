package rlz

import (
	"testing"

	"github.com/Fantom-foundation/Tosca/go/ct/st"
)

func TestEffect_Change(t *testing.T) {
	pcAdd1 := Change(func(s *st.State) {
		s.Pc += 1
	})

	state := st.NewState(st.NewCode([]byte{}))
	state.Pc = 0

	pcAdd1.Apply(state)
	if state.Pc != 1 {
		t.Errorf("effect did not apply")
	}
}
