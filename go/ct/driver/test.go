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
	"fmt"
	"os"
	"regexp"
	"runtime"
	"runtime/pprof"
	"sync/atomic"
	"time"

	"github.com/Fantom-foundation/Tosca/go/ct/rlz"
	"github.com/Fantom-foundation/Tosca/go/ct/spc"
	"github.com/Fantom-foundation/Tosca/go/ct/st"
	"github.com/dsnet/golib/unitconv"
	"github.com/urfave/cli/v2"
)

var TestCmd = cli.Command{
	Action: doTest,
	Name:   "test",
	Usage:  "Check test case rule coverage",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:  "filter",
			Usage: "check only rules which name matches the given regex",
			Value: ".*",
		},
		&cli.IntFlag{
			Name:  "jobs",
			Usage: "number of jobs run simultaneously",
			Value: runtime.NumCPU(),
		},
		&cli.Uint64Flag{
			Name:  "seed",
			Usage: "seed for the random number generator",
		},
		&cli.StringFlag{
			Name:  "cpuprofile",
			Usage: "store CPU profile in the provided filename",
		},
		&cli.BoolFlag{
			Name:  "full-mode",
			Usage: "if enabled, test cases targeting rules other than the one generating the case will be executed",
		},
	},
}

func doTest(context *cli.Context) error {
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
	filter, err := regexp.Compile(context.String("filter"))
	if err != nil {
		return err
	}

	jobCount := context.Int("jobs")
	seed := context.Uint64("seed")
	fullMode := context.Bool("full-mode")

	issuesCollector := issuesCollector{}
	var skippedCount atomic.Int32

	printIssueCounts := func(relativeTime time.Duration, rate float64, current int64) {
		fmt.Printf(
			"[t=%4d:%02d] - Processing ~%s tests per second, total %d, skipped %d, found issues %d\n",
			int(relativeTime.Seconds())/60, int(relativeTime.Seconds())%60,
			unitconv.FormatPrefix(rate, unitconv.SI, 0), current, skippedCount.Load(), issuesCollector.NumIssues(),
		)
	}

	opTest := func(state *st.State) rlz.ConsumerResult {
		rules := spc.Spec.GetRulesFor(state)
		if len(rules) > 1 {
			s0 := state.Clone()
			defer s0.Release()

			rules[0].Effect.Apply(s0)
			for i := 1; i < len(rules)-1; i++ {
				s := state.Clone()
				defer s.Release()
				rules[i].Effect.Apply(s)
				if !s.Eq(s0) {
					issuesCollector.AddIssue(state, fmt.Errorf("multiple conflicting rules for state: %v", rules))
					return rlz.ConsumeContinue
				}
			}
		}
		return rlz.ConsumeContinue
	}

	fmt.Printf("Testing Conformance Tests with seed %d ...\n", seed)
	rules := spc.FilterRules(spc.Spec.GetRules(), filter)
	err = spc.ForEachState(rules, opTest, printIssueCounts, jobCount, seed, fullMode)
	if err != nil {
		return fmt.Errorf("error generating States: %w", err)
	}

	// Summarize the result.
	if skippedCount.Load() > 0 {
		fmt.Printf("Number of skipped tests: %d", skippedCount.Load())
	}

	if len(issuesCollector.issues) == 0 {
		fmt.Printf("All tests passed successfully!\n")
		return nil
	}

	for _, issue := range issuesCollector.issues {
		fmt.Printf("----------------------------\n")
		fmt.Printf("%s\n", issue.err)
	}
	return fmt.Errorf("failed to pass %d test cases", len(issuesCollector.issues))
}
