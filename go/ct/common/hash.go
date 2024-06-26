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
	"github.com/Fantom-foundation/Tosca/go/vm"
	"pgregory.net/rand"
)

func GetRandomHash(rnd *rand.Rand) vm.Hash {
	var res vm.Hash
	rnd.Read(res[:])
	return res
}
