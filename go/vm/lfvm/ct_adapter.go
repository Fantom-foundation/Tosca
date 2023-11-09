package lfvm

import (
	"github.com/Fantom-foundation/Tosca/go/ct"
	"github.com/Fantom-foundation/Tosca/go/ct/st"
)

type ctAdapter struct{}

func NewConformanceTestingTarget() ct.Evm {
	return ctAdapter{}
}

func (ctAdapter) StepN(state *st.State, numSteps int) (*st.State, error) {
	code := make([]byte, state.Code.Length())
	state.Code.CopyTo(code)

	pcMap, err := GenPcMapWithoutSuperInstructions(code)
	if err != nil {
		return nil, err
	}

	c, err := ConvertCtStateToLfvmContext(state, pcMap)
	if err != nil {
		return nil, err
	}

	for i := 0; c.status == RUNNING && i < numSteps; i++ {
		step(c)
	}

	return ConvertLfvmContextToCtState(c, state.Code, pcMap)
}
