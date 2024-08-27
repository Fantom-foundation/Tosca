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

func Keccak256ForKey(key tosca.Key) tosca.Hash {
	return keccak256_C_Key(key)
}

var keccakHasherPool = sync.Pool{New: func() any { return sha3.NewLegacyKeccak256() }}

func keccak256_Go(data []byte) tosca.Hash {
	hasher := keccakHasherPool.Get().(keccakHasher)
	hasher.Reset()
	hasher.Write(data)
	var res tosca.Hash
	hasher.Read(res[:])
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

func keccak256_C_Key(key tosca.Key) tosca.Hash {
	// The address is passed as 4x 64-bit integer values through the stack to
	// avoid the need of allocating heap memory for the key.
	return tosca.Hash(C.tosca_lfvm_keccak256_32byte(
		C.uint64_t(
			uint64(key[7])<<56|uint64(key[6])<<48|uint64(key[5])<<40|uint64(key[4])<<32|
				uint64(key[3])<<24|uint64(key[2])<<16|uint64(key[1])<<8|uint64(key[0])<<0),
		C.uint64_t(
			uint64(key[15])<<56|uint64(key[14])<<48|uint64(key[13])<<40|uint64(key[12])<<32|
				uint64(key[11])<<24|uint64(key[10])<<16|uint64(key[9])<<8|uint64(key[8])<<0),
		C.uint64_t(
			uint64(key[23])<<56|uint64(key[22])<<48|uint64(key[21])<<40|uint64(key[20])<<32|
				uint64(key[19])<<24|uint64(key[18])<<16|uint64(key[17])<<8|uint64(key[16])<<0),
		C.uint64_t(
			uint64(key[31])<<56|uint64(key[30])<<48|uint64(key[29])<<40|uint64(key[28])<<32|
				uint64(key[27])<<24|uint64(key[26])<<16|uint64(key[25])<<8|uint64(key[24])<<0),
	))
}
