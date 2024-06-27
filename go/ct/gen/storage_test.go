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
	"github.com/Fantom-foundation/Tosca/go/vm"
)

func TestStorageGenerator_UnconstraintGeneratorCanProduceStorage(t *testing.T) {
	generator := NewStorageGenerator()
	if _, err := generator.Generate(nil, rand.New(0)); err != nil {
		t.Fatalf("unexpected error during build: %v", err)
	}
}

func TestStorageGenerator_StorageConfigurationsAreEnforced(t *testing.T) {
	for _, cfg := range []vm.StorageStatus{
		vm.StorageAssigned,
		vm.StorageAdded,
		vm.StorageAddedDeleted,
		vm.StorageDeletedRestored,
		vm.StorageDeletedAdded,
		vm.StorageDeleted,
		vm.StorageModified,
		vm.StorageModifiedDeleted,
		vm.StorageModifiedRestored,
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
		case vm.StorageAdded:
			fail = !org.IsZero() || !cur.IsZero() || new.IsZero()
		case vm.StorageAddedDeleted:
			fail = !org.IsZero() || cur.IsZero() || !new.IsZero()
		case vm.StorageDeletedRestored:
			fail = org.IsZero() || !cur.IsZero() || !org.Eq(new)
		case vm.StorageDeletedAdded:
			fail = org.IsZero() || !cur.IsZero() || new.IsZero() || org.Eq(new)
		case vm.StorageDeleted:
			fail = org.IsZero() || !cur.Eq(org) || !new.IsZero()
		case vm.StorageModified:
			fail = org.IsZero() || !cur.Eq(org) || new.IsZero() || new.Eq(cur)
		case vm.StorageModifiedDeleted:
			fail = org.IsZero() || cur.IsZero() || org.Eq(cur) || !new.IsZero()
		case vm.StorageModifiedRestored:
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
	generator.BindConfiguration(vm.StorageDeleted, v1, v2)
	generator.BindConfiguration(vm.StorageAdded, v1, v2)

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
	generator.BindConfiguration(vm.StorageModified, v1, v2)
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
	original.BindConfiguration(vm.StorageModified, v1, v2)
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
	base.BindConfiguration(vm.StorageAdded, v1, v2)
	base.BindWarm(v2)

	clone1 := base.Clone()
	clone1.BindConfiguration(vm.StorageDeleted, v3, v4)
	clone1.BindWarm(v4)

	clone2 := base.Clone()
	clone2.BindConfiguration(vm.StorageModifiedRestored, v3, v4)
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
	generator.BindConfiguration(vm.StorageDeleted, v1, v2)
	generator.BindWarm(v1)

	backup := generator.Clone()

	generator.BindConfiguration(vm.StorageModified, v2, v3)
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
	generator.BindConfiguration(vm.StorageAssigned, key, newValue)

	assignment := Assignment{}

	for i := 0; i < 100; i++ {

		storage, err := generator.Generate(assignment, rand.New(0))
		if err != nil {
			t.Fatal(err)
		}
		storageOriginal := storage.GetOriginal(assignment[key])
		storageCurrent := storage.GetCurrent(assignment[key])
		storageNew := assignment[newValue]

		got := vm.GetStorageStatus(vm.Word(storageOriginal.Bytes32be()), vm.Word(storageCurrent.Bytes32be()), vm.Word(storageNew.Bytes32be()))
		want := vm.StorageAssigned

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
		want     vm.StorageStatus
	}{
		"StorageAssigned": {
			original: NewU256(0),
			current:  NewU256(0),
			new:      NewU256(0),
			want:     vm.StorageAssigned,
		},
		"StorageAdded": {
			original: NewU256(0),
			current:  NewU256(0),
			new:      NewU256(1),
			want:     vm.StorageAdded,
		},
		"StorageDeleted": {
			original: NewU256(1),
			current:  NewU256(1),
			new:      NewU256(0),
			want:     vm.StorageDeleted,
		},
		"StorageModified": {
			original: NewU256(1),
			current:  NewU256(1),
			new:      NewU256(2),
			want:     vm.StorageModified,
		},
		"StorageDeletedAdded": {
			original: NewU256(1),
			current:  NewU256(0),
			new:      NewU256(2),
			want:     vm.StorageDeletedAdded,
		},
		"StorageModifiedDeleted": {
			original: NewU256(1),
			current:  NewU256(2),
			new:      NewU256(0),
			want:     vm.StorageModifiedDeleted,
		},
		"StorageDeletedRestored": {
			original: NewU256(1),
			current:  NewU256(0),
			new:      NewU256(1),
			want:     vm.StorageDeletedRestored,
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

	allConfigs := []vm.StorageStatus{
		vm.StorageAssigned,
		vm.StorageAdded,
		vm.StorageDeleted,
		vm.StorageModified,
		vm.StorageDeletedAdded,
		vm.StorageModifiedDeleted,
		vm.StorageDeletedRestored,
		vm.StorageAddedDeleted,
		vm.StorageModifiedRestored,
	}
	mustBeZeroConfigs := []vm.StorageStatus{
		vm.StorageAddedDeleted,
		vm.StorageDeleted,
		vm.StorageModifiedDeleted,
	}
	mustNotBeZeroConfigs := []vm.StorageStatus{
		vm.StorageAssigned,
		vm.StorageAdded,
		vm.StorageDeletedRestored,
		vm.StorageDeletedAdded,
		vm.StorageModified,
		vm.StorageModifiedRestored,
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
