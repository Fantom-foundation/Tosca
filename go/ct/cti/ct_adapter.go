package cti

import (
	"errors"
	"math"

	"github.com/Fantom-foundation/Tosca/go/ct"
	"github.com/holiman/uint256"
)

type CtAdapter struct {
	state State
}

func (evm *CtAdapter) StepN(init ct.State, numSteps int) (ct.State, error) {
	evm.state = decodeCtState(init)
	evm.state.StepN(numSteps)
	return encodeCtState(evm.state)
}

func decodeCtState(input ct.State) (output State) {
	// ct.Failed maps to cti.Invalid
	output.Status = Status(input.Status)
	output.Static = input.Static

	output.Pc = int(input.Pc)
	output.GasLeft = input.Gas

	output.Code = make([]OpCode, len(input.Code))
	for i := range input.Code {
		output.Code[i] = OpCode(input.Code[i])
	}

	output.Stack = make([]uint256.Int, input.Stack.Size())
	for i := range output.Stack {
		output.Stack[i] = input.Stack.Get(input.Stack.Size() - 1 - i)
	}

	output.Memory = input.Memory.ReadFrom(0, uint64(input.Memory.Size()))

	output.host = &adapterHost{
		storage: input.Storage.ToMap(),
	}

	return
}

func encodeCtState(input State) (output ct.State, err error) {
	if input.Status < Invalid {
		output.Status = ct.StatusCode(input.Status)
	} else {
		output.Status = ct.Failed
	}
	output.Static = input.Static

	if input.Pc > math.MaxUint16 {
		return output, errors.New("program counter out of range")
	}
	output.Pc = uint16(input.Pc)
	output.Gas = input.GasLeft

	output.Code = make([]byte, len(input.Code))
	for i := range input.Code {
		output.Code[i] = byte(input.Code[i])
	}

	for _, v := range input.Stack {
		output.Stack.Push(v)
	}

	output.Memory.Set(input.Memory)

	adapterHost, ok := input.host.(*adapterHost)
	if !ok {
		return output, errors.New("unable to convert generic host into CT state")
	}
	for k, v := range adapterHost.storage {
		output.Storage.Set(k, v)
	}

	return
}

type adapterHost struct {
	storage map[uint256.Int]uint256.Int
}

func (h *adapterHost) GetStorage(key uint256.Int) uint256.Int {
	return h.storage[key]
}

func (h *adapterHost) SetStorage(key uint256.Int, value uint256.Int) {
	h.storage[key] = value
}
