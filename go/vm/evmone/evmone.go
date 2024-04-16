//
// Copyright (c) 2024 Fantom Foundation
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at fantom.foundation/bsl11.
//
// Change Date: 2028-4-16
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by the GNU Lesser General Public Licence v3
//

package evmone

/*
#cgo LDFLAGS: -Wl,-rpath,${SRCDIR}/../../../third_party/evmone/build/lib
*/
import "C"

import (
	"fmt"

	"github.com/Fantom-foundation/Tosca/go/vm"
	"github.com/Fantom-foundation/Tosca/go/vm/evmc"
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
	// as the default "evmone" VM and as the "evmone-basic" VM.
	vm.RegisterInterpreter("evmone", evmone)
	vm.RegisterInterpreter("evmone-basic", evmone)

	// A second instance is configured to use the advanced execution mode.
	evmone, err = evmc.LoadEvmcInterpreter("libevmone.so")
	if err != nil {
		panic(fmt.Errorf("failed to load evmone library: %s", err))
	}
	if err := evmone.SetOption("advanced", "on"); err != nil {
		panic(fmt.Errorf("failed to configure evmone advanced mode: %v", err))
	}
	vm.RegisterInterpreter("evmone-advanced", evmone)
}
