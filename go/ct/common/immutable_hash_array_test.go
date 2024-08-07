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
