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

package examples

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"
	"golang.org/x/crypto/sha3"
)

func GetSha3Example() Example {
	// Implement a loop computing x iterative hashes.
	code := []byte{
		// Parse the input parameter.
		byte(vm.PUSH1), 4,
		byte(vm.CALLDATALOAD),

		// Implement the loop header.
		byte(vm.JUMPDEST),
		byte(vm.DUP1),
		byte(vm.ISZERO),
		byte(vm.PUSH1), 24,
		byte(vm.JUMPI),

		// Compute one hash step.
		byte(vm.PUSH1), 32,
		byte(vm.PUSH1), 0,
		byte(vm.SHA3),
		byte(vm.PUSH1), 0,
		byte(vm.MSTORE),

		// Decrement loop iterator.
		byte(vm.PUSH1), 1,
		byte(vm.SWAP1),
		byte(vm.SUB),

		// Jump back to start of the loop.
		byte(vm.PUSH1), 3,
		byte(vm.JUMP),

		byte(vm.JUMPDEST),

		// Mask out everything but the last byte.
		byte(vm.PUSH1), 0,
		byte(vm.MLOAD),
		byte(vm.PUSH1), 255,
		byte(vm.AND),
		byte(vm.PUSH1), 0,
		byte(vm.MSTORE),

		// Return the result.
		byte(vm.PUSH1), 32,
		byte(vm.PUSH1), 0,
		byte(vm.RETURN),
	}

	return exampleSpec{
		Name:      "sha3",
		code:      code,
		reference: sha3Ref,
	}.build()
}

func sha3Ref(x int) int {
	var hash common.Hash
	hasher := sha3.NewLegacyKeccak256()
	for i := 0; i < x; i++ {
		hasher.Reset()
		hasher.Write(hash[:])
		hasher.Sum(hash[0:0])
	}
	return int(hash[31])
}
