package main

import (
	"errors"
	"fmt"
	"regexp"
	"sort"

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
		&cli.StringFlag{
			Name:  "filter",
			Usage: "run only rules which name matches the given regex",
			Value: ".*",
		},
		&cli.BoolFlag{
			Name:  "list",
			Usage: "list all rules by name",
		},
		&cli.IntFlag{
			Name:  "max-errors",
			Usage: "maximum number of errors to display (0 displays all errors)",
			Value: 1,
		},
		&cli.Uint64Flag{
			Name:  "seed",
			Usage: "seed for the random number generator",
		},
	},
}

var evms = map[string]ct.Evm{
	"lfvm": lfvm.NewConformanceTestingTarget(),
}

func doRun(context *cli.Context) error {
	if context.Bool("list") {
		rules := spc.Spec.GetRules()
		sort.Slice(rules, func(i, j int) bool { return rules[i].Name < rules[j].Name })
		for _, rule := range rules {
			fmt.Println(rule.Name)
		}
		return nil
	}

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

	filter, err := regexp.Compile(context.String("filter"))
	if err != nil {
		return err
	}

	rnd := rand.New(context.Uint64("seed"))

	rules := spc.Spec.GetRules()
	for _, rule := range rules {
		if !filter.MatchString(rule.Name) {
			continue
		}

		fmt.Println(rule)
		errs := rule.EnumerateTestCases(rnd, func(state *st.State) error {
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
				errMsg := fmt.Sprintln(result.Diff(expected))
				errMsg += fmt.Sprintln("input state:", input)
				errMsg += fmt.Sprintln("result state:", result)
				errMsg += fmt.Sprintln("expected state:", expected)
				return fmt.Errorf(errMsg)
			}

			return nil
		})

		maxErrors := context.Int("max-errors")
		if maxErrors <= 0 {
			maxErrors = len(errs)
		} else {
			maxErrors = min(len(errs), maxErrors)
		}

		printErrors := errs[0:maxErrors]
		err := errors.Join(printErrors...)

		if err != nil {
			err = errors.Join(err, fmt.Errorf("total errors: %d", len(errs)))
			return err
		}
	}

	return nil
}
