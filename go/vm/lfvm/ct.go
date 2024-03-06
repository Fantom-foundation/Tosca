package lfvm

import (
	"fmt"
	"sync"

	"github.com/Fantom-foundation/Tosca/go/ct"
	"github.com/Fantom-foundation/Tosca/go/ct/common"
	"github.com/Fantom-foundation/Tosca/go/ct/st"
	"github.com/Fantom-foundation/Tosca/go/ct/utils"
	"github.com/Fantom-foundation/Tosca/go/vm"
)

func NewConformanceTestingTarget() ct.Evm {
	return ctAdapter{}
}

type ctAdapter struct{}

func (a ctAdapter) StepN(state *st.State, numSteps int) (*st.State, error) {
	// No need to run everything that is not in a running state.
	if state.Status != st.Running {
		return state, nil
	}

	params := utils.ToVmParameters(state)

	var codeHash vm.Hash
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

	memory, err := convertCtMemoryToLfvmMemory(state)
	if err != nil {
		return nil, err
	}

	// Set up execution context.
	var ctxt = &context{
		pc:       int32(pcMap.evmToLfvm[state.Pc]),
		params:   params,
		context:  params.Context,
		gas:      params.Gas,
		refund:   vm.Gas(state.GasRefund),
		stack:    convertCtStackToLfvmStack(state.Stack),
		memory:   memory,
		status:   RUNNING,
		code:     converted,
		isBerlin: params.Revision >= vm.R09_Berlin,
		isLondon: params.Revision >= vm.R10_London,
	}

	defer func() {
		ReturnStack(ctxt.stack)
	}()

	// Run interpreter.
	for i := 0; ctxt.status == RUNNING && i < numSteps; i++ {
		step(ctxt)
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
	state.Gas = uint64(ctxt.gas)
	state.GasRefund = uint64(ctxt.refund)
	state.Stack = convertLfvmStackToCtStack(ctxt.stack)
	state.Memory = convertLfvmMemoryToCtMemory(ctxt)

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

func convertLfvmStatusToCtStatus(status Status) (st.StatusCode, error) {
	switch status {
	case RUNNING:
		return st.Running, nil
	case STOPPED:
		return st.Stopped, nil
	case REVERTED:
		return st.Reverted, nil
	case RETURNED:
		return st.Returned, nil
	case SUICIDED:
		// Suicide is not yet modeled by the CT, and for now it just maps to the STOPPED status.
		return st.Stopped, nil
	case INVALID_INSTRUCTION:
		return st.Failed, nil
	case OUT_OF_GAS:
		return st.Failed, nil
	case SEGMENTATION_FAULT:
		return st.Failed, nil
	case ERROR:
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

func convertLfvmStackToCtStack(stack *Stack) *st.Stack {
	result := st.NewStack()
	for i := 0; i < stack.len(); i++ {
		val := stack.Data()[i]
		result.Push(common.NewU256(val[3], val[2], val[1], val[0]))
	}
	return result
}

func convertCtMemoryToLfvmMemory(state *st.State) (*Memory, error) {
	data := state.Memory.Read(0, uint64(state.Memory.Size()))

	memory := NewMemory()
	memory.EnsureCapacityWithoutGas(uint64(len(data)), nil)
	err := memory.Set(0, uint64(len(data)), data)
	return memory, err
}

func convertLfvmMemoryToCtMemory(ctx *context) *st.Memory {
	memory := st.NewMemory()
	memory.Set(ctx.memory.GetSlice(0, ctx.memory.Len()))
	return memory
}
