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

	. "github.com/Fantom-foundation/Tosca/go/ct/common"
	"pgregory.net/rand"
)

func TestAccountsGenerator_UnconstrainedGeneratorCanProduceBalance(t *testing.T) {
	rnd := rand.New(0)
	generator := NewAccountGenerator()
	accountAddress := RandomAddress(rnd)
	if _, err := generator.Generate(nil, rnd, accountAddress); err != nil {
		t.Errorf("Unexpected error during generation: %v", err)
	}
}

func TestAccountsGenerator_WarmConstraintIsEnforced(t *testing.T) {
	v1 := Variable("v1")
	assignment := Assignment{}
	assignment[v1] = NewU256(42)

	rnd := rand.New(0)
	generator := NewAccountGenerator()
	accountAddress := RandomAddress(rnd)
	generator.BindWarm(v1)
	accounts, err := generator.Generate(assignment, rnd, accountAddress)
	if err != nil {
		t.Errorf("Unexpected error during generation: %v", err)
	}

	if !accounts.IsWarm(NewAddressFromInt(42)) {
		t.Errorf("Expected constraint address to be warm, but generated state marked as cold")
	}
}

func TestAccountsGenerator_ConflictingWarmColdConstraintsAreDetected(t *testing.T) {
	v1 := Variable("v1")
	assignment := Assignment{}
	assignment[v1] = NewU256(42)

	rnd := rand.New(0)
	generator := NewAccountGenerator()
	accountAddress := NewAddress(NewU256(42))

	generator.BindCold(v1)
	generator.BindWarm(v1)

	_, err := generator.Generate(assignment, rnd, accountAddress)
	if !errors.Is(err, ErrUnsatisfiable) {
		t.Errorf("Conflicting warm/cold constraints not detected")
	}
}

func TestAccountsGenerator_WarmColdConstraintsNoAssignment(t *testing.T) {
	v1 := Variable("v1")
	v2 := Variable("v2")
	assignment := Assignment{}
	rnd := rand.New(0)
	generator := NewAccountGenerator()

	generator.BindWarm(v1)
	generator.BindCold(v2)

	accounts, err := generator.Generate(assignment, rnd, NewAddressFromInt(8))
	if err != nil {
		t.Fatalf("Unexpected error during balance generation")
	}

	pos1, found1 := assignment[v1]
	pos2, found2 := assignment[v2]

	if !found1 || !found2 {
		t.Fatalf("Variable not bound by generator")
	}
	if !accounts.IsWarm(NewAddress(pos1)) {
		t.Errorf("Expected address to be warm but got cold")
	}
	if accounts.IsWarm(NewAddress(pos2)) {
		t.Errorf("Expected address to be cold but got warm")
	}
}

func BenchmarkAccountGenWithConstraint(b *testing.B) {
	v1 := Variable("v1")
	v2 := Variable("v2")
	assignment := Assignment{}
	rnd := rand.New(0)
	generator := NewAccountGenerator()

	generator.BindWarm(v1)
	generator.BindCold(v2)
	for i := 0; i < b.N; i++ {
		_, err := generator.Generate(assignment, rnd, NewAddressFromInt(8))
		if err != nil {
			b.Fatalf("Invalid benchmark, Unexpected error during balance generation %v", err)
		}
	}
}

func BenchmarkAccountGenWithOutConstraint(b *testing.B) {
	assignment := Assignment{}
	rnd := rand.New(0)
	generator := NewAccountGenerator()

	for i := 0; i < b.N; i++ {
		_, err := generator.Generate(assignment, rnd, NewAddressFromInt(8))
		if err != nil {
			b.Fatalf("Invalid benchmark, Unexpected error during balance generation %v", err)
		}
	}
}
