package main

import (
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"pgregory.net/rand"

	"github.com/Fantom-foundation/Tosca/go/ct"
	"github.com/Fantom-foundation/Tosca/go/ct/common"
	"github.com/Fantom-foundation/Tosca/go/ct/gen"
	"github.com/Fantom-foundation/Tosca/go/ct/rlz"
	"github.com/Fantom-foundation/Tosca/go/ct/spc"
	"github.com/Fantom-foundation/Tosca/go/ct/st"
	"github.com/dsnet/golib/unitconv"
	"github.com/urfave/cli/v2"
	"golang.org/x/exp/maps"
)

var ProbeCmd = AddCommonFlags(cli.Command{
	Action:    doProbe,
	Name:      "probe",
	Usage:     "Run random tests on an EVM implementation",
	ArgsUsage: "<EVM>",
	Flags: []cli.Flag{
		&cli.IntFlag{
			Name:  "jobs",
			Usage: "number of jobs run simultaneously",
			Value: runtime.NumCPU(),
		},
		&cli.Uint64Flag{
			Name:  "seed",
			Usage: "seed for the random number generator",
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

	jobCount := context.Int("jobs")
	if jobCount <= 0 {
		jobCount = runtime.NumCPU()
	}

	seed := context.Uint64("seed")

	// The constraints to be placed on generated states.
	condition := rlz.And(
		rlz.IsCode(rlz.Pc()),
		rlz.RevisionBounds(common.R07_Istanbul, common.R10_London),
	)

	fmt.Printf("Start random tests on %s using %d jobs, seed %d, and constraints %s ...\n", evmIdentifier, jobCount, seed, condition)

	// Run a progress printer in the background.
	counter := atomic.Uint64{}
	stopProgressPrinter := make(chan struct{})
	var progressGroup sync.WaitGroup
	progressGroup.Add(1)
	go func() {
		defer progressGroup.Done()
		start := time.Now()
		last := uint64(0)
		for {
			select {
			case <-stopProgressPrinter:
				return
			case <-time.After(5 * time.Second):
				relativeTime := time.Since(start)
				current := counter.Load()
				diff := current - last
				last = current
				rate := float64(diff) / 5
				fmt.Printf(
					"[t=%4d:%02d] - Processing ~%s tests per second, total %d\n",
					int(relativeTime.Seconds())/60, int(relativeTime.Seconds())%60,
					unitconv.FormatPrefix(rate, unitconv.SI, 0), current,
				)
			}
		}
	}()

	issuesCollector := &issuesCollector{}

	var wg sync.WaitGroup
	wg.Add(jobCount)
	for i := 0; i < jobCount; i++ {
		go func() {
			defer wg.Done()
			rnd := rand.New(seed)
			generator := gen.NewStateGenerator()
			condition.Restrict(generator)

			for issuesCollector.NumIssues() == 0 {
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
		return fmt.Errorf(formatDiffForUser(state, result, expected, rules[0].Name))
	}
	return nil
}
