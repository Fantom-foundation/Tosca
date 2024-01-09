package st

import (
	"strings"
	"testing"

	. "github.com/Fantom-foundation/Tosca/go/ct/common"
)

func TestStorage_NewStorage(t *testing.T) {
	s := NewStorage()
	s.Current[NewU256(42)] = NewU256(1)
	s.Original[NewU256(42)] = NewU256(2)
	s.MarkWarm(NewU256(42))

	if want, got := true, s.IsWarm(NewU256(42)); want != got {
		t.Fatalf("IsWarm is broken, want %v, got %v", want, got)
	}
	if want, got := false, s.IsWarm(NewU256(43)); want != got {
		t.Fatalf("IsWarm is broken, want %v, got %v", want, got)
	}
}

func TestStorage_Clone(t *testing.T) {
	s1 := NewStorage()
	s1.Current[NewU256(42)] = NewU256(1)
	s1.Original[NewU256(42)] = NewU256(2)
	s1.MarkWarm(NewU256(42))

	s2 := s1.Clone()
	if !s1.Eq(s2) {
		t.Fatalf("Clones are not equal")
	}

	s2.Current[NewU256(42)] = NewU256(3)
	if s1.Eq(s2) {
		t.Fatalf("Clones are not independent")
	}
	s2.Current[NewU256(42)] = NewU256(1)

	s2.Original[NewU256(42)] = NewU256(4)
	if s1.Eq(s2) {
		t.Fatalf("Clones are not independent")
	}
	s2.Original[NewU256(42)] = NewU256(2)

	s2.MarkCold(NewU256(42))
	if s1.Eq(s2) {
		t.Fatalf("Clones are not independent")
	}
	s2.MarkWarm(NewU256(42))
}

func TestStorage_Diff(t *testing.T) {
	s1 := NewStorage()
	s1.Current[NewU256(42)] = NewU256(1)
	s1.Original[NewU256(42)] = NewU256(2)
	s1.MarkWarm(NewU256(42))

	s2 := s1.Clone()

	diff := s1.Diff(s2)
	if len(diff) != 0 {
		t.Fatalf("Clone are different: %v", diff)
	}

	s2.Current[NewU256(42)] = NewU256(3)
	diff = s1.Diff(s2)
	if !strings.Contains(diff[0], "current") {
		t.Fatalf("Difference in current not found: %v", diff)
	}

	delete(s2.Current, NewU256(42))
	diff = s1.Diff(s2)
	if !strings.Contains(diff[0], "current") {
		t.Fatalf("Difference in current not found: %v", diff)
	}

	s2 = s1.Clone()
	s2.Original[NewU256(42)] = NewU256(4)
	diff = s1.Diff(s2)
	if !strings.Contains(diff[0], "original") {
		t.Fatalf("Difference in original not found: %v", diff)
	}

	delete(s2.Original, NewU256(42))
	diff = s1.Diff(s2)
	if !strings.Contains(diff[0], "original") {
		t.Fatalf("Difference in original not found: %v", diff)
	}

	s2 = s1.Clone()
	s2.MarkCold(NewU256(42))
	diff = s1.Diff(s2)
	if !strings.Contains(diff[0], "warm") {
		t.Fatalf("Difference in warm not found: %v", diff)
	}

	s2 = s1.Clone()
	s2.MarkWarm(NewU256(43))
	diff = s1.Diff(s2)
	if !strings.Contains(diff[0], "warm") {
		t.Fatalf("Difference in warm not found: %v", diff)
	}
}

func TestStorage_ZeroConsideredPresent(t *testing.T) {
	s1 := NewStorage()

	s2 := s1.Clone()
	s2.Current[NewU256(42)] = NewU256(0)

	diff := s1.Diff(s2)
	if len(diff) != 0 {
		t.Fatalf("Missing zero considered different: %v", diff)
	}
	if !s1.Eq(s2) || !s2.Eq(s1) {
		t.Fatalf("%v and %v considered different", s1.Current[NewU256(42)], s2.Current[NewU256(42)])
	}

	s2.Current[NewU256(42)] = NewU256(3)
	diff = s1.Diff(s2)
	if !strings.Contains(diff[0], "current") {
		t.Fatalf("Difference in current not found: %v", diff)
	}
	if s1.Eq(s2) || s2.Eq(s1) {
		t.Fatalf("%v and %v considered equal", s1.Current[NewU256(42)], s2.Current[NewU256(42)])
	}

	s1.Current[NewU256(42)] = NewU256(0)
	s2.Current[NewU256(42)] = NewU256(0)

	diff = s1.Diff(s2)
	if len(diff) != 0 {
		t.Fatalf("Zero values considered different: %v", diff)
	}
	if !s1.Eq(s2) || !s2.Eq(s1) {
		t.Fatalf("%v and %v considered different", s1.Current[NewU256(42)], s2.Current[NewU256(42)])
	}
}
