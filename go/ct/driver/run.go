package main

import (
	"errors"
	"fmt"
	"os"
	"regexp"
	"runtime"
	"runtime/pprof"
	"sync"
	"sync/atomic"
	"time"

	"pgregory.net/rand"

	"github.com/Fantom-foundation/Tosca/go/ct"
	"github.com/Fantom-foundation/Tosca/go/ct/common"
	"github.com/Fantom-foundation/Tosca/go/ct/rlz"
	"github.com/Fantom-foundation/Tosca/go/ct/spc"
	"github.com/Fantom-foundation/Tosca/go/ct/st"
	"github.com/Fantom-foundation/Tosca/go/vm/evmzero"
	vm "github.com/Fantom-foundation/Tosca/go/vm/geth"
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
	"lfvm":    lfvm.NewConformanceTestingTarget(),
	"geth":    vm.NewConformanceTestingTarget(),
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

	var errorsPrinted atomic.Int32
	var errorCount atomic.Int32
	var skippedCount atomic.Int32

	ruleCh := make(chan rlz.Rule, jobCount)

	for i := 0; i < jobCount; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for rule := range ruleCh {
				tstart := time.Now()

				enumeratedCount := 0
				errs := rule.EnumerateTestCases(rand.New(context.Uint64("seed")), func(state *st.State) error {
					if applies, err := rule.Condition.Check(state); !applies || err != nil {
						if err == nil {
							err = rlz.ErrInapplicable
						}
						return err
					}

					// TODO: program counter pointing to data not supported by LFVM
					// converter.
					if evmIdentifier == "lfvm" && !state.Code.IsCode(int(state.Pc)) {
						skippedCount.Add(1)
						return rlz.ErrSkipped
					}

					enumeratedCount++

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

				if enumeratedCount == 0 {
					errs = append(errs, common.ConstErr("None of the generated states fulfilled all the conditions"))
				}

				ok := "OK"
				if len(errs) > 0 {
					ok = "Failed"
				}

				errorCount.Add(int32(len(errs)))

				errorsToPrint := len(errs)
				if maxErrors := context.Int("max-errors"); maxErrors > 0 {
					errorsLeftToPrint := max(maxErrors-int(errorsPrinted.Load()), 0)
					errorsToPrint = min(len(errs), errorsLeftToPrint)
				}

				errorsPrinted.Add(int32(errorsToPrint))

				printErrors := errs[0:errorsToPrint]
				err := errors.Join(printErrors...)

				mutex.Lock()
				{
					fmt.Printf("%v: (rules enumerated: %v) %v (%v)\n", ok, enumeratedCount, rule, time.Since(tstart).Round(10*time.Millisecond))

					if err != nil {
						fmt.Println(err)
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

	if cnt := skippedCount.Load(); cnt > 0 {
		fmt.Printf("Skipped tests: %d\n", cnt)
	}
	if cnt := errorCount.Load(); cnt > 0 {
		return fmt.Errorf("total errors: %d", cnt)
	}

	return nil
}
