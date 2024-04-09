package spc

import (
	"regexp"
	"strings"

	"github.com/Fantom-foundation/Tosca/go/ct/common"
	. "github.com/Fantom-foundation/Tosca/go/ct/rlz"
	"github.com/Fantom-foundation/Tosca/go/ct/st"
)

type specificationMap struct {
	rules map[string][]Rule
}

var Spec = func() Specification {
	rules := getAllRules()
	return NewSpecificationMap(rules...)
}()

func (s *specificationMap) GetRules() []Rule {
	// allocating space for 5 rules per rule, checked with GetAllRules benchmark
	allRules := make([]Rule, 0, len(s.rules)*5)

	for _, rules := range s.rules {
		allRules = append(allRules, rules...)
	}

	return allRules
}

func NewSpecificationMap(rules ...Rule) Specification {
	spec := &specificationMap{}
	spec.rules = make(map[string][]Rule)
	for _, rule := range rules {
		opString := ruleToOpString(rule)
		spec.rules[opString] = append(spec.rules[opString], rule)
	}

	return spec
}

func (s *specificationMap) GetRulesFor(state *st.State) []Rule {
	result := []Rule{}
	var opString string

	if state.Revision == common.R99_UnknownNextRevision || state.Status != st.Running {
		opString = "noOp"
	} else {
		op, err := state.Code.GetOperation(int(state.Pc))
		if err != nil {
			opString = "noOp"
		} else {
			opString = op.String()
		}
	}

	for _, rule := range s.rules[opString] {
		if valid, err := rule.Condition.Check(state); valid && err == nil {
			result = append(result, rule)
		}
	}

	return result
}

func ruleToOpString(rule Rule) string {
	var ruleString string
	opString := rule.Condition.String()

	reg := regexp.MustCompile(`code\[PC\] = ([^\s]+)`)
	substring := reg.FindAllStringSubmatch(opString, 1)
	if substring == nil {
		ruleString = "noOp"
		return ruleString
	}

	ruleString = strings.TrimPrefix(substring[0][0], "code[PC] = ")

	return ruleString
}
