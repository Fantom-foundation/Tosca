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
	"strings"
	"testing"

	. "github.com/Fantom-foundation/Tosca/go/ct/common"
)

func TestStorage_NewStorage(t *testing.T) {
	s := NewStorageBuilder().
		SetCurrent(NewU256(42), NewU256(1)).
		SetOriginal(NewU256(42), NewU256(2)).
		SetWarm(NewU256(42), true).
		Build()

	if want, got := true, s.IsWarm(NewU256(42)); want != got {
		t.Fatalf("IsWarm is broken, want %v, got %v", want, got)
	}
	if want, got := false, s.IsWarm(NewU256(43)); want != got {
		t.Fatalf("IsWarm is broken, want %v, got %v", want, got)
	}
}

func TestStorage_Clone(t *testing.T) {
	tests := map[string]struct {
		change func(*Storage)
	}{
		"set-current": {func(s *Storage) {
			s.SetCurrent(NewU256(1), NewU256(17))
		}},
		"set-original": {func(s *Storage) {
			s.SetOriginal(NewU256(1), NewU256(17))
		}},
		"set-warm": {func(s *Storage) {
			s.MarkWarm(NewU256(1))
		}},
		"remove-current": {func(s *Storage) {
			s.RemoveCurrent(NewU256(42))
		}},
		"remove-original": {func(s *Storage) {
			s.RemoveOriginal(NewU256(42))
		}},
		"unset-warm": {func(s *Storage) {
			s.MarkCold(NewU256(42))
		}},
	}

	s1 := NewStorageBuilder().
		SetCurrent(NewU256(42), NewU256(1)).
		SetOriginal(NewU256(42), NewU256(2)).
		SetWarm(NewU256(42), true).
		Build()

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			s2 := s1.Clone()
			if !s1.Eq(s2) {
				t.Fatalf("clones are not equal")
			}
			test.change(s2)
			if s1.Eq(s2) {
				t.Errorf("clones are not independent")
			}
		})
	}
}

func TestStorage_Diff(t *testing.T) {
	s1 := NewStorageBuilder().
		SetCurrent(NewU256(42), NewU256(1)).
		SetOriginal(NewU256(42), NewU256(2)).
		SetWarm(NewU256(42), true).
		Build()

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
	s1 := NewStorageBuilder().Build()

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
	original := NewStorageBuilder().
		SetCurrent(keyAndValue, keyAndValue).
		SetOriginal(keyAndValue, keyAndValue).
		SetWarm(keyAndValue, true).
		Build()

	for i := 0; i < b.N; i++ {
		clone := original.Clone()

		_ = clone.GetCurrent(keyAndValue)
		_ = clone.GetOriginal(keyAndValue)
		_ = clone.IsWarm(keyAndValue)
	}
}

func BenchmarkStorage_CloneModified(b *testing.B) {
	builder := NewStorageBuilder()
	for i := 0; i < 32; i++ {
		builder.SetCurrent(NewU256(uint64(i)), NewU256(uint64(i)))
		builder.SetOriginal(NewU256(uint64(8+i)), NewU256(uint64(i)))
		builder.SetWarm(NewU256(uint64(16+i)), i%2 == 0)
	}
	original := builder.Build()
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
