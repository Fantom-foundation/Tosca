package st

import (
	"strings"
	"testing"

	. "github.com/Fantom-foundation/Tosca/go/ct/common"
)

func TestBalance_MarkWarmMarksAddressesAsWarm(t *testing.T) {
	b := NewBalance()
	b.MarkWarm(NewAddressFromInt(42))

	if want, got := true, b.IsWarm(NewAddressFromInt(42)); want != got {
		t.Fatalf("IsWarm is broken, want %v, got %v", want, got)
	}
	if want, got := false, b.IsWarm(NewAddressFromInt(43)); want != got {
		t.Fatalf("IsWarm is broken, want %v, got %v", want, got)
	}
}

func TestBalance_Clone(t *testing.T) {
	a := NewAddressFromInt(42)
	b := NewAddressFromInt(48)
	tests := map[string]struct {
		change func(*Balance)
	}{
		"add-balance": {func(balance *Balance) {
			balance.Current[b] = NewU256(3)
		}},
		"modify-balance": {func(balance *Balance) {
			balance.Current[a] = NewU256(3)
		}},
		"remove-balance": {func(balance *Balance) {
			delete(balance.Current, a)
		}},
		"mark-cold": {func(balance *Balance) {
			balance.MarkCold(a)
		}},
		"mark-warm": {func(balance *Balance) {
			balance.MarkWarm(b)
		}},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			b1 := NewBalance()
			b1.Current[a] = NewU256(1)
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

func TestBalance_AccountsWithZeroBalanceAreTreatedTheSameByEqAndDiff(t *testing.T) {
	b1 := NewBalance()
	b1.Current[Address{1}] = NewU256(0)
	b2 := NewBalance()

	equal := b1.Eq(b2)
	diff := b1.Diff(b2)

	if equal != (len(diff) == 0) {
		t.Errorf("Eq and Diff not compatible, Eq returns %t, Diff %v", equal, diff)
	}
}

func TestBalance_Diff(t *testing.T) {
	a := NewAddressFromInt(42)
	b := NewAddressFromInt(48)
	tests := map[string]struct {
		change  func(*Balance)
		outcome string
	}{
		"add-balance": {func(balance *Balance) {
			balance.Current[b] = NewU256(3)
		}, "Different current entry"},
		"modify-balance": {func(balance *Balance) {
			balance.Current[a] = NewU256(3)
		}, "Different current entry"},
		"remove-balance": {func(balance *Balance) {
			delete(balance.Current, a)
		}, "Different current entry"},
		"mark-cold": {func(balance *Balance) {
			balance.MarkCold(a)
		}, "Different warm entry"},
		"mark-warm": {func(balance *Balance) {
			balance.MarkWarm(b)
		}, "Different warm entry"},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			b1 := NewBalance()
			b1.Current[a] = NewU256(1)
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
