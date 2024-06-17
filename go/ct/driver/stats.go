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
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	cliUtils "github.com/Fantom-foundation/Tosca/go/ct/driver/cli"
	"github.com/Fantom-foundation/Tosca/go/ct/rlz"
	"github.com/Fantom-foundation/Tosca/go/ct/spc"
	"github.com/Fantom-foundation/Tosca/go/ct/st"
	"github.com/dsnet/golib/unitconv"
	"github.com/urfave/cli/v2"
	"golang.org/x/exp/maps"
)

var StatsCmd = cliUtils.AddCommonFlags(cli.Command{
	Action: doStats,
	Name:   "stats",
	Usage:  "Computes statistics on rule coverage",
	Flags: []cli.Flag{
		cliUtils.FilterFlag,
		cliUtils.JobsFlag,
		cliUtils.SeedFlag,
		cliUtils.FullModeFlag,
	},
})

func doStats(context *cli.Context) error {

	filter, err := cliUtils.FilterFlag.Fetch(context)
	if err != nil {
		return err
	}

	jobCount := cliUtils.JobsFlag.Fetch(context)
	seed := cliUtils.SeedFlag.Fetch(context)
	fullMode := cliUtils.FullModeFlag.Fetch(context)

	specification := spc.Spec
	rules := spc.FilterRules(spc.Spec.GetRules(), filter)
	statsCollector := newStatsCollector(rules)

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

func newStatsCollector(rules []rlz.Rule) *statsCollector {
	stats := ruleStatistics{make(map[string]ruleInfo)}
	for _, rule := range rules {
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
