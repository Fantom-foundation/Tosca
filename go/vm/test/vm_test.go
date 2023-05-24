package vm_test

import (
	"fmt"
	"testing"

	"github.com/Fantom-foundation/Tosca/go/examples"
	_ "github.com/Fantom-foundation/Tosca/go/vm/evmone"
	_ "github.com/Fantom-foundation/Tosca/go/vm/evmzero"
	_ "github.com/Fantom-foundation/Tosca/go/vm/lfvm"
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
		for _, variant := range Variants {
			evm := GetCleanEVM(London, variant, nil)
			for i := 0; i < 10; i++ {
				t.Run(fmt.Sprintf("%s-%s-%d", example.Name, variant, i), func(t *testing.T) {
					want := example.RunReference(i)
					got, err := example.RunOn(evm.GetInterpreter(), i)
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
			reference := GetCleanEVM(revision, "geth", nil)
			for _, variant := range Variants {
				evm := GetCleanEVM(revision, variant, nil)
				for i := 0; i < 10; i++ {
					t.Run(fmt.Sprintf("%s-%s-%s-%d", example.Name, revision, variant, i), func(t *testing.T) {
						want, err := example.RunOn(reference.GetInterpreter(), i)
						if err != nil {
							t.Fatalf("failed to run reference VM: %v", err)
						}

						got, err := example.RunOn(evm.GetInterpreter(), i)
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

func BenchmarkInc(b *testing.B) {
	args := []int{1, 10}
	for _, i := range args {
		b.Run(fmt.Sprintf("%d", i), func(b *testing.B) {
			benchmark(b, examples.GetIncrementExample(), i)
		})
	}
}

func BenchmarkFib(b *testing.B) {
	args := []int{1, 5, 10, 15, 20}
	for _, i := range args {
		b.Run(fmt.Sprintf("%d", i), func(b *testing.B) {
			benchmark(b, examples.GetFibExample(), i)
		})
	}
}

func BenchmarkSha3(b *testing.B) {
	args := []int{1, 10, 100, 1000}
	for _, i := range args {
		b.Run(fmt.Sprintf("%d", i), func(b *testing.B) {
			benchmark(b, examples.GetSha3Example(), i)
		})
	}
}

func BenchmarkArith(b *testing.B) {
	args := []int{1, 10, 100, 280}
	for _, i := range args {
		b.Run(fmt.Sprintf("%d", i), func(b *testing.B) {
			benchmark(b, examples.GetArithmeticExample(), i)
		})
	}
}

func BenchmarkMemory(b *testing.B) {
	args := []int{1, 10, 100, 1000, 10000}
	for _, i := range args {
		b.Run(fmt.Sprintf("%d", i), func(b *testing.B) {
			benchmark(b, examples.GetMemoryExample(), i)
		})
	}
}

func BenchmarkAnalysis(b *testing.B) {
	examples := []examples.Example{
		examples.GetJumpdestAnalysisExample(),
		examples.GetStopAnalysisExample(),
		examples.GetPush1AnalysisExample(),
		examples.GetPush32AnalysisExample(),
	}
	for _, example := range examples {
		b.Run(fmt.Sprintf("%s", example.Name), func(b *testing.B) {
			benchmark(b, example, 0)
		})
	}
}

func benchmark(b *testing.B, example examples.Example, arg int) {
	// compute expected value
	wanted := example.RunReference(arg)

	for _, variant := range Variants {
		evm := GetCleanEVM(London, variant, nil)
		b.Run(variant, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				got, err := example.RunOn(evm.GetInterpreter(), arg)
				if err != nil {
					b.Fatalf("running the %s example failed: %v", example.Name, err)
				}

				if wanted != got.Result {
					b.Fatalf("unexpected result, wanted %d, got %d", wanted, got.Result)
				}
			}
		})
	}
}
