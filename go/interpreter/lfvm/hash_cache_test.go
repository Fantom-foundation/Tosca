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
	"bytes"
	"fmt"
	"reflect"
	"testing"

	"github.com/Fantom-foundation/Tosca/go/tosca"
)

func TestHashCache_EmptyCacheHasCapacityAndIsClean(t *testing.T) {

	cache := newHashCache(3, 3)

	if reflect.DeepEqual(cache.entries32, []hashCacheEntry32{}) {
		t.Fatalf("cache32 should have 3 entries, but got %d", len(cache.entries32))
	}
	if reflect.DeepEqual(cache.entries64, []hashCacheEntry64{}) {
		t.Fatalf("cache64 should have 3 entries, but got %d", len(cache.entries64))
	}
	if reflect.DeepEqual(cache.index32, map[[32]byte]*hashCacheEntry32{}) {
		t.Fatalf("index32 should be 1, but got %d", len(cache.index32))
	}
	if reflect.DeepEqual(cache.index64, map[[64]byte]*hashCacheEntry64{}) {
		t.Fatalf("index64 should be 1, but got %d", len(cache.index64))
	}
	if cache.nextFree32 != 1 {
		t.Fatalf("nextFree32 should be 1, but got %d", cache.nextFree32)
	}
	if cache.nextFree64 != 1 {
		t.Fatalf("nextFree64 should be 1, but got %d", cache.nextFree64)
	}
	if cache.head32.key != [32]byte{} {
		t.Fatalf("head32 should be zero value, but got %v", cache.head32)
	}
	if cache.head64.key != [64]byte{} {
		t.Fatalf("head64 should be zero value, but got %v", cache.head64)
	}
	if cache.tail32 != cache.head32 {
		t.Fatalf("tail32 should be same as head32, but got %v", cache.tail32)
	}
	if cache.tail64 != cache.head64 {
		t.Fatalf("tail64 should be nil as head32, but got %v", cache.tail64)
	}
}

func compareHashCaches(got, want *hashCache) (bool, error) {
	if !reflect.DeepEqual(got.entries32, want.entries32) {
		return false, fmt.Errorf("cache32 unequals, want %v but got %v", want.entries32, got.entries32)
	}
	if !reflect.DeepEqual(got.entries64, want.entries64) {
		return false, fmt.Errorf("cache64 unequals, want %v but got %v", want.entries64, got.entries64)
	}
	if !reflect.DeepEqual(got.index32, want.index32) {
		return false, fmt.Errorf("index32 should be 1, but got %d", len(got.index32))
	}
	if !reflect.DeepEqual(got.index64, want.index64) {
		return false, fmt.Errorf("index64 should be 1, but got %d", len(got.index64))
	}
	if got.nextFree32 != want.nextFree32 {
		return false, fmt.Errorf("nextFree32 should be 1, but got %d", got.nextFree32)
	}
	if got.nextFree64 != want.nextFree64 {
		return false, fmt.Errorf("nextFree64 should be 1, but got %d", got.nextFree64)
	}
	if got.head32.key != want.head32.key {
		return false, fmt.Errorf("head32 should be zero value, but got %v", got.head32)
	}
	if got.head64.key != want.head64.key {
		return false, fmt.Errorf("head64 should be zero value, but got %v", got.head64)
	}
	if got.tail32.key != want.tail32.key {
		return false, fmt.Errorf("tail32 should be same as head32, but got %v", got.tail32)
	}
	if got.tail64.key != want.tail64.key {
		return false, fmt.Errorf("tail64 should be nil as head32, but got %v", got.tail64)
	}
	return true, nil
}

func TestHashCache_CanAndReturnStoreHash(t *testing.T) {

	hashOf1 := tosca.Hash{0x5f, 0xe7, 0xf9, 0x77, 0xe7, 0x1d, 0xba, 0x2e, 0xa1, 0xa6,
		0x8e, 0x21, 0x05, 0x7b, 0xee, 0xbb, 0x9b, 0xe2, 0xac, 0x30, 0xc6,
		0x41, 0x0a, 0xa3, 0x8d, 0x4f, 0x3f, 0xbe, 0x41, 0xdc, 0xff, 0xd2}

	data32 := []byte{0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0a,
		0x0b, 0x0c, 0x0d, 0x0e, 0x0f, 0x10, 0x11, 0x12, 0x13, 0x14, 0x15,
		0x16, 0x17, 0x18, 0x19, 0x1a, 0x1b, 0x1c, 0x1d, 0x1e, 0x1f}

	data64 := []byte{0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0a,
		0x0b, 0x0c, 0x0d, 0x0e, 0x0f, 0x10, 0x11, 0x12, 0x13, 0x14, 0x15,
		0x16, 0x17, 0x18, 0x19, 0x1a, 0x1b, 0x1c, 0x1d, 0x1e, 0x1f, 0x20,
		0x21, 0x22, 0x23, 0x24, 0x25, 0x26, 0x27, 0x28, 0x29, 0x2a, 0x2b,
		0x2c, 0x2d, 0x2e, 0x2f, 0x30, 0x31, 0x32, 0x33, 0x34, 0x35, 0x36,
		0x37, 0x38, 0x39, 0x3a, 0x3b, 0x3c, 0x3d, 0x3e, 0x3f}

	tests := map[string]struct {
		data  []byte
		want  tosca.Hash
		cache *hashCache
	}{
		"new data": {
			[]byte{1},
			hashOf1,
			newHashCache(3, 3)},
		"32 bytes": {
			data32,
			tosca.Hash{0x8a, 0xe1, 0xaa, 0x59, 0x7f, 0xa1, 0x46, 0xeb, 0xd3, 0xaa,
				0x2c, 0xed, 0xdf, 0x36, 0x06, 0x68, 0xde, 0xa5, 0xe5, 0x26, 0x56,
				0x7e, 0x92, 0xb0, 0x32, 0x18, 0x16, 0xa4, 0xe8, 0x95, 0xbd, 0x2d},
			func() *hashCache {
				cache := newHashCache(3, 3)
				cache.getHash32(&context{stack: NewStack()}, data32)
				return cache
			}()},
		"64 bytes": {
			data64,
			tosca.Hash{0x0, 0x20, 0x30, 0xbd, 0xe3, 0xd4, 0xcf, 0x89, 0x91, 0x96,
				0x49, 0x77, 0x5c, 0xd7, 0x18, 0x75, 0xc4, 0xd0, 0xab, 0x17, 0x08,
				0xa3, 0x80, 0xe0, 0x3f, 0xef, 0xc3, 0xa2, 0x8a, 0xa2, 0x48, 0x31},
			func() *hashCache {
				cache := newHashCache(3, 3)
				cache.getHash64(&context{stack: NewStack()}, data64)
				return cache
			}()},
	}

	code := Code{Instruction{STOP, 0x0000}}

	for _, test := range tests {
		t.Run(fmt.Sprintf("hash(%v)", test.data), func(t *testing.T) {
			cache := newHashCache(3, 3)
			ctxt := &context{
				code:  code,
				stack: NewStack(),
			}
			hash := cache.hash(ctxt, test.data)
			if hash != test.want {
				t.Fatalf("hash(%v) got %x, want %x", test.data, hash, test.want)
			}
			if ok, err := compareHashCaches(cache, test.cache); !ok {
				t.Fatalf("%v", err)
			}

		})
	}
}

func TestHashCache_TouchLastElement32(t *testing.T) {
	ctxt := &context{}
	cache := newHashCache(3, 3)
	cache.getHash32(ctxt, []byte{byte(1)})
	cache.getHash32(ctxt, []byte{byte(2)})
	cache.getHash32(ctxt, []byte{byte(3)})
	cache.getHash32(ctxt, []byte{byte(1)})

	if cache.tail32.key != [32]byte{byte(2)} {
		t.Fatalf("cache tail should have been updated")
	}
}

func TestHashCache_TouchLastElement64(t *testing.T) {
	ctxt := &context{}
	cache := newHashCache(3, 3)
	cache.getHash64(ctxt, []byte{byte(1)})
	cache.getHash64(ctxt, []byte{byte(2)})
	cache.getHash64(ctxt, []byte{byte(3)})
	cache.getHash64(ctxt, []byte{byte(1)})

	if cache.tail64.key != [64]byte{byte(2)} {
		t.Fatalf("cache tail should have been updated")
	}
}

func TestHashCache_ElementsMaintainLRUOrder32(t *testing.T) {
	cache := newHashCache(3, 3)
	ctxt := &context{
		code:  Code{Instruction{STOP, 0x0000}},
		stack: NewStack(),
	}
	data1 := []byte{0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0a,
		0x0b, 0x0c, 0x0d, 0x0e, 0x0f, 0x10, 0x11, 0x12, 0x13, 0x14, 0x15,
		0x16, 0x17, 0x18, 0x19, 0x1a, 0x1b, 0x1c, 0x1d, 0x1e, 0x1f}
	data2 := bytes.Clone(data1)
	data2[31] = 0x01
	data3 := bytes.Clone(data1)
	data3[31] = 0x02
	_ = cache.hash(ctxt, data1)
	_ = cache.hash(ctxt, data2)
	_ = cache.hash(ctxt, data3)
	entry1 := cache.hash(ctxt, data1)
	if entry1 != cache.head32.hash {
		t.Fatalf("last hashed element should be kept as head")
	}
	entry3 := cache.hash(ctxt, data3)
	if entry3 != cache.head32.hash {
		t.Fatalf("last hashed element should be kept as head")
	}
	entry2 := cache.hash(ctxt, data2)
	if entry2 != cache.head32.hash {
		t.Fatalf("last hashed element should be kept as head")
	}
}

func TestHashCache_ElementsMaintainLRUOrder64(t *testing.T) {
	cache := newHashCache(3, 3)
	ctxt := &context{
		code:  Code{Instruction{STOP, 0x0000}},
		stack: NewStack(),
	}
	data1 := []byte{0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0a,
		0x0b, 0x0c, 0x0d, 0x0e, 0x0f, 0x10, 0x11, 0x12, 0x13, 0x14, 0x15,
		0x16, 0x17, 0x18, 0x19, 0x1a, 0x1b, 0x1c, 0x1d, 0x1e, 0x1f, 0x20,
		0x21, 0x22, 0x23, 0x24, 0x25, 0x26, 0x27, 0x28, 0x29, 0x2a, 0x2b,
		0x2c, 0x2d, 0x2e, 0x2f, 0x30, 0x31, 0x32, 0x33, 0x34, 0x35, 0x36,
		0x37, 0x38, 0x39, 0x3a, 0x3b, 0x3c, 0x3d, 0x3e, 0x3f}
	data2 := bytes.Clone(data1)
	data2[63] = 0x01
	data3 := bytes.Clone(data1)
	data3[63] = 0x02
	_ = cache.hash(ctxt, data1)
	_ = cache.hash(ctxt, data2)
	_ = cache.hash(ctxt, data3)
	entry1 := cache.hash(ctxt, data1)
	if entry1 != cache.head64.hash {
		t.Fatalf("last hashed element should be kept as head")
	}
	entry3 := cache.hash(ctxt, data3)
	if entry3 != cache.head64.hash {
		t.Fatalf("last hashed element should be kept as head")
	}
	entry2 := cache.hash(ctxt, data2)
	if entry2 != cache.head64.hash {
		t.Fatalf("last hashed element should be kept as head")
	}
}

func TestHashCache_MaxSizeDeletesOldest32(t *testing.T) {
	cache := newHashCache(3, 3)
	ctxt := &context{
		code:  Code{Instruction{STOP, 0x0000}},
		stack: NewStack(),
	}
	data1 := []byte{0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0a,
		0x0b, 0x0c, 0x0d, 0x0e, 0x0f, 0x10, 0x11, 0x12, 0x13, 0x14, 0x15,
		0x16, 0x17, 0x18, 0x19, 0x1a, 0x1b, 0x1c, 0x1d, 0x1e, 0x1f}
	data2 := bytes.Clone(data1)
	data2[31] = 0x01
	data3 := bytes.Clone(data1)
	data3[31] = 0x02
	data4 := bytes.Clone(data1)
	data4[31] = 0x03

	_ = cache.hash(ctxt, data1)
	_ = cache.hash(ctxt, data2)
	_ = cache.hash(ctxt, data3)

	if cache.tail32.key != [32]byte(data1) {
		t.Fatalf("oldest entry should be %v, but got %v", data1, cache.tail32.key)
	}
	if cache.head32.key != [32]byte(data3) {
		t.Fatalf("newest entry should be %v, but got %v", data3, cache.head32.key)
	}

	_ = cache.hash(ctxt, data4)

	if cache.tail32.key != [32]byte(data2) {
		t.Fatalf("oldest entry should be updated to %v, but got %v", data2, cache.tail32.key)
	}
	if cache.head32.key != [32]byte(data4) {
		t.Fatalf("newest entry should be updated to %v, but got %v", data4, cache.head32.key)
	}
}

func TestHashCache_MaxSizeDeletesOldest64(t *testing.T) {
	cache := newHashCache(3, 3)
	ctxt := &context{
		code:  Code{Instruction{STOP, 0x0000}},
		stack: NewStack(),
	}
	data1 := []byte{0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0a,
		0x0b, 0x0c, 0x0d, 0x0e, 0x0f, 0x10, 0x11, 0x12, 0x13, 0x14, 0x15,
		0x16, 0x17, 0x18, 0x19, 0x1a, 0x1b, 0x1c, 0x1d, 0x1e, 0x1f, 0x20,
		0x21, 0x22, 0x23, 0x24, 0x25, 0x26, 0x27, 0x28, 0x29, 0x2a, 0x2b,
		0x2c, 0x2d, 0x2e, 0x2f, 0x30, 0x31, 0x32, 0x33, 0x34, 0x35, 0x36,
		0x37, 0x38, 0x39, 0x3a, 0x3b, 0x3c, 0x3d, 0x3e, 0x3f}
	data2 := bytes.Clone(data1)
	data2[63] = 0x01
	data3 := bytes.Clone(data1)
	data3[63] = 0x02
	data4 := bytes.Clone(data1)
	data4[63] = 0x03

	_ = cache.hash(ctxt, data1)
	_ = cache.hash(ctxt, data2)
	_ = cache.hash(ctxt, data3)

	if cache.tail64.key != [64]byte(data1) {
		t.Fatalf("oldest entry should be %v, but got %v", data1, cache.tail64.key)
	}
	if cache.head64.key != [64]byte(data3) {
		t.Fatalf("newest entry should be %v, but got %v", data3, cache.head64.key)
	}

	_ = cache.hash(ctxt, data4)

	if cache.tail64.key != [64]byte(data2) {
		t.Fatalf("oldest entry should be updated to %v, but got %v", data2, cache.tail64.key)
	}
	if cache.head64.key != [64]byte(data4) {
		t.Fatalf("newest entry should be updated to %v, but got %v", data4, cache.head64.key)
	}
}
