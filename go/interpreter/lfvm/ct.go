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
	"github.com/Fantom-foundation/Tosca/go/ct"
	"github.com/Fantom-foundation/Tosca/go/ct/common"
	"github.com/Fantom-foundation/Tosca/go/ct/st"
	"github.com/Fantom-foundation/Tosca/go/ct/utils"
	"github.com/Fantom-foundation/Tosca/go/tosca"
	lru "github.com/hashicorp/golang-lru/v2"
)

func NewConformanceTestingTarget() ct.Evm {

	// Can only fail for invalid configuration. Configuration is hardcoded.
	sanctionedVm, _ := NewInterpreter(Config{})

	// can only fail for non-positive size
	cache, _ := lru.New[[32]byte, *pcMap](4096)

	return &ctAdapter{
		vm:         sanctionedVm,
		pcMapCache: cache,
	}
}

type ctAdapter struct {
	vm         *lfvm
	pcMapCache *lru.Cache[[32]byte, *pcMap]
}

func (a *ctAdapter) StepN(state *st.State, numSteps int) (*st.State, error) {
	params := utils.ToVmParameters(state)
	if params.Revision > newestSupportedRevision {
		return state, &tosca.ErrUnsupportedRevision{Revision: params.Revision}
	}

	// No need to run anything that is not in a running state.
	if state.Status != st.Running {
		return state, nil
	}

	converted := a.vm.converter.Convert(
		params.Code,
		params.CodeHash,
	)

	pcMap := a.getPcMap(state.Code)

	memory := convertCtMemoryToLfvmMemory(state.Memory)

	// Set up execution context.
	var ctxt = &context{
		pc:           int32(pcMap.evmToLfvm[state.Pc]),
		params:       params,
		context:      params.Context,
		gas:          params.Gas,
		refund:       tosca.Gas(state.GasRefund),
		stack:        convertCtStackToLfvmStack(state.Stack),
		memory:       memory,
		code:         converted,
		returnData:   state.LastCallReturnData.ToBytes(),
		withShaCache: a.vm.config.WithShaCache,
	}

	defer func() {
		ReturnStack(ctxt.stack)
	}()

	// Run interpreter.
	status := statusRunning
	for i := 0; status == statusRunning && i < numSteps; i++ {
		status = execute(ctxt, true)
	}

	// Update the resulting state.
	state.Status = convertLfvmStatusToCtStatus(status)

	if status == statusRunning {
		state.Pc = pcMap.lfvmToEvm[ctxt.pc]
	}

	state.Gas = ctxt.gas
	state.GasRefund = ctxt.refund
	state.Stack = convertLfvmStackToCtStack(ctxt.stack, state.Stack)
	state.Memory = convertLfvmMemoryToCtMemory(ctxt.memory)
	state.LastCallReturnData = common.NewBytes(ctxt.returnData)
	if status == statusReturned || status == statusReverted {
		state.ReturnData = common.NewBytes(ctxt.returnData)
	}

	return state, nil
}

func (a *ctAdapter) getPcMap(code *st.Code) *pcMap {
	hash := code.Hash()
	pcMap, found := a.pcMapCache.Get(hash)
	if found {
		return pcMap
	}
	byteCode := code.Copy()
	pcMap = genPcMap(byteCode)
	a.pcMapCache.Add(hash, pcMap)
	return pcMap
}

// pcMap is a bidirectional map to map program counters between evm <-> lfvm.
type pcMap struct {
	evmToLfvm []uint16
	lfvmToEvm []uint16
}

// genPcMap creates a bidirectional program counter map for a given code,
// allowing mapping from a program counter in evm code to lfvm and vice versa.
func genPcMap(code []byte) *pcMap {
	evmToLfvm := make([]uint16, len(code)+1)
	lfvmToEvm := make([]uint16, len(code)+1)

	config := ConversionConfig{
		WithSuperInstructions: false,
	}
	res := convertWithObserver(code, config, func(evm, lfvm int) {
		evmToLfvm[evm] = uint16(lfvm)
		lfvmToEvm[lfvm] = uint16(evm)
	})

	// A program counter may correctly point to the position after the last
	// instruction, which would lead to an implicit STOP.
	evmToLfvm[len(code)] = uint16(len(res))

	// The LFVM code could also be longer than the input code if extra padding
	// of truncated PUSH instructions has been added.
	if len(res)+1 > len(lfvmToEvm) {
		lfvmToEvm = append(lfvmToEvm, make([]uint16, len(res)+1-len(lfvmToEvm))...)
	}
	lfvmToEvm[len(res)] = uint16(len(code))

	// Locations pointing to JUMP_TO instructions in LFVM need to be updated to
	// the position of the jump target.
	for i := 0; i < len(res); i++ {
		if res[i].opcode == JUMP_TO {
			lfvmToEvm[i] = res[i].arg
		}
	}

	return &pcMap{
		evmToLfvm: evmToLfvm,
		lfvmToEvm: lfvmToEvm,
	}
}

func convertLfvmStatusToCtStatus(status status) st.StatusCode {
	switch status {
	case statusRunning:
		return st.Running
	case statusReturned, statusStopped:
		return st.Stopped
	case statusReverted:
		return st.Reverted
	case statusSelfDestructed:
		return st.Stopped
	case statusFailed:
		return st.Failed
	}
	return st.Failed
}

func convertCtStackToLfvmStack(stack *st.Stack) *stack {
	result := NewStack()
	for i := stack.Size() - 1; i >= 0; i-- {
		val := stack.Get(i).Uint256()
		result.push(&val)
	}
	return result
}

func convertLfvmStackToCtStack(stack *stack, result *st.Stack) *st.Stack {
	len := stack.len()
	result.Resize(len)
	for i := 0; i < len; i++ {
		result.Set(len-i-1, common.NewU256FromUint256(stack.get(i)))
	}
	return result
}

func convertCtMemoryToLfvmMemory(memory *st.Memory) *Memory {
	data := memory.Read(0, uint64(memory.Size()))
	mem := NewMemory()
	words := tosca.SizeInWords(uint64(len(data)))
	mem.store = make([]byte, words*32)
	copy(mem.store, data)
	mem.currentMemoryCost = tosca.Gas((words*words)/512 + (3 * words))
	return mem
}

func convertLfvmMemoryToCtMemory(memory *Memory) *st.Memory {
	result := st.NewMemory()
	result.Set(memory.store)
	return result
}
