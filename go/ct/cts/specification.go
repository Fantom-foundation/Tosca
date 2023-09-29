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
			ct.In(ct.Op(ct.Pc()), getInvalidOps()),
		),
		Effect: Fail(),
	},

	// --- STOP ---

	ct.Rule{
		Name: "stop_terminates_interpreter",
		Condition: ct.And(
			ct.Eq(ct.Status(), ct.Running),
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
			ct.Eq(ct.Op(ct.Pc()), ct.POP),
			ct.Lt(ct.Gas(), 2),
		),
		Effect: Fail(),
	},

	ct.Rule{
		Name: "pop_with_no_element",
		Condition: ct.And(
			ct.Eq(ct.Status(), ct.Running),
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
			ct.Eq(ct.Op(ct.Pc()), ct.PUSH1),
			ct.Lt(ct.Gas(), 3),
		),
		Effect: Fail(),
	},

	ct.Rule{
		Name: "push1_with_no_empty_space",
		Condition: ct.And(
			ct.Eq(ct.Status(), ct.Running),
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
			ct.Eq(ct.Op(ct.Pc()), ct.ADD),
			ct.Lt(ct.Gas(), 3),
		),
		Effect: Fail(),
	},

	ct.Rule{
		Name: "add_with_too_few_elements",
		Condition: ct.And(
			ct.Eq(ct.Status(), ct.Running),
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
		case ct.STOP, ct.POP, ct.ADD, ct.PUSH1:
			// skip
		default:
			res = append(res, op)
		}
	}
	return res
}
