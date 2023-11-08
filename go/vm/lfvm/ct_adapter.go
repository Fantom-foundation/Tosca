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
	c, err := ConvertCtStateToLfvmContext(state)
	if err != nil {
		return nil, err
	}

	for i := 0; c.status == RUNNING && i < numSteps; i++ {
		step(c)
	}

	return ConvertLfvmContextToCtState(c, state.Code)
}
