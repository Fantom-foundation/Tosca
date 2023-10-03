package cti_test

import (
	"strings"
	"testing"

	"github.com/Fantom-foundation/Tosca/go/ct"
	"github.com/Fantom-foundation/Tosca/go/ct/cti"
	"github.com/Fantom-foundation/Tosca/go/ct/cts"
)

func TestComplianceTest_DerivedTestCases(t *testing.T) {
	spec := cts.Specification
	adapter := cti.CtAdapter{}
	for _, rule := range spec.GetRules() {
		rule := rule
		t.Run(rule.Name, func(t *testing.T) {
			t.Parallel()
			rule.EnumerateTestCases(func(state ct.State) {
				run(spec, adapter, state, t)
			})
		})
	}
}

func TestComplianceTest_RandomTestCases(t *testing.T) {
	const N = 10000

	spec := cts.Specification
	adapter := cti.CtAdapter{}
	for i := 0; i < N; i++ {
		state := ct.GetRandomState()
		run(spec, adapter, state, t)
	}
}

func run(spec ct.Specification, interpreter cti.CtAdapter, state ct.State, t *testing.T) {
	t.Helper()

	// Skip test where PC is pointing to Data (this are unreachable states).
	if ct.IsData(ct.Pc()).Check(state) {
		return
	}

	in := *state.Clone()

	// run on interpreter
	got, err := interpreter.StepN(in, 1)
	if err != nil {
		t.Fatalf("evaluation failed with error: %v", err)
	}

	// check rule for this in specification
	rules := spec.GetRulesFor(in)
	if len(rules) != 1 {
		t.Fatalf("missing rule for input state %v", in)
	}

	rule := rules[0]
	want := *state.Clone()
	want = rule.Effect.Apply(want)

	if !want.Equal(&got) {
		diffs := ct.Diff(&want, &got)
		t.Fatalf("Unexpected result state after rule '%s' with in %v, wanted %v, got %v, diffs:\n%v",
			rule.Name,
			&in,
			&want,
			&got,
			strings.Join(diffs, "\n\t"),
		)
	}
}
