package ct_test

import (
	"testing"

	"pgregory.net/rand"

	"github.com/Fantom-foundation/Tosca/go/ct/spc"
	"github.com/Fantom-foundation/Tosca/go/ct/st"
	"github.com/Fantom-foundation/Tosca/go/vm/lfvm"
)

func TestSpecOnLfvm(t *testing.T) {
	evm := lfvm.NewConformanceTestingTarget()

	rules := spc.Spec.GetRules()
	for _, rule := range rules {
		rule := rule
		t.Run(rule.Name, func(t *testing.T) {
			rule.EnumerateTestCases(rand.New(0), func(state *st.State) {
				if applies, err := rule.Condition.Check(state); !applies || err != nil {
					return
				}

				// TODO: program counter pointing to data not supported by LFVM
				// converter.
				if !state.Code.IsCode(int(state.Pc)) {
					return
				}

				input := state.Clone()
				expected := state.Clone()
				rule.Effect.Apply(expected)

				result, err := evm.StepN(state, 1)
				if err != nil {
					t.Fatal(err)
				}

				if !result.Eq(expected) {
					t.Error(result.Diff(expected))
					t.Error("input state:", input)
					t.Error("result state:", result)
					t.Error("expected state:", expected)
					t.FailNow()
				}
			})
		})
	}
}
