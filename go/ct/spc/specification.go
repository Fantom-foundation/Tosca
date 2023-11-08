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

	// --- End ---

	return NewSpecification(rules...)
}()

func binaryOp(
	op OpCode,
	costs uint64,
	effect func(a, b U256) U256,
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
				s.Gas = s.Gas - costs
				s.Pc++
				a := s.Stack.Pop()
				b := s.Stack.Pop()
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
