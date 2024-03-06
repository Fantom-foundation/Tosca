package common

import (
	"fmt"

	"github.com/Fantom-foundation/Tosca/go/ct/common"
	"github.com/Fantom-foundation/Tosca/go/ct/st"
	"github.com/Fantom-foundation/Tosca/go/vm"
	"github.com/ethereum/evmc/v10/bindings/go/evmc"
)

// LoadEvmcVMSteppable attempts to load an EVM implementation from a given library.
// The `library` parameter should name the library file, while the actual
// path to the library should be enforced using an rpath (see evmone
// implementation for an example).
func LoadEvmcVMSteppable(library string) (*EvmcVMSteppable, error) {
	vm, err := evmc.LoadSteppable(library)
	if err != nil {
		return nil, err
	}
	return &EvmcVMSteppable{vm: vm}, nil
}

type EvmcVMSteppable struct {
	vm *evmc.VMSteppable
}

func (e *EvmcVMSteppable) StepN(
	params vm.Parameters,
	state *st.State,
	numSteps int,
) (*st.State, error) {
	host_ctx := HostContext{
		params:  params,
		context: params.Context,
	}

	revision, err := toEvmcRevision(params.Revision)
	if err != nil {
		return nil, err
	}

	var codeHash *evmc.Hash
	if params.CodeHash != nil {
		codeHash = new(evmc.Hash)
		*codeHash = evmc.Hash(*params.CodeHash)
	}

	stepStatus, err := convertCtStatusToEvmcStatus(state.Status)
	if err != nil {
		return nil, err
	}

	result, err := e.vm.StepN(evmc.StepParameters{
		Context:        &host_ctx,
		Revision:       revision,
		Kind:           evmc.Call,
		Static:         params.Static,
		Depth:          params.Depth,
		Gas:            int64(state.Gas),
		GasRefund:      int64(state.GasRefund),
		Recipient:      evmc.Address(params.Recipient),
		Sender:         evmc.Address(params.Sender),
		Input:          params.Input,
		Value:          evmc.Hash(params.Value),
		CodeHash:       codeHash,
		Code:           params.Code,
		StepStatusCode: stepStatus,
		Pc:             uint64(state.Pc),
		Stack:          convertCtStackToEvmcStack(state.Stack),
		Memory:         convertCtMemoryToEvmcMemory(state.Memory),
		NumSteps:       numSteps,
	})
	if err != nil {
		return nil, err
	}

	// Process results into a result state.
	state.Status, err = convertEvmcStatusToCtStatus(result.StepStatusCode)
	if err != nil {
		return nil, err
	}
	state.Pc = uint16(result.Pc)
	state.Gas = uint64(result.GasLeft)
	state.GasRefund = uint64(result.GasRefund)
	state.Memory = convertEvmcMemoryToCtMemory(result.Memory)
	state.Stack, err = convertEvmcStackToCtStack(result.Stack)
	if err != nil {
		return nil, err
	}

	return state, nil
}

func convertCtStatusToEvmcStatus(status st.StatusCode) (evmc.StepStatus, error) {
	switch status {
	case st.Running:
		return evmc.Running, nil
	case st.Stopped:
		return evmc.Stopped, nil
	case st.Returned:
		return evmc.Returned, nil
	case st.Reverted:
		return evmc.Reverted, nil
	case st.Failed:
		return evmc.Failed, nil
	}
	return evmc.Failed, fmt.Errorf("unknown status code: %v", status)
}

func convertEvmcStatusToCtStatus(stepStatus evmc.StepStatus) (st.StatusCode, error) {
	switch stepStatus {
	case evmc.Running:
		return st.Running, nil
	case evmc.Stopped:
		return st.Stopped, nil
	case evmc.Returned:
		return st.Returned, nil
	case evmc.Reverted:
		return st.Reverted, nil
	case evmc.Failed:
		return st.Failed, nil
	}
	return st.Failed, fmt.Errorf("unknown status code: %v", stepStatus)
}

func convertCtStackToEvmcStack(stack *st.Stack) []byte {
	stackBytes := stack.Size() * 32
	evmcStack := make([]byte, stackBytes)
	for i := stack.Size() - 1; i >= 0; i-- {
		val := stack.Get(i).Bytes32be()
		copy(evmcStack[stackBytes-(i+1)*32:], val[:])
	}
	return evmcStack
}

func convertEvmcStackToCtStack(stack []byte) (*st.Stack, error) {
	if len(stack)%32 != 0 {
		return nil, fmt.Errorf("stack size is not a multiple of 32")
	}
	result := st.NewStack()
	for i := len(stack) - 32; i >= 0; i -= 32 {
		val := common.NewU256FromBytes(stack[i : i+32]...)
		result.Push(val)
	}
	return result, nil
}

func convertCtMemoryToEvmcMemory(memory *st.Memory) []byte {
	return memory.Read(0, uint64(memory.Size()))
}

func convertEvmcMemoryToCtMemory(memory []byte) *st.Memory {
	result := st.NewMemory()
	result.Set(memory)
	return result
}
