package geth

import (
	"github.com/Fantom-foundation/Tosca/go/ct"
	"github.com/Fantom-foundation/Tosca/go/ct/st"
)

type ctAdapter struct {
}

func NewConformanceTestingTarget() ct.Evm {
	return ctAdapter{}
}

func (ctAdapter) StepN(state *st.State, numSteps int) (*st.State, error) {
	gethInterpreter, interpreterState, err := ConvertCtStateToGeth(state)

	if err != nil {
		return nil, err
	}

	for i := 0; i < numSteps && !interpreterState.Halted; i++ {
		if !gethInterpreter.interpreter.Step(interpreterState) {
			break
		}
	}

	state, err = ConvertGethToCtState(gethInterpreter, interpreterState)

	if err != nil {
		return nil, err
	}

	return state, nil
}
