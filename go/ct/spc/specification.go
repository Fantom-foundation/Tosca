package spc

import (
	"fmt"
	"strings"

	. "github.com/Fantom-foundation/Tosca/go/ct/common"
	. "github.com/Fantom-foundation/Tosca/go/ct/rlz"
	"github.com/Fantom-foundation/Tosca/go/ct/st"
)

// Specification defines the interface for handling specifications.
type Specification interface {
	// GetRules provides access to all rules within the specification.
	GetRules() []Rule

	// GetRulesFor gives you all rules that apply to the given State (i.e. where
	// the rule's Condition holds).
	GetRulesFor(*st.State) []Rule
}

type specification struct {
	rules []Rule
}

func NewSpecification(rules ...Rule) Specification {
	return &specification{rules}
}

func (s *specification) GetRules() []Rule {
	return s.rules
}

func (s *specification) GetRulesFor(state *st.State) []Rule {
	result := []Rule{}
	for _, rule := range s.rules {
		if valid, err := rule.Condition.Check(state); valid && err == nil {
			result = append(result, rule)
		}
	}
	return result
}

////////////////////////////////////////////////////////////

func boolToU256(value bool) U256 {
	if value {
		return NewU256(1)
	}
	return NewU256(0)
}

var Spec = func() Specification {
	rules := []Rule{}

	// --- Terminal States ---

	rules = append(rules, []Rule{
		{
			Name:      "stopped_is_end",
			Condition: Eq(Status(), st.Stopped),
			Effect:    NoEffect(),
		},

		{
			Name:      "returned_is_end",
			Condition: Eq(Status(), st.Returned),
			Effect:    NoEffect(),
		},

		{
			Name:      "reverted_is_end",
			Condition: Eq(Status(), st.Reverted),
			Effect:    NoEffect(),
		},

		{
			Name:      "failed_is_end",
			Condition: Eq(Status(), st.Failed),
			Effect:    NoEffect(),
		},
	}...)

	// --- STOP ---

	rules = append(rules, Rule{
		Name: "stop_terminates_interpreter",
		Condition: And(
			Eq(Status(), st.Running),
			Eq(Op(Pc()), STOP),
		),
		Effect: Change(func(s *st.State) {
			s.Status = st.Stopped
			s.Pc++
		}),
	})

	// --- Arithmetic ---

	rules = append(rules, binaryOp(ADD, 3, func(a, b U256) U256 {
		return a.Add(b)
	})...)

	rules = append(rules, binaryOp(MUL, 5, func(a, b U256) U256 {
		return a.Mul(b)
	})...)

	rules = append(rules, binaryOp(SUB, 3, func(a, b U256) U256 {
		return a.Sub(b)
	})...)

	rules = append(rules, binaryOp(DIV, 5, func(a, b U256) U256 {
		return a.Div(b)
	})...)

	rules = append(rules, binaryOp(SDIV, 5, func(a, b U256) U256 {
		return a.SDiv(b)
	})...)

	rules = append(rules, binaryOp(MOD, 5, func(a, b U256) U256 {
		return a.Mod(b)
	})...)

	rules = append(rules, binaryOp(SMOD, 5, func(a, b U256) U256 {
		return a.SMod(b)
	})...)

	rules = append(rules, trinaryOp(ADDMOD, 8, func(a, b, n U256) U256 {
		return a.AddMod(b, n)
	})...)

	rules = append(rules, trinaryOp(MULMOD, 8, func(a, b, n U256) U256 {
		return a.MulMod(b, n)
	})...)

	rules = append(rules, binaryOpWithDynamicCost(EXP, 10, func(a, e U256) U256 {
		return a.Exp(e)
	}, func(a, e U256) uint64 {
		const gasFactor = uint64(50)
		expBytes := e.Bytes32be()
		for i := 0; i < 32; i++ {
			if expBytes[i] != 0 {
				return gasFactor * uint64(32-i)
			}
		}
		return 0
	})...)

	rules = append(rules, binaryOp(SIGNEXTEND, 5, func(b, x U256) U256 {
		return x.SignExtend(b)
	})...)

	rules = append(rules, binaryOp(LT, 3, func(a, b U256) U256 {
		return boolToU256(a.Lt(b))
	})...)

	rules = append(rules, binaryOp(GT, 3, func(a, b U256) U256 {
		return boolToU256(a.Gt(b))
	})...)

	rules = append(rules, binaryOp(SLT, 3, func(a, b U256) U256 {
		return boolToU256(a.Slt(b))
	})...)

	rules = append(rules, binaryOp(SGT, 3, func(a, b U256) U256 {
		return boolToU256(a.Sgt(b))
	})...)

	rules = append(rules, binaryOp(EQ, 3, func(a, b U256) U256 {
		return boolToU256(a.Eq(b))
	})...)

	rules = append(rules, unaryOp(ISZERO, 3, func(a U256) U256 {
		return boolToU256(a.IsZero())
	})...)

	rules = append(rules, binaryOp(AND, 3, func(a, b U256) U256 {
		return a.And(b)
	})...)

	rules = append(rules, binaryOp(OR, 3, func(a, b U256) U256 {
		return a.Or(b)
	})...)

	rules = append(rules, binaryOp(XOR, 3, func(a, b U256) U256 {
		return a.Xor(b)
	})...)

	rules = append(rules, unaryOp(NOT, 3, func(a U256) U256 {
		return a.Not()
	})...)

	rules = append(rules, binaryOp(BYTE, 3, func(i, x U256) U256 {
		if i.Gt(NewU256(31)) {
			return NewU256(0)
		}
		return NewU256(uint64(x.Bytes32be()[i.Uint64()]))
	})...)

	rules = append(rules, binaryOp(SHL, 3, func(shift, value U256) U256 {
		return value.Shl(shift)
	})...)

	rules = append(rules, binaryOp(SHR, 3, func(shift, value U256) U256 {
		return value.Shr(shift)
	})...)

	rules = append(rules, binaryOp(SAR, 3, func(shift, value U256) U256 {
		return value.Srsh(shift)
	})...)

	// --- JUMP ---

	rules = append(rules, []Rule{
		{
			Name: "jump_with_too_little_gas",
			Condition: And(
				Eq(Status(), st.Running),
				Eq(Op(Pc()), JUMP),
				Lt(Gas(), 8),
			),
			Effect: FailEffect(),
		},

		{
			Name: "jump_with_too_few_elements",
			Condition: And(
				Eq(Status(), st.Running),
				Eq(Op(Pc()), JUMP),
				Ge(Gas(), 8),
				Lt(StackSize(), 1),
			),
			Effect: FailEffect(),
		},

		{
			Name: "jump_to_data",
			Condition: And(
				Ge(StackSize(), 1),
				IsData(Param(0)),
				Eq(Status(), st.Running),
				Eq(Op(Pc()), JUMP),
				Ge(Gas(), 8),
			),
			Effect: FailEffect(),
		},

		{
			Name: "jump_to_invalid_destination",
			Condition: And(
				Eq(Status(), st.Running),
				Eq(Op(Pc()), JUMP),
				Ge(Gas(), 8),
				Ge(StackSize(), 1),
				IsCode(Param(0)),
				Ne(Op(Param(0)), JUMPDEST),
			),
			Effect: FailEffect(),
		},

		{
			Name: "jump_valid_target",
			Condition: And(
				Eq(Status(), st.Running),
				Eq(Op(Pc()), JUMP),
				Ge(Gas(), 8),
				Ge(StackSize(), 1),
				IsCode(Param(0)),
				Eq(Op(Param(0)), JUMPDEST),
			),
			Effect: Change(func(s *st.State) {
				s.Gas -= 8
				target := s.Stack.Pop()
				s.Pc = uint16(target.Uint64())
			}),
		},
	}...)

	// --- JUMPI ---

	rules = append(rules, []Rule{
		{
			Name: "jumpi_with_too_little_gas",
			Condition: And(
				Eq(Status(), st.Running),
				Eq(Op(Pc()), JUMPI),
				Lt(Gas(), 10),
			),
			Effect: FailEffect(),
		},

		{
			Name: "jumpi_with_too_few_elements",
			Condition: And(
				Eq(Status(), st.Running),
				Eq(Op(Pc()), JUMPI),
				Ge(Gas(), 10),
				Lt(StackSize(), 2),
			),
			Effect: FailEffect(),
		},

		{
			Name: "jumpi_not_taken",
			Condition: And(
				Eq(Status(), st.Running),
				Eq(Op(Pc()), JUMPI),
				Ge(Gas(), 10),
				Ge(StackSize(), 2),
				Eq(Param(1), NewU256(0)),
			),
			Effect: Change(func(s *st.State) {
				s.Gas -= 10
				s.Stack.Pop()
				s.Stack.Pop()
				s.Pc += 1
			}),
		},

		{
			Name: "jumpi_to_data",
			Condition: And(
				Ge(StackSize(), 2),
				IsData(Param(0)),
				Ne(Param(1), NewU256(0)),
				Eq(Status(), st.Running),
				Eq(Op(Pc()), JUMPI),
				Ge(Gas(), 10),
			),
			Effect: FailEffect(),
		},

		{
			Name: "jumpi_to_invalid_destination",
			Condition: And(
				Eq(Status(), st.Running),
				Eq(Op(Pc()), JUMPI),
				Ge(Gas(), 10),
				Ge(StackSize(), 2),
				IsCode(Param(0)),
				Ne(Op(Param(0)), JUMPDEST),
				Ne(Param(1), NewU256(0)),
			),
			Effect: FailEffect(),
		},

		{
			Name: "jumpi_valid_target",
			Condition: And(
				Eq(Status(), st.Running),
				Eq(Op(Pc()), JUMPI),
				Ge(Gas(), 10),
				Ge(StackSize(), 2),
				IsCode(Param(0)),
				Eq(Op(Param(0)), JUMPDEST),
				Ne(Param(1), NewU256(0)),
			),
			Effect: Change(func(s *st.State) {
				s.Gas -= 10
				target := s.Stack.Pop()
				s.Stack.Pop()
				s.Pc = uint16(target.Uint64())
			}),
		},
	}...)

	// --- JUMPDEST ---

	rules = append(rules, []Rule{
		{
			Name: "jumpdest_with_too_little_gas",
			Condition: And(
				Eq(Status(), st.Running),
				Eq(Op(Pc()), JUMPDEST),
				Lt(Gas(), 1),
			),
			Effect: FailEffect(),
		},

		{
			Name: "jumpdest_regular",
			Condition: And(
				Eq(Status(), st.Running),
				Eq(Op(Pc()), JUMPDEST),
				Ge(Gas(), 1),
			),
			Effect: Change(func(s *st.State) {
				s.Gas -= 1
				s.Pc++
			}),
		},
	}...)

	// --- End ---

	return NewSpecification(rules...)
}()

func binaryOpWithDynamicCost(
	op OpCode,
	costs uint64,
	effect func(a, b U256) U256,
	dynamicCost func(a, b U256) uint64,
) []Rule {
	name := strings.ToLower(op.String())
	return []Rule{
		{
			Name: fmt.Sprintf("%v_regular", name),
			Condition: And(
				Eq(Status(), st.Running),
				Eq(Op(Pc()), op),
				Ge(Gas(), costs),
				Ge(StackSize(), 2),
			),
			Parameter: []Parameter{
				NumericParameter{},
				NumericParameter{},
			},
			Effect: Change(func(s *st.State) {
				s.Pc++
				a := s.Stack.Pop()
				b := s.Stack.Pop()
				cost := costs + dynamicCost(a, b)
				// TODO: Improve handling of dynamic gas through dedicated constraint.
				if s.Gas < cost {
					s.Status = st.Failed
					s.Gas = 0
					return
				}
				s.Gas = s.Gas - cost
				s.Stack.Push(effect(a, b))
			}),
		},

		{
			Name: fmt.Sprintf("%v_with_too_little_gas", name),
			Condition: And(
				Eq(Status(), st.Running),
				Eq(Op(Pc()), op),
				Lt(Gas(), costs),
			),
			Effect: FailEffect(),
		},

		{
			Name: fmt.Sprintf("%v_with_too_few_elements", name),
			Condition: And(
				Eq(Status(), st.Running),
				Eq(Op(Pc()), op),
				Ge(Gas(), costs),
				Lt(StackSize(), 2),
			),
			Effect: FailEffect(),
		},
	}
}

func binaryOp(
	op OpCode,
	costs uint64,
	effect func(a, b U256) U256,
) []Rule {
	return binaryOpWithDynamicCost(op, costs, effect, func(_, _ U256) uint64 { return 0 })
}

func trinaryOp(
	op OpCode,
	costs uint64,
	effect func(a, b, c U256) U256,
) []Rule {
	name := strings.ToLower(op.String())
	return []Rule{
		{
			Name: fmt.Sprintf("%v_regular", name),
			Condition: And(
				Eq(Status(), st.Running),
				Eq(Op(Pc()), op),
				Ge(Gas(), costs),
				Ge(StackSize(), 3),
			),
			Parameter: []Parameter{
				NumericParameter{},
				NumericParameter{},
				NumericParameter{},
			},
			Effect: Change(func(s *st.State) {
				s.Gas = s.Gas - costs
				s.Pc++
				a := s.Stack.Pop()
				b := s.Stack.Pop()
				c := s.Stack.Pop()
				s.Stack.Push(effect(a, b, c))
			}),
		},

		{
			Name: fmt.Sprintf("%v_with_too_little_gas", name),
			Condition: And(
				Eq(Status(), st.Running),
				Eq(Op(Pc()), op),
				Lt(Gas(), costs),
			),
			Effect: FailEffect(),
		},

		{
			Name: fmt.Sprintf("%v_with_too_few_elements", name),
			Condition: And(
				Eq(Status(), st.Running),
				Eq(Op(Pc()), op),
				Ge(Gas(), costs),
				Lt(StackSize(), 3),
			),
			Effect: FailEffect(),
		},
	}
}

func unaryOp(
	op OpCode,
	costs uint64,
	effect func(a U256) U256,
) []Rule {
	name := strings.ToLower(op.String())
	return []Rule{
		{
			Name: fmt.Sprintf("%v_regular", name),
			Condition: And(
				Eq(Status(), st.Running),
				Eq(Op(Pc()), op),
				Ge(Gas(), costs),
				Ge(StackSize(), 1),
			),
			Parameter: []Parameter{
				NumericParameter{},
			},
			Effect: Change(func(s *st.State) {
				s.Pc++
				a := s.Stack.Pop()
				s.Gas = s.Gas - costs
				s.Stack.Push(effect(a))
			}),
		},

		{
			Name: fmt.Sprintf("%v_with_too_little_gas", name),
			Condition: And(
				Eq(Status(), st.Running),
				Eq(Op(Pc()), op),
				Lt(Gas(), costs),
			),
			Effect: FailEffect(),
		},

		{
			Name: fmt.Sprintf("%v_with_too_few_elements", name),
			Condition: And(
				Eq(Status(), st.Running),
				Eq(Op(Pc()), op),
				Ge(Gas(), costs),
				Lt(StackSize(), 1),
			),
			Effect: FailEffect(),
		},
	}
}
