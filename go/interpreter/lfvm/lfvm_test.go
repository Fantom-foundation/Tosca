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
	"fmt"
	"testing"

	"github.com/Fantom-foundation/Tosca/go/tosca"
)

func TestNewInterpreter_ProducesInstanceWithSanctionedProperties(t *testing.T) {
	lfvm, err := NewInterpreter(Config{})
	if err != nil {
		t.Fatalf("failed to create LFVM instance: %v", err)
	}
	if lfvm.config.WithShaCache != true {
		t.Fatalf("LFVM is not configured with sha cache")
	}
	if lfvm.config.ConversionConfig.WithSuperInstructions != false {
		t.Fatalf("LFVM is configured with super instructions")
	}
}

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

func TestLfvm_InterpreterReturnsErrorWhenExecutingUnsupportedRevision(t *testing.T) {
	vm, err := tosca.NewInterpreter("lfvm")
	if err != nil {
		t.Fatalf("lfvm is not registered: %v", err)
	}

	params := tosca.Parameters{}
	params.Revision = newestSupportedRevision + 1

	_, err = vm.Run(params)
	if want, got := fmt.Sprintf("unsupported revision %d", params.Revision), err.Error(); want != got {
		t.Fatalf("unexpected error: want %q, got %q", want, got)
	}
}

func TestLfvm_newVm_returnsErrorWithWrongConfiguration(t *testing.T) {
	config := config{
		ConversionConfig: ConversionConfig{CacheSize: maxCachedCodeLength / 2},
	}
	_, err := newVm(config)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
}
