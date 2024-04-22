//
// Copyright (c) 2024 Fantom Foundation
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at fantom.foundation/bsl11.
//
// Change Date: 2028-4-16
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by the GNU Lesser General Public Licence v3
//

package main

import (
	"fmt"
	"os"
	"regexp"
	"runtime"
	"runtime/pprof"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/Fantom-foundation/Tosca/go/ct/rlz"
	"github.com/Fantom-foundation/Tosca/go/ct/spc"
	"github.com/Fantom-foundation/Tosca/go/ct/st"
	"github.com/dsnet/golib/unitconv"
	"github.com/urfave/cli/v2"
	"golang.org/x/exp/maps"
)

var StatsCmd = cli.Command{
	Action: doStats,
	Name:   "stats",
	Usage:  "Computes statistics on rule coverage",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:  "filter",
			Usage: "check only rules which name matches the given regex",
			Value: ".*",
		},
		&cli.IntFlag{
			Name:  "jobs",
			Usage: "number of jobs run simultaneously",
			Value: runtime.NumCPU(),
		},
		&cli.Uint64Flag{
			Name:  "seed",
			Usage: "seed for the random number generator",
		},
		&cli.StringFlag{
			Name:  "cpuprofile",
			Usage: "store CPU profile in the provided filename",
		},
		&cli.BoolFlag{
			Name:  "full-mode",
			Usage: "if enabled, test cases targeting rules other than the one generating the case will be executed",
		},
	},
}

func doStats(context *cli.Context) error {
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
	filter, err := regexp.Compile(context.String("filter"))
	if err != nil {
		return err
	}

	jobCount := context.Int("jobs")
	seed := context.Uint64("seed")
	fullMode := context.Bool("full-mode")

	specification := spc.Spec
	statsCollector := newStatsCollector(specification)

	printIssueCounts := func(relativeTime time.Duration, rate float64, current int64) {
		fmt.Printf(
			"[t=%4d:%02d] - Processing ~%s tests per second, total %d\n",
			int(relativeTime.Seconds())/60, int(relativeTime.Seconds())%60,
			unitconv.FormatPrefix(rate, unitconv.SI, 0), current,
		)
	}

	opTest := func(state *st.State) rlz.ConsumerResult {
		for _, rule := range specification.GetRulesFor(state) {
			statsCollector.registerTestFor(rule.Name)
		}
		return rlz.ConsumeContinue
	}

	fmt.Printf("Evaluating Conformance Tests with seed %d using %d jobs ...\n", seed, jobCount)
	rules := spc.FilterRules(spc.Spec.GetRules(), filter)
	err = spc.ForEachState(rules, opTest, printIssueCounts, jobCount, seed, fullMode)
	if err != nil {
		return fmt.Errorf("error evaluating rules: %w", err)
	}

	// Summarize the result.
	fmt.Printf("%v", statsCollector.getStatistics())
	return nil
}

type statsCollector struct {
	statistics ruleStatistics
	mu         sync.Mutex
}

func newStatsCollector(spec spc.Specification) *statsCollector {
	stats := ruleStatistics{make(map[string]ruleInfo)}
	for _, rule := range spec.GetRules() {
		stats.data[rule.Name] = ruleInfo{} // initialize all rules with 0
	}
	return &statsCollector{statistics: stats}
}

func (c *statsCollector) registerTestFor(ruleName string) {
	c.mu.Lock()
	c.statistics.registerTestFor(ruleName)
	c.mu.Unlock()
}

func (c *statsCollector) getStatistics() *ruleStatistics {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.statistics.clone()
}

type ruleStatistics struct {
	data map[string]ruleInfo
}

func (s *ruleStatistics) registerTestFor(rule string) {
	if s.data == nil {
		s.data = make(map[string]ruleInfo)
	}
	stats := s.data[rule]
	stats.numTests++
	s.data[rule] = stats
}

func (s *ruleStatistics) getNumTestsFor(rule string) uint64 {
	return s.data[rule].numTests
}

func (s *ruleStatistics) clone() *ruleStatistics {
	return &ruleStatistics{maps.Clone(s.data)}
}

func (s *ruleStatistics) String() string {
	builder := strings.Builder{}

	rules := maps.Keys(s.data)
	sort.Slice(rules, func(i, j int) bool { return rules[i] < rules[j] })

	builder.WriteString("rule,num_tests\n")
	for _, rule := range rules {
		builder.WriteString(fmt.Sprintf("%s,%d\n", rule, s.data[rule].numTests))
	}
	return builder.String()
}

type ruleInfo struct {
	numTests uint64
}
