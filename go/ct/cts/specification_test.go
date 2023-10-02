package cts

import (
	"testing"

	"github.com/Fantom-foundation/Tosca/go/ct"
)

func TestSpecification_RulesCoverTestCases(t *testing.T) {
	rules := Specification.GetRules()
	for _, rule := range rules {
		t.Run(rule.Name, func(t *testing.T) {
			for _, cur := range ct.GetTestSamples(rule) {
				rules := Specification.GetRulesFor(cur)
				if len(rules) == 0 {
					t.Fatalf("no specification for state %v", &cur)
				}
				if len(rules) > 1 {
					t.Fatalf("multiple rules for state %v: %v", &cur, rules)
				}
			}
		})
	}
}

func TestSpecification_RulesCoverRandomStates(t *testing.T) {
	const N = 10000

	for i := 0; i < N; i++ {
		state := ct.GetRandomState()
		rules := Specification.GetRulesFor(state)
		if len(rules) == 0 {
			t.Fatalf("no specification for state %v", &state)
		}
		if len(rules) > 1 {
			t.Fatalf("multiple rules for state %v: %v", &state, rules)
		}
	}
}
