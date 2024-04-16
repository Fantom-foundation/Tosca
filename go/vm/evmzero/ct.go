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

package evmzero

import (
	"fmt"

	"github.com/Fantom-foundation/Tosca/go/ct"
	"github.com/Fantom-foundation/Tosca/go/ct/common"
	"github.com/Fantom-foundation/Tosca/go/ct/st"
	"github.com/Fantom-foundation/Tosca/go/ct/utils"
	"github.com/Fantom-foundation/Tosca/go/vm/evmc"
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
	// Hack: Special handling for unknown revision, because evmzero cannot represent an invalid revision.
	// So we mark the status as failed already.
	// TODO: Fix this once we add full revision support to the CT and evmzero.
	if state.Revision > common.R10_London {
		state.Status = st.Failed
		return state, nil
	}

	// No need to run anything that is not in a running state.
	if state.Status != st.Running {
		return state, nil
	}

	return evmzeroSteppable.StepN(utils.ToVmParameters(state), state, numSteps)
}
