package evmzero

import (
	"testing"

	"github.com/Fantom-foundation/Tosca/go/examples"
	"github.com/Fantom-foundation/Tosca/go/vm"
)

func TestFib10(t *testing.T) {
	const arg = 10

	example := examples.GetFibExample()

	// compute expected value
	wanted := example.RunReference(arg)

	vm := vm.GetVirtualMachine("evmzero")
	got, err := example.RunOn(vm, arg)
	if err != nil {
		t.Fatalf("running the fib example failed: %v", err)
	}

	if got.Result != wanted {
		t.Fatalf("unexpected result, wanted %v, got %v", wanted, got.Result)
	}
}

func TestEvmzero_DumpProfile(t *testing.T) {
	example := examples.GetFibExample()
	vmInstance, ok := vm.GetVirtualMachine("evmzero-profiling").(vm.ProfilingVM)
	if !ok || vmInstance == nil {
		t.Fatalf("profiling evmzero configuration does not support profiling")
	}
	for i := 0; i < 10; i++ {
		example.RunOn(vmInstance, 10)
		vmInstance.DumpProfile()
		if i == 5 {
			vmInstance.ResetProfile()
		}
	}
}

func BenchmarkNewEvmcInterpreter(b *testing.B) {
	b.Run("evmzero", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			vm.GetVirtualMachine("evmzero")
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
		vm := vm.GetVirtualMachine("evmzero")
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
