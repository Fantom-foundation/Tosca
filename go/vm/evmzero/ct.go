package evmzero

import (
	"fmt"

	"github.com/Fantom-foundation/Tosca/go/ct"
	"github.com/Fantom-foundation/Tosca/go/ct/common"
	"github.com/Fantom-foundation/Tosca/go/ct/st"
	"github.com/Fantom-foundation/Tosca/go/ct/utils"

	vmc "github.com/Fantom-foundation/Tosca/go/common"
)

var evmzeroSteppable *vmc.EvmcVMSteppable

func init() {
	vmSteppable, err := vmc.LoadEvmcVMSteppable("libevmzero.so")
	if err != nil {
		panic(fmt.Errorf("failed to load evmzero library: %s", err))
	}
	evmzeroSteppable = vmSteppable
}

func NewConformanceTestingTarget() ct.Evm {
	return ctAdapter{}
}

type ctAdapter struct{}

func (a ctAdapter) StepN(state *st.State, numSteps int) (*st.State, error) {
	// TODO: generalize

	// Hack: Special handling for unknown revision, because evmzero cannot represent an invalid revision.
	// So we mark the status as failed already.
	// TODO: Fix this once we add full revision support to the CT and evmzero.
	if state.Revision > common.R10_London {
		state.Status = st.Failed
		return state, nil
	}
	return evmzeroSteppable.StepN(utils.ToVmParameters(state), state, numSteps)
}
