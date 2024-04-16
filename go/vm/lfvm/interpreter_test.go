//
// Copyright (c) 2024 Fantom Foundation
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at fantom.foundation/bsl11.
//
// Change Date: 2028-4-16
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by the GNU Lesser General Public Licence v3
//

package lfvm

import (
	"encoding/hex"
	"log"
	"testing"

	"github.com/ethereum/go-ethereum/params"
	"go.uber.org/mock/gomock"

	"github.com/Fantom-foundation/Tosca/go/vm"
)

// To run the benchmark use
//  go test ./core/vm/lfvm -bench=.*Fib.* --benchtime 10s

type example struct {
	code     []byte // Some contract code
	function uint32 // The identifier of the function in the contract to be called
}

const MAX_STACK_SIZE int = 1024
const GAS_START vm.Gas = 1 << 32

func getEmptyContext() context {
	code := make([]Instruction, 0)
	data := make([]byte, 0)
	return getContext(code, data, nil, 0, GAS_START, vm.R07_Istanbul)
}

func getContext(code Code, data []byte, runContext vm.RunContext, stackPtr int, gas vm.Gas, revision vm.Revision) context {

	// Create execution context.
	ctxt := context{
		params: vm.Parameters{
			Revision: revision,
			Gas:      gas,
			Input:    data,
		},
		context:  runContext,
		gas:      gas,
		stack:    NewStack(),
		memory:   NewMemory(),
		status:   RUNNING,
		code:     code,
		isBerlin: revision >= vm.R09_Berlin,
		isLondon: revision >= vm.R10_London,
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
	ctx := getEmptyContext()
	ctx.gas = 100
	ok := ctx.UseGas(10)
	if !ok {
		t.Errorf("expected not failed useGas function, got failed")
	}
	if ctx.gas != 90 {
		t.Errorf("expected gas in context is 90, got %d", ctx.gas)
	}
	ok = ctx.UseGas(100)
	if ok {
		t.Errorf("expected failed useGas function, got ok")
	}
	if ctx.gas != 90 {
		t.Errorf("expected gas in context is 90 also after failing, got %d", ctx.gas)
	}
	if ctx.status != OUT_OF_GAS {
		t.Errorf("expected OUT_OF_GAS status 6, got %d", ctx.status)
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
	mockCalls   func(*vm.MockRunContext)
	gasStart    vm.Gas
	gasConsumed vm.Gas
	gasRefund   vm.Gas
}

func getInstructions(start OpCode, end OpCode) (opCodes []OpCode) {
	for i := start; i <= end; i++ {
		opCodes = append(opCodes, OpCode(i))
	}
	return
}

func getInstructionsWithGas(start OpCode, end OpCode, gas vm.Gas) (opCodes []OpCodeWithGas) {
	for i := start; i <= end; i++ {
		opCode := OpCodeWithGas{OpCode(i), gas}
		opCodes = append(opCodes, opCode)
	}
	return
}

var fullStackFailOpCodes = []OpCode{
	MSIZE, ADDRESS, ORIGIN, CALLER, CALLVALUE, CALLDATASIZE,
	CODESIZE, GASPRICE, COINBASE, TIMESTAMP, NUMBER,
	DIFFICULTY, GASLIMIT, PC, GAS, RETURNDATASIZE,
	SELFBALANCE, CHAINID, BASEFEE,
	PUSH1_PUSH1_PUSH1_SHL_SUB,
	PUSH1_DUP1, PUSH1_PUSH1, PUSH1_PUSH4_DUP3,
}

var emptyStackFailOpCodes = []OpCode{
	POP, ADD, SUB, MUL, DIV, SDIV, MOD, SMOD, EXP, SIGNEXTEND,
	SHA3, LT, GT, SLT, SGT, EQ, AND, XOR, OR, BYTE,
	SHL, SHR, SAR, ADDMOD, MULMOD, ISZERO, NOT, BALANCE, CALLDATALOAD, EXTCODESIZE,
	BLOCKHASH, MLOAD, SLOAD, EXTCODEHASH, JUMP, SELFDESTRUCT,
	MSTORE, MSTORE8, SSTORE, JUMPI, RETURN, REVERT,
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
		func(mock *vm.MockRunContext) {
			mock.EXPECT().GetStorage(vm.Address{0}, toKey(0)).Return(toWord(0))
		}, GAS_START, 803, 0},
	{"SLOAD Berlin", []Instruction{{PUSH1, 0}, {SLOAD, 0}}, 0, 0, STOPPED, true, false,
		func(mock *vm.MockRunContext) {
			mock.EXPECT().GetStorage(vm.Address{0}, toKey(0)).Return(toWord(0))
			mock.EXPECT().IsSlotInAccessList(vm.Address{0}, toKey(0)).Return(true, true)
		}, GAS_START, 103, 0},
	{"SLOAD Berlin no slot", []Instruction{{PUSH1, 0}, {SLOAD, 0}}, 0, 0, STOPPED, true, false,
		func(mock *vm.MockRunContext) {
			mock.EXPECT().GetStorage(vm.Address{0}, toKey(0)).Return(toWord(0))
			mock.EXPECT().IsSlotInAccessList(vm.Address{0}, toKey(0)).Return(false, false)
			mock.EXPECT().AccessStorage(vm.Address{0}, toKey(0))
		}, GAS_START, 2103, 0},
	{"SSTORE same value", []Instruction{{PUSH1, 0}, {PUSH1, 0}, {SSTORE, 0}}, 0, 0, STOPPED, false, false,
		func(mock *vm.MockRunContext) {
			mock.EXPECT().GetStorage(vm.Address{0}, toKey(0)).Return(toWord(0))
			mock.EXPECT().SetStorage(vm.Address{0}, toKey(0), toWord(0))
		}, GAS_START, 806, 0},
	{"SSTORE diff value, same state as db, db is 0", []Instruction{{PUSH1, 1 << 8}, {PUSH1, 0}, {SSTORE, 0}}, 0, 0, STOPPED, false, false,
		func(mock *vm.MockRunContext) {
			mock.EXPECT().GetStorage(vm.Address{0}, toKey(0)).Return(toWord(0))
			mock.EXPECT().GetCommittedStorage(vm.Address{0}, toKey(0)).Return(toWord(0))
			mock.EXPECT().SetStorage(vm.Address{0}, toKey(0), toWord(1))
		}, GAS_START, 20006, 0},
	{"SSTORE diff value, same state as db, val is 0", []Instruction{{PUSH1, 0}, {PUSH1, 0}, {SSTORE, 0}}, 0, 0, STOPPED, false, false,
		func(mock *vm.MockRunContext) {
			mock.EXPECT().GetStorage(vm.Address{0}, toKey(0)).Return(toWord(1))
			mock.EXPECT().GetCommittedStorage(vm.Address{0}, toKey(0)).Return(toWord(1))
			mock.EXPECT().SetStorage(vm.Address{0}, toKey(0), toWord(0))
		}, GAS_START, 5006, vm.Gas(params.SstoreClearsScheduleRefundEIP2200)},
	{"SSTORE diff value, diff state as db, db it not 0, state is 0", []Instruction{{PUSH1, 1 << 8}, {PUSH1, 0}, {SSTORE, 0}}, 0, 0, STOPPED, false, false,
		func(mock *vm.MockRunContext) {
			mock.EXPECT().GetStorage(vm.Address{0}, toKey(0)).Return(toWord(0))
			mock.EXPECT().GetCommittedStorage(vm.Address{0}, toKey(0)).Return(toWord(2))
			mock.EXPECT().SetStorage(vm.Address{0}, toKey(0), toWord(1))
		}, GAS_START, 806, vm.Gas(-int(params.SstoreClearsScheduleRefundEIP2200))},
	{"SSTORE diff value, diff state as db, db it not 0, val is 0", []Instruction{{PUSH1, 0}, {PUSH1, 0}, {SSTORE, 0}}, 0, 0, STOPPED, false, false,
		func(mock *vm.MockRunContext) {
			mock.EXPECT().GetStorage(vm.Address{0}, toKey(0)).Return(toWord(1))
			mock.EXPECT().GetCommittedStorage(vm.Address{0}, toKey(0)).Return(toWord(2))
			mock.EXPECT().SetStorage(vm.Address{0}, toKey(0), toWord(0))
		}, GAS_START, 806, vm.Gas(params.SstoreClearsScheduleRefundEIP2200)},
	{"SSTORE diff value, diff state as db, db same as val, db is 0", []Instruction{{PUSH1, 0}, {PUSH1, 1 << 8}, {SSTORE, 0}}, 0, 0, STOPPED, false, false,
		func(mock *vm.MockRunContext) {
			mock.EXPECT().GetStorage(vm.Address{0}, toKey(1)).Return(toWord(1))
			mock.EXPECT().GetCommittedStorage(vm.Address{0}, toKey(1)).Return(toWord(0))
			mock.EXPECT().SetStorage(vm.Address{0}, toKey(1), toWord(0))
		}, GAS_START, 806, vm.Gas(params.SstoreSetGasEIP2200 - params.SloadGasEIP2200)},
	{"SSTORE diff value, diff state as db, db same as val, db is not 0", []Instruction{{PUSH1, 2 << 8}, {PUSH1, 0}, {SSTORE, 0}}, 0, 0, STOPPED, false, false,
		func(mock *vm.MockRunContext) {
			mock.EXPECT().GetStorage(vm.Address{0}, toKey(0)).Return(toWord(1))
			mock.EXPECT().GetCommittedStorage(vm.Address{0}, toKey(0)).Return(toWord(2))
			mock.EXPECT().SetStorage(vm.Address{0}, toKey(0), toWord(2))
		}, GAS_START, 806, vm.Gas(params.SstoreResetGasEIP2200 - params.SloadGasEIP2200)},
	{"SSTORE Berlin same value", []Instruction{{PUSH1, 0}, {PUSH1, 0}, {SSTORE, 0}}, 0, 0, STOPPED, true, false,
		func(mock *vm.MockRunContext) {
			mock.EXPECT().GetStorage(vm.Address{0}, toKey(0)).Return(toWord(0))
			mock.EXPECT().IsSlotInAccessList(vm.Address{0}, toKey(0)).Return(true, false)
			mock.EXPECT().AccessStorage(vm.Address{0}, toKey(0))
			mock.EXPECT().SetStorage(vm.Address{0}, toKey(0), toWord(0))
		}, GAS_START, 2206, 0},
	{"SSTORE Berlin diff value, same state as db, db is 0", []Instruction{{PUSH1, 1 << 8}, {PUSH1, 0}, {SSTORE, 0}}, 0, 0, STOPPED, true, false,
		func(mock *vm.MockRunContext) {
			mock.EXPECT().GetStorage(vm.Address{0}, toKey(0)).Return(toWord(0))
			mock.EXPECT().IsSlotInAccessList(vm.Address{0}, toKey(0)).Return(true, true)
			mock.EXPECT().GetCommittedStorage(vm.Address{0}, toKey(0)).Return(toWord(0))
			mock.EXPECT().SetStorage(vm.Address{0}, toKey(0), toWord(1))
		}, GAS_START, 20006, 0},
	{"SSTORE Berlin diff value, same state as db, val is 0", []Instruction{{PUSH1, 0}, {PUSH1, 0}, {SSTORE, 0}}, 0, 0, STOPPED, true, false,
		func(mock *vm.MockRunContext) {
			mock.EXPECT().GetStorage(vm.Address{0}, toKey(0)).Return(toWord(1))
			mock.EXPECT().IsSlotInAccessList(vm.Address{0}, toKey(0)).Return(true, true)
			mock.EXPECT().GetCommittedStorage(vm.Address{0}, toKey(0)).Return(toWord(1))
			mock.EXPECT().SetStorage(vm.Address{0}, toKey(0), toWord(0))
		}, GAS_START, 2906, vm.Gas(params.SstoreClearsScheduleRefundEIP2200)},
	{"SSTORE Berlin diff value, diff state as db, db it not 0, state is 0", []Instruction{{PUSH1, 1 << 8}, {PUSH1, 0}, {SSTORE, 0}}, 0, 0, STOPPED, true, false,
		func(mock *vm.MockRunContext) {
			mock.EXPECT().GetStorage(vm.Address{0}, toKey(0)).Return(toWord(0))
			mock.EXPECT().IsSlotInAccessList(vm.Address{0}, toKey(0)).Return(true, true)
			mock.EXPECT().GetCommittedStorage(vm.Address{0}, toKey(0)).Return(toWord(2))
			mock.EXPECT().SetStorage(vm.Address{0}, toKey(0), toWord(1))
		}, GAS_START, 106, vm.Gas(-int(params.SstoreClearsScheduleRefundEIP2200))},
	{"SSTORE Berlin diff value, diff state as db, db it not 0, val is 0", []Instruction{{PUSH1, 0}, {PUSH1, 0}, {SSTORE, 0}}, 0, 0, STOPPED, true, false,
		func(mock *vm.MockRunContext) {
			mock.EXPECT().GetStorage(vm.Address{0}, toKey(0)).Return(toWord(1))
			mock.EXPECT().IsSlotInAccessList(vm.Address{0}, toKey(0)).Return(true, true)
			mock.EXPECT().GetCommittedStorage(vm.Address{0}, toKey(0)).Return(toWord(2))
			mock.EXPECT().SetStorage(vm.Address{0}, toKey(0), toWord(0))
		}, GAS_START, 106, vm.Gas(params.SstoreClearsScheduleRefundEIP2200)},
	{"SSTORE Berlin diff value, diff state as db, db same as val, db is 0", []Instruction{{PUSH1, 0}, {PUSH1, 1 << 8}, {SSTORE, 0}}, 0, 0, STOPPED, true, false,
		func(mock *vm.MockRunContext) {
			mock.EXPECT().GetStorage(vm.Address{0}, toKey(1)).Return(toWord(1))
			mock.EXPECT().IsSlotInAccessList(vm.Address{0}, toKey(1)).Return(true, true)
			mock.EXPECT().GetCommittedStorage(vm.Address{0}, toKey(1)).Return(toWord(0))
			mock.EXPECT().SetStorage(vm.Address{0}, toKey(1), toWord(0))
		}, GAS_START, 106, vm.Gas(params.SstoreSetGasEIP2200 - params.WarmStorageReadCostEIP2929)},
	{"SSTORE Berlin diff value, diff state as db, db same as val, db is not 0", []Instruction{{PUSH1, 2 << 8}, {PUSH1, 0}, {SSTORE, 0}}, 0, 0, STOPPED, true, false,
		func(mock *vm.MockRunContext) {
			mock.EXPECT().GetStorage(vm.Address{0}, toKey(0)).Return(toWord(1))
			mock.EXPECT().IsSlotInAccessList(vm.Address{0}, toKey(0)).Return(true, true)
			mock.EXPECT().GetCommittedStorage(vm.Address{0}, toKey(0)).Return(toWord(2))
			mock.EXPECT().SetStorage(vm.Address{0}, toKey(0), toWord(2))
		}, GAS_START, 106, vm.Gas((params.SstoreResetGasEIP2200 - params.ColdSloadCostEIP2929) - params.WarmStorageReadCostEIP2929)},
}

type OpCodeWithGas struct {
	OpCode
	gas vm.Gas
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
			runContext := vm.NewMockRunContext(ctrl)
			if test.mockCalls != nil {
				test.mockCalls(runContext)
			}
			revision := vm.R07_Istanbul
			if test.isBerlin {
				revision = vm.R09_Berlin
			}
			if test.isLondon {
				revision = vm.R10_London
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
	converted, err := convert(example.code, with_super_instructions)
	if err != nil {
		b.Fatalf("error converting code: %v", err)
	}

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
		params: vm.Parameters{
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

func toKey(value byte) vm.Key {
	res := vm.Key{}
	res[len(res)-1] = value
	return res
}

func toWord(value byte) vm.Word {
	res := vm.Word{}
	res[len(res)-1] = value
	return res
}
