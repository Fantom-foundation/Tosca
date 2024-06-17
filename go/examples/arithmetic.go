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
	"math"

	"github.com/holiman/uint256"
)

func GetArithmeticExample() Example {
	/* Solidity code for the arithmetic example:
	// SPDX-License-Identifier: MIT
	pragma solidity ^0.8;

	contract Arithmetic {
		function arithmetic(int n) public pure returns (int) {
			unchecked {
				uint result = 0;
				for(uint i = 1; i <= uint(n); ++i) {
					result += i;
					result *= i;
					result += i * i;
					result -= i;
					result /= i;
					result *= (i % 3) + 1;
					result += i * i * i;
				}
				return int(result % uint(int(type(int32).max)));
			}
		}
	}
	*/
	code, err := hex.DecodeString("608060405234801561001057600080fd5b506004361061002b5760003560e01c8063cc821c0914610030575b600080fd5b61004a60048036038101906100459190610127565b610060565b6040516100579190610163565b60405180910390f35b600080600090506000600190505b8381116100cb578082019150808202915080810282019150808203915080828161009b5761009a61017e565b5b0491506001600382816100b1576100b061017e565b5b06018202915080818202028201915080600101905061006e565b50637fffffff60030b81816100e3576100e261017e565b5b06915050919050565b600080fd5b6000819050919050565b610104816100f1565b811461010f57600080fd5b50565b600081359050610121816100fb565b92915050565b60006020828403121561013d5761013c6100ec565b5b600061014b84828501610112565b91505092915050565b61015d816100f1565b82525050565b60006020820190506101786000830184610154565b92915050565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052601260045260246000fdfea2646970667358221220475b1df27897da64202d55f39cc1333578da5d82cf48eb365fe0baf54c00e31964736f6c63430008140033")
	if err != nil {
		log.Fatalf("Unable to decode arithmetic-code: %v", err)
	}

	return exampleSpec{
		Name:      "arithmetic",
		code:      code,
		function:  0xCC821C09,
		reference: arithmetic,
	}.build()
}

func arithmetic(n int) int {
	iterations := uint256.NewInt(uint64(n))
	result := uint256.NewInt(0)
	for i := uint256.NewInt(1); i.Lt(iterations) || i.Eq(iterations); i.AddUint64(i, 1) {
		iSquared := i.Clone().Mul(i, i)
		iCubed := iSquared.Clone().Mul(iSquared, i)
		iMod3 := i.Clone().Mod(i, uint256.NewInt(3))
		result.Add(result, i)
		result.Mul(result, i)
		result.Add(result, iSquared)
		result.Sub(result, i)
		result.Div(result, i)
		result.Mul(result, iMod3.AddUint64(iMod3, 1))
		result.Add(result, iCubed)
	}
	maxInt32 := uint256.NewInt(math.MaxInt32)
	result.Mod(result, maxInt32)
	return int(result[0])
}
