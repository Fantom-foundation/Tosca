// Copyright (c) 2024 Fantom Foundation
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at fantom.foundation/bsl11.
//
// Change Date: 2028-4-16
//
// On the date above, in accordance with the Business Source License, use of
// this software will be governed by the GNU Lesser General Public License v3.

package lfvm

/*
#include "keccak.h"
*/
import "C"

import (
	"sync"
	"unsafe"

	"github.com/Fantom-foundation/Tosca/go/tosca"
	"golang.org/x/crypto/sha3"
)

func Keccak256(data []byte) tosca.Hash {
	return keccak256_C(data)
}

func Keccak256For32byte(data [32]byte) tosca.Hash {
	return keccak256_C_32byte(data)
}

var keccakHasherPool = sync.Pool{New: func() any { return sha3.NewLegacyKeccak256() }}

func keccak256_Go(data []byte) tosca.Hash {
	hasher := keccakHasherPool.Get().(keccakHasher)
	hasher.Reset()
	_, _ = hasher.Write(data) // keccak256 never returns an error
	var res tosca.Hash
	_, _ = hasher.Read(res[:]) // keccak256 never returns an error
	keccakHasherPool.Put(hasher)
	return res
}

type keccakHasher interface {
	Reset()
	Write(in []byte) (int, error)
	Read(out []byte) (int, error)
}

var emptyKeccak256Hash = keccak256_Go([]byte{})

func keccak256_C(data []byte) tosca.Hash {
	if len(data) == 0 {
		return emptyKeccak256Hash
	}
	res := C.tosca_lfvm_keccak256(unsafe.Pointer(&data[0]), C.size_t(len(data)))
	return tosca.Hash(res)
}

func keccak256_C_32byte(data [32]byte) tosca.Hash {
	// The address is passed as 4x 64-bit integer values through the stack to
	// avoid the need of allocating heap memory for the key.
	return tosca.Hash(C.tosca_lfvm_keccak256_32byte(
		C.uint64_t(
			uint64(data[7])<<56|uint64(data[6])<<48|uint64(data[5])<<40|uint64(data[4])<<32|
				uint64(data[3])<<24|uint64(data[2])<<16|uint64(data[1])<<8|uint64(data[0])<<0),
		C.uint64_t(
			uint64(data[15])<<56|uint64(data[14])<<48|uint64(data[13])<<40|uint64(data[12])<<32|
				uint64(data[11])<<24|uint64(data[10])<<16|uint64(data[9])<<8|uint64(data[8])<<0),
		C.uint64_t(
			uint64(data[23])<<56|uint64(data[22])<<48|uint64(data[21])<<40|uint64(data[20])<<32|
				uint64(data[19])<<24|uint64(data[18])<<16|uint64(data[17])<<8|uint64(data[16])<<0),
		C.uint64_t(
			uint64(data[31])<<56|uint64(data[30])<<48|uint64(data[29])<<40|uint64(data[28])<<32|
				uint64(data[27])<<24|uint64(data[26])<<16|uint64(data[25])<<8|uint64(data[24])<<0),
	))
}
