package spc

import (
	. "github.com/Fantom-foundation/Tosca/go/ct/common"
	. "github.com/Fantom-foundation/Tosca/go/ct/rlz"
	"github.com/Fantom-foundation/Tosca/go/ct/st"
)

type specificationMap struct {
	rules map[OpCode][]Rule
}

var SpecMap = func() Specification {
	rules := getAllRules()
	return NewSpecification(rules...)
}()

func NewSpecificationMap(rules ...Rule) Specification {
	spec := &specificationMap{}
	for _, rule := range rules {
		op := ruleNameToOpcode(rule.Name)
		spec.rules[op] = append(spec.rules[op], rule)
	}
	return spec
}

func (s *specificationMap) GetRules() []Rule {
	// TODO check wether this is sufficient to allocate 3 rules per opcode

	allRules := make([]Rule, 0, len(s.rules)*3)
	for _, rules := range s.rules {
		allRules = append(allRules, rules...)
	}

	return allRules
}

func (s *specificationMap) GetRulesFor(state *st.State) []Rule {
	result := []Rule{}
	op, err := state.Code.GetOperation(int(state.Pc))
	if err != nil {
		return result
	}

	for _, rule := range s.rules[op] {
		if valid, err := rule.Condition.Check(state); valid && err == nil {
			result = append(result, rule)
		}
	}

	return result
}

func ruleNameToOpcode(name string) OpCode {
	return INVALID
}
