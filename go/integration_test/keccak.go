// Copyright (c) 2024 Fantom Foundation
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at fantom.foundation/bsl11.
//
// Change Date: 2028-4-16
//
// On the date above, in accordance with the Business Source License, use of
// this software will be governed by the GNU Lesser General Public License v3.

package integration_test

import (
	"github.com/Fantom-foundation/Tosca/go/tosca"
	"golang.org/x/crypto/sha3"
)

func Keccak256Hash(data []byte) tosca.Hash {
	hasher := sha3.NewLegacyKeccak256()
	hasher.Write(data)
	var hash tosca.Hash
	hasher.Sum(hash[0:0])
	return hash
}
