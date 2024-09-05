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
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/Fantom-foundation/Tosca/go/tosca"
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

func TestHashCache_UsesProvidedHashingFunction(t *testing.T) {
	hash := func(i int) tosca.Hash {
		return tosca.Hash{byte(i)}
	}

	cache := newHashCache(10, hash)
	for i := 0; i < 100; i++ {
		want := hash(i)
		got := cache.getHash(i)
		if want != got {
			t.Errorf("expected hash to be %x, but got %x", want, got)
		}
	}
}

func TestHashCache_HashesAreCached(t *testing.T) {
	// To test whether results are cached, we use a hash function that
	// returns different results at each call.
	counter := 0
	hash := func(int) tosca.Hash {
		counter++
		return tosca.Hash{byte(counter)}
	}

	cache := newHashCache(10, hash)

	hash1 := cache.getHash(1)
	hash2 := cache.getHash(2)

	if hash1 == hash2 {
		t.Fatalf("expected different hashes for different inputs")
	}

	if want, got := hash1, cache.getHash(1); want != got {
		t.Errorf("expected hash to be %v, but got %v", want, got)
	}
	if want, got := hash2, cache.getHash(2); want != got {
		t.Errorf("expected hash to be %v, but got %v", want, got)
	}
}

func TestHashCache_CapacityIsIncreasedToAtLeast2(t *testing.T) {
	capacity := []int{-1000, -1, 0, 1, 2, 3, 1000}
	for _, c := range capacity {
		cache := newHashCache(c, func(int) tosca.Hash {
			return tosca.Hash{}
		})
		want := c
		if want < 2 {
			want = 2
		}
		if got := len(cache.entries); got != want {
			t.Errorf("expected cache to have %d entries, but got %d", want, got)
		}
	}
}

func TestHashCache_RespectsCapacity(t *testing.T) {
	hash := func(int) tosca.Hash {
		return tosca.Hash{}
	}
	for _, capacity := range []int{2, 10, 42, 123} {
		t.Run(fmt.Sprintf("capacity=%d", capacity), func(t *testing.T) {
			cache := newHashCache(capacity, hash)
			for i := 0; i < 2*capacity; i++ {
				cache.getHash(i)

				if got := len(cache.entries); got != capacity {
					t.Errorf("expected cache to have %d entries, but got %d", capacity, got)
				}

				want := i + 1
				if want > capacity {
					want = capacity
				}
				if got := len(cache.index); got != want {
					t.Errorf("expected cache to have %d entries in the index, but got %d", want, got)
				}
			}
		})
	}
}

func TestHashCache_UsesLruReplacementOrder(t *testing.T) {
	sequence := []struct {
		touchedKey int
		lruOrder   []int
	}{
		{0, []int{0}}, // < not really needed, just here for clarity
		// New keys are added to the front, and the last key is removed when the
		// capacity is reached.
		{1, []int{1, 0}},
		{2, []int{2, 1, 0}},
		{3, []int{3, 2, 1}},
		{4, []int{4, 3, 2}},
		// Accessing an existing key moves it to the front.
		{2, []int{2, 4, 3}}, // < moves last element to the front
		{4, []int{4, 2, 3}}, // < moves an element in the middle to the front
		{4, []int{4, 2, 3}}, // < touches the front element
	}

	cache := newHashCache(3, func(int) tosca.Hash {
		return tosca.Hash{}
	})

	// The cache is initiated with the zero value.
	got := checkIndexAndGetLruOrder(t, cache)
	if want := []int{0}; !slices.Equal(want, got) {
		t.Errorf("expected initial order to be %v, but got %v", want, got)
	}

	for i, step := range sequence {
		cache.getHash(step.touchedKey)
		got := checkIndexAndGetLruOrder(t, cache)
		if want := step.lruOrder; !slices.Equal(want, got) {
			t.Errorf(
				"after step %d expected order to be %v, but got %v",
				i, want, got,
			)
		}
	}
}

func TestHashCache_ReusingKeysBeforeReachingTheCapacityLimitDoesNotLeadToDuplicates(t *testing.T) {
	sequence := []struct {
		touchedKey int
		lruOrder   []int
	}{
		{1, []int{1, 0}},
		{0, []int{0, 1}},
		{1, []int{1, 0}},
		{0, []int{0, 1}},
		{1, []int{1, 0}},
	}

	cache := newHashCache(3, func(int) tosca.Hash {
		return tosca.Hash{}
	})

	for i, step := range sequence {
		cache.getHash(step.touchedKey)
		got := checkIndexAndGetLruOrder(t, cache)
		if want := step.lruOrder; !slices.Equal(want, got) {
			t.Errorf(
				"after step %d expected order to be %v, but got %v",
				i, want, got,
			)
		}
	}
}

func checkIndexAndGetLruOrder(t *testing.T, h *hashCache[int]) []int {
	t.Helper()

	getForwardOrder := func(h *hashCache[int]) []int {
		var res []int
		for e := h.head; e != nil; e = e.succ {
			res = append(res, e.key)
		}
		return res
	}

	getBackwardOrder := func(h *hashCache[int]) []int {
		var res []int
		for e := h.tail; e != nil; e = e.pred {
			res = append(res, e.key)
		}
		slices.Reverse(res)
		return res
	}

	forward := getForwardOrder(h)
	backward := getBackwardOrder(h)

	// Check that the double-linked list is consistent.
	if !slices.Equal(forward, backward) {
		t.Errorf("expected forward and backward order to be identical but got %v and %v", forward, backward)
	}

	// Check that there are no duplicates in the keys.
	seen := make(map[int]struct{})
	for _, k := range forward {
		if _, found := seen[k]; found {
			t.Errorf("expected key %d to be unique, but it is duplicated", k)
		}
		seen[k] = struct{}{}
	}

	// Check that the index has the same size as the keys.
	if want, got := len(forward), len(h.index); want != got {
		t.Errorf("expected index to have %d entries, but got %d", want, got)
	}

	// Check that the keys are in the index.
	for _, k := range forward {
		if _, found := h.index[k]; !found {
			t.Errorf("expected key %d to be in the index, but it is not", k)
		}
	}
	return forward
}

func TestHashCache_AccessesAreThreadSafe(t *testing.T) {
	// This test is designed to detect race conditions in cases in combination
	// with Go's data race detection. It should be run with the -race flag.
	cache := newHashCache(10, func(int) tosca.Hash {
		return tosca.Hash{}
	})

	const (
		threads  = 10
		accesses = 1000
	)

	var wg sync.WaitGroup
	wg.Add(threads)
	for i := 0; i < threads; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < accesses; j++ {
				cache.getHash(j % 10)
			}
		}()
	}
	wg.Wait()

	// Check that the cache is consistent.
	checkIndexAndGetLruOrder(t, cache)
}

func TestHashCache_ConcurrentThreadsCanNotIntroduceDuplicates(t *testing.T) {
	const key = 12
	var barrier sync.WaitGroup
	barrier.Add(2)
	cache := newHashCache(10, func(i int) tosca.Hash {
		if i != key {
			return tosca.Hash{}
		}
		// Wait for two go-routines to compute the hash at the same time.
		barrier.Done()
		barrier.Wait()
		return tosca.Hash{}
	})

	var wg sync.WaitGroup
	wg.Add(2)
	for i := 0; i < 2; i++ {
		go func() {
			defer wg.Done()
			cache.getHash(key)
		}()
	}
	wg.Wait()

	// Check that the cache is consistent.
	checkIndexAndGetLruOrder(t, cache)
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

func BenchmarkSha3HashCache_Hits(b *testing.B) {
	for _, size := range []int{16, 32, 64, 128} {
		b.Run(fmt.Sprintf("size=%d", size), func(b *testing.B) {
			benchmarkSha3HashCache(b, size, false)
		})
	}
}

func BenchmarkSha3HashCache_Miss(b *testing.B) {
	for _, size := range []int{16, 32, 64, 128} {
		b.Run(fmt.Sprintf("size=%d", size), func(b *testing.B) {
			benchmarkSha3HashCache(b, size, true)
		})
	}
}
