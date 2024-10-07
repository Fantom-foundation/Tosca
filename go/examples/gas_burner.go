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
	"encoding/hex"
	"log"
)

// GetGasBurnerExample provides an example code for tests and benchmarks that
// runs a loop burning gas. The example is derived from the following solidity
// code:
//
//	 function burn(uint32 x) public view returns(uint32) {
//		uint256 initialGas = gasleft();
//		uint256 wantGas = initialGas - x;
//		while (gasleft() > wantGas) {}
//		return x;
//	 }
//
// The idea for this example is derived from a feature request enabling
// contracts to explicitly consume gas in a controled way.
func GetGasBurnerExample() Example {
	// An implementation of the gas-burner function in EVM byte code.
	code, err := hex.DecodeString("608060405234801561001057600080fd5b506004361061002b5760003560e01c80637a5984c414610030575b600080fd5b61004a600480360381019061004591906100cf565b610060565b604051610057919061010b565b60405180910390f35b6000805a905060008363ffffffff168261007a919061015f565b90505b805a1161007d578392505050919050565b600080fd5b600063ffffffff82169050919050565b6100ac81610093565b81146100b757600080fd5b50565b6000813590506100c9816100a3565b92915050565b6000602082840312156100e5576100e461008e565b5b60006100f3848285016100ba565b91505092915050565b61010581610093565b82525050565b600060208201905061012060008301846100fc565b92915050565b6000819050919050565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052601160045260246000fd5b600061016a82610126565b915061017583610126565b925082820390508181111561018d5761018c610130565b5b9291505056fea2646970667358221220545ed7c000c64c800b0c49c868c9db66915a43a262d774e8f0b1e4b44e7488fe64736f6c637828302e382e32352d646576656c6f702e323032342e322e32342b636f6d6d69742e64626137353465630059")
	if err != nil {
		log.Fatalf("Unable to decode gas-burner-code: %v", err)
	}

	return exampleSpec{
		Name:      "gas_burner",
		Code:      code,
		function:  0x7a5984c4, // function selector for the burn function
		reference: burnGas,
	}.build()
}

func burnGas(x int) int {
	return x
}
