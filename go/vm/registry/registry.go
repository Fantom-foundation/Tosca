package registry

import (
	"fmt"
	"strings"

	geth "github.com/ethereum/go-ethereum/core/vm"
)

// This package provides a registry for VM instances. Ideally this registry would
// be placed into the go-opera repository (replacing the current VM-implementation
// registry), however, to limit the impact of Tosca on go-opera it is defined here.
//
// The registry is intended to be used only by Tosca-internal code. The public
// interface forwarding calls to this package is found in the vm package of the
// parent directory. Please only refer to this when importing Tosca into your
// project.

// vm_registry is a global registry for VM instances of different implementations
// and configurations.
var vm_registry = map[string]VirtualMachine{}

// RegisterVirtualMachine can be used to register a new VirtualMachine instance to
// be exported for general use in the binary. The name is not case-sensitive, and
// a panic is triggered if a VM was bound to the same name before, or the VM is nil.
// This function is mainly intented to be used by package initialization code.
func RegisterVirtualMachine(name string, vm VirtualMachine) {
	key := strings.ToLower(name)
	if vm == nil {
		panic(fmt.Sprintf("invalid initialization: cannot register nil-VM using `%s`", key))
	}
	if _, found := vm_registry[key]; found {
		panic(fmt.Sprintf("invalid initialization: multiple VMs registered for `%s`", key))
	}
	vm_registry[key] = vm
	geth.RegisterInterpreterFactory(name, func(evm *geth.EVM, cfg geth.Config) geth.EVMInterpreter {
		return vm.NewInterpreter(evm, cfg)
	})
}

// RegisterInterpreterFactory is a convenience function for the registration call
// above for cases where a VM implementation offers nothing more than means to create
// interpreter instances.
func RegisterInterpreterFactory(name string, factory geth.InterpreterFactory) {
	RegisterVirtualMachine(name, &simpleVm{factory})
}

// GetVirtualMachine performs a lookup for the given name (case-insensitive) in the
// VM registry. The result is nil if no VM was registered under the given name.
func GetVirtualMachine(name string) VirtualMachine {
	return vm_registry[strings.ToLower(name)]
}

type simpleVm struct {
	factory geth.InterpreterFactory
}

func (e *simpleVm) NewInterpreter(evm *geth.EVM, cfg geth.Config) geth.EVMInterpreter {
	return e.factory(evm, cfg)
}

// VirtualMachine is a copy of the vm.VirtualMachine interface to break
// cyclic dependencies.
type VirtualMachine interface {
	NewInterpreter(evm *geth.EVM, cfg geth.Config) geth.EVMInterpreter
}

// ProfilingVM is a copy of the vm.ProfilingVM interface to break
// cyclic dependencies.
type ProfilingVM interface {
	ResetProfile()
	DumpProfile()
}
