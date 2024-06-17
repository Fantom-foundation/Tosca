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
	"strings"
	"testing"

	"github.com/Fantom-foundation/Tosca/go/ct/rlz"
	"github.com/Fantom-foundation/Tosca/go/ct/spc"
	"go.uber.org/mock/gomock"
)

func TestRuleStatistics_EmptyCanBePrinted(t *testing.T) {
	stats := ruleStatistics{}
	if want, got := "rule,num_tests\n", stats.String(); want != got {
		t.Errorf("unexpected result, wanted %s, got %s", want, got)
	}
}

func TestRuleStatistics_RulesArePrintedSorted(t *testing.T) {
	stats := ruleStatistics{}
	stats.registerTestFor("b")
	stats.registerTestFor("a")
	stats.registerTestFor("c")
	stats.registerTestFor("b")
	if want, got := "rule,num_tests\na,1\nb,2\nc,1\n", stats.String(); want != got {
		t.Errorf("unexpected result, wanted %s, got %s", want, got)
	}
}

func TestRuleStatistics_CloneContainsCopy(t *testing.T) {
	stats := ruleStatistics{}
	stats.registerTestFor("a")
	stats.registerTestFor("b")

	clone := stats.clone()
	clone.registerTestFor("a")
	clone.registerTestFor("c")

	if want, got := "rule,num_tests\na,1\nb,1\n", stats.String(); want != got {
		t.Errorf("unexpected result, wanted %s, got %s", want, got)
	}

	if want, got := "rule,num_tests\na,2\nb,1\nc,1\n", clone.String(); want != got {
		t.Errorf("unexpected result, wanted %s, got %s", want, got)
	}
}

func TestStatisticCollector_InitializesAllRulesToZero(t *testing.T) {
	ctrl := gomock.NewController(t)
	spec := spc.NewMockSpecification(ctrl)
	spec.EXPECT().GetRules().Return([]rlz.Rule{
		{Name: "a"}, {Name: "b"},
	})

	collector := newStatsCollector(spec.GetRules())
	stats := collector.getStatistics()

	if want, got := 2, len(stats.data); want != got {
		t.Errorf("unexpected number of entries, wanted %d, got %d", want, got)
	}

	print := stats.String()
	if !strings.Contains(print, "a,0") {
		t.Errorf("missing rule 'a' in result")
	}
	if !strings.Contains(print, "b,0") {
		t.Errorf("missing rule 'b' in result")
	}
}

func TestStatisticCollector_RegisteringTestsIncreasesCounter(t *testing.T) {
	ctrl := gomock.NewController(t)
	spec := spc.NewMockSpecification(ctrl)
	spec.EXPECT().GetRules().Return([]rlz.Rule{{Name: "a"}})

	collector := newStatsCollector(spec.GetRules())
	if want, got := uint64(0), collector.getStatistics().getNumTestsFor("a"); want != got {
		t.Errorf("unexpected number of tests for 'a', wanted %d, got %d", want, got)
	}

	collector.registerTestFor("a")
	if want, got := uint64(1), collector.getStatistics().getNumTestsFor("a"); want != got {
		t.Errorf("unexpected number of tests for 'a', wanted %d, got %d", want, got)
	}

	collector.registerTestFor("a")
	if want, got := uint64(2), collector.getStatistics().getNumTestsFor("a"); want != got {
		t.Errorf("unexpected number of tests for 'a', wanted %d, got %d", want, got)
	}

	collector.registerTestFor("b")
	if want, got := uint64(1), collector.getStatistics().getNumTestsFor("b"); want != got {
		t.Errorf("unexpected number of tests for 'b', wanted %d, got %d", want, got)
	}
}
