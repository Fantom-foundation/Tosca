// Copyright (c) 2024 Fantom Foundation
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at fantom.foundation/bsl11.
//
// Change Date: 2028-4-16
//
// On the date above, in accordance with the Business Source License, use of
// this software will be governed by the GNU Lesser General Public License v3.

package st

import (
	"reflect"
	"testing"

	. "github.com/Fantom-foundation/Tosca/go/ct/common"
)

func getTransientChanges() map[string]func(*TransientStorage) {
	return map[string]func(*TransientStorage){
		"set": func(t *TransientStorage) {
			t.Set(NewU256(1), NewU256(17))
		},
		"remove-current": func(t *TransientStorage) {
			t.Set(NewU256(42), NewU256(0))
		},
	}
}
func TestTransient_Clone(t *testing.T) {
	tests := getTransientChanges()

	t1 := &TransientStorage{}
	t1.Set(NewU256(42), NewU256(1))

	for name, change := range tests {
		t.Run(name, func(t *testing.T) {
			t2 := t1.Clone()
			if !reflect.DeepEqual(t1, t2) {
				t.Errorf("Clone is broken, want %v, got %v", t1, t2)
			}
			change(t2)
			if reflect.DeepEqual(t1, t2) {
				t.Errorf("Clone is not independent.")
			}
		})
	}
}

func TestTransient_Eq(t *testing.T) {
	tests := getTransientChanges()

	t1 := &TransientStorage{}
	t1.Set(NewU256(42), NewU256(1))

	for name, change := range tests {
		t.Run(name, func(t *testing.T) {
			t2 := t1.Clone()
			if !t1.Eq(t2) {
				t.Errorf("Equal is broken clones should be the same")
			}
			change(t2)
			if t1.Eq(t2) {
				t.Errorf("Equal is broken, changes are not detected")
			}
		})
	}
}

func TestTransient_Diff(t *testing.T) {
	tests := getTransientChanges()

	t1 := &TransientStorage{}
	t1.Set(NewU256(42), NewU256(1))

	for name, change := range tests {
		t.Run(name, func(t *testing.T) {
			t2 := t1.Clone()
			if diff := t1.Diff(t2); len(diff) != 0 {
				t.Errorf("Diff is broken, want empty, got %v", diff)
			}
			change(t2)
			if diff := t1.Diff(t2); len(diff) == 0 {
				t.Errorf("Diff is broken, changes are not detected")
			}
		})
	}
}

func TestTransient_EqAndDiffAreCompatible(t *testing.T) {
	tests := getTransientChanges()

	t1 := &TransientStorage{}
	t1.Set(NewU256(42), NewU256(1))

	for name, change := range tests {
		t.Run(name, func(t *testing.T) {
			t2 := t1.Clone()
			if t1.Eq(t2) != (len(t1.Diff(t2)) == 0) {
				t.Errorf("Equal is broken clones should be the same")
			}
			change(t2)
			if t1.Eq(t2) != (len(t1.Diff(t2)) == 0) {
				t.Errorf("Diff is broken, changes are not detected")
			}
		})
	}
}
