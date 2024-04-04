package st

import (
	"strings"
	"testing"

	. "github.com/Fantom-foundation/Tosca/go/ct/common"
)

func TestStorage_NewStorage(t *testing.T) {
	s := NewStorage()
	s.SetCurrent(NewU256(42), NewU256(1))
	s.SetOriginal(NewU256(42), NewU256(2))
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
	s1.SetCurrent(NewU256(42), NewU256(1))
	s1.SetOriginal(NewU256(42), NewU256(2))
	s1.MarkWarm(NewU256(42))

	s2 := s1.Clone()
	if !s1.Eq(s2) {
		t.Fatalf("Clones are not equal")
	}

	s2.SetCurrent(NewU256(42), NewU256(3))
	if s1.Eq(s2) {
		t.Fatalf("Clones are not independent")
	}
	s2.SetCurrent(NewU256(42), NewU256(1))

	s2.SetOriginal(NewU256(42), NewU256(4))
	if s1.Eq(s2) {
		t.Fatalf("Clones are not independent")
	}
	s2.SetOriginal(NewU256(42), NewU256(2))

	s2.MarkCold(NewU256(42))
	if s1.Eq(s2) {
		t.Fatalf("Clones are not independent")
	}
	s2.MarkWarm(NewU256(42))
}

func TestStorage_Diff(t *testing.T) {
	s1 := NewStorage()
	s1.SetCurrent(NewU256(42), NewU256(1))
	s1.SetOriginal(NewU256(42), NewU256(2))
	s1.MarkWarm(NewU256(42))

	s2 := s1.Clone()

	diff := s1.Diff(s2)
	if len(diff) != 0 {
		t.Fatalf("Clone are different: %v", diff)
	}

	s2.SetCurrent(NewU256(42), NewU256(3))
	diff = s1.Diff(s2)
	if !strings.Contains(diff[0], "current") {
		t.Fatalf("Difference in current not found: %v", diff)
	}

	s2.RemoveCurrent(NewU256(42))
	diff = s1.Diff(s2)
	if !strings.Contains(diff[0], "current") {
		t.Fatalf("Difference in current not found: %v", diff)
	}

	s2 = s1.Clone()
	s2.SetOriginal(NewU256(42), NewU256(4))
	diff = s1.Diff(s2)
	if !strings.Contains(diff[0], "original") {
		t.Fatalf("Difference in original not found: %v", diff)
	}

	s2.RemoveOriginal(NewU256(42))
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
	s2.SetCurrent(NewU256(42), NewU256(0))

	diff := s1.Diff(s2)
	if len(diff) != 0 {
		t.Fatalf("Missing zero considered different: %v", diff)
	}
	if !s1.Eq(s2) || !s2.Eq(s1) {
		t.Fatalf("%v and %v considered different", s1.GetCurrent(NewU256(42)), s2.GetCurrent(NewU256(42)))
	}

	s2.SetCurrent(NewU256(42), NewU256(3))
	diff = s1.Diff(s2)
	if !strings.Contains(diff[0], "current") {
		t.Fatalf("Difference in current not found: %v", diff)
	}
	if s1.Eq(s2) || s2.Eq(s1) {
		t.Fatalf("%v and %v considered equal", s1.GetCurrent(NewU256(42)), s2.GetCurrent(NewU256(42)))
	}

	s1.SetCurrent(NewU256(42), NewU256(0))
	s2.SetCurrent(NewU256(42), NewU256(0))

	diff = s1.Diff(s2)
	if len(diff) != 0 {
		t.Fatalf("Zero values considered different: %v", diff)
	}
	if !s1.Eq(s2) || !s2.Eq(s1) {
		t.Fatalf("%v and %v considered different", s1.GetCurrent(NewU256(42)), s2.GetCurrent(NewU256(42)))
	}
}

func BenchmarkStorage_CloneNotModified(b *testing.B) {
	keyAndValue := NewU256(42)
	original := NewStorage()
	original.SetCurrent(keyAndValue, keyAndValue)
	original.SetOriginal(keyAndValue, keyAndValue)
	original.MarkWarm(keyAndValue)

	for i := 0; i < b.N; i++ {
		clone := original.Clone()

		_ = clone.GetCurrent(keyAndValue)
		_ = clone.GetOriginal(keyAndValue)
		_ = clone.IsWarm(keyAndValue)
	}
}

func BenchmarkStorage_CloneModified(b *testing.B) {
	original := NewStorage()
	for i := 0; i < 32; i++ {
		original.SetCurrent(NewU256(uint64(i)), NewU256(uint64(i)))
		original.SetOriginal(NewU256(uint64(8+i)), NewU256(uint64(i)))
		original.MarkWarmCold(NewU256(uint64(16+i)), i%2 == 0)
	}

	keyAndValue := NewU256(42)

	for i := 0; i < b.N; i++ {
		clone := original.Clone()
		clone.SetCurrent(keyAndValue, keyAndValue)
		clone.SetOriginal(keyAndValue, keyAndValue)
		clone.MarkWarm(keyAndValue)

		_ = clone.GetCurrent(keyAndValue)
		_ = clone.GetOriginal(keyAndValue)
		_ = clone.IsWarm(keyAndValue)
	}

}
