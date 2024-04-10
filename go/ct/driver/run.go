package main

import (
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
	"github.com/Fantom-foundation/Tosca/go/vm/evmzero"
	"github.com/Fantom-foundation/Tosca/go/vm/geth"
	"github.com/Fantom-foundation/Tosca/go/vm/lfvm"
	"github.com/dsnet/golib/unitconv"
	"github.com/urfave/cli/v2"
	"golang.org/x/exp/maps"
	"pgregory.net/rand"
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
			issuesCollector.AddIssue(state, err)
		}

		return rlz.ConsumeContinue
	}

	fmt.Printf("Starting Conformance Tests with seed %d ...\n", seed)

	err = forEachState(opRun, printIssueCounts, jobCount, seed, fullMode, filter)
	if err != nil {
		return fmt.Errorf("error generating States: %w", err)
	}
	issues := issuesCollector.issues

	// Summarize the result.
	if skippedCount.Load() > 0 {
		fmt.Printf("Number of skipped tests: %d", skippedCount.Load())
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

func forEachState(
	opFunction func(state *st.State) rlz.ConsumerResult,
	printIssueCounts func(relativeTime time.Duration, rate float64, current int64),
	numJobs int,
	seed uint64,
	fullMode bool,
	filter *regexp.Regexp,
) error {
	// The execution of test cases is distributed to parallel goroutines in a three-step
	// process:
	//   - this goroutine writes the list of rules to be tested into a channel
	//   - a team of goroutines fetches rules from the first channel, runs the
	//     test state enumeration for the retrieved rule, and forward those states
	//     into a second channel
	//   - another team of goroutines fetches test-input states from the second
	//     channel and processes the actual tests.
	// Additionally, a goroutine periodically reporting progress information to the
	// console is started.
	// To avoid dead-locks in this goroutine, consuming goroutines are started before
	// producing routines. Thus, the order in which goroutines and teams of goroutines
	// are started below is in the reverse order as listed above.

	var stateWaitGroup sync.WaitGroup
	var testCounter atomic.Int64
	var abortTests atomic.Bool
	abortTests.Store(false)

	done := make(chan bool)
	printerDone := make(chan bool)
	go func() {
		defer close(printerDone)
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()
		startTime := time.Now()
		lastTime := startTime
		lastTestCounter := int64(0)
		for {
			select {
			case <-done:
				return
			case curTime := <-ticker.C:
				cur := testCounter.Load()

				diffCounter := cur - lastTestCounter
				diffTime := curTime.Sub(lastTime)

				lastTime = curTime
				lastTestCounter = cur

				relativeTime := curTime.Sub(startTime)
				rate := float64(diffCounter) / diffTime.Seconds()
				printIssueCounts(relativeTime, rate, cur)
			}
		}
	}()

	// Run goroutines processing the actual tests.
	stateWaitGroup.Add(numJobs)
	stateChannel := make(chan *st.State, 10*numJobs)
	for i := 0; i < numJobs; i++ {
		go func() {
			defer stateWaitGroup.Done()
			for state := range stateChannel {
				testCounter.Add(1)
				consumeStatus := opFunction(state)
				if consumeStatus == rlz.ConsumeAbort {
					abortTests.Store(true)
				}
				st.ReturnState(state)
			}
		}()
	}

	// Generate test states in parallel (generation can be the bottleneck if there
	// are many workers processing test cases in parallel).
	ruleChannel := make(chan rlz.Rule, 10*numJobs)
	var rulesWaitGroup sync.WaitGroup
	rulesWaitGroup.Add(numJobs)

	var errorMutex sync.Mutex
	var returnError error

	for i := 0; i < numJobs; i++ {
		go func() {
			defer rulesWaitGroup.Done()
			for rule := range ruleChannel {
				if abortTests.Load() {
					continue // keep consume rules in the ruleChannel
				}
				// random is re-seeded for each rule to be reproducible.
				rand := rand.New(seed)
				err := rule.EnumerateTestCases(rand, func(state *st.State) rlz.ConsumerResult {
					if abortTests.Load() {
						return rlz.ConsumeAbort
					}
					if !fullMode {
						if applies, err := rule.Condition.Check(state); !applies || err != nil {
							return rlz.ConsumeContinue
						}
					}

					stateChannel <- state.Clone()
					return rlz.ConsumeContinue
				})
				if err != nil {
					abortTests.Store(true)
					errorMutex.Lock()
					returnError = err
					errorMutex.Unlock()
					continue
				}
			}
		}()
	}

	// Feed the rule generator workers with rules.
	rules := spc.Spec.GetRules()
	rules = filterRules(rules, filter)
	for _, rule := range rules {
		ruleChannel <- rule
	}

	close(ruleChannel)
	rulesWaitGroup.Wait()

	close(stateChannel)
	stateWaitGroup.Wait() // < releases when all test cases are processed

	// Wait for the printer to be finished.
	close(done)   // < signals progress printer to stop
	<-printerDone // < blocks until channel is closed by progress printer

	return returnError
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
	rules = filterRules(rules, filter)
	if len(rules) == 0 {
		return nil // < this is fine, the targeted rules are filtered out by the user
	}

	// TODO: enable optional rule consistency check
	rule := rules[0]
	expected := input.Clone()
	defer st.ReturnState(expected)

	rule.Effect.Apply(expected)

	result, err := evm.StepN(input.Clone(), 1)
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

func filterRules(rules []rlz.Rule, filter *regexp.Regexp) []rlz.Rule {
	if filter == nil {
		return rules
	}
	res := make([]rlz.Rule, 0, len(rules))
	for _, rule := range rules {
		if filter.MatchString(rule.Name) {
			res = append(res, rule)
		}
	}
	return res
}
