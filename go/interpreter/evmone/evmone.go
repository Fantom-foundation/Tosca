// Copyright (c) 2024 Fantom Foundation
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at fantom.foundation/bsl11.
//
// Change Date: 2028-4-16
//
// On the date above, in accordance with the Business Source License, use of
// this software will be governed by the GNU Lesser General Public License v3.

package evmone

/*
#cgo LDFLAGS: -Wl,-rpath,${SRCDIR}/../../../third_party/evmone/build/lib
*/
import "C"

import (
	"fmt"

	"github.com/Fantom-foundation/Tosca/go/interpreter/evmc"
	"github.com/Fantom-foundation/Tosca/go/tosca"
)

func init() {
	// In the CGO instructions at the top of this file the build directory
	// of the evmone project is added to the rpath of the resulting library.
	// This way, the libevmone.so file can be found during runtime, even if
	// the LD_LIBRARY_PATH is not set accordingly.
	evmone, err := evmc.LoadEvmcInterpreter("libevmone.so")
	if err != nil {
		panic(fmt.Errorf("failed to load evmone library: %s", err))
	}
	// This instance remains in its basic configuration and is registered
	// as the default "evmone" VM and as the "evmone-basic" tosca.
	tosca.MustRegisterInterpreterFactory("evmone", func(any) (tosca.Interpreter, error) {
		return &evmoneInstance{evmone}, nil
	})
	tosca.MustRegisterInterpreterFactory("evmone-basic", func(any) (tosca.Interpreter, error) {
		return &evmoneInstance{evmone}, nil
	})

	// A second instance is configured to use the advanced execution mode.
	evmone, err = evmc.LoadEvmcInterpreter("libevmone.so")
	if err != nil {
		panic(fmt.Errorf("failed to load evmone library: %s", err))
	}
	if err := evmone.SetOption("advanced", "on"); err != nil {
		panic(fmt.Errorf("failed to configure evmone advanced mode: %v", err))
	}
	tosca.MustRegisterInterpreterFactory("evmone-advanced", func(any) (tosca.Interpreter, error) {
		return &evmoneInstance{evmone}, nil
	})
}

type evmoneInstance struct {
	e *evmc.EvmcInterpreter
}

const newestSupportedRevision = tosca.R13_Cancun

func (e *evmoneInstance) Run(params tosca.Parameters) (tosca.Result, error) {
	if params.Revision > newestSupportedRevision {
		return tosca.Result{}, &tosca.ErrUnsupportedRevision{Revision: params.Revision}
	}
	return e.e.Run(params)
}
