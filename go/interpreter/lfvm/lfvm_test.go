package lfvm

import (
	"testing"

	"github.com/Fantom-foundation/Tosca/go/tosca"
)

func TestLfvm_OfficialConfigurationHasSanctionedProperties(t *testing.T) {
	vm := tosca.GetInterpreter("lfvm")
	if vm == nil {
		t.Fatal("lfvm is not registered")
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
