package examples

import (
	"encoding/hex"
	"log"
)

func GetMemoryExample() Example {
	/* Solidity code for the memory example:
	// SPDX-License-Identifier: MIT
	pragma solidity ^0.8;

	contract Memory {
		function mem(int n) public pure returns (int) {
			unchecked {
				uint size = uint(n);
				uint[] memory values = new uint[](size);
				for(uint i = 0; i < size; ++i) {
					values[i] = i;
				}
				uint[] memory values_copy = new uint[](size);
				for(uint i = 0; i < size; ++i) {
					values_copy[i] = values[i];
				}
				return n > 0 ? int(values_copy[size - 1]) : n;
			}
		}
	}
	*/
	code, err := hex.DecodeString("608060405234801561001057600080fd5b506004361061002b5760003560e01c8063e88ae78114610030575b600080fd5b61004a600480360381019061004591906101fa565b610060565b6040516100579190610236565b60405180910390f35b60008082905060008167ffffffffffffffff81111561008257610081610251565b5b6040519080825280602002602001820160405280156100b05781602001602082028036833780820191505090505b50905060005b828110156100e957808282815181106100d2576100d1610280565b5b6020026020010181815250508060010190506100b6565b5060008267ffffffffffffffff81111561010657610105610251565b5b6040519080825280602002602001820160405280156101345781602001602082028036833780820191505090505b50905060005b838110156101875782818151811061015557610154610280565b5b60200260200101518282815181106101705761016f610280565b5b60200260200101818152505080600101905061013a565b506000851361019657846101b5565b8060018403815181106101ac576101ab610280565b5b60200260200101515b9350505050919050565b600080fd5b6000819050919050565b6101d7816101c4565b81146101e257600080fd5b50565b6000813590506101f4816101ce565b92915050565b6000602082840312156102105761020f6101bf565b5b600061021e848285016101e5565b91505092915050565b610230816101c4565b82525050565b600060208201905061024b6000830184610227565b92915050565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052604160045260246000fd5b7f4e487b7100000000000000000000000000000000000000000000000000000000600052603260045260246000fdfea26469706673582212203716c9113aa44253d08def843c557fb75d452feb1691eebd862b74f93b98986864736f6c63430008140033")
	if err != nil {
		log.Fatalf("Unable to decode memory-code: %v", err)
	}

	return exampleSpec{
		Name:      "memory",
		code:      code,
		function:  0xE88AE781,
		reference: memory,
	}.build()
}

func memory(n int) int {
	values := make([]int, n)
	for i := 0; i < n; i++ {
		values[i] = i
	}
	valuesCopy := make([]int, n)
	for i := 0; i < n; i++ {
		valuesCopy[i] = values[i]
	}
	if n > 0 {
		return valuesCopy[n-1]
	}
	return n
}
