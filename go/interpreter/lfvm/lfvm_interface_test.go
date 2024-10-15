// Copyright (c) 2024 Fantom Foundation
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at fantom.foundation/bsl11.
//
// Change Date: 2028-4-16
//
// On the date above, in accordance with the Business Source License, use of
// this software will be governed by the GNU Lesser General Public License v3.

package lfvm_test

import (
	"testing"

	"github.com/Fantom-foundation/Tosca/go/interpreter/lfvm"
	"github.com/Fantom-foundation/Tosca/go/tosca"
)

// TestLfvm_RegisterExperimentalConfigurations tests the registration of
// experimental configurations.
// This test is slightly different from other tests because of dealing with the
// global registry:
// - It is declared in it's own package to avoid leaking the registration to other tests.
// - It tests different properties, in one single function. The reason is that the
// order of different functions may change, invalidating the test.
func TestLfvm_RegisterExperimentalConfigurations(t *testing.T) {

	// Fist registration must succeed.
	err := lfvm.RegisterExperimentalInterpreterConfigurations()
	if err != nil {
		t.Fatalf("failed to register experimental configurations: %v", err)
	}

	// Registering a second time must fail.
	err = lfvm.RegisterExperimentalInterpreterConfigurations()
	if err == nil {
		t.Fatalf("expected error when registering experimental configurations twice")
	}

	// Check that lfvm is registered by default, in addition to experimental configurations
	if _, ok := tosca.GetAllRegisteredInterpreters()["lfvm"]; !ok {
		t.Fatalf("lfvm is not registered")
	}

	// Construct all registered interpreter configurations
	for name, factory := range tosca.GetAllRegisteredInterpreters() {
		t.Run(name, func(t *testing.T) {
			vm, err := factory(lfvm.Config{})
			if err != nil {
				t.Fatalf("failed to create interpreter: %v", err)
			}

			// Vms are opaque, we can't check their configuration directly.
			// We can only check that they do execute some basic code.
			params := tosca.Parameters{}
			params.Code = []byte{byte(lfvm.PUSH1), 0xff, byte(lfvm.POP), byte(lfvm.STOP)}
			params.Gas = 5
			res, err := vm.Run(params)
			if err != nil {
				t.Fatalf("failed to run interpreter: %v", err)
			}

			if want, got := true, res.Success; want != got {
				t.Fatalf("unexpected success result: want %v, got %v", want, got)
			}
			if want, got := tosca.Gas(0), res.GasLeft; want != got {
				t.Fatalf("unexpected gas used: want %v, got %v", want, got)
			}
		})
	}
}
