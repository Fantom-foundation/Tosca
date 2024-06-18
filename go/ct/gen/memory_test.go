// Copyright (c) 2024 Fantom Foundation
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at fantom.foundation/bsl11.
//
// Change Date: 2028-4-16
//
// On the date above, in accordance with the Business Source License, use of
// this software will be governed by the GNU Lesser General Public License v3.

package gen

import (
	"testing"

	"pgregory.net/rand"
)

func TestMemoryGenerator_UnconstrainedGeneratorCanProduceMemory(t *testing.T) {
	rnd := rand.New(0)
	generator := NewMemoryGenerator()
	if _, err := generator.Generate(rnd); err != nil {
		t.Fatalf("unexpected error during build: %v", err)
	}
}

func TestMemoryGenerator_SizeIsMultipleOf32(t *testing.T) {
	rnd := rand.New(0)
	generator := NewMemoryGenerator()
	for i := 0; i < 10; i++ {
		memory, err := generator.Generate(rnd)
		if err != nil {
			t.Fatalf("unexpected error during build: %v", err)
		}
		if memory.Size()%32 != 0 {
			t.Fatalf("memory size is not a multiple of 32, got %v", memory.Size())
		}
	}
}
