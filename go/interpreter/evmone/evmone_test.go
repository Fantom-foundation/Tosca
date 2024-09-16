// Copyright (c) 2024 Fantom Foundation
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at fantom.foundation/bsl11.
//
// Change Date: 2028-4-16
//
// On the date above, in accordance with the Business Source License, use of
// this software will be governed by the GNU Lesser General Public License v3.

package evmone

import (
	"testing"

	"github.com/Fantom-foundation/Tosca/go/examples"
	"github.com/Fantom-foundation/Tosca/go/tosca"
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
			interpreter, err := tosca.NewInterpreter(variant)
			if err != nil {
				t.Fatalf("failed to load evmone interpreter: %v", err)
			}
			got, err := example.RunOn(interpreter, arg)
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
			interpreter, err := tosca.NewInterpreter(variant)
			if err != nil {
				b.Fatalf("failed to load evmone interpreter: %v", err)
			}
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
}
