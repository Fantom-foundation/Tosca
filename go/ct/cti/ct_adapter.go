package cti

import (
	"bytes"
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

	// TODO: this should be deep-copied; but for efficiency this is skipped
	output.host = &adapterHost{
		storage:       input.Storage.ToMap(),
		recordedCalls: input.PastCalls,
		futureResults: input.FutureResults,
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
	// TODO: this should be deep-copied; but for efficiency this is skipped
	output.PastCalls = adapterHost.recordedCalls
	output.FutureResults = adapterHost.futureResults

	return
}

type adapterHost struct {
	storage       map[uint256.Int]uint256.Int
	futureResults []ct.CallResult
	recordedCalls []ct.CallDescription
}

func (h *adapterHost) GetStorage(key uint256.Int) uint256.Int {
	return h.storage[key]
}

func (h *adapterHost) SetStorage(key uint256.Int, value uint256.Int) {
	h.storage[key] = value
}

func (h *adapterHost) Call(
	gasSent uint256.Int,
	address uint256.Int,
	value uint256.Int,
	message []byte,
) (
	success bool,
	gasLeft uint256.Int,
	result []byte,
) {
	// TODO: handle mismatches of expectations and actual calls better
	if len(h.futureResults) == 0 {
		panic("unexpected call -- no expectation set")
	}

	callResult := h.futureResults[0]
	h.futureResults = h.futureResults[1:]

	success = callResult.Success
	gasLeft = callResult.GasLeft
	result = bytes.Clone(callResult.Response)

	h.recordedCalls = append(h.recordedCalls, ct.CallDescription{
		GasSent: gasSent,
		Address: address,
		Value:   value,
		Message: bytes.Clone(message),
		Result: ct.CallResult{
			Success:  success,
			GasLeft:  gasLeft,
			Response: bytes.Clone(result),
		},
	})

	return
}
