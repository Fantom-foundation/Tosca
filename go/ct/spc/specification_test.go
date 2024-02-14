package spc

import (
	"testing"

	"pgregory.net/rand"

	"github.com/Fantom-foundation/Tosca/go/ct/gen"
	"github.com/Fantom-foundation/Tosca/go/ct/st"
)

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
		if len(rules) > 1 {
			s0 := state.Clone()
			rules[0].Effect.Apply(s0)
			for i := 1; i < len(rules)-1; i++ {
				s := state.Clone()
				rules[i].Effect.Apply(s)
				if !s.Eq(s0) {
					t.Fatalf("multiple conflicting rules for state %v: %v", state, rules)
				}
			}
		}
	}
}

func TestSpecification_EachRuleProducesAMatchingTestCase(t *testing.T) {
	for _, rule := range Spec.GetRules() {
		rule := rule
		t.Run(rule.Name, func(t *testing.T) {
			t.Parallel()
			hits := 0
			misses := 0
			rnd := rand.New(0)
			rule.EnumerateTestCases(rnd, func(state *st.State) error {
				match, err := rule.Condition.Check(state)
				if err != nil {
					t.Errorf("failed to check rule condition for %v: %v", rule.Name, err)
				} else if match {
					hits++
				} else {
					misses++
				}
				return nil
			})

			if hits == 0 {
				t.Errorf("no matching test case produced for rule %v", rule.Name)
			}
			if misses == 0 {
				t.Errorf("no non-matching test case produced for rule %v", rule.Name)
			}
		})
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
