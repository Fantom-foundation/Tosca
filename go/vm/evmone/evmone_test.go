package evmone

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

	interpreter := NewInterpreter(&vm.EVM{}, vm.Config{})

	got, err := example.RunOn(interpreter, arg)
	if err != nil {
		t.Fatalf("running the fib example failed: %v", err)
	}

	if got.Result != wanted {
		t.Fatalf("unexpected result, wanted %v, got %v", wanted, got.Result)
	}
}

func BenchmarkFib10(b *testing.B) {
	benchmarkFib(b, 10)
}

func benchmarkFib(b *testing.B, arg int) {
	example := examples.GetFibExample()

	// compute expected value
	wanted := example.RunReference(arg)

	interpreter := NewInterpreter(&vm.EVM{}, vm.Config{})

	for i := 0; i < b.N; i++ {
		got, err := example.RunOn(interpreter, arg)
		if err != nil {
			b.Fatalf("running the fib example failed: %v", err)
		}

		if wanted != got.Result {
			b.Fatalf("unexpected result, wanted %d, got %d", wanted, got)
		}
	}
}
