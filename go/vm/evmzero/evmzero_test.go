//
// Copyright (c) 2024 Fantom Foundation
//
// Use of this software is governed by the Business Source License included
// in the LICENSE.TXT file and at fantom.foundation/bsl11.
//
// Change Date: 2028-4-16
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by the GNU Lesser General Public Licence v3
//

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

	interpreter := vm.GetInterpreter("evmzero")
	got, err := example.RunOn(interpreter, arg)
	if err != nil {
		t.Fatalf("running the fib example failed: %v", err)
	}

	if got.Result != wanted {
		t.Fatalf("unexpected result, wanted %v, got %v", wanted, got.Result)
	}
}

func TestEvmzero_DumpProfile(t *testing.T) {
	example := examples.GetFibExample()
	interpreter, ok := vm.GetInterpreter("evmzero-profiling").(vm.ProfilingInterpreter)
	if !ok || interpreter == nil {
		t.Fatalf("profiling evmzero configuration does not support profiling")
	}
	for i := 0; i < 10; i++ {
		example.RunOn(interpreter, 10)
		interpreter.DumpProfile()
		if i == 5 {
			interpreter.ResetProfile()
		}
	}
}

func BenchmarkNewEvmcInterpreter(b *testing.B) {
	b.Run("evmzero", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			vm.GetInterpreter("evmzero")
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
		interpreter := vm.GetInterpreter("evmzero")
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
