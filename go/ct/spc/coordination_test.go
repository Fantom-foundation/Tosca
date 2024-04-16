package spc

import (
	"math/rand"
	"regexp"
	"runtime"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Fantom-foundation/Tosca/go/ct/gen"
	"github.com/Fantom-foundation/Tosca/go/ct/rlz"
	"github.com/Fantom-foundation/Tosca/go/ct/st"
)

// Test Condition
type testCondition struct {
	fits      bool
	numStates int
}

func condition(fitsRule bool, numStates int) rlz.Condition {
	return &testCondition{fits: fitsRule, numStates: numStates}
}

func (e testCondition) Check(s *st.State) (bool, error) {
	return e.fits, nil
}

func (e testCondition) Restrict(generator *gen.StateGenerator) {
}

// the test condition produces exactly one test state per rule
func (e testCondition) EnumerateTestCases(generator *gen.StateGenerator, consume func(*gen.StateGenerator) rlz.ConsumerResult) rlz.ConsumerResult {
	for i := 0; i < e.numStates; i++ {
		g := generator.Clone()
		res := consume(g)
		if res == rlz.ConsumeAbort {
			return rlz.ConsumeAbort
		}
	}
	return rlz.ConsumeContinue
}

func (e testCondition) String() string {
	return "Test Condition"
}

func TestCoordination_GenerateMultipleStatesPerRule(t *testing.T) {
	numJobs := runtime.NumCPU()
	seed := 0
	var counter atomic.Int64
	opFunction := func(state *st.State) rlz.ConsumerResult {
		counter.Add(1)
		return rlz.ConsumeContinue
	}
	printFunction := func(time time.Duration, rate float64, current int64) {}

	statesPerRule := 15

	//var condition testCondition
	rules := []rlz.Rule{}
	testRule := rlz.Rule{
		Name:      "test_rule",
		Condition: condition(true, statesPerRule),
		Effect:    rlz.NoEffect(),
	}
	rules = append(rules, testRule)
	err := ForEachState(rules, opFunction, printFunction, numJobs, uint64(seed), false)
	if err != nil {
		t.Errorf("Unexpected error during state generation %v", err)
	}

	if counter.Load() != int64(statesPerRule) {
		t.Errorf("unexpected number of evaluated states: %d", counter.Load())
	}
}

func TestCoordination_AllRulesAreEnumerated(t *testing.T) {
	numJobs := runtime.NumCPU()
	seed := 0
	var counter atomic.Int64
	opFunction := func(state *st.State) rlz.ConsumerResult {
		counter.Add(1)
		return rlz.ConsumeContinue
	}
	printFunction := func(time time.Duration, rate float64, current int64) {}

	//var condition testCondition
	rules := []rlz.Rule{}
	testRuleFit := rlz.Rule{
		Name:      "test_rule",
		Condition: condition(true, 1),
		Effect:    rlz.NoEffect(),
	}
	testRuleNoFit := rlz.Rule{
		Name:      "test_rule",
		Condition: condition(false, 1),
		Effect:    rlz.NoEffect(),
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

func TestCoordination_FilterRules(t *testing.T) {
	filters := []string{"add", "sub", "mul", "copy", "call"}
	for _, subString := range filters {
		filter, err := regexp.Compile(subString)
		if err != nil {
			t.Error("regular expression not compilable")
		}
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
	}
}

func TestCoordination_EmptyRules(t *testing.T) {
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

func TestCoordination_RightNumberOfGoroutinesIsStarted(t *testing.T) {
	numJobs := 4
	seed := 0
	fullMode := false
	filter := regexp.MustCompile(".*")

	opFunction := func(state *st.State) rlz.ConsumerResult {

		// 3 goroutines (sweeper, scavenger and finalizer) are started by default
		if want, got := numJobs*2+3, runtime.NumGoroutine(); want != got {
			t.Errorf("wrong number of go routines during execution: want %d, got %d", want, got)
		}
		return rlz.ConsumeAbort
	}
	printFunction := func(time time.Duration, rate float64, current int64) {}

	rules := FilterRules(Spec.GetRules(), filter)
	ForEachState(rules, opFunction, printFunction, numJobs, uint64(seed), fullMode)

}
