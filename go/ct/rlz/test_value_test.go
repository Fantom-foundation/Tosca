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
	"slices"
	"testing"

	"github.com/Fantom-foundation/Tosca/go/tosca"
)

func TestGetPropertyTestValues_ExtractsTestValuesFromConstraints(t *testing.T) {
	values := getPropertyTestValues(And(
		Eq(StackSize(), 12),
		Gt(Gas(), tosca.Gas(5)),
		Le(Gas(), tosca.Gas(50)),
	))

	if want, got := 2, len(values); want != got {
		t.Fatalf("unexpected values size, wanted %d, got %d", want, got)
	}

	if len(values[Gas().Property()]) == 0 {
		t.Errorf("expected test values for gas, got %v", values)
	}
	if len(values[StackSize().Property()]) == 0 {
		t.Errorf("expected test values for stack size, got %v", values)
	}
}

func TestRemoveDuplicates_CanSuccessfullyRemoveDuplicates(t *testing.T) {
	tests := map[string][]uint16{
		"nil":            nil,
		"empty":          {},
		"single":         {1},
		"different":      {1, 2, 3},
		"unordered":      {3, 1, 2},
		"pair":           {1, 1},
		"multiple pairs": {1, 2, 1, 2},
	}

	for name, list := range tests {
		t.Run(name, func(t *testing.T) {

			input := []TestValue{}
			for _, cur := range list {
				input = append(input, NewTestValue(Property("X"), uint16Domain{}, cur, nil))
			}
			output := removeDuplicates(slices.Clone(input))

			// check that output contains only elements from the input
			for _, a := range output {
				found := slices.ContainsFunc(input, func(cur TestValue) bool {
					return cur.Compare(a) == 0
				})
				if !found {
					t.Errorf("unknown element in output: %v", a)
				}
			}

			// check that there are no duplicates
			for i, a := range output {
				for j, b := range output {
					if i != j && a.Compare(b) == 0 {
						t.Errorf("duplicated element in output: %v", a)
					}
				}
			}
		})
	}
}

func TestTestValue_Compare(t *testing.T) {
	newValue := func(property string, value uint16) TestValue {
		return NewTestValue(Property(property), uint16Domain{}, value, nil)
	}
	tests := []struct {
		a, b   TestValue
		result int
	}{
		{newValue("a", 1), newValue("a", 1), 0},
		{newValue("a", 1), newValue("a", 2), -1},
		{newValue("a", 1), newValue("b", 1), -1},
		{newValue("a", 2), newValue("a", 1), 1},
		{newValue("b", 1), newValue("a", 1), 1},
	}

	for _, test := range tests {
		want := test.result
		got := test.a.Compare(test.b)
		if want != got {
			t.Errorf("comparing %v and %v, wanted %d, got %d", test.a, test.b, want, got)
		}
	}
}
