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
	"fmt"

	"github.com/Fantom-foundation/Tosca/go/vm"
	"pgregory.net/rand"
)

// ImmutableHashArray is an immutable array of 256 vm.Hash that can be trivially cloned.
// if default initialized, it is considered as if all hashes are zero.
type ImmutableHashArray struct {
	data *[256]vm.Hash
}

func NewImmutableHashArray(hashes ...vm.Hash) ImmutableHashArray {
	var data [256]vm.Hash
	copy(data[:], hashes)
	return ImmutableHashArray{data: &data}
}

func (b ImmutableHashArray) Equal(other ImmutableHashArray) bool {
	if b.data == nil && other.data == nil {
		return true
	}
	if b.data == nil {
		return *other.data == [256]vm.Hash{}
	}
	if other.data == nil {
		return *b.data == [256]vm.Hash{}
	}
	return b.data == other.data || (b.data != nil && other.data != nil && *b.data == *other.data)
}

func NewRandomImmutableHashArray(rnd *rand.Rand) ImmutableHashArray {
	hashes := ImmutableHashArray{}
	hashes.data = new([256]vm.Hash)
	for i := 0; i < 256; i++ {
		hashes.data[i] = GetRandomHash(rnd)
	}
	return hashes
}

func (b ImmutableHashArray) String() string {
	return fmt.Sprintf("%x", b.data)
}

func (b ImmutableHashArray) MarshalJSON() ([]byte, error) {
	return json.Marshal(b.data)
}

func (h *ImmutableHashArray) UnmarshalJSON(data []byte) error {
	// Unmarshal the JSON array into a slice of vm.Hash
	var hashes [256]vm.Hash
	err := json.Unmarshal(data, &hashes)
	if err != nil {
		return err
	}

	// Check the length of the ImmutableHashArray data
	if len(hashes) != len(h.data) {
		return fmt.Errorf("invalid ImmutableHashArray length")
	}

	// Copy the slice into the ImmutableHashArray array
	if string(data) == "null" {
		h.data = nil
	} else {
		h.data = &hashes
	}
	return nil
}

// Get returns the hash at the given index or panics if out of range
func (b ImmutableHashArray) Get(index uint64) vm.Hash {
	if b.data == nil && index < 256 {
		return vm.Hash{}
	}
	// if data is nil, we still want to return ouf of range error
	if index >= 256 {
		panic(fmt.Sprintf("index out of range: %d", index))
	}
	return b.data[index]
}
