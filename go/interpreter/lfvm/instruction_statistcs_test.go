// Copyright (c) 2024 Fantom Foundation
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at fantom.foundation/bsl11.
//
// Change Date: 2028-4-16
//
// On the date above, in accordance with the Business Source License, use of
// this software will be governed by the GNU Lesser General Public License v3.

package lfvm

import (
	"fmt"
	"strings"
	"testing"

	"github.com/Fantom-foundation/Tosca/go/tosca"
	"github.com/Fantom-foundation/Tosca/go/tosca/vm"
)

func TestStatisticsRunner_RunWithStatistics(t *testing.T) {
	// Get tosca.Parameters
	params := tosca.Parameters{
		Input:  []byte{},
		Static: true,
		Gas:    10,
		Code:   []byte{byte(STOP), 0},
	}
	code := []Instruction{{STOP, 0}}

	statsRunner := &statisticRunner{
		stats: newStatistics(),
	}
	config := config{
		runner: statsRunner,
	}
	_, err := run(config, params, code)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if got := statsRunner.stats.singleCount[uint64(STOP)]; got != 1 {
		t.Errorf("unexpected statistics: want 1 stop, got %v", got)
	}
}

func TestStatisticsRunner_DumpProfilePrintsExpectedOutput(t *testing.T) {

	tests := map[string]struct {
		code         tosca.Code
		findInOutput []string
	}{
		"singles": {tosca.Code{byte(vm.STOP)},
			[]string{
				"Steps: 1",
				"STOP                          : 1 (100.00%)",
			}},
		"pairs": {tosca.Code{byte(vm.PUSH1), 0x01, byte(vm.STOP)},
			[]string{
				"Steps: 2",
				"PUSH1                         : 1 (50.00%)",
				"STOP                          : 1 (50.00%)",
				"PUSH1                         STOP                          : 1"}},
		"triples": {tosca.Code{byte(vm.PUSH1), 0x01, byte(vm.PUSH1), 0x01, byte(vm.STOP)},
			[]string{
				"Steps: 3",
				"PUSH1                         : 2 (66.67%)",
				"STOP                          : 1 (33.33%)",
				"PUSH1                         PUSH1                         STOP                          : 1"}},
		"quads": {tosca.Code{byte(vm.PUSH1), 0x01, byte(vm.PUSH1), 0x01, byte(vm.PUSH1), 0x01, byte(vm.STOP)},
			[]string{
				"Steps: 4",
				"PUSH1                         : 3 (75.00%)",
				"STOP                          : 1 (25.00%)",
				"PUSH1                         PUSH1                         PUSH1                         : 1 (25.00%)",
				"PUSH1                         PUSH1                         STOP                          : 1 (25.00%)",
				"PUSH1                         PUSH1                         PUSH1                         STOP                          : 1 (25.00%)",
			}},
	}

	for name, test := range tests {
		t.Run(fmt.Sprintf("%v", name), func(t *testing.T) {
			statsRunner := &statisticRunner{
				stats: newStatistics(),
			}

			instance, err := newVm(config{
				runner: statsRunner,
			})
			if err != nil {
				t.Fatalf("Failed to create VM: %v", err)
			}
			instance.ResetProfile()
			//run code
			_, err = instance.Run(tosca.Parameters{Input: []byte{}, Static: true, Gas: 10,
				Code: test.code})
			if err != nil {
				t.Fatalf("Failed to run code: %v", err)
			}

			// Run testing code
			instance.DumpProfile()

			out := statsRunner.stats.print()
			for _, s := range test.findInOutput {
				if !strings.Contains(string(out), s) {
					t.Errorf("did not find occurrences of %v in %v", s, string(out))
				}
			}
		})
	}
}

func TestStatisticsRunner_getSummaryInitializesNewStatsWhenUninitialized(t *testing.T) {
	statsRunner := &statisticRunner{
		stats: nil,
	}
	_ = statsRunner.getSummary()
	if statsRunner.stats == nil {
		t.Errorf("summary should have been initialized")
	}
}

func TestStatisticsRunner_runInitializesNewStatsWhenUninitialized(t *testing.T) {
	statsRunner := &statisticRunner{
		stats: nil,
	}
	_, _ = statsRunner.run(&context{
		code:  []Instruction{{STOP, 0}},
		stack: NewStack(),
	})
	if statsRunner.stats == nil {
		t.Errorf("run should have initialized stats")
	}
}

func TestStatisticsRunner_statisticsStopsWhenExecutionEncountersAnError(t *testing.T) {
	statsRunner := &statisticRunner{
		stats: nil,
	}
	_, _ = statsRunner.run(&context{
		// this code should not reach a STOP since MCOPY should fail because
		// there are not enough items on the stack
		code:  []Instruction{{MCOPY, 0}, {STOP, 0}},
		stack: NewStack(),
	})
	if statsRunner.stats.singleCount[uint64(STOP)] != 0 {
		t.Errorf("unexpected statistics: stop should not be executed, got %v", statsRunner.stats.singleCount[uint64(STOP)])
	}
}

func TestStatisticsRunner_print_getTopN_returnFirstNElementsOfManyMore(t *testing.T) {
	want := "\n----- Statistics ------\n"
	want += "\nSteps: 0\n"
	want += "\nSingles:\n"
	want += "\tPUSH5                         : 5 (+Inf%)\n"
	want += "\tPUSH4                         : 4 (+Inf%)\n"
	want += "\tPUSH3                         : 3 (+Inf%)\n"
	want += "\tPUSH2                         : 2 (+Inf%)\n"
	want += "\tSTOP                          : 1 (+Inf%)\n"
	want += "\nPairs:\n\nTriples:\n\nQuads:\n\n"

	stats := statistics{
		singleCount: map[uint64]uint64{
			uint64(STOP):  1,
			uint64(PUSH1): 1,
			uint64(PUSH2): 2,
			uint64(PUSH3): 3,
			uint64(PUSH4): 4,
			uint64(PUSH5): 5,
		},
	}
	got := stats.print()
	if got != want {
		t.Errorf("unexpected output: want %v, got %v", want, got)
	}
}
