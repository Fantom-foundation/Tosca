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
	"sync/atomic"
	"time"

	cliUtils "github.com/Fantom-foundation/Tosca/go/ct/driver/cli"
	"github.com/Fantom-foundation/Tosca/go/ct/rlz"
	"github.com/Fantom-foundation/Tosca/go/ct/spc"
	"github.com/Fantom-foundation/Tosca/go/ct/st"
	"github.com/dsnet/golib/unitconv"
	"github.com/urfave/cli/v2"
)

var TestCmd = cliUtils.AddCommonFlags(cli.Command{
	Action: doTest,
	Name:   "test",
	Usage:  "Check test case rule coverage",
	Flags: []cli.Flag{
		cliUtils.FilterFlag,
		cliUtils.JobsFlag,
		cliUtils.SeedFlag,
		cliUtils.FullModeFlag,
	},
})

func doTest(context *cli.Context) error {

	filter, err := cliUtils.FilterFlag.Fetch(context)
	if err != nil {
		return err
	}
	jobCount := cliUtils.JobsFlag.Fetch(context)
	seed := cliUtils.SeedFlag.Fetch(context)
	fullMode := cliUtils.FullModeFlag.Fetch(context)

	issuesCollector := cliUtils.IssuesCollector{}
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

	if issuesCollector.NumIssues() == 0 {
		fmt.Printf("All tests passed successfully!\n")
		return nil
	}

	for _, issue := range issuesCollector.GetIssues() {
		fmt.Printf("----------------------------\n")
		fmt.Printf("%s\n", issue.Error())
	}
	return fmt.Errorf("failed to pass %d test cases", issuesCollector.NumIssues())
}
