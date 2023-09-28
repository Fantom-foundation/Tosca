package cts

import "github.com/Fantom-foundation/Tosca/go/ct"

var Specification = ct.NewSpecification(

	// --- Terminal States ---

	ct.Rule{
		Name:      "stopped_is_end",
		Condition: ct.Eq(ct.Status(), ct.Stopped),
		Effect:    ct.NoEffect(),
	},

	ct.Rule{
		Name:      "returned_is_end",
		Condition: ct.Eq(ct.Status(), ct.Returned),
		Effect:    ct.NoEffect(),
	},

	ct.Rule{
		Name:      "reverted_is_end",
		Condition: ct.Eq(ct.Status(), ct.Reverted),
		Effect:    ct.NoEffect(),
	},

	ct.Rule{
		Name:      "failed_is_end",
		Condition: ct.Eq(ct.Status(), ct.Failed),
		Effect:    ct.NoEffect(),
	},

	// --- Invalid Operations ---

	ct.Rule{
		Name: "invalid_operation",
		Condition: ct.And(
			ct.Eq(ct.Status(), ct.Running),
			ct.In(ct.Op(ct.Pc()), getInvalidOps()),
		),
		Effect: ct.Update(func(s ct.State) ct.State {
			s.Status = ct.Failed
			s.Gas = 0
			return s
		}),
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
		Effect: ct.Update(func(s ct.State) ct.State {
			s.Status = ct.Failed
			s.Gas = 0
			return s
		}),
	},

	ct.Rule{
		Name: "pop_with_no_element",
		Condition: ct.And(
			ct.Eq(ct.Status(), ct.Running),
			ct.Eq(ct.Op(ct.Pc()), ct.POP),
			ct.Ge(ct.Gas(), 2),
			ct.Lt(ct.StackSize(), 1),
		),
		Effect: ct.Update(func(s ct.State) ct.State {
			s.Status = ct.Failed
			s.Gas = 0
			return s
		}),
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
			s.Stack = s.Stack[:len(s.Stack)-1]
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
		Effect: ct.Update(func(s ct.State) ct.State {
			s.Status = ct.Failed
			s.Gas = 0
			return s
		}),
	},

	ct.Rule{
		Name: "add_with_too_few_elements",
		Condition: ct.And(
			ct.Eq(ct.Status(), ct.Running),
			ct.Eq(ct.Op(ct.Pc()), ct.ADD),
			ct.Ge(ct.Gas(), 3),
			ct.Lt(ct.StackSize(), 2),
		),
		Effect: ct.Update(func(s ct.State) ct.State {
			s.Status = ct.Failed
			s.Gas = 0
			return s
		}),
	},

	ct.Rule{
		Name: "add_regular",
		Condition: ct.And(
			ct.Eq(ct.Status(), ct.Running),
			ct.Eq(ct.Op(ct.Pc()), ct.ADD),
			ct.Ge(ct.Gas(), 3),
			ct.Ge(ct.StackSize(), 2),
		),
		Effect: ct.Update(func(s ct.State) ct.State {
			s.Gas = s.Gas - 3

			// TODO:
			//  - add ways to stress-test arithmetic
			a := s.Stack[len(s.Stack)-1]
			b := s.Stack[len(s.Stack)-1]
			s.Stack = s.Stack[:len(s.Stack)-1]
			s.Stack[len(s.Stack)-1].Add(&a, &b)
			return s
		}),
	},
)

func getInvalidOps() []ct.OpCode {
	res := make([]ct.OpCode, 0, 256)
	for i := 0; i < 256; i++ {
		op := ct.OpCode(i)
		switch op {
		case ct.STOP, ct.POP, ct.ADD:
			// skip
		default:
			res = append(res, op)
		}
	}
	return res
}
