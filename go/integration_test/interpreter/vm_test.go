// Copyright (c) 2024 Fantom Foundation
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at fantom.foundation/bsl11.
//
// Change Date: 2028-4-16
//
// On the date above, in accordance with the Business Source License, use of
// this software will be governed by the GNU Lesser General Public License v3.

package interpreter_test

import (
	"fmt"
	"testing"

	"github.com/Fantom-foundation/Tosca/go/examples"
	"github.com/Fantom-foundation/Tosca/go/lib/rust"
	"github.com/Fantom-foundation/Tosca/go/tosca"
	"go.uber.org/mock/gomock"
	// Enable this import to see C/C++ symbols in CPU profile data.
	// This import is commented out because it would affect all binaries this
	// package gets imported in and in some cases this library causes Go
	// symbols to be hidden. Also, the library has build issues on MacOS.
	// _ "github.com/ianlancetaylor/cgosymbolizer"
)

var (
	testExamples = []examples.Example{
		examples.GetIncrementExample(),
		examples.GetFibExample(),
		examples.GetSha3Example(),
		examples.GetArithmeticExample(),
		examples.GetMemoryExample(),
		examples.GetJumpdestAnalysisExample(),
		examples.GetStopAnalysisExample(),
		examples.GetPush1AnalysisExample(),
		examples.GetPush32AnalysisExample(),
	}
)

func TestExamples_ComputesCorrectResult(t *testing.T) {
	for _, example := range testExamples {
		for _, variant := range getAllInterpreterVariantsForTests() {
			vm, err := tosca.NewInterpreter(variant)
			if err != nil {
				t.Fatalf("failed to load %s interpreter: %v", variant, err)
			}
			for i := 0; i < 10; i++ {
				t.Run(fmt.Sprintf("%s-%s-%d", example.Name, variant, i), func(t *testing.T) {
					want := example.RunReference(i)
					got, err := example.RunOn(vm, i)
					if err != nil {
						t.Fatalf("error processing contract: %v", err)
					}
					if want != got.Result {
						t.Fatalf("incorrect result, wanted %d, got %d", want, got.Result)
					}
				})
			}
		}
	}
}

func TestExamples_ComputesCorrectGasPrice(t *testing.T) {
	for _, example := range testExamples {
		for _, revision := range revisions {
			reference, err := tosca.NewInterpreter("geth")
			if err != nil {
				t.Fatalf("failed to load geth interpreter: %v", err)
			}
			for _, variant := range getAllInterpreterVariantsForTests() {
				vm, err := tosca.NewInterpreter(variant)
				if err != nil {
					t.Fatalf("failed to load %s interpreter: %v", variant, err)
				}
				for i := 0; i < 10; i++ {
					t.Run(fmt.Sprintf("%s-%s-%s-%d", example.Name, revision, variant, i), func(t *testing.T) {
						want, err := example.RunOn(reference, i)
						if err != nil {
							t.Fatalf("failed to run reference VM: %v", err)
						}

						got, err := example.RunOn(vm, i)
						if err != nil {
							t.Fatalf("error processing contract: %v", err)
						}

						if want.UsedGas != got.UsedGas {
							t.Errorf("incorrect gas usage, wanted %d, got %d", want.UsedGas, got.UsedGas)
						}
					})
				}
			}
		}
	}
}

func BenchmarkEmpty(b *testing.B) {
	ctxt := gomock.NewController(b)
	runContext := tosca.NewMockRunContext(ctxt)
	emptyRunParameters := tosca.Parameters{
		Context: runContext,
	}
	for _, variant := range getAllInterpreterVariantsForTests() {
		interpreter, err := tosca.NewInterpreter(variant)
		if err != nil {
			b.Fatalf("failed to load %s interpreter: %v", variant, err)
		}
		b.Run(variant, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				_, err := interpreter.Run(emptyRunParameters)
				if err != nil {
					b.Fatalf("error running empty example: %v", err)
				}
			}
		})
	}
}

func BenchmarkStaticOverhead(b *testing.B) {
	defer rust.DumpRustCoverageData("/tmp/pgo-data/static_overhead") // dump profiling data for pgo
	// this is just to the benchmark name consists of 3 parts and can be matched by regex
	b.Run("1", func(b *testing.B) {
		benchmark(b, examples.GetStaticOverheadExample(), 1)
	})
}

func BenchmarkInc(b *testing.B) {
	defer rust.DumpRustCoverageData("/tmp/pgo-data/inc") // dump profiling data for pgo
	args := []int{1, 10}
	for _, i := range args {
		b.Run(fmt.Sprintf("%d", i), func(b *testing.B) {
			benchmark(b, examples.GetIncrementExample(), i)
		})
	}
}

func BenchmarkFib(b *testing.B) {
	defer rust.DumpRustCoverageData("/tmp/pgo-data/fib") // dump profiling data for pgo
	args := []int{1, 5, 10, 15, 20}
	for _, i := range args {
		b.Run(fmt.Sprintf("%d", i), func(b *testing.B) {
			benchmark(b, examples.GetFibExample(), i)
		})
	}
}

func BenchmarkSha3(b *testing.B) {
	defer rust.DumpRustCoverageData("/tmp/pgo-data/sha3") // dump profiling data for pgo
	args := []int{1, 10, 100, 1000}
	for _, i := range args {
		b.Run(fmt.Sprintf("%d", i), func(b *testing.B) {
			benchmark(b, examples.GetSha3Example(), i)
		})
	}
}

func BenchmarkArith(b *testing.B) {
	defer rust.DumpRustCoverageData("/tmp/pgo-data/arith") // dump profiling data for pgo
	args := []int{1, 10, 100, 280}
	for _, i := range args {
		b.Run(fmt.Sprintf("%d", i), func(b *testing.B) {
			benchmark(b, examples.GetArithmeticExample(), i)
		})
	}
}

func BenchmarkMemory(b *testing.B) {
	defer rust.DumpRustCoverageData("/tmp/pgo-data/memory") // dump profiling data for pgo
	args := []int{1, 10, 100, 1000, 10000}
	for _, i := range args {
		b.Run(fmt.Sprintf("%d", i), func(b *testing.B) {
			benchmark(b, examples.GetMemoryExample(), i)
		})
	}
}

func BenchmarkAnalysis(b *testing.B) {
	defer rust.DumpRustCoverageData("/tmp/pgo-data/analysis") // dump profiling data for pgo
	examples := []examples.Example{
		examples.GetJumpdestAnalysisExample(),
		examples.GetStopAnalysisExample(),
		examples.GetPush1AnalysisExample(),
		examples.GetPush32AnalysisExample(),
	}
	for _, example := range examples {
		b.Run(example.Name, func(b *testing.B) {
			benchmark(b, example, 0)
		})
	}
}

func benchmark(b *testing.B, example examples.Example, arg int) {
	// compute expected value
	wanted := example.RunReference(arg)

	for _, variant := range getAllInterpreterVariantsForTests() {
		evm, err := tosca.NewInterpreter(variant)
		if err != nil {
			b.Fatalf("failed to load %s interpreter: %v", variant, err)
		}
		if pvm, ok := evm.(tosca.ProfilingInterpreter); ok {
			pvm.ResetProfile()
		}
		active := false
		b.Run(variant, func(b *testing.B) {
			active = true
			for i := 0; i < b.N; i++ {
				got, err := example.RunOn(evm, arg)
				if err != nil {
					b.Fatalf("running the %s example failed: %v", example.Name, err)
				}

				if wanted != got.Result {
					b.Fatalf("unexpected result, wanted %d, got %d", wanted, got.Result)
				}
			}
		})
		if pvm, ok := evm.(tosca.ProfilingInterpreter); active && ok {
			pvm.DumpProfile()
		}
	}
}
