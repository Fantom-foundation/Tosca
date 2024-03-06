package gen

import (
	"errors"
	"testing"

	. "github.com/Fantom-foundation/Tosca/go/ct/common"
	"pgregory.net/rand"
)

func TestBalanceGenerator_UnconstrainedGeneratorCanProduceBalance(t *testing.T) {
	rnd := rand.New(0)
	generator := NewBalanceGenerator()
	accountAddress, err := RandAddress(rnd)
	if err != nil {
		t.Errorf("Unexpected random address generation error: %v", err)
	}
	if _, err := generator.Generate(nil, rnd, accountAddress); err != nil {
		t.Errorf("Unexpected error during generation: %v", err)
	}
}

func TestBalanceGenerator_WarmConstraintIsEnforced(t *testing.T) {
	v1 := Variable("v1")
	assignment := Assignment{}
	assignment[v1] = NewU256(42)

	rnd := rand.New(0)
	generator := NewBalanceGenerator()
	accountAddress, err := RandAddress(rnd)
	if err != nil {
		t.Errorf("Unexpected random address generation error: %v", err)
	}
	generator.BindWarm(v1)
	balance, err := generator.Generate(assignment, rnd, accountAddress)
	if err != nil {
		t.Errorf("Unexpected error during generation: %v", err)
	}

	if !balance.IsWarm(NewAddressFromInt(42)) {
		t.Errorf("Expected constraint address to be warm, but generated state marked as cold")
	}
}

func TestBalanceGenerator_ConflictingWarmColdConstraintsAreDetected(t *testing.T) {
	v1 := Variable("v1")
	assignment := Assignment{}
	assignment[v1] = NewU256(42)

	rnd := rand.New(0)
	generator := NewBalanceGenerator()
	accountAddress := NewAddress(NewU256(42))

	generator.BindCold(v1)
	generator.BindWarm(v1)

	_, err := generator.Generate(assignment, rnd, accountAddress)
	if !errors.Is(err, ErrUnsatisfiable) {
		t.Errorf("Conflicting warm/cold constraints not detected")
	}
}

func TestBalanceGenerator_WarmColdConstraintsNoAssignment(t *testing.T) {
	v1 := Variable("v1")
	v2 := Variable("v2")
	assignment := Assignment{}
	rnd := rand.New(0)
	generator := NewBalanceGenerator()

	generator.BindWarm(v1)
	generator.BindCold(v2)

	balance, err := generator.Generate(assignment, rnd, NewAddressFromInt(8))
	if err != nil {
		t.Fatalf("Unexpected error during balance generation")
	}

	pos1, found1 := assignment[v1]
	pos2, found2 := assignment[v2]

	if !found1 || !found2 {
		t.Fatalf("Variable not bound by generator")
	}
	if !balance.IsWarm(NewAddress(pos1)) {
		t.Errorf("Expected address to be warm but got cold")
	}
	if balance.IsWarm(NewAddress(pos2)) {
		t.Errorf("Expected address to be cold but got warm")
	}
}
