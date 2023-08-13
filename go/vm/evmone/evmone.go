package evmone

/*
#cgo LDFLAGS: -Wl,-rpath,${SRCDIR}/../../../third_party/evmone/build/lib
*/
import "C"

import (
	"fmt"

	"github.com/Fantom-foundation/Tosca/go/common"
	"github.com/Fantom-foundation/Tosca/go/vm/registry"
	"github.com/ethereum/go-ethereum/core/vm"
)

var evmoneBasic *common.EvmcVM
var evmoneAdvanced *common.EvmcVM

func init() {
	// In the CGO instructions at the top of this file the build directory
	// of the evmone project is added to the rpath of the resulting library.
	// This way, the libevmone.so file can be found during runtime, even if
	// the LD_LIBRARY_PATH is not set accordingly.
	vm, err := common.LoadEvmcVM("libevmone.so")
	if err != nil {
		panic(fmt.Errorf("failed to load evmone library: %s", err))
	}
	// This instance remains in its basic configuration.
	evmoneBasic = vm

	// A second instance is configured to use the advanced execution mode.
	vm, err = common.LoadEvmcVM("libevmone.so")
	if err != nil {
		panic(fmt.Errorf("failed to load evmone library: %s", err))
	}
	if err := vm.SetOption("advanced", "on"); err != nil {
		panic(fmt.Errorf("failed to configure evmone advnaced mode: %v", err))
	}
	evmoneAdvanced = vm
}

func newInterpreter(vm *common.EvmcVM, evm *vm.EVM, cfg vm.Config) vm.EVMInterpreter {
	return common.NewEvmcInterpreter(vm, evm, cfg)
}

func NewBasicInterpreter(evm *vm.EVM, cfg vm.Config) vm.EVMInterpreter {
	return newInterpreter(evmoneBasic, evm, cfg)
}

func NewAdvancedInterpreter(evm *vm.EVM, cfg vm.Config) vm.EVMInterpreter {
	return newInterpreter(evmoneAdvanced, evm, cfg)
}

func init() {
	registry.RegisterInterpreterFactory("evmone-basic", NewBasicInterpreter)
	registry.RegisterInterpreterFactory("evmone-advanced", NewAdvancedInterpreter)

	// We use the basic version as the default since it showed better performance in
	// benchmarks (to verify on your system, run benchmarks in go/vm/vm_test.go).
	registry.RegisterInterpreterFactory("evmone", NewBasicInterpreter)
}
