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
	"errors"
	"fmt"
	"math"
	"regexp"
	"sync/atomic"
	"time"

	"github.com/Fantom-foundation/Tosca/go/ct"
	"github.com/Fantom-foundation/Tosca/go/ct/common"
	cliUtils "github.com/Fantom-foundation/Tosca/go/ct/driver/cli"
	"github.com/Fantom-foundation/Tosca/go/ct/rlz"
	"github.com/Fantom-foundation/Tosca/go/ct/spc"
	"github.com/Fantom-foundation/Tosca/go/ct/st"
	"github.com/Fantom-foundation/Tosca/go/interpreter/evmrs"
	"github.com/Fantom-foundation/Tosca/go/interpreter/evmzero"
	"github.com/Fantom-foundation/Tosca/go/interpreter/geth"
	"github.com/Fantom-foundation/Tosca/go/interpreter/lfvm"
	"github.com/Fantom-foundation/Tosca/go/tosca"
	"github.com/dsnet/golib/unitconv"
	"github.com/urfave/cli/v2"
	"golang.org/x/exp/maps"
)

var RunCmd = cliUtils.AddCommonFlags(cli.Command{
	Action:    doRun,
	Name:      "run",
	Usage:     "Run Conformance Tests on an EVM implementation",
	ArgsUsage: "<EVM>",
	Flags: []cli.Flag{
		cliUtils.FilterFlag,
		cliUtils.JobsFlag,
		cliUtils.SeedFlag,
		cliUtils.FullModeFlag, // < TODO: make every run a full mode once tests pass
		&cli.IntFlag{
			Name:  "max-errors",
			Usage: "aborts testing after the given number of issues",
			Value: 100,
		},
	},
})

var evms = map[string]ct.Evm{
	"lfvm":    lfvm.NewConformanceTestingTarget(),
	"geth":    geth.NewConformanceTestingTarget(),
	"evmzero": evmzero.NewConformanceTestingTarget(),
	"evmrs":   evmrs.NewConformanceTestingTarget(),
}

func doRun(context *cli.Context) error {
	defer common.DumpCppCoverageData()

	jobCount := cliUtils.JobsFlag.Fetch(context)
	seed := cliUtils.SeedFlag.Fetch(context)
	fullMode := cliUtils.FullModeFlag.Fetch(context)
	filter, err := cliUtils.FilterFlag.Fetch(context)
	if err != nil {
		return err
	}

	maxErrors := context.Int("max-errors")
	if maxErrors <= 0 {
		maxErrors = math.MaxInt
	}

	var evmIdentifier string
	if context.Args().Len() >= 1 {
		evmIdentifier = context.Args().Get(0)
	}
	evm, ok := evms[evmIdentifier]
	if !ok {
		return fmt.Errorf("invalid EVM identifier, use one of: %v", maps.Keys(evms))
	}

	issuesCollector := cliUtils.IssuesCollector{}
	var skippedCount atomic.Int32
	var numUnsupportedTests atomic.Int32

	printIssueCounts := func(relativeTime time.Duration, rate float64, current int64) {
		fmt.Printf(
			"[t=%4d:%02d] - Processing ~%s tests per second, total %d, skipped %d, found issues %d\n",
			int(relativeTime.Seconds())/60, int(relativeTime.Seconds())%60,
			unitconv.FormatPrefix(rate, unitconv.SI, 0), current, skippedCount.Load(), issuesCollector.NumIssues(),
		)
	}

	opRun := func(state *st.State) rlz.ConsumerResult {
		if issuesCollector.NumIssues() >= maxErrors {
			return rlz.ConsumeAbort
		}

		// TODO: program counter pointing to data not supported by LFVM
		// converter. Fix this.
		if evmIdentifier == "lfvm" && !state.Code.IsCode(int(state.Pc)) {
			skippedCount.Add(1)
			return rlz.ConsumeContinue
		}

		if err := runTest(state, evm, filter); err != nil {
			targetError := &tosca.ErrUnsupportedRevision{}
			if errors.As(err, &targetError) {
				numUnsupportedTests.Add(1)
				return rlz.ConsumeContinue
			}

			issuesCollector.AddIssue(state, fmt.Errorf("failed to process input state %v: %w", state, err))
		}

		return rlz.ConsumeContinue
	}

	fmt.Printf("Starting Conformance Tests with seed %d ...\n", seed)

	rules := spc.FilterRules(spc.Spec.GetRules(), filter)

	err = spc.ForEachState(rules, opRun, printIssueCounts, jobCount, seed, fullMode)
	if err != nil {
		return fmt.Errorf("error generating States: %w", err)
	}
	issues := issuesCollector.GetIssues()

	// Summarize the result.
	if skippedCount.Load() > 0 {
		fmt.Printf("Number of skipped tests: %d\n", skippedCount.Load())
	}

	if numUnsupportedTests.Load() > 0 {
		fmt.Printf("Number of tests with unsupported revision: %d\n", numUnsupportedTests.Load())
	}

	if len(issues) == 0 {
		fmt.Printf("All tests passed successfully!\n")
		return nil
	}

	err = issuesCollector.ExportIssues()
	if err != nil {
		return err
	}

	return fmt.Errorf("failed to pass %d test cases", len(issues))
}

// runTest runs a single test specified by the input state on the given EVM. The
// function returns an error in case the execution did not work as expected.
func runTest(input *st.State, evm ct.Evm, filter *regexp.Regexp) error {
	rules := spc.Spec.GetRulesFor(input)
	if len(rules) == 0 {
		return nil // < TODO: make this an error once the specification is complete
		//return fmt.Errorf("no rule found for state %v", input)
	}

	// filter out unwanted rules
	rules = spc.FilterRules(rules, filter)
	if len(rules) == 0 {
		return nil // < this is fine, the targeted rules are filtered out by the user
	}

	// TODO: enable optional rule consistency check
	rule := rules[0]
	expected := input.Clone()
	defer expected.Release()

	rule.Effect.Apply(expected)

	result, err := evm.StepN(input.Clone(), 1)
	if err != nil {
		return err
	}
	defer result.Release()

	if result.Eq(expected) {
		return nil
	}
	return fmt.Errorf(formatDiffForUser(input, result, expected, rule.Name))
}

func formatDiffForUser(input, result, expected *st.State, ruleName string) string {
	res := fmt.Sprintln("input state:", input)
	res += fmt.Sprintln("result state:", result)
	res += fmt.Sprintln("expected state:", expected)
	res += fmt.Sprintln("expectation defined by rule: ", ruleName)
	res += "Differences:\n"
	for _, diff := range result.Diff(expected) {
		res += fmt.Sprintf("\t%s\n", diff)
	}
	return res
}
