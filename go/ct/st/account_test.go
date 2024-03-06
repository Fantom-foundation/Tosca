package st

import (
	"strings"
	"testing"

	. "github.com/Fantom-foundation/Tosca/go/ct/common"
)

func TestAccount_MarkWarmMarksAddressesAsWarm(t *testing.T) {
	b := NewAccount()
	b.MarkWarm(NewAddressFromInt(42))

	if want, got := true, b.IsWarm(NewAddressFromInt(42)); want != got {
		t.Fatalf("IsWarm is broken, want %v, got %v", want, got)
	}
	if want, got := false, b.IsWarm(NewAddressFromInt(43)); want != got {
		t.Fatalf("IsWarm is broken, want %v, got %v", want, got)
	}
}

func TestAccount_Clone(t *testing.T) {
	a := NewAddressFromInt(42)
	b := NewAddressFromInt(48)
	tests := map[string]struct {
		change func(*Account)
	}{
		"add-balance": {func(account *Account) {
			account.Balance[b] = NewU256(3)
		}},
		"modify-balance": {func(account *Account) {
			account.Balance[a] = NewU256(3)
		}},
		"remove-balance": {func(account *Account) {
			delete(account.Balance, a)
		}},
		"mark-cold": {func(account *Account) {
			account.MarkCold(a)
		}},
		"mark-warm": {func(account *Account) {
			account.MarkWarm(b)
		}},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			b1 := NewAccount()
			b1.Balance[a] = NewU256(1)
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

func TestAccount_AccountsWithZeroBalanceAreTreatedTheSameByEqAndDiff(t *testing.T) {
	b1 := NewAccount()
	b1.Balance[Address{1}] = NewU256(0)
	b2 := NewAccount()

	equal := b1.Eq(b2)
	diff := b1.Diff(b2)

	if equal != (len(diff) == 0) {
		t.Errorf("Eq and Diff not compatible, Eq returns %t, Diff %v", equal, diff)
	}
}

func TestAccount_Diff(t *testing.T) {
	a := NewAddressFromInt(42)
	b := NewAddressFromInt(48)
	tests := map[string]struct {
		change  func(*Account)
		outcome string
	}{
		"add-balance": {func(account *Account) {
			account.Balance[b] = NewU256(3)
		}, "Different balance entry"},
		"modify-balance": {func(account *Account) {
			account.Balance[a] = NewU256(3)
		}, "Different balance entry"},
		"remove-balance": {func(account *Account) {
			delete(account.Balance, a)
		}, "Different balance entry"},
		"mark-cold": {func(account *Account) {
			account.MarkCold(a)
		}, "Different warm entry"},
		"mark-warm": {func(account *Account) {
			account.MarkWarm(b)
		}, "Different warm entry"},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			b1 := NewAccount()
			b1.Balance[a] = NewU256(1)
			b1.MarkWarm(a)
			b2 := b1.Clone()
			diff := b1.Diff(b2)
			if len(diff) != 0 {
				t.Errorf("Clone are different: %v", diff)
			}
			test.change(b2)
			diff = b1.Diff(b2)
			if !strings.Contains(diff[0], test.outcome) {
				t.Errorf("Difference in balance not found: %v", diff)
			}
		})
	}
}
