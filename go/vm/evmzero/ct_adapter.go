package evmzero

import (
	"github.com/Fantom-foundation/Tosca/go/ct"
	"github.com/Fantom-foundation/Tosca/go/ct/st"
)

type ctAdapter struct{}

func NewConformanceTestingTarget() ct.Evm {
	return ctAdapter{}
}

func (ctAdapter) StepN(state *st.State, numSteps int) (*st.State, error) {
	return CreateEvaluation(state).Run(numSteps)
}
