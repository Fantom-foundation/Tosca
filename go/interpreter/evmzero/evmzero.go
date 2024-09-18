// Copyright (c) 2024 Fantom Foundation
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at fantom.foundation/bsl11.
//
// Change Date: 2028-4-16
//
// On the date above, in accordance with the Business Source License, use of
// this software will be governed by the GNU Lesser General Public License v3.

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

	"github.com/Fantom-foundation/Tosca/go/interpreter/evmc"
	"github.com/Fantom-foundation/Tosca/go/tosca"
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
		tosca.MustRegisterInterpreterFactory("evmzero", func(any) (tosca.Interpreter, error) {
			return &evmzeroInstance{evm}, nil
		})
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
		tosca.MustRegisterInterpreterFactory("evmzero-logging", func(any) (tosca.Interpreter, error) {
			return &evmzeroInstance{evm}, nil
		})
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
		tosca.MustRegisterInterpreterFactory("evmzero-no-analysis-cache", func(any) (tosca.Interpreter, error) {
			return &evmzeroInstance{evm}, nil
		})
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
		tosca.MustRegisterInterpreterFactory("evmzero-no-sha3-cache", func(any) (tosca.Interpreter, error) {
			return &evmzeroInstance{evm}, nil
		})
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
		tosca.MustRegisterInterpreterFactory("evmzero-profiling", func(any) (tosca.Interpreter, error) {
			return &evmzeroInstanceWithProfiler{&evmzeroInstance{evm}}, nil
		})
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
		tosca.MustRegisterInterpreterFactory("evmzero-profiling-external", func(any) (tosca.Interpreter, error) {
			return &evmzeroInstanceWithProfiler{&evmzeroInstance{evm}}, nil
		})
	}
}

type evmzeroInstance struct {
	e *evmc.EvmcInterpreter
}

const newestSupportedRevision = tosca.R13_Cancun

func (e *evmzeroInstance) Run(params tosca.Parameters) (tosca.Result, error) {
	if params.Revision > newestSupportedRevision {
		return tosca.Result{}, &tosca.ErrUnsupportedRevision{Revision: params.Revision}
	}
	return e.e.Run(params)
}

// evmzeroInstanceWithProfiler implements the tosca.ProfilingVM interface and is used for all
// configurations collecting profiling data.
type evmzeroInstanceWithProfiler struct {
	*evmzeroInstance
}

func (e *evmzeroInstanceWithProfiler) DumpProfile() {
	C.evmzero_dump_profile(e.e.GetEvmcVM().GetHandle())
}

func (e *evmzeroInstanceWithProfiler) ResetProfile() {
	C.evmzero_reset_profiler(e.e.GetEvmcVM().GetHandle())
}
