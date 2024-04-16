//
// Copyright (c) 2024 Fantom Foundation
//
// Use of this software is governed by the Business Source License included
// in the LICENSE.TXT file and at fantom.foundation/bsl11.
//
// Change Date: 2028-4-16
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by the GNU Lesser General Public Licence v3
//

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

	"github.com/Fantom-foundation/Tosca/go/vm"
	"github.com/Fantom-foundation/Tosca/go/vm/evmc"
)

func init() {
	// In the CGO instructions at the top of this file the build directory
	// of the evmzero project is added to the rpath of the resulting library.
	// This way, the libevmzero.so file can be found during runtime, even if
	// the LD_LIBRARY_PATH is not set accordingly.
	{
		evm, err := evmc.LoadEvmcInterpreter("libevmzero.so")
		if err != nil {
			panic(fmt.Errorf("failed to load evmzero library: %s", err))
		}
		// This instance remains in its basic configuration.
		vm.RegisterInterpreter("evmzero", evm)
	}

	// We create a second instance in which we enable logging.
	{
		evm, err := evmc.LoadEvmcInterpreter("libevmzero.so")
		if err != nil {
			panic(fmt.Errorf("failed to load evmzero library: %s", err))
		}
		if err = evm.SetOption("logging", "true"); err != nil {
			panic(fmt.Errorf("failed to configure EVM instance: %s", err))
		}
		vm.RegisterInterpreter("evmzero-logging", evm)
	}

	// A third instance without analysis cache.
	{
		evm, err := evmc.LoadEvmcInterpreter("libevmzero.so")
		if err != nil {
			panic(fmt.Errorf("failed to load evmzero library: %s", err))
		}
		if err = evm.SetOption("analysis_cache", "false"); err != nil {
			panic(fmt.Errorf("failed to configure EVM instance: %s", err))
		}
		vm.RegisterInterpreter("evmzero-no-analysis-cache", evm)
	}

	// Another instance without SHA3 cache.
	{
		evm, err := evmc.LoadEvmcInterpreter("libevmzero.so")
		if err != nil {
			panic(fmt.Errorf("failed to load evmzero library: %s", err))
		}
		if err = evm.SetOption("sha3_cache", "false"); err != nil {
			panic(fmt.Errorf("failed to configure EVM instance: %s", err))
		}
		vm.RegisterInterpreter("evmzero-no-sha3-cache", evm)
	}

	// Another instance in which we enable profiling.
	{
		evm, err := evmc.LoadEvmcInterpreter("libevmzero.so")
		if err != nil {
			panic(fmt.Errorf("failed to load evmzero library: %s", err))
		}
		if err = evm.SetOption("profiling", "true"); err != nil {
			panic(fmt.Errorf("failed to configure EVM instance: %s", err))
		}
		vm.RegisterInterpreter("evmzero-profiling", &evmzeroInstanceWithProfiler{evm})
	}

	// Another instance in which we enable profiling external.
	{
		evm, err := evmc.LoadEvmcInterpreter("libevmzero.so")
		if err != nil {
			panic(fmt.Errorf("failed to load evmzero library: %s", err))
		}
		if err = evm.SetOption("profiling_external", "true"); err != nil {
			panic(fmt.Errorf("failed to configure EVM instance: %s", err))
		}
		vm.RegisterInterpreter("evmzero-profiling-external", &evmzeroInstanceWithProfiler{evm})
	}
}

// evmzeroInstanceWithProfiler implements the vm.ProfilingVM interface and is used for all
// configurations collecting profiling data.
type evmzeroInstanceWithProfiler struct {
	*evmc.EvmcInterpreter
}

func (e *evmzeroInstanceWithProfiler) DumpProfile() {
	C.evmzero_dump_profile(e.GetEvmcVM().GetHandle())
}

func (e *evmzeroInstanceWithProfiler) ResetProfile() {
	C.evmzero_reset_profiler(e.GetEvmcVM().GetHandle())
}
