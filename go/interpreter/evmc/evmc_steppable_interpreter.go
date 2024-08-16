// Copyright (c) 2024 Fantom Foundation
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at fantom.foundation/bsl11.
//
// Change Date: 2028-4-16
//
// On the date above, in accordance with the Business Source License, use of
// this software will be governed by the GNU Lesser General Public License v3.

package evmc

import (
	"fmt"

	"github.com/Fantom-foundation/Tosca/go/ct/common"
	"github.com/Fantom-foundation/Tosca/go/ct/st"
	"github.com/Fantom-foundation/Tosca/go/tosca"
	"github.com/ethereum/evmc/v11/bindings/go/evmc"
)

// LoadSteppableEvmcInterpreter attempts to load an Interpreter implementation from a
// given library. The `library` parameter should name the library file, while
// the actual path to the library should be enforced using an rpath (see evmone
// implementation for an example).
func LoadSteppableEvmcInterpreter(library string) (*SteppableEvmcInterpreter, error) {
	vm, err := evmc.LoadSteppable(library)
	if err != nil {
		return nil, err
	}
	return &SteppableEvmcInterpreter{vm: vm}, nil
}

type SteppableEvmcInterpreter struct {
	vm *evmc.VMSteppable
}

func (e *SteppableEvmcInterpreter) StepN(
	params tosca.Parameters,
	state *st.State,
	numSteps int,
) (*st.State, error) {

	host_ctx := hostContext{
		params:  params,
		context: params.Context,
	}

	host_ctx.evmcBlobHashes = make([]evmc.Hash, 0, len(params.BlobHashes))
	for _, hash := range params.BlobHashes {
		host_ctx.evmcBlobHashes = append(host_ctx.evmcBlobHashes, evmc.Hash(hash))
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
		LastCallResult: state.LastCallReturnData.ToBytes(),
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

	if result.StepStatusCode == evmc.Returned || result.StepStatusCode == evmc.Reverted {
		state.ReturnData = common.NewBytes(result.Output)
	}
	state.Pc = uint16(result.Pc)
	state.Gas = tosca.Gas(result.GasLeft)
	state.GasRefund = tosca.Gas(result.GasRefund)
	state.Memory = convertEvmcMemoryToCtMemory(result.Memory)
	state.Stack, err = convertEvmcStackToCtStack(result.Stack, state.Stack)
	state.LastCallReturnData = common.NewBytes(result.LastCallReturnData)
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
	case evmc.Reverted:
		return st.Reverted, nil
	case evmc.Returned:
		return st.Stopped, nil
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

func convertEvmcStackToCtStack(stack []byte, result *st.Stack) (*st.Stack, error) {
	if len(stack)%32 != 0 {
		return nil, fmt.Errorf("stack size is not a multiple of 32")
	}
	result.Resize(0)
	for i := 0; i <= len(stack)-32; i += 32 {
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
