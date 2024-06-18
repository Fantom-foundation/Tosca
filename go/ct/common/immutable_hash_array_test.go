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
	"testing"

	"github.com/Fantom-foundation/Tosca/go/vm"
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
		"single": {
			NewImmutableHashArray(vm.Hash{1}), NewImmutableHashArray(vm.Hash{1}), true,
		},
		"diff": {
			NewImmutableHashArray(vm.Hash{1}), NewImmutableHashArray(vm.Hash{2}), false,
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
	b1 := NewImmutableHashArray(vm.Hash{1})
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

	zeroHash := "["
	for i := 0; i < 256; i++ {
		zeroHash += hash0
		zeroHash += ","
	}
	zeroHash = zeroHash[:len(zeroHash)-1] + "]"

	oneHash := "[" + hash1 + ","
	for i := 0; i < 255; i++ {
		oneHash += hash0
		oneHash += ","
	}
	oneHash = oneHash[:len(oneHash)-1] + "]"

	tests := map[string]struct {
		hashes  ImmutableHashArray
		encoded string
	}{
		"empty": {
			NewImmutableHashArray(vm.Hash{}), zeroHash,
		},
		"single": {
			NewImmutableHashArray(vm.Hash{1}), oneHash,
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

	oneHash := vm.Hash{1}
	hashes := NewImmutableHashArray(oneHash)
	hash := hashes.Get(0)
	if hash != oneHash {
		t.Errorf("unexpected hash: %v", hash)
	}
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("expected panic")
		}
	}()
	_ = hashes.Get(256)

}
