// Copyright (c) 2024 Fantom Foundation
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at fantom.foundation/bsl11.
//
// Change Date: 2028-4-16
//
// On the date above, in accordance with the Business Source License, use of
// this software will be governed by the GNU Lesser General Public License v3.

package ct

import (
	"fmt"

	"github.com/Fantom-foundation/Tosca/go/ct"
	"github.com/Fantom-foundation/Tosca/go/ct/common"
	"github.com/Fantom-foundation/Tosca/go/ct/st"
	"github.com/Fantom-foundation/Tosca/go/ct/utils"
	"github.com/Fantom-foundation/Tosca/go/interpreter/lfvm"
	"github.com/Fantom-foundation/Tosca/go/tosca"
	lru "github.com/hashicorp/golang-lru/v2"
	"github.com/holiman/uint256"
)

func NewConformanceTestingTarget() ct.Evm {
	// can only produce an error if ConversionConfig.CacheSize is set to
	// less than maxCachedCodeLength = 24_576 bytes, but default is 1GiB,
	// so this can't fail.
	converter, err := lfvm.NewConverter(lfvm.ConversionConfig{
		WithSuperInstructions: false,
	})
	if err != nil {
		panic("failed to create converter: " + err.Error())
	}
	cache, _ := lru.New[[32]byte, *pcMap](4096) // can only fail for non-positive size
	return &ctAdapter{
		converter:  converter,
		pcMapCache: cache,
	}
}

type ctAdapter struct {
	converter  *lfvm.Converter
	pcMapCache *lru.Cache[[32]byte, *pcMap]
}

func (a *ctAdapter) StepN(state *st.State, numSteps int) (*st.State, error) {
	params := utils.ToVmParameters(state)
	if params.Revision > lfvm.NewestSupportedRevision {
		return state, &tosca.ErrUnsupportedRevision{Revision: params.Revision}
	}

	// No need to run anything that is not in a running state.
	if state.Status != st.Running {
		return state, nil
	}

	converted := a.converter.Convert(
		params.Code,
		params.CodeHash,
	)

	pcMap := a.getPcMap(state.Code)

	memory, err := convertCtMemoryToLfvmMemory(state.Memory)
	if err != nil {
		return nil, err
	}

	// Set up execution context.
	var ctxt = lfvm.NewContext(
		params,         // params
		params.Context, //context
		converted,      // code
		lfvm.StatusRunning,
		int32(pcMap.evmToLfvm[state.Pc]),       // pc:
		params.Gas,                             // gas
		tosca.Gas(state.GasRefund),             // refund
		convertCtStackToLfvmStack(state.Stack), // stack
		memory,
		state.LastCallReturnData.ToBytes(), // returnData
		*uint256.NewInt(0), *uint256.NewInt(0), false,
	)

	defer func() {
		lfvm.ReturnStack(ctxt)
	}()

	// Run interpreter.
	for i := 0; ctxt.IsRunning() && i < numSteps; i++ {
		lfvm.Step(ctxt)
	}

	result, err := lfvm.GetOutput(ctxt)
	if err != nil {
		ctxt.SignalOutofGas()
	}

	// Update the resulting state.
	state.Status, err = convertLfvmStatusToCtStatus(ctxt.GetStatus())
	if err != nil {
		return nil, err
	}
	if ctxt.IsRunning() {
		state.Pc = pcMap.lfvmToEvm[ctxt.GetPc()]
	}

	state.Gas = ctxt.GetGas()
	state.GasRefund = ctxt.GetRefund()
	state.Stack = convertLfvmStackToCtStack(ctxt.GetStack(), state.Stack)
	state.Memory = convertLfvmMemoryToCtMemory(ctxt.GetMemory())
	state.LastCallReturnData = common.NewBytes(ctxt.GetReturnData())
	state.ReturnData = common.NewBytes(result)

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

	config := lfvm.ConversionConfig{
		WithSuperInstructions: false,
	}
	res := lfvm.ConvertWithObserver(code, config, func(evm, lfvm int) {
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
		// if res[i].opcode == lfvm.JUMP_TO {
		if res.IsIndexOp(i, lfvm.JUMP_TO) {
			// lfvmToEvm[i] = res[i].arg
			lfvmToEvm[i] = res.GetArgOf(i)
		}
	}

	return &pcMap{
		evmToLfvm: evmToLfvm,
		lfvmToEvm: lfvmToEvm,
	}
}

func convertLfvmStatusToCtStatus(status lfvm.Status) (st.StatusCode, error) {
	switch status {
	case lfvm.StatusRunning:
		return st.Running, nil
	case lfvm.StatusReturned, lfvm.StatusStopped:
		return st.Stopped, nil
	case lfvm.StatusReverted:
		return st.Reverted, nil
	case lfvm.StatusSelfDestructed:
		return st.Stopped, nil
	case lfvm.StatusInvalidInstruction, lfvm.StatusOutOfGas, lfvm.StatusError:
		return st.Failed, nil
	default:
		return st.Failed, fmt.Errorf("unable to convert lfvm status %v to ct status", status)
	}
}

func convertCtStackToLfvmStack(stack *st.Stack) *lfvm.Stack {
	result := lfvm.NewStack()
	for i := stack.Size() - 1; i >= 0; i-- {
		val := stack.Get(i).Uint256()
		result.Push(&val)
	}
	return result
}

func convertLfvmStackToCtStack(stack *lfvm.Stack, result *st.Stack) *st.Stack {
	len := stack.Len()
	result.Resize(len)
	for i := 0; i < len; i++ {
		result.Set(len-i-1, common.NewU256FromUint256(&stack.Data()[i]))
	}
	return result
}

func convertCtMemoryToLfvmMemory(memory *st.Memory) (*lfvm.Memory, error) {
	data := memory.Read(0, uint64(memory.Size()))

	result := lfvm.NewMemory()
	err := result.SetWithCapacityCheck(0, uint64(len(data)), data)
	return result, err
}

func convertLfvmMemoryToCtMemory(memory *lfvm.Memory) *st.Memory {
	result := st.NewMemory()
	result.Set(memory.GetSlice(0, memory.Len()))
	return result
}
