package ct_test

import (
	"testing"

	"pgregory.net/rand"

	"github.com/Fantom-foundation/Tosca/go/ct"
	. "github.com/Fantom-foundation/Tosca/go/ct/common"
	"github.com/Fantom-foundation/Tosca/go/ct/gen"
	. "github.com/Fantom-foundation/Tosca/go/ct/rlz"
	"github.com/Fantom-foundation/Tosca/go/ct/spc"
	"github.com/Fantom-foundation/Tosca/go/ct/st"
	"github.com/Fantom-foundation/Tosca/go/vm"
	"github.com/Fantom-foundation/Tosca/go/vm/evmzero"
	"github.com/Fantom-foundation/Tosca/go/vm/lfvm"
)

var evms = map[string]ct.Evm{
	// "geth":    vm.NewConformanceTestingTarget(), // < TODO: fix and reenable
	"lfvm":    lfvm.NewConformanceTestingTarget(),
	"evmzero": evmzero.NewConformanceTestingTarget(),
}

func TestCt_ExplicitCases(t *testing.T) {
	tests := map[string]Condition{
		"jump_to_2^32": And(
			Eq(Status(), st.Running),
			Eq(Op(Pc()), JUMP),
			Eq(Op(Constant(NewU256(0))), JUMPDEST),
			Eq(Param(0), NewU256(1<<32)),
			Ge(Gas(), vm.Gas(8)),
		),
		"jumpi_to_2^32": And(
			Eq(Status(), st.Running),
			Eq(Op(Pc()), JUMPI),
			Eq(Op(Constant(NewU256(0))), JUMPDEST),
			Eq(Param(0), NewU256(1<<32)),
			Ne(Param(1), NewU256(0)),
			Ge(Gas(), vm.Gas(10)),
		),
	}

	random := rand.New(0)
	for name, condition := range tests {
		t.Run(name, func(t *testing.T) {
			generator := gen.NewStateGenerator()
			condition.Restrict(generator)

			input, err := generator.Generate(random)
			if err != nil {
				t.Fatalf("failed to generate satisfying state: %v", err)
			}
			if ok, err := condition.Check(input); !ok || err != nil {
				t.Fatalf("failed to generate satisfying state: %v, %v, %v", input, ok, err)
			}

			rules := spc.Spec.GetRulesFor(input)
			if len(rules) == 0 {
				t.Fatalf("no rule for test state: %v", input)
			}

			output := input.Clone()
			rules[0].Effect.Apply(output)

			for name, evm := range evms {
				t.Run(name, func(t *testing.T) {
					res, err := evm.StepN(input.Clone(), 1)
					if err != nil {
						t.Fatalf("failed to run test case: %v", err)
					}
					if !res.Eq(output) {
						t.Errorf("Invalid result, wanted %v, got %v", output, res)
						for _, diff := range output.Diff(res) {
							t.Errorf(diff)
						}
					}
				})
			}

		})
	}
}
