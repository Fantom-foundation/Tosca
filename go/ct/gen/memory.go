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
	"pgregory.net/rand"

	"github.com/Fantom-foundation/Tosca/go/ct/st"
)

type MemoryGenerator struct {
}

func NewMemoryGenerator() *MemoryGenerator {
	return &MemoryGenerator{}
}

func (g *MemoryGenerator) Generate(rnd *rand.Rand) (*st.Memory, error) {
	// Pick a size; since memory is always grown in 32 byte steps, we also
	// generate only memory segments where size is a multiple of 32.
	size := 32 * rnd.Intn(10)

	data := make([]byte, size)
	_, _ = rnd.Read(data) // rnd.Read never returns an error

	return st.NewMemory(data...), nil
}

func (g *MemoryGenerator) Clone() *MemoryGenerator {
	return &MemoryGenerator{}
}

func (*MemoryGenerator) Restore(*MemoryGenerator) {
}

func (g *MemoryGenerator) String() string {
	return "{}"
}
