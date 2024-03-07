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
	"github.com/Fantom-foundation/Tosca/go/vm"
)

func init() {
	// In the CGO instructions at the top of this file the build directory
	// of the evmzero project is added to the rpath of the resulting library.
	// This way, the libevmzero.so file can be found during runtime, even if
	// the LD_LIBRARY_PATH is not set accordingly.
	cur, err := common.LoadEvmcInterpreter("libevmzero.so")
	if err != nil {
		panic(fmt.Errorf("failed to load evmzero library: %s", err))
	}
	// This instance remains in its basic configuration.
	evmzero := cur

	// We create a second instance in which we enable logging.
	cur, err = common.LoadEvmcInterpreter("libevmzero.so")
	if err != nil {
		panic(fmt.Errorf("failed to load evmzero library: %s", err))
	}
	if err = cur.SetOption("logging", "true"); err != nil {
		panic(fmt.Errorf("failed to configure EVM instance: %s", err))
	}
	evmzeroWithLogging := cur

	// A third instance without analysis cache.
	cur, err = common.LoadEvmcInterpreter("libevmzero.so")
	if err != nil {
		panic(fmt.Errorf("failed to load evmzero library: %s", err))
	}
	if err = cur.SetOption("analysis_cache", "false"); err != nil {
		panic(fmt.Errorf("failed to configure EVM instance: %s", err))
	}
	evmzeroWithoutAnalysisCache := cur

	// Another instance without SHA3 cache.
	cur, err = common.LoadEvmcInterpreter("libevmzero.so")
	if err != nil {
		panic(fmt.Errorf("failed to load evmzero library: %s", err))
	}
	if err = cur.SetOption("sha3_cache", "false"); err != nil {
		panic(fmt.Errorf("failed to configure EVM instance: %s", err))
	}
	evmzeroWithoutSha3Cache := cur

	// Another instance in which we enable profiling.
	cur, err = common.LoadEvmcInterpreter("libevmzero.so")
	if err != nil {
		panic(fmt.Errorf("failed to load evmzero library: %s", err))
	}
	if err = cur.SetOption("profiling", "true"); err != nil {
		panic(fmt.Errorf("failed to configure EVM instance: %s", err))
	}
	evmzeroWithProfiling := cur

	// Another instance in which we enable profiling external.
	cur, err = common.LoadEvmcInterpreter("libevmzero.so")
	if err != nil {
		panic(fmt.Errorf("failed to load evmzero library: %s", err))
	}
	if err = cur.SetOption("profiling_external", "true"); err != nil {
		panic(fmt.Errorf("failed to configure EVM instance: %s", err))
	}
	evmzeroWithProfilingExternal := cur

	vm.RegisterInterpreter("evmzero", evmzero)
	vm.RegisterInterpreter("evmzero-logging", evmzeroWithLogging)
	vm.RegisterInterpreter("evmzero-no-analysis-cache", evmzeroWithoutAnalysisCache)
	vm.RegisterInterpreter("evmzero-no-sha3-cache", evmzeroWithoutSha3Cache)
	vm.RegisterInterpreter("evmzero-profiling", &evmzeroInstanceWithProfiler{evmzeroWithProfiling})
	vm.RegisterInterpreter("evmzero-profiling-external", &evmzeroInstanceWithProfiler{evmzeroWithProfilingExternal})
}

// evmzeroInstanceWithProfiler implements the vm.ProfilingVM interface and is used for all
// configurations collecting profiling data.
type evmzeroInstanceWithProfiler struct {
	*common.EvmcInterpreter
}

func (e *evmzeroInstanceWithProfiler) DumpProfile() {
	C.evmzero_dump_profile(e.GetEvmcVM().GetHandle())
}

func (e *evmzeroInstanceWithProfiler) ResetProfile() {
	C.evmzero_reset_profiler(e.GetEvmcVM().GetHandle())
}
