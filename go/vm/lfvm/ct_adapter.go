package lfvm

import (
	"sync"

	"github.com/Fantom-foundation/Tosca/go/ct"
	"github.com/Fantom-foundation/Tosca/go/ct/st"
)

type ctAdapter struct{}

func NewConformanceTestingTarget() ct.Evm {
	return ctAdapter{}
}

var pcMapCache = struct {
	maxSize int
	data    map[[32]byte]*PcMap
	mutex   sync.Mutex
}{
	maxSize: 4096,
	data:    make(map[[32]byte]*PcMap),
}

func getPcMap(code *st.Code) (*PcMap, error) {
	pcMapCache.mutex.Lock()
	defer pcMapCache.mutex.Unlock()

	if len(pcMapCache.data) > pcMapCache.maxSize {
		pcMapCache.data = make(map[[32]byte]*PcMap)
	}

	pcMap, ok := pcMapCache.data[code.Hash()]

	if !ok {
		byteCode := make([]byte, code.Length())
		code.CopyTo(byteCode)
		pcMap, err := GenPcMapWithoutSuperInstructions(byteCode)
		if err != nil {
			return nil, err
		}
		pcMapCache.data[code.Hash()] = pcMap
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
