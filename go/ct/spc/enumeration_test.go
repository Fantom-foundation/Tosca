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
	"math/rand"
	"regexp"
	"runtime"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Fantom-foundation/Tosca/go/ct/gen"
	"github.com/Fantom-foundation/Tosca/go/ct/rlz"
	"github.com/Fantom-foundation/Tosca/go/ct/st"
)

// Test condition used for state enumeration tests
type testCondition struct {
	fits      bool // wether the enumerated test states fit the rule
	numStates int  // the number of test cases to be generated
}

func condition(fitsRule bool, numStates int) rlz.Condition {
	return &testCondition{fits: fitsRule, numStates: numStates}
}

func (e testCondition) Check(s *st.State) (bool, error) {
	return e.fits, nil
}

func (e testCondition) Restrict(generator *gen.StateGenerator) {
}

// GetTestValues produces test values for the specified number of test states.
func (e testCondition) GetTestValues() []rlz.TestValue {
	res := []rlz.TestValue{}
	property := rlz.Property("test")
	domain := rlz.StackSize().Domain()
	for i := 0; i < e.numStates; i++ {
		res = append(res, rlz.NewTestValue(property, domain, i, func(*gen.StateGenerator, int) {}))
	}
	return res
}

func (e testCondition) String() string {
	return "Test Condition"
}

func TestEnumeration_GenerateMultipleStatesPerRule(t *testing.T) {
	numJobs := runtime.NumCPU()
	seed := 0
	var counter atomic.Int64
	opFunction := func(state *st.State) rlz.ConsumerResult {
		counter.Add(1)
		return rlz.ConsumeContinue
	}
	printFunction := func(time time.Duration, rate float64, current int64) {}

	statesPerRule := 15
	rules := []rlz.Rule{{Condition: condition(true, statesPerRule)}}

	err := ForEachState(rules, opFunction, printFunction, numJobs, uint64(seed), false)
	if err != nil {
		t.Errorf("Unexpected error during state generation %v", err)
	}

	if counter.Load() != int64(statesPerRule) {
		t.Errorf("unexpected number of evaluated states: %d", counter.Load())
	}
}

func TestEnumeration_DisabledFullModeFiltersNonMatchingRules(t *testing.T) {
	numJobs := runtime.NumCPU()
	seed := 0
	var counter atomic.Int64
	opFunction := func(state *st.State) rlz.ConsumerResult {
		counter.Add(1)
		return rlz.ConsumeContinue
	}
	printFunction := func(time time.Duration, rate float64, current int64) {}

	rules := []rlz.Rule{}
	testRuleFit := rlz.Rule{
		Condition: condition(true, 1),
	}
	testRuleNoFit := rlz.Rule{
		Condition: condition(false, 1),
	}

	for i := 0; i < rand.Intn(42); i++ {
		rules = append(rules, testRuleFit)
		rules = append(rules, testRuleNoFit)
	}
	numRules := len(rules)

	err := ForEachState(rules, opFunction, printFunction, numJobs, uint64(seed), false)
	if err != nil {
		t.Errorf("Unexpected error during state generation %v", err)
	}

	if counter.Load() != int64(numRules)/2 {
		t.Errorf("unexpected number of evaluated states: %d", counter.Load())
	}

	counter.Store(0)
	err = ForEachState(rules, opFunction, printFunction, numJobs, uint64(seed), true)
	if err != nil {
		t.Errorf("Unexpected error during state generation %v", err)
	}

	if counter.Load() != int64(numRules) {
		t.Errorf("wrong number of evaluated states in full mode: %d vs. %d", counter.Load(), numRules)
	}
}

func TestEnumeration_AbortedEnumeration(t *testing.T) {
	numJobs := runtime.NumCPU()
	seed := 0
	numRules := 42
	numStates := 42
	var counterContinue atomic.Int64
	var counterAbort atomic.Int64
	opFunctionContinue := func(state *st.State) rlz.ConsumerResult {
		counterContinue.Add(1)
		return rlz.ConsumeContinue
	}
	opFunctionAbort := func(state *st.State) rlz.ConsumerResult {
		counterAbort.Add(1)
		return rlz.ConsumeAbort
	}
	printFunction := func(time time.Duration, rate float64, current int64) {}
	rules := []rlz.Rule{}
	testRule := rlz.Rule{
		Condition: condition(true, numStates),
	}
	for i := 0; i < numRules; i++ {
		rules = append(rules, testRule)
	}
	err := ForEachState(rules, opFunctionContinue, printFunction, numJobs, uint64(seed), true)
	if err != nil {
		t.Errorf("Unexpected error during state generation %v", err)
	}

	err = ForEachState(rules, opFunctionAbort, printFunction, numJobs, uint64(seed), true)
	if err != nil {
		t.Errorf("Unexpected error during state generation %v", err)
	}

	if int(counterContinue.Load()) != numRules*numStates {
		t.Errorf("wrong number of generated test cases")
	}

	if counterAbort.Load() >= counterContinue.Load() {
		t.Errorf("state enumeration did not abort correctly, number of evaluated states %d", counterAbort.Load())
	}

}

func TestEnumeration_FilterRules(t *testing.T) {
	filters := []string{"add", "sub", "mul", "copy", "call"}
	for _, subString := range filters {
		filter := regexp.MustCompile(subString)
		rules := Spec.GetRules()
		rules = FilterRules(rules, filter)

		ruleNames := make([]string, 0, len(rules))
		for _, rule := range rules {
			ruleNames = append(ruleNames, rule.Name)
		}

		for _, name := range ruleNames {
			if !strings.Contains(name, subString) {
				t.Errorf("rules not filtered correctly %v", name)
			}
		}

		for _, rule := range Spec.GetRules() {
			if strings.Contains(rule.Name, subString) {
				if !slices.Contains(ruleNames, rule.Name) {
					t.Errorf("rule %v is missing in filtered rules", rule.Name)
				}
			}
		}
	}
}

func TestEnumeration_EmptyRules(t *testing.T) {
	numJobs := 1
	seed := 0
	fullMode := false
	opFunction := func(state *st.State) rlz.ConsumerResult {
		return rlz.ConsumeContinue
	}
	printFunction := func(time time.Duration, rate float64, current int64) {}
	rules := []rlz.Rule{}
	err := ForEachState(rules, opFunction, printFunction, numJobs, uint64(seed), fullMode)
	if err != nil {
		t.Errorf("Unexpected error during state generation %v", err)
	}
}

func TestEnumeration_RightNumberOfGoroutinesIsStarted(t *testing.T) {
	numJobs := runtime.NumCPU()
	seed := 0
	fullMode := false

	activeJobs := 0
	limitReached := false
	condition := sync.NewCond(&sync.Mutex{})
	opFunction := func(state *st.State) rlz.ConsumerResult {
		// make sure that numJobs are active at the same time
		condition.L.Lock()
		defer condition.L.Unlock()
		if limitReached {
			return rlz.ConsumeAbort
		}

		activeJobs++
		condition.Broadcast()
		for activeJobs < numJobs {
			condition.Wait()
		}
		limitReached = true
		return rlz.ConsumeAbort
	}

	printFunction := func(time time.Duration, rate float64, current int64) {}
	err := ForEachState(Spec.GetRules(), opFunction, printFunction, numJobs, uint64(seed), fullMode)
	if err != nil {
		t.Errorf("Unexpected error in ForEachState %v", err)
	}

	if activeJobs != numJobs {
		t.Errorf("unexpected number of active jobs, wanted %d, got %d", numJobs, activeJobs)
	}
}
