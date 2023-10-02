package cti

import (
	"testing"

	"github.com/Fantom-foundation/Tosca/go/ct"
	"github.com/holiman/uint256"
)

func TestCtiAdapter(t *testing.T) {
	s := ct.State{
		Status: ct.Running,
		Gas:    100,
		Code:   []byte{byte(ct.ADD)},
		Stack:  ct.NewStack([]uint256.Int{*uint256.NewInt(21), *uint256.NewInt(42)}),
	}

	var evm CtAdapter

	s, err := evm.StepN(s, 2)
	if err != nil {
		t.Error(err)
	}

	top := s.Stack.Get(0)

	if s.Status != ct.Stopped || !top.Eq(uint256.NewInt(21+42)) {
		t.Fail()
	}
}
