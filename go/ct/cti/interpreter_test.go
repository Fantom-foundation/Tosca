package cti

import (
	"testing"

	"github.com/holiman/uint256"
	"golang.org/x/exp/slices"
)

func TestSTOP(t *testing.T) {
	s := State{
		Status:  Running,
		GasLeft: 100,
		Code:    []OpCode{STOP},
	}
	s.Run()
	if s.Status != Done {
		t.Fail()
	}
}

func TestSTOP_Empty(t *testing.T) {
	s := State{
		Status:  Running,
		GasLeft: 100,
		Code:    []OpCode{},
	}
	s.Run()
	if s.Status != Done {
		t.Fail()
	}
}

func TestADD(t *testing.T) {
	s := State{
		Status:  Running,
		GasLeft: 100,
		Code:    []OpCode{ADD},
		Stack:   []uint256.Int{*uint256.NewInt(21), *uint256.NewInt(42)},
	}
	s.Run()
	ok := s.Status == Done &&
		s.GasLeft == 100-3 &&
		slices.Equal(s.Stack, []uint256.Int{*uint256.NewInt(21 + 42)})
	if !ok {
		t.Fail()
	}
}

func TestADD_OutOfGas(t *testing.T) {
	s := State{
		Status:  Running,
		GasLeft: 1,
		Code:    []OpCode{ADD},
		Stack:   []uint256.Int{*uint256.NewInt(21), *uint256.NewInt(42)},
	}
	s.Run()
	if s.Status != ErrorGas {
		t.Fail()
	}
}

func TestADD_StackUnderflow(t *testing.T) {
	s := State{
		Status:  Running,
		GasLeft: 100,
		Code:    []OpCode{ADD},
		Stack:   []uint256.Int{*uint256.NewInt(42)},
	}
	s.Run()
	if s.Status != ErrorStackUnderflow {
		t.Fail()
	}
}

func TestJUMP_Valid(t *testing.T) {
	s := State{
		Status:  Running,
		GasLeft: 100,
		Code:    []OpCode{JUMP, PUSH1, JUMPDEST /* invalid */, JUMPDEST},
		Stack:   []uint256.Int{*uint256.NewInt(3)},
	}
	s.Run()
	ok := s.Status == Done &&
		s.GasLeft == 100-8-1
	if !ok {
		t.Fail()
	}
}

func TestJUMP_Invalid(t *testing.T) {
	s := State{
		Status:  Running,
		GasLeft: 100,
		Code:    []OpCode{JUMP, PUSH1, JUMPDEST /* invalid */, JUMPDEST},
		Stack:   []uint256.Int{*uint256.NewInt(2)},
	}
	s.Run()
	if s.Status != ErrorJump {
		t.Fail()
	}
}

func TestMLOAD(t *testing.T) {
	s := State{
		Status:  Running,
		GasLeft: 10,
		Code:    []OpCode{MLOAD},
		Stack:   []uint256.Int{*uint256.NewInt(2)},
		Memory: []byte{
			0, 0, 0, 0, 0, 0, 0, 0,
			0, 0, 0, 0, 0, 0, 0, 0,
			0, 0, 0, 0, 0, 0, 0, 0,
			0, 0, 0, 0, 0, 0, 0, 0x2a},
	}
	s.Run()

	expectedMem := make([]byte, 64)
	expectedMem[31] = 42

	ok := s.Status == Done &&
		s.GasLeft == 4 &&
		s.Stack[0].Eq(uint256.NewInt(0x2a0000)) &&
		slices.Equal(s.Memory, expectedMem)
	if !ok {
		t.Fail()
	}
}

func TestMSTORE(t *testing.T) {
	s := State{
		Status:  Running,
		GasLeft: 10,
		Code:    []OpCode{MSTORE},
		Stack:   []uint256.Int{*uint256.NewInt(42), *uint256.NewInt(2)},
	}
	s.Run()

	expectedMem := make([]byte, 64)
	expectedMem[33] = 42

	ok := s.Status == Done &&
		s.GasLeft == 1 &&
		slices.Equal(s.Memory, expectedMem)
	if !ok {
		t.Fail()
	}
}

func TestMSTORE8(t *testing.T) {
	s := State{
		Status:  Running,
		GasLeft: 10,
		Code:    []OpCode{MSTORE8},
		Stack:   []uint256.Int{*uint256.NewInt(0x3b2a), *uint256.NewInt(2)},
	}
	s.Run()

	expectedMem := make([]byte, 32)
	expectedMem[2] = 0x2a

	ok := s.Status == Done &&
		s.GasLeft == 4 &&
		slices.Equal(s.Memory, expectedMem)
	if !ok {
		t.Fail()
	}
}
