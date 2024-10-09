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
	"math"
	"math/rand"
	"slices"
	"sync"
	"testing"
	"time"
	"unsafe"

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
		const instructionSize = int(unsafe.Sizeof(Instruction{}))
		converter, err := NewConverter(ConversionConfig{
			CacheSize: limit * maxCachedCodeLength * instructionSize,
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
	res := convertWithObserver(code, ConversionConfig{}, func(evm, lfvm int) {
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
		if want, got := OpCode(code[p.evm]), res[p.lfvm].opcode; want != got {
			t.Errorf("Expected %v, got %v", want, got)
		}
	}
}

func TestConvertWithObserver_PreservesJumpDestLocations(t *testing.T) {
	r := rand.New(rand.NewSource(int64(time.Now().Nanosecond())))

	for i := 0; i < 100; i++ {
		code := make([]byte, 100)
		r.Read(code)

		mapping := map[int]int{}
		res := convertWithObserver(code, ConversionConfig{}, func(evm, lfvm int) {
			if _, found := mapping[evm]; found {
				t.Errorf("Duplicate mapping for EVM position %d", evm)
			}
			mapping[evm] = lfvm
		})

		// Check that all operations are mapped to matching operations.
		for evm, lfvm := range mapping {
			if want, got := OpCode(code[evm]), res[lfvm].opcode; want != got {
				t.Errorf("Expected %v, got %v", want, got)
			}
		}

		// Check that the position of JUMPDESTs is preserved.
		for evm, lfvm := range mapping {
			if vm.OpCode(code[evm]) == vm.JUMPDEST {
				if evm != lfvm {
					t.Errorf("Expected JUMPDEST at %d, got %d", evm, lfvm)
				}
			}
		}

		// Check that all JUMPDEST operations got mapped.
		for i := 0; i < len(code); i++ {
			cur := vm.OpCode(code[i])
			if cur == vm.JUMPDEST {
				if _, found := mapping[i]; !found {
					t.Errorf("JUMPDEST at %d not mapped", i)
				}
			}
			if vm.PUSH1 <= cur && cur <= vm.PUSH32 {
				i += int(cur - vm.PUSH1 + 1)
			}
		}
	}
}

func TestConvert_ProgramCounterBeyond16bitAreConvertedIntoInvalidInstructions(t *testing.T) {
	max := math.MaxUint16
	positions := []int{0, 1, max / 2, max - 1, max, max + 1}
	code := make([]byte, max+2)
	for _, pos := range positions {
		code[pos] = byte(vm.PC)
	}
	res := convert(code, ConversionConfig{})

	for _, pos := range positions {
		want := PC
		if pos > max {
			want = INVALID
		}
		if got := res[pos].opcode; want != got {
			t.Errorf("Expected %v at position %d, got %v", want, pos, got)
		}
	}
}

func TestConvert_BaseInstructionsAreConvertedToEquivalents(t *testing.T) {
	config := ConversionConfig{
		WithSuperInstructions: false,
	}
	for _, op := range allOpCodesWhere(OpCode.isBaseInstruction) {
		t.Run(op.String(), func(t *testing.T) {
			code := []byte{byte(op)}
			res := convert(code, config)
			if want, got := op, res[0].opcode; want != got {
				t.Errorf("Expected %v, got %v", want, got)
			}
		})
	}
}

func TestConvert_PushOperationsUsePaddedImmediateData(t *testing.T) {
	data := []byte{}
	for i := 0; i < 32; i++ {
		data = append(data, byte(i+1))
	}
	for op := PUSH1; op <= PUSH32; op++ {
		t.Run(op.String(), func(t *testing.T) {
			// Test all possible truncated push data lengths.
			length := int(op) - int(PUSH1) + 1
			for i := 0; i <= length; i++ {
				code := append([]byte{byte(op)}, data[:i]...)
				res := convert(code, ConversionConfig{})

				// the push operation is correct
				if want, got := op, res[0].opcode; want != got {
					t.Errorf("Expected %v, got %v", want, got)
				}

				// all the rest is data
				for i, op := range res[1:] {
					if op.opcode != DATA {
						t.Errorf("Expected DATA at position %d, got %v", i, op.opcode)
					}
				}

				// there is enough data
				if got, want := len(res), length/2+length%2; got != want {
					t.Errorf("Expected %d instructions, got %d", want, got)
				}

				// re-construct data
				gotData := make([]byte, length+1)
				for i, op := range res {
					gotData[i*2] = byte(op.arg >> 8)
					gotData[i*2+1] = byte(op.arg)
				}
				gotData = gotData[:length]

				// make sure prefix is correct
				if want, got := data[:i], gotData[:i]; !bytes.Equal(want, got) {
					t.Errorf("Expected %x, got %x", want, got)
				}

				// make sure zero padding is correct
				if want, got := make([]byte, length-i), gotData[i:]; !bytes.Equal(want, got) {
					t.Errorf("Expected %x, got %x", want, got)
				}
			}
		})
	}
}

func TestConvert_AllJumpToOperationsPointToSubsequentJumpdest(t *testing.T) {
	r := rand.New(rand.NewSource(int64(time.Now().Nanosecond())))

	counter := 0
	for i := 0; i < 1000; i++ {
		code := make([]byte, 100)
		r.Read(code)
		res := convert(code, ConversionConfig{})

		for i, instruction := range res {
			if instruction.opcode == JUMP_TO {
				counter++
				trg := instruction.arg
				if trg <= uint16(i) {
					t.Errorf("JUMP_TO %d points to preceding position %d", trg, i)
				}
				if trg >= uint16(len(res)) {
					t.Fatalf("JUMP_TO %d out of bounds", trg)
				}
				if res[trg].opcode != JUMPDEST {
					t.Errorf("JUMP_TO %d does not point to JUMPDEST", trg)
				}

				// Everything from the JUMP_TO to to the jump destination is a
				// NOOP instruction.
				for pos := i + 1; pos < int(trg); pos++ {
					if res[pos].opcode != NOOP {
						t.Errorf("Expected NOOP at position %d, got %v", pos, res[pos].opcode)
					}
				}
			}
		}
	}
	if counter == 0 {
		t.Errorf("No JUMP_TO operations found")
	}
}

func TestConvert_SI_WhenEnabledSuperInstructionsAreUsed(t *testing.T) {
	config := ConversionConfig{
		WithSuperInstructions: true,
	}
	for _, op := range allOpCodesWhere(OpCode.isSuperInstruction) {
		t.Run(op.String(), func(t *testing.T) {
			code := []byte{}
			for _, op := range op.decompose() {
				code = append(code, byte(op))
				if PUSH1 <= op && op <= PUSH32 {
					code = append(code, make([]byte, int(op)-int(PUSH1)+1)...)
				}
			}
			res := convert(code, config)
			if want, got := op, res[0].opcode; want != got {
				t.Errorf("Expected %v, got %v", want, got)
			}
		})
	}
}

func TestConvert_SI_WhenDisabledNoSuperInstructionsAreUsed(t *testing.T) {
	config := ConversionConfig{
		WithSuperInstructions: false,
	}
	for _, op := range allOpCodesWhere(OpCode.isSuperInstruction) {
		t.Run(op.String(), func(t *testing.T) {
			code := []byte{}
			for _, op := range op.decompose() {
				code = append(code, byte(op))
			}

			res := convert(code, config)
			for i, instr := range res {
				if instr.opcode.isSuperInstruction() {
					t.Errorf("Super instruction %v used at position %d", instr.opcode, i)
				}
			}
		})
	}
}

func TestConverter_SI_FallsBackToLFVMInstructionsWhenNoSuperInstructionIsFit(t *testing.T) {

	config := ConversionConfig{
		WithSuperInstructions: true,
	}
	code := []byte{byte(PUSH2), 0x12, 0x34, byte(ADD), byte(PUSH1), 0x56, byte(SUB)}
	convertedCode := convert(code, config)
	if len(convertedCode) != 4 {
		t.Fatalf("Expected 4 instructions, got %d", len(convertedCode))
	}
	for _, inst := range convertedCode {
		if inst.opcode.isSuperInstruction() {
			t.Errorf("Super instruction %v used", inst.opcode)
		}
	}
}

func benchmarkConvertCode(b *testing.B, code []byte, config ConversionConfig) {
	converter, err := NewConverter(config)
	if err != nil {
		b.Fatalf("failed to create converter: %v", err)
	}
	for i := 0; i < b.N; i++ {
		converter.Convert(code, nil)
	}
}

func BenchmarkConvertLongExampleCodeNoCache(b *testing.B) {
	benchmarkConvertCode(b, longExampleCode, ConversionConfig{CacheSize: -1})
}

func BenchmarkConvertLongExampleCode(b *testing.B) {
	benchmarkConvertCode(b, longExampleCode, ConversionConfig{})
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
