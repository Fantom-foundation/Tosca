package main

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/Fantom-foundation/Tosca/go/ct/spc"
	"github.com/Fantom-foundation/Tosca/go/ct/st"
	"github.com/urfave/cli/v2"
	"golang.org/x/exp/maps"
)

var RegressionsCmd = cli.Command{
	Action:    doRegressionTests,
	Name:      "regressions",
	Usage:     "Run Conformance Tests on regression test inputs on an EVM implementation",
	ArgsUsage: "<EVM>",
	Flags: []cli.Flag{
		&cli.StringSliceFlag{
			Name:  "input",
			Usage: "run given input file, or all files in the given directory (recursively)",
			Value: cli.NewStringSlice("./regression_inputs"),
		},
	},
}

func enumerateInputs(inputs []string) ([]string, error) {
	var inputFiles []string

	for _, input := range inputs {
		path, err := filepath.Abs(input)
		if err != nil {
			return nil, err
		}

		stat, err := os.Stat(path)
		if err != nil {
			return nil, err
		}
		if !stat.IsDir() {
			inputFiles = append(inputFiles, path)
			continue
		}

		entries, err := os.ReadDir(input)
		if err != nil {
			return nil, err
		}

		for _, entry := range entries {
			filePath := filepath.Join(path, entry.Name())
			if entry.IsDir() {
				recInputs, err := enumerateInputs([]string{filePath})
				if err != nil {
					return nil, err
				}
				inputFiles = append(inputFiles, recInputs...)
			} else {
				inputFiles = append(inputFiles, filePath)
			}
		}
	}

	return inputFiles, nil
}

func doRegressionTests(context *cli.Context) error {
	var evmIdentifier string
	if context.Args().Len() >= 1 {
		evmIdentifier = context.Args().Get(0)
	}

	evm, ok := evms[evmIdentifier]
	if !ok {
		return fmt.Errorf("invalid EVM identifier, use one of: %v", maps.Keys(evms))
	}

	inputs := context.StringSlice("input")
	inputs, err := enumerateInputs(inputs)
	if err != nil {
		return err
	}

	for _, input := range inputs {
		state, err := st.ImportStateJSON(input)
		if err != nil {
			fmt.Printf("Failed to import state from %v: %v\n", input, err)
			continue
		}

		rules := spc.Spec.GetRulesFor(state)

		if len(rules) == 0 {
			fmt.Printf("No rules apply for input %v\n", input)
			continue
		}

		evaluationCount := 0

		for _, rule := range rules {
			tstart := time.Now()

			input := state.Clone()
			expected := state.Clone()
			rule.Effect.Apply(expected)

			result, err := evm.StepN(input.Clone(), 1)
			if err != nil {
				fmt.Printf("Failed to evaluate rule %v: %v\n", rule, err)
				continue
			}

			if !result.Eq(expected) {
				errMsg := fmt.Sprintln(result.Diff(expected))
				errMsg += fmt.Sprintln("input state:", input)
				errMsg += fmt.Sprintln("result state:", result)
				errMsg += fmt.Sprintln("expected state:", expected)
				fmt.Printf("Failed to evaluate rule %v: %v\n", rule, errMsg)
				continue
			}

			evaluationCount++

			fmt.Printf("OK: (rules evaluated: %v) %v (%v)\n", evaluationCount, rule, time.Since(tstart).Round(10*time.Millisecond))
		}
	}

	return nil
}
