package gen

import (
	"errors"
	"testing"

	"pgregory.net/rand"

	. "github.com/Fantom-foundation/Tosca/go/ct/common"
)

func TestStorageGenerator_UnconstraintGeneratorCanProduceStorage(t *testing.T) {
	generator := NewStorageGenerator()
	if _, err := generator.Generate(nil, rand.New(0)); err != nil {
		t.Fatalf("unexpected error during build: %v", err)
	}
}

func TestStorageGenerator_CurrentStorageValueIsEnforced(t *testing.T) {
	v1 := Variable("v1")
	assignment := Assignment{}
	assignment[v1] = NewU256(42)

	generator := NewStorageGenerator()
	generator.BindCurrent(v1, NewU256(15))
	storage, err := generator.Generate(assignment, rand.New(0))
	if err != nil {
		t.Fatal(err)
	}

	if storage.Current[NewU256(42)] != NewU256(15) {
		t.Fail()
	}
}

func TestStorageGenerator_ConflictingValueConstraintsAreDetected(t *testing.T) {
	v1 := Variable("v1")
	assignment := Assignment{}
	assignment[v1] = NewU256(42)

	generator := NewStorageGenerator()
	generator.BindCurrent(v1, NewU256(1))
	generator.BindCurrent(v1, NewU256(2))

	_, err := generator.Generate(assignment, rand.New(0))
	if !errors.Is(err, ErrUnsatisfiable) {
		t.Fatal("Conflicting warm/cold constraints not detected")
	}
}

func TestStorageGenerator_WarmConstraintIsEnforced(t *testing.T) {
	v1 := Variable("v1")
	assignment := Assignment{}
	assignment[v1] = NewU256(42)

	generator := NewStorageGenerator()
	generator.BindWarm(v1)
	storage, err := generator.Generate(assignment, rand.New(0))
	if err != nil {
		t.Fatal(err)
	}

	if !storage.IsWarm(NewU256(42)) {
		t.Fail()
	}
}

func TestStorageGenerator_ConflictingWarmColdConstraintsAreDetected(t *testing.T) {
	v1 := Variable("v1")
	assignment := Assignment{}
	assignment[v1] = NewU256(42)

	generator := NewStorageGenerator()
	generator.BindCold(v1)
	generator.BindWarm(v1)

	_, err := generator.Generate(assignment, rand.New(0))
	if !errors.Is(err, ErrUnsatisfiable) {
		t.Fatal("Conflicting warm/cold constraints not detected")
	}
}

func TestStorageGenerator_AssignmentIsUpdated(t *testing.T) {
	v1 := Variable("v1")
	assignment := Assignment{}

	generator := NewStorageGenerator()
	generator.BindCurrent(v1, NewU256(1))
	_, err := generator.Generate(assignment, rand.New(0))
	if err != nil {
		t.Fatal(err)
	}

	if _, isAssigned := assignment[v1]; !isAssigned {
		t.Fail()
	}
}

func TestStorageGenerator_ClonesAreEqual(t *testing.T) {
	v1 := Variable("v1")

	original := NewStorageGenerator()
	original.BindCurrent(v1, NewU256(1))
	original.BindWarm(v1)

	clone := original.Clone()

	if want, got := original.String(), clone.String(); want != got {
		t.Errorf("invalid clone, wanted %s, got %s", want, got)
	}
}

func TestStorageGenerator_ClonesAreIndependent(t *testing.T) {
	v1 := Variable("v1")
	v2 := Variable("v2")

	base := NewStorageGenerator()
	base.BindCurrent(v1, NewU256(1))
	base.BindWarm(v1)

	clone1 := base.Clone()
	clone1.BindCurrent(v2, NewU256(2))
	clone1.BindWarm(v2)

	clone2 := base.Clone()
	clone2.BindCurrent(v2, NewU256(3))
	clone2.BindCold(v2)

	want := "{[$v1]=0000000000000000 0000000000000000 0000000000000000 0000000000000001,[$v2]=0000000000000000 0000000000000000 0000000000000000 0000000000000002,warm($v1),warm($v2)}"
	if got := clone1.String(); got != want {
		t.Errorf("invalid clone, wanted %s, got %s", want, got)
	}

	want = "{[$v1]=0000000000000000 0000000000000000 0000000000000000 0000000000000001,[$v2]=0000000000000000 0000000000000000 0000000000000000 0000000000000003,warm($v1),cold($v2)}"
	if got := clone2.String(); got != want {
		t.Errorf("invalid clone, wanted %s, got %s", want, got)
	}
}

func TestStorageGenerator_ClonesCanBeUsedToResetGenerator(t *testing.T) {
	v1 := Variable("v1")
	v2 := Variable("v2")

	generator := NewStorageGenerator()
	generator.BindCurrent(v1, NewU256(1))
	generator.BindWarm(v1)

	backup := generator.Clone()

	generator.BindCurrent(v2, NewU256(2))
	generator.BindWarm(v2)

	want := "{[$v1]=0000000000000000 0000000000000000 0000000000000000 0000000000000001,[$v2]=0000000000000000 0000000000000000 0000000000000000 0000000000000002,warm($v1),warm($v2)}"
	if got := generator.String(); got != want {
		t.Errorf("invalid clone, wanted %s, got %s", want, got)
	}

	generator.Restore(backup)

	want = "{[$v1]=0000000000000000 0000000000000000 0000000000000000 0000000000000001,warm($v1)}"
	if got := generator.String(); got != want {
		t.Errorf("invalid clone, wanted %s, got %s", want, got)
	}
}
