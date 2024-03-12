package lfvm

import (
	"github.com/Fantom-foundation/Tosca/go/vm"
)

// Registers the long-form EVM as a possible interpreter implementation.
func init() {
	vm.RegisterInterpreter("lfvm", &VM{})
	vm.RegisterInterpreter("lfvm-no-sha-cache", &VM{no_shaCache: true})
	vm.RegisterInterpreter("lfvm-si", &VM{with_super_instructions: true})
	vm.RegisterInterpreter("lfvm-si-no-sha-cache", &VM{with_super_instructions: true, no_shaCache: true})
	vm.RegisterInterpreter("lfvm-stats", &VM{with_statistics: true})
	vm.RegisterInterpreter("lfvm-si-stats", &VM{with_super_instructions: true, with_statistics: true})
	vm.RegisterInterpreter("lfvm-no-code-cache", &VM{no_code_cache: true})
	vm.RegisterInterpreter("lfvm-logging", &VM{logging: true})
}

type VM struct {
	with_super_instructions bool
	with_statistics         bool
	no_shaCache             bool
	no_code_cache           bool
	logging                 bool
}

func (v *VM) Run(params vm.Parameters) (vm.Result, error) {
	var codeHash vm.Hash
	if params.CodeHash != nil {
		codeHash = *params.CodeHash
	}

	converted, err := Convert(
		params.Code,
		v.with_super_instructions,
		params.CodeHash == nil,
		v.no_code_cache,
		codeHash,
	)
	if err != nil {
		return vm.Result{}, err
	}

	return Run(params, converted, v.with_statistics, v.no_shaCache, v.logging)
}

func (e *VM) DumpProfile() {
	if e.with_statistics {
		printCollectedInstructionStatistics()
	}
}

func (e *VM) ResetProfile() {
	if e.with_statistics {
		resetCollectedInstructionStatistics()
	}
}
