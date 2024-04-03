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

func TestStorageGenerator_StorageConfigurationsAreEnforced(t *testing.T) {
	for _, cfg := range []StorageCfg{StorageAdded, StorageAddedDeleted, StorageDeletedRestored, StorageDeletedAdded, StorageDeleted, StorageModified, StorageModifiedDeleted, StorageModifiedRestored} {
		vKey := Variable("key")
		vNewValue := Variable("newValue")

		assignment := Assignment{}

		generator := NewStorageGenerator()
		generator.BindConfiguration(cfg, vKey, vNewValue)
		storage, err := generator.Generate(assignment, rand.New(0))
		if err != nil {
			t.Fatal(err)
		}

		org := storage.GetOriginal(assignment[vKey])
		cur := storage.GetCurrent(assignment[vKey])
		new := assignment[vNewValue]

		fail := false
		switch cfg {
		case StorageAdded:
			fail = !org.IsZero() || !cur.IsZero() || new.IsZero()
		case StorageAddedDeleted:
			fail = !org.IsZero() || cur.IsZero() || !new.IsZero()
		case StorageDeletedRestored:
			fail = org.IsZero() || !cur.IsZero() || !org.Eq(new)
		case StorageDeletedAdded:
			fail = org.IsZero() || !cur.IsZero() || new.IsZero() || org.Eq(new)
		case StorageDeleted:
			fail = org.IsZero() || !cur.Eq(org) || !new.IsZero()
		case StorageModified:
			fail = org.IsZero() || !cur.Eq(org) || new.IsZero() || new.Eq(cur)
		case StorageModifiedDeleted:
			fail = org.IsZero() || cur.IsZero() || org.Eq(cur) || !new.IsZero()
		case StorageModifiedRestored:
			fail = org.IsZero() || cur.IsZero() || org.Eq(cur) || !org.Eq(new)
		}

		if fail {
			t.Fatalf("%v failed:\norg: %v\ncur: %v\nnew: %v", cfg, org, cur, new)
		}
	}
}

func TestStorageGenerator_ConflictingStorageConfigurationsAreDetected(t *testing.T) {
	v1 := Variable("v1")
	v2 := Variable("v2")

	generator := NewStorageGenerator()
	generator.BindConfiguration(StorageDeleted, v1, v2)
	generator.BindConfiguration(StorageAdded, v1, v2)

	_, err := generator.Generate(nil, rand.New(0))
	if !errors.Is(err, ErrUnsatisfiable) {
		t.Fatal("Conflicting value constraints for current not detected")
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
	v2 := Variable("v2")
	assignment := Assignment{}

	generator := NewStorageGenerator()
	generator.BindConfiguration(StorageModified, v1, v2)
	_, err := generator.Generate(assignment, rand.New(0))
	if err != nil {
		t.Fatal(err)
	}

	if _, isAssigned := assignment[v1]; !isAssigned {
		t.Fatal("v1 not assigned by original value constraint")
	}
	if _, isAssigned := assignment[v1]; !isAssigned {
		t.Fatal("v2 not assigned by current value constraint")
	}
}

func TestStorageGenerator_ClonesAreEqual(t *testing.T) {
	v1 := Variable("v1")
	v2 := Variable("v2")

	original := NewStorageGenerator()
	original.BindConfiguration(StorageModified, v1, v2)
	original.BindWarm(v2)

	clone := original.Clone()

	if want, got := original.String(), clone.String(); want != got {
		t.Errorf("invalid clone, wanted %s, got %s", want, got)
	}
}

func TestStorageGenerator_ClonesAreIndependent(t *testing.T) {
	v1 := Variable("v1")
	v2 := Variable("v2")
	v3 := Variable("v3")
	v4 := Variable("v4")

	base := NewStorageGenerator()
	base.BindConfiguration(StorageAdded, v1, v2)
	base.BindWarm(v2)

	clone1 := base.Clone()
	clone1.BindConfiguration(StorageDeleted, v3, v4)
	clone1.BindWarm(v4)

	clone2 := base.Clone()
	clone2.BindConfiguration(StorageModifiedRestored, v3, v4)
	clone2.BindCold(v4)

	want := "{cfg[$v1]=(StorageAdded,$v2),cfg[$v3]=(StorageDeleted,$v4),warm($v2),warm($v4)}"
	if got := clone1.String(); got != want {
		t.Errorf("invalid clone, wanted\n%s\n\tgot\n%s", want, got)
	}

	want = "{cfg[$v1]=(StorageAdded,$v2),cfg[$v3]=(StorageModifiedRestored,$v4),warm($v2),cold($v4)}"
	if got := clone2.String(); got != want {
		t.Errorf("invalid clone, wanted\n%s\n\tgot\n%s", want, got)
	}
}

func TestStorageGenerator_ClonesCanBeUsedToResetGenerator(t *testing.T) {
	v1 := Variable("v1")
	v2 := Variable("v2")
	v3 := Variable("v3")

	generator := NewStorageGenerator()
	generator.BindConfiguration(StorageDeleted, v1, v2)
	generator.BindWarm(v1)

	backup := generator.Clone()

	generator.BindConfiguration(StorageModified, v2, v3)
	generator.BindWarm(v2)

	want := "{cfg[$v1]=(StorageDeleted,$v2),cfg[$v2]=(StorageModified,$v3),warm($v1),warm($v2)}"
	if got := generator.String(); got != want {
		t.Errorf("invalid clone, wanted\n%s\ngot\n%s", want, got)
	}

	generator.Restore(backup)

	if want, got := "{cfg[$v1]=(StorageDeleted,$v2),warm($v1)}", generator.String(); got != want {
		t.Errorf("invalid clone, wanted \n%s\n\tgot\n%s", want, got)
	}
}
