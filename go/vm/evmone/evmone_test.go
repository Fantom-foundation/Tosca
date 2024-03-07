package evmone

import (
	"testing"

	"github.com/Fantom-foundation/Tosca/go/examples"
	"github.com/Fantom-foundation/Tosca/go/vm"
)

var variants = []string{
	"evmone",
	"evmone-basic",
	"evmone-advanced",
}

func TestFib10(t *testing.T) {
	const arg = 10

	example := examples.GetFibExample()

	// compute expected value
	wanted := example.RunReference(arg)

	for _, variant := range variants {
		t.Run(variant, func(t *testing.T) {
			vm := vm.GetInterpreter(variant)
			got, err := example.RunOn(vm, arg)
			if err != nil {
				t.Fatalf("running the fib example failed: %v", err)
			}

			if got.Result != wanted {
				t.Fatalf("unexpected result, wanted %v, got %v", wanted, got.Result)
			}
		})
	}
}

func BenchmarkFib10(b *testing.B) {
	benchmarkFib(b, 10)
}

func benchmarkFib(b *testing.B, arg int) {
	example := examples.GetFibExample()

	// compute expected value
	wanted := example.RunReference(arg)

	for _, variant := range variants {
		b.Run(variant, func(b *testing.B) {
			vm := vm.GetInterpreter(variant)
			for i := 0; i < b.N; i++ {
				got, err := example.RunOn(vm, arg)
				if err != nil {
					b.Fatalf("running the fib example failed: %v", err)
				}

				if wanted != got.Result {
					b.Fatalf("unexpected result, wanted %d, got %d", wanted, got.Result)
				}
			}
		})
	}
}
