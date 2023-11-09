package lfvm

import (
	"github.com/Fantom-foundation/Tosca/go/ct"
	"github.com/Fantom-foundation/Tosca/go/ct/st"
)

type ctAdapter struct{}

func NewConformanceTestingTarget() ct.Evm {
	return ctAdapter{}
}

var pcMapCache = map[[32]byte]*PcMap{}

func getPcMap(code *st.Code) (*PcMap, error) {
	pcMap, ok := pcMapCache[code.Hash()]

	if !ok {
		byteCode := make([]byte, code.Length())
		code.CopyTo(byteCode)
		pcMap, err := GenPcMapWithoutSuperInstructions(byteCode)
		if err != nil {
			return nil, err
		}
		pcMapCache[code.Hash()] = pcMap
		return pcMap, nil
	}

	return pcMap, nil
}

func (ctAdapter) StepN(state *st.State, numSteps int) (*st.State, error) {
	pcMap, err := getPcMap(state.Code)
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
