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
	"testing"

	"github.com/Fantom-foundation/Tosca/go/tosca"
	"pgregory.net/rand"
)

func TestAddress_NewAddressFrom(t *testing.T) {
	addressU256 := NewAddress(NewU256(42))
	addressInt := NewAddressFromInt(42)

	if addressU256 != addressInt {
		t.Errorf("Address from U256 and int should be the same: %v vs %v", addressU256, addressInt)
	}

	if addressU256.String() != "0x000000000000000000000000000000000000002a" {
		t.Errorf("Address from U256 has the wrong value")
	}

	if addressInt.String() != "0x000000000000000000000000000000000000002a" {
		t.Errorf("Address from int has the wrong value")
	}
}

func TestAddress_ToU256(t *testing.T) {
	want := NewU256(42)
	address := NewAddress(want)

	if got := AddressToU256(address); want != got {
		t.Errorf("Conversion from U256 is broken: got %v, want %v", got, want)
	}
}

func TestAddress_RandomAddress(t *testing.T) {
	address1 := tosca.Address{}
	rnd := rand.New(0)
	address2 := RandomAddress(rnd)

	if address1 == address2 {
		t.Errorf("Random Address is same as default value")
	}
}
