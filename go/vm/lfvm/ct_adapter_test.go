package lfvm

import (
	"testing"

	ct "github.com/Fantom-foundation/Tosca/go/ct/common"
	"github.com/Fantom-foundation/Tosca/go/ct/st"
)

func TestCtAdapter_Add(t *testing.T) {
	s := &st.State{
		Status:   st.Running,
		Revision: st.Istanbul,
		Pc:       0,
		Gas:      100,
		Code: st.NewCode([]byte{
			byte(ct.PUSH1), 3,
			byte(ct.PUSH1), 4,
			byte(ct.ADD),
		}),
		Stack: st.NewStack(),
	}

	c := NewConformanceTestingTarget()

	s, err := c.StepN(s, 4)

	if err != nil {
		t.Fatalf("unexpected conversion error: %v", err)
	}

	if want, got := st.Stopped, s.Status; want != got {
		t.Fatalf("unexpected status: wanted %v, got %v", want, got)
	}

	if want, got := ct.NewU256(3+4), s.Stack.Get(0); !want.Eq(got) {
		t.Errorf("unexpected result: wanted %s, got %s", want, got)
	}
}

func TestCtAdapter_Interface(t *testing.T) {
	// Compile time check that ctAdapter implements the st.Evm interface.
	var _ st.Evm = ctAdapter{}
}
