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
	"slices"
	"testing"

	"pgregory.net/rand"

	. "github.com/Fantom-foundation/Tosca/go/ct/common"
	"github.com/Fantom-foundation/Tosca/go/tosca"
)

func TestStorageGenerator_UnconstraintGeneratorCanProduceStorage(t *testing.T) {
	generator := NewStorageGenerator()
	if _, err := generator.Generate(nil, rand.New(0)); err != nil {
		t.Fatalf("unexpected error during build: %v", err)
	}
}

func TestStorageGenerator_StorageConfigurationsAreEnforced(t *testing.T) {
	for _, cfg := range []tosca.StorageStatus{
		tosca.StorageAssigned,
		tosca.StorageAdded,
		tosca.StorageAddedDeleted,
		tosca.StorageDeletedRestored,
		tosca.StorageDeletedAdded,
		tosca.StorageDeleted,
		tosca.StorageModified,
		tosca.StorageModifiedDeleted,
		tosca.StorageModifiedRestored,
	} {
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
		case tosca.StorageAdded:
			fail = !org.IsZero() || !cur.IsZero() || new.IsZero()
		case tosca.StorageAddedDeleted:
			fail = !org.IsZero() || cur.IsZero() || !new.IsZero()
		case tosca.StorageDeletedRestored:
			fail = org.IsZero() || !cur.IsZero() || !org.Eq(new)
		case tosca.StorageDeletedAdded:
			fail = org.IsZero() || !cur.IsZero() || new.IsZero() || org.Eq(new)
		case tosca.StorageDeleted:
			fail = org.IsZero() || !cur.Eq(org) || !new.IsZero()
		case tosca.StorageModified:
			fail = org.IsZero() || !cur.Eq(org) || new.IsZero() || new.Eq(cur)
		case tosca.StorageModifiedDeleted:
			fail = org.IsZero() || cur.IsZero() || org.Eq(cur) || !new.IsZero()
		case tosca.StorageModifiedRestored:
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
	generator.BindConfiguration(tosca.StorageDeleted, v1, v2)
	generator.BindConfiguration(tosca.StorageAdded, v1, v2)

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
	generator.BindConfiguration(tosca.StorageModified, v1, v2)
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
	original.BindConfiguration(tosca.StorageModified, v1, v2)
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
	base.BindConfiguration(tosca.StorageAdded, v1, v2)
	base.BindWarm(v2)

	clone1 := base.Clone()
	clone1.BindConfiguration(tosca.StorageDeleted, v3, v4)
	clone1.BindWarm(v4)

	clone2 := base.Clone()
	clone2.BindConfiguration(tosca.StorageModifiedRestored, v3, v4)
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
	generator.BindConfiguration(tosca.StorageDeleted, v1, v2)
	generator.BindWarm(v1)

	backup := generator.Clone()

	generator.BindConfiguration(tosca.StorageModified, v2, v3)
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

func TestStorageGenerator_StorageAssignedCanBeSatisfied(t *testing.T) {
	key := Variable("key")
	newValue := Variable("newValue")
	generator := NewStorageGenerator()
	generator.BindConfiguration(tosca.StorageAssigned, key, newValue)

	assignment := Assignment{}

	for i := 0; i < 100; i++ {

		storage, err := generator.Generate(assignment, rand.New(0))
		if err != nil {
			t.Fatal(err)
		}
		storageOriginal := storage.GetOriginal(assignment[key])
		storageCurrent := storage.GetCurrent(assignment[key])
		storageNew := assignment[newValue]

		got := tosca.GetStorageStatus(tosca.Word(storageOriginal.Bytes32be()), tosca.Word(storageCurrent.Bytes32be()), tosca.Word(storageNew.Bytes32be()))
		want := tosca.StorageAssigned

		if got != want {
			t.Fatalf("unexpected storage status, got original: %v, current: %v, new: %v", storageOriginal, storageCurrent, storageNew)
		}
	}
}

func TestStorageGenerator_CheckStorageStatusConfig(t *testing.T) {

	tests := map[string]struct {
		original U256
		current  U256
		new      U256
		want     tosca.StorageStatus
	}{
		"StorageAssigned": {
			original: NewU256(0),
			current:  NewU256(0),
			new:      NewU256(0),
			want:     tosca.StorageAssigned,
		},
		"StorageAdded": {
			original: NewU256(0),
			current:  NewU256(0),
			new:      NewU256(1),
			want:     tosca.StorageAdded,
		},
		"StorageDeleted": {
			original: NewU256(1),
			current:  NewU256(1),
			new:      NewU256(0),
			want:     tosca.StorageDeleted,
		},
		"StorageModified": {
			original: NewU256(1),
			current:  NewU256(1),
			new:      NewU256(2),
			want:     tosca.StorageModified,
		},
		"StorageDeletedAdded": {
			original: NewU256(1),
			current:  NewU256(0),
			new:      NewU256(2),
			want:     tosca.StorageDeletedAdded,
		},
		"StorageModifiedDeleted": {
			original: NewU256(1),
			current:  NewU256(2),
			new:      NewU256(0),
			want:     tosca.StorageModifiedDeleted,
		},
		"StorageDeletedRestored": {
			original: NewU256(1),
			current:  NewU256(0),
			new:      NewU256(1),
			want:     tosca.StorageDeletedRestored,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			if !CheckStorageStatusConfig(test.want, test.original, test.current, test.new) {
				t.Fatalf("unexpected storage status, got original: %v, current: %v, new: %v. but wanted: %v",
					test.original, test.current, test.new, test.want)
			}
		})
	}
}

func TestStorageGenerator_NewValueMustBeZeroMustNotBeZero(t *testing.T) {

	allConfigs := []tosca.StorageStatus{
		tosca.StorageAssigned,
		tosca.StorageAdded,
		tosca.StorageDeleted,
		tosca.StorageModified,
		tosca.StorageDeletedAdded,
		tosca.StorageModifiedDeleted,
		tosca.StorageDeletedRestored,
		tosca.StorageAddedDeleted,
		tosca.StorageModifiedRestored,
	}
	mustBeZeroConfigs := []tosca.StorageStatus{
		tosca.StorageAddedDeleted,
		tosca.StorageDeleted,
		tosca.StorageModifiedDeleted,
	}
	mustNotBeZeroConfigs := []tosca.StorageStatus{
		tosca.StorageAssigned,
		tosca.StorageAdded,
		tosca.StorageDeletedRestored,
		tosca.StorageDeletedAdded,
		tosca.StorageModified,
		tosca.StorageModifiedRestored,
	}

	for _, cfg := range allConfigs {
		t.Run(cfg.String(), func(t *testing.T) {
			if NewValueMustBeZero(cfg) != slices.Contains(mustBeZeroConfigs, cfg) {
				t.Fatalf("unexpected result for NewValueMustBeZero(%v)", cfg)
			}
			if NewValueMustNotBeZero(cfg) != slices.Contains(mustNotBeZeroConfigs, cfg) {
				t.Fatalf("unexpected result for NewValueMustNotBeZero(%v)", cfg)
			}
		})
	}
}

func TestStorageGenerator_UnspecifiedVariableIsNotWarm(t *testing.T) {

	vKey := Variable("key")
	vNewValue := Variable("newValue")

	generator := NewStorageGenerator()
	generator.BindConfiguration(tosca.StorageModified, vKey, vNewValue)

	assignment := Assignment{}
	rnd := rand.New(0)

	for i := 0; i < 1000; i++ {
		storage, err := generator.Generate(assignment, rnd)
		if err != nil {
			t.Fatal(err)
		}

		if storage.IsWarm(assignment[vKey]) {
			t.Fatalf("unexpected variable %v is warm at iteration %v", vKey, i)
		}
	}

}
