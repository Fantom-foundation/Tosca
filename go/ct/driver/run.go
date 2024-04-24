//
// Copyright (c) 2024 Fantom Foundation
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at fantom.foundation/bsl11.
//
// Change Date: 2028-4-16
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by the GNU Lesser General Public Licence v3
//

package main

import (
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"runtime/pprof"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Fantom-foundation/Tosca/go/ct"
	"github.com/Fantom-foundation/Tosca/go/ct/rlz"
	"github.com/Fantom-foundation/Tosca/go/ct/spc"
	"github.com/Fantom-foundation/Tosca/go/ct/st"
	"github.com/Fantom-foundation/Tosca/go/vm"
	"github.com/Fantom-foundation/Tosca/go/vm/evmzero"
	"github.com/Fantom-foundation/Tosca/go/vm/geth"
	"github.com/Fantom-foundation/Tosca/go/vm/lfvm"
	"github.com/dsnet/golib/unitconv"
	"github.com/urfave/cli/v2"
	"golang.org/x/exp/maps"
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
		&cli.IntFlag{
			Name:  "jobs",
			Usage: "number of jobs run simultaneously",
			Value: runtime.NumCPU(),
		},
		&cli.IntFlag{
			Name:  "max-errors",
			Usage: "aborts testing after the given number of issues",
			Value: -1,
		},
		&cli.Uint64Flag{
			Name:  "seed",
			Usage: "seed for the random number generator",
		},
		&cli.StringFlag{
			Name:  "cpuprofile",
			Usage: "store CPU profile in the provided filename",
		},
		&cli.BoolFlag{ // < TODO: make every run a full mode once tests pass
			Name:  "full-mode",
			Usage: "if enabled, test cases targeting rules other than the one generating the case will be executed",
		},
	},
}

var evms = map[string]ct.Evm{
	"lfvm":    lfvm.NewConformanceTestingTarget(),
	"geth":    geth.NewConformanceTestingTarget(),
	"evmzero": evmzero.NewConformanceTestingTarget(),
}

func doRun(context *cli.Context) error {
	if cpuprofileFilename := context.String("cpuprofile"); cpuprofileFilename != "" {
		f, err := os.Create(cpuprofileFilename)
		if err != nil {
			return fmt.Errorf("could not create CPU profile: %w", err)
		}
		if err := pprof.StartCPUProfile(f); err != nil {
			return fmt.Errorf("could not start CPU profile: %w", err)
		}
		defer pprof.StopCPUProfile()
	}

	var evmIdentifier string
	if context.Args().Len() >= 1 {
		evmIdentifier = context.Args().Get(0)
	}

	evm, ok := evms[evmIdentifier]
	if !ok {
		return fmt.Errorf("invalid EVM identifier, use one of: %v", maps.Keys(evms))
	}

	filter, err := regexp.Compile(context.String("filter"))
	if err != nil {
		return err
	}

	jobCount := context.Int("jobs")
	if jobCount <= 0 {
		jobCount = runtime.NumCPU()
	}

	seed := context.Uint64("seed")
	maxErrors := context.Int("max-errors")
	if maxErrors <= 0 {
		maxErrors = math.MaxInt
	}
	fullMode := context.Bool("full-mode")

	issuesCollector := issuesCollector{}
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

		if err := runTest(state, evm, filter, &numUnsupportedTests); err != nil {
			issuesCollector.AddIssue(state, err)
			fmt.Printf("Error: %v\n", err)
		}

		return rlz.ConsumeContinue
	}

	fmt.Printf("Starting Conformance Tests with seed %d ...\n", seed)

	rules := spc.FilterRules(spc.Spec.GetRules(), filter)

	err = spc.ForEachState(rules, opRun, printIssueCounts, jobCount, seed, fullMode)
	if err != nil {
		return fmt.Errorf("error generating States: %w", err)
	}
	issues := issuesCollector.issues

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

	if len(issues) > 0 {
		jsonDir, err := os.MkdirTemp("", "ct_issues_*")
		if err != nil {
			return fmt.Errorf("failed to create output directory for %d issues", len(issues))
		}
		for i, issue := range issuesCollector.issues {
			fmt.Printf("----------------------------\n")
			fmt.Printf("%s\n", issue.err)

			// If there is an input state for this issue, it is exported into a file
			// to aid its debugging using the regression test infrastructure.
			if issue.input != nil {
				path := filepath.Join(jsonDir, fmt.Sprintf("issue_%06d.json", i))
				if err := st.ExportStateJSON(issue.input, path); err == nil {
					fmt.Printf("Input state dumped to %s\n", path)
				} else {
					fmt.Printf("failed to dump state: %v\n", err)
				}
			}
		}
	}

	return fmt.Errorf("failed to pass %d test cases", len(issues))
}

// runTest runs a single test specified by the input state on the given EVM. The
// function returns an error in case the execution did not work as expected.
func runTest(input *st.State, evm ct.Evm, filter *regexp.Regexp, numUnsupportedTests *atomic.Int32) error {
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
	defer result.Release()

	targetError := &vm.ErrUnsupportedRevision{}
	if errors.As(err, &targetError) {
		numUnsupportedTests.Add(1)
		return nil
	}

	if err != nil {
		return fmt.Errorf("failed to process input state %v: %w", input, err)
	}

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

type issue struct {
	input *st.State
	err   error
}

type issuesCollector struct {
	issues []issue
	mu     sync.Mutex
}

func (c *issuesCollector) AddIssue(state *st.State, err error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	var clone *st.State
	if state != nil {
		clone = state.Clone()
	}
	c.issues = append(c.issues, issue{clone, err})
}

func (c *issuesCollector) NumIssues() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.issues)
}
