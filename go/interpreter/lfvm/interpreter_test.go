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
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"reflect"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/Fantom-foundation/Tosca/go/tosca"
	"github.com/holiman/uint256"
	"go.uber.org/mock/gomock"
)

func TestContext_useGas_ReturnsErrorIfOutOfGasOrNegativeCost(t *testing.T) {
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

func TestInterpreter_CanDispatchExecutableInstructions(t *testing.T) {

	for _, op := range allOpCodesWhere(isExecutable) {
		t.Run(op.String(), func(t *testing.T) {
			forEachRevision(t, op, func(t *testing.T, revision tosca.Revision) {

				ctrl := gomock.NewController(t)
				mock := tosca.NewMockRunContext(ctrl)
				// mock all to satisfy any instruction
				mock.EXPECT().AccessAccount(gomock.Any()).Return(tosca.WarmAccess).AnyTimes()
				mock.EXPECT().GetBalance(gomock.Any()).AnyTimes()
				mock.EXPECT().GetNonce(gomock.Any()).AnyTimes()
				mock.EXPECT().GetCodeSize(gomock.Any()).AnyTimes()
				mock.EXPECT().GetCode(gomock.Any()).AnyTimes()
				mock.EXPECT().AccountExists(gomock.Any()).AnyTimes()
				mock.EXPECT().AccessStorage(gomock.Any(), gomock.Any()).AnyTimes()
				mock.EXPECT().GetStorage(gomock.Any(), gomock.Any()).AnyTimes()
				mock.EXPECT().SetStorage(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
				mock.EXPECT().Call(gomock.Any(), gomock.Any()).AnyTimes()
				mock.EXPECT().EmitLog(gomock.Any()).AnyTimes()
				mock.EXPECT().SelfDestruct(gomock.Any(), gomock.Any()).AnyTimes()
				mock.EXPECT().GetTransientStorage(gomock.Any(), gomock.Any()).AnyTimes()
				mock.EXPECT().SetTransientStorage(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()

				ctx := context{
					params: tosca.Parameters{
						BlockParameters: tosca.BlockParameters{
							Revision: revision,
						},
					},
					context: mock,
					// enough gas to satisfy any instruction
					gas:    1 << 32,
					stack:  NewStack(),
					memory: NewMemory(),
					code:   generateCodeFor(op),
				}
				err := fillStackFor(op, ctx.stack, ctx.code)
				if err != nil {
					t.Fatalf("unexpected creating stack: %v", err)
				}

				_, err = vanillaRunner{}.run(&ctx)
				if err != nil {
					t.Errorf("execution failed: %v", err)
				}
			})
		})
	}
}

func TestInterpreter_ExecutionTerminates(t *testing.T) {

	tests := map[string]struct {
		code []Instruction
	}{
		"empty code":          {code: []Instruction{}},
		"single stop":         {code: []Instruction{{STOP, 0}}},
		"pc bigger than code": {code: []Instruction{{PUSH1, 0}}},
		"revert":              {code: []Instruction{{REVERT, 0}}},
		"return":              {code: []Instruction{{RETURN, 0}}},
		"selfdestruct":        {code: []Instruction{{SELFDESTRUCT, 0}}},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {

			ctxt := getEmptyContext()
			ctxt.code = test.code
			ctxt.stack.push(uint256.NewInt(1))
			ctxt.stack.push(uint256.NewInt(2))
			ctxt.stack.push(uint256.NewInt(3))
			// runcontext is needed for selfdestruct
			mockContext := tosca.NewMockRunContext(gomock.NewController(t))
			mockContext.EXPECT().GetBalance(gomock.Any()).Return(tosca.Value{1}).AnyTimes()
			mockContext.EXPECT().GetNonce(gomock.Any()).Return(uint64(1)).AnyTimes()
			mockContext.EXPECT().SelfDestruct(gomock.Any(), gomock.Any()).Return(true).AnyTimes()
			ctxt.context = mockContext

			status, err := steps(&ctxt, false)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if status == statusRunning {
				t.Errorf("failed to terminate execution, status is %v", status)
			}
		})
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
	_, err := run(config{}, params, code)
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

func TestInterpreter_EmptyCodeBypassesRunnerAndSucceeds(t *testing.T) {
	code := []Instruction{}
	params := tosca.Parameters{}
	config := config{
		runner: NewMockrunner(gomock.NewController(t)),
	}

	result, err := run(config, params, code)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if !result.Success {
		t.Errorf("unexpected result: want success, got %v", result.Success)
	}
}

func TestInterpreter_run_ReturnsErrorOnRuntimeError(t *testing.T) {

	runner := NewMockrunner(gomock.NewController(t))
	code := []Instruction{{JUMPDEST, 0}}
	params := tosca.Parameters{Gas: 20}
	config := config{runner: runner}

	expectedError := fmt.Errorf("runtime error")

	runner.EXPECT().run(gomock.Any()).Return(statusFailed, expectedError)

	_, err := run(config, params, code)
	if !errors.Is(err, expectedError) {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestRun_GenerateResult(t *testing.T) {

	baseOutput := []byte{0x1, 0x2, 0x3}
	baseGas := tosca.Gas(2)
	baseRefund := tosca.Gas(3)

	tests := map[string]struct {
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
		"failure": {
			status: statusFailed,
			expectedResult: tosca.Result{
				Success: false,
			},
		},
		"unknown status": {
			status:         statusFailed + 1,
			expectedErr:    fmt.Errorf("unexpected error in interpreter, unknown status: %v", statusFailed+1),
			expectedResult: tosca.Result{},
		},
	}

	for name, test := range tests {
		t.Run(fmt.Sprintf("%v", name), func(t *testing.T) {

			ctxt := context{}
			ctxt.refund = baseRefund
			ctxt.gas = baseGas
			ctxt.returnData = bytes.Clone(baseOutput)

			res, err := generateResult(test.status, &ctxt)

			if test.expectedErr != nil && strings.Compare(err.Error(), test.expectedErr.Error()) != 0 {
				t.Errorf("unexpected error: want \"%v\", got \"%v\"", test.expectedErr, err)
			}
			if !reflect.DeepEqual(res, test.expectedResult) {
				t.Errorf("unexpected result: want %v, got %v", test.expectedResult, res)
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

func TestSteps_DetectsNonExecutableCode(t *testing.T) {

	nonExecutableOpCodes := []OpCode{
		INVALID,
		NOOP,
		DATA,
	}
	undefinedOpCodeRegex := regexp.MustCompile(`^op\(0x[0-9a-fA-F]+\)`)
	isUndefined :=
		func(op OpCode) bool {
			return undefinedOpCodeRegex.MatchString(op.String())
		}
	nonExecutableOpCodes = append(nonExecutableOpCodes, allOpCodesWhere(isUndefined)...)

	for _, opCode := range nonExecutableOpCodes {
		t.Run(opCode.String(), func(t *testing.T) {
			ctxt := getEmptyContext()
			ctxt.code = []Instruction{{opCode, 0}}

			_, err := steps(&ctxt, false)
			if want, got := errInvalidOpCode, err; want != got {
				t.Errorf("unexpected error: want %v, got %v", want, got)
			}
		})
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

func TestSteps_FailsWithLessGasThanStaticCost(t *testing.T) {

	for _, op := range allOpCodes() {
		t.Run(op.String(), func(t *testing.T) {
			forEachRevision(t, op, func(t *testing.T, revision tosca.Revision) {

				expectedGas := getStaticGasPrices(revision).get(op)
				if expectedGas == 0 {
					t.Skip("operation has static cost zero")
				}

				ctxt := getEmptyContext()
				ctxt.code = []Instruction{{op, 0}}
				ctxt.stack.stackPointer = 20
				ctxt.gas = expectedGas - 1

				_, err := steps(&ctxt, false)
				if want, got := errOutOfGas, err; want != got {
					t.Errorf("unexpected error: want %v, got %v", want, got)
				}
			})
		})
	}
}

func TestInterpreter_InstructionsFailWhenExecutedInRevisionsEarlierThanIntroducedIn(t *testing.T) {
	for _, op := range allOpCodes() {
		introducedIn := _introducedIn.get(op)
		for revision := tosca.R07_Istanbul; revision < introducedIn; revision++ {
			t.Run(fmt.Sprintf("%v/%v", op, revision), func(t *testing.T) {
				ctxt := getEmptyContext()
				ctxt.code = []Instruction{{op, 0}}
				ctxt.params.BlockParameters.Revision = revision
				ctxt.stack.stackPointer = 20

				_, err := steps(&ctxt, false)
				if want, got := errInvalidRevision, err; want != got {
					t.Errorf("unexpected error: want %v, got %v", want, got)
				}
			})
		}
	}
}

func TestInterpreter_ExecuteReturnsFailureOnExecutionError(t *testing.T) {

	ctxt := context{
		code:  generateCodeFor(INVALID),
		stack: NewStack(),
	}

	status := execute(&ctxt, false)
	if want, got := statusFailed, status; want != got {
		t.Errorf("unexpected status: want %v, got %v", want, got)
	}
}

////////////////////////////////////////////////////////////////////////////////
// Benchmarks

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

////////////////////////////////////////////////////////////////////////////////
// test utilities

func Test_generateCodeForOps(t *testing.T) {
	tests := map[OpCode]int{
		PUSH1:                     1,
		PUSH2:                     1,
		PUSH3:                     2,
		PUSH4:                     2,
		PUSH5:                     3,
		PUSH6:                     3,
		PUSH31:                    16,
		PUSH32:                    16,
		PUSH1_PUSH1:               1,
		PUSH1_PUSH4_DUP3:          3,
		PUSH1_PUSH1_PUSH1_SHL_SUB: 2,
	}
	for op, test := range tests {
		t.Run(op.String(), func(t *testing.T) {
			code := generateCodeFor(op)
			if want, got := test, len(code); want != got {
				t.Errorf("expected %d instructions, got %d", want, got)
			}
		})
	}
}

// generateCodeFor generates valid LFVM code for one instruction.
// Appends necessary DATA instructions to the code to satisfy stack requirements.
// Adds JUMPDEST instruction after JUMP instructions.
func generateCodeFor(op OpCode) Code {

	var code []Instruction

	switch op {
	case PUSH1_PUSH4_DUP3:
		code = append(code, Instruction{op, 0}, Instruction{DATA, 0})
	case PUSH1_PUSH1_PUSH1_SHL_SUB:
		code = append(code, Instruction{op, 0}, Instruction{DATA, 0})
	case PUSH2_JUMP:
		code = append(code, Instruction{op, 1}) // hardcoded jump destination
	case PUSH2_JUMPI:
		code = append(code, Instruction{op, 1}) // hardcoded jump destination
	default:
		code = append(code, Instruction{op, 0})
	}

	for _, op := range append(op.decompose(), op) {
		if PUSH3 <= op && op <= PUSH32 {
			n := int(op) - int(PUSH3) + 3
			numInstructions := n/2 + n%2
			for i := 0; i < numInstructions-1; i++ {
				code = append(code, Instruction{DATA, 0})
			}
		}
	}

	if isJump(op) {
		code = append(code, Instruction{JUMPDEST, 0})
	}

	if op == JUMP_TO {
		// prevent endless loop by having jump to itself
		code[0].arg = uint16(len(code))
		code = append(code, Instruction{JUMPDEST, 0})
	}

	return code
}

// fillStackFor fills the stack with the required number of elements for the given opcode.
// For Jump instructions, it also encodes the PC for the the first jump destination found in code
func fillStackFor(op OpCode, stack *stack, code Code) error {
	limits := _precomputedStackLimits.get(op)
	stack.stackPointer = limits.min

	// jump instructions need a valid jump destination
	if isJump(op) {
		counter := slices.IndexFunc(code, func(v Instruction) bool {
			return v.opcode == JUMPDEST
		})
		if counter == -1 {
			return fmt.Errorf("missing JUMPDEST instruction")
		}

		for i := 0; i < limits.min; i++ {
			stack.data[i] = *uint256.NewInt(uint64(counter))
		}
	}

	return nil
}

var _isUndefinedOpCodeRegex = regexp.MustCompile(`^op\(0x[0-9A-Fa-f]+\)$`)

func isExecutable(op OpCode) bool {
	if slices.Contains([]OpCode{INVALID, NOOP, DATA}, op) {
		return false
	}
	return !_isUndefinedOpCodeRegex.MatchString(op.String())
}

func isJump(op OpCode) bool {
	ops := append(op.decompose(), op)
	return slices.ContainsFunc(ops, func(op OpCode) bool {
		return op == JUMP || op == JUMPI
	})
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

		res := ctxt.returnData
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

var _introducedIn = newOpCodePropertyMap(func(op OpCode) tosca.Revision {
	switch op {
	case BASEFEE:
		return tosca.R10_London
	case PUSH0:
		return tosca.R12_Shanghai
	case BLOBHASH:
		return tosca.R13_Cancun
	case BLOBBASEFEE:
		return tosca.R13_Cancun
	case TLOAD:
		return tosca.R13_Cancun
	case TSTORE:
		return tosca.R13_Cancun
	case MCOPY:
		return tosca.R13_Cancun
	}
	return tosca.R07_Istanbul
})

// forEachRevision runs a test for each revision starting from the revision
// where the operation was introduced.
// It creates a new testing scope to name the test after the revision.
func forEachRevision(
	t *testing.T, op OpCode,
	f func(t *testing.T, revision tosca.Revision)) {

	for revision := tosca.R07_Istanbul; revision <= newestSupportedRevision; revision++ {
		if revision < _introducedIn.get(op) {
			continue
		}
		t.Run(revision.String(), func(t *testing.T) {
			f(t, revision)
		})
	}
}
