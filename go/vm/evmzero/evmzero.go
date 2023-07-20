package evmzero

/*
#cgo LDFLAGS: -L${SRCDIR}/../../../cpp/build/vm/evmzero -levmzero -Wl,-rpath,${SRCDIR}/../../../cpp/build/vm/evmzero
// Declarations for evmzero API exceeding EVMC requirements.
void evmzero_dump_profile(void* vm);
void evmzero_reset_profiler(void* vm);
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
var evmzeroWithoutSha3Cache *common.EvmcVM
var evmzeroWithProfiling *common.EvmcVM

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

	// Another instance without SHA3 cache.
	vm, err = common.LoadEvmcVM("libevmzero.so")
	if err != nil {
		panic(fmt.Errorf("failed to load evmzero library: %s", err))
	}
	if err = vm.SetOption("sha3_cache", "false"); err != nil {
		panic(fmt.Errorf("failed to configure EVM instance: %s", err))
	}
	evmzeroWithoutSha3Cache = vm

	// Another instance in which we enable profiling.
	vm, err = common.LoadEvmcVM("libevmzero.so")
	if err != nil {
		panic(fmt.Errorf("failed to load evmzero library: %s", err))
	}
	if err = vm.SetOption("profiling", "true"); err != nil {
		panic(fmt.Errorf("failed to configure EVM instance: %s", err))
	}
	evmzeroWithProfiling = vm
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

func newInterpreterWithoutSha3Cache(evm *vm.EVM, cfg vm.Config) vm.EVMInterpreter {
	return common.NewEvmcInterpreter(evmzeroWithoutSha3Cache, evm, cfg)
}

func newProfilingInterpreter(evm *vm.EVM, cfg vm.Config) vm.EVMInterpreter {
	return common.NewEvmcInterpreter(evmzeroWithProfiling, evm, cfg)
}

func init() {
	vm.RegisterInterpreterFactory("evmzero", newInterpreter)
	vm.RegisterInterpreterFactory("evmzero-logging", newLoggingInterpreter)
	vm.RegisterInterpreterFactory("evmzero-no-analysis-cache", newInterpreterWithoutAnalysisCache)
	vm.RegisterInterpreterFactory("evmzero-no-sha3-cache", newInterpreterWithoutSha3Cache)
	vm.RegisterInterpreterFactory("evmzero-profiling", newProfilingInterpreter)
}

// DumpProfile prints a snapshot of the profiling data collected since the last reset to stdout.
// In the future this interface will be changed to return the result instead of printing it.
func DumpProfile(interpreter vm.EVMInterpreter) {
	if evmc, ok := interpreter.(*common.EvmcInterpreter); ok {
		C.evmzero_dump_profile(evmc.GetEvmcVM().GetHandle())
	} else {
		fmt.Printf("Cannot dump profiler data for non-evmzero interpreter.\n")
	}
}

func ResetProfiler(interpreter vm.EVMInterpreter) {
	if evmc, ok := interpreter.(*common.EvmcInterpreter); ok {
		C.evmzero_reset_profiler(evmc.GetEvmcVM().GetHandle())
	} else {
		fmt.Printf("Cannot reset profiler for non-evmzero interpreter.\n")
	}
}
