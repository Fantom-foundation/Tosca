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
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"pgregory.net/rand"

	"github.com/Fantom-foundation/Tosca/go/ct"
	"github.com/Fantom-foundation/Tosca/go/ct/common"
	cliUtils "github.com/Fantom-foundation/Tosca/go/ct/driver/cli"
	"github.com/Fantom-foundation/Tosca/go/ct/gen"
	"github.com/Fantom-foundation/Tosca/go/ct/rlz"
	"github.com/Fantom-foundation/Tosca/go/ct/spc"
	"github.com/Fantom-foundation/Tosca/go/ct/st"
	"github.com/dsnet/golib/unitconv"
	"github.com/urfave/cli/v2"
	"golang.org/x/exp/maps"
)

var ProbeCmd = cliUtils.AddCommonFlags(cli.Command{
	Action:    doProbe,
	Name:      "probe",
	Usage:     "Run random tests on an EVM implementation",
	ArgsUsage: "<EVM>",
	Flags: []cli.Flag{
		cliUtils.FilterFlag,
		cliUtils.SeedFlag,
		&cli.IntFlag{
			Name:  "timeout",
			Usage: "maximum time in minutes to run the probing. Default 30 minutes",
			Value: 30,
		},
	},
})

func doProbe(context *cli.Context) error {
	var evmIdentifier string
	if context.Args().Len() >= 1 {
		evmIdentifier = context.Args().Get(0)
	}

	evm, ok := evms[evmIdentifier]
	if !ok {
		return fmt.Errorf("invalid EVM identifier, use one of: %v", maps.Keys(evms))
	}

	jobCount := cliUtils.JobsFlag.Fetch(context)
	if jobCount <= 0 {
		jobCount = runtime.NumCPU()
	}

	seed := cliUtils.SeedFlag.Fetch(context)

	timeout := time.Duration(context.Int("timeout")) * time.Minute
	hasTimeouted := atomic.Bool{}
	hasTimeouted.Store(false)

	// The constraints to be placed on generated states.
	condition := rlz.And(
		rlz.IsCode(rlz.Pc()),
		rlz.RevisionBounds(common.MinRevision, common.NewestFullySupportedRevision),
	)

	fmt.Printf("Start random tests on %s using %d jobs, seed %d, timeout %v and constraints %s ...\n", evmIdentifier, jobCount, seed, timeout, condition)

	// Run a progress printer in the background.
	counter := atomic.Uint64{}
	stopProgressPrinter := make(chan struct{})
	var progressGroup sync.WaitGroup
	progressGroup.Add(1)
	go func() {
		defer progressGroup.Done()
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()
		startTime := time.Now()
		last := uint64(0)
		for {
			select {
			case <-stopProgressPrinter:
				return
			case curTime := <-ticker.C:
				relativeTime := curTime.Sub(startTime)
				current := counter.Load()
				diff := current - last
				last = current
				rate := float64(diff) / 5
				fmt.Printf(
					"[t=%4d:%02d] - Processing ~%s tests per second, total %d\n",
					int(relativeTime.Seconds())/60, int(relativeTime.Seconds())%60,
					unitconv.FormatPrefix(rate, unitconv.SI, 0), current,
				)
				if relativeTime > timeout {
					fmt.Printf("Timeout reached, stopping the probing\n")
					hasTimeouted.Store(true)
					return
				}
			}
		}
	}()

	issuesCollector := &cliUtils.IssuesCollector{}

	var wg sync.WaitGroup
	wg.Add(jobCount)
	for i := 0; i < jobCount; i++ {
		ii := i
		go func() {
			defer wg.Done()
			rnd := rand.New(seed + uint64(ii))
			generator := gen.NewStateGenerator()
			condition.Restrict(generator)

			for issuesCollector.NumIssues() == 0 && !hasTimeouted.Load() {
				state, err := generator.Generate(rnd)
				if err != nil {
					issuesCollector.AddIssue(nil, err)
					return
				}
				if err := testState(spc.Spec, state, evm); err != nil {
					issuesCollector.AddIssue(state, err)
					return
				}
				state.Release()
				counter.Add(1)
			}
		}()
	}

	wg.Wait()
	close(stopProgressPrinter)
	progressGroup.Wait()

	// Summarize the result.
	fmt.Printf("Random tests completed, %d tests executed\n", counter.Load())
	numIssues := issuesCollector.NumIssues()
	if numIssues == 0 {
		fmt.Printf("All tests passed successfully!\n")
	} else if err := issuesCollector.ExportIssues(); err != nil {
		return err
	}
	fmt.Printf("Issues found: %d\n", issuesCollector.NumIssues())

	return nil
}

func testState(specification spc.Specification, state *st.State, evm ct.Evm) error {

	// Check that there is a rule for the state (completeness check).
	rules := specification.GetRulesFor(state)
	if len(rules) == 0 {
		return fmt.Errorf("no rules for state %s", state.String())
	}

	// Check soundness.
	expected := state.Clone()
	defer expected.Release()
	rules[0].Effect.Apply(expected)
	if len(rules) > 1 {
		for i := 1; i < len(rules); i++ {
			have := state.Clone()
			defer have.Release()
			rules[i].Effect.Apply(have)
			if !expected.Eq(have) {
				return fmt.Errorf(
					"rules %s and %s produce different results, diff %s",
					rules[0].Name,
					rules[i].Name,
					expected.Diff(have),
				)
			}
		}
	}

	// Check that the EVM behaves as expected.
	result, err := evm.StepN(state.Clone(), 1)
	if err != nil {
		return err
	}
	defer result.Release()

	if !expected.Eq(result) {
		return errors.New(formatDiffForUser(state, result, expected, rules[0].Name))
	}
	return nil
}
