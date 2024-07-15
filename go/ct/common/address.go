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
	"github.com/Fantom-foundation/Tosca/go/tosca"
	"pgregory.net/rand"
)

func NewAddress(in U256) tosca.Address {
	return in.Bytes20be()
}

func NewAddressFromInt(in uint64) tosca.Address {
	return NewAddress(NewU256(in))
}

func AddressToU256(a tosca.Address) U256 {
	return NewU256FromBytes(a[:]...)
}

func RandomAddress(rnd *rand.Rand) tosca.Address {
	address := tosca.Address{}
	_, _ = rnd.Read(address[:]) // rnd.Read never returns an error
	return address
}
