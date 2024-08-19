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
#cgo LDFLAGS: -Wl,-rpath,${SRCDIR}/../../../target/release
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
