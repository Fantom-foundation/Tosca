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
	"strings"
	"testing"

	"github.com/Fantom-foundation/Tosca/go/tosca"
	"github.com/Fantom-foundation/Tosca/go/tosca/vm"
	"github.com/holiman/uint256"
	"go.uber.org/mock/gomock"
)

// To run the benchmark use
//  go test ./core/vm/lfvm -bench=.*Fib.* --benchtime 10s

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
		context:  runContext,
		gas:      gas,
		stack:    NewStack(),
		memory:   NewMemory(),
		status:   RUNNING,
		code:     code,
		revision: revision,
	}

	// Move the stack pointer to the required hight.
	// For the tests using the resulting context the actual
	// stack content is not relevant. It is merely used for
	// checking stack over- or under-flows.
	ctxt.stack.stack_ptr = stackPtr

	return ctxt
}

// Test UseGas function and correct status after running out of gas
func TestGasFunc(t *testing.T) {

	tests := map[string]struct {
		initialGas   tosca.Gas
		cost         tosca.Gas
		resultingGas tosca.Gas
		expected     bool
		status       Status
	}{
		"Zero amount":       {100, 0, 100, true, RUNNING},
		"Sufficient gas":    {100, 10, 90, true, RUNNING},
		"Insufficient gas":  {10, 100, 10, false, OUT_OF_GAS},
		"All gas":           {100, 100, 0, true, RUNNING},
		"Negative cost":     {100, -100, 100, false, OUT_OF_GAS},
		"Negative gas":      {-100, 100, -100, false, OUT_OF_GAS},
		"Negative Negative": {-100, -100, -100, false, OUT_OF_GAS},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			ctx := getEmptyContext()
			ctx.gas = test.initialGas
			ok := ctx.UseGas(test.cost)
			if ok != test.expected {
				t.Errorf("expected UseGas to return %v, got %v", test.initialGas >= test.cost, ok)
			}
			if ctx.status != test.status {
				t.Errorf("expected status to be %v, got %v", test.status, ctx.status)
				return
			}
			if ctx.gas != test.resultingGas {
				t.Errorf("expected gas to be %v, got %v", test.resultingGas, ctx.gas)
			}
		})
	}
}

type OpcodeTest struct {
	name        string
	code        []Instruction
	stackPtrPos int
	argData     uint16
	endStatus   Status
	isBerlin    bool // < TODO: replace with revision
	isLondon    bool
	mockCalls   func(*tosca.MockRunContext)
	gasStart    tosca.Gas
	gasConsumed tosca.Gas
	gasRefund   tosca.Gas
}

func getInstructions(start OpCode, end OpCode) (opCodes []OpCode) {
	for i := start; i <= end; i++ {
		opCodes = append(opCodes, OpCode(i))
	}
	return
}

func getInstructionsWithGas(start OpCode, end OpCode, gas tosca.Gas) (opCodes []OpCodeWithGas) {
	for i := start; i <= end; i++ {
		opCode := OpCodeWithGas{OpCode(i), gas}
		opCodes = append(opCodes, opCode)
	}
	return
}

var fullStackFailOpCodes = []OpCode{
	MSIZE, ADDRESS, ORIGIN, CALLER, CALLVALUE, CALLDATASIZE,
	CODESIZE, GASPRICE, COINBASE, TIMESTAMP, NUMBER,
	PREVRANDAO, GASLIMIT, PC, GAS, RETURNDATASIZE,
	SELFBALANCE, CHAINID, BASEFEE, BLOBBASEFEE,
	PUSH0, PUSH1_PUSH1_PUSH1_SHL_SUB,
	PUSH1_DUP1, PUSH1_PUSH1, PUSH1_PUSH4_DUP3,
}

var emptyStackFailOpCodes = []OpCode{
	POP, ADD, SUB, MUL, DIV, SDIV, MOD, SMOD, EXP, SIGNEXTEND,
	SHA3, LT, GT, SLT, SGT, EQ, AND, XOR, OR, BYTE,
	SHL, SHR, SAR, ADDMOD, MULMOD, ISZERO, NOT, BALANCE, CALLDATALOAD, EXTCODESIZE,
	BLOCKHASH, MCOPY, MLOAD, SLOAD, EXTCODEHASH, JUMP, SELFDESTRUCT, BLOBHASH,
	MSTORE, MSTORE8, SSTORE, TLOAD, TSTORE, JUMPI, RETURN, REVERT,
	CALLDATACOPY, CODECOPY, RETURNDATACOPY,
	EXTCODECOPY, CREATE, CREATE2, CALL, CALLCODE,
	STATICCALL, DELEGATECALL, POP_POP, POP_JUMP,
	SWAP2_POP, PUSH1_ADD, PUSH1_SHL, SWAP2_SWAP1_POP_JUMP,
	PUSH2_JUMPI, ISZERO_PUSH2_JUMPI, SWAP2_SWAP1,
	DUP2_LT, SWAP1_POP_SWAP2_SWAP1, POP_SWAP2_SWAP1_POP,
	AND_SWAP1_POP_SWAP2_SWAP1, SWAP1_POP, DUP2_MSTORE,
}

func addFullStackFailOpCodes(tests []OpcodeTest) []OpcodeTest {
	var addedTests []OpcodeTest
	addedTests = append(addedTests, tests...)
	var opCodes []OpCode
	opCodes = append(opCodes, fullStackFailOpCodes...)
	opCodes = append(opCodes, getInstructions(PUSH1, PUSH32)...)
	opCodes = append(opCodes, getInstructions(DUP1, DUP16)...)
	for _, opCode := range opCodes {
		addedTests = append(addedTests, OpcodeTest{opCode.String(), []Instruction{{opCode, 1}}, MAX_STACK_SIZE, 0, ERROR, false, false, nil, GAS_START, 0, 0})
	}
	return addedTests
}

func addEmptyStackFailOpCodes(tests []OpcodeTest) []OpcodeTest {
	var addedTests []OpcodeTest
	addedTests = append(addedTests, tests...)
	var opCodes []OpCode
	opCodes = append(opCodes, emptyStackFailOpCodes...)
	opCodes = append(opCodes, getInstructions(DUP1, DUP16)...)
	opCodes = append(opCodes, getInstructions(SWAP1, SWAP16)...)
	opCodes = append(opCodes, getInstructions(LOG0, LOG4)...)
	for _, opCode := range opCodes {
		addedTests = append(addedTests, OpcodeTest{opCode.String(), []Instruction{{opCode, 1}}, 0, 0, ERROR, false, false, nil, GAS_START, 0, 0})
	}
	return addedTests
}
func TestContainsAllMaxStackBoundryInstructions(t *testing.T) {
	set := make(map[OpCode]bool)
	fullStackFailOpcodes := addFullStackFailOpCodes(nil)
	for _, op := range fullStackFailOpcodes {
		set[op.code[0].opcode] = true
	}
	for op := OpCode(0); op < NUM_OPCODES; op++ {
		insStack := getStaticStackInternal(op)
		if _, exists := set[op]; exists != (MAX_STACK_SIZE-insStack.stackMax > 0) {
			t.Errorf("OpCode %v adding %v to stack, is not contained in FullStackFailOpCodes", op.String(), MAX_STACK_SIZE-insStack.stackMax)
		}
	}
}

func TestContainsAllMinStackBoundryInstructions(t *testing.T) {
	set := make(map[OpCode]bool)
	emptyStackFailOpcodes := addEmptyStackFailOpCodes(nil)
	for _, op := range emptyStackFailOpcodes {
		set[op.code[0].opcode] = true
	}
	for op := OpCode(0); op < NUM_OPCODES; op++ {
		insStack := getStaticStackInternal(op)
		if _, exists := set[op]; exists != (insStack.stackMin > 0) {
			t.Errorf("OpCode %v with minimum stack size of %v values, is not contained in EmptyStackFailOpcodes", op.String(), insStack.stackMin)
		}
	}
}

func TestStackMinBoundry(t *testing.T) {

	// Add tests for execution
	for _, test := range addEmptyStackFailOpCodes(nil) {

		// Create execution context.
		ctxt := getEmptyContext()
		ctxt.code = test.code
		ctxt.stack.stack_ptr = test.stackPtrPos

		// Run testing code
		run(&ctxt)

		// Check the result.
		if ctxt.status != test.endStatus {
			t.Errorf("execution failed %v: status is %v, wanted %v", test.name, ctxt.status, test.endStatus)
		} else {
			t.Log("Success", test.name)
		}
	}
}

func TestStackMaxBoundry(t *testing.T) {

	// Add tests for execution
	for _, test := range addFullStackFailOpCodes(nil) {

		// Create execution context.
		ctxt := getEmptyContext()
		ctxt.code = test.code
		ctxt.stack.stack_ptr = test.stackPtrPos

		// Run testing code
		run(&ctxt)

		// Check the result.
		if ctxt.status != test.endStatus {
			t.Errorf("execution failed %v: status is %v, wanted %v", test.name, ctxt.status, test.endStatus)
		} else {
			t.Log("Success", test.name)
		}
	}
}

var opcodeTests = []OpcodeTest{
	{"POP", []Instruction{{PUSH1, 1 << 8}, {POP, 0}}, 0, 0, STOPPED, false, false, nil, GAS_START, 5, 0},
	{"JUMP", []Instruction{{PUSH1, 2 << 8}, {JUMP, 0}, {JUMPDEST, 0}}, 0, 0, STOPPED, false, false, nil, GAS_START, 12, 0},
	{"JUMPI", []Instruction{{PUSH1, 1 << 8}, {PUSH1, 3 << 8}, {JUMPI, 0}, {JUMPDEST, 0}}, 0, 0, STOPPED, false, false, nil, GAS_START, 17, 0},
	{"JUMPDEST", []Instruction{{JUMPDEST, 0}}, 0, 0, STOPPED, false, false, nil, GAS_START, 1, 0},
	{"RETURN", []Instruction{{RETURN, 0}}, 20, 0, RETURNED, false, false, nil, GAS_START, 0, 0},
	{"REVERT", []Instruction{{REVERT, 0}}, 20, 0, REVERTED, false, false, nil, GAS_START, 0, 0},
	{"PC", []Instruction{{PC, 0}}, 0, 0, STOPPED, false, false, nil, GAS_START, 2, 0},
	{"STOP", []Instruction{{STOP, 0}}, 0, 0, STOPPED, false, false, nil, GAS_START, 0, 0},
	{"SLOAD", []Instruction{{PUSH1, 0}, {SLOAD, 0}}, 0, 0, STOPPED, false, false,
		func(mock *tosca.MockRunContext) {
			mock.EXPECT().GetStorage(tosca.Address{0}, toKey(0)).Return(toWord(0))
		}, GAS_START, 803, 0},
	{"SLOAD Berlin", []Instruction{{PUSH1, 0}, {SLOAD, 0}}, 0, 0, STOPPED, true, false,
		func(mock *tosca.MockRunContext) {
			mock.EXPECT().GetStorage(tosca.Address{0}, toKey(0)).Return(toWord(0))
			mock.EXPECT().IsSlotInAccessList(tosca.Address{0}, toKey(0)).Return(true, true)
		}, GAS_START, 103, 0},
	{"SLOAD Berlin no slot", []Instruction{{PUSH1, 0}, {SLOAD, 0}}, 0, 0, STOPPED, true, false,
		func(mock *tosca.MockRunContext) {
			mock.EXPECT().GetStorage(tosca.Address{0}, toKey(0)).Return(toWord(0))
			mock.EXPECT().IsSlotInAccessList(tosca.Address{0}, toKey(0)).Return(false, false)
			mock.EXPECT().AccessStorage(tosca.Address{0}, toKey(0))
		}, GAS_START, 2103, 0},
	{"SSTORE same value", []Instruction{{PUSH1, 0}, {PUSH1, 0}, {SSTORE, 0}}, 0, 0, STOPPED, false, false,
		func(mock *tosca.MockRunContext) {
			mock.EXPECT().GetStorage(tosca.Address{0}, toKey(0)).Return(toWord(0))
			mock.EXPECT().SetStorage(tosca.Address{0}, toKey(0), toWord(0))
		}, GAS_START, 806, 0},
	{"SSTORE diff value, same state as db, db is 0", []Instruction{{PUSH1, 1 << 8}, {PUSH1, 0}, {SSTORE, 0}}, 0, 0, STOPPED, false, false,
		func(mock *tosca.MockRunContext) {
			mock.EXPECT().GetStorage(tosca.Address{0}, toKey(0)).Return(toWord(0))
			mock.EXPECT().GetCommittedStorage(tosca.Address{0}, toKey(0)).Return(toWord(0))
			mock.EXPECT().SetStorage(tosca.Address{0}, toKey(0), toWord(1))
		}, GAS_START, 20006, 0},
	{"SSTORE diff value, same state as db, val is 0", []Instruction{{PUSH1, 0}, {PUSH1, 0}, {SSTORE, 0}}, 0, 0, STOPPED, false, false,
		func(mock *tosca.MockRunContext) {
			mock.EXPECT().GetStorage(tosca.Address{0}, toKey(0)).Return(toWord(1))
			mock.EXPECT().GetCommittedStorage(tosca.Address{0}, toKey(0)).Return(toWord(1))
			mock.EXPECT().SetStorage(tosca.Address{0}, toKey(0), toWord(0))
		}, GAS_START, 5006, SstoreClearsScheduleRefundEIP2200},
	{"SSTORE diff value, diff state as db, db it not 0, state is 0", []Instruction{{PUSH1, 1 << 8}, {PUSH1, 0}, {SSTORE, 0}}, 0, 0, STOPPED, false, false,
		func(mock *tosca.MockRunContext) {
			mock.EXPECT().GetStorage(tosca.Address{0}, toKey(0)).Return(toWord(0))
			mock.EXPECT().GetCommittedStorage(tosca.Address{0}, toKey(0)).Return(toWord(2))
			mock.EXPECT().SetStorage(tosca.Address{0}, toKey(0), toWord(1))
		}, GAS_START, 806, tosca.Gas(-int(SstoreClearsScheduleRefundEIP2200))},
	{"SSTORE diff value, diff state as db, db it not 0, val is 0", []Instruction{{PUSH1, 0}, {PUSH1, 0}, {SSTORE, 0}}, 0, 0, STOPPED, false, false,
		func(mock *tosca.MockRunContext) {
			mock.EXPECT().GetStorage(tosca.Address{0}, toKey(0)).Return(toWord(1))
			mock.EXPECT().GetCommittedStorage(tosca.Address{0}, toKey(0)).Return(toWord(2))
			mock.EXPECT().SetStorage(tosca.Address{0}, toKey(0), toWord(0))
		}, GAS_START, 806, SstoreClearsScheduleRefundEIP2200},
	{"SSTORE diff value, diff state as db, db same as val, db is 0", []Instruction{{PUSH1, 0}, {PUSH1, 1 << 8}, {SSTORE, 0}}, 0, 0, STOPPED, false, false,
		func(mock *tosca.MockRunContext) {
			mock.EXPECT().GetStorage(tosca.Address{0}, toKey(1)).Return(toWord(1))
			mock.EXPECT().GetCommittedStorage(tosca.Address{0}, toKey(1)).Return(toWord(0))
			mock.EXPECT().SetStorage(tosca.Address{0}, toKey(1), toWord(0))
		}, GAS_START, 806, SstoreSetGasEIP2200 - SloadGasEIP2200},
	{"SSTORE diff value, diff state as db, db same as val, db is not 0", []Instruction{{PUSH1, 2 << 8}, {PUSH1, 0}, {SSTORE, 0}}, 0, 0, STOPPED, false, false,
		func(mock *tosca.MockRunContext) {
			mock.EXPECT().GetStorage(tosca.Address{0}, toKey(0)).Return(toWord(1))
			mock.EXPECT().GetCommittedStorage(tosca.Address{0}, toKey(0)).Return(toWord(2))
			mock.EXPECT().SetStorage(tosca.Address{0}, toKey(0), toWord(2))
		}, GAS_START, 806, SstoreResetGasEIP2200 - SloadGasEIP2200},
	{"SSTORE Berlin same value", []Instruction{{PUSH1, 0}, {PUSH1, 0}, {SSTORE, 0}}, 0, 0, STOPPED, true, false,
		func(mock *tosca.MockRunContext) {
			mock.EXPECT().GetStorage(tosca.Address{0}, toKey(0)).Return(toWord(0))
			mock.EXPECT().IsSlotInAccessList(tosca.Address{0}, toKey(0)).Return(true, false)
			mock.EXPECT().AccessStorage(tosca.Address{0}, toKey(0))
			mock.EXPECT().SetStorage(tosca.Address{0}, toKey(0), toWord(0))
		}, GAS_START, 2206, 0},
	{"SSTORE Berlin diff value, same state as db, db is 0", []Instruction{{PUSH1, 1 << 8}, {PUSH1, 0}, {SSTORE, 0}}, 0, 0, STOPPED, true, false,
		func(mock *tosca.MockRunContext) {
			mock.EXPECT().GetStorage(tosca.Address{0}, toKey(0)).Return(toWord(0))
			mock.EXPECT().IsSlotInAccessList(tosca.Address{0}, toKey(0)).Return(true, true)
			mock.EXPECT().GetCommittedStorage(tosca.Address{0}, toKey(0)).Return(toWord(0))
			mock.EXPECT().SetStorage(tosca.Address{0}, toKey(0), toWord(1))
		}, GAS_START, 20006, 0},
	{"SSTORE Berlin diff value, same state as db, val is 0", []Instruction{{PUSH1, 0}, {PUSH1, 0}, {SSTORE, 0}}, 0, 0, STOPPED, true, false,
		func(mock *tosca.MockRunContext) {
			mock.EXPECT().GetStorage(tosca.Address{0}, toKey(0)).Return(toWord(1))
			mock.EXPECT().IsSlotInAccessList(tosca.Address{0}, toKey(0)).Return(true, true)
			mock.EXPECT().GetCommittedStorage(tosca.Address{0}, toKey(0)).Return(toWord(1))
			mock.EXPECT().SetStorage(tosca.Address{0}, toKey(0), toWord(0))
		}, GAS_START, 2906, SstoreClearsScheduleRefundEIP2200},
	{"SSTORE Berlin diff value, diff state as db, db it not 0, state is 0", []Instruction{{PUSH1, 1 << 8}, {PUSH1, 0}, {SSTORE, 0}}, 0, 0, STOPPED, true, false,
		func(mock *tosca.MockRunContext) {
			mock.EXPECT().GetStorage(tosca.Address{0}, toKey(0)).Return(toWord(0))
			mock.EXPECT().IsSlotInAccessList(tosca.Address{0}, toKey(0)).Return(true, true)
			mock.EXPECT().GetCommittedStorage(tosca.Address{0}, toKey(0)).Return(toWord(2))
			mock.EXPECT().SetStorage(tosca.Address{0}, toKey(0), toWord(1))
		}, GAS_START, 106, tosca.Gas(-int(SstoreClearsScheduleRefundEIP2200))},
	{"SSTORE Berlin diff value, diff state as db, db it not 0, val is 0", []Instruction{{PUSH1, 0}, {PUSH1, 0}, {SSTORE, 0}}, 0, 0, STOPPED, true, false,
		func(mock *tosca.MockRunContext) {
			mock.EXPECT().GetStorage(tosca.Address{0}, toKey(0)).Return(toWord(1))
			mock.EXPECT().IsSlotInAccessList(tosca.Address{0}, toKey(0)).Return(true, true)
			mock.EXPECT().GetCommittedStorage(tosca.Address{0}, toKey(0)).Return(toWord(2))
			mock.EXPECT().SetStorage(tosca.Address{0}, toKey(0), toWord(0))
		}, GAS_START, 106, SstoreClearsScheduleRefundEIP2200},
	{"SSTORE Berlin diff value, diff state as db, db same as val, db is 0", []Instruction{{PUSH1, 0}, {PUSH1, 1 << 8}, {SSTORE, 0}}, 0, 0, STOPPED, true, false,
		func(mock *tosca.MockRunContext) {
			mock.EXPECT().GetStorage(tosca.Address{0}, toKey(1)).Return(toWord(1))
			mock.EXPECT().IsSlotInAccessList(tosca.Address{0}, toKey(1)).Return(true, true)
			mock.EXPECT().GetCommittedStorage(tosca.Address{0}, toKey(1)).Return(toWord(0))
			mock.EXPECT().SetStorage(tosca.Address{0}, toKey(1), toWord(0))
		}, GAS_START, 106, SstoreSetGasEIP2200 - WarmStorageReadCostEIP2929},
	{"SSTORE Berlin diff value, diff state as db, db same as val, db is not 0", []Instruction{{PUSH1, 2 << 8}, {PUSH1, 0}, {SSTORE, 0}}, 0, 0, STOPPED, true, false,
		func(mock *tosca.MockRunContext) {
			mock.EXPECT().GetStorage(tosca.Address{0}, toKey(0)).Return(toWord(1))
			mock.EXPECT().IsSlotInAccessList(tosca.Address{0}, toKey(0)).Return(true, true)
			mock.EXPECT().GetCommittedStorage(tosca.Address{0}, toKey(0)).Return(toWord(2))
			mock.EXPECT().SetStorage(tosca.Address{0}, toKey(0), toWord(2))
		}, GAS_START, 106, (SstoreResetGasEIP2200 - ColdSloadCostEIP2929) - WarmStorageReadCostEIP2929},
}

type OpCodeWithGas struct {
	OpCode
	gas tosca.Gas
}

func addOKOpCodes(tests []OpcodeTest) []OpcodeTest {
	var addedTests []OpcodeTest
	addedTests = append(addedTests, tests...)
	for i := PUSH1; i <= PUSH32; i++ {
		code := []Instruction{{i, 1}}
		dataNum := int((i - PUSH1) / 2)
		for j := 0; j < dataNum; j++ {
			code = append(code, Instruction{DATA, 1})
		}
		addedTests = append(addedTests, OpcodeTest{i.String(), code, 20, 0, STOPPED, false, false, nil, GAS_START, 3, 0})
	}
	var opCodes []OpCodeWithGas
	opCodes = append(opCodes, getInstructionsWithGas(DUP1, SWAP16, 3)...)
	opCodes = append(opCodes, getInstructionsWithGas(ADD, SUB, 3)...)
	opCodes = append(opCodes, getInstructionsWithGas(MUL, SMOD, 5)...)
	opCodes = append(opCodes, getInstructionsWithGas(ADDMOD, MULMOD, 8)...)
	opCodes = append(opCodes, OpCodeWithGas{EXP, 10})
	opCodes = append(opCodes, OpCodeWithGas{SIGNEXTEND, 5})
	opCodes = append(opCodes, OpCodeWithGas{SHA3, 30})
	opCodes = append(opCodes, getInstructionsWithGas(LT, SAR, 3)...)
	opCodes = append(opCodes, OpCodeWithGas{SWAP1_POP_SWAP2_SWAP1, 11})
	opCodes = append(opCodes, OpCodeWithGas{POP_SWAP2_SWAP1_POP, 10})
	opCodes = append(opCodes, OpCodeWithGas{POP_POP, 4})
	opCodes = append(opCodes, getInstructionsWithGas(PUSH1_SHL, PUSH1_DUP1, 6)...)
	//opCodes = append(opCodes, OpCodeWithGas{PUSH2_JUMP, 11})
	opCodes = append(opCodes, OpCodeWithGas{PUSH2_JUMPI, 13})
	opCodes = append(opCodes, OpCodeWithGas{PUSH1_PUSH1, 6})
	opCodes = append(opCodes, OpCodeWithGas{SWAP1_POP, 5})
	opCodes = append(opCodes, OpCodeWithGas{SWAP2_SWAP1, 6})
	opCodes = append(opCodes, OpCodeWithGas{SWAP2_POP, 5})
	opCodes = append(opCodes, OpCodeWithGas{DUP2_MSTORE, 9})
	opCodes = append(opCodes, OpCodeWithGas{DUP2_LT, 6})
	for _, opCode := range opCodes {
		code := []Instruction{{opCode.OpCode, 0}}
		addedTests = append(addedTests, OpcodeTest{opCode.String(), code, 20, 0, STOPPED, false, false, nil, GAS_START, opCode.gas, 0})
	}
	return addedTests
}

func TestOKInstructionPath(t *testing.T) {
	for _, test := range addOKOpCodes(opcodeTests) {
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
			run(&ctxt)

			// Check the result.
			if ctxt.status != test.endStatus {
				t.Errorf("execution failed: status is %v, wanted %v", ctxt.status, test.endStatus)
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

func TestRunReturnsEmptyResultOnEmptyCode(t *testing.T) {
	// Create execution context.
	ctxt := getEmptyContext()
	// Get tosca.Parameters
	ctxt.params = tosca.Parameters{
		Input:  []byte{},
		Static: true,
		Gas:    10,
	}
	ctxt.code = make([]Instruction, 0)

	// Run testing code
	result, err := Run(ctxt.params, ctxt.code, false, true, false)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if result.Output != nil {
		t.Errorf("unexpected output: want nil, got %v", result.Output)
	}
	if result.GasLeft != ctxt.params.Gas {
		t.Errorf("unexpected gas left: want %v, got %v", ctxt.params.Gas, result.GasLeft)
	}
	if !result.Success {
		t.Errorf("unexpected success: want true, got false")
	}
}

func TestRunWithStatistics(t *testing.T) {
	// Create execution context.
	ctxt := getEmptyContext()
	// Get tosca.Parameters
	ctxt.params = tosca.Parameters{
		Input:  []byte{},
		Static: true,
		Gas:    10,
		Code:   []byte{byte(STOP), 0},
	}
	ctxt.code = []Instruction{{STOP, 0}}

	// Run testing code
	_, err := Run(ctxt.params, ctxt.code, true, true, false)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if got := global_statistics.single_count[uint64(STOP)]; got != 1 {
		t.Errorf("unexpected statistics: want 1 stop, got %v", got)
	}
}

func TestRunWithLogging(t *testing.T) {
	// Create execution context.
	ctxt := getEmptyContext()
	instructions := []Instruction{
		{PUSH1, 1},
		{STOP, 0}}

	// Get tosca.Parameters
	ctxt.params = tosca.Parameters{
		Input:  []byte{},
		Static: true,
		Gas:    10,
		Code:   []byte{0x0},
	}
	ctxt.code = instructions

	// redirect stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Run testing code
	_, err := Run(ctxt.params, ctxt.code, false, true, true)
	// read the output
	w.Close()
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

func TestRunBasic(t *testing.T) {

	// Create execution context.
	ctxt := getEmptyContext()
	instructions := []Instruction{
		{PUSH1, 1},
		{STOP, 0}}

	// Get tosca.Parameters
	ctxt.params = tosca.Parameters{
		Input:  []byte{},
		Static: true,
		Gas:    10,
		Code:   []byte{0x0},
	}
	ctxt.code = instructions

	// redirect stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Run testing code
	_, err := Run(ctxt.params, ctxt.code, false, true, false)
	// read the output
	w.Close()
	out, _ := io.ReadAll(r)
	os.Stdout = old

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// check the output
	if len(string(out)) != 0 {
		t.Errorf("unexpected output: want \"\", got %v", string(out))
	}

	if global_statistics.count > 1 {
		t.Errorf("unexpected statistics: want none, got %v", global_statistics.count)
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
		ctxt.result_size = uint256.Int{uint64(len(baseOutput))}
		return ctxt
	}

	tests := map[string]struct {
		setup          func(*context)
		expectedErr    error
		expectedResult tosca.Result
	}{
		"invalid instruction": {func(ctx *context) { ctx.status = INVALID_INSTRUCTION }, nil, tosca.Result{Success: false}},
		"out of gas":          {func(ctx *context) { ctx.status = OUT_OF_GAS }, nil, tosca.Result{Success: false}},
		"max init code": {func(ctx *context) { ctx.status = MAX_INIT_CODE_SIZE_EXCEEDED }, nil,
			tosca.Result{Success: false}},
		"error": {func(ctx *context) { ctx.status = ERROR }, nil, tosca.Result{Success: false}},
		"returned": {func(ctx *context) { ctx.status = RETURNED }, nil, tosca.Result{Success: true,
			Output: baseOutput, GasLeft: baseGas, GasRefund: baseRefund}},
		"reverted": {func(ctx *context) { ctx.status = REVERTED }, nil,
			tosca.Result{Success: false, Output: baseOutput, GasLeft: baseGas, GasRefund: 0}},
		"stopped": {func(ctx *context) { ctx.status = STOPPED }, nil,
			tosca.Result{Success: true, Output: nil, GasLeft: baseGas, GasRefund: baseRefund}},
		"suicide": {func(ctx *context) { ctx.status = SUICIDED }, nil,
			tosca.Result{Success: true, Output: nil, GasLeft: baseGas, GasRefund: baseRefund}},
		"unknown status": {func(ctx *context) { ctx.status = ERROR + 1 },
			errUnknownStatus{ERROR + 1}, tosca.Result{}},
		"getOuput fail": {func(ctx *context) {
			ctx.status = RETURNED
			ctx.result_size = uint256.Int{1, 1}
		}, nil, tosca.Result{Success: false}},
	}

	for name, test := range tests {
		t.Run(fmt.Sprintf("%v", name), func(t *testing.T) {

			ctxt := getCtxt()
			test.setup(&ctxt)

			res, err := generateResult(&ctxt)

			// Check the result.
			if !errors.Is(err, test.expectedErr) {
				t.Errorf("unexpected error: want %v, got %v", test.expectedErr, err)
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
		"size overflow": {func(ctx *context) { ctx.result_size = uint256.Int{1, 1} }, errGasUintOverflow},
		"offset overflow": {func(ctx *context) {
			ctx.result_size = uint256.Int{1}
			ctx.result_offset = uint256.Int{1, 1}
		}, errGasUintOverflow},
		"memory overflow": {func(ctx *context) {
			ctx.result_size = uint256.Int{math.MaxUint64 - 1}
			ctx.result_offset = uint256.Int{2}
		}, errGasUintOverflow},
	}

	for name, test := range tests {
		t.Run(fmt.Sprintf("%v", name), func(t *testing.T) {
			ctxt := getEmptyContext()
			test.setup(&ctxt)
			ctxt.status = RETURNED

			// Run testing code
			_, err := getOutput(&ctxt)
			if !errors.Is(err, test.expectedErr) {
				t.Errorf("unexpected error: want error, got nil")
			}
		})
	}
}

func TestDumpProfilePrintsExpectedOutput(t *testing.T) {

	tests := map[string]struct {
		code         tosca.Code
		findInOutput []string
	}{
		"singles": {tosca.Code{byte(vm.STOP)},
			[]string{
				"Steps: 1",
				"STOP                          : 1 (100.00%)",
			}},
		"pairs": {tosca.Code{byte(vm.PUSH1), 0x01, byte(vm.STOP)},
			[]string{
				"Steps: 2",
				"PUSH1                         : 1 (50.00%)",
				"STOP                          : 1 (50.00%)",
				"PUSH1                         STOP                          : 1"}},
		"triples": {tosca.Code{byte(vm.PUSH1), 0x01, byte(vm.PUSH1), 0x01, byte(vm.STOP)},
			[]string{
				"Steps: 3",
				"PUSH1                         : 2 (66.67%)",
				"STOP                          : 1 (33.33%)",
				"PUSH1                         PUSH1                         STOP                          : 1"}},
		"quads": {tosca.Code{byte(vm.PUSH1), 0x01, byte(vm.PUSH1), 0x01, byte(vm.PUSH1), 0x01, byte(vm.STOP)},
			[]string{
				"Steps: 4",
				"PUSH1                         : 3 (75.00%)",
				"STOP                          : 1 (25.00%)",
				"PUSH1                         PUSH1                         PUSH1                         : 1 (25.00%)",
				"PUSH1                         PUSH1                         STOP                          : 1 (25.00%)",
				"PUSH1                         PUSH1                         PUSH1                         STOP                          : 1 (25.00%)",
			}},
	}

	for name, test := range tests {
		t.Run(fmt.Sprintf("%v", name), func(t *testing.T) {
			r, w, err := os.Pipe()
			if err != nil {
				t.Fatalf("Failed to create pipe: %v", err)
			}

			// redirect stdout
			old := os.Stderr
			os.Stderr = w
			log.SetOutput(os.Stderr)

			instance, err := NewVm(Config{
				WithStatistics: true,
			})
			if err != nil {
				t.Fatalf("Failed to create VM: %v", err)
			}
			instance.ResetProfile()
			//run code
			instance.Run(tosca.Parameters{Input: []byte{}, Static: true, Gas: 10,
				Code: test.code})

			// Run testing code
			instance.DumpProfile()

			// read the output
			w.Close()
			out, _ := io.ReadAll(r)
			os.Stderr = old
			log.SetOutput(os.Stderr)

			for _, s := range test.findInOutput {
				if !strings.Contains(string(out), s) {
					t.Errorf("did not find ocurrences of %v in %v", s, string(out))
				}
			}
		})
	}
}

func TestStepsProperlyHandlesJUMP_TO(t *testing.T) {
	// Create execution context.
	ctxt := getEmptyContext()
	instructions := []Instruction{
		{JUMP_TO, 0x02},
		{RETURN, 0},
		{STOP, 0},
	}

	// Get tosca.Parameters
	ctxt.params = tosca.Parameters{
		Input:  []byte{},
		Static: false,
		Gas:    10,
		Code:   []byte{0x0},
	}
	ctxt.code = instructions

	// Run testing code
	steps(&ctxt, false)

	if ctxt.status != STOPPED {
		t.Errorf("unexpected status: want STOPPED, got %v", ctxt.status)
	}
}

func TestStepsDetectsNonExecutableCode(t *testing.T) {
	// Create execution context.
	instructions := []struct {
		instruction []Instruction
		status      Status
	}{
		{[]Instruction{{NUM_EXECUTABLE_OPCODES - 1, 0x0101}, {DATA, 0x0001}, {STOP, 0}}, STOPPED},
		{[]Instruction{{NUM_EXECUTABLE_OPCODES, 0}}, ERROR},
		{[]Instruction{{NUM_EXECUTABLE_OPCODES + 1, 0}}, ERROR},
	}

	for _, test := range instructions {
		ctxt := getEmptyContext()
		// Get tosca.Parameters
		ctxt.params = tosca.Parameters{
			Input:  []byte{},
			Static: false,
			Gas:    10,
			Code:   []byte{0x0},
		}
		ctxt.code = test.instruction

		// Run testing code
		steps(&ctxt, false)

		if ctxt.status != test.status {
			t.Errorf("unexpected status: want STOPPED, got %v", ctxt.status)
		}
	}
}

func TestStepsDoesNotExecuteCodeIfStatic(t *testing.T) {

	tests := map[string]struct {
		instructions []Instruction
		status       Status
	}{
		"mstore": {[]Instruction{{PUSH1, 0}, {PUSH1, 0}, {MSTORE, 0}}, STOPPED},
		"sstore": {[]Instruction{{PUSH1, 0}, {PUSH1, 0}, {SSTORE, 0}}, ERROR},
		"LOG0":   {[]Instruction{{PUSH1, 0}, {LOG0, 0}}, ERROR},
		"LOG1":   {[]Instruction{{PUSH1, 0}, {PUSH1, 0}, {LOG1, 0}}, ERROR},
		"LOG2": {[]Instruction{{PUSH1, 0}, {PUSH1, 0}, {PUSH1, 0}, {LOG2, 0}},
			ERROR},
		"LOG3": {[]Instruction{{PUSH1, 0}, {PUSH1, 0}, {PUSH1, 0}, {PUSH1, 0},
			{LOG3, 0}}, ERROR},
		"LOG4": {[]Instruction{{PUSH1, 0}, {PUSH1, 0}, {PUSH1, 0}, {PUSH1, 0},
			{PUSH1, 0}, {LOG3, 0}}, ERROR},
		"CREATE":       {[]Instruction{{PUSH1, 0}, {PUSH1, 0}, {CREATE, 0}}, ERROR},
		"CREATE2":      {[]Instruction{{PUSH1, 0}, {PUSH1, 0}, {CREATE2, 0}}, ERROR},
		"SELFDESTRUCT": {[]Instruction{{PUSH1, 0}, {SELFDESTRUCT, 0}}, ERROR},
		"TSTORE":       {[]Instruction{{PUSH1, 0}, {PUSH1, 0}, {TSTORE, 0}}, ERROR},
		"CALL":         {[]Instruction{{PUSH1, 1}, {PUSH1, 1}, {PUSH1, 1}, {CALL, 0}}, ERROR},
	}

	for name, test := range tests {
		t.Run(fmt.Sprintf("%v", name), func(t *testing.T) {
			ctxt := getEmptyContext()
			// Get tosca.Parameters
			ctxt.params = tosca.Parameters{
				Input:  []byte{},
				Static: true,
				Gas:    10,
				Code:   []byte{0x0},
			}
			ctxt.code = test.instructions

			// Run testing code
			steps(&ctxt, false)

			if ctxt.status != test.status {
				t.Errorf("unexpected status: want %v, got %v", test.status, ctxt.status)
			}
		})
	}
}

func TestStepsFailsOnTooLittleGas(t *testing.T) {
	// Create execution context.
	ctxt := getEmptyContext()
	instructions := []Instruction{
		{PUSH1, 0},
	}

	// Get tosca.Parameters
	ctxt.params = tosca.Parameters{
		Input:  []byte{},
		Static: false,
		Gas:    2,
		Code:   []byte{0x0},
	}
	ctxt.gas = 2
	ctxt.code = instructions

	// Run testing code
	steps(&ctxt, false)

	if ctxt.status != OUT_OF_GAS {
		t.Errorf("unexpected status: want OUT_OF_GAS, got %v", ctxt.status)
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

func benchmarkFib(b *testing.B, arg int, with_super_instructions bool) {
	example := getFibExample()

	// Convert example to LFVM format.
	converted := convert(example.code, ConversionConfig{})

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
		ctxt.status = RUNNING
		ctxt.gas = 1 << 31
		ctxt.stack.stack_ptr = 0

		// Run the code (actual benchmark).
		run(&ctxt)

		// Check the result.
		if ctxt.status != RETURNED {
			b.Fatalf("execution failed: status is %v", ctxt.status)
		}

		size := ctxt.result_size
		if size.Uint64() != 32 {
			b.Fatalf("unexpected length of end; wanted 32, got %d", size.Uint64())
		}
		res := make([]byte, size.Uint64())
		offset := ctxt.result_offset
		ctxt.memory.CopyData(offset.Uint64(), res)

		got := (int(res[28]) << 24) | (int(res[29]) << 16) | (int(res[30]) << 8) | (int(res[31]) << 0)
		if wanted != got {
			b.Fatalf("unexpected result, wanted %d, got %d", wanted, got)
		}
	}
}

func BenchmarkFib10(b *testing.B) {
	benchmarkFib(b, 10, false)
}

func BenchmarkFib10_SI(b *testing.B) {
	benchmarkFib(b, 10, true)
}

var sink bool

func BenchmarkIsWriteInstruction(b *testing.B) {
	for i := 0; i < b.N; i++ {
		sink = isWriteInstruction(OpCode(i % int(NUM_EXECUTABLE_OPCODES)))
	}
}

func toKey(value byte) tosca.Key {
	res := tosca.Key{}
	res[len(res)-1] = value
	return res
}

func toWord(value byte) tosca.Word {
	res := tosca.Word{}
	res[len(res)-1] = value
	return res
}
