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

	. "github.com/Fantom-foundation/Tosca/go/ct/common"
	"pgregory.net/rand"
)

func TestSelfDestructedGenerator_UnconstrainedGeneratorCanProduceBalance(t *testing.T) {
	rnd := rand.New(0)
	generator := NewSelfDestructedGenerator()
	if _, err := generator.Generate(Assignment{}, rnd); err != nil {
		t.Errorf("Unexpected error during generation: %v", err)
	}
}

func TestSelfDestructedGenerator_DestructedConstraintIsEnforced(t *testing.T) {
	v1 := Variable("v1")
	assignment := Assignment{}
	assignment[v1] = NewU256(42)

	rnd := rand.New(0)
	generator := NewSelfDestructedGenerator()
	generator.BindHasSelfDestructed(v1)
	hasSelfDestructed, err := generator.Generate(assignment, rnd)
	if err != nil {
		t.Errorf("Unexpected error during generation: %v", err)
	}

	if _, exist := hasSelfDestructed[NewAddressFromInt(42)]; !exist {
		t.Errorf("Expected constraint address to be self desctructed, but generated state marked as not")
	}
}

func TestSelfDestructedGenerator_ConflictingSelfDestructedConstraintsAreDetected(t *testing.T) {
	v1 := Variable("v1")
	assignment := Assignment{}
	assignment[v1] = NewU256(42)

	rnd := rand.New(0)
	generator := NewSelfDestructedGenerator()

	generator.BindHasSelfDestructed(v1)
	generator.BindHasNotSelfDestructed(v1)

	_, err := generator.Generate(assignment, rnd)
	if !errors.Is(err, ErrUnsatisfiable) {
		t.Errorf("Conflicting warm/cold constraints not detected")
	}
}

func TestSelfDestructedGenerator_SelfDestructedConstraintsNoAssignment(t *testing.T) {
	v1 := Variable("v1")
	v2 := Variable("v2")
	assignment := Assignment{}
	rnd := rand.New(0)
	generator := NewSelfDestructedGenerator()

	generator.BindHasSelfDestructed(v1)
	generator.BindHasNotSelfDestructed(v2)

	hasSelfDestructed, err := generator.Generate(assignment, rnd)
	if err != nil {
		t.Fatalf("Unexpected error during balance generation")
	}

	pos1, found1 := assignment[v1]
	pos2, found2 := assignment[v2]

	if !found1 || !found2 {
		t.Fatalf("Variable not bound by generator")
	}
	if _, exist := hasSelfDestructed[NewAddress(pos1)]; !exist {
		t.Errorf("Expected address to be warm but got cold")
	}
	if _, exist := hasSelfDestructed[NewAddress(pos2)]; exist {
		t.Errorf("Expected address to be cold but got warm")
	}
}

func BenchmarkSelfDestructedGenWithConstraint(b *testing.B) {
	v1 := Variable("v1")
	v2 := Variable("v2")
	assignment := Assignment{}
	rnd := rand.New(0)
	generator := NewSelfDestructedGenerator()

	generator.BindHasSelfDestructed(v1)
	generator.BindHasNotSelfDestructed(v2)

	for i := 0; i < b.N; i++ {
		generator.Generate(assignment, rnd)
	}
}

func BenchmarkSelfDestructedGenWithOutConstraint(b *testing.B) {
	assignment := Assignment{}
	rnd := rand.New(0)
	generator := NewSelfDestructedGenerator()

	for i := 0; i < b.N; i++ {
		generator.Generate(assignment, rnd)
	}
}
