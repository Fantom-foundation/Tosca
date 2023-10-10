package cti

import (
	"testing"

	"github.com/holiman/uint256"
)

func stackEq(s *State, expected []uint256.Int) bool {
	if len(s.Stack) != len(expected) {
		return false
	}
	for i := 0; i < len(expected); i++ {
		if s.Stack[i] != expected[i] {
			return false
		}
	}
	return true
}

func memoryEq(s *State, expected []byte) bool {
	if len(s.Memory) != len(expected) {
		return false
	}
	for i := range s.Memory {
		if s.Memory[i] != expected[i] {
			return false
		}
	}
	return true
}

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
		stackEq(&s, []uint256.Int{*uint256.NewInt(21 + 42)})
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

func TestMStore(t *testing.T) {
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
		memoryEq(&s, expectedMem)
	if !ok {
		t.Fail()
	}
}
