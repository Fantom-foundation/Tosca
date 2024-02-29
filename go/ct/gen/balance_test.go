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
		t.Fatalf("unexpected error during build: %v", err)
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
		t.Fatal(err)
	}

	if !balance.IsWarm(NewAddressFromInt(42)) {
		t.Fail()
	}
}

func TestBalanceGenerator_ConflictingWarmColdConstraintsAreDetected(t *testing.T) {
	v1 := Variable("v1")
	assignment := Assignment{}
	assignment[v1] = NewU256(42)

	rnd := rand.New(0)
	generator := NewBalanceGenerator()
	accountAddress, err := RandAddress(rnd)
	if err != nil {
		t.Errorf("Unexpected random address generation error: %v", err)
	}

	generator.BindCold(v1)
	generator.BindWarm(v1)

	_, err = generator.Generate(assignment, rnd, accountAddress)
	if !errors.Is(err, ErrUnsatisfiable) {
		t.Fatal("Conflicting warm/cold constraints not detected")
	}
}

func TestBalanceGenerator_ClonesAreEqual(t *testing.T) {
	v1 := Variable("v1")
	v2 := Variable("v2")

	original := NewBalanceGenerator()
	original.BindConfiguration(v1, v2)
	original.BindWarm(v2)

	clone := original.Clone()

	if want, got := original.String(), clone.String(); want != got {
		t.Errorf("invalid clone, wanted %s, got %s", want, got)
	}
}

func TestBalanceGenerator_ClonesAreIndependent(t *testing.T) {
	v1 := Variable("v1")
	v2 := Variable("v2")
	v3 := Variable("v3")
	v4 := Variable("v4")

	base := NewBalanceGenerator()
	base.BindConfiguration(v1, v2)
	base.BindWarm(v2)

	clone1 := base.Clone()
	clone1.BindConfiguration(v3, v4)
	clone1.BindWarm(v4)

	clone2 := base.Clone()
	clone2.BindConfiguration(v3, v4)
	clone2.BindCold(v4)

	want := "{cfg[$v1]=$v2,cfg[$v3]=$v4,warm($v2),warm($v4)}"
	if got := clone1.String(); got != want {
		t.Errorf("invalid clone, wanted\n%s\n\tgot\n%s", want, got)
	}

	want = "{cfg[$v1]=$v2,cfg[$v3]=$v4,warm($v2),cold($v4)}"
	if got := clone2.String(); got != want {
		t.Errorf("invalid clone, wanted\n%s\n\tgot\n%s", want, got)
	}
}

func TestBalanceGenerator_ClonesCanBeUsedToResetGenerator(t *testing.T) {
	v1 := Variable("v1")
	v2 := Variable("v2")
	v3 := Variable("v3")

	generator := NewBalanceGenerator()
	generator.BindConfiguration(v1, v2)
	generator.BindWarm(v1)

	backup := generator.Clone()

	generator.BindConfiguration(v2, v3)
	generator.BindWarm(v2)

	want := "{cfg[$v1]=$v2,cfg[$v2]=$v3,warm($v1),warm($v2)}"
	if got := generator.String(); got != want {
		t.Errorf("invalid clone, wanted\n%s\ngot\n%s", want, got)
	}

	generator.Restore(backup)

	if want, got := "{cfg[$v1]=$v2,warm($v1)}", generator.String(); got != want {
		t.Errorf("invalid clone, wanted \n%s\n\tgot\n%s", want, got)
	}
}
