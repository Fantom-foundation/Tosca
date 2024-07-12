// Copyright (c) 2024 Fantom Foundation
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at fantom.foundation/bsl11.
//
// Change Date: 2028-4-16
//
// On the date above, in accordance with the Business Source License, use of
// this software will be governed by the GNU Lesser General Public License v3.

package processor

import (
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/Fantom-foundation/Tosca/go/tosca"
	"github.com/Fantom-foundation/Tosca/go/tosca/vm"
)

func TestWorldState_Equal(t *testing.T) {

	tests := map[string]struct {
		a, b WorldState
	}{
		"both_nil": {},
		"left_hand_side_nil": {
			b: WorldState{},
		},
		"zero_accounts_are_ignored": {
			a: WorldState{
				{1}: Account{},
			},
			b: WorldState{
				{2}: Account{},
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			if !test.a.Equal(test.b) {
				t.Errorf("world states %v and %v are expected to be equivalent, but they are not", test.a, test.b)
			}
		})
	}
}

func TestWorldState_Clone(t *testing.T) {
	tests := map[string]struct {
		a WorldState
	}{
		"empty": {},
		"singleton": {
			a: WorldState{
				{1}: Account{},
			},
		},
		"multiple": {
			a: WorldState{
				{1}: Account{Balance: tosca.NewValue(100)},
				{2}: Account{Balance: tosca.NewValue(200)},
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			clone := test.a.Clone()
			if !test.a.Equal(clone) {
				t.Errorf("expected world state %v and its clone %v to be equal", test.a, clone)
			}
		})
	}
}

func TestWorldState_ClonesAreIndependent(t *testing.T) {
	addr := tosca.Address{1}
	key := tosca.Key{1}
	original := WorldState{
		addr: Account{Storage: Storage{
			key: {0x01},
		}},
	}

	clone := original.Clone()
	clone[addr].Storage[key] = tosca.Word{0x02}

	if original[addr].Storage[key] != (tosca.Word{0x01}) {
		t.Errorf("expected the original account to be independent from its clone")
	}
}

func TestWorldState_Diff(t *testing.T) {
	tests := map[string]struct {
		a, b     WorldState
		expected []string
	}{
		"both_nil": {},
		"identical": {
			a: WorldState{},
			b: WorldState{},
		},
		"different_accounts": {
			a:        WorldState{{1}: Account{Balance: tosca.NewValue(100)}},
			b:        WorldState{{1}: Account{Balance: tosca.NewValue(200)}},
			expected: []string{"0x0100000000000000000000000000000000000000/different balance: 100 != 200"},
		},
		"extra_accounts": {
			a: WorldState{
				{1}: Account{Balance: tosca.NewValue(100)},
			},
			b: WorldState{
				{1}: Account{Balance: tosca.NewValue(100)},
				{2}: Account{Balance: tosca.NewValue(200)},
			},
			expected: []string{"0x0200000000000000000000000000000000000000/different balance: 0 != 200"},
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			diffs := test.a.Diff(test.b)
			slices.Sort(test.expected)
			want := strings.Join(test.expected, ",")
			slices.Sort(diffs)
			got := strings.Join(diffs, ",")

			if want != got {
				t.Errorf("expected diffs [%v], but got [%v]", want, got)
			}
		})
	}
}

func TestAccount_Equal(t *testing.T) {
	tests := map[string]struct {
		a, b Account
	}{
		"both_zero": {},
		"identical_non_zero": {
			a: Account{
				Balance: tosca.NewValue(100),
				Nonce:   4,
				Code:    tosca.Code{byte(vm.STOP)},
				Storage: map[tosca.Key]tosca.Word{
					{1}: {0x01},
					{4}: {0x0F},
				},
			},
			b: Account{
				Balance: tosca.NewValue(100),
				Nonce:   4,
				Code:    tosca.Code{byte(vm.STOP)},
				Storage: map[tosca.Key]tosca.Word{
					{1}: {0x01},
					{4}: {0x0F},
				},
			},
		},
		"zero_storage_in_left_hand_side_is_ignored": {
			a: Account{Storage: map[tosca.Key]tosca.Word{{1}: {}}},
			b: Account{Storage: map[tosca.Key]tosca.Word{}},
		},
		"zero_storage_in_right_hand_side_is_ignored": {
			a: Account{Storage: map[tosca.Key]tosca.Word{}},
			b: Account{Storage: map[tosca.Key]tosca.Word{{1}: {}}},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			if !test.a.Equal(&test.b) {
				t.Errorf("expected accounts %v and %v to be equal", test.a, test.b)
			}
		})
	}
}

func TestAccount_NotEqual(t *testing.T) {
	tests := map[string]struct {
		a, b Account
	}{
		"different_balance": {
			a: Account{Balance: tosca.NewValue(100)},
			b: Account{Balance: tosca.NewValue(200)},
		},
		"different_nonce": {
			a: Account{Nonce: 4},
			b: Account{Nonce: 5},
		},
		"different_code": {
			a: Account{Code: tosca.Code{byte(vm.STOP)}},
			b: Account{Code: tosca.Code{byte(vm.ADD)}},
		},
		"different_storage": {
			a: Account{Storage: map[tosca.Key]tosca.Word{{1}: {0x01}}},
			b: Account{Storage: map[tosca.Key]tosca.Word{{1}: {0x02}}},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			if test.a.Equal(&test.b) {
				t.Errorf("expected accounts %v and %v to be not equal", test.a, test.b)
			}
		})
	}
}

func TestAccount_Clone(t *testing.T) {
	tests := map[string]struct {
		a Account
	}{
		"empty": {},
		"non_empty": {
			a: Account{
				Balance: tosca.NewValue(100),
				Nonce:   4,
				Code:    tosca.Code{byte(vm.STOP)},
				Storage: map[tosca.Key]tosca.Word{
					{1}: {0x01},
					{4}: {0x0F},
				},
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			clone := test.a.Clone()
			if !test.a.Equal(&clone) {
				t.Errorf("expected account %v and its clone %v to be equal", test.a, clone)
			}
		})
	}
}

func TestAccount_Diff(t *testing.T) {
	tests := map[string]struct {
		prefix   string
		a, b     Account
		expected []string
	}{
		"both_nil": {},
		"identical": {
			a: Account{},
			b: Account{},
		},
		"different_balance": {
			a:        Account{Balance: tosca.NewValue(100)},
			b:        Account{Balance: tosca.NewValue(200)},
			expected: []string{"different balance: 100 != 200"},
		},
		"different_nonce": {
			a:        Account{Nonce: 4},
			b:        Account{Nonce: 5},
			expected: []string{"different nonce: 4 != 5"},
		},
		"different_code": {
			a:        Account{Code: tosca.Code{byte(vm.STOP)}},
			b:        Account{Code: tosca.Code{byte(vm.ADD), byte(vm.MUL)}},
			expected: []string{"different code: 0x00 != 0x0102"},
		},
		"different_storage": {
			a:        Account{Storage: map[tosca.Key]tosca.Word{{1}: {0x01}}},
			b:        Account{Storage: map[tosca.Key]tosca.Word{{1}: {0x02}}},
			expected: []string{"Storage/different value for key 0x0100000000000000000000000000000000000000000000000000000000000000: 0x0100000000000000000000000000000000000000000000000000000000000000 != 0x0200000000000000000000000000000000000000000000000000000000000000"},
		},
		"different_balance_with_prefix": {
			prefix:   "myContext/",
			a:        Account{Balance: tosca.NewValue(100)},
			b:        Account{Balance: tosca.NewValue(200)},
			expected: []string{"myContext/different balance: 100 != 200"},
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			diffs := test.a.Diff(test.prefix, &test.b)
			slices.Sort(test.expected)
			want := strings.Join(test.expected, ",")
			slices.Sort(diffs)
			got := strings.Join(diffs, ",")

			if want != got {
				t.Errorf("expected diffs [%v], but got [%v]", want, got)
			}
		})
	}
}

func TestStorage_Equal(t *testing.T) {
	tests := map[string]struct {
		a, b Storage
	}{
		"both_nil": {},
		"left_hand_side_nil": {
			b: Storage{},
		},
		"right_hand_side_nil": {
			b: Storage{},
		},
		"singleton": {
			a: Storage{{1}: {0x01}},
			b: Storage{{1}: {0x01}},
		},
		"multiple": {
			a: Storage{
				{1}: {0x01},
				{2}: {0x02},
			},
			b: Storage{
				{1}: {0x01},
				{2}: {0x02},
			},
		},
		"zero_values_are_ignored": {
			a: Storage{{1}: {0x00}},
			b: Storage{{2}: {0x00}},
		},
		"zero_values_in_left_hand_side_is_ignored": {
			a: Storage{{1}: {0x00}},
			b: Storage{},
		},
		"zero_values_in_right_hand_side_is_ignored": {
			a: Storage{},
			b: Storage{{1}: {0x00}},
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			if !test.a.Equal(test.b) {
				t.Errorf("expected storages %v and %v to be equal", test.a, test.b)
			}
		})
	}
}

func TestStorage_NotEqual(t *testing.T) {
	tests := map[string]struct {
		a, b Storage
	}{
		"different_value_for_same_key": {
			a: Storage{{1}: {0x01}},
			b: Storage{{1}: {0x02}},
		},
		"extra_non_zero_entry_in_left_hand_side": {
			a: Storage{{1}: {0x01}, {2}: {0x02}},
			b: Storage{{1}: {0x01}},
		},
		"extra_non_zero_entry_in_right_hand_side": {
			a: Storage{{1}: {0x01}},
			b: Storage{{1}: {0x01}, {2}: {0x02}},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			if test.a.Equal(test.b) {
				t.Errorf("expected storages %v and %v to be not equal", test.a, test.b)
			}
		})
	}
}

func TestStorage_Clone(t *testing.T) {
	tests := map[string]struct {
		a Storage
	}{
		"empty": {},
		"singleton": {
			a: Storage{{1}: {0x01}},
		},
		"multiple": {
			a: Storage{
				{1}: {0x01},
				{2}: {0x02},
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			clone := test.a.Clone()
			if !test.a.Equal(clone) {
				t.Errorf("expected storage %v and its clone %v to be equal", test.a, clone)
			}
		})
	}
}

func TestStorage_Diff(t *testing.T) {
	tests := map[string]struct {
		prefix   string
		a, b     Storage
		expected []string
	}{
		"both_nil": {},
		"identical": {
			a: Storage{{1}: {0x01}},
			b: Storage{{1}: {0x01}},
		},
		"different_value": {
			a:        Storage{{1}: {0x01}},
			b:        Storage{{1}: {0x02}},
			expected: []string{"different value for key 0x0100000000000000000000000000000000000000000000000000000000000000: 0x0100000000000000000000000000000000000000000000000000000000000000 != 0x0200000000000000000000000000000000000000000000000000000000000000"},
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			diffs := test.a.Diff(test.prefix, test.b)
			slices.Sort(test.expected)
			want := strings.Join(test.expected, ",")
			slices.Sort(diffs)
			got := strings.Join(diffs, ",")

			if want != got {
				t.Errorf("expected diffs [%v], but got [%v]", want, got)
			}
		})
	}
}

func TestEqualMapsIgnoringZero_Equal(t *testing.T) {
	tests := map[string]struct {
		a, b map[int]int
	}{
		"both_nil": {},
		"left_hand_side_nil": {
			b: map[int]int{},
		},
		"right_hand_side_nil": {
			b: map[int]int{},
		},
		"singleton": {
			a: map[int]int{1: 2},
			b: map[int]int{1: 2},
		},
		"multiple": {
			a: map[int]int{
				1: 2,
				3: 4,
			},
			b: map[int]int{
				1: 2,
				3: 4,
			},
		},
		"zero_values_are_ignored": {
			a: map[int]int{1: 0},
			b: map[int]int{2: 0},
		},
		"zero_values_in_left_hand_side_is_ignored": {
			a: map[int]int{1: 0},
			b: map[int]int{},
		},
		"zero_values_in_right_hand_side_is_ignored": {
			a: map[int]int{},
			b: map[int]int{1: 0},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			if !equalMapsIgnoringZero(test.a, test.b, func(a, b int) bool { return a == b }) {
				t.Errorf("expected maps %v and %v to be equal", test.a, test.b)
			}
		})
	}
}

func TestEqualMapsIgnoringZero_NonEqual(t *testing.T) {
	tests := map[string]struct {
		a, b map[int]int
	}{
		"different_value_for_same_key": {
			a: map[int]int{1: 2},
			b: map[int]int{1: 3},
		},
		"extra_non_zero_entry_in_left_hand_side": {
			a: map[int]int{1: 2, 3: 4},
			b: map[int]int{1: 2},
		},
		"extra_non_zero_entry_in_right_hand_side": {
			a: map[int]int{1: 2},
			b: map[int]int{1: 2, 3: 4},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			if equalMapsIgnoringZero(test.a, test.b, func(a, b int) bool { return a == b }) {
				t.Errorf("expected maps %v and %v to be not equal", test.a, test.b)
			}
		})
	}
}

func TestDiffMaps(t *testing.T) {
	tests := map[string]struct {
		prefix   string
		a, b     map[int]int
		expected []string
	}{
		"both nil": {},
		"identical": {
			a: map[int]int{1: 2},
			b: map[int]int{1: 2},
		},
		"different value": {
			a:        map[int]int{1: 1},
			b:        map[int]int{1: 2},
			expected: []string{"different value for 1: 1 != 2"},
		},
		"additional key in first map": {
			a:        map[int]int{1: 2, 2: 4},
			b:        map[int]int{1: 2},
			expected: []string{"different value for 2: 4 != 0"},
		},
		"additional key in second map": {
			a:        map[int]int{1: 2},
			b:        map[int]int{1: 2, 2: 4},
			expected: []string{"different value for 2: 0 != 4"},
		},
		"different keys": {
			a: map[int]int{1: 3},
			b: map[int]int{2: 3},
			expected: []string{
				"different value for 1: 3 != 0",
				"different value for 2: 0 != 3",
			},
		},
		"different value with prefix": {
			prefix:   "myContext/",
			a:        map[int]int{1: 1},
			b:        map[int]int{1: 2},
			expected: []string{"myContext/different value for 1: 1 != 2"},
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			diffs := diffMaps(test.prefix, test.a, test.b, func(k int, a, b int) []string {
				if a == b {
					return nil
				}
				return []string{
					fmt.Sprintf("different value for %d: %d != %d", k, a, b),
				}
			})

			slices.Sort(test.expected)
			want := strings.Join(test.expected, ",")
			slices.Sort(diffs)
			got := strings.Join(diffs, ",")

			if want != got {
				t.Errorf("expected diffs [%v], but got [%v]", want, got)
			}
		})
	}
}
