package tosca

// This is the official public interface of the Tosca project,
// to be used by client code to run code on a range of provided
// EVM implementations.

import (
	"github.com/Fantom-foundation/Tosca/go/vm"

	// These are the officially exported EVM variants provided
	// by this package. Other VM implementations may be present
	// in this repository, but they are not (yet) intended for
	// general use.

	// EVM implementations offered externally.
	_ "github.com/Fantom-foundation/Tosca/go/vm/evmone"
	_ "github.com/Fantom-foundation/Tosca/go/vm/evmzero"
	_ "github.com/Fantom-foundation/Tosca/go/vm/lfvm"
)

func GetVirtualMachine(name string) VirtualMachine {
	return vm.GetVirtualMachine(name)
}

// A few type alias for user convenience.
type VirtualMachine = vm.VirtualMachine
type Parameters = vm.Parameters
type Result = vm.Result
