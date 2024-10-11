// Copyright (c) 2024 Fantom Foundation
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at fantom.foundation/bsl11.
//
// Change Date: 2028-4-16
//
// On the date above, in accordance with the Business Source License, use of
// this software will be governed by the GNU Lesser General Public License v3.

package examples

import (
	"github.com/ethereum/go-ethereum/core/vm"
)

// This example tries to represent the worst case for very short contracts.
// In particular, it tries to trigger all allocations that happen when using an
// evmc interpreter, which, depending on how it is implemented, may need to copy
// data into new allocations.
// - code being not empty may cause jump analysis to allocate memory
// - opcode calldatacopy causes memory expansion
// - opcode return causes output not to be empty
func GetStaticOverheadExample() Example {
	code := []byte{
		byte(vm.PUSH1), 4, // push size 4
		byte(vm.PUSH1), 32, // push offset 32
		byte(vm.PUSH1), 28, // push destOffset 28
		byte(vm.CALLDATACOPY), // copy 4 bytes at offset 32 from call data into memory at offset 28
		byte(vm.PUSH1), 32,    // push len 32
		byte(vm.PUSH1), 0, // push offset 0
		byte(vm.RETURN), // return 32 bytes at offset 0
	}

	return exampleSpec{
		Name:      "static_overhead",
		Code:      code,
		reference: StaticOverheadRef,
	}.build()
}

func StaticOverheadRef(x int) int {
	return int(x)
}
