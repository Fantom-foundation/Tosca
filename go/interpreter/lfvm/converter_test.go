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
	"slices"
	"testing"

	"github.com/Fantom-foundation/Tosca/go/tosca"
	"github.com/Fantom-foundation/Tosca/go/tosca/vm"
)

func TestConvertLongExampleCode(t *testing.T) {
	clearConversionCache()
	_ = Convert(longExampleCode, true, false, false, tosca.Hash{})
}

func TestConverterLongExampleLength(t *testing.T) {
	clearConversionCache()
	index := 1 << 16
	newLongCode := make([]byte, index+3)
	newLongCode[index+1] = byte(vm.PC)
	res := Convert(newLongCode, false, false, false, tosca.Hash{})
	if res[index+1].opcode != INVALID {
		t.Errorf("last instruction should be invalid but got %v", res[index+1])
	}
}

func TestConversionCacheSizeLimit(t *testing.T) {
	// This test checks that the conversion cache does not grow indefinitely
	// by converting a large number of different code snippets.
	clearConversionCache()
	const limit = codeCacheCapacity
	for i := 0; i < limit*10; i++ {
		hash := tosca.Hash{byte(i), byte(i >> 8), byte(i >> 16), byte(i >> 24)}
		_ = Convert([]byte{0}, false, false, false, hash)
	}
	if got := len(cache.Keys()); got > limit {
		t.Errorf("Conversion cache grew to %d entries", got)
	}
}

func TestConversion_CacheDoesNotCotainsCode(t *testing.T) {
	// This test checks that the conversion cache does not contain the code
	// after the conversion is done.
	clearConversionCache()
	code := Code{Instruction{STOP, 0x0000}}
	hash := tosca.Hash{byte(1), byte(1 >> 8), byte(1 >> 16), byte(1 >> 24)}
	cache.Add(hash, code)
	result := Convert([]byte{0}, false, false, false, hash)
	if wanted, _ := cache.Get(hash); !slices.Equal(result, wanted) {
		t.Errorf("Conversion cache contains the code")
	}
}

func TestConversion_GenPcMapFailsWithSuperInstructions(t *testing.T) {
	_, err := GenPcMapWithSuperInstructions([]byte{0x00})
	if err == nil {
		t.Errorf("prorgam counter mapping does not support super instructions yet")
	}
}

func BenchmarkConvertLongExampleCode(b *testing.B) {
	for i := 0; i < b.N; i++ {
		clearConversionCache()
		_ = Convert(longExampleCode, false, false, false, tosca.Hash{byte(i)})
	}
}

func BenchmarkConversionCacheLookupSpeed(b *testing.B) {
	// This benchmark measures the lookup speed of the conversion cache
	// by converting the same code snippet multiple times.
	clearConversionCache()
	for i := 0; i < b.N; i++ {
		_ = Convert([]byte{}, false, false, false, tosca.Hash{})
	}
}

func BenchmarkConversionCacheUpdateSpeed(b *testing.B) {
	// This benchmark measures the update speed of the conversion cache
	// by converting codes with different (reported) hashes.
	clearConversionCache()
	for i := 0; i < b.N; i++ {
		hash := tosca.Hash{byte(i), byte(i >> 8), byte(i >> 16), byte(i >> 24)}
		_ = Convert([]byte{}, false, false, false, hash)
	}
}
