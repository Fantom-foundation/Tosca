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
	"testing"

	"github.com/Fantom-foundation/Tosca/go/ct/common"
	"pgregory.net/rand"
)

func TestTransient_UnconstrainedGeneratorCanProduceTransientStorage(t *testing.T) {
	rnd := rand.New(0)
	generator := NewTransientGenerator()
	assignment := Assignment{}
	transient, err := generator.Generate(assignment, rnd)
	if err != nil {
		t.Errorf("Unexpected error during generation: %v", err)
	}
	if transient == nil {
		t.Errorf("Expected transient storage to be generated, but got nil")
	}
	if len(transient.GetStorageKeys()) == 0 {
		t.Errorf("Expected transient storage to be non-empty, but got empty")
	}
}

func TestTransient_SetConstraintIsEnforced(t *testing.T) {
	v1 := Variable("v1")
	assignment := Assignment{}
	assignment[v1] = common.NewU256(42)

	rnd := rand.New(0)
	generator := NewTransientGenerator()
	generator.BindSet(v1)

	transient, err := generator.Generate(assignment, rnd)
	if err != nil {
		t.Errorf("Expected error during generation, %v", err)
	}
	if !transient.IsSet(common.NewU256(42)) {
		t.Errorf("Expected constraint to be set, but generated state marked as unset")
	}
}

func TestTransient_UnSetConstraintIsEnforced(t *testing.T) {
	v1 := Variable("v1")
	assignment := Assignment{}
	assignment[v1] = common.NewU256(42)

	rnd := rand.New(0)
	generator := NewTransientGenerator()
	generator.BindNotSet(v1)

	transient, err := generator.Generate(assignment, rnd)
	if err != nil {
		t.Errorf("Expected error during generation, %v", err)
	}
	if !transient.IsNotSet(common.NewU256(42)) {
		t.Errorf("Expected constraint to be not set, but generated state marked as set")
	}
}

func TestTransient_ConflictingSetUnsetConstraintsAreDetected(t *testing.T) {
	v1 := Variable("v1")
	assignment := Assignment{}
	assignment[v1] = common.NewU256(42)

	rnd := rand.New(0)
	generator := NewTransientGenerator()
	generator.BindSet(v1)
	generator.BindNotSet(v1)

	_, err := generator.Generate(assignment, rnd)
	if err == nil {
		t.Errorf("Expected error during generation, but got nil")
	}
}
