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
	config := interpreterConfig{
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
