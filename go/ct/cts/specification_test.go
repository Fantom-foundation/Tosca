package cts

import (
	"fmt"
	"testing"

	"github.com/Fantom-foundation/Tosca/go/ct"
)

func TestSpecification_RulesCoverTestCases(t *testing.T) {
	cases := Specification.GetTestCases()
	fmt.Printf("generated %d test cases\n", len(cases))
	for _, cur := range cases {
		rules := Specification.GetRulesFor(cur)
		if len(rules) == 0 {
			t.Fatalf("no specification for state %v", cur)
		}
		if len(rules) > 1 {
			t.Fatalf("multiple rules for state %v: %v", cur, rules)
		}
	}
}

func TestSpecification_RulesCoverRandomStates(t *testing.T) {
	const N = 1000

	for i := 0; i < N; i++ {
		state := ct.GetRandomState()
		rules := Specification.GetRulesFor(state)
		if len(rules) == 0 {
			t.Fatalf("no specification for state %v", state)
		}
		if len(rules) > 1 {
			t.Fatalf("multiple rules for state %v: %v", state, rules)
		}
	}
}
