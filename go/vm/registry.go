package vm

import (
	"fmt"
	"strings"

	"golang.org/x/exp/maps"
)

// This file provides a registry for VM instances in Tosca.
//
// The registry is intended to be used by all client applications that would
// like to use VM services. However, not all implementations are available
// by default. Experimental VMs are excluded, but can be added by registering
// them through a call to RegisterVirtualMachine(). Typically, the registration
// should be done by the package providing a VM implementation implicitly
// during initialization, such that beyond importing the implementation package
// no additional work is required.

// GetVirtualMachine performs a lookup for the given name (case-insensitive) in the
// VM registry. The result is nil if no VM was registered under the given name.
func GetVirtualMachine(name string) VirtualMachine {
	return vm_registry[strings.ToLower(name)]
}

// GetAllRegisteredVMs obtains a full index of all registered VMs.
func GetAllRegisteredVMs() map[string]VirtualMachine {
	return maps.Clone(vm_registry)
}

// RegisterVirtualMachine can be used to register a new VirtualMachine instance to
// be exported for general use in the binary. The name is not case-sensitive, and
// a panic is triggered if a VM was bound to the same name before, or the VM is nil.
// This function is mainly intended to be used by package initialization code.
func RegisterVirtualMachine(name string, vm VirtualMachine) {
	key := strings.ToLower(name)
	if vm == nil {
		panic(fmt.Sprintf("invalid initialization: cannot register nil-VM using `%s`", key))
	}
	if _, found := vm_registry[key]; found {
		panic(fmt.Sprintf("invalid initialization: multiple VMs registered for `%s`", key))
	}
	vm_registry[key] = vm
}

// vm_registry is a global registry for VM instances of different implementations
// and configurations.
var vm_registry = map[string]VirtualMachine{}
