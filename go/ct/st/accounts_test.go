package st

import (
	"fmt"
	"strings"
	"testing"

	. "github.com/Fantom-foundation/Tosca/go/ct/common"
	"github.com/Fantom-foundation/Tosca/go/vm"
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
	if want, got := true, acc.IsCold(addr); want != got {
		t.Fatalf("IsCold is broken, want %v, got %v", want, got)
	}
	acc.SetWarm(addr, true)
	if want, got := false, acc.IsCold(addr); want != got {
		t.Fatalf("SetWarm is broken, want %v, got %v", want, got)
	}
	acc.SetWarm(addr, false)
	if want, got := true, acc.IsCold(addr); want != got {
		t.Fatalf("SetWarm to cold is broken, want %v, got %v", want, got)
	}

}

func TestAccounts_Clone(t *testing.T) {
	a := NewAddressFromInt(42)
	b := NewAddressFromInt(48)
	tests := map[string]struct {
		change func(*Accounts)
	}{
		"add-balance": {func(accounts *Accounts) {
			accounts.balance[b] = NewU256(3)
		}},
		"modify-balance": {func(accounts *Accounts) {
			accounts.balance[a] = NewU256(3)
		}},
		"remove-balance": {func(accounts *Accounts) {
			delete(accounts.balance, a)
		}},
		"add-code": {func(accounts *Accounts) {
			accounts.code[b] = NewBytes([]byte{byte(ADD), byte(PUSH1), 5, byte(PUSH2)})
		}},
		"modify-code": {func(accounts *Accounts) {
			accounts.code[a] = NewBytes([]byte{byte(SUB), byte(BALANCE), 5, byte(SHA3)})
		}},
		"remove-code": {func(accounts *Accounts) {
			delete(accounts.code, a)
		}},
		"mark-cold": {func(accounts *Accounts) {
			accounts.MarkCold(a)
		}},
		"mark-warm": {func(accounts *Accounts) {
			accounts.MarkWarm(b)
		}},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			b1 := NewAccounts()
			b1.balance[a] = NewU256(1)
			b1.code[a] = NewBytes([]byte{byte(SUB), byte(SWAP1), 5, byte(PUSH2)})
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
	a1.balance[vm.Address{1}] = NewU256(0)
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
		"add-balance": {func(accounts *Accounts) {
			accounts.balance[b] = NewU256(3)
		}, "Different balance entry"},
		"modify-balance": {func(accounts *Accounts) {
			accounts.balance[a] = NewU256(3)
		}, "Different balance entry"},
		"remove-balance": {func(accounts *Accounts) {
			delete(accounts.balance, a)
		}, "Different balance entry"},
		"add-code": {func(accounts *Accounts) {
			accounts.code[b] = NewBytes([]byte{byte(ADD), byte(PUSH1), 5, byte(PUSH2)})
		}, "Different code entry"},
		"modify-code": {func(accounts *Accounts) {
			accounts.code[a] = NewBytes([]byte{byte(SUB), byte(BALANCE), 5, byte(SHA3)})
		}, "Different code entry"},
		"remove-code": {func(accounts *Accounts) {
			delete(accounts.code, a)
		}, "Different code entry"},
		"mark-cold": {func(accounts *Accounts) {
			accounts.MarkCold(a)
		}, "Different account warm entry"},
		"mark-warm": {func(accounts *Accounts) {
			accounts.MarkWarm(b)
		}, "Different account warm entry"},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			a1 := NewAccounts()
			a1.balance[a] = NewU256(1)
			a1.code[a] = NewBytes([]byte{byte(SUB), byte(SWAP1), 5, byte(PUSH2)})
			a1.MarkWarm(a)
			a2 := a1.Clone()
			diff := a1.Diff(a2)
			if len(diff) != 0 {
				t.Errorf("clones are different: %v", diff)
			}
			test.change(a2)
			diff = a1.Diff(a2)
			if !strings.Contains(diff[0], test.outcome) {
				t.Errorf("difference in accounts not found: %v", diff)
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

	addr := vm.Address{}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			accounts := NewAccounts()
			if test.balance != nil {
				accounts.balance[addr] = *test.balance
			}
			if test.code != nil {
				accounts.code[addr] = NewBytes(test.code)
			}
			if want, got := test.empty, accounts.IsEmpty(addr); want != got {
				t.Errorf("unexpected result, wanted %t, got %t", want, got)
			}
		})
	}

}

func TestAccounts_Exists(t *testing.T) {
	acc := NewAccounts()
	addr := NewAddressFromInt(42)
	if want, got := false, acc.Exist(addr); want != got {
		t.Errorf("Exist is broken, want %v but got %v", want, got)
	}
	acc.SetBalance(addr, NewU256(1))
	if want, got := true, acc.Exist(addr); want != got {
		t.Errorf("Exist is broken, want %v but got %v", want, got)
	}
	delete(acc.balance, addr)
	if want, got := false, acc.Exist(addr); want != got {
		t.Errorf("Exist is broken, want %v but got %v", want, got)
	}
	acc.SetCode(addr, NewBytes([]byte{1}))
	if want, got := true, acc.Exist(addr); want != got {
		t.Errorf("Exist is broken, want %v but got %v", want, got)
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
		t.Errorf("AccountsBuilder balance is broken, wante %v but got %v", want, got)
	}
	if want, got := NewBytes([]byte{1, 2, 3}), acc.GetCode(addr2); want != got {
		t.Errorf("AccountsBuilder code is broken, wante %v but got %v", want, got)
	}
	if want, got := true, acc.IsWarm(addr1) && acc.IsWarm(addr2); want != got {
		t.Errorf("AccountsBuilder warm is broken, wante %v but got %v", want, got)
	}
}

func TestAccounts_String(t *testing.T) {
	addr := NewAddressFromInt(42)
	acc := NewAccounts()
	acc.balance[addr] = NewU256(1)
	acc.code[addr] = NewBytes([]byte{1})
	acc.warm[addr] = struct{}{}
	want := fmt.Sprintf("\tAccount.Balance:\n\t    [%v]=%v\n", addr, NewU256(1))
	want += fmt.Sprintf("\tAccount.Code:\n\t    [%v]=%v\n", addr, NewBytes([]byte{1}))
	want += fmt.Sprintf("\tAccount.Warm:\n\t    [%v]={}\n", addr)
	if got := acc.String(); want != got {
		t.Errorf("Accounts.String broken, wanted:\n %v\n but got:\n %v", want, got)
	}
}

// -- Benchmarks

func accountInit(a vm.Address) *Accounts {
	ab := NewAccountsBuilder()
	ab.SetBalance(a, NewU256(1))
	ab.SetCode(a, NewBytes([]byte{byte(SUB), byte(SWAP1), 5, byte(PUSH2)}))
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

func BenchmarkAccountCloneModifyCode(b *testing.B) {
	a := NewAddressFromInt(42)
	b1 := accountInit(a)
	for i := 0; i < b.N; i++ {
		b2 := b1.Clone()
		b2.SetCode(a, NewBytes([]byte{byte(ADD), byte(PUSH1), 5, byte(PUSH2)}))
	}
}

func BenchmarkAccountCloneModifyWarm(b *testing.B) {
	a := NewAddressFromInt(42)
	b1 := accountInit(a)
	for i := 0; i < b.N; i++ {
		b2 := b1.Clone()
		b2.SetWarm(a, false)
	}
}
