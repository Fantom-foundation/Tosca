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
	"testing"

	"github.com/Fantom-foundation/Tosca/go/ct/common"
	"pgregory.net/rand"
)

func TestTransient_UnconstrainedGeneratorCanProduceTransientStorage(t *testing.T) {
	rnd := rand.New(0)
	generator := NewTransientStorageGenerator()
	assignment := Assignment{}
	transient, err := generator.Generate(assignment, rnd)
	if err != nil {
		t.Errorf("Unexpected error during generation: %v", err)
	}
	if transient == nil {
		t.Errorf("Expected transient storage to be generated, but got nil")
	}
	if transient.IsAllZero() {
		t.Errorf("Expected transient storage to be non-empty, but got empty")
	}
}

func TestTransient_ConstraintIsEnforced(t *testing.T) {
	v1 := Variable("v1")
	tests := []struct {
		bind   func(*TransientStorageGenerator)
		isZero bool
	}{
		{
			bind: func(t *TransientStorageGenerator) {
				t.BindToNonZero(v1)
			},
			isZero: false,
		},
		{
			bind: func(t *TransientStorageGenerator) {
				t.BindToZero(v1)
			},
			isZero: true,
		},
	}

	for _, test := range tests {
		assignment := Assignment{}
		assignment[v1] = common.NewU256(42)

		rnd := rand.New(0)
		generator := NewTransientStorageGenerator()
		test.bind(generator)

		transient, err := generator.Generate(assignment, rnd)
		if err != nil {
			t.Errorf("Expected error during generation, %v", err)
		}
		if transient.IsZero(common.NewU256(42)) != test.isZero {
			t.Errorf("Constraint was not enforced")
		}
	}
}

func TestTransient_ConflictingSetUnsetConstraintsAreDetected(t *testing.T) {
	v1 := Variable("v1")
	assignment := Assignment{}
	assignment[v1] = common.NewU256(42)

	rnd := rand.New(0)
	generator := NewTransientStorageGenerator()
	generator.BindToNonZero(v1)
	generator.BindToZero(v1)

	_, err := generator.Generate(assignment, rnd)
	if err == nil {
		t.Errorf("Expected error during generation, but got nil")
	}
}

func TestTransient_ConstraintsOnTheSameKey(t *testing.T) {
	v1 := Variable("v1")
	v2 := Variable("v2")
	tests := []struct {
		change  func(*TransientStorageGenerator)
		success bool
	}{
		{
			change: func(t *TransientStorageGenerator) {
				t.BindToNonZero(v1)
				t.BindToNonZero(v2)
			},
			success: true,
		},
		{
			change: func(t *TransientStorageGenerator) {
				t.BindToNonZero(v1)
				t.BindToZero(v2)
			},
			success: false,
		},
	}

	for _, test := range tests {
		assignment := Assignment{}
		assignment[v1] = common.NewU256(42)
		assignment[v2] = common.NewU256(42)

		rnd := rand.New(0)
		generator := NewTransientStorageGenerator()

		test.change(generator)

		_, err := generator.Generate(assignment, rnd)
		if (err == nil) != test.success {
			t.Errorf("Constraints on the same key were not enforced correctly")
		}
	}
}

func TestTransient_UnboundVariables(t *testing.T) {
	v1 := Variable("v1")
	v2 := Variable("v2")
	assignment := Assignment{}
	rnd := rand.New(0)

	generator := NewTransientStorageGenerator()
	generator.BindToNonZero(v1)
	generator.BindToZero(v2)
	transientStorage, err := generator.Generate(assignment, rnd)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	addressV1 := assignment[v1]
	if addressV1.IsZero() {
		t.Errorf("assignment of v1 has not been updated during generation")
	}
	addressV2 := assignment[v2]
	if addressV2.IsZero() {
		t.Errorf("assignment of v2 has not been updated during generation")
	}

	if transientStorage.IsZero(addressV1) {
		t.Errorf("transient storage of v1 is zero")
	}
	if !transientStorage.IsZero(addressV2) {
		t.Errorf("transient storage of v2 is non zero")
	}
}

func TestTransient_MixOfBoundAndUnboundVariables(t *testing.T) {
	v1 := Variable("v1")
	v2 := Variable("v2")
	assignment := Assignment{}
	assignment[v2] = common.NewU256(42)

	rnd := rand.New(0)
	generator := NewTransientStorageGenerator()
	generator.BindToNonZero(v1)
	generator.BindToZero(v2)
	_, err := generator.Generate(assignment, rnd)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}
