package evmzero

import (
	"testing"

	"github.com/Fantom-foundation/Tosca/go/examples"
	"github.com/ethereum/go-ethereum/core/vm"
)

func TestFib10(t *testing.T) {
	const arg = 10

	example := examples.GetFibExample()

	// compute expected value
	wanted := example.RunReference(arg)

	t.Run("evmzero", func(t *testing.T) {
		interpreter := vm.NewInterpreter("evmzero", &vm.EVM{}, vm.Config{})

		got, err := example.RunOn(interpreter, arg)
		if err != nil {
			t.Fatalf("running the fib example failed: %v", err)
		}

		if got.Result != wanted {
			t.Fatalf("unexpected result, wanted %v, got %v", wanted, got.Result)
		}
	})
}

func TestEvmzero_DumpProfiler(t *testing.T) {
	example := examples.GetFibExample()
	interpreter := vm.NewInterpreter("evmzero-profiling", &vm.EVM{}, vm.Config{})
	for i := 0; i < 10; i++ {
		example.RunOn(interpreter, 10)
		// This is just printing to the output stream, but it could
		// be something more sophisticated sending back some string or
		// otherwise encoded statistics data ...
		DumpProfiler(interpreter)
		if i == 5 {
			ResetProfiler(interpreter)
		}
	}
}

func BenchmarkNewEvmcInterpreter(b *testing.B) {
	b.Run("evmzero", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			vm.NewInterpreter("evmzero", &vm.EVM{}, vm.Config{})
		}
	})
}

func BenchmarkFib10(b *testing.B) {
	benchmarkFib(b, 10)
}

func benchmarkFib(b *testing.B, arg int) {
	example := examples.GetFibExample()

	// compute expected value
	wanted := example.RunReference(arg)

	b.Run("evmzero", func(b *testing.B) {
		interpreter := vm.NewInterpreter("evmzero", &vm.EVM{}, vm.Config{})

		for i := 0; i < b.N; i++ {
			got, err := example.RunOn(interpreter, arg)
			if err != nil {
				b.Fatalf("running the fib example failed: %v", err)
			}

			if wanted != got.Result {
				b.Fatalf("unexpected result, wanted %d, got %d", wanted, got.Result)
			}
		}
	})
}
