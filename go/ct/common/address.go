//
// Copyright (c) 2024 Fantom Foundation
//
// Use of this software is governed by the Business Source License included
// in the LICENSE.TXT file and at fantom.foundation/bsl11.
//
// Change Date: 2028-4-16
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by the GNU Lesser General Public Licence v3
//

package common

import (
	"github.com/Fantom-foundation/Tosca/go/vm"
	"pgregory.net/rand"
)

func NewAddress(in U256) vm.Address {
	return in.internal.Bytes20()
}

func NewAddressFromInt(in uint64) vm.Address {
	return NewAddress(NewU256(in))
}

func AddressToU256(a vm.Address) U256 {
	return NewU256FromBytes(a[:]...)
}

// Deprecated: use RandomAddress instead
func RandAddress(rnd *rand.Rand) (vm.Address, error) {
	return RandomAddress(rnd), nil
}

func RandomAddress(rnd *rand.Rand) vm.Address {
	address := vm.Address{}
	rnd.Read(address[:]) // never returns an error
	return address
}
