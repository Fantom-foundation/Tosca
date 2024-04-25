//
// Copyright (c) 2024 Fantom Foundation
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at fantom.foundation/bsl11.
//
// Change Date: 2028-4-16
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by the GNU Lesser General Public Licence v3
//

package gen

import (
	"errors"
	"testing"

	"pgregory.net/rand"
)

func TestSelfDestructedGenerator_UnconstrainedGeneratorCanGenerate(t *testing.T) {
	rnd := rand.New(0)
	generator := NewSelfDestructedGenerator()
	_, err := generator.Generate(rnd)
	if err != nil {
		t.Errorf("unexpected error during generation: %v", err)
	}
}

func TestSelfDestructedGenerator_SelfDestructedConstraintIsEnforced(t *testing.T) {
	rnd := rand.New(0)

	tests := map[string]struct {
		wantGenerated   bool
		constraintEffet func(g *SelfDestructedGenerator)
	}{
		"SelfDestruct":    {true, func(g *SelfDestructedGenerator) { g.MarkAsSelfDestructed() }},
		"NotSelfDestruct": {false, func(g *SelfDestructedGenerator) { g.MarkAsNotSelfDestructed() }},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			generator := NewSelfDestructedGenerator()
			test.constraintEffet(generator)
			hasSelfDestructed, err := generator.Generate(rnd)

			if err != nil {
				t.Errorf("Unexpected error during generation: %v", err)
			}

			if hasSelfDestructed != test.wantGenerated {
				t.Errorf("unexpected generates has-self-destructed value")
			}
		})
	}
}

func TestSelfDestructedGenerator_ConflictingSelfDestructedConstraintsAreDetected(t *testing.T) {
	rnd := rand.New(0)
	generator := NewSelfDestructedGenerator()
	generator.MarkAsSelfDestructed()
	generator.MarkAsNotSelfDestructed()

	_, err := generator.Generate(rnd)
	if !errors.Is(err, ErrUnsatisfiable) {
		t.Errorf("Conflicting has-self-destructed constraints not detected")
	}
}

func TestSelfDestructedGenerator_String(t *testing.T) {
	generator := NewSelfDestructedGenerator()
	str := generator.String()
	want := "{mustDestroy(false) mustNotDestroy(false)}"
	if str != want {
		t.Errorf("unexpected string: wanted %v, but got %v", want, str)
	}
}

func TestSelfDestructedGenerator_Restore(t *testing.T) {
	gen1 := NewSelfDestructedGenerator()
	gen2 := NewSelfDestructedGenerator()
	gen2.mustNotSelfDestructed = true
	gen2.mustSelfDestructed = true

	gen1.Restore(gen2)
	if !gen1.mustNotSelfDestructed || !gen1.mustSelfDestructed {
		t.Error("selfDestructedGenerator's restore is broken")
	}
}

func BenchmarkSelfDestructedGenWithConstraint(b *testing.B) {
	rnd := rand.New(0)
	generator := NewSelfDestructedGenerator()
	generator.MarkAsNotSelfDestructed()

	for i := 0; i < b.N; i++ {
		generator.Generate(rnd)
	}
}

func BenchmarkSelfDestructedGenWithOutConstraint(b *testing.B) {
	rnd := rand.New(0)
	generator := NewSelfDestructedGenerator()

	for i := 0; i < b.N; i++ {
		generator.Generate(rnd)
	}
}
