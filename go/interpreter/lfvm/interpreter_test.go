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
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log"
	"math"
	"os"
	"reflect"
	"regexp"
	"strings"
	"testing"

	"github.com/Fantom-foundation/Tosca/go/tosca"
	"github.com/holiman/uint256"
	"go.uber.org/mock/gomock"
)

func TestContext_useGas_ReturnsErrorIfOutOfGasAndConsumesAllRemainingGas(t *testing.T) {
	tests := map[string]struct {
		available tosca.Gas
		required  tosca.Gas
	}{
		"no available gas and no gas required":      {0, 0},
		"no available gas":                          {0, 100},
		"no available gas and infinite need":        {0, -1},
		"gas available and infinite need":           {100, -100},
		"gas available with no need":                {100, 0},
		"sufficient gas":                            {100, 10},
		"insufficient gas":                          {10, 100},
		"all gas":                                   {100, 100},
		"almost all gas":                            {100, 99},
		"one unit too much":                         {100, 101},
		"negative available gas":                    {-100, 100},
		"negative available gas with infinite need": {-100, -100},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			ctx := context{
				gas: test.available,
			}
			err := ctx.useGas(test.required)

			// Check that the result of UseGas indicates whether there was
			// enough gas.
			want := test.required >= 0 && test.available >= test.required
			success := err == nil
			if want != success {
				t.Errorf("expected UseGas to return %v, got %v", want, success)
			}

			// Check that the remaining gas is correct.
			wantGas := tosca.Gas(0)
			if err == nil {
				wantGas = test.available - test.required
			}
			if ctx.gas != wantGas {
				t.Errorf("expected gas to be %v, got %v", wantGas, ctx.gas)
			}
		})
	}
}

func TestContext_isAtLeast_RespectsOrderOfRevisions(t *testing.T) {
	revisions := []tosca.Revision{
		tosca.R07_Istanbul,
		tosca.R09_Berlin,
		tosca.R10_London,
		tosca.R11_Paris,
		tosca.R12_Shanghai,
		tosca.R13_Cancun,
	}

	for _, is := range revisions {
		context := context{
			params: tosca.Parameters{
				BlockParameters: tosca.BlockParameters{
					Revision: is,
				},
			},
		}

		for _, trg := range revisions {
			if want, got := is >= trg, context.isAtLeast(trg); want != got {
				t.Errorf("revision %v should be at least %v: %t, got %t", is, trg, want, got)
			}
		}
	}

}

type example struct {
	code     []byte // Some contract code
	function uint32 // The identifier of the function in the contract to be called
}

const MAX_STACK_SIZE int = 1024
const GAS_START tosca.Gas = 1 << 32

func getEmptyContext() context {
	code := make([]Instruction, 0)
	data := make([]byte, 0)
	return getContext(code, data, nil, 0, GAS_START, tosca.R07_Istanbul)
}

func getContext(code Code, data []byte, runContext tosca.RunContext, stackPtr int, gas tosca.Gas, revision tosca.Revision) context {

	// Create execution context.
	ctxt := context{
		params: tosca.Parameters{
			BlockParameters: tosca.BlockParameters{
				Revision: revision,
			},
			Gas:   gas,
			Input: data,
		},
		context: runContext,
		gas:     gas,
		stack:   NewStack(),
		memory:  NewMemory(),
		code:    code,
	}

	// Move the stack pointer to the required hight.
	// For the tests using the resulting context the actual
	// stack content is not relevant. It is merely used for
	// checking stack over- or under-flows.
	ctxt.stack.stackPointer = stackPtr

	return ctxt
}

func TestInterpreter_step_DetectsLowerStackLimitViolation(t *testing.T) {
	// Add tests for execution

	for _, op := range allOpCodes() {

		usage := computeStackUsage(op)
		if usage.from >= 0 {
			continue
		}

		ctxt := getEmptyContext()
		ctxt.code = []Instruction{{op, 0}}

		_, err := steps(&ctxt, false)
		if want, got := errStackUnderflow, err; want != got {
			t.Errorf("expected stack-underflow for %v to be detected, got %v", op, got)
		}
	}
}

func TestInterpreter_step_DetectsUpperStackLimitViolation(t *testing.T) {
	// Add tests for execution
	for _, op := range allOpCodes() {
		// Ignore operations that do not need any data on the stack.
		usage := computeStackUsage(op)
		if usage.to <= 0 {
			continue
		}

		ctxt := getEmptyContext()
		ctxt.code = []Instruction{{op, 0}}
		ctxt.stack.stackPointer = maxStackSize

		_, err := steps(&ctxt, false)
		if want, got := errStackOverflow, err; want != got {
			t.Errorf("expected stack-underflow for %v to be detected, got %v", op, got)
		}
	}
}

type OpcodeTest struct {
	name        string
	code        []Instruction
	stackPtrPos int
	argData     uint16
	isBerlin    bool // < TODO: replace with revision
	isLondon    bool
	mockCalls   func(*tosca.MockRunContext)
	gasStart    tosca.Gas
	gasConsumed tosca.Gas
	gasRefund   tosca.Gas
	endStatus   status
	err         error
}

type OpCodeWithGas struct {
	OpCode
	gas tosca.Gas
}

func generateOpCodesInRange(start OpCode, end OpCode) []OpCode {
	opCodes := make([]OpCode, end-start+1)
	for i := start; i <= end; i++ {
		opCodes[i-start] = i
	}
	return opCodes
}

// TODO: redo, static gas
func TestInstructionsGasConsumption(t *testing.T) {

	var tests []OpcodeTest

	for _, op := range generateOpCodesInRange(PUSH1, PUSH32) {
		code := []Instruction{{op, 1}}
		dataNum := int((op - PUSH1) / 2)
		for j := 0; j < dataNum; j++ {
			code = append(code, Instruction{DATA, 1})
		}
		tests = append(tests, OpcodeTest{op.String(), code, 20, 0, false, false, nil, GAS_START, 3, 0, statusStopped, nil})
	}

	attachGasTo := func(gas tosca.Gas, opCodes ...OpCode) []OpCodeWithGas {
		opCodesWithGas := make([]OpCodeWithGas, len(opCodes))
		for i, opCode := range opCodes {
			opCodesWithGas[i] = OpCodeWithGas{opCode, gas}
		}
		return opCodesWithGas
	}

	var opCodes []OpCodeWithGas
	opCodes = append(opCodes, attachGasTo(2, generateOpCodesInRange(COINBASE, CHAINID)...)...)
	opCodes = append(opCodes, attachGasTo(3, ADD, SUB)...)
	opCodes = append(opCodes, attachGasTo(5, MUL, DIV, SDIV, MOD, SMOD, SIGNEXTEND)...)
	opCodes = append(opCodes, attachGasTo(3, generateOpCodesInRange(DUP1, DUP16)...)...)
	opCodes = append(opCodes, attachGasTo(3, generateOpCodesInRange(SWAP1, SWAP16)...)...)
	opCodes = append(opCodes, attachGasTo(3, generateOpCodesInRange(LT, SAR)...)...)
	opCodes = append(opCodes, attachGasTo(8, ADDMOD, MULMOD)...)
	opCodes = append(opCodes, attachGasTo(10, EXP)...)
	opCodes = append(opCodes, attachGasTo(30, SHA3)...)
	opCodes = append(opCodes, attachGasTo(11, SWAP1_POP_SWAP2_SWAP1)...)
	opCodes = append(opCodes, attachGasTo(10, POP_SWAP2_SWAP1_POP)...)
	opCodes = append(opCodes, attachGasTo(4, POP_POP)...)
	opCodes = append(opCodes, attachGasTo(6, generateOpCodesInRange(PUSH1_SHL, PUSH1_DUP1)...)...)
	// opCodes = append(opCodes, applyGasTo(11, PUSH2_JUMP)...) // FIXME: this seems to be broken
	opCodes = append(opCodes, attachGasTo(13, PUSH2_JUMPI)...)
	opCodes = append(opCodes, attachGasTo(6, PUSH1_PUSH1)...)
	opCodes = append(opCodes, attachGasTo(5, SWAP1_POP)...)
	opCodes = append(opCodes, attachGasTo(6, SWAP2_SWAP1)...)
	opCodes = append(opCodes, attachGasTo(5, SWAP2_POP)...)
	opCodes = append(opCodes, attachGasTo(9, DUP2_MSTORE)...)
	opCodes = append(opCodes, attachGasTo(6, DUP2_LT)...)

	for _, opCode := range opCodes {
		code := []Instruction{{opCode.OpCode, 0}}
		tests = append(tests, OpcodeTest{
			name:        opCode.String(),
			code:        code,
			stackPtrPos: 20,
			endStatus:   statusStopped,
			gasStart:    GAS_START,
			gasConsumed: opCode.gas,
		})
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			runContext := tosca.NewMockRunContext(ctrl)
			if test.mockCalls != nil {
				test.mockCalls(runContext)
			}
			revision := tosca.R07_Istanbul
			if test.isBerlin {
				revision = tosca.R09_Berlin
			}
			if test.isLondon {
				revision = tosca.R10_London
			}
			ctxt := getContext(test.code, make([]byte, 0), runContext, test.stackPtrPos, test.gasStart, revision)

			// Run testing code
			status, err := vanillaRunner{}.run(&ctxt)
			if want, got := test.err, err; want != got {
				t.Errorf("unexpected return: error is %v, wanted %v", got, want)
			}

			if want, got := test.endStatus, status; want != got {
				t.Errorf("execution failed: status is %v, wanted %v", got, want)
			}

			// Check gas consumption
			if want, got := test.gasConsumed, test.gasStart-ctxt.gas; want != got {
				t.Errorf("execution failed: gas consumption is %v, wanted %v", got, want)
			}

			// Check gas refund
			if want, got := test.gasRefund, ctxt.refund; want != got {
				t.Errorf("execution failed: gas refund is %v, wanted %v", got, want)
			}
		})
	}
}

func TestInterpreter_Run_ReturnsEmptyResultOnEmptyCode(t *testing.T) {
	// Get tosca.Parameters
	params := tosca.Parameters{
		Input:  []byte{},
		Static: true,
		Gas:    10,
	}
	code := make([]Instruction, 0)

	// Run testing code
	result, err := run(interpreterConfig{}, params, code)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if result.Output != nil {
		t.Errorf("unexpected output: want nil, got %v", result.Output)
	}
	if want, got := params.Gas, result.GasLeft; want != got {
		t.Errorf("unexpected gas left: want %v, got %v", want, got)
	}
	if !result.Success {
		t.Errorf("unexpected success: want true, got false")
	}
}

func TestInterpreter_Logging_PrintsToSTDOUT(t *testing.T) {
	code := []Instruction{
		{PUSH1, 1},
		{STOP, 0},
	}

	params := tosca.Parameters{
		Input:  []byte{},
		Static: true,
		Gas:    10,
		Code:   []byte{0x0},
	}

	// redirect stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Run testing code
	_, err := run(interpreterConfig{
		runner: loggingRunner{},
	}, params, code)
	// read the output
	_ = w.Close() // ignore error in test
	out, _ := io.ReadAll(r)
	os.Stdout = old

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// check the output
	if !strings.Contains(string(out), "STOP") {
		t.Errorf("unexpected output: want STOP, got %v", string(out))
	}
}

func TestInterpreter_Vanilla_RunsWithoutOutput(t *testing.T) {

	code := []Instruction{
		{PUSH1, 1},
		{STOP, 0},
	}

	params := tosca.Parameters{
		Input:  []byte{},
		Static: true,
		Gas:    10,
		Code:   []byte{0x0},
	}

	// redirect stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Run testing code
	_, err := run(interpreterConfig{}, params, code)
	// read the output
	_ = w.Close() // ignore error in test
	out, _ := io.ReadAll(r)
	os.Stdout = old

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if len(string(out)) != 0 {
		t.Errorf("unexpected output: want \"\", got %v", string(out))
	}
}

func TestRunGenerateResult(t *testing.T) {

	baseOutput := []byte{0x1, 0x2, 0x3}
	baseGas := tosca.Gas(2)
	baseRefund := tosca.Gas(3)

	getCtxt := func() context {
		ctxt := getEmptyContext()
		ctxt.gas = baseGas
		ctxt.refund = baseRefund
		ctxt.memory = NewMemory()
		ctxt.memory.store = baseOutput
		ctxt.resultSize = uint256.Int{uint64(len(baseOutput))}
		return ctxt
	}

	tests := map[string]struct {
		setup          func(ctx *context)
		status         status
		expectedErr    error
		expectedResult tosca.Result
	}{
		"returned": {
			status: statusReturned,
			expectedResult: tosca.Result{
				Success:   true,
				Output:    baseOutput,
				GasLeft:   baseGas,
				GasRefund: baseRefund,
			},
		},
		"reverted": {
			status: statusReverted,
			expectedResult: tosca.Result{
				Success:   false,
				Output:    baseOutput,
				GasLeft:   baseGas,
				GasRefund: 0,
			},
		},
		"stopped": {
			status: statusStopped,
			expectedResult: tosca.Result{
				Success:   true,
				Output:    nil,
				GasLeft:   baseGas,
				GasRefund: baseRefund,
			},
		},
		"suicide": {
			status: statusSelfDestructed,
			expectedResult: tosca.Result{
				Success:   true,
				Output:    nil,
				GasLeft:   baseGas,
				GasRefund: baseRefund,
			},
		},
		"unknown status": {
			status:         statusSelfDestructed + 1,
			expectedErr:    fmt.Errorf("unexpected error in interpreter, unknown status: %v", statusSelfDestructed+1),
			expectedResult: tosca.Result{},
		},
		"getOutput fail": {
			setup: func(ctx *context) {
				ctx.resultSize = uint256.Int{1, 1}
			},
			expectedResult: tosca.Result{
				Success: false,
			},
		},
	}

	for name, test := range tests {
		t.Run(fmt.Sprintf("%v", name), func(t *testing.T) {

			ctxt := getCtxt()
			if test.setup != nil {
				test.setup(&ctxt)
			}

			res, err := generateResult(test.status, &ctxt)

			if err != nil && test.expectedErr != nil && strings.Compare(err.Error(), test.expectedErr.Error()) != 0 {
				t.Errorf("unexpected error: want \"%v\", got \"%v\"", test.expectedErr, err)
			}
			if !reflect.DeepEqual(res, test.expectedResult) {
				t.Errorf("unexpected result: want %v, got %v", test.expectedResult, res)
			}
		})
	}
}

func TestGetOutputReturnsExpectedErrors(t *testing.T) {

	tests := map[string]struct {
		setup       func(*context)
		expectedErr error
	}{
		"size overflow": {
			setup:       func(ctx *context) { ctx.resultSize = uint256.Int{1, 1} },
			expectedErr: errOverflow,
		},
		"offset overflow": {
			setup: func(ctx *context) {
				ctx.resultSize = uint256.Int{1}
				ctx.resultOffset = uint256.Int{1, 1}
			},
			expectedErr: errOverflow,
		},
		"memory overflow": {
			setup: func(ctx *context) {
				ctx.resultSize = uint256.Int{math.MaxUint64 - 1}
				ctx.resultOffset = uint256.Int{2}
			},
			expectedErr: errOverflow,
		},
	}

	for name, test := range tests {
		t.Run(fmt.Sprintf("%v", name), func(t *testing.T) {
			ctxt := getEmptyContext()
			test.setup(&ctxt)

			_, err := getOutput(statusReturned, &ctxt)
			if !errors.Is(err, test.expectedErr) {
				t.Errorf("unexpected error: want error, got nil")
			}
		})
	}
}

func TestStepsProperlyHandlesJUMP_TO(t *testing.T) {
	ctxt := getEmptyContext()
	instructions := []Instruction{
		{JUMP_TO, 0x02},
		{RETURN, 0},
		{STOP, 0},
	}

	ctxt.params = tosca.Parameters{
		Input:  []byte{},
		Static: false,
		Gas:    10,
		Code:   []byte{0x0},
	}
	ctxt.code = instructions

	status, err := steps(&ctxt, false)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if status != statusStopped {
		t.Errorf("unexpected status: want STOPPED, got %v", status)
	}
}

func TestStepsDetectsNonExecutableCode(t *testing.T) {
	nonExecutableOpCodes := []OpCode{
		INVALID,
		NOOP,
		DATA,
	}

	re := regexp.MustCompile(`^op\(0x[0-9a-fA-F]{2}\)`)
	for op := OpCode(0); op < numOpCodes; op++ {
		if re.MatchString(op.String()) {
			nonExecutableOpCodes = append(nonExecutableOpCodes, op)
		}
	}

	for _, opCode := range nonExecutableOpCodes {
		ctxt := getEmptyContext()
		ctxt.params = tosca.Parameters{
			Input:  []byte{},
			Static: false,
			Gas:    10,
			Code:   []byte{0x0},
		}
		ctxt.code = []Instruction{{opCode, 0}}

		_, err := steps(&ctxt, false)
		if want, got := errInvalidOpCode, err; want != got {
			t.Errorf("unexpected return: want %v, got %v", want, got)
		}
	}
}

func TestSteps_StaticContextViolation(t *testing.T) {
	tests := []struct {
		op          OpCode
		stack       []uint256.Int
		minRevision tosca.Revision
	}{
		{op: SSTORE},
		{op: LOG0},
		{op: LOG1},
		{op: LOG2},
		{op: LOG3},
		{op: LOG4},
		{op: CREATE},
		{op: CREATE2},
		{op: SELFDESTRUCT},
		{
			op:          TSTORE,
			minRevision: tosca.R13_Cancun,
		},
		{
			op: CALL,
			stack: []uint256.Int{
				{}, {}, {}, {},
				*uint256.NewInt(1), // value != 0: static violation
				{}, {},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.op.String(), func(t *testing.T) {
			ctxt := getEmptyContext()
			// Get tosca.Parameters
			ctxt.params = tosca.Parameters{
				Input:  []byte{},
				Static: true,
				Gas:    10,
				Code:   []byte{0x0},
			}
			ctxt.code = []Instruction{{test.op, 0}}
			ctxt.params.BlockParameters.Revision = test.minRevision

			if len(test.stack) == 0 {
				// add enough stack elements to pass stack bounds check
				ctxt.stack.stackPointer = 50
			} else {
				// otherwise prefill the stack with provided data
				copy(ctxt.stack.data[:len(test.stack)], test.stack)
				ctxt.stack.stackPointer = len(test.stack)
			}

			_, err := steps(&ctxt, false)
			if want, got := errStaticContextViolation, err; want != got {
				t.Errorf("unexpected error: want %v, got %v", want, got)
			}
		})
	}
}

// FIXME: rewrite as static gas check (for all opcodes)
func TestStepsFailsOnTooLittleGas(t *testing.T) {
	ctxt := getEmptyContext()
	instructions := []Instruction{
		{PUSH1, 0},
	}

	ctxt.params = tosca.Parameters{
		Input:  []byte{},
		Static: false,
		Gas:    2,
		Code:   []byte{0x0},
	}
	ctxt.gas = 2
	ctxt.code = instructions

	_, err := steps(&ctxt, false)
	if want, got := errNotEnoughStaticGas, err; want != got {
		t.Errorf("unexpected error: want %v, got %v", want, got)
	}
}

////////////////////////////////////////////////////////////////////////////////
// Benchmarks

// To run the benchmark use
//  go test ./core/vm/lfvm -bench=.*Fib.* --benchtime 10s

func BenchmarkFib10(b *testing.B) {
	benchmarkFib(b, 10, false)
}

func BenchmarkFib10_SI(b *testing.B) {
	benchmarkFib(b, 10, true)
}

func BenchmarkSatisfiesStackRequirements(b *testing.B) {
	context := &context{
		stack: NewStack(),
	}

	opCodes := allOpCodes()
	for i := 0; i < b.N; i++ {
		_ = checkStackLimits(context.stack.len(), opCodes[i%len(opCodes)])
	}
}

func benchmarkFib(b *testing.B, arg int, with_super_instructions bool) {
	example := getFibExample()

	// Convert example to LFVM format.
	converted := convert(example.code, ConversionConfig{WithSuperInstructions: with_super_instructions})

	// Create input data.

	// See details of argument encoding: t.ly/kBl6
	data := make([]byte, 4+32) // < the parameter is padded up to 32 bytes

	// Encode function selector in big-endian format.
	data[0] = byte(example.function >> 24)
	data[1] = byte(example.function >> 16)
	data[2] = byte(example.function >> 8)
	data[3] = byte(example.function)

	// Encode argument as a big-endian value.
	data[4+28] = byte(arg >> 24)
	data[5+28] = byte(arg >> 16)
	data[6+28] = byte(arg >> 8)
	data[7+28] = byte(arg)

	// Create execution context.
	ctxt := context{
		params: tosca.Parameters{
			Input:  data,
			Static: true,
		},
		gas:    1 << 62,
		code:   converted,
		stack:  NewStack(),
		memory: NewMemory(),
	}

	// Compute expected value.
	wanted := fib(arg)

	for i := 0; i < b.N; i++ {
		// Reset the context.
		ctxt.pc = 0
		ctxt.gas = 1 << 31
		ctxt.stack.stackPointer = 0

		// Run the code (actual benchmark).
		status, err := vanillaRunner{}.run(&ctxt)
		if err != nil {
			b.Fatalf("execution failed: %v", err)
		}

		if status != statusReturned {
			b.Fatalf("execution failed: status is %v", status)
		}

		size := ctxt.resultSize
		if size.Uint64() != 32 {
			b.Fatalf("unexpected length of end; wanted 32, got %d", size.Uint64())
		}
		res := make([]byte, size.Uint64())
		offset := ctxt.resultOffset

		data, err := ctxt.memory.getSlice(offset.Uint64(), size.Uint64(), &ctxt)
		if err != nil {
			b.Fatalf("unexpected error: %v", err)
		}
		copy(res, data)

		got := (int(res[28]) << 24) | (int(res[29]) << 16) | (int(res[30]) << 8) | (int(res[31]) << 0)
		if wanted != got {
			b.Fatalf("unexpected result, wanted %d, got %d", wanted, got)
		}
	}
}

func getFibExample() example {
	// An implementation of the fib function in EVM byte code.
	code, err := hex.DecodeString("608060405234801561001057600080fd5b506004361061002b5760003560e01c8063f9b7c7e514610030575b600080fd5b61004a600480360381019061004591906100f6565b610060565b6040516100579190610132565b60405180910390f35b600060018263ffffffff161161007957600190506100b0565b61008e600283610089919061017c565b610060565b6100a360018461009e919061017c565b610060565b6100ad91906101b4565b90505b919050565b600080fd5b600063ffffffff82169050919050565b6100d3816100ba565b81146100de57600080fd5b50565b6000813590506100f0816100ca565b92915050565b60006020828403121561010c5761010b6100b5565b5b600061011a848285016100e1565b91505092915050565b61012c816100ba565b82525050565b60006020820190506101476000830184610123565b92915050565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052601160045260246000fd5b6000610187826100ba565b9150610192836100ba565b9250828203905063ffffffff8111156101ae576101ad61014d565b5b92915050565b60006101bf826100ba565b91506101ca836100ba565b9250828201905063ffffffff8111156101e6576101e561014d565b5b9291505056fea26469706673582212207fd33e47e97ce5871bb05401e6710238af535ae8aeaab013ca9a9c29152b8a1b64736f6c637827302e382e31372d646576656c6f702e323032322e382e392b636f6d6d69742e62623161386466390058")
	if err != nil {
		log.Fatalf("Unable to decode fib-code: %v", err)
	}

	return example{
		code:     code,
		function: 0xF9B7C7E5, // The function selector for the fib function.
	}
}

func fib(x int) int {
	if x <= 1 {
		return 1
	}
	return fib(x-1) + fib(x-2)
}
