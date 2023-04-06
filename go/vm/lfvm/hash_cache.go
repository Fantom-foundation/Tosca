package lfvm

import (
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"golang.org/x/crypto/sha3"
)

// hashCacheEntry32 is an entry of a cache for hashes of 32-byte long inputs.
type hashCacheEntry32 struct {
	// key is the input value cache entries are indexed by.
	key [32]byte
	// hash is the cached (Sha3) hash of the key.
	hash common.Hash
	// pred/succ pointers are used for a double linked list for the LRU order.
	pred, succ *hashCacheEntry32
}

// hashCacheEntry64 is an entry of a cache for hashes of 64-byte long inputs.
type hashCacheEntry64 struct {
	// key is the input value cache entries are indexed by.
	key [64]byte
	// hash is the cached (Sha3) hash of the key.
	hash common.Hash
	// pred/succ pointers are used for a double linked list for the LRU order.
	pred, succ *hashCacheEntry64
}

// HashCache is an LRU governed fixed-capacity cache for SHA3 hashes.
// The cache maintains hashes for hashed input data of size 32 and 64,
// which are the vast majority of values hashed when running EVM
// instructions.
type HashCache struct {
	// Hash infrastructure for 32-byte long inputs.
	entries32      []hashCacheEntry32
	index32        map[[32]byte]*hashCacheEntry32
	head32, tail32 *hashCacheEntry32
	nextFree32     int
	lock32         sync.Mutex

	// Hash infrastructure for 64-byte long inputs.
	entries64      []hashCacheEntry64
	index64        map[[64]byte]*hashCacheEntry64
	head64, tail64 *hashCacheEntry64
	nextFree64     int
	lock64         sync.Mutex

	// Statistics.
	hit, miss int
}

// newHashCache creates a HashCache with the given capacity of entries.
func newHashCache(capacity32 int, capacity64 int) *HashCache {
	res := &HashCache{
		entries32: make([]hashCacheEntry32, capacity32),
		index32:   make(map[[32]byte]*hashCacheEntry32, capacity32),
		entries64: make([]hashCacheEntry64, capacity64),
		index64:   make(map[[64]byte]*hashCacheEntry64, capacity64),
	}

	// To avoid the need for handling the special case of an empty cache
	// in every lookup operation we initialize the cache with one value.
	// Since values are never removed, just evicted to make space for another,
	// the cache will never be empty.
	hasher := sha3.NewLegacyKeccak256().(keccakState)

	// Insert first 32-byte element (all zeros).
	res.head32 = res.getFree32()
	res.tail32 = res.head32

	hasher.Reset()
	var data32 [32]byte
	hasher.Write(data32[:])
	var hash32 common.Hash
	hasher.Read(hash32[:])
	res.head32.hash = hash32

	res.index32[data32] = res.head32

	// Insert first 64-byte element (all zeros).
	res.head64 = res.getFree64()
	res.tail64 = res.head64

	hasher.Reset()
	var data64 [64]byte
	hasher.Write(data64[:])
	var hash64 common.Hash
	hasher.Read(hash64[:])
	res.head64.hash = hash64

	res.index64[data64] = res.head64

	return res
}

// hash fetches a cached hash or computes the hash for the provided data
// using the hasher in the given context.
func (h *HashCache) hash(c *context, data []byte) common.Hash {
	if len(data) == 32 {
		return h.getHash32(c, data)
	}
	if len(data) == 64 {
		return h.getHash64(c, data)
	}
	h.miss++
	return getHash(c, data)
}

func (h *HashCache) getHash32(c *context, data []byte) common.Hash {
	var key [32]byte
	copy(key[:], data)
	h.lock32.Lock()
	if entry, found := h.index32[key]; found {
		h.hit++
		// Move entry to the front.
		if entry != h.head32 {
			// Remove from current place.
			entry.pred.succ = entry.succ
			if entry.succ != nil {
				entry.succ.pred = entry.pred
			} else {
				h.tail32 = entry.pred
			}
			// Add to front
			entry.pred = nil
			entry.succ = h.head32
			h.head32.pred = entry
			h.head32 = entry
		}
		h.lock32.Unlock()
		return entry.hash
	}
	h.miss++

	// Compute the hash without holding the lock.
	h.lock32.Unlock()
	hash := getHash(c, data)
	h.lock32.Lock()
	defer h.lock32.Unlock()

	// We need to check that the key has not be added concurrently.
	if _, found := h.index32[key]; found {
		// If it was added concurrently, we are done.
		return hash
	}

	// The key is still not present, so we add it.
	entry := h.getFree32()
	entry.key = key
	entry.hash = hash
	entry.pred = nil
	entry.succ = h.head32
	h.head32.pred = entry
	h.head32 = entry
	h.index32[key] = entry
	return entry.hash
}

func (h *HashCache) getHash64(c *context, data []byte) common.Hash {
	var key [64]byte
	copy(key[:], data)
	h.lock64.Lock()
	if entry, found := h.index64[key]; found {
		h.hit++
		// Move entry to the front.
		if entry != h.head64 {
			// Remove from current place.
			entry.pred.succ = entry.succ
			if entry.succ != nil {
				entry.succ.pred = entry.pred
			} else {
				h.tail64 = entry.pred
			}
			// Add to front
			entry.pred = nil
			entry.succ = h.head64
			h.head64.pred = entry
			h.head64 = entry
		}
		h.lock64.Unlock()
		return entry.hash
	}
	h.miss++

	// Compute the hash without holding the lock.
	h.lock64.Unlock()
	hash := getHash(c, data)
	h.lock64.Lock()
	defer h.lock64.Unlock()

	// We need to check that the key has not be added concurrently.
	if _, found := h.index64[key]; found {
		// If it was added concurrently, we are done.
		return hash
	}

	// The key is still not present, so we add it.
	entry := h.getFree64()
	entry.key = key
	entry.hash = hash
	entry.pred = nil
	entry.succ = h.head64
	h.head64.pred = entry
	h.head64 = entry
	h.index64[key] = entry
	return entry.hash
}

func (h *HashCache) getFree32() *hashCacheEntry32 {
	// If there are still free entries, use on of those.
	if h.nextFree32 < len(h.entries32) {
		res := &h.entries32[h.nextFree32]
		h.nextFree32++
		return res
	}
	// Use the tail.
	res := h.tail32
	h.tail32 = h.tail32.pred
	h.tail32.succ = nil
	delete(h.index32, res.key)
	return res
}

func (h *HashCache) getFree64() *hashCacheEntry64 {
	// If there are still free entries, use on of those.
	if h.nextFree64 < len(h.entries64) {
		res := &h.entries64[h.nextFree64]
		h.nextFree64++
		return res
	}
	// Use the tail.
	res := h.tail64
	h.tail64 = h.tail64.pred
	h.tail64.succ = nil
	delete(h.index64, res.key)
	return res
}

// getHash computes a Sha3 hash of the given data using the hasher
// instance in the provided context.
func getHash(c *context, data []byte) common.Hash {
	res := common.Hash{}

	if c.hasher == nil {
		c.hasher = sha3.NewLegacyKeccak256().(keccakState)
	} else {
		c.hasher.Reset()
	}

	c.hasher.Write(data)
	c.hasher.Read(res[:])
	return res
}
