package lfvm

import (
	"regexp"
	"testing"

	"github.com/Fantom-foundation/Tosca/go/tosca"
	"github.com/holiman/uint256"
	"go.uber.org/mock/gomock"
)

// testDeclaration is the type used to define test cases in this file.
type testDeclaration struct {
	// nameOverride is an alternative name for the test case. if empty,
	// the name for the first opcode in code is used.
	nameOverride string
	// code is a slice of instructions to be executed.
	code []Instruction
	// mockCalls allows to mock the context calls.
	mockCalls func(*tosca.MockRunContext)
	// statusOverride allows to define the expected status : If nil, statusStopped
	statusOverride *status
	// pcAfter allows to override the expected PC after execution. If nil, len(code) is expected.
	pcAfter *pcOverride
	// stack defines the stack state before and the size after execution.
	stack stackTest
	// revisionConstraint allows to define the revision constraint for the test. Default is every revision.
	revisionConstraint revisionConstraint
	// staticCtx allows to define if the context is static.
	staticCtx bool
	// gas defines the gas before and after execution.
	gas testParameter[tosca.Gas]
	// gasRefund defines the gas refund after execution.
	gasRefund tosca.Gas
}

// generateSingleTestCase generates a test case for each opcode.
// The instances in the slice test functionality provided by the interpreter.
// For the sake of simplicity, they do not test specific behavior of each opcode,
// but instead they evaluate correctness of the following parameters:
//   - Stack size before and after execution, where specific values are not
//     checked (e.g. binary operators stack size is 2 before and 1 after
//     execution, specific values vary with each case)
//   - PC position after execution
//   - Status after execution (stopped, returned, reverted, error...)
//   - Gas cost (both static and dynamic, since it's computation reasonability
//     is shared between the interpreter and opcodes implementation)
//   - Gas refound (if any)
//   - Memory TBD
//   - call data & return data TBD
func generateTestCases() []testDeclaration {

	tests := []testDeclaration{}

	///////////////////////////////////
	// Special opcodes

	tests = append(tests, []testDeclaration{
		{
			code:           []Instruction{{RETURN, 0}},
			stack:          stackSize(2, 0),
			statusOverride: expectReturned(),
			gas:            cost(0),
		},
		{
			code:           []Instruction{{REVERT, 0}},
			stack:          stackSize(2, 0),
			statusOverride: expectReverted(),
			gas:            cost(0),
		},
		{
			code:           []Instruction{{INVALID, 0}},
			statusOverride: expectInvalidInstruction(),
			pcAfter:        overridePc(0),
		},
		{
			code:           []Instruction{{NOOP, 0}},
			statusOverride: expectOutOfGas(),
			pcAfter:        overridePc(0),
		},
		{
			code:           []Instruction{{DATA, 0}},
			statusOverride: expectOutOfGas(),
			pcAfter:        overridePc(0),
		},
		{
			code: []Instruction{{STOP, 0}},
		},
	}...)

	///////////////////////////////////
	// trivial tests:
	// - no revision overrides
	// - no context access
	// - no dynamic gas
	// - execution is never aborted (under favorable gas and stack conditions)

	addTrivialTest := func(
		tests []testDeclaration,
		gas gas,
		stack stackTest,
		opCodes ...OpCode,
	) []testDeclaration {
		for _, op := range opCodes {
			code := []Instruction{{op, 0}}
			tests = append(tests, testDeclaration{
				code:  code,
				stack: stack,
				gas:   cost(gas),
			})
		}
		return tests
	}

	tests = addTrivialTest(tests, 1, invariantStack(), JUMPDEST)

	tests = addTrivialTest(tests, gas(3), stackSize(2, 1), ADD, SUB)
	tests = addTrivialTest(tests, gas(3), stackSize(2, 1), AND, OR, XOR, BYTE)
	tests = addTrivialTest(tests, gas(3), stackSize(2, 1), LT, GT, SLT, SGT, EQ)
	tests = addTrivialTest(tests, gas(5), stackSize(2, 1), MUL, DIV, SDIV, MOD, SMOD, SIGNEXTEND)
	tests = addTrivialTest(tests, gas(3), stackSize(2, 1), getOpCodesInRange(SHL, SAR)...)
	tests = addTrivialTest(tests, gas(60), stackSize(2, 1), EXP)
	tests = addTrivialTest(tests, gas(39), stackSize(2, 1), SHA3)

	tests = addTrivialTest(tests, gas(2), stackSize(0, 1), PC)
	tests = addTrivialTest(tests, gas(2), stackSize(0, 1), getOpCodesInRange(COINBASE, CHAINID)...)
	tests = addTrivialTest(tests, gas(2), stackSize(0, 1), GASPRICE, GAS)
	tests = addTrivialTest(tests, gas(3), stackSize(1, 1), ISZERO, NOT)

	tests = addTrivialTest(tests, gas(8), stackSize(3, 1), ADDMOD, MULMOD)
	tests = addTrivialTest(tests, gas(3), stackSize(16, 17), getOpCodesInRange(DUP1, DUP16)...)
	tests = addTrivialTest(tests, gas(3), stackSize(17, 17), getOpCodesInRange(SWAP1, SWAP16)...)

	///////////////////////////////////
	// Stack manipulation

	// PUSH1 - PUSH32
	tests = append(tests, generateTestCaseFor(
		func(op OpCode) testDeclaration {
			dataNum := int((op - PUSH1) / 2)
			code := attachDataToOp(op, dataNum)
			return testDeclaration{
				code:  code,
				stack: stackSize(0, 1),
				gas:   cost(3),
			}
		},
		getOpCodesInRange(PUSH1, PUSH32))...)

	// PUSH0
	tests = append(tests, testDeclaration{
		code:               []Instruction{{PUSH0, 0}},
		stack:              stackSize(0, 1),
		revisionConstraint: validFrom(tosca.R12_Shanghai),
		gas:                cost(2),
	})
	// POP
	tests = addTrivialTest(tests, gas(2), stackSize(1, 0), POP)

	///////////////////////////////////
	// SLOAD - STORE

	tests = append(tests, []testDeclaration{
		{
			nameOverride:       "SLOAD",
			code:               []Instruction{{SLOAD, 0}},
			stack:              stackSize(1, 1),
			revisionConstraint: validBefore(tosca.R09_Berlin),
			mockCalls: func(mock *tosca.MockRunContext) {
				mock.EXPECT().GetStorage(tosca.Address{}, toKey(1)).Return(toWord(0))
			},
			gas: cost(800),
		},
	}...)

	///////////////////////////////////
	// Super instructions (without jump)

	tests = addTrivialTest(tests, gas(4), stackSize(2, 0), POP_POP)
	tests = addTrivialTest(tests, gas(5), stackSize(2, 1), SWAP1_POP)
	tests = addTrivialTest(tests, gas(5), stackSize(3, 2), SWAP2_POP)
	tests = addTrivialTest(tests, gas(6), stackSize(1, 1), PUSH1_SHL, PUSH1_ADD)
	tests = addTrivialTest(tests, gas(6), stackSize(1, 3), PUSH1_DUP1)
	tests = addTrivialTest(tests, gas(6), stackSize(2, 2), DUP2_LT)
	tests = addTrivialTest(tests, gas(6), stackSize(3, 3), SWAP2_SWAP1)
	tests = addTrivialTest(tests, gas(6), stackSize(0, 2), PUSH1_PUSH1)
	tests = addTrivialTest(tests, gas(10), stackSize(5, 3), POP_SWAP2_SWAP1_POP)
	tests = addTrivialTest(tests, gas(11), stackSize(4, 3), SWAP1_POP_SWAP2_SWAP1)
	tests = addTrivialTest(tests, gas(12), stackSize(2, 1), DUP2_MSTORE)
	tests = addTrivialTest(tests, gas(14), stackSize(5, 3), AND_SWAP1_POP_SWAP2_SWAP1)

	testWithData := func(gas gas, stack stackTest, requiredDataOps int, opCodes ...OpCode) []testDeclaration {
		tests := make([]testDeclaration, len(opCodes))
		for i, op := range opCodes {
			code := attachDataToOp(op, requiredDataOps)
			tests[i] = testDeclaration{
				code:  code,
				stack: stack,
				gas:   cost(gas),
			}
		}
		return tests
	}

	tests = append(tests, testWithData(15, stackSize(0, 1), 1, PUSH1_PUSH1_PUSH1_SHL_SUB)...)
	tests = append(tests, testWithData(9, stackSize(1, 4), 2, PUSH1_PUSH4_DUP3)...)

	///////////////////////////////////
	// Jump instructions

	tests = append(tests, []testDeclaration{

		{
			nameOverride: "JUMP",
			code:         []Instruction{{PUSH1, 2 << 8}, {JUMP, 0}, {JUMPDEST, 0}},
			gas:          cost(3 + 8 + 1),
		},
		{
			nameOverride: "JUMPI",
			code:         []Instruction{{PUSH1, 1 << 8}, {PUSH1, 3 << 8}, {JUMPI, 0}, {JUMPDEST, 0}},
			gas:          cost(3 + 3 + 10 + 1),
		},
		{
			nameOverride: "POP_JUMP",
			code:         []Instruction{{PUSH1, 3 << 8}, {PUSH1, 0}, {POP_JUMP, 0}, {JUMPDEST, 0}},
			gas:          cost(3 + 3 + 2 + 8 + 1),
		},
		{
			nameOverride: "ISZERO_PUSH2_JUMPI",
			code:         []Instruction{{PUSH1, 2 << 8}, {ISZERO_PUSH2_JUMPI, 0}, {JUMPDEST, 0}},
			gas:          cost(3 + 3 + 3 + 10 + 1),
		},
		{
			nameOverride: "PUSH2_JUMP",
			code:         []Instruction{{PUSH2_JUMP, 1}, {JUMPDEST, 0}},
			gas:          cost(3 + 8 + 1),
		},
		{
			nameOverride: "PUSH2_JUMPI",
			code:         []Instruction{{PUSH1, 2 << 8}, {PUSH2_JUMPI, 2}, {JUMPDEST, 0}},
			gas:          cost(3 + 3 + 10 + 1),
		},
		{
			nameOverride: "SWAP2_SWAP1_POP_JUMP",
			code: []Instruction{
				{PUSH1, 4 << 8},
				{PUSH1, 0},
				{PUSH1, 0},
				{SWAP2_SWAP1_POP_JUMP, 0}, {JUMPDEST, 0}},
			stack: stackSize(1, 2),
			gas:   cost(3*3 + 3*2 + 2 + 8 + 1),
		},
		// LFVM jump_to extension
		{
			code: []Instruction{{JUMP_TO, 2}, {NOOP, 0}},
		},
	}...)

	///////////////////////////////////
	// Log

	tests = append(tests,
		generateTestCaseFor(func(op OpCode) testDeclaration {
			n := int(op - LOG0)

			stackBefore := make([]tosca.Word, n+2)
			for i := 0; i < n+2; i++ {
				stackBefore[i] = toWord(1) //< value 1
			}

			return testDeclaration{
				code: []Instruction{{op, 0}},
				mockCalls: func(mock *tosca.MockRunContext) {
					mock.EXPECT().EmitLog(gomock.Any())

				},
				stack: stackWithValues(stackBefore, 0),
				gas:   cost(gas(375 + (n * 375) + 8*1 + 3)),
			}
		}, getOpCodesInRange(LOG0, LOG4))...,
	)

	///////////////////////////////////
	// TRIVIAL TESTS introduced in revisions

	tests = append(tests, testDeclaration{
		code:               []Instruction{{BASEFEE, 0}, {STOP, 0}},
		stack:              stackSize(0, 1),
		revisionConstraint: validFrom(tosca.R10_London),
		gas:                cost(gas(2)),
	})

	///////////////////////////////////
	// CALL opcodes

	callGas := toWord(700)
	address := toWord(1) //< address parameter found int the stack
	value := toWord(2)   //< value parameter found in the stack
	argsOffset := toWord(3)
	argsSize := toWord(4)
	retOffset := toWord(5)
	retSize := toWord(6)

	tests = append(tests, []testDeclaration{
		{
			code: []Instruction{{CALL, 0}},
			stack: stackWithValues([]tosca.Word{
				retSize,
				retOffset,
				argsSize,
				argsOffset,
				value,
				address,
				callGas,
			}, 1),
			revisionConstraint: validOnlyIn(tosca.R07_Istanbul),
			mockCalls: func(mock *tosca.MockRunContext) {
				mock.EXPECT().AccountExists(gomock.Any()).Return(true)              // non-empty account
				mock.EXPECT().GetBalance(gomock.Any()).Return(tosca.Value(callGas)) // enough balance to transfer value
				mock.EXPECT().Call(gomock.Any(), gomock.Any()).Return(tosca.CallResult{}, nil)
			},
			gas: cost(gas(
				700 + // static
					3 + // mem-expansion
					9000 - 2300 + // call-stipend
					700 + // value-transfer
					2300, // call-stipend return
			)),
		},

		{
			nameOverride: "CALL EIP-2929",
			code:         []Instruction{{CALL, 0}},
			stack: stackWithValues([]tosca.Word{
				retSize,
				retOffset,
				argsSize,
				argsOffset,
				value,
				address,
				callGas,
			}, 1),
			revisionConstraint: validFrom(tosca.R09_Berlin),
			mockCalls: func(mock *tosca.MockRunContext) {
				mock.EXPECT().AccountExists(gomock.Any()).Return(true)              // non-empty account
				mock.EXPECT().GetBalance(gomock.Any()).Return(tosca.Value(callGas)) // enough balance to transfer value
				mock.EXPECT().IsAddressInAccessList(gomock.Any()).Return(true)      // warm
				mock.EXPECT().Call(gomock.Any(), gomock.Any()).Return(tosca.CallResult{}, nil)
			},
			gas: cost(gas(
				0 + // static
					3 + // mem-expansion
					100 + // warm access
					9000 - 2300 + // call-stipend
					700 + // value-transfer
					2300, // call-stipend return
			)),
		},

		{
			code: []Instruction{{DELEGATECALL, 0}},
			stack: stackWithValues([]tosca.Word{
				retSize,
				retOffset,
				argsSize,
				argsOffset,
				address,
				callGas,
			}, 1),
			revisionConstraint: validBefore(tosca.R09_Berlin),
			mockCalls: func(mock *tosca.MockRunContext) {
				mock.EXPECT().Call(gomock.Any(), gomock.Any()).Return(tosca.CallResult{}, nil)
			},
			gas: cost(gas(
				700 + // static
					3 + // mem-expansion
					700, // value-transfer
			)),
		},
	}...)

	return tests
}

// FIXME: this integration test does not cover all the opcodes. this test will fail
func testInterpreter_TestCasesAreComplete(t *testing.T) {

	// This test is here to ensure that all opcodes are covered by the test cases.
	// If an opcode is not covered, the test will fail and the missing opcode will be printed.
	testedOps := map[OpCode]map[tosca.Revision]bool{}
	for _, test := range generateTestCases() {
		for _, instr := range test.code {
			if _, ok := testedOps[instr.opcode]; !ok {
				testedOps[instr.opcode] = map[tosca.Revision]bool{}
			}
			forEachSupportedRevision(t, test, func(t *testing.T, revision tosca.Revision) {
				testedOps[instr.opcode][revision] = true
			})
		}
	}

	re := regexp.MustCompile(`^[A-Z0-9_]+$`)
	for i := 0; i < numOpCodes; i++ {
		op := OpCode(i)
		if !re.MatchString(op.String()) {
			continue
		}

		if _, ok := testedOps[op]; !ok {
			t.Errorf("opcode %v is not tested", op)
		} else {
			// FIXME: false negatives, some ops just wont exist in older revisions
			for revision := tosca.R07_Istanbul; revision <= newestSupportedRevision; revision++ {
				if !testedOps[op][revision] {
					t.Errorf("opcode %v is not tested for revision %v", op, revision)
				}
			}
		}
	}
}

func TestInterpreter_EndToEndInstructionRun(t *testing.T) {

	for _, test := range generateTestCases() {

		name := test.nameOverride
		if len(name) == 0 {
			name = test.code[0].opcode.String()
		}

		t.Run(name, func(t *testing.T) {

			forEachSupportedRevision(t, test, func(t *testing.T, revision tosca.Revision) {
				ctrl := gomock.NewController(t)
				runContext := tosca.NewMockRunContext(ctrl)
				if test.mockCalls != nil {
					test.mockCalls(runContext)
				}
				ctxt := makeContext(runContext, revision, test)

				// Run testing code
				vanillaRunner{}.run(&ctxt)

				// Check the result.
				expectedStatus := statusStopped
				if test.statusOverride != nil {
					expectedStatus = *test.statusOverride
				}
				if want, got := expectedStatus, ctxt.status; want != got {
					t.Errorf("execution failed: status is %v, wanted %v", got, want)
				}

				expectedPC := int32(len(test.code))
				if test.pcAfter != nil {
					expectedPC = int32(*test.pcAfter)
				}
				if want, got := expectedPC, ctxt.pc; want != got {
					t.Errorf("execution failed: pc is %v, wanted %v", got, want)
				}

				if want, got := test.stack.sizeAfter, ctxt.stack.len(); want != got {
					t.Errorf("execution failed: stack size is %v, wanted %v", got, want)
				}

				// Check gas consumption
				if want, got := test.gas.after, ctxt.gas; want != got {
					t.Errorf("execution failed: gas consumption is %v, wanted %v",
						test.gas.before-got,
						test.gas.before-want)
				}

				// Check gas refund
				if want, got := test.gasRefund, ctxt.refund; want != got {
					t.Errorf("execution failed: gas refund is %v, wanted %v", got, want)
				}
			})
		})
	}
}

////////////////////////////////////////////////////////////////////////////////
// Helper utils

// testParameter is a tool that allows to define before-after pairs of values for a test.
type testParameter[T any] struct {
	before, after T
}

type gas tosca.Gas
type pcOverride int32

// overridePc will mark the tests to check for a specific PC value.
// When not used, len(code) is used as the expected PC value.
func overridePc(pc int32) *pcOverride {
	res := pcOverride(pc)
	return &res
}

// revisionConstraint is a helper struct to define a revision constraint for a test.
// revisions constraints are defined by the [min, max) range.
// zero value means no constraint.
type revisionConstraint struct {
	min, max *tosca.Revision
}

// isValidFor checks if the constraint is valid for revision.
func (rc revisionConstraint) isValidFor(revision tosca.Revision) bool {
	if rc.min != nil && revision < *rc.min {
		return false
	}
	if rc.max != nil && revision >= *rc.max {
		return false
	}
	return true
}

// validBefore creates a revision constraint that is valid for revisions greater equal than argument.
func validFrom(revision tosca.Revision) revisionConstraint {
	return revisionConstraint{min: &revision}
}

// validBefore creates a revision constraint that is valid for revisions older than argument.
func validBefore(revision tosca.Revision) revisionConstraint {
	return revisionConstraint{max: &revision}
}

// validInRange creates a revision constraint that is valid only for revision.
func validOnlyIn(revision tosca.Revision) revisionConstraint {
	next := revision + 1
	return revisionConstraint{min: &revision, max: &next}
}

// validInRange creates a revision constraint that is valid in the range [min, max]
func validInRange(min, max tosca.Revision) revisionConstraint {
	next := max + 1
	return revisionConstraint{min: &min, max: &next}
}

// generateOpCodesInRange generates a slice of opcodes in the range [start, end].
func getOpCodesInRange(start, end OpCode) []OpCode {
	opCodes := make([]OpCode, end-start+1)
	for i := start; i <= end; i++ {
		opCodes[i-start] = i
	}
	return opCodes
}

func generateTestCaseFor(f func(op OpCode) testDeclaration, opCodes []OpCode) []testDeclaration {
	tests := make([]testDeclaration, len(opCodes))
	for i, op := range opCodes {
		tests[i] = f(op)
	}
	return tests
}

func toKey(value byte) tosca.Key {
	res := tosca.Key{}
	res[len(res)-1] = value
	return res
}

func toWord(value int) tosca.Word {
	v := uint256.NewInt(uint64(value))
	return v.Bytes32()
}

type stackTest struct {
	before    []tosca.Word
	sizeAfter int
}

func stackSize(beforeSize, sizeAfter int) stackTest {
	before := make([]tosca.Word, beforeSize)
	for i := range before {
		before[i] = toWord(0x01)
	}
	return stackWithValues(before, sizeAfter)
}

func stackWithValues(before []tosca.Word, sizeAfter int) stackTest {
	return stackTest{before, sizeAfter}
}

func invariantStack() stackTest {
	return stackSize(0, 0)
}

func cost(gas gas) testParameter[tosca.Gas] {
	return testParameter[tosca.Gas]{GAS_START, GAS_START - tosca.Gas(gas)}
}

func attachDataToOp(op OpCode, dataInstructions int) []Instruction {
	code := []Instruction{{op, 0}}
	for i := 0; i < dataInstructions; i++ {
		code = append(code, Instruction{DATA, 0})
	}
	return code
}

func forEachSupportedRevision(t *testing.T, test testDeclaration, f func(t *testing.T, revision tosca.Revision)) {
	for revision := tosca.R07_Istanbul; revision <= newestSupportedRevision; revision++ {
		if !test.revisionConstraint.isValidFor(revision) {
			continue
		}
		t.Run(revision.String(), func(t *testing.T) { f(t, revision) })
	}
}

func expectOutOfGas() *status {
	v := statusOutOfGas
	return &v
}
func expectReverted() *status {
	v := statusReverted
	return &v
}

func expectReturned() *status {
	v := statusReturned
	return &v
}

func expectInvalidInstruction() *status {
	v := statusInvalidInstruction
	return &v
}

func makeContext(
	runContext tosca.RunContext,
	revision tosca.Revision,
	test testDeclaration,
) context {

	ctx := context{
		params: tosca.Parameters{
			BlockParameters: tosca.BlockParameters{
				Revision: revision,
			},
			Gas:    test.gas.before,
			Input:  []byte{},
			Static: test.staticCtx,
		},
		context: runContext,
		gas:     test.gas.before,
		stack:   NewStack(),
		memory:  NewMemory(),
		status:  statusRunning,
		code:    test.code,
	}

	for i, v := range test.stack.before {
		ctx.stack.data[i] = *uint256.NewInt(0).SetBytes(v[:])
	}
	ctx.stack.stackPointer = len(test.stack.before)

	return ctx
}

func TestIntegrationTestUtils_RevisionRagesAreCorrect(t *testing.T) {

	for i := tosca.R07_Istanbul; i <= newestSupportedRevision; i++ {
		only := validOnlyIn(i)
		for j := tosca.R07_Istanbul; j <= newestSupportedRevision; j++ {
			if want, got := i == j, only.isValidFor(j); want != got {
				t.Errorf("only %v check failed for %v: got %v, wanted %v", i, j, got, want)
			}

			inRange := validInRange(i, j)
			for k := tosca.R07_Istanbul; k <= newestSupportedRevision; k++ {
				inside := i <= k && k <= j
				if want, got := inside, inRange.isValidFor(k); want != got {
					t.Errorf("inRange %v-%v check failed for %v: got %v, wanted %v", i, j, k, got, want)
				}
			}
			after := validFrom(i)
			for k := tosca.R07_Istanbul; k <= newestSupportedRevision; k++ {
				if want, got := i <= k, after.isValidFor(k); want != got {
					t.Errorf("after %v check failed for %v: got %v, wanted %v", i, k, got, want)
				}
			}
			before := validBefore(j)
			for k := tosca.R07_Istanbul; k <= newestSupportedRevision; k++ {
				if want, got := k < j, before.isValidFor(k); want != got {
					t.Errorf("before %v check failed for %v: got %v, wanted %v", j, k, got, want)
				}
			}
		}
	}
}
