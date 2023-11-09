package main

import (
	"fmt"

	"pgregory.net/rand"

	"github.com/Fantom-foundation/Tosca/go/ct"
	"github.com/Fantom-foundation/Tosca/go/ct/spc"
	"github.com/Fantom-foundation/Tosca/go/ct/st"
	"github.com/Fantom-foundation/Tosca/go/vm/lfvm"
	"github.com/urfave/cli/v2"
)

var RunCmd = cli.Command{
	Action:    doRun,
	Name:      "run",
	Usage:     "Run Conformance Tests on an EVM implementation",
	ArgsUsage: "<EVM>",
	Flags: []cli.Flag{
		&cli.Uint64Flag{
			Name:  "seed",
			Usage: "seed for the random number genertor",
		},
	},
}

var evms = map[string]ct.Evm{
	"lfvm": lfvm.NewConformanceTestingTarget(),
}

func doRun(context *cli.Context) error {
	var evmIdentifier string
	if context.Args().Len() >= 1 {
		evmIdentifier = context.Args().Get(0)
	}

	evm, ok := evms[evmIdentifier]
	if !ok {
		availableIdentifiers := make([]string, 0, len(evms))
		for k := range evms {
			availableIdentifiers = append(availableIdentifiers, k)
		}
		return fmt.Errorf("invalid EVM identifier, use one of: %v", availableIdentifiers)
	}

	rnd := rand.New(context.Uint64("seed"))

	rules := spc.Spec.GetRules()
	for _, rule := range rules {
		fmt.Println(rule)
		err := rule.EnumerateTestCases(rnd, func(state *st.State) error {
			if applies, err := rule.Condition.Check(state); !applies || err != nil {
				return err
			}

			// TODO: program counter pointing to data not supported by LFVM
			// converter.
			if !state.Code.IsCode(int(state.Pc)) {
				return nil // ignored
			}

			input := state.Clone()
			expected := state.Clone()
			rule.Effect.Apply(expected)

			result, err := evm.StepN(input.Clone(), 1)
			if err != nil {
				return err
			}

			if !result.Eq(expected) {
				fmt.Println(result.Diff(expected))
				fmt.Println("input state:", input)
				fmt.Println("result state:", result)
				fmt.Println("expected state:", expected)
				return fmt.Errorf("EVM not conformant")
			}

			return nil
		})
		if err != nil {
			return err
		}
	}

	return nil
}
