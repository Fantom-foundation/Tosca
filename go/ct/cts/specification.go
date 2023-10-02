package cts

import (
	"github.com/Fantom-foundation/Tosca/go/ct"
	"github.com/holiman/uint256"
)

var Specification = ct.NewSpecification(

	// --- Terminal States ---

	ct.Rule{
		Name:      "stopped_is_end",
		Condition: ct.Eq(ct.Status(), ct.Stopped),
		Effect:    NoEffect(),
	},

	ct.Rule{
		Name:      "returned_is_end",
		Condition: ct.Eq(ct.Status(), ct.Returned),
		Effect:    NoEffect(),
	},

	ct.Rule{
		Name:      "reverted_is_end",
		Condition: ct.Eq(ct.Status(), ct.Reverted),
		Effect:    NoEffect(),
	},

	ct.Rule{
		Name:      "failed_is_end",
		Condition: ct.Eq(ct.Status(), ct.Failed),
		Effect:    NoEffect(),
	},

	// --- Invalid Operations ---

	ct.Rule{
		Name: "invalid_operation",
		Condition: ct.And(
			ct.Eq(ct.Status(), ct.Running),
			ct.IsCode(ct.Pc()),
			ct.In(ct.Op(ct.Pc()), getInvalidOps()),
		),
		Effect: Fail(),
	},

	ct.Rule{
		Name: "invalid_pc",
		Condition: ct.And(
			ct.Eq(ct.Status(), ct.Running),
			ct.IsData(ct.Pc()),
		),
		Effect: Fail(),
	},

	// --- STOP ---

	ct.Rule{
		Name: "stop_terminates_interpreter",
		Condition: ct.And(
			ct.Eq(ct.Status(), ct.Running),
			ct.IsCode(ct.Pc()),
			ct.Eq(ct.Op(ct.Pc()), ct.STOP),
		),
		Effect: ct.Update(func(s ct.State) ct.State {
			s.Status = ct.Stopped
			return s
		}),
	},

	// --- POP ---

	ct.Rule{
		Name: "pop_with_too_little_gas",
		Condition: ct.And(
			ct.Eq(ct.Status(), ct.Running),
			ct.IsCode(ct.Pc()),
			ct.Eq(ct.Op(ct.Pc()), ct.POP),
			ct.Lt(ct.Gas(), 2),
		),
		Effect: Fail(),
	},

	ct.Rule{
		Name: "pop_with_no_element",
		Condition: ct.And(
			ct.Eq(ct.Status(), ct.Running),
			ct.IsCode(ct.Pc()),
			ct.Eq(ct.Op(ct.Pc()), ct.POP),
			ct.Ge(ct.Gas(), 2),
			ct.Lt(ct.StackSize(), 1),
		),
		Effect: Fail(),
	},

	ct.Rule{
		Name: "pop_regular",
		Condition: ct.And(
			ct.Eq(ct.Status(), ct.Running),
			ct.IsCode(ct.Pc()),
			ct.Eq(ct.Op(ct.Pc()), ct.POP),
			ct.Ge(ct.Gas(), 2),
			ct.Ge(ct.StackSize(), 1),
		),
		Effect: ct.Update(func(s ct.State) ct.State {
			s.Gas = s.Gas - 2
			s.Pc++
			s.Stack.Pop()
			return s
		}),
	},

	// --- PUSH1 ---

	ct.Rule{
		Name: "push1_with_too_little_gas",
		Condition: ct.And(
			ct.Eq(ct.Status(), ct.Running),
			ct.IsCode(ct.Pc()),
			ct.Eq(ct.Op(ct.Pc()), ct.PUSH1),
			ct.Lt(ct.Gas(), 3),
		),
		Effect: Fail(),
	},

	ct.Rule{
		Name: "push1_with_no_empty_space",
		Condition: ct.And(
			ct.Eq(ct.Status(), ct.Running),
			ct.IsCode(ct.Pc()),
			ct.Eq(ct.Op(ct.Pc()), ct.PUSH1),
			ct.Ge(ct.Gas(), 3),
			ct.Ge(ct.StackSize(), 1024),
		),
		Effect: Fail(),
	},

	ct.Rule{
		Name: "push1_regular",
		Condition: ct.And(
			ct.Eq(ct.Status(), ct.Running),
			ct.IsCode(ct.Pc()),
			ct.Eq(ct.Op(ct.Pc()), ct.PUSH1),
			ct.Ge(ct.Gas(), 3),
			ct.Lt(ct.StackSize(), 1024),
		),
		Effect: ct.Update(func(s ct.State) ct.State {
			s.Gas = s.Gas - 3
			value := uint256.NewInt(0)
			if int(s.Pc+1) < len(s.Code) {
				value.SetBytes(s.Code[s.Pc+1 : s.Pc+2])
			}
			s.Stack.Push(*value)
			s.Pc = s.Pc + 2
			return s
		}),
	},

	// --- ADD ---

	ct.Rule{
		Name: "add_with_too_little_gas",
		Condition: ct.And(
			ct.Eq(ct.Status(), ct.Running),
			ct.IsCode(ct.Pc()),
			ct.Eq(ct.Op(ct.Pc()), ct.ADD),
			ct.Lt(ct.Gas(), 3),
		),
		Effect: Fail(),
	},

	ct.Rule{
		Name: "add_with_too_few_elements",
		Condition: ct.And(
			ct.Eq(ct.Status(), ct.Running),
			ct.IsCode(ct.Pc()),
			ct.Eq(ct.Op(ct.Pc()), ct.ADD),
			ct.Ge(ct.Gas(), 3),
			ct.Lt(ct.StackSize(), 2),
		),
		Effect: Fail(),
	},

	ct.Rule{
		Name: "add_regular",
		Condition: ct.And(
			ct.Eq(ct.Status(), ct.Running),
			ct.IsCode(ct.Pc()),
			ct.Eq(ct.Op(ct.Pc()), ct.ADD),
			ct.Ge(ct.Gas(), 3),
			ct.Ge(ct.StackSize(), 2),
		),
		Parameter: []ct.Parameter{
			ct.NumericParameter{},
			ct.NumericParameter{},
		},
		Effect: ct.Update(func(s ct.State) ct.State {
			s.Gas = s.Gas - 3
			s.Pc++
			a := s.Stack.Pop()
			b := s.Stack.Pop()
			s.Stack.Push(*a.Add(&a, &b))
			return s
		}),
	},

	// --- JUMP ---

	ct.Rule{
		Name: "jump_with_too_little_gas",
		Condition: ct.And(
			ct.Eq(ct.Status(), ct.Running),
			ct.IsCode(ct.Pc()),
			ct.Eq(ct.Op(ct.Pc()), ct.JUMP),
			ct.Lt(ct.Gas(), 8),
		),
		Effect: Fail(),
	},

	ct.Rule{
		Name: "jump_with_too_few_elements",
		Condition: ct.And(
			ct.Eq(ct.Status(), ct.Running),
			ct.IsCode(ct.Pc()),
			ct.Eq(ct.Op(ct.Pc()), ct.JUMP),
			ct.Ge(ct.Gas(), 8),
			ct.Lt(ct.StackSize(), 1),
		),
		Effect: Fail(),
	},

	ct.Rule{
		Name: "jump_to_data",
		Condition: ct.And(
			ct.Ge(ct.StackSize(), 1),
			ct.IsData(ct.Param(0)),
			ct.Eq(ct.Status(), ct.Running),
			ct.IsCode(ct.Pc()),
			ct.Eq(ct.Op(ct.Pc()), ct.JUMP),
			ct.Ge(ct.Gas(), 8),
		),
		Effect: Fail(),
	},

	ct.Rule{
		Name: "jump_to_invalid_destination",
		Condition: ct.And(
			ct.Eq(ct.Status(), ct.Running),
			ct.IsCode(ct.Pc()),
			ct.Eq(ct.Op(ct.Pc()), ct.JUMP),
			ct.Ge(ct.Gas(), 8),
			ct.Ge(ct.StackSize(), 1),
			ct.IsCode(ct.Param(0)),
			ct.Ne(ct.Op(ct.Param(0)), ct.JUMPDEST),
		),
		Effect: Fail(),
	},

	ct.Rule{
		Name: "jump_valid_target",
		Condition: ct.And(
			ct.Eq(ct.Status(), ct.Running),
			ct.IsCode(ct.Pc()),
			ct.Eq(ct.Op(ct.Pc()), ct.JUMP),
			ct.Ge(ct.Gas(), 8),
			ct.Ge(ct.StackSize(), 1),
			ct.IsCode(ct.Param(0)),
			ct.Eq(ct.Op(ct.Param(0)), ct.JUMPDEST),
		),
		Effect: ct.Update(func(s ct.State) ct.State {
			s.Gas = s.Gas - 8
			target := s.Stack.Pop()
			s.Pc = uint16(target.Uint64())
			return s
		}),
	},

	// --- JUMPI ---

	ct.Rule{
		Name: "jumpi_with_too_little_gas",
		Condition: ct.And(
			ct.Eq(ct.Status(), ct.Running),
			ct.IsCode(ct.Pc()),
			ct.Eq(ct.Op(ct.Pc()), ct.JUMPI),
			ct.Lt(ct.Gas(), 10),
		),
		Effect: Fail(),
	},

	ct.Rule{
		Name: "jumpi_with_too_few_elements",
		Condition: ct.And(
			ct.Eq(ct.Status(), ct.Running),
			ct.IsCode(ct.Pc()),
			ct.Eq(ct.Op(ct.Pc()), ct.JUMPI),
			ct.Ge(ct.Gas(), 10),
			ct.Lt(ct.StackSize(), 2),
		),
		Effect: Fail(),
	},

	ct.Rule{
		Name: "jumpi_not_taken",
		Condition: ct.And(
			ct.Eq(ct.Status(), ct.Running),
			ct.IsCode(ct.Pc()),
			ct.Eq(ct.Op(ct.Pc()), ct.JUMPI),
			ct.Ge(ct.Gas(), 10),
			ct.Ge(ct.StackSize(), 2),
			ct.Eq(ct.Param(1), *uint256.NewInt(0)),
		),
		Effect: ct.Update(func(s ct.State) ct.State {
			s.Gas = s.Gas - 10
			s.Stack.Pop()
			s.Stack.Pop()
			s.Pc = s.Pc + 1
			return s
		}),
	},

	ct.Rule{
		Name: "jumpi_to_data",
		Condition: ct.And(
			ct.Ge(ct.StackSize(), 2),
			ct.IsData(ct.Param(0)),
			ct.Ne(ct.Param(1), *uint256.NewInt(0)),
			ct.Eq(ct.Status(), ct.Running),
			ct.IsCode(ct.Pc()),
			ct.Eq(ct.Op(ct.Pc()), ct.JUMPI),
			ct.Ge(ct.Gas(), 10),
		),
		Effect: Fail(),
	},

	ct.Rule{
		Name: "jumpi_to_invalid_destination",
		Condition: ct.And(
			ct.Eq(ct.Status(), ct.Running),
			ct.IsCode(ct.Pc()),
			ct.Eq(ct.Op(ct.Pc()), ct.JUMPI),
			ct.Ge(ct.Gas(), 10),
			ct.Ge(ct.StackSize(), 2),
			ct.IsCode(ct.Param(0)),
			ct.Ne(ct.Op(ct.Param(0)), ct.JUMPDEST),
			ct.Ne(ct.Param(1), *uint256.NewInt(0)),
		),
		Effect: Fail(),
	},

	ct.Rule{
		Name: "jumpi_valid_target",
		Condition: ct.And(
			ct.Eq(ct.Status(), ct.Running),
			ct.IsCode(ct.Pc()),
			ct.Eq(ct.Op(ct.Pc()), ct.JUMPI),
			ct.Ge(ct.Gas(), 10),
			ct.Ge(ct.StackSize(), 2),
			ct.IsCode(ct.Param(0)),
			ct.Eq(ct.Op(ct.Param(0)), ct.JUMPDEST),
			ct.Ne(ct.Param(1), *uint256.NewInt(0)),
		),
		Effect: ct.Update(func(s ct.State) ct.State {
			s.Gas = s.Gas - 10
			target := s.Stack.Pop()
			s.Stack.Pop()
			s.Pc = uint16(target.Uint64())
			return s
		}),
	},

	// --- JUMPDEST ---

	ct.Rule{
		Name: "jumpdest_with_too_little_gas",
		Condition: ct.And(
			ct.Eq(ct.Status(), ct.Running),
			ct.IsCode(ct.Pc()),
			ct.Eq(ct.Op(ct.Pc()), ct.JUMPDEST),
			ct.Lt(ct.Gas(), 1),
		),
		Effect: Fail(),
	},

	ct.Rule{
		Name: "jumpdest_regular",
		Condition: ct.And(
			ct.Eq(ct.Status(), ct.Running),
			ct.IsCode(ct.Pc()),
			ct.Eq(ct.Op(ct.Pc()), ct.JUMPDEST),
			ct.Ge(ct.Gas(), 1),
		),
		Effect: ct.Update(func(s ct.State) ct.State {
			s.Gas = s.Gas - 1
			s.Pc++
			return s
		}),
	},
)

func NoEffect() ct.Effect {
	return ct.Update(func(s ct.State) ct.State { return s })
}

func Fail() ct.Effect {
	return ct.Update(func(s ct.State) ct.State {
		s.Status = ct.Failed
		s.Gas = 0
		return s
	})
}

func getInvalidOps() []ct.OpCode {
	res := make([]ct.OpCode, 0, 256)
	for i := 0; i < 256; i++ {
		op := ct.OpCode(i)
		switch op {
		case ct.STOP, ct.POP, ct.ADD, ct.PUSH1, ct.JUMP, ct.JUMPI, ct.JUMPDEST:
			// skip
		default:
			res = append(res, op)
		}
	}
	return res
}
