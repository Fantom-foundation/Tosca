package lfvm

import (
	"github.com/ethereum/go-ethereum/core/vm"
)

type EVMInterpreter struct {
	evm                     *vm.EVM
	cfg                     vm.Config
	with_super_instructions bool
	with_shadow_evm         bool
	with_statistics         bool
	readOnly                bool
	no_shaCache             bool
}

// Registers the long-form EVM as a possible interpreter implementation.
func init() {
	vm.RegisterInterpreterFactory("lfvm", func(evm *vm.EVM, cfg vm.Config) vm.EVMInterpreter {
		return &EVMInterpreter{evm: evm, cfg: cfg}
	})
	vm.RegisterInterpreterFactory("lfvm-no-sha-cache", func(evm *vm.EVM, cfg vm.Config) vm.EVMInterpreter {
		return &EVMInterpreter{evm: evm, cfg: cfg, no_shaCache: true}
	})
	vm.RegisterInterpreterFactory("lfvm-si", func(evm *vm.EVM, cfg vm.Config) vm.EVMInterpreter {
		return &EVMInterpreter{evm: evm, cfg: cfg, with_super_instructions: true}
	})
	vm.RegisterInterpreterFactory("lfvm-si-no-sha-cache", func(evm *vm.EVM, cfg vm.Config) vm.EVMInterpreter {
		return &EVMInterpreter{evm: evm, cfg: cfg, with_super_instructions: true, no_shaCache: true}
	})
	vm.RegisterInterpreterFactory("lfvm-dbg", func(evm *vm.EVM, cfg vm.Config) vm.EVMInterpreter {
		return &EVMInterpreter{evm: evm, cfg: cfg, with_shadow_evm: true}
	})
	vm.RegisterInterpreterFactory("lfvm-stats", func(evm *vm.EVM, cfg vm.Config) vm.EVMInterpreter {
		return &EVMInterpreter{evm: evm, cfg: cfg, with_statistics: true}
	})
	vm.RegisterInterpreterFactory("lfvm-si-stats", func(evm *vm.EVM, cfg vm.Config) vm.EVMInterpreter {
		return &EVMInterpreter{evm: evm, cfg: cfg, with_super_instructions: true, with_statistics: true}
	})
}

func (e *EVMInterpreter) Run(contract *vm.Contract, input []byte, readOnly bool) (ret []byte, err error) {
	converted, err := Convert(*contract.CodeAddr, contract.Code, e.with_super_instructions, e.evm.Context.BlockNumber.Uint64(), input == nil)
	if err != nil {
		panic(err)
		//return nil, err
	}

	// Make sure the readOnly is only set if we aren't in readOnly yet.
	// This also makes sure that the readOnly flag isn't removed for child calls.
	if readOnly && !e.readOnly {
		e.readOnly = true
		defer func() { e.readOnly = false }()
	}
	return Run(e.evm, e.cfg, contract, converted, input, e.readOnly, e.evm.StateDB, e.with_shadow_evm, e.with_statistics, e.no_shaCache)
}
