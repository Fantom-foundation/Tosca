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
	tests := map[struct {
		mustDestruct    bool
		mustNotDestruct bool
	}]string{
		{true, true}:   "{false}",
		{true, false}:  "{mustBeSelfDestructed}",
		{false, true}:  "{mustNotBeSelfDestructed}",
		{false, false}: "{true}",
	}
	for values, want := range tests {
		generator := NewSelfDestructedGenerator()
		if values.mustDestruct {
			generator.MarkAsSelfDestructed()
		}
		if values.mustNotDestruct {
			generator.MarkAsNotSelfDestructed()
		}
		str := generator.String()
		if str != want {
			t.Errorf("unexpected string: wanted %v, but got %v", want, str)
		}
	}
}

func TestSelfDestructedGenerator_Restore(t *testing.T) {
	gen1 := NewSelfDestructedGenerator()
	gen2 := NewSelfDestructedGenerator()
	gen2.mustNotBeSelfDestructed = true
	gen2.mustBeSelfDestructed = true

	gen1.Restore(gen2)
	if !gen1.mustNotBeSelfDestructed || !gen1.mustBeSelfDestructed {
		t.Error("selfDestructedGenerator's restore is broken")
	}
}
