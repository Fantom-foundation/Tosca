package st

import (
	"strings"
	"testing"

	. "github.com/Fantom-foundation/Tosca/go/ct/common"
)

func TestBalance_NewBalance(t *testing.T) {
	b := NewBalance()
	b.Current[NewAddressFromInt(42)] = NewU256(42)
	b.MarkWarm(NewAddressFromInt(42))

	if want, got := true, b.IsWarm(NewAddressFromInt(42)); want != got {
		t.Fatalf("IsWarm is broken, want %v, got %v", want, got)
	}
	if want, got := false, b.IsWarm(NewAddressFromInt(43)); want != got {
		t.Fatalf("IsWarm is broken, want %v, got %v", want, got)
	}
}

func TestBalance_Clone(t *testing.T) {
	b1 := NewBalance()
	a := NewAddressFromInt(42)
	b1.Current[a] = NewU256(1)
	b1.MarkWarm(a)

	b2 := b1.Clone()
	if !b1.Eq(b2) {
		t.Fatalf("Clones are not equal")
	}

	b2.Current[NewAddressFromInt(42)] = NewU256(3)
	if b1.Eq(b2) {
		t.Fatalf("Clones are not independent")
	}
	b2.Current[NewAddressFromInt(42)] = NewU256(1)

	b2.MarkCold(NewAddressFromInt(42))
	if b1.Eq(b2) {
		t.Fatalf("Clones are not independent")
	}
	b2.MarkWarm(NewAddressFromInt(42))
}

func TestBalance_Diff(t *testing.T) {
	b1 := NewBalance()
	b1.Current[NewAddressFromInt(42)] = NewU256(1)
	b1.MarkWarm(NewAddressFromInt(42))

	b2 := b1.Clone()

	diff := b1.Diff(b2)
	if len(diff) != 0 {
		t.Fatalf("Clone are different: %v", diff)
	}

	b2.Current[NewAddressFromInt(42)] = NewU256(3)
	diff = b1.Diff(b2)
	if !strings.Contains(diff[0], "current") {
		t.Fatalf("Difference in current not found: %v", diff)
	}

	delete(b2.Current, NewAddressFromInt(42))
	diff = b1.Diff(b2)
	if !strings.Contains(diff[0], "current") {
		t.Fatalf("Difference in current not found: %v", diff)
	}

	b2 = b1.Clone()
	b2.MarkCold(NewAddressFromInt(42))
	diff = b1.Diff(b2)
	if !strings.Contains(diff[0], "warm") {
		t.Fatalf("Difference in warm not found: %v", diff)
	}

	b2 = b1.Clone()
	b2.MarkWarm(NewAddressFromInt(43))
	diff = b1.Diff(b2)
	if !strings.Contains(diff[0], "warm") {
		t.Fatalf("Difference in warm not found: %v", diff)
	}
}
