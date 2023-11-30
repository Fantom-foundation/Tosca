package main

import (
	"errors"
	"fmt"
	"os"
	"regexp"
	"runtime"
	"runtime/pprof"
	"sync"
	"time"

	"pgregory.net/rand"

	"github.com/Fantom-foundation/Tosca/go/ct"
	"github.com/Fantom-foundation/Tosca/go/ct/rlz"
	"github.com/Fantom-foundation/Tosca/go/ct/spc"
	"github.com/Fantom-foundation/Tosca/go/ct/st"
	"github.com/Fantom-foundation/Tosca/go/vm/lfvm"
	"github.com/urfave/cli/v2"
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
			Usage: "maximum number of errors to display (0 displays all errors)",
			Value: 1,
		},
		&cli.Uint64Flag{
			Name:  "seed",
			Usage: "seed for the random number generator",
		},
		&cli.StringFlag{
			Name:  "cpuprofile",
			Usage: "store CPU profile in the provided filename",
		},
	},
}

var evms = map[string]ct.Evm{
	"lfvm": lfvm.NewConformanceTestingTarget(),
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
		availableIdentifiers := make([]string, 0, len(evms))
		for k := range evms {
			availableIdentifiers = append(availableIdentifiers, k)
		}
		return fmt.Errorf("invalid EVM identifier, use one of: %v", availableIdentifiers)
	}

	filter, err := regexp.Compile(context.String("filter"))
	if err != nil {
		return err
	}

	jobCount := context.Int("jobs")
	if jobCount <= 0 {
		jobCount = runtime.NumCPU()
	}

	var mutex sync.Mutex
	var wg sync.WaitGroup

	failed := false
	errorCount := 0

	ruleCh := make(chan rlz.Rule, jobCount)

	for i := 0; i < jobCount; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for rule := range ruleCh {
				tstart := time.Now()

				errs := rule.EnumerateTestCases(rand.New(context.Uint64("seed")), func(state *st.State) error {
					if applies, err := rule.Condition.Check(state); !applies || err != nil {
						return err
					}

					// TODO: program counter pointing to data not supported by LFVM
					// converter.
					if !state.Code.IsCode(int(state.Pc)) {
						return nil // ignored
					}

					input := state.Clone()
					expected := state.Clone()
					rule.Effect.Apply(expected)

					result, err := evm.StepN(input.Clone(), 1)
					if err != nil {
						return err
					}

					if !result.Eq(expected) {
						errMsg := fmt.Sprintln(result.Diff(expected))
						errMsg += fmt.Sprintln("input state:", input)
						errMsg += fmt.Sprintln("result state:", result)
						errMsg += fmt.Sprintln("expected state:", expected)
						return fmt.Errorf(errMsg)
					}

					return nil
				})

				mutex.Lock()
				{
					ok := "OK"
					if len(errs) > 0 {
						ok = "Failed"
					}
					fmt.Printf("%v: %v (%v)\n", ok, rule, time.Since(tstart).Round(10*time.Millisecond))

					errorsToPrint := len(errs)
					if maxErrors := context.Int("max-errors"); maxErrors > 0 {
						errorsToPrint = min(len(errs), maxErrors-errorCount)
					}
					errorCount += errorsToPrint

					printErrors := errs[0:errorsToPrint]
					err := errors.Join(printErrors...)
					if err != nil {
						fmt.Println(err)
						failed = true
					}
				}
				mutex.Unlock()
			}
		}()
	}

	rules := spc.Spec.GetRules()
	for _, rule := range rules {
		if filter.MatchString(rule.Name) {
			ruleCh <- rule
		}
	}

	close(ruleCh)
	wg.Wait()

	if failed {
		return fmt.Errorf("total errors: %d", errorCount)
	}
	return nil
}
