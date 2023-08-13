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
	"github.com/Fantom-foundation/Tosca/go/vm/registry"
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

func init() {
	registry.RegisterVirtualMachine("evmzero", &evmzeroInstance{evmzero})
	registry.RegisterVirtualMachine("evmzero-logging", &evmzeroInstance{evmzeroWithLogging})
	registry.RegisterVirtualMachine("evmzero-no-analysis-cache", &evmzeroInstance{evmzeroWithoutAnalysisCache})
	registry.RegisterVirtualMachine("evmzero-no-sha3-cache", &evmzeroInstance{evmzeroWithoutSha3Cache})
	registry.RegisterVirtualMachine("evmzero-profiling", &evmzeroInstanceWithProfiler{evmzeroInstance{evmzeroWithProfiling}})
}

// evmzeroInstance implements the vm.VirtualMachine interface and is used for all
// configurations not collecting profiling data.
type evmzeroInstance struct {
	vm *common.EvmcVM
}

func (e *evmzeroInstance) NewInterpreter(evm *vm.EVM, cfg vm.Config) vm.EVMInterpreter {
	return common.NewEvmcInterpreter(e.vm, evm, cfg)
}

// evmzeroInstanceWithProfiler implements the vm.ProfilingVM interface and is used for all
// configurations collecting profiling data.
type evmzeroInstanceWithProfiler struct {
	evmzeroInstance
}

func (e *evmzeroInstanceWithProfiler) DumpProfile() {
	C.evmzero_dump_profile(e.evmzeroInstance.vm.GetEvmcVM().GetHandle())
}

func (e *evmzeroInstanceWithProfiler) ResetProfile() {
	C.evmzero_reset_profiler(e.evmzeroInstance.vm.GetEvmcVM().GetHandle())
}
