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

import (
	"fmt"
	"math/rand"
	"testing"
	"time"
)

func TestSha3HashCache_hash_ProducesCorrectHashesForInputs(t *testing.T) {
	inputs := [][]byte{
		{},
		{0},
		{1, 2, 3, 4, 5},
		make([]byte, 32),
		make([]byte, 64),
		make([]byte, 123),
	}

	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	for i := 0; i < 100; i++ {
		input := make([]byte, r.Intn(150))
		r.Read(input)
		inputs = append(inputs, input)
	}

	cache := newSha3HashCache(10, 10)
	for _, input := range inputs {
		want := Keccak256(input)
		got := cache.hash(input)
		if want != got {
			t.Errorf("expected hash to be %x, but got %x", want, got)
		}
	}
}

func benchmarkSha3HashCache(b *testing.B, inputSize int, mutateInput bool) {
	input := make([]byte, inputSize)
	cache := newSha3HashCache(128, 128)
	for i := 0; i < b.N; i++ {
		if mutateInput {
			input[0] = byte(i)
		}
		cache.hash(input)
	}
}

func BenchmarkHashCache_Hits(b *testing.B) {
	for _, size := range []int{16, 32, 64, 128} {
		b.Run(fmt.Sprintf("size=%d", size), func(b *testing.B) {
			benchmarkSha3HashCache(b, size, false)
		})
	}
}

func BenchmarkHashCache_Miss(b *testing.B) {
	for _, size := range []int{16, 32, 64, 128} {
		b.Run(fmt.Sprintf("size=%d", size), func(b *testing.B) {
			benchmarkSha3HashCache(b, size, true)
		})
	}
}
