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
	"math/rand"
	"reflect"
	"slices"
	"sync"
	"testing"
	"time"

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

func TestConverter_ConverterIsThreadSafe(t *testing.T) {
	// This test is to be run with --race to detect concurrency issues.
	const (
		NumGoroutines = 100
		NumSteps      = 1000
	)

	converter, err := NewConverter(ConversionConfig{})
	if err != nil {
		t.Fatalf("failed to create converter: %v", err)
	}
	code := []byte{byte(vm.STOP)}
	hash := tosca.Hash{byte(1)}

	var wg sync.WaitGroup
	wg.Add(NumGoroutines)
	for i := 0; i < NumGoroutines; i++ {
		go func(i int) {
			defer wg.Done()
			for j := 0; j < NumSteps; j++ {
				// read a value every go routine is requesting
				converter.Convert(code, &hash)
				// convert a value only this go routine is requesting
				converter.Convert(code, &tosca.Hash{byte(i), byte(j)})
			}
		}(i)
	}
	wg.Wait()
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

func TestConvertWithObserver_MapsEvmToLfvmPositions(t *testing.T) {
	code := []byte{
		byte(vm.ADD),
		byte(vm.PUSH1), 1,
		byte(vm.PUSH3), 1, 2, 3,
		byte(vm.SWAP1),
		byte(vm.JUMPDEST),
	}

	type pair struct {
		evm, lfvm int
	}
	var pairs []pair
	res := ConvertWithObserver(code, ConversionConfig{}, func(evm, lfvm int) {
		pairs = append(pairs, pair{evm, lfvm})
	})

	want := []pair{
		{0, 0},
		{1, 1},
		{3, 2},
		{7, 4},
		{8, 8},
	}

	if !slices.Equal(pairs, want) {
		t.Errorf("Expected %v, got %v", want, pairs)
	}

	for _, p := range pairs {
		in := vm.OpCode(code[p.evm])
		want := op_2_op[in]
		if vm.PUSH1 <= in && in <= vm.PUSH32 {
			want = PUSH1 + OpCode(in-vm.PUSH1)
		}
		got := res[p.lfvm].opcode
		if want != got {
			t.Errorf("Expected %v, got %v", want, got)
		}
	}
}

func TestConvertWithObserver_PreservesJumpDestLocations(t *testing.T) {
	r := rand.New(rand.NewSource(int64(time.Now().Nanosecond())))

	for i := 0; i < 100; i++ {
		code := make([]byte, 100)
		r.Read(code)

		type pair struct {
			evm, lfvm int
		}
		var pairs []pair
		res := ConvertWithObserver(code, ConversionConfig{}, func(evm, lfvm int) {
			pairs = append(pairs, pair{evm, lfvm})
		})

		// Check that all operations are mapped to matching operations.
		for _, p := range pairs {
			in := vm.OpCode(code[p.evm])
			want := op_2_op[in]
			if vm.PUSH1 <= in && in <= vm.PUSH32 {
				want = PUSH1 + OpCode(in-vm.PUSH1)
			}
			got := res[p.lfvm].opcode
			if want != got {
				t.Errorf("Expected %v, got %v", want, got)
			}
		}

		// Check that the position of JUMPDESTs is preserved.
		for _, p := range pairs {
			if vm.OpCode(code[p.evm]) == vm.JUMPDEST {
				if p.evm != p.lfvm {
					t.Errorf("Expected JUMPDEST at %d, got %d", p.evm, p.lfvm)
				}
			}
		}
	}
}

func TestConvertToLfvm_CodeWithSuperInstructions(t *testing.T) {
	tests := map[string]struct {
		evmCode []byte
		want    Code
	}{
		"PUSH1PUSH4DUP3": {
			[]byte{byte(vm.PUSH1), 0x01,
				byte(vm.PUSH4), 0x01, 0x02, 0x03, 0x04,
				byte(vm.DUP3)},
			Code{Instruction{PUSH1_PUSH4_DUP3, 0x0100},
				Instruction{DATA, 0x0102},
				Instruction{DATA, 0x0304},
			}},
		"PUSH1_PUSH1_PUSH1_SHL_SUB": {
			[]byte{byte(vm.PUSH1), 0x01,
				byte(vm.PUSH1), 0x01,
				byte(vm.PUSH1), 0x01,
				byte(vm.SHL),
				byte(vm.SUB)},
			Code{Instruction{PUSH1_PUSH1_PUSH1_SHL_SUB, 0x0101},
				Instruction{DATA, 0x0001},
			}},
		"AND_SWAP1_POP_SWAP2_SWAP1": {
			[]byte{byte(vm.AND), byte(vm.SWAP1), byte(vm.POP),
				byte(vm.SWAP2), byte(vm.SWAP1)},
			Code{Instruction{AND_SWAP1_POP_SWAP2_SWAP1, 0x0000}}},
		"ISZERO_PUSH2_JUMPI": {
			[]byte{byte(vm.ISZERO),
				byte(vm.PUSH2), 0x01, 0x02,
				byte(vm.JUMPI)},
			Code{Instruction{ISZERO_PUSH2_JUMPI, 0x0102}}},
		"SWAP2_SWAP1_POP_JUMP": {
			[]byte{byte(vm.SWAP2), byte(vm.SWAP1), byte(vm.POP),
				byte(vm.JUMP)},
			Code{Instruction{SWAP2_SWAP1_POP_JUMP, 0x0000}}},
		"SWAP1_POP_SWAP2_SWAP1": {
			[]byte{byte(vm.SWAP1), byte(vm.POP), byte(vm.SWAP2),
				byte(vm.SWAP1)},
			Code{Instruction{SWAP1_POP_SWAP2_SWAP1, 0x0000}}},
		"POP_SWAP2_SWAP1_POP": {
			[]byte{byte(vm.POP), byte(vm.SWAP2), byte(vm.SWAP1),
				byte(vm.POP)},
			Code{Instruction{POP_SWAP2_SWAP1_POP, 0x0000}}},
		"PUSH2_JUMP": {
			[]byte{byte(vm.PUSH2), 0x01, 0x02,
				byte(vm.JUMP)},
			Code{Instruction{PUSH2_JUMP, 0x0102}}},
		"PUSH2_JUMPI": {
			[]byte{byte(vm.PUSH2), 0x01, 0x02,
				byte(vm.JUMPI)},
			Code{Instruction{PUSH2_JUMPI, 0x0102}}},
		"PUSH1_PUSH1": {
			[]byte{byte(vm.PUSH1), 0x01,
				byte(vm.PUSH1), 0x01},
			Code{Instruction{PUSH1_PUSH1, 0x0101}}},
		"PUSH1_ADD": {
			[]byte{byte(vm.PUSH1), 0x01,
				byte(vm.ADD)},
			Code{Instruction{PUSH1_ADD, 0x0001}}},
		"PUSH1_SHL": {
			[]byte{byte(vm.PUSH1), 0x01,
				byte(vm.SHL)},
			Code{Instruction{PUSH1_SHL, 0x0001}}},
		"PUSH1_DUP1": {
			[]byte{byte(vm.PUSH1), 0x01,
				byte(vm.DUP1)},
			Code{Instruction{PUSH1_DUP1, 0x0001}}},
		"SWAP1_POP": {
			[]byte{byte(vm.SWAP1), byte(vm.POP)},
			Code{Instruction{SWAP1_POP, 0x0000}}},
		"POP_JUMP": {
			[]byte{byte(vm.POP), byte(vm.JUMP)},
			Code{Instruction{POP_JUMP, 0x0000}}},
		"POP_POP": {
			[]byte{byte(vm.POP), byte(vm.POP)},
			Code{Instruction{POP_POP, 0x0000}}},
		"SWAP2_SWAP1": {
			[]byte{byte(vm.SWAP2), byte(vm.SWAP1)},
			Code{Instruction{SWAP2_SWAP1, 0x0000}}},
		"SWAP2_POP": {
			[]byte{byte(vm.SWAP2), byte(vm.POP)},
			Code{Instruction{SWAP2_POP, 0x0000}}},
		"DUP2_MSTORE": {
			[]byte{byte(vm.DUP2), byte(vm.MSTORE)},
			Code{Instruction{DUP2_MSTORE, 0x0000}}},
		"DUP2_LT": {
			[]byte{byte(vm.DUP2), byte(vm.LT)},
			Code{Instruction{DUP2_LT, 0x0000}}},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			options := ConversionConfig{WithSuperInstructions: true}
			got := convert(test.evmCode, options)
			if !reflect.DeepEqual(test.want, got) {
				t.Fatalf("unexpected code, wanted %v, got %v", test.want, got)
			}
		})
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
