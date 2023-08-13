package vm

// These are the officially exported EVM variants provided
// by this package. Other VM implementations may be present
// in this repository, but they are not (yet) intended for
// external use.

import (
	// EVM implementations offered externaly.
	_ "github.com/Fantom-foundation/Tosca/go/vm/evmone"
	_ "github.com/Fantom-foundation/Tosca/go/vm/evmzero"
	_ "github.com/Fantom-foundation/Tosca/go/vm/lfvm"

	// Implementation of registry support
	"github.com/Fantom-foundation/Tosca/go/vm/registry"
	geth "github.com/ethereum/go-ethereum/core/vm"
)

// VirtualMachine (VM) represents an instance of an EVM-byte-code execution engine
// loaded in memory. Each instance can host multiple interpreter instances, each
// capable of running a single EVM contract invocation. Multiple interpreters may
// exist at the same time, processing contracts in parallel. However, at any time
// each interpreter instance may only process a single contract.
type VirtualMachine interface {
	// NewInterpreter creates a new interpreter instance based on this VM instance.
	NewInterpreter(evm *geth.EVM, cfg geth.Config) geth.EVMInterpreter
}

// ProfilingVM is an optional extension to the VirtualMachine interface above which
// may be implemented by VM implementations collecting statistical data regarding
// their execution.
type ProfilingVM interface {
	// ResetProfile resets the operation statistic collected by the underlying VM implementation.
	// Use this, for instance, at the beginning of a benchmark. It should not be called while
	// running operations on the VM implementations in parallel.
	ResetProfile()

	// DumpProfile prints a snapshot of the profiling data collected since the last reset to stdout.
	// In the future this interface will be changed to return the result instead of printing it.
	DumpProfile()
}

// GetVirtualMachine provides access to all VM implementations loaded in the current binary.
func GetVirtualMachine(name string) VirtualMachine {
	return registry.GetVirtualMachine(name)
}
