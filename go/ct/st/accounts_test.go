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
	"fmt"
	"strings"
	"testing"

	. "github.com/Fantom-foundation/Tosca/go/ct/common"
	"github.com/Fantom-foundation/Tosca/go/tosca"
	"github.com/Fantom-foundation/Tosca/go/tosca/vm"
)

func TestAccounts_MarkWarmMarksAddressesAsWarm(t *testing.T) {
	b := NewAccounts()
	b.MarkWarm(NewAddressFromInt(42))

	if want, got := true, b.IsWarm(NewAddressFromInt(42)); want != got {
		t.Fatalf("IsWarm is broken, want %v, got %v", want, got)
	}
	if want, got := false, b.IsWarm(NewAddressFromInt(8)); want != got {
		t.Fatalf("IsWarm is broken, want %v, got %v", want, got)
	}
}

func TestAccounts_SetWarm(t *testing.T) {
	acc := NewAccounts()
	addr := NewAddressFromInt(42)
	if want, got := false, acc.IsWarm(addr); want != got {
		t.Fatalf("IsWarm is broken, want %v, got %v", want, got)
	}
	acc.MarkWarm(addr)
	if want, got := true, acc.IsWarm(addr); want != got {
		t.Fatalf("MarkWarm is broken, want %v, got %v", want, got)
	}
}

func TestAccounts_Clone(t *testing.T) {
	a := NewAddressFromInt(42)
	b := NewAddressFromInt(48)
	tests := map[string]struct {
		change func(*Accounts)
	}{
		"modify-balance": {func(accounts *Accounts) {
			accounts.SetBalance(b, NewU256(3))
		}},
		"mark-warm": {func(accounts *Accounts) {
			accounts.MarkWarm(b)
		}},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			b1 := NewAccounts()
			b1.SetBalance(a, NewU256(1))
			b1.MarkWarm(a)
			b2 := b1.Clone()
			if !b1.Eq(b2) {
				t.Fatalf("clones are not equal")
			}
			test.change(b2)
			if b1.Eq(b2) {
				t.Errorf("clones are not independent")
			}
		})
	}
}

func TestAccounts_AccountsWithZeroBalanceAreTreatedTheSameByEqAndDiff(t *testing.T) {
	a1 := NewAccounts()
	a1.SetBalance(tosca.Address{1}, NewU256(0))
	a2 := NewAccounts()

	equal := a1.Eq(a2)
	diff := a1.Diff(a2)

	if equal != (len(diff) == 0) {
		t.Errorf("Eq and Diff not compatible, Eq returns %t, Diff %v", equal, diff)
	}
}

func TestAccounts_Diff(t *testing.T) {
	a := NewAddressFromInt(42)
	b := NewAddressFromInt(48)
	tests := map[string]struct {
		change  func(*Accounts)
		outcome string
	}{
		"modify-balance": {func(accounts *Accounts) {
			accounts.SetBalance(a, NewU256(3))
		}, "Different account entry"},
		"mark-warm": {func(accounts *Accounts) {
			accounts.MarkWarm(b)
		}, "Different account warm entry"},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			a1 := NewAccountsBuilder().
				SetBalance(a, NewU256(1)).
				SetCode(a, NewBytes([]byte{byte(vm.SUB), byte(vm.SWAP1), 5, byte(vm.PUSH2)})).
				SetWarm(a).
				Build()
			a2 := a1.Clone()
			diff := a1.Diff(a2)
			if len(diff) != 0 {
				t.Errorf("clones are different: %v", diff)
			}
			test.change(a2)
			diff = a1.Diff(a2)
			if !strings.Contains(diff[0], test.outcome) {
				t.Errorf("difference in accounts not found: %v, wanted: %v", diff, test.outcome)
			}
		})
	}
}

func TestAccounts_IsEmptyDependsOnBalanceAndCode(t *testing.T) {
	zero := NewU256(0)
	nonzero := NewU256(1)
	tests := map[string]struct {
		balance *U256
		code    []byte
		empty   bool
	}{
		"no_balance_no_code":                 {empty: true},
		"zero_balance_no_code":               {balance: &zero, empty: true},
		"nonzero_balance_no_code":            {balance: &nonzero, empty: false},
		"no_balance_with_empty_code":         {code: []byte{}, empty: true},
		"no_balance_with_nonempty_code":      {code: []byte{1, 2, 3}, empty: false},
		"nonzero_balance_with_empty_code":    {balance: &nonzero, code: []byte{}, empty: false},
		"nonzero_balance_with_nonempty_code": {balance: &nonzero, code: []byte{1, 2, 3}, empty: false},
	}

	addr := tosca.Address{}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			builder := NewAccountsBuilder()
			if test.balance != nil {
				builder.SetBalance(addr, *test.balance)
			}
			if test.code != nil {
				builder.SetCode(addr, NewBytes(test.code))
			}
			accounts := builder.Build()
			if want, got := test.empty, accounts.IsEmpty(addr); want != got {
				t.Errorf("unexpected result, wanted %t, got %t", want, got)
			}
		})
	}

}

func TestAccounts_Exists(t *testing.T) {
	acc := NewAccounts()
	addr := NewAddressFromInt(42)
	if want, got := false, acc.Exists(addr); want != got {
		t.Errorf("Exists is broken, want %v but got %v", want, got)
	}
	acc.SetBalance(addr, NewU256(0))
	if want, got := true, acc.Exists(addr); want != got {
		t.Errorf("Exists is broken, want %v but got %v", want, got)
	}
	acc.SetBalance(addr, NewU256(1))
	if want, got := true, acc.Exists(addr); want != got {
		t.Errorf("Exists is broken, want %v but got %v", want, got)
	}
}

func TestAccountsBuilder_NewAccountsBuilder(t *testing.T) {
	addr1 := NewAddressFromInt(42)
	addr2 := NewAddressFromInt(24)
	ab := NewAccountsBuilder()
	ab.SetBalance(addr1, NewU256(1))
	ab.SetCode(addr2, NewBytes([]byte{1, 2, 3}))
	ab.SetWarm(addr1)
	ab.SetWarm(addr2)
	acc := ab.Build()
	if want, got := NewU256(1), acc.GetBalance(addr1); !want.Eq(got) {
		t.Errorf("AccountsBuilder balance is broken, want %v but got %v", want, got)
	}
	if want, got := NewBytes([]byte{1, 2, 3}), acc.GetCode(addr2); want != got {
		t.Errorf("AccountsBuilder code is broken, want %v but got %v", want, got)
	}
	if want, got := true, acc.IsWarm(addr1) && acc.IsWarm(addr2); want != got {
		t.Errorf("AccountsBuilder warm is broken, want %v but got %v", want, got)
	}
}

func TestAccounts_String(t *testing.T) {
	addr := NewAddressFromInt(42)
	builder := NewAccountsBuilder()
	builder.SetBalance(addr, NewU256(1))
	builder.SetCode(addr, NewBytes([]byte{1}))
	builder.SetWarm(addr)
	want := "Accounts:\n"
	want += fmt.Sprintf("\t%v:\n", addr)
	want += fmt.Sprintf("\t\tBalance: %v\n", NewU256(1))
	want += fmt.Sprintf("\t\tCode: %v\n", NewBytes([]byte{1}))
	want += fmt.Sprintf("Warm Accounts:\n\t\t%v\n", addr)
	if got := builder.Build().String(); want != got {
		t.Errorf("Accounts.String broken, wanted:\n %v\n but got:\n %v", want, got)
	}
}

// -- Benchmarks

func accountInit(a tosca.Address) *Accounts {
	ab := NewAccountsBuilder()
	ab.SetBalance(a, NewU256(1))
	ab.SetCode(a, NewBytes([]byte{byte(vm.SUB), byte(vm.SWAP1), 5, byte(vm.PUSH2)}))
	ab.SetWarm(a)
	acc := ab.Build()
	return acc
}

func BenchmarkAccountClone(b *testing.B) {
	a := NewAddressFromInt(42)
	b1 := accountInit(a)
	for i := 0; i < b.N; i++ {
		b1.Clone()
	}
}

func BenchmarkAccountCloneModifyBalance(b *testing.B) {
	a := NewAddressFromInt(42)
	b1 := accountInit(a)
	for i := 0; i < b.N; i++ {
		b2 := b1.Clone()
		b2.SetBalance(a, NewU256(3))
	}
}
