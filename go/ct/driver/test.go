package main

import (
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/Fantom-foundation/Tosca/go/ct/spc"
	"github.com/Fantom-foundation/Tosca/go/ct/st"
	"github.com/urfave/cli/v2"
	"pgregory.net/rand"
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
	},
}

func doTest(context *cli.Context) error {
	filter, err := regexp.Compile(context.String("filter"))
	if err != nil {
		return nil
	}

	var mutex sync.Mutex
	var wg sync.WaitGroup

	failed := false

	rules := spc.Spec.GetRules()
	for _, rule := range rules {
		if !filter.MatchString(rule.Name) {
			continue
		}

		rule := rule

		wg.Add(1)
		go func() {
			defer wg.Done()

			// TODO: For now, check that we get at least one rule matching for
			// the full set of test cases (and at most one rule for every test
			// case). Later we'll enforce that exactly one rule applies to every
			// single test case.
			atLeastOne := false

			tstart := time.Now()

			rnd := rand.New(context.Uint64("seed"))
			errs := rule.EnumerateTestCases(rnd, func(state *st.State) error {
				rules := spc.Spec.GetRulesFor(state)
				if len(rules) > 0 {
					atLeastOne = true
				}
				if len(rules) > 1 {
					s0 := state.Clone()
					rules[0].Effect.Apply(s0)
					for i := 1; i < len(rules)-1; i++ {
						s := state.Clone()
						rules[i].Effect.Apply(s)
						if !s.Eq(s0) {
							return fmt.Errorf("multiple conflicting rules for state %v: %v", state, rules)
						}
					}
				}
				return nil
			})

			mutex.Lock()
			defer mutex.Unlock()

			if len(errs) != 0 {
				builder := strings.Builder{}
				builder.WriteString(fmt.Sprintf("FAIL: %v\n", rule.Name))
				for _, e := range errs {
					builder.WriteString(fmt.Sprintf("%v\n", e))
				}
				fmt.Print(builder.String())
				failed = true
				return
			}

			if !atLeastOne {
				fmt.Printf("FAIL: %v: No rule matches any of the generated test cases\n", rule.Name)
				failed = true
				return
			}

			fmt.Printf("OK: %v (%v)\n", rule.Name, time.Since(tstart).Round(10*time.Millisecond))
		}()
	}

	wg.Wait()

	if failed {
		return fmt.Errorf("coverage failed")
	}

	return nil
}
