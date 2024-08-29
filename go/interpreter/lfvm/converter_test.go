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

func TestNewConverter_UsesDefaultCapacity(t *testing.T) {
	converter, err := NewConverter(ConversionConfig{})
	if err != nil {
		t.Fatalf("failed to create converter: %v", err)
	}
	if want, got := (1 << 30), converter.config.CacheSize; got != want {
		t.Errorf("Expected default cache capacity of %d, got %d", want, got)
	}
}

func TestNewConverter_CacheCanBeDisabled(t *testing.T) {
	converter, err := NewConverter(ConversionConfig{
		CacheSize: -1,
	})
	if err != nil {
		t.Fatalf("failed to create converter: %v", err)
	}
	if want, got := -1, converter.config.CacheSize; got != want {
		t.Errorf("Expected default cache capacity of %d, got %d", want, got)
	}
	if converter.cache != nil {
		t.Errorf("Expected cache to be disabled")
	}
	// Conversion should still work without a nil pointer dereference.
	converter.Convert([]byte{0}, &tosca.Hash{0})
}

func TestNewConverter_TooSmallCapacityLeadsToCreationIssues(t *testing.T) {
	_, err := NewConverter(ConversionConfig{
		CacheSize: maxCachedCodeLength / 2,
	})
	if err == nil {
		t.Fatalf("expected error when creating converter with too small cache size")
	}
}

func TestConverter_LongExampleCode(t *testing.T) {
	converter, err := NewConverter(ConversionConfig{})
	if err != nil {
		t.Fatalf("failed to create converter: %v", err)
	}
	converter.Convert(longExampleCode, nil)
}

func TestConverter_LongExampleLength(t *testing.T) {
	converter, err := NewConverter(ConversionConfig{})
	if err != nil {
		t.Fatalf("failed to create converter: %v", err)
	}
	index := 1 << 16
	newLongCode := make([]byte, index+3)
	newLongCode[index+1] = byte(vm.PC)
	res := converter.Convert(newLongCode, nil)
	if res[index+1].opcode != INVALID {
		t.Errorf("last instruction should be invalid but got %v", res[index+1])
	}
}

func TestConverter_InputsAreCachedUsingHashAsKey(t *testing.T) {
	converter, err := NewConverter(ConversionConfig{})
	if err != nil {
		t.Fatalf("failed to create converter: %v", err)
	}
	code := []byte{byte(vm.STOP)}
	hash := tosca.Hash{byte(1)}
	want := converter.Convert(code, &hash)
	got := converter.Convert(code, &hash)
	if &want[0] != &got[0] { // < it needs to be the same slice
		t.Errorf("cached conversion result not returned")
	}
}

func TestConverter_CacheSizeLimitIsEnforced(t *testing.T) {
	for _, limit := range []int{10, 100, 1000} {
		converter, err := NewConverter(ConversionConfig{
			CacheSize: limit * maxCachedCodeLength,
		})
		if err != nil {
			t.Fatalf("failed to create converter: %v", err)
		}
		for i := 0; i < limit*10; i++ {
			hash := tosca.Hash{byte(i), byte(i >> 8), byte(i >> 16)}
			converter.Convert([]byte{0}, &hash)
		}
		if got := len(converter.cache.Keys()); got > limit {
			t.Errorf("Conversion cache grew to %d entries", got)
		}
	}
}

func TestConverter_ExceedinglyLongCodesAreNotCached(t *testing.T) {
	converter, err := NewConverter(ConversionConfig{})
	if err != nil {
		t.Fatalf("failed to create converter: %v", err)
	}
	if want, got := 0, len(converter.cache.Keys()); want != got {
		t.Errorf("Expected %d entries in the cache, got %d", want, got)
	}
	converter.Convert([]byte{0}, &tosca.Hash{0})
	if want, got := 1, len(converter.cache.Keys()); want != got {
		t.Errorf("Expected %d entries in the cache, got %d", want, got)
	}
	// Codes with an excessive length should not be cached.
	converter.Convert(make([]byte, maxCachedCodeLength+1), &tosca.Hash{1})
	if want, got := 1, len(converter.cache.Keys()); want != got {
		t.Errorf("Expected %d entries in the cache, got %d", want, got)
	}
}

func TestConverter_ResultsAreCached(t *testing.T) {
	converter, err := NewConverter(ConversionConfig{})
	if err != nil {
		t.Fatalf("failed to create converter: %v", err)
	}
	code := []byte{byte(vm.STOP)}
	hash := tosca.Hash{byte(1)}
	want := converter.Convert(code, &hash)
	if got, found := converter.cache.Get(hash); !found || !slices.Equal(want, got) {
		t.Errorf("converted code not added to cache")
	}
}

func TestConvert_AllValidOperationsAreCoveredByConversionTable(t *testing.T) {
	// Test that all EVM instructions are covered.
	for i := 0; i < 256; i++ {
		code := vm.OpCode(i)
		if !vm.IsValid(code) {
			continue
		}

		// Push operations are not required to be mapped, they are handled explicitly.
		if vm.PUSH1 <= code && code <= vm.PUSH32 {
			continue
		}

		if op_2_op[code] == INVALID && vm.IsValid(code) {
			t.Errorf("Missing instruction coverage for %v", code)
		}
	}
}

func TestConverterGenPcMapFailsWithSuperInstructions(t *testing.T) {
	_, err := GenPcMapWithSuperInstructions([]byte{0x00})
	if err == nil {
		t.Errorf("program counter mapping does not support super instructions yet")
	}
}

func BenchmarkConvertLongExampleCode(b *testing.B) {
	converter, err := NewConverter(ConversionConfig{})
	if err != nil {
		b.Fatalf("failed to create converter: %v", err)
	}
	for i := 0; i < b.N; i++ {
		converter.Convert(longExampleCode, nil)
	}
}

func BenchmarkConversionCacheLookupSpeed(b *testing.B) {
	// This benchmark measures the lookup speed of the conversion cache
	// by converting the same code snippet multiple times.
	converter, err := NewConverter(ConversionConfig{})
	if err != nil {
		b.Fatalf("failed to create converter: %v", err)
	}
	hash := &tosca.Hash{0}
	for i := 0; i < b.N; i++ {
		converter.Convert(nil, hash)
	}
}

func BenchmarkConversionCacheUpdateSpeed(b *testing.B) {
	// This benchmark measures the update speed of the conversion cache
	// by converting codes with different (reported) hashes.
	converter, err := NewConverter(ConversionConfig{})
	if err != nil {
		b.Fatalf("failed to create converter: %v", err)
	}
	for i := 0; i < b.N; i++ {
		hash := tosca.Hash{byte(i), byte(i >> 8), byte(i >> 16), byte(i >> 24)}
		converter.Convert(nil, &hash)
	}
}
