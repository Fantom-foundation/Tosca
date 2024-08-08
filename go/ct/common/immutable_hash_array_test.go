// Copyright (c) 2024 Fantom Foundation
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at fantom.foundation/bsl11.
//
// Change Date: 2028-4-16
//
// On the date above, in accordance with the Business Source License, use of
// this software will be governed by the GNU Lesser General Public License v3.

package common

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/Fantom-foundation/Tosca/go/tosca"
	"pgregory.net/rand"
)

func TestImmutableHashArray_Equal(t *testing.T) {

	tests := map[string]struct {
		hashes1 ImmutableHashArray
		hashes2 ImmutableHashArray
		equal   bool
	}{
		"empty": {
			NewImmutableHashArray(), NewImmutableHashArray(), true,
		},
		"nil-first": {
			ImmutableHashArray{}, NewImmutableHashArray(), true,
		},
		"nil-second": {
			NewImmutableHashArray(), ImmutableHashArray{}, true,
		},
		"single": {
			NewImmutableHashArray(tosca.Hash{1}), NewImmutableHashArray(tosca.Hash{1}), true,
		},
		"diff": {
			NewImmutableHashArray(tosca.Hash{1}), NewImmutableHashArray(tosca.Hash{2}), false,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			if test.hashes1.Equal(test.hashes2) != test.equal {
				t.Errorf("unexpected equality: %v vs %v", test.hashes1, test.hashes2)
			}
		})
	}
}

func TestImmutableHashArray_AssignmentProducesEqualValue(t *testing.T) {
	b1 := NewImmutableHashArray(tosca.Hash{1})
	b2 := b1

	if !b1.Equal(b2) {
		t.Errorf("assigned value is not equal: %v vs %v", b1, b2)
	}
}

func TestImmutableHashArray_CanBeJsonEncoded(t *testing.T) {
	const (
		hash0 = "[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0]"
		hash1 = "[1,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0]"
	)

	zeroHash := "[" + strings.Repeat(hash0+",", 255)
	zeroHash += hash0 + "]"
	oneHash := "[" + hash1 + ","
	oneHash += strings.Repeat(hash0+",", 254)
	oneHash += hash0 + "]"

	tests := map[string]struct {
		hashes  ImmutableHashArray
		encoded string
	}{
		"empty": {
			NewImmutableHashArray(tosca.Hash{}), zeroHash,
		},
		"single": {
			NewImmutableHashArray(tosca.Hash{1}), oneHash,
		},
		"nil": {
			ImmutableHashArray{}, "null",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			encoded, err := json.Marshal(test.hashes)
			if err != nil {
				t.Fatalf("failed to encode into JSON: %v", err)
			}
			encodedString := string(encoded)
			if want, got := test.encoded, encodedString; want != got {
				t.Errorf("unexpected JSON encoding, wanted %v, got %v", want, got)
			}

			var restored ImmutableHashArray
			if err := json.Unmarshal(encoded, &restored); err != nil {
				t.Fatalf("failed to restore ImmutableHashArray: %v", err)
			}
			if !test.hashes.Equal(restored) {
				t.Errorf("unexpected restored value, wanted %v, got %v", test.hashes, restored)
			}
		})
	}

	t.Run("short-encoded", func(t *testing.T) {
		encoded, err := json.Marshal(zeroHash[:len(zeroHash)-1])
		if err != nil {
			t.Fatalf("failed to encode into JSON: %v", err)
		}
		var restored ImmutableHashArray
		if err := json.Unmarshal(encoded, &restored); err == nil {
			t.Fatalf("an error should have been produced but instead got nil")
		}
	})
}

func TestImmutableHashArray_Get(t *testing.T) {

	tests := map[string]struct {
		hashes ImmutableHashArray
	}{
		"default-initialized": {ImmutableHashArray{}},
		"constructed":         {NewImmutableHashArray()},
	}

	zeroHash := tosca.Hash{}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {

			hash := test.hashes.Get(0)
			if hash != zeroHash {
				t.Errorf("unexpected hash: %v", hash)
			}
			defer func() {
				if r := recover(); r == nil {
					t.Errorf("expected panic")
				}
			}()
			_ = test.hashes.Get(256)
		})
	}
}

func TestINmmutableHashArray_NewRandomImmutableHashArray(t *testing.T) {
	rnd := rand.New()

	hashes := []ImmutableHashArray{}
	for i := 0; i < 10; i++ {
		hashes = append(hashes, NewRandomImmutableHashArray(rnd))
	}
	for i := 0; i < 10; i++ {
		for j := 0; j < i; j++ {
			if hashes[i].Equal(hashes[j]) {
				t.Errorf("random hashes are not random, got %v and %v", hashes[i], hashes[j])
			}
		}
	}
}

func TestImmutableHashArray_String(t *testing.T) {
	hashes := NewImmutableHashArray(tosca.Hash{1})

	hash0 := strings.Repeat("0", 64) + " "
	hash1 := "&[01" + strings.Repeat("0", 62) + " "

	want := hash1 + strings.Repeat(hash0, 255)[:255*65-1] + "]"

	if got := hashes.String(); strings.Compare(want, got) != 0 {
		t.Errorf("unexpected string, wanted %v, got %v", want, got)
	}
}
