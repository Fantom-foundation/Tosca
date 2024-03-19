package main

import (
	"fmt"
	"math"
	"os"
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
	"lfvm": lfvm.NewConformanceTestingTarget(),
	// "geth":    vm.NewConformanceTestingTarget(), // < TODO: fix and reenable
	"evmzero": evmzero.NewConformanceTestingTarget(),
}

func doRun(context *cli.Context) error {
	if cpuprofileFilename := context.String("cpuprofile"); cpuprofileFilename != "" {
		f, err := os.Create(cpuprofileFilename)
		if err != nil {
			return fmt.Errorf("could not create CPU profile: %s", err)
		}
		if err := pprof.StartCPUProfile(f); err != nil {
			return fmt.Errorf("could not start CPU profile: %s", err)
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
	fullMode := context.Bool("full-mode")

	// Run the actual tests.
	issues := runTests(evmIdentifier, evm, jobCount, seed, filter, fullMode, maxErrors)

	// Summarize the result.
	if len(issues) == 0 {
		fmt.Printf("All tests passed successfully!\n")
		return nil
	}

	for _, issue := range issues {
		// TODO: write input state of found issues into files
		fmt.Printf("----------------------------\n")
		fmt.Printf("%s\n", issue.err)
	}
	return fmt.Errorf("failed to pass %d test cases", len(issues))
}

// runTests orchestrates the parallel execution of all tests derived from the EVM
// specification using numJobs parallel workers.
func runTests(
	evmName string,
	evm ct.Evm,
	numJobs int,
	seed uint64,
	filter *regexp.Regexp,
	fullMode bool,
	maxNumIssues int,
) []issue {
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
	var skippedCount atomic.Int32
	issuesCollector := issuesCollector{}

	fmt.Printf("Starting Conformance Tests on %v with seed %d ..\n", evmName, seed)
	if maxNumIssues > 0 {
		fmt.Printf("Testing will abort after identifying %d issue(s)\n", maxNumIssues)
	} else {
		maxNumIssues = math.MaxInt
	}

	// Run a goroutine printing some progress information for the user.
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
				fmt.Printf(
					"[t=%4d:%02d] - Processing ~%s tests per second, total %d, skipped %d, found issues %d\n",
					int(relativeTime.Seconds())/60, int(relativeTime.Seconds())%60,
					unitconv.FormatPrefix(rate, unitconv.SI, 0), cur, skippedCount.Load(), issuesCollector.NumIssues(),
				)
			}
		}
	}()

	// Run goroutines processing the actual tests.
	stateWaitGroup.Add(numJobs)
	stateChannel := make(chan *st.State, 10*numJobs)
	for i := 0; i < numJobs; i++ {
		go func() {
			defer stateWaitGroup.Done()
			for input := range stateChannel {
				if issuesCollector.NumIssues() >= maxNumIssues {
					continue
				}
				testCounter.Add(1)

				// TODO: program counter pointing to data not supported by LFVM
				// converter. Fix this.
				if evmName == "lfvm" && !input.Code.IsCode(int(input.Pc)) {
					skippedCount.Add(1)
					continue
				}

				if err := runTest(input, evm, filter); err != nil {
					issuesCollector.AddIssue(input, err)
				}
			}
		}()
	}

	// Generate test states in parallel (generation can be the bottleneck if there
	// are many workers processing test cases in parallel).
	ruleChannel := make(chan rlz.Rule, 10*numJobs)
	var rulesWaitGroup sync.WaitGroup
	rulesWaitGroup.Add(numJobs)
	for i := 0; i < numJobs; i++ {
		go func() {
			defer rulesWaitGroup.Done()
			for rule := range ruleChannel {
				if issuesCollector.NumIssues() >= maxNumIssues {
					continue
				}
				// random is re-seeded for each rule to be reproducible.
				rand := rand.New(seed)
				err := rule.EnumerateTestCases(rand, func(state *st.State) rlz.ConsumerResult {
					if issuesCollector.NumIssues() >= maxNumIssues {
						return rlz.ConsumeAbort
					}
					if !fullMode {
						if applies, err := rule.Condition.Check(state); !applies || err != nil {
							if err != nil {
								issuesCollector.AddIssue(state, err)
							}
							return rlz.ConsumeContinue
						}
					}

					stateChannel <- state.Clone()
					return rlz.ConsumeContinue
				})
				if err != nil {
					issuesCollector.AddIssue(nil, fmt.Errorf("failed to enumerate test cases for %v: %w", rule, err))
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

	return issuesCollector.issues
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
	rule.Effect.Apply(expected)

	result, err := evm.StepN(input.Clone(), 1)
	if err != nil {
		return fmt.Errorf("failed to process input state %v: %w", input, err)
	}

	if result.Eq(expected) {
		return nil
	}
	errMsg := fmt.Sprintln("input state:", input)
	errMsg += fmt.Sprintln("result state:", result)
	errMsg += fmt.Sprintln("expected state:", expected)
	errMsg += fmt.Sprintln("expectation defined by rule: ", rule.Name)
	errMsg += "Differences:\n"
	for _, diff := range result.Diff(expected) {
		errMsg += fmt.Sprintf("\t%s\n", diff)
	}
	return fmt.Errorf(errMsg)
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
