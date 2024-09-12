// Copyright (c) 2024 Fantom Foundation
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at fantom.foundation/bsl11.
//
// Change Date: 2028-4-16
//
// On the date above, in accordance with the Business Source License, use of
// this software will be governed by the GNU Lesser General Public License v3.

package lfvm

import (
	"testing"

	"github.com/Fantom-foundation/Tosca/go/tosca"
)

func TestLfvm_OfficialConfigurationHasSanctionedProperties(t *testing.T) {
	vm, err := tosca.NewInterpreter("lfvm")
	if err != nil {
		t.Fatalf("lfvm is not registered: %v", err)
	}
	lfvm, ok := vm.(*lfvm)
	if !ok {
		t.Fatalf("unexpected interpreter implementation, got %T", vm)
	}
	if lfvm.config.WithShaCache != true {
		t.Fatalf("lfvm is not configured with sha cache")
	}
	if lfvm.config.ConversionConfig.WithSuperInstructions != false {
		t.Fatalf("lfvm is configured with super instructions")
	}
}
