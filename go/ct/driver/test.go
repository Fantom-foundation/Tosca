package main

import (
	"fmt"
	"log"
	"math"
	"net/http"
	"regexp"
	"runtime"
	"sync/atomic"
	"time"

	_ "net/http/pprof"

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
		&cli.Uint64Flag{
			Name:  "seed",
			Usage: "seed for the random number generator",
		},
		&cli.StringFlag{
			Name:  "cpuprofile",
			Usage: "store CPU profile in the provided filename",
		},
		&cli.Int64Flag{
			Name:  "diagnostic-port",
			Usage: "enable hosting of a realtime diagnostic server by providing a port",
			Value: 0,
		},
		&cli.BoolFlag{ // < TODO: make every run a full mode once tests pass
			Name:  "full-mode",
			Usage: "if enabled, test cases targeting rules other than the one generating the case will be executed",
		},
	},
}

func doTest(context *cli.Context) error {

	port := context.Int64("diagnostic-port")
	if port > 0 {
		if port > math.MaxUint16 {
			return fmt.Errorf("invalid port for diagnostic server: %d", port)
		}
		fmt.Printf("Starting diagnostic server at port http://localhost:%d (see https://pkg.go.dev/net/http/pprof#hdr-Usage_examples for usage examples)", port)
		fmt.Printf("Block and mutex sampling rate is set to 100%% for diagnostics, which may impact overall performance")
		go func() {
			addr := fmt.Sprintf("localhost:%d", port)
			log.Println(http.ListenAndServe(addr, nil))
		}()
		runtime.SetBlockProfileRate(1)
		runtime.SetMutexProfileFraction(1)
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
	fullMode := context.Bool("full-mode")

	issuesCollector := issuesCollector{}
	var skippedCount atomic.Int32
	var atLeastOne atomic.Bool

	printIssueCounts := func(relativeTime time.Duration, rate float64, current int64) {
		fmt.Printf(
			"[t=%4d:%02d] - Processing ~%s tests per second, total %d, skipped %d, found issues %d\n",
			int(relativeTime.Seconds())/60, int(relativeTime.Seconds())%60,
			unitconv.FormatPrefix(rate, unitconv.SI, 0), current, skippedCount.Load(), issuesCollector.NumIssues(),
		)
	}

	opTest := func(state *st.State) rlz.ConsumerResult {
		rules := spc.Spec.GetRulesFor(state)
		if len(rules) > 0 {
			atLeastOne.Store(true)
		}
		if len(rules) > 1 {
			s0 := state.Clone()
			rules[0].Effect.Apply(s0)
			for i := 1; i < len(rules)-1; i++ {
				s := state.Clone()
				rules[i].Effect.Apply(s)
				if !s.Eq(s0) {
					issuesCollector.AddIssue(nil, fmt.Errorf("multiple conflicting rules for state %v: %v", state, rules))
					return rlz.ConsumeContinue
				}
			}
		}
		return rlz.ConsumeContinue
	}

	fmt.Printf("Testing Conformance Tests with seed %d ...\n", seed)

	err = forEachState(opTest, printIssueCounts, jobCount, seed, fullMode, filter)
	if err != nil {
		return fmt.Errorf("error: %v", err)
	}

	// Summarize the result.
	if skippedCount.Load() > 0 {
		fmt.Printf("Number of skipped tests: %d", skippedCount.Load())
	}

	if len(issuesCollector.issues) == 0 {
		fmt.Printf("All tests passed successfully!\n")
		return nil
	}

	if atLeastOne.Load() {
		fmt.Printf("No rule matches any of the generated test cases\n")
	}

	for _, issue := range issuesCollector.issues {
		// TODO: write input state of found issues into files
		fmt.Printf("----------------------------\n")
		fmt.Printf("%s\n", issue.err)
	}
	return fmt.Errorf("failed to pass %d test cases", len(issuesCollector.issues))
}
