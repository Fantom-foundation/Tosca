// Copyright (c) 2024 Fantom Foundation
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at fantom.foundation/bsl11.
//
// Change Date: 2028-4-16
//
// On the date above, in accordance with the Business Source License, use of
// this software will be governed by the GNU Lesser General Public License v3.

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	cliUtils "github.com/Fantom-foundation/Tosca/go/ct/driver/cli"
	"github.com/Fantom-foundation/Tosca/go/ct/spc"
	"github.com/Fantom-foundation/Tosca/go/ct/st"
	"github.com/urfave/cli/v2"
	"golang.org/x/exp/maps"
)

var RegressionsCmd = cliUtils.AddCommonFlags(cli.Command{
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
})

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

			// TODO: do not only skip state but change 'pc_on_data_is_ignored' rule to anyEffect, see #954
			// Pc on data is not supported
			if !state.Code.IsCode(int(state.Pc)) {
				continue
			}

			result, err := evm.StepN(input.Clone(), 1)
			if err != nil {
				fmt.Printf("Failed to evaluate rule %v: %v\n", rule, err)
				continue
			}

			if !result.Eq(expected) {
				fmt.Printf("Failed to evaluate rule %v: %v\n", rule, formatDiffForUser(input, result, expected, rule.Name))
				continue
			}

			evaluationCount++

			fmt.Printf("OK: (rules evaluated: %v) %v (%v)\n", evaluationCount, rule, time.Since(tstart).Round(10*time.Millisecond))
		}
	}

	return nil
}
