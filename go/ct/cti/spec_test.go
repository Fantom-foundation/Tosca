package cti_test

import (
	"fmt"
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
				if err := run(spec, adapter, state, t); err != nil {
					t.Fatalf("Failed test case: %v", err)
				}
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
		if err := run(spec, adapter, state, t); err != nil {
			t.Fatalf("Failed test case: %v", err)
		}
	}
}

func run(spec ct.Specification, interpreter cti.CtAdapter, state ct.State, t *testing.T) error {
	t.Helper()

	// Skip test where PC is pointing to Data (this are unreachable states).
	if ct.IsData(ct.Pc()).Check(state) {
		return nil
	}

	// run on interpreter
	in := *state.Clone()
	got, err := interpreter.StepN(in, 1)
	if err != nil {
		return fmt.Errorf("evaluation failed with error: %v", err)
	}

	// check rule for this in specification
	in = *state.Clone()
	rules := spec.GetRulesFor(in)
	if len(rules) != 1 {
		return fmt.Errorf("missing rule for input state %v", in)
	}

	rule := rules[0]
	want := rule.Effect.Apply(in)

	if !want.Equal(&got) {
		diffs := ct.Diff(&want, &got)
		return fmt.Errorf("Unexpected result state after rule '%s' with in %v, wanted %v, got %v, diffs:\n%v",
			rule.Name,
			&state,
			&want,
			&got,
			strings.Join(diffs, "\n\t"),
		)
	}
	return nil
}
