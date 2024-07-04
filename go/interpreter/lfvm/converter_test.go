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
	"testing"

	"github.com/Fantom-foundation/Tosca/go/tosca"
)

func TestConvertLongExampleCode(t *testing.T) {
	clearConversionCache()
	_, err := Convert(longExampleCode, false, false, false, tosca.Hash{})
	if err != nil {
		t.Errorf("Failed to convert example code with error %v", err)
	}
}

func TestConversionCacheSizeLimit(t *testing.T) {
	// This test checks that the conversion cache does not grow indefinitely
	// by converting a large number of different code snippets.
	clearConversionCache()
	const limit = codeCacheCapacity
	for i := 0; i < limit*10; i++ {
		hash := tosca.Hash{byte(i), byte(i >> 8), byte(i >> 16), byte(i >> 24)}
		_, err := Convert([]byte{0}, false, false, false, hash)
		if err != nil {
			t.Errorf("Failed to convert example code with error %v", err)
		}
	}
	if got := len(cache.Keys()); got > limit {
		t.Errorf("Conversion cache grew to %d entries", got)
	}

}

func BenchmarkConvertLongExampleCode(b *testing.B) {
	for i := 0; i < b.N; i++ {
		clearConversionCache()
		_, err := Convert(longExampleCode, false, false, false, tosca.Hash{byte(i)})
		if err != nil {
			b.Errorf("Failed to convert example code with error %v", err)
		}
	}
}

func BenchmarkConversionCacheLookupSpeed(b *testing.B) {
	// This benchmark measures the lookup speed of the conversion cache
	// by converting the same code snippet multiple times.
	clearConversionCache()
	for i := 0; i < b.N; i++ {
		_, err := Convert([]byte{}, false, false, false, tosca.Hash{})
		if err != nil {
			b.Errorf("Failed to convert example code with error %v", err)
		}
	}
}

func BenchmarkConversionCacheUpdateSpeed(b *testing.B) {
	// This benchmark measures the update speed of the conversion cache
	// by converting codes with different (reported) hashes.
	clearConversionCache()
	for i := 0; i < b.N; i++ {
		hash := tosca.Hash{byte(i), byte(i >> 8), byte(i >> 16), byte(i >> 24)}
		_, err := Convert([]byte{}, false, false, false, hash)
		if err != nil {
			b.Errorf("Failed to convert example code with error %v", err)
		}
	}
}
