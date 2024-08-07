// Copyright (c) 2024 Fantom Foundation
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at fantom.foundation/bsl11.
//
// Change Date: 2028-4-16
//
// On the date above, in accordance with the Business Source License, use of
// this software will be governed by the GNU Lesser General Public License v3.

package lfvm

import (
	"fmt"
	"sync"

	"github.com/Fantom-foundation/Tosca/go/ct"
	"github.com/Fantom-foundation/Tosca/go/ct/common"
	"github.com/Fantom-foundation/Tosca/go/ct/st"
	"github.com/Fantom-foundation/Tosca/go/ct/utils"
	"github.com/Fantom-foundation/Tosca/go/tosca"
)

func NewConformanceTestingTarget() ct.Evm {
	return ctAdapter{}
}

type ctAdapter struct{}

func (a ctAdapter) StepN(state *st.State, numSteps int) (*st.State, error) {
	params := utils.ToVmParameters(state)
	if params.Revision > newestSupportedRevision {
		return state, &tosca.ErrUnsupportedRevision{Revision: params.Revision}
	}

	// No need to run anything that is not in a running state.
	if state.Status != st.Running {
		return state, nil
	}

	var codeHash tosca.Hash
	if params.CodeHash != nil {
		codeHash = *params.CodeHash
	}

	converted, err := Convert(
		params.Code,
		false, /* no super instructions */
		params.CodeHash == nil,
		false, /* with code cache */
		codeHash,
	)
	if err != nil {
		return nil, err
	}

	pcMap, err := getPcMap(state.Code)
	if err != nil {
		return nil, err
	}

	memory, err := convertCtMemoryToLfvmMemory(state.Memory)
	if err != nil {
		return nil, err
	}

	// Set up execution context.
	var ctxt = &context{
		pc:          int32(pcMap.evmToLfvm[state.Pc]),
		params:      params,
		context:     params.Context,
		gas:         params.Gas,
		refund:      tosca.Gas(state.GasRefund),
		stack:       convertCtStackToLfvmStack(state.Stack),
		memory:      memory,
		status:      RUNNING,
		code:        converted,
		revision:    params.Revision,
		return_data: state.LastCallReturnData.ToBytes(),
	}

	defer func() {
		ReturnStack(ctxt.stack)
	}()

	// Run interpreter.
	for i := 0; ctxt.status == RUNNING && i < numSteps; i++ {
		step(ctxt)
	}

	result, err := getResult(ctxt)
	if err != nil {
		ctxt.status = OUT_OF_GAS
	}

	// Update the resulting state.
	state.Status, err = convertLfvmStatusToCtStatus(ctxt.status)
	if err != nil {
		return nil, err
	}
	if ctxt.status == RUNNING {
		var ok bool
		state.Pc, ok = pcMap.lfvmToEvm[uint16(ctxt.pc)]
		if !ok {
			return nil, fmt.Errorf("failed to convert program counter %d", ctxt.pc)
		}
	}

	state.Gas = ctxt.gas
	state.GasRefund = ctxt.refund
	state.Stack = convertLfvmStackToCtStack(ctxt.stack, state.Stack)
	state.Memory = convertLfvmMemoryToCtMemory(ctxt.memory)
	state.LastCallReturnData = common.NewBytes(ctxt.return_data)
	state.ReturnData = common.NewBytes(result)

	return state, nil
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
		byteCode := code.Copy()
		pcMap, err := GenPcMapWithoutSuperInstructions(byteCode)
		if err != nil {
			return nil, err
		}
		pcMapCache.data[code.Hash()] = pcMap
		return pcMap, nil
	}

	return pcMap, nil
}

func convertLfvmStatusToCtStatus(status Status) (st.StatusCode, error) {
	switch status {
	case RUNNING:
		return st.Running, nil
	case RETURNED, STOPPED:
		return st.Stopped, nil
	case REVERTED:
		return st.Reverted, nil
	case SUICIDED:
		// Suicide is not yet modeled by the CT, and for now it just maps to the STOPPED status.
		return st.Stopped, nil
	case INVALID_INSTRUCTION, OUT_OF_GAS, SEGMENTATION_FAULT, MAX_INIT_CODE_SIZE_EXCEEDED, ERROR:
		return st.Failed, nil
	default:
		return st.Failed, fmt.Errorf("unable to convert lfvm status %v to ct status", status)
	}
}

func convertCtStackToLfvmStack(stack *st.Stack) *Stack {
	result := NewStack()
	for i := stack.Size() - 1; i >= 0; i-- {
		val := stack.Get(i).Uint256()
		result.push(&val)
	}
	return result
}

func convertLfvmStackToCtStack(stack *Stack, result *st.Stack) *st.Stack {
	len := stack.len()
	result.Resize(len)
	for i := 0; i < len; i++ {
		result.Set(len-i-1, common.NewU256FromUint256(&stack.Data()[i]))
	}
	return result
}

func convertCtMemoryToLfvmMemory(memory *st.Memory) (*Memory, error) {
	data := memory.Read(0, uint64(memory.Size()))

	result := NewMemory()
	err := result.SetWithCapacityCheck(0, uint64(len(data)), data)
	return result, err
}

func convertLfvmMemoryToCtMemory(memory *Memory) *st.Memory {
	result := st.NewMemory()
	result.Set(memory.GetSlice(0, memory.Len()))
	return result
}
