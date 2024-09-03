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
	"sync"

	"github.com/Fantom-foundation/Tosca/go/tosca"
)

// HashCache is an LRU governed fixed-capacity cache for SHA3 hashes.
// The cache maintains hashes for hashed input data of size 32 and 64,
// which are the vast majority of values hashed when running EVM
// instructions.
type Sha3HashCache struct {
	cache32 *hashCache[[32]byte]
	cache64 *hashCache[[64]byte]
}

// newHashCache creates a HashCache with the given capacity of entries.
func newSha3HashCache(capacity32 int, capacity64 int) *Sha3HashCache {
	return &Sha3HashCache{
		cache32: newHashCache(capacity32, func(key [32]byte) tosca.Hash { return Keccak256For32byte(key) }),
		cache64: newHashCache(capacity64, func(key [64]byte) tosca.Hash { return Keccak256(key[:]) }),
	}
}

// hash fetches a cached hash or computes the hash for the provided data
// using the hasher in the given context.
func (h *Sha3HashCache) hash(data []byte) tosca.Hash {
	if len(data) == 32 {
		var key [32]byte
		copy(key[:], data)
		return h.cache32.getHash(key)
	}
	if len(data) == 64 {
		var key [64]byte
		copy(key[:], data)
		return h.cache64.getHash(key)
	}
	return Keccak256(data)
}

type hashCache[K comparable] struct {
	entries    []hashCacheEntry[K]
	index      map[K]*hashCacheEntry[K]
	head, tail *hashCacheEntry[K]
	nextFree   int
	lock       sync.Mutex
	hash       func(K) tosca.Hash
}

func newHashCache[K comparable](capacity int, hash func(K) tosca.Hash) *hashCache[K] {
	res := &hashCache[K]{
		entries: make([]hashCacheEntry[K], capacity),
		index:   make(map[K]*hashCacheEntry[K], capacity),
		hash:    hash,
	}

	// To avoid the need for handling the special case of an empty cache
	// in every lookup operation we initialize the cache with one value.
	// Since values are never removed, just evicted to make space for another,
	// the cache will never be empty.

	// Insert first element (zero value).
	res.head = res.getFree()
	res.tail = res.head
	var key K
	res.head.hash = hash(key)
	res.index[key] = res.head
	return res
}

func (h *hashCache[K]) getHash(key K) tosca.Hash {
	h.lock.Lock()
	if entry, found := h.index[key]; found {
		// Move entry to the front.
		if entry != h.head {
			// Remove from current place.
			entry.pred.succ = entry.succ
			if entry.succ != nil {
				entry.succ.pred = entry.pred
			} else {
				h.tail = entry.pred
			}
			// Add to front
			entry.pred = nil
			entry.succ = h.head
			h.head.pred = entry
			h.head = entry
		}
		h.lock.Unlock()
		return entry.hash
	}

	// Compute the hash without holding the lock.
	h.lock.Unlock()
	hash := h.hash(key)
	h.lock.Lock()
	defer h.lock.Unlock()

	// We need to check that the key has not be added concurrently.
	if _, found := h.index[key]; found {
		// If it was added concurrently, we are done.
		return hash
	}

	// The key is still not present, so we add it.
	entry := h.getFree()
	entry.key = key
	entry.hash = hash
	entry.pred = nil
	entry.succ = h.head
	h.head.pred = entry
	h.head = entry
	h.index[key] = entry
	return entry.hash
}

func (h *hashCache[K]) getFree() *hashCacheEntry[K] {
	// If there are still free entries, use on of those.
	if h.nextFree < len(h.entries) {
		res := &h.entries[h.nextFree]
		h.nextFree++
		return res
	}
	// Use the tail.
	res := h.tail
	h.tail = h.tail.pred
	h.tail.succ = nil
	delete(h.index, res.key)
	return res
}

// hashCacheEntry is an entry of a cache for hashes of values of type K.
type hashCacheEntry[K any] struct {
	// key is the input value cache entries are indexed by.
	key K
	// hash is the cached (Sha3) hash of the key.
	hash tosca.Hash
	// pred/succ pointers are used for a double linked list for the LRU order.
	pred, succ *hashCacheEntry[K]
}
