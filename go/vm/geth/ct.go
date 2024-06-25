// Copyright (c) 2024 Fantom Foundation
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at fantom.foundation/bsl11.
//
// Change Date: 2028-4-16
//
// On the date above, in accordance with the Business Source License, use of
// this software will be governed by the GNU Lesser General Public License v3.

package geth

import (
	"fmt"

	"github.com/Fantom-foundation/Tosca/go/ct"
	"github.com/Fantom-foundation/Tosca/go/ct/common"
	"github.com/Fantom-foundation/Tosca/go/ct/st"
	"github.com/Fantom-foundation/Tosca/go/ct/utils"
	"github.com/Fantom-foundation/Tosca/go/vm"
	geth_common "github.com/ethereum/go-ethereum/common"
	geth_vm "github.com/ethereum/go-ethereum/core/vm"
	"github.com/holiman/uint256"
)

func NewConformanceTestingTarget() ct.Evm {
	return ctAdapter{}
}

type ctAdapter struct{}

func (a ctAdapter) StepN(state *st.State, numSteps int) (*st.State, error) {
	parameters := utils.ToVmParameters(state)
	if parameters.Revision > newestSupportedRevision {
		return state, &vm.ErrUnsupportedRevision{Revision: parameters.Revision}
	}

	// No need to run anything that is not in a running state.
	if state.Status != st.Running {
		return state, nil
	}

	evm, contract, stateDb := createGethInterpreterContext(parameters)
	stateDb.refund = uint64(state.GasRefund)

	evm.CallInterceptor = &callInterceptor{parameters, stateDb, state.ReadOnly}

	interpreterState := geth_vm.InterpreterState{
		Contract:           contract,
		ReadOnly:           state.ReadOnly,
		Input:              state.CallData.ToBytes(),
		Status:             geth_vm.Running,
		Pc:                 uint64(state.Pc),
		Stack:              convertCtStackToGethStack(state),
		Memory:             convertCtMemoryToGethMemory(state),
		LastCallReturnData: state.LastCallReturnData.ToBytes(),
	}

	interpreter := evm.Interpreter()
	for i := 0; i < numSteps && interpreterState.Status == geth_vm.Running; i++ {
		interpreter.(*geth_vm.EVMInterpreter).Step(&interpreterState)
	}

	// Update the resulting state.
	var err error
	state.Status, err = convertGethStatusToCtStatus(&interpreterState)
	if err != nil {
		return nil, err
	}
	if state.Status == st.Running {
		state.Pc = uint16(interpreterState.Pc)
	}

	state.Gas = vm.Gas(contract.Gas)
	state.GasRefund = vm.Gas(stateDb.GetRefund())
	state.Stack = convertGethStackToCtStack(&interpreterState, state.Stack)
	state.Memory = convertGethMemoryToCtMemory(&interpreterState)
	state.LastCallReturnData = common.NewBytes(interpreterState.LastCallReturnData)

	if interpreterState.ReturnData != nil {
		state.ReturnData = common.NewBytes(interpreterState.ReturnData)
	}

	return state, nil
}

func convertGethStatusToCtStatus(state *geth_vm.InterpreterState) (st.StatusCode, error) {
	switch state.Status {
	case geth_vm.Running:
		return st.Running, nil
	case geth_vm.Reverted:
		return st.Reverted, nil
	case geth_vm.Stopped:
		return st.Stopped, nil
	case geth_vm.Failed:
		return st.Failed, nil
	}
	return 0, fmt.Errorf("unable to convert geth status to ct status")
}

func convertCtMemoryToGethMemory(state *st.State) *geth_vm.Memory {
	data := state.Memory.Read(0, uint64(state.Memory.Size()))
	memory := geth_vm.NewMemory()
	// Set internal memory gas cost state so future grow operations compute the correct cost.
	geth_vm.MemoryGasCost(memory, uint64(len(data)))
	memory.Resize(uint64(len(data)))
	memory.Set(0, uint64(len(data)), data)
	return memory
}

func convertGethMemoryToCtMemory(state *geth_vm.InterpreterState) *st.Memory {
	memory := st.NewMemory()
	memory.Set(state.Memory.Data())
	return memory
}

func convertCtStackToGethStack(state *st.State) *geth_vm.Stack {
	stack := geth_vm.NewStack()
	for i := state.Stack.Size() - 1; i >= 0; i-- {
		val := state.Stack.Get(i).Uint256()
		stack.Push(&val)
	}
	return stack
}

func convertGethStackToCtStack(state *geth_vm.InterpreterState, stack *st.Stack) *st.Stack {
	stack.Resize(0)
	for i := 0; i < state.Stack.Len(); i++ {
		val := state.Stack.Data()[i]
		stack.Push(common.NewU256(val[3], val[2], val[1], val[0]))
	}
	return stack
}

type callInterceptor struct {
	parameters vm.Parameters
	stateDb    *stateDbAdapter
	static     bool
}

func (i *callInterceptor) makeCall(kind vm.CallKind, callParam vm.CallParameters) (vm.CallResult, error) {
	res, _ := i.parameters.Context.Call(kind, callParam)

	i.handleGasRefund(res.GasRefund)
	err := geth_vm.ErrExecutionReverted
	if res.Success {
		err = nil
	}
	return res, err
}

func (i *callInterceptor) Call(env *geth_vm.EVM, me geth_vm.ContractRef, addr geth_common.Address, data []byte, gas uint64, value *uint256.Int) ([]byte, uint64, error) {
	have := i.stateDb.GetBalance(me.Address())
	if value.Cmp(have) > 0 {
		return nil, gas, geth_vm.ErrInsufficientBalance
	}

	kind := vm.Call
	if i.static {
		kind = vm.StaticCall
	}

	res, err := i.makeCall(kind, vm.CallParameters{
		Sender:    vm.Address(me.Address()),
		Recipient: vm.Address(addr),
		Value:     vm.ValueFromUint256(value),
		Input:     data,
		Gas:       vm.Gas(gas),
	})
	return res.Output, uint64(res.GasLeft), err
}

func (i *callInterceptor) CallCode(env *geth_vm.EVM, me geth_vm.ContractRef, addr geth_common.Address, data []byte, gas uint64, value *uint256.Int) ([]byte, uint64, error) {
	kind := vm.CallCode

	have := i.stateDb.GetBalance(me.Address())
	if value.Cmp(have) > 0 {
		return nil, gas, geth_vm.ErrInsufficientBalance
	}

	res, err := i.makeCall(kind, vm.CallParameters{
		Sender:      vm.Address(me.Address()),
		Recipient:   vm.Address(me.Address()),
		Value:       vm.ValueFromUint256(value),
		Input:       data,
		CodeAddress: vm.Address(addr),
		Gas:         vm.Gas(gas),
	})

	return res.Output, uint64(res.GasLeft), err
}

func (i *callInterceptor) DelegateCall(env *geth_vm.EVM, me geth_vm.ContractRef, addr geth_common.Address, data []byte, gas uint64) ([]byte, uint64, error) {
	res, err := i.makeCall(vm.DelegateCall, vm.CallParameters{
		Sender:    i.parameters.Sender,
		Recipient: i.parameters.Recipient,
		Value:     i.parameters.Value,
		Input:     data,
		Gas:       vm.Gas(gas),
	})
	return res.Output, uint64(res.GasLeft), err
}

func (i *callInterceptor) StaticCall(env *geth_vm.EVM, me geth_vm.ContractRef, addr geth_common.Address, input []byte, gas uint64) ([]byte, uint64, error) {
	res, err := i.makeCall(vm.StaticCall, vm.CallParameters{
		Sender:    vm.Address(me.Address()),
		Recipient: vm.Address(addr),
		Input:     input,
		Gas:       vm.Gas(gas),
	})
	return res.Output, uint64(res.GasLeft), err
}

func (i *callInterceptor) Create(env *geth_vm.EVM, me geth_vm.ContractRef, code []byte, gas uint64, value *uint256.Int) ([]byte, geth_common.Address, uint64, error) {
	have := i.stateDb.GetBalance(me.Address())
	if value.Cmp(have) > 0 {
		return nil, geth_common.Address{}, gas, geth_vm.ErrInsufficientBalance
	}

	res, err := i.makeCall(vm.Create, vm.CallParameters{
		Sender: vm.Address(me.Address()),
		Value:  vm.ValueFromUint256(value),
		Gas:    vm.Gas(gas),
		Input:  code,
	})

	return res.Output, geth_common.Address(res.CreatedAddress), uint64(res.GasLeft), err

}

func (i *callInterceptor) Create2(env *geth_vm.EVM, me geth_vm.ContractRef, code []byte, gas uint64, value *uint256.Int, salt *uint256.Int) ([]byte, geth_common.Address, uint64, error) {
	have := i.stateDb.GetBalance(me.Address())
	if value.Cmp(have) > 0 {
		return nil, geth_common.Address{}, gas, geth_vm.ErrInsufficientBalance
	}

	res, err := i.makeCall(vm.Create2, vm.CallParameters{
		Sender: vm.Address(me.Address()),
		Value:  vm.ValueFromUint256(value),
		Gas:    vm.Gas(gas),
		Input:  code,
		Salt:   salt.Bytes32(),
	})

	return res.Output, geth_common.Address(res.CreatedAddress), uint64(res.GasLeft), err
}

func (i *callInterceptor) handleGasRefund(refund vm.Gas) {
	if refund < 0 {
		i.stateDb.SubRefund(uint64(-refund))
	} else {
		i.stateDb.AddRefund(uint64(refund))
	}
}
