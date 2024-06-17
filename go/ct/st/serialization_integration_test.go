// Copyright (c) 2024 Fantom Foundation
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at fantom.foundation/bsl11.
//
// Change Date: 2028-4-16
//
// On the date above, in accordance with the Business Source License, use of
// this software will be governed by the GNU Lesser General Public License v3.

package st_test

import (
	"path/filepath"
	"testing"

	"pgregory.net/rand"

	"github.com/Fantom-foundation/Tosca/go/ct/gen"
	"github.com/Fantom-foundation/Tosca/go/ct/st"
)

func TestSerialization_EndToEndTest(t *testing.T) {
	const N = 100
	rnd := rand.New(0)
	gen := gen.NewStateGenerator()

	for i := 0; i < N; i++ {
		state, err := gen.Generate(rnd)
		if err != nil {
			t.Fatalf("failed to generate random state: %v", err)
		}

		path := filepath.Join(t.TempDir(), "state.json")
		if err := st.ExportStateJSON(state, path); err != nil {
			t.Fatalf("failed to write state to file: %v", err)
		}

		restored, err := st.ImportStateJSON(path)
		if err != nil {
			t.Fatalf("failed to read state from file: %v", err)
		}

		if !state.Eq(restored) {
			t.Errorf("failed to restore state\nwanted: %v\ngot: %v\n", state, restored)
			for _, cur := range state.Diff(restored) {
				t.Errorf("%s\n", cur)
			}
		}
	}
}

func BenchmarkSerliazation_EndToEnd(b *testing.B) {
	rnd := rand.New(0)
	gen := gen.NewStateGenerator()

	for i := 0; i < b.N; i++ {
		state, err := gen.Generate(rnd)
		if err != nil {
			b.Fatalf("failed to generate random state: %v", err)
		}

		path := filepath.Join(b.TempDir(), "state.json")
		if err := st.ExportStateJSON(state, path); err != nil {
			b.Fatalf("failed to write state to file: %v", err)
		}

		_, err = st.ImportStateJSON(path)
		if err != nil {
			b.Fatalf("failed to read state from file: %v", err)
		}
	}

}
