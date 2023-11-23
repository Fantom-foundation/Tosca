package spc

import (
	"testing"

	"pgregory.net/rand"

	"github.com/Fantom-foundation/Tosca/go/ct/gen"
	"github.com/Fantom-foundation/Tosca/go/ct/st"
)

func TestSpecification_RulesCoverTestCases(t *testing.T) {
	rules := Spec.GetRules()
	for _, rule := range rules {
		rule := rule
		t.Run(rule.Name, func(t *testing.T) {
			t.Parallel()

			// TODO: For now, check that we get at least one rule matching for
			// the full set of test cases (and at most one rule for every test
			// case). Later we'll enforce that exactly one rule applies to every
			// single test case.
			atLeastOne := false

			rule.EnumerateTestCases(rand.New(0), func(state *st.State) error {
				rules := Spec.GetRulesFor(state)
				if len(rules) > 0 {
					atLeastOne = true
				}
				if len(rules) > 1 {
					t.Fatalf("multiple rules for state %v: %v", state, rules)
				}
				return nil
			})

			if !atLeastOne {
				t.Fatalf("No rule matches any of the generated test cases")
			}
		})
	}
}

func TestSpecification_RulesCoverRandomStates(t *testing.T) {
	const N = 10000

	rnd := rand.New(0)
	generator := gen.NewStateGenerator()

	for i := 0; i < N; i++ {
		state, err := generator.Generate(rnd)
		if err != nil {
			t.Fatalf("failed building state: %v", err)
		}

		rules := Spec.GetRulesFor(state)

		// TODO: Enforce that exactly one rule applies
		if len(rules) > 1 {
			t.Fatalf("multiple rules for state %v: %v", &state, rules)
		}
	}
}

func BenchmarkSpecification_RulesConditionCheck(b *testing.B) {
	state, err := gen.NewStateGenerator().Generate(rand.New(0))
	if err != nil {
		b.Fatal(err)
	}

	for i := 0; i < b.N; i++ {
		rules := Spec.GetRules()
		for _, rule := range rules {
			rule.Condition.Check(state)
		}
	}
}
