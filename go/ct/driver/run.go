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

	var err error
	var filter *regexp.Regexp
	if pattern := context.String("filter"); pattern != ".*" {
		if filter, err = regexp.Compile(pattern); err != nil {
			return err
		}
	}

	jobCount := context.Int("jobs")
	if jobCount <= 0 {
		jobCount = runtime.NumCPU()
	}

	seed := context.Uint64("seed")
	maxErrors := context.Int("max-errors")
	issues := runTests(evmIdentifier, evm, jobCount, seed, filter, maxErrors)

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

func runTests(evmName string, evm ct.Evm, jobCount int, seed uint64, filter *regexp.Regexp, maxNumIssues int) []issue {
	var wg sync.WaitGroup
	var testCounter atomic.Int64
	var skippedCount atomic.Int32
	issuesCollector := issuesCollector{}

	fmt.Printf("Starting Conformance Tests on %v with seed %d ..\n", evmName, seed)
	if maxNumIssues > 0 {
		fmt.Printf("Testing will abort after identifying %d issues\n", maxNumIssues)
	} else {
		maxNumIssues = math.MaxInt
	}

	wg.Add(jobCount)
	stateCh := make(chan *st.State, 10*jobCount)
	for i := 0; i < jobCount; i++ {
		go func() {
			defer wg.Done()
			for input := range stateCh {
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

	// Run a go-routine printing some progress information for the user.
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
					"[t=%4d:%02d] - Processing ~%.1e tests per second, total %d, skipped %d, found issues %d\n",
					int(relativeTime.Seconds())/60, int(relativeTime.Seconds())%60,
					rate, cur, skippedCount.Load(), issuesCollector.NumIssues(),
				)
			}
		}
	}()

	// Generate test states and wait until they are all finished.
	rand := rand.New(seed)
	rules := spc.Spec.GetRules()
	rules = filterRules(rules, filter)
	for _, rule := range rules {
		if issuesCollector.NumIssues() < maxNumIssues {
			fmt.Printf("Processing %v ...\n", rule.Name)
			rule.EnumerateTestCases(rand, func(state *st.State) rlz.ConsumerResult {
				if issuesCollector.NumIssues() > maxNumIssues {
					return rlz.ConsumeAbort
				}
				stateCh <- state.Clone()
				return rlz.ConsumeContinue
			})
		}
	}
	close(stateCh)
	wg.Wait() // < releases when all test cases are processed

	// Wait for the printer to be finished.
	close(done)   // < signals progress printer to stop
	<-printerDone // < blocks until channel is closed by progress printer

	return issuesCollector.issues
}

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
	c.issues = append(c.issues, issue{state.Clone(), err})
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
