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

import (
	"fmt"

	"github.com/Fantom-foundation/Tosca/go/ct"
	"github.com/Fantom-foundation/Tosca/go/ct/st"
	"github.com/Fantom-foundation/Tosca/go/ct/utils"
	"github.com/Fantom-foundation/Tosca/go/interpreter/evmc"
	"github.com/Fantom-foundation/Tosca/go/tosca"
)

var evmzeroSteppable *evmc.SteppableEvmcInterpreter

func init() {
	interpreter, err := evmc.LoadSteppableEvmcInterpreter("libevmzero.so")
	if err != nil {
		panic(fmt.Errorf("failed to load evmzero library: %s", err))
	}
	evmzeroSteppable = interpreter
}

func NewConformanceTestingTarget() ct.Evm {
	return ctAdapter{}
}

type ctAdapter struct{}

func (a ctAdapter) StepN(state *st.State, numSteps int) (*st.State, error) {
	vmParams := utils.ToVmParameters(state)
	if vmParams.Revision > newestSupportedRevision {
		return state, &tosca.ErrUnsupportedRevision{Revision: vmParams.Revision}
	}

	// No need to run anything that is not in a running state.
	if state.Status != st.Running {
		return state, nil
	}

	return evmzeroSteppable.StepN(vmParams, state, numSteps)
}
