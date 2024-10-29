// Copyright (c) 2024 Fantom Foundation
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at fantom.foundation/bsl11.
//
// Change Date: 2028-4-16
//
// On the date above, in accordance with the Business Source License, use of
// this software will be governed by the GNU Lesser General Public License v3.

package rlz

import (
	"strings"
	"testing"

	"golang.org/x/exp/maps"
	"pgregory.net/rand"

	. "github.com/Fantom-foundation/Tosca/go/ct/common"
	"github.com/Fantom-foundation/Tosca/go/ct/st"
	"github.com/Fantom-foundation/Tosca/go/tosca/vm"
)

func TestRule_GenerateSatisfyingState(t *testing.T) {
	tests := []Condition{
		And(), // = anything
		Eq(Status(), st.Failed),
		Eq(Pc(), NewU256(42)),
		And(Eq(Status(), st.Failed), Eq(Pc(), NewU256(42))),
		And(Eq(Op(Pc()), vm.ADD)),
		And(Eq(Op(Pc()), vm.JUMP), Eq(Op(Param(0)), vm.JUMPDEST)),
		And(Eq(Op(Constant(NewU256(12))), vm.ADD), Eq(Op(Constant(NewU256(3))), vm.JUMP)),
		And(Eq(Balance(SelfAddress()), NewU256(42))),
		And(Gt(Balance(SelfAddress()), NewU256(0))),
	}

	rnd := rand.New(0)

	for _, test := range tests {
		rule := Rule{Condition: test}
		state, err := rule.GenerateSatisfyingState(rnd)
		if err != nil {
			t.Errorf("Failed to generate state: %v", err)
		}

		satisfied, err := test.Check(state)
		if err != nil {
			t.Errorf("Condition check error %v", err)
		}
		if !satisfied {
			t.Errorf("Generated state does not satisfy condition '%v': %v", test, state)
		}
	}
}

func TestRule_EnumerateTestCases(t *testing.T) {
	tests := []Condition{
		And(), // = anything
		Eq(Status(), st.Failed),
		Eq(Pc(), NewU256(42)),
		And(Eq(Status(), st.Failed), Eq(Pc(), NewU256(42))),
		And(Eq(Op(Pc()), vm.ADD)),
		And(Eq(Op(Pc()), vm.JUMP), Eq(Op(Param(0)), vm.JUMPDEST)),
		And(Eq(Balance(SelfAddress()), NewU256(42))),
		And(Gt(Balance(SelfAddress()), NewU256(0))),
		And(Eq(Balance(ToAddress(Param(0))), NewU256(20))),
		And(Eq(Balance(SelfAddress()), NewU256(0))),
	}

	rnd := rand.New(0)

	for _, test := range tests {
		matches := 0
		misses := 0

		rule := Rule{Condition: test}
		err := rule.EnumerateTestCases(rnd, func(sample *st.State) ConsumerResult {
			match, err := test.Check(sample)
			if err != nil {
				t.Errorf("Condition check error %v", err)
			}
			if match {
				matches++
			} else {
				misses++
			}
			return ConsumeContinue
		})
		if err != nil {
			t.Errorf("EnumerateTestCases failed %v", err)
		}
		if matches == 0 {
			t.Errorf("none of the %d generated samples for %v is a match", matches+misses, test)
		}
		if matches+misses > 1 && misses == 0 {
			t.Errorf("none of the %d generated samples for %v is a miss", matches+misses, test)
		}
	}
}

func TestRule_GetTestCaseEnumerationInfo(t *testing.T) {
	conditions := []Condition{
		Eq(Status(), st.Failed),
		Eq(Pc(), NewU256(42)),
	}

	rule := &Rule{
		Condition: And(conditions...),
		Parameter: []Parameter{
			GasParameter{},
			MemoryOffsetParameter{},
		},
	}

	info := rule.GetTestCaseEnumerationInfo()

	if want, got := len(conditions), len(info.conditions); want != got {
		t.Fatalf("unexpected length of condition domain sizes, wanted %d, got %d", want, got)
	}
	sizes := map[Property]int{
		Status().Property(): len(statusCodeDomain{}.Samples(st.Failed)),
		Pc().Property():     len(pcDomain{}.Samples(NewU256(42))),
	}
	total := 1
	for _, property := range maps.Keys(sizes) {
		if want, got := sizes[property], len(info.propertyDomains[property]); want != got {
			t.Errorf("unexpected domain size for %v, wanted %d, got %d", property, want, got)
		}
		total *= sizes[property]
	}

	if want, got := len(rule.Parameter), len(info.parameterDomainSizes); want != got {
		t.Fatalf("unexpected number of parameter domain sizes, wanted %d, got %d", want, got)
	}
	for i, got := range info.parameterDomainSizes {
		want := len(rule.Parameter[i].Samples()) + 1 // 1 random value
		if want != got {
			t.Errorf("unexpected size of parameter domain %d - wanted %d, got %d", i, want, got)
		}
		total *= want
	}

	if got := info.TotalNumberOfCases(); total != got {
		t.Errorf("unexpected total number of test cases, wanted %d, got %d", total, got)
	}
}

func TestTestCaseEnumerationInfo_PrintEmptyIsNice(t *testing.T) {
	info := TestCaseEnumerationInfo{}
	print := info.String()
	if !strings.Contains(print, "Conditions:\n\t-none-") {
		t.Errorf("missing summary for conditions, got %s", print)
	}
	if !strings.Contains(print, "Domains:\n\t-none-") {
		t.Errorf("missing summary for property domains, got %s", print)
	}
	if !strings.Contains(print, "Parameters:\n\t-none-") {
		t.Errorf("missing summary for parameters, got %s", print)
	}
	if !strings.Contains(print, "Total number of cases: 1") {
		t.Errorf("missing summary of total test cases, got %s", print)
	}
}

func TestTestCaseEnumerationInfo_ConditionsAreSortedAlphabetically(t *testing.T) {
	info := TestCaseEnumerationInfo{
		conditions: []string{"b", "a"},
	}
	print := info.String()
	if !strings.Contains(print, "a\n\tb") {
		t.Errorf("constraints are not listed as expected, got %s", print)
	}
}

func TestTestCaseEnumerationInfo_PropertyDomainsAreSortedAlphabetically(t *testing.T) {
	info := TestCaseEnumerationInfo{
		propertyDomains: map[Property][]string{
			"b": {"x", "y", "z"},
			"a": {"1", "2"},
		},
	}
	print := info.String()
	if !strings.Contains(print, "a: N=2, {1, 2}\n\tb: N=3, {x, y, z}") {
		t.Errorf("domains are not listed as expected, got %s", print)
	}
}

func TestTestCaseEnumerationInfo_ParameterDomainsArePrintedInOrder(t *testing.T) {
	info := TestCaseEnumerationInfo{
		parameterDomainSizes: []int{10, 12, 14},
	}
	print := info.String()
	if !strings.Contains(print, "0: 10\n\t1: 12\n\t2: 14") {
		t.Errorf("parameter domains are not listed as expected, got %s", print)
	}
}
