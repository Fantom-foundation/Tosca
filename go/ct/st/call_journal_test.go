// Copyright (c) 2024 Fantom Foundation
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at fantom.foundation/bsl11.
//
// Change Date: 2028-4-16
//
// On the date above, in accordance with the Business Source License, use of
// this software will be governed by the GNU Lesser General Public License v3.

package st

import (
	"bytes"
	"strings"
	"testing"

	"github.com/Fantom-foundation/Tosca/go/ct/common"
	"github.com/Fantom-foundation/Tosca/go/tosca"
)

func TestCallJournal_CallMovesFutureToPastCall(t *testing.T) {
	journal := NewCallJournal()

	journal.Future = []FutureCall{{
		Success:   true,
		Output:    common.NewBytes([]byte{1, 2, 3}),
		GasCosts:  5,
		GasRefund: 2,
	}}

	res := journal.Call(tosca.StaticCall, tosca.CallParameters{
		Sender:      tosca.Address{1},
		Recipient:   tosca.Address{2},
		Value:       tosca.Value{3},
		Input:       []byte{4, 5},
		Gas:         6,
		Salt:        tosca.Hash{7},
		CodeAddress: tosca.Address{8},
	})

	if want, got := true, res.Success; want != got {
		t.Errorf("unexpected success result, wanted %t, got %t", want, got)
	}

	if want, got := []byte{1, 2, 3}, res.Output; !bytes.Equal(want, got) {
		t.Errorf("unexpected output, wanted %v, got %v", want, got)
	}

	if want, got := tosca.Gas(1), res.GasLeft; want != got {
		t.Errorf("unexpected remaining gas, wanted %d, got %d", want, got)
	}

	if want, got := tosca.Gas(2), res.GasRefund; want != got {
		t.Errorf("unexpected refund gas, wanted %d, got %d", want, got)
	}

	if len(journal.Past) != 1 {
		t.Fatalf("no past call was recorded")
	}
	want := PastCall{
		Kind:        tosca.StaticCall,
		Recipient:   tosca.Address{2},
		Sender:      tosca.Address{1},
		Input:       common.NewBytes([]byte{4, 5}),
		Value:       tosca.Value{3},
		Gas:         tosca.Gas(6),
		CodeAddress: tosca.Address{8},
	}

	if got := journal.Past[0]; !want.Equal(&got) {
		t.Errorf(
			"failed to record past call, wanted %v, got %v, diff %v",
			want, got, want.Diff(&got),
		)
	}
}

func TestCallJournal_EqualDetectsDifferences(t *testing.T) {
	tests := map[string]struct {
		modify func(c *CallJournal)
	}{
		"added_past":      {func(c *CallJournal) { c.Past = append(c.Past, PastCall{}) }},
		"removed_past":    {func(c *CallJournal) { c.Past = c.Past[0 : len(c.Past)-1] }},
		"modified_past":   {func(c *CallJournal) { c.Past[0].Gas++ }},
		"added_future":    {func(c *CallJournal) { c.Future = append(c.Future, FutureCall{}) }},
		"removed_future":  {func(c *CallJournal) { c.Future = c.Future[0 : len(c.Future)-1] }},
		"modified_future": {func(c *CallJournal) { c.Future[0].GasCosts++ }},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			j1 := NewCallJournal()
			j1.Past = []PastCall{{}}
			j1.Future = []FutureCall{{}}
			j2 := j1.Clone()
			test.modify(j2)
			if j1.Equal(j2) {
				t.Errorf("failed to detect difference between %v and %v", j1, j2)
			}
		})
	}
}

func TestCallJournal_DiffDetectsDifferences(t *testing.T) {
	tests := map[string]struct {
		modify func(c *CallJournal)
		issue  string
	}{
		"added_past": {
			func(c *CallJournal) { c.Past = append(c.Past, PastCall{}) },
			"different length of past calls",
		},
		"removed_past": {
			func(c *CallJournal) { c.Past = c.Past[0 : len(c.Past)-1] },
			"different length of past calls",
		},
		"modified_past": {
			func(c *CallJournal) { c.Past[0].Gas++ },
			"Past Call 0: different gas",
		},
		"added_future": {
			func(c *CallJournal) { c.Future = append(c.Future, FutureCall{}) },
			"different length of future calls",
		},
		"removed_future": {
			func(c *CallJournal) { c.Future = c.Future[0 : len(c.Future)-1] },
			"different length of future calls",
		},
		"modified_future": {
			func(c *CallJournal) { c.Future[0].GasCosts++ },
			"Future Call 0: different gas costs",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			j1 := NewCallJournal()
			j1.Past = []PastCall{{}}
			j1.Future = []FutureCall{{}}
			j2 := j1.Clone()
			test.modify(j2)

			diffs := j1.Diff(j2)
			if len(diffs) != 1 {
				t.Errorf("failed to detect difference, got %v", diffs)
			}

			diff := diffs[0]
			if !strings.Contains(diff, test.issue) {
				t.Errorf("invalid diff reported, expected a string with %s, got %s", test.issue, diff)
			}
		})
	}
}

func TestPastCall_EqualDetectsDifferences(t *testing.T) {
	tests := map[string]struct {
		modify func(c *PastCall)
	}{
		"kind":         {func(c *PastCall) { c.Kind = tosca.DelegateCall }},
		"recipient":    {func(c *PastCall) { c.Recipient[0]++ }},
		"sender":       {func(c *PastCall) { c.Sender[0]++ }},
		"input":        {func(c *PastCall) { c.Input = common.NewBytes([]byte{1, 2, 3}) }},
		"value":        {func(c *PastCall) { c.Value[0]++ }},
		"gas":          {func(c *PastCall) { c.Gas++ }},
		"code address": {func(c *PastCall) { c.CodeAddress[0]++ }},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			c1 := PastCall{}
			c2 := c1
			test.modify(&c2)
			if c1.Equal(&c2) {
				t.Errorf("failed to detect difference between %v and %v", c1, c2)
			}
		})
	}
}

func TestPastCall_DiffDetectsDifferences(t *testing.T) {
	tests := map[string]struct {
		modify func(c *PastCall)
	}{
		"kind":         {func(c *PastCall) { c.Kind = tosca.DelegateCall }},
		"recipient":    {func(c *PastCall) { c.Recipient[0]++ }},
		"sender":       {func(c *PastCall) { c.Sender[0]++ }},
		"input":        {func(c *PastCall) { c.Input = common.NewBytes([]byte{1, 2, 3}) }},
		"value":        {func(c *PastCall) { c.Value[0]++ }},
		"gas":          {func(c *PastCall) { c.Gas++ }},
		"code address": {func(c *PastCall) { c.CodeAddress[0]++ }},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			c1 := PastCall{}
			c2 := c1
			test.modify(&c2)
			diffs := c1.Diff(&c2)
			if want, got := 1, len(diffs); want != got {
				t.Fatalf("unexpected number of differences, wanted %d, got %d", want, got)
			}
			diff := diffs[0]
			if !strings.Contains(diff, name) {
				t.Errorf("unexpected diff, wanted string containing %s, got %s", name, diff)
			}
		})
	}
}

func TestFutureCall_EqualDetectsDifferences(t *testing.T) {
	tests := map[string]struct {
		modify func(c *FutureCall)
	}{
		"success":   {func(c *FutureCall) { c.Success = true }},
		"gas costs": {func(c *FutureCall) { c.GasCosts++ }},
		"refund":    {func(c *FutureCall) { c.GasRefund++ }},
		"output":    {func(c *FutureCall) { c.Output = common.NewBytes([]byte{1, 2, 3}) }},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			c1 := FutureCall{}
			c2 := c1
			test.modify(&c2)
			if c1.Equal(&c2) {
				t.Errorf("failed to detect difference between %v and %v", c1, c2)
			}
		})
	}
}

func TestFutureCall_DiffDetectsDifferences(t *testing.T) {
	tests := map[string]struct {
		modify func(c *FutureCall)
	}{
		"success":   {func(c *FutureCall) { c.Success = true }},
		"gas costs": {func(c *FutureCall) { c.GasCosts++ }},
		"refund":    {func(c *FutureCall) { c.GasRefund++ }},
		"output":    {func(c *FutureCall) { c.Output = common.NewBytes([]byte{1, 2, 3}) }},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			c1 := FutureCall{}
			c2 := c1
			test.modify(&c2)
			diffs := c1.Diff(&c2)
			if want, got := 1, len(diffs); want != got {
				t.Fatalf("unexpected number of differences, wanted %d, got %d", want, got)
			}
			diff := diffs[0]
			if !strings.Contains(diff, name) {
				t.Errorf("unexpected diff, wanted string containing %s, got %s", name, diff)
			}
		})
	}
}
