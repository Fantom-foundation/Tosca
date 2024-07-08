// Copyright (c) 2024 Fantom Foundation
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at fantom.foundation/bsl11.
//
// Change Date: 2028-4-16
//
// On the date above, in accordance with the Business Source License, use of
// this software will be governed by the GNU Lesser General Public License v3.

package spc

import (
	"math"
	"regexp"
	"slices"
	"strings"
	"testing"

	"pgregory.net/rand"

	"github.com/Fantom-foundation/Tosca/go/ct/gen"
	"github.com/Fantom-foundation/Tosca/go/ct/rlz"
	"github.com/Fantom-foundation/Tosca/go/ct/st"
	"github.com/Fantom-foundation/Tosca/go/tosca"
)

func TestSpecification_SpecificationIsSound(t *testing.T) {
	const N = 100000

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

func TestSpecification_SpecificationIsComplete(t *testing.T) {
	const N = 100000
	rnd := rand.New(0)
	generator := gen.NewStateGenerator()
	for i := 0; i < N; i++ {
		state, err := generator.Generate(rnd)
		if err != nil {
			t.Errorf("failed to generate a random state: %v", err)
		}
		rules := Spec.GetRulesFor(state)
		if len(rules) == 0 {
			t.Fatalf("no rule found for \n%v", state)
		}
	}
}

func TestSpecification_EachRuleProducesAMatchingTestCase(t *testing.T) {
	for _, rule := range Spec.GetRules() {
		rule := rule
		t.Run(rule.Name, func(t *testing.T) {
			t.Parallel()
			rnd := rand.New(0)
			hits := 0
			misses := 0
			rule.EnumerateTestCases(rnd, func(state *st.State) rlz.ConsumerResult {
				match, err := rule.Condition.Check(state)
				if err != nil {
					t.Errorf("failed to check rule condition for %v: %v", rule.Name, err)
				} else if match {
					hits++
				} else {
					misses++
				}
				if hits > 0 && misses > 0 {
					return rlz.ConsumeAbort
				}
				return rlz.ConsumeContinue
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

// TestSpecification_SpecifiedRuleProducesMatchingTestCases is used to verify stability of specific rules.
func TestSpecification_SpecifiedRuleProducesMatchingTestCases(t *testing.T) {

	allRules := Spec.GetRules()

	tests := []string{
		"sstore_with_too_little_gas_",
		"pc_on_data_is_ignored",
		"unknown_revision_is_end",
	}

	for _, ruleName := range tests {
		t.Run(ruleName, func(t *testing.T) {

			rnd := rand.New(0)
			filter := regexp.MustCompile(ruleName)
			rules := FilterRules(allRules, filter)
			if len(rules) == 0 {
				t.Fatalf("no rule found for filter %v", filter)
			}

			rule := rules[0]
			gen := gen.NewStateGenerator()
			rule.Condition.Restrict(gen)
			for i := 0; i < 10000; i++ {
				state, err := gen.Generate(rnd)
				if err != nil {
					t.Fatalf("failed to generate a random state at iteration %v: %v", i, err)
				}
				pass, err := rule.Condition.Check(state)
				if err != nil {
					t.Fatalf("at iteration %v failed to check rule condition for %v: %v", i, rule.Name, err)
				}
				if !pass {
					t.Fatalf("at iteration %v State %v \nFailed for conditions: %v\n", i, state, rule.Condition)
				}
			}
		})
	}
}

func TestSpecificationMap_NumberOfTests(t *testing.T) {
	rulesMap := Spec.GetRules()
	rules := getAllRules()

	if len(rulesMap) != len(rules) {
		t.Errorf("Different number of rules: %d vs. %d", len(rules), len(rulesMap))
	}
}

func listGetRulesFor(state *st.State) []rlz.Rule {
	rules := getAllRules()
	result := []rlz.Rule{}
	for _, rule := range rules {
		if valid, err := rule.Condition.Check(state); valid && err == nil {
			result = append(result, rule)
		}
	}
	return result
}

func TestSpecificationMap_SameRulesPerOperation(t *testing.T) {
	const N = 1000

	rnd := rand.New(0)
	generator := gen.NewStateGenerator()

	for i := 0; i < N; i++ {
		state, err := generator.Generate(rnd)
		if err != nil {
			t.Fatalf("failed building state: %v", err)
		}

		op, _ := state.Code.GetOperation(int(state.Pc))
		allRulesForState := listGetRulesFor(state)
		rulesFromMap := Spec.GetRulesFor(state)

		if len(allRulesForState) != len(rulesFromMap) {
			t.Errorf("different number of rules for %s: %d vs %d", op, len(allRulesForState), len(rulesFromMap))
		}

		for _, rule := range allRulesForState {
			found := false
			for _, ruleFromMap := range rulesFromMap {
				if rule.Name == ruleFromMap.Name {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("rule %v not found in map", rule.Name)
			}
		}
	}
}

func TestSpecification_OperationNotExecutedIfNotRunning(t *testing.T) {
	// list of known no operations
	knownNoOps := []string{"stopped_is_end", "reverted_is_end", "failed_is_end", "unknown_revision_is_end"}
	statusFreeRules := []string{"unknown_revision_is_end", "pc_on_data_is_ignored"}

	rules := getAllRules()
	for _, rule := range rules {
		opString := rule.Condition.String()
		reg := regexp.MustCompile(`status = ([^\s]+)`)
		substring := reg.FindAllStringSubmatch(opString, 1)

		if len(substring) > 0 {
			statusString := strings.TrimPrefix(substring[0][0], "status = ")

			if statusString != "running" && !slices.Contains(knownNoOps, rule.Name) {
				if ruleToOpString(rule) != "noOp" {
					t.Errorf("Rule has code operation constrain but no status")
				}
				t.Errorf("Rule is not an operation but not in list of known no operations")
			}
		} else {
			if !slices.Contains(statusFreeRules, rule.Name) {
				t.Errorf("Status not defined in rule %s", rule.Name)
			}
		}
	}
}

func TestSpecification_AtMostOneCodeAtPC(t *testing.T) {
	rules := getAllRules()

	for _, rule := range rules {
		opString := rule.Condition.String()

		reg := regexp.MustCompile(`code\[PC\] = ([^\s]+)`)
		substring := reg.FindAllStringSubmatch(opString, 1)
		if len(substring) > 1 {
			t.Errorf("It is not possible to have multiple code constrains on the same code location")
		}
	}
}

// TODO: re-enable this test when runtime is not an issue anymore.
// note: before commiting run it locally where timeout is not an issue.
func Specification_NumberOfTestCasesMatchesRuleInfo(t *testing.T) {
	rules := getAllRules()

	for _, rule := range rules {
		rule := rule
		t.Run(rule.Name, func(t *testing.T) {
			t.Parallel()

			info := rule.GetTestCaseEnumerationInfo()

			counter := 0
			rand := rand.New(0)
			rule.EnumerateTestCases(rand, func(*st.State) rlz.ConsumerResult {
				counter++
				return rlz.ConsumeContinue
			})

			if got, limit := counter, info.TotalNumberOfCases(); got > limit {
				t.Errorf("inconsistent number of test cases, got %d, limit %d", got, limit)
			}
		})
	}
}

func BenchmarkSpecification_GetState(b *testing.B) {
	N := 10000
	rnd := rand.New(0)
	generator := gen.NewStateGenerator()

	states := make([]*st.State, 0, N)
	for i := 0; i < N; i++ {
		state, err := generator.Generate(rnd)
		if err != nil {
			b.Fatalf("failed building state: %v", err)
		}
		states = append(states, state)
	}

	b.ResetTimer()
	for _, state := range states {
		Spec.GetRulesFor(state)
	}
}

func BenchmarkSpecification_GetAllRules(b *testing.B) {
	for i := 0; i < b.N; i++ {
		Spec.GetRules()
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

func TestSumWithOverflow(t *testing.T) {
	max := tosca.Gas(math.MaxInt64)
	tests := map[string]struct {
		inputs   []tosca.Gas
		result   tosca.Gas
		overflow bool
	}{
		"nil": {
			inputs: nil,
			result: 0,
		},
		"empty": {
			inputs: []tosca.Gas{},
			result: 0,
		},
		"single": {
			inputs: []tosca.Gas{12},
			result: 12,
		},
		"single_max": {
			inputs: []tosca.Gas{max},
			result: max,
		},
		"pair_without_overflow": {
			inputs: []tosca.Gas{1, 2},
			result: 3,
		},
		"pair_with_overflow": {
			inputs:   []tosca.Gas{max - 1, 2},
			overflow: true,
		},
		"triple_without_overflow": {
			inputs: []tosca.Gas{1, 2, 3},
			result: 6,
		},
		"triple_with_overflow_in_first_pair": {
			inputs:   []tosca.Gas{max - 1, 2, 4},
			overflow: true,
		},
		"triple_with_overflow_with_last_element": {
			inputs:   []tosca.Gas{max - 3, 2, 4},
			overflow: true,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			got, overflow := sumWithOverflow(test.inputs...)
			if test.overflow {
				if !overflow {
					t.Errorf("expected sum to overflow, but it did not")
				}
			} else if want := test.result; want != got {
				t.Errorf("unexpected result, wanted %d, got %d", want, got)
			}
		})
	}
}
