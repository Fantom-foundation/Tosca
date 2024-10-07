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

func GetFfiOverheadExample() Example {
	code := []byte{
		byte(vm.PUSH1), 255,
		byte(vm.PUSH1), 0,
		byte(vm.MSTORE),
		byte(vm.PUSH1), 32,
		byte(vm.PUSH1), 0,
		byte(vm.RETURN),
	}

	return exampleSpec{
		Name:      "ffi",
		Code:      code,
		reference: FfiOverheadRef,
	}.build()
}

func FfiOverheadRef(x int) int {
	return int(255)
}
