package vm

import (
	"fmt"
	"testing"

	"github.com/ethereum/go-ethereum/core/vm"

	"github.com/Fantom-foundation/Tosca/go/examples"
	_ "github.com/Fantom-foundation/Tosca/go/vm/evmone"
	_ "github.com/Fantom-foundation/Tosca/go/vm/lfvm"
)

var (
	testExamples = []examples.Example{
		examples.GetIncrementExample(),
		examples.GetFibExample(),
		examples.GetSha3Example(),
	}
)

func TestExamples_ComputesCorrectResult(t *testing.T) {
	evm := newTestEVM(London)
	for _, example := range testExamples {
		for _, variant := range variants {
			interpreter := vm.NewInterpreter(variant, evm, vm.Config{})
			for i := 0; i < 10; i++ {
				t.Run(fmt.Sprintf("%s-%s-%d", example.Name, variant, i), func(t *testing.T) {
					want := example.RunReference(i)
					got, err := example.RunOn(interpreter, i)
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
			evm := newTestEVM(revision)
			reference := vm.NewInterpreter("geth", evm, vm.Config{})
			for _, variant := range variants {
				interpreter := vm.NewInterpreter(variant, evm, vm.Config{})
				for i := 0; i < 10; i++ {
					t.Run(fmt.Sprintf("%s-%s-%s-%d", example.Name, revision, variant, i), func(t *testing.T) {
						want, err := example.RunOn(reference, i)
						if err != nil {
							t.Fatalf("failed to run reference VM: %v", err)
						}

						got, err := example.RunOn(interpreter, i)
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
func benchmark(b *testing.B, example examples.Example, arg int) {
	// compute expected value
	wanted := example.RunReference(arg)

	evm := newTestEVM(London)

	for _, variant := range variants {
		b.Run(variant, func(b *testing.B) {
			interpreter := vm.NewInterpreter(variant, evm, vm.Config{})

			for i := 0; i < b.N; i++ {
				got, err := example.RunOn(interpreter, arg)
				if err != nil {
					b.Fatalf("running the fib example failed: %v", err)
				}

				if wanted != got.Result {
					b.Fatalf("unexpected result, wanted %d, got %d", wanted, got)
				}
			}
		})
	}
}
