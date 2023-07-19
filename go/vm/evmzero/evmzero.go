package evmzero

/*
#cgo LDFLAGS: -Wl,-rpath,${SRCDIR}/../../../cpp/build/vm/evmzero
*/
import "C"

import (
	"fmt"

	"github.com/Fantom-foundation/Tosca/go/common"
	"github.com/ethereum/go-ethereum/core/vm"
)

var evmzero *common.EvmcVM
var evmzeroWithLogging *common.EvmcVM
var evmzeroWithoutAnalysisCache *common.EvmcVM

func init() {
	// In the CGO instructions at the top of this file the build directory
	// of the evmzero project is added to the rpath of the resulting library.
	// This way, the libevmzero.so file can be found during runtime, even if
	// the LD_LIBRARY_PATH is not set accordingly.
	vm, err := common.LoadEvmcVM("libevmzero.so")
	if err != nil {
		panic(fmt.Errorf("failed to load evmzero library: %s", err))
	}
	// This instance remains in its basic configuration.
	evmzero = vm

	// We create a second instance in which we enable logging.
	vm, err = common.LoadEvmcVM("libevmzero.so")
	if err != nil {
		panic(fmt.Errorf("failed to load evmzero library: %s", err))
	}
	if err = vm.SetOption("logging", "true"); err != nil {
		panic(fmt.Errorf("failed to configure EVM instance: %s", err))
	}
	evmzeroWithLogging = vm

	// A third instance without analysis cache.
	vm, err = common.LoadEvmcVM("libevmzero.so")
	if err != nil {
		panic(fmt.Errorf("failed to load evmzero library: %s", err))
	}
	if err = vm.SetOption("analysis_cache", "false"); err != nil {
		panic(fmt.Errorf("failed to configure EVM instance: %s", err))
	}
	evmzeroWithoutAnalysisCache = vm
}

func newInterpreter(evm *vm.EVM, cfg vm.Config) vm.EVMInterpreter {
	return common.NewEvmcInterpreter(evmzero, evm, cfg)
}

func newLoggingInterpreter(evm *vm.EVM, cfg vm.Config) vm.EVMInterpreter {
	return common.NewEvmcInterpreter(evmzeroWithLogging, evm, cfg)
}

func newInterpreterWithoutAnalysisCache(evm *vm.EVM, cfg vm.Config) vm.EVMInterpreter {
	return common.NewEvmcInterpreter(evmzeroWithoutAnalysisCache, evm, cfg)
}

func init() {
	vm.RegisterInterpreterFactory("evmzero", newInterpreter)
	vm.RegisterInterpreterFactory("evmzero-logging", newLoggingInterpreter)
	vm.RegisterInterpreterFactory("evmzero-no-analysis-cache", newInterpreterWithoutAnalysisCache)
}
