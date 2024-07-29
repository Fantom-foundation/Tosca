// Copyright (c) 2024 Fantom Foundation
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at fantom.foundation/bsl11.
//
// Change Date: 2028-4-16
//
// On the date above, in accordance with the Business Source License, use of
// this software will be governed by the GNU Lesser General Public License v3.

package evmrs

/*
#cgo LDFLAGS: -Wl,-rpath,${SRCDIR}/../../../target/debug
*/
import "C"

import (
	"fmt"

	"github.com/Fantom-foundation/Tosca/go/interpreter/evmc"
	"github.com/Fantom-foundation/Tosca/go/tosca"
)

func init() {
	// In the CGO instructions at the top of this file the build directory
	// of the evmrs project is added to the rpath of the resulting library.
	// This way, the libevmrs.so file can be found during runtime, even if
	// the LD_LIBRARY_PATH is not set accordingly.
	{
		evm, err := evmc.LoadEvmcInterpreter("libevmrs.so")
		if err != nil {
			panic(fmt.Errorf("failed to load evmrs library: %s", err))
		}
		// This instance remains in its basic configuration.
		tosca.RegisterInterpreter("evmrs", &evmrsInstance{evm})
	}

	// We create a second instance in which we enable logging.
	{
		evm, err := evmc.LoadEvmcInterpreter("libevmrs.so")
		if err != nil {
			panic(fmt.Errorf("failed to load evmrs library: %s", err))
		}
		if err = evm.SetOption("logging", "true"); err != nil {
			panic(fmt.Errorf("failed to configure EVM instance: %s", err))
		}
		tosca.RegisterInterpreter("evmrs-logging", &evmrsInstance{evm})
	}

	// A third instance without analysis cache.
	{
		evm, err := evmc.LoadEvmcInterpreter("libevmrs.so")
		if err != nil {
			panic(fmt.Errorf("failed to load evmrs library: %s", err))
		}
		if err = evm.SetOption("analysis_cache", "false"); err != nil {
			panic(fmt.Errorf("failed to configure EVM instance: %s", err))
		}
		tosca.RegisterInterpreter("evmrs-no-analysis-cache", &evmrsInstance{evm})
	}

	// Another instance without SHA3 cache.
	{
		evm, err := evmc.LoadEvmcInterpreter("libevmrs.so")
		if err != nil {
			panic(fmt.Errorf("failed to load evmrs library: %s", err))
		}
		if err = evm.SetOption("sha3_cache", "false"); err != nil {
			panic(fmt.Errorf("failed to configure EVM instance: %s", err))
		}
		tosca.RegisterInterpreter("evmrs-no-sha3-cache", &evmrsInstance{evm})
	}

	// Another instance in which we enable profiling.
	{
		evm, err := evmc.LoadEvmcInterpreter("libevmrs.so")
		if err != nil {
			panic(fmt.Errorf("failed to load evmrs library: %s", err))
		}
		if err = evm.SetOption("profiling", "true"); err != nil {
			panic(fmt.Errorf("failed to configure EVM instance: %s", err))
		}
		tosca.RegisterInterpreter("evmrs-profiling", &evmrsInstanceWithProfiler{&evmrsInstance{evm}})
	}

	// Another instance in which we enable profiling external.
	{
		evm, err := evmc.LoadEvmcInterpreter("libevmrs.so")
		if err != nil {
			panic(fmt.Errorf("failed to load evmrs library: %s", err))
		}
		if err = evm.SetOption("profiling_external", "true"); err != nil {
			panic(fmt.Errorf("failed to configure EVM instance: %s", err))
		}
		tosca.RegisterInterpreter("evmrs-profiling-external", &evmrsInstanceWithProfiler{&evmrsInstance{evm}})
	}
}

type evmrsInstance struct {
	e *evmc.EvmcInterpreter
}

const newestSupportedRevision = tosca.R13_Cancun

func (e *evmrsInstance) Run(params tosca.Parameters) (tosca.Result, error) {
	if params.Revision > newestSupportedRevision {
		return tosca.Result{}, &tosca.ErrUnsupportedRevision{Revision: params.Revision}
	}
	return e.e.Run(params)
}

// evmrsInstanceWithProfiler implements the tosca.ProfilingVM interface and is used for all
// configurations collecting profiling data.
type evmrsInstanceWithProfiler struct {
	*evmrsInstance
}
