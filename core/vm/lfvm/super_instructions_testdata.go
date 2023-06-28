package lfvm

import (
	"github.com/holiman/uint256"
)

// ----------------------------- Super Instructions -----------------------------

type tTestDataSuperOp struct {
	name   string         // test description
	op     func(*context) // tested operation
	code   []Instruction  // input code
	data   []uint256.Int  // input data (in reverse order)
	res    []uint256.Int  // expected result
	res_pc int32          // expected code pointer
	status Status         // expected status
	gas    uint64         // required gas
}

// super operations
var testDataSuperOp = []tTestDataSuperOp{

	// operation Push1_Add
	// code[0].arg + data[0]
	{
		name: "opPush1_Add: 0 + 0",
		op:   opPush1_Add,
		code: []Instruction{
			{opcode: PUSH1_ADD, arg: 0}},
		data: []uint256.Int{
			{0, 0, 0, 0}},
		res: []uint256.Int{
			{0, 0, 0, 0}},
		status: RUNNING,
	},
	{
		name: "opPush1_Add: 1 + 0",
		op:   opPush1_Add,
		code: []Instruction{
			{opcode: PUSH1_ADD, arg: 1}},
		data: []uint256.Int{
			{0, 0, 0, 0}},
		res: []uint256.Int{
			{1, 0, 0, 0}},
		status: RUNNING,
	},
	{
		name: "opPush1_Add: 0 + 1",
		op:   opPush1_Add,
		code: []Instruction{
			{opcode: PUSH1_ADD, arg: 0}},
		data: []uint256.Int{
			{1, 0, 0, 0}},
		res: []uint256.Int{
			{1, 0, 0, 0}},
		status: RUNNING,
	},
	{
		name: "opPush1_Add: 1 + 1",
		op:   opPush1_Add,
		code: []Instruction{
			{opcode: PUSH1_ADD, arg: 1}},
		data: []uint256.Int{
			{1, 0, 0, 0}},
		res: []uint256.Int{
			{2, 0, 0, 0}},
		status: RUNNING,
	},
	{
		name: "opPush1_Add: 123 + 456",
		op:   opPush1_Add,
		code: []Instruction{
			{opcode: PUSH1_ADD, arg: 123}},
		data: []uint256.Int{
			{456, 0, 0, 0}},
		res: []uint256.Int{
			{579, 0, 0, 0}},
		status: RUNNING,
	},
	{
		name: "opPush1_Add: 0xFFFF + 0",
		op:   opPush1_Add,
		code: []Instruction{
			{opcode: PUSH1_ADD, arg: 0xFFFF}},
		data: []uint256.Int{
			{0, 0, 0, 0}},
		res: []uint256.Int{
			{0xFFFF, 0, 0, 0}},
		status: RUNNING,
	},
	{
		name: "opPush1_Add: 0xFFFF + 0x01",
		op:   opPush1_Add,
		code: []Instruction{
			{opcode: PUSH1_ADD, arg: 0xFFFF}},
		data: []uint256.Int{
			{0x01, 0, 0, 0}},
		res: []uint256.Int{
			{0x010000, 0, 0, 0}},
		status: RUNNING,
	},
	{
		name: "opPush1_Add: 0xFFFF + max64 (overflow to 2nd uint64)",
		op:   opPush1_Add,
		code: []Instruction{
			{opcode: PUSH1_ADD, arg: 0xFFFF}},
		data: []uint256.Int{
			{0xFFFFFFFFFFFFFFFF, 0, 0, 0}},
		res: []uint256.Int{
			{0xFFFE, 0x01, 0, 0}},
		status: RUNNING,
	},
	{
		name: "opPush1_Add: 0xFFFF + max256 (overflow uint64)",
		op:   opPush1_Add,
		code: []Instruction{
			{opcode: PUSH1_ADD, arg: 0xFFFF}},
		data: []uint256.Int{
			{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF}},
		res: []uint256.Int{
			{0xFFFE, 0x00, 0x00, 0x00}},
		status: RUNNING,
	},
	{
		name: "opPush1_Add: 0xFFFF + max256 (with prev data, overflow uint64)",
		op:   opPush1_Add,
		code: []Instruction{
			{opcode: PUSH1_ADD, arg: 0xFFFF},
			{opcode: AND, arg: 0x7777}},
		data: []uint256.Int{
			{0x1234567890ABCDEF, 0x1234567890ABCDEF, 0x1234567890ABCDEF, 0x1234567890ABCDEF},
			{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF}},
		res: []uint256.Int{
			{0x1234567890ABCDEF, 0x1234567890ABCDEF, 0x1234567890ABCDEF, 0x1234567890ABCDEF},
			{0xFFFE, 0x00, 0x00, 0x00}},
		status: RUNNING,
	},
	{
		name: "opPush1_Add: 0xFFFF + (max256-0xFFFE) (overflow uint256)",
		op:   opPush1_Add,
		code: []Instruction{
			{opcode: PUSH1_ADD, arg: 0xFFFF}},
		data: []uint256.Int{
			{0xFFFFFFFFFFFF0001, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF}},
		res: []uint256.Int{
			{0x00, 0x00, 0x00, 0x00}},
		status: RUNNING,
	},

	// operation Push1_Shl
	// data[0] = data, code[0].arg = shift
	{
		name: "opPush1_Shl: left shift 0 on number 0",
		op:   opPush1_Shl,
		code: []Instruction{
			{opcode: PUSH1_SHL, arg: 0x00}},
		data: []uint256.Int{
			{0x00, 0x00, 0x00, 0x00}},
		res: []uint256.Int{
			{0x00, 0x00, 0x00, 0x00}},
		status: RUNNING,
	},
	{
		name: "opPush1_Shl: left shift 0 on number 0xFF0..0",
		op:   opPush1_Shl,
		code: []Instruction{
			{opcode: PUSH1_SHL, arg: 0x00}},
		data: []uint256.Int{
			{0x00, 0x00, 0x00, 0xFF00000000000000}},
		res: []uint256.Int{
			{0x00, 0x00, 0x00, 0xFF00000000000000}},
		status: RUNNING,
	},
	{
		name: "opPush1_Shl: left shift 0 on number 0x00AB0..0",
		op:   opPush1_Shl,
		code: []Instruction{
			{opcode: PUSH1_SHL, arg: 0x00}},
		data: []uint256.Int{
			{0x00, 0x00, 0x00, 0x00AB000000000000}},
		res: []uint256.Int{
			{0x00, 0x00, 0x00, 0x00AB000000000000}},
		status: RUNNING,
	},
	{
		name: "opPush1_Shl: left shift 1",
		op:   opPush1_Shl,
		code: []Instruction{
			{opcode: PUSH1_SHL, arg: 0x01}},
		data: []uint256.Int{
			{0x00, 0x00, 0x00, 0x123456789ABCDEF0}},
		res: []uint256.Int{
			{0x00, 0x00, 0x00, 0x2468ACF13579BDE0}},
		status: RUNNING,
	},
	{
		name: "opPush1_Shl: left shift 0 on number 0x63F..F",
		op:   opPush1_Shl,
		code: []Instruction{
			{opcode: PUSH1_SHL, arg: 0x00}},
		data: []uint256.Int{
			{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0x63FFFFFFFFFFFFFF}},
		res: []uint256.Int{
			{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0x63FFFFFFFFFFFFFF}},
		status: RUNNING,
	},
	{
		name: "opPush1_Shl: left shift 1 on number 0xFF84F..F",
		op:   opPush1_Shl,
		code: []Instruction{
			{opcode: PUSH1_SHL, arg: 0x01}},
		data: []uint256.Int{
			{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFF84FFFFFFFFFFFF}},
		res: []uint256.Int{
			{0xFFFFFFFFFFFFFFFE, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFF09FFFFFFFFFFFF}},
		status: RUNNING,
	},
	{
		name: "opPush1_Shl: left shift 8",
		op:   opPush1_Shl,
		code: []Instruction{
			{opcode: PUSH1_SHL, arg: 0x08}},
		data: []uint256.Int{
			{0x1234000000000000, 0x5678000000000000, 0x9ABC000000000000, 0xDEF0000000000000}},
		res: []uint256.Int{
			{0x3400000000000000, 0x7800000000000012, 0xBC00000000000056, 0xF00000000000009A}},
		status: RUNNING,
	},
	{
		name: "opPush1_Shl: left shift 64",
		op:   opPush1_Shl,
		code: []Instruction{
			{opcode: PUSH1_SHL, arg: 64}},
		data: []uint256.Int{
			{0x1234000000000000, 0x5678000000000000, 0x9ABC000000000000, 0xDEF0000000000000}},
		res: []uint256.Int{
			{0x00, 0x1234000000000000, 0x5678000000000000, 0x9ABC000000000000}},
		status: RUNNING,
	},
	{
		name: "opPush1_Shl: left shift 256",
		op:   opPush1_Shl,
		code: []Instruction{
			{opcode: PUSH1_SHL, arg: 256}},
		data: []uint256.Int{
			{0x00, 0x00, 0x00, 0x00},
			{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFF32, 0xFFFFFFFFFFFFFFFF}},
		res: []uint256.Int{
			{0x00, 0x00, 0x00, 0x00},
			{0x00, 0x00, 0x00, 0x00}},
		status: RUNNING,
	},
	{
		name: "opPush1_Shl: left shift 0xFFFF",
		op:   opPush1_Shl,
		code: []Instruction{
			{opcode: PUSH1_SHL, arg: 0xFFFF}},
		data: []uint256.Int{
			{0x00, 0x00, 0x00, 0x00},
			{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFF32, 0xFFFFFFFFFFFFFFFF}},
		res: []uint256.Int{
			{0x00, 0x00, 0x00, 0x00},
			{0x00, 0x00, 0x00, 0x00}},
		status: RUNNING,
	},

	// operation Push2_Jump
	// code[0].arg is a jump (-1), stack ????
	{
		name: "opPush2_Jump: all zeros, invalid opcode",
		op:   opPush2_Jump,
		code: []Instruction{
			{opcode: NOOP, arg: 0x00}},
		data: []uint256.Int{
			{0x00, 0x00, 0x00, 0x00},
			{0x00, 0x00, 0x00, 0x00}},
		res: []uint256.Int{
			{0x00, 0x00, 0x00, 0x00},
			{0x00, 0x00, 0x00, 0x00}},
		res_pc: 0x00 - 0x01,
		status: ERROR,
	},
	{
		name: "opPush2_Jump: all zeros, valid opcode",
		op:   opPush2_Jump,
		code: []Instruction{
			{opcode: JUMPDEST, arg: 0x00}},
		data: []uint256.Int{
			{0x00, 0x00, 0x00, 0x00},
			{0x00, 0x00, 0x00, 0x00}},
		res: []uint256.Int{
			{0x00, 0x00, 0x00, 0x00},
			{0x00, 0x00, 0x00, 0x00}},
		res_pc: 0x00 - 0x01,
		status: RUNNING,
	},
	{
		name: "opPush2_Jump: no data",
		op:   opPush2_Jump,
		code: []Instruction{
			{opcode: JUMPDEST, arg: 0x00}},
		data:   []uint256.Int{},
		res:    []uint256.Int{},
		res_pc: 0x00 - 0x01,
		status: RUNNING,
	},
	{
		name: "opPush2_Jump: jump 1, invalid opcode",
		op:   opPush2_Jump,
		code: []Instruction{
			{opcode: JUMPDEST, arg: 0x01},
			{opcode: NOOP, arg: 0x00}},
		data:   []uint256.Int{},
		res:    []uint256.Int{},
		res_pc: 0x01 - 0x01,
		status: ERROR,
	},
	{
		name: "opPush2_Jump: jump 1, valid opcode",
		op:   opPush2_Jump,
		code: []Instruction{
			{opcode: NOOP, arg: 0x01},
			{opcode: JUMPDEST, arg: 0x00}},
		data:   []uint256.Int{},
		res:    []uint256.Int{},
		res_pc: 0x01 - 0x01,
		status: RUNNING,
	},

	// operation Push2_Jumpi
	// if data[0] is not zero, then code[0].arg is a jump (-1)
	{
		name: "opPush2_Jumpi: all zeros",
		op:   opPush2_Jumpi,
		code: []Instruction{
			{opcode: NOOP, arg: 0x00}},
		data: []uint256.Int{
			{0x00, 0x00, 0x00, 0x00},
			{0x00, 0x00, 0x00, 0x00}},
		res: []uint256.Int{
			{0x00, 0x00, 0x00, 0x00}},
		res_pc: 0x00,
		status: RUNNING,
	},
	{
		name: "opPush2_Jumpi: invalid opcode",
		op:   opPush2_Jumpi,
		code: []Instruction{
			{opcode: NOOP, arg: 0x00}},
		data: []uint256.Int{
			{0x01, 0x00, 0x00, 0x00}},
		res:    []uint256.Int{},
		res_pc: 0x00 - 0x01,
		status: ERROR,
	},
	{
		name: "opPush2_Jumpi: valid opcode",
		op:   opPush2_Jumpi,
		code: []Instruction{
			{opcode: JUMPDEST, arg: 0x00}},
		data: []uint256.Int{
			{0x01, 0x00, 0x00, 0x00}},
		res:    []uint256.Int{},
		res_pc: 0x00 - 0x01,
		status: RUNNING,
	},
	{
		name: "opPush2_Jumpi: jump 1, invalid opcode",
		op:   opPush2_Jumpi,
		code: []Instruction{
			{opcode: JUMPDEST, arg: 0x01},
			{opcode: NOOP, arg: 0x00}},
		data: []uint256.Int{
			{0x02, 0x00, 0x00, 0x00}},
		res:    []uint256.Int{},
		res_pc: 0x01 - 0x01,
		status: ERROR,
	},
	{
		name: "opPush2_Jumpi: jump 1, valid opcode",
		op:   opPush2_Jumpi,
		code: []Instruction{
			{opcode: NOOP, arg: 0x01},
			{opcode: JUMPDEST, arg: 0x00}},
		data: []uint256.Int{
			{0x00, 0x00, 0x00, 0x8000000000000000}},
		res:    []uint256.Int{},
		res_pc: 0x01 - 0x01,
		status: RUNNING,
	},

	// operation Pop_Jump
	// data[0] throw away, data[1] is a jump (-1)
	{
		name: "opPop_Jump: all zeros, invalid jump 0",
		op:   opPop_Jump,
		code: []Instruction{},
		data: []uint256.Int{
			{0x00, 0x00, 0x00, 0x00},
			{0x00, 0x00, 0x00, 0x00}},
		res:    []uint256.Int{},
		res_pc: 0x00,
		status: ERROR,
	},
	{
		name: "opPop_Jump: all zeros, valid jump 1",
		op:   opPop_Jump,
		code: []Instruction{},
		data: []uint256.Int{
			{0x00, 0x00, 0x00, 0x00},
			{0x01, 0x00, 0x00, 0x00}},
		res:    []uint256.Int{},
		res_pc: 0x01 - 0x01,
		status: ERROR,
	},
	{
		name: "opPop_Jump: all zeros, invalid jump 2",
		op:   opPop_Jump,
		code: []Instruction{},
		data: []uint256.Int{
			{0x00, 0x00, 0x00, 0x00},
			{0x02, 0x00, 0x00, 0x00}},
		res:    []uint256.Int{},
		res_pc: 0x00,
		status: ERROR,
	},

	// operation IsZero_Push2_Jumpi
	// if data[0] is zero, then code[0] is a jump (-1)
	{
		name: "opIsZero_Push2_Jumpi: all zeros, invalid opcode",
		op:   opIsZero_Push2_Jumpi,
		code: []Instruction{
			{opcode: NOOP, arg: 0x00}},
		data: []uint256.Int{
			{0x00, 0x00, 0x00, 0x00}},
		res:    []uint256.Int{},
		res_pc: 0x00 - 0x01,
		status: ERROR,
	},
	{
		name: "opIsZero_Push2_Jumpi: all zeros, valid opcode",
		op:   opIsZero_Push2_Jumpi,
		code: []Instruction{
			{opcode: JUMPDEST, arg: 0x00}},
		data: []uint256.Int{
			{0x00, 0x00, 0x00, 0x00}},
		res:    []uint256.Int{},
		res_pc: 0x00 - 0x01,
		status: RUNNING,
	},
	{
		name: "opIsZero_Push2_Jumpi: data is not zero",
		op:   opIsZero_Push2_Jumpi,
		code: []Instruction{
			{opcode: ISZERO_PUSH2_JUMPI, arg: 0x00}},
		data: []uint256.Int{
			{0x01, 0x00, 0x00, 0x00}},
		res:    []uint256.Int{},
		res_pc: 0x00,
		status: RUNNING,
	},
	{
		name: "opIsZero_Push2_Jumpi: jump 1, invalid opcode",
		op:   opIsZero_Push2_Jumpi,
		code: []Instruction{
			{opcode: JUMPDEST, arg: 0x01},
			{opcode: NOOP, arg: 0x00}},
		data: []uint256.Int{
			{0x00, 0x00, 0x00, 0x00}},
		res:    []uint256.Int{},
		res_pc: 0x01 - 0x01,
		status: ERROR,
	},
	{
		name: "opIsZero_Push2_Jumpi: jump 1, valid opcode",
		op:   opIsZero_Push2_Jumpi,
		code: []Instruction{
			{opcode: NOOP, arg: 0x01},
			{opcode: JUMPDEST, arg: 0x00}},
		data: []uint256.Int{
			{0x00, 0x00, 0x00, 0x00}},
		res:    []uint256.Int{},
		res_pc: 0x01 - 0x01,
		status: RUNNING,
	},

	// operation Swap2_Swap1_Pop_Jump
	// data[0] is a jump (-1), data[1] throw away, res[0] = data[2]
	{
		name: "opSwap2_Swap1_Pop_Jump: all zeros",
		op:   opSwap2_Swap1_Pop_Jump,
		code: []Instruction{},
		data: []uint256.Int{
			{0x00, 0x00, 0x00, 0x00},
			{0x00, 0x00, 0x00, 0x00},
			{0x00, 0x00, 0x00, 0x00}},
		res: []uint256.Int{
			{0x00, 0x00, 0x00, 0x00}},
		res_pc: 0x00 - 0x01,
		status: RUNNING,
	},
	{
		name: "opSwap2_Swap1_Pop_Jump: small numbers",
		op:   opSwap2_Swap1_Pop_Jump,
		code: []Instruction{},
		data: []uint256.Int{
			{0x01, 0x00, 0x00, 0x00},
			{0x02, 0x00, 0x00, 0x00},
			{0x03, 0x00, 0x00, 0x00}},
		res: []uint256.Int{
			{0x03, 0x00, 0x00, 0x00}},
		res_pc: 0x01 - 0x01,
		status: RUNNING,
	},
	{
		name: "opSwap2_Swap1_Pop_Jump: larger numbers",
		op:   opSwap2_Swap1_Pop_Jump,
		code: []Instruction{},
		data: []uint256.Int{
			{0x10, 0x00, 0x00, 0x00},
			{0x20, 0x00, 0x00, 0x00},
			{0x30, 0x00, 0x00, 0x00}},
		res: []uint256.Int{
			{0x30, 0x00, 0x00, 0x00}},
		res_pc: 0x10 - 0x01,
		status: RUNNING,
	},
	{
		name: "opSwap2_Swap1_Pop_Jump: jump",
		op:   opSwap2_Swap1_Pop_Jump,
		code: []Instruction{},
		data: []uint256.Int{
			{0x010000, 0x00, 0x00, 0x00},
			{0x20, 0x00, 0x00, 0x00},
			{0x30, 0x00, 0x00, 0x00}},
		res: []uint256.Int{
			{0x30, 0x00, 0x00, 0x00}},
		res_pc: 0x10000 - 0x01,
		status: RUNNING,
	},
	{
		name: "opSwap2_Swap1_Pop_Jump: big jump",
		op:   opSwap2_Swap1_Pop_Jump,
		code: []Instruction{},
		data: []uint256.Int{
			{0x80000000, 0x00, 0x00, 0x00},
			{0x20, 0x00, 0x00, 0x00},
			{0x30, 0x00, 0x00, 0x00}},
		res: []uint256.Int{
			{0x30, 0x00, 0x00, 0x00}},
		res_pc: 0x80000000 - 0x01,
		status: RUNNING,
	},
	{
		name: "opSwap2_Swap1_Pop_Jump: jump overflow",
		op:   opSwap2_Swap1_Pop_Jump,
		code: []Instruction{},
		data: []uint256.Int{
			{0x80000001, 0x00, 0x00, 0x00},
			{0x20, 0x00, 0x00, 0x00},
			{0x30, 0x00, 0x00, 0x00}},
		res: []uint256.Int{
			{0x30, 0x00, 0x00, 0x00}},
		res_pc: -0x80000000,
		status: ERROR, // ????
	},
	{
		name: "opSwap2_Swap1_Pop_Jump: jump negative number",
		op:   opSwap2_Swap1_Pop_Jump,
		code: []Instruction{},
		data: []uint256.Int{
			{0xFFFFFFFF, 0x00, 0x00, 0x00},
			{0x20, 0x00, 0x00, 0x00},
			{0x30, 0x00, 0x00, 0x00}},
		res: []uint256.Int{
			{0x30, 0x00, 0x00, 0x00}},
		res_pc: -2,
		status: ERROR, // ????
	},
	{
		name: "opSwap2_Swap1_Pop_Jump: jump truncate to int32",
		op:   opSwap2_Swap1_Pop_Jump,
		code: []Instruction{},
		data: []uint256.Int{
			{0x100000001, 0x00, 0x00, 0x00},
			{0x20, 0x00, 0x00, 0x00},
			{0x30, 0x00, 0x00, 0x00}},
		res: []uint256.Int{
			{0x30, 0x00, 0x00, 0x00}},
		res_pc: 0,
		status: RUNNING,
	},
	{
		name: "opSwap2_Swap1_Pop_Jump: big numbers (with data in stack)",
		op:   opSwap2_Swap1_Pop_Jump,
		code: []Instruction{},
		data: []uint256.Int{
			{0x1111111111111111, 0x2222222222222222, 0x3333333333333333, 0x4444444444444444},
			{0x5555555555555555, 0x6666666666666666, 0x7777777777777777, 0x8888888888888888},
			{0x9999, 0x00, 0x00, 0x00},
			{0xDDDDDDDDDDDDDDDD, 0xEEEEEEEEEEEEEEEE, 0x0000000000000000, 0xFFFFFFFFFFFFFFFF},
			{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF}},
		res: []uint256.Int{
			{0x1111111111111111, 0x2222222222222222, 0x3333333333333333, 0x4444444444444444},
			{0x5555555555555555, 0x6666666666666666, 0x7777777777777777, 0x8888888888888888},
			{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF}},
		res_pc: 0x9999 - 0x01,
		status: RUNNING,
	},

	// operation Push1_Push4_Dup3
	// high byte from code[0].arg in stack, code[1].arg * 0x10000 + code[2].arg in stack, then dup(3)
	{
		name: "opPush1_Push4_Dup3: all zeros",
		op:   opPush1_Push4_Dup3,
		code: []Instruction{
			{opcode: PUSH1_PUSH4_DUP3, arg: 0x00},
			{opcode: PUSH1_PUSH4_DUP3, arg: 0x00},
			{opcode: PUSH1_PUSH4_DUP3, arg: 0x00}},
		data: []uint256.Int{
			{0x00, 0x00, 0x00, 0x00}},
		res: []uint256.Int{
			{0x00, 0x00, 0x00, 0x00},
			{0x00, 0x00, 0x00, 0x00},
			{0x00, 0x00, 0x00, 0x00},
			{0x00, 0x00, 0x00, 0x00}},
		res_pc: 0x02,
		status: RUNNING,
	},
	{
		name: "opPush1_Push4_Dup3: small numbers",
		op:   opPush1_Push4_Dup3,
		code: []Instruction{
			{opcode: NOOP, arg: 0x01},
			{opcode: NOOP, arg: 0x02},
			{opcode: NOOP, arg: 0x03}},
		data: []uint256.Int{
			{0x04, 0x00, 0x00, 0x00}},
		res: []uint256.Int{
			{0x04, 0x00, 0x00, 0x00},
			{0x00, 0x00, 0x00, 0x00},
			{0x020003, 0x00, 0x00, 0x00},
			{0x04, 0x00, 0x00, 0x00}},
		res_pc: 0x02,
		status: RUNNING,
	},
	{
		name: "opPush1_Push4_Dup3: larger numbers",
		op:   opPush1_Push4_Dup3,
		code: []Instruction{
			{opcode: NOOP, arg: 0x10},
			{opcode: NOOP, arg: 0x20},
			{opcode: NOOP, arg: 0x30}},
		data: []uint256.Int{
			{0x40, 0x50, 0x60, 0x70}},
		res: []uint256.Int{
			{0x40, 0x50, 0x60, 0x70},
			{0x00, 0x00, 0x00, 0x00},
			{0x200030, 0x00, 0x00, 0x00},
			{0x40, 0x50, 0x60, 0x70}},
		res_pc: 0x02,
		status: RUNNING,
	},
	{
		name: "opPush1_Push4_Dup3: big numbers",
		op:   opPush1_Push4_Dup3,
		code: []Instruction{
			{opcode: NOOP, arg: 0xFEDC},
			{opcode: NOOP, arg: 0x1234},
			{opcode: NOOP, arg: 0x5678}},
		data: []uint256.Int{
			{0x40, 0x50, 0x60, 0x70}},
		res: []uint256.Int{
			{0x40, 0x50, 0x60, 0x70},
			{0xFE, 0x00, 0x00, 0x00},
			{0x12345678, 0x00, 0x00, 0x00},
			{0x40, 0x50, 0x60, 0x70}},
		res_pc: 0x02,
		status: RUNNING,
	},

	// operation And_Swap1_Pop_Swap2_Swap1
	// res[0] = data[3] & data[4], res[1] = data[0], res[2] = data[1], data[2] throw away
	{
		name: "opAnd_Swap1_Pop_Swap2_Swap1: all zeros",
		op:   opAnd_Swap1_Pop_Swap2_Swap1,
		code: []Instruction{},
		data: []uint256.Int{
			{0x00, 0x00, 0x00, 0x00},
			{0x00, 0x00, 0x00, 0x00},
			{0x00, 0x00, 0x00, 0x00},
			{0x00, 0x00, 0x00, 0x00},
			{0x00, 0x00, 0x00, 0x00}},
		res: []uint256.Int{
			{0x00, 0x00, 0x00, 0x00},
			{0x00, 0x00, 0x00, 0x00},
			{0x00, 0x00, 0x00, 0x00}},
		status: RUNNING,
	},
	{
		name: "opAnd_Swap1_Pop_Swap2_Swap1: small numbers",
		op:   opAnd_Swap1_Pop_Swap2_Swap1,
		code: []Instruction{},
		data: []uint256.Int{
			{0x02, 0x00, 0x00, 0x00},
			{0x03, 0x00, 0x00, 0x00},
			{0x04, 0x00, 0x00, 0x00},
			{0x05, 0x00, 0x00, 0x00},
			{0xFF, 0x00, 0x00, 0x00}},
		res: []uint256.Int{
			{0x05, 0x00, 0x00, 0x00},
			{0x02, 0x00, 0x00, 0x00},
			{0x03, 0x00, 0x00, 0x00}},
		status: RUNNING,
	},
	{
		name: "opAnd_Swap1_Pop_Swap2_Swap1: larger numbers",
		op:   opAnd_Swap1_Pop_Swap2_Swap1,
		code: []Instruction{},
		data: []uint256.Int{
			{0x5678, 0x0A, 0x0B, 0x0C},
			{0x9ABC, 0x0D, 0x0E, 0x0F},
			{0xDEF0, 0x04, 0x03, 0x02},
			{0x4321, 0x07, 0x06, 0x05},
			{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF}},
		res: []uint256.Int{
			{0x4321, 0x07, 0x06, 0x05},
			{0x5678, 0x0A, 0x0B, 0x0C},
			{0x9ABC, 0x0D, 0x0E, 0x0F}},
		status: RUNNING,
	},
	{
		name: "opAnd_Swap1_Pop_Swap2_Swap1: big numbers, all bites of first data (1)",
		op:   opAnd_Swap1_Pop_Swap2_Swap1,
		code: []Instruction{},
		data: []uint256.Int{
			{0x1111111111111111, 0x2222222222222222, 0x3333333333333333, 0x4444444444444444},
			{0x5555555555555555, 0x6666666666666666, 0x7777777777777777, 0x8888888888888888},
			{0x9999999999999999, 0xAAAAAAAAAAAAAAAA, 0xBBBBBBBBBBBBBBBB, 0xCCCCCCCCCCCCCCCC},
			{0xDDDDDDDDDDDDDDDD, 0xEEEEEEEEEEEEEEEE, 0x0000000000000000, 0xFFFFFFFFFFFFFFFF},
			{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF}},
		res: []uint256.Int{
			{0xDDDDDDDDDDDDDDDD, 0xEEEEEEEEEEEEEEEE, 0x0000000000000000, 0xFFFFFFFFFFFFFFFF},
			{0x1111111111111111, 0x2222222222222222, 0x3333333333333333, 0x4444444444444444},
			{0x5555555555555555, 0x6666666666666666, 0x7777777777777777, 0x8888888888888888}},
		status: RUNNING,
	},
	{
		name: "opAnd_Swap1_Pop_Swap2_Swap1: big numbers, all bites of first data (2)",
		op:   opAnd_Swap1_Pop_Swap2_Swap1,
		code: []Instruction{},
		data: []uint256.Int{
			{0x1111111111111111, 0x2222222222222222, 0x3333333333333333, 0x4444444444444444},
			{0x5555555555555555, 0x6666666666666666, 0x7777777777777777, 0x8888888888888888},
			{0x9999999999999999, 0xAAAAAAAAAAAAAAAA, 0xBBBBBBBBBBBBBBBB, 0xCCCCCCCCCCCCCCCC},
			{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF},
			{0xDDDDDDDDDDDDDDDD, 0xEEEEEEEEEEEEEEEE, 0x0000000000000000, 0xFFFFFFFFFFFFFFFF}},
		res: []uint256.Int{
			{0xDDDDDDDDDDDDDDDD, 0xEEEEEEEEEEEEEEEE, 0x0000000000000000, 0xFFFFFFFFFFFFFFFF},
			{0x1111111111111111, 0x2222222222222222, 0x3333333333333333, 0x4444444444444444},
			{0x5555555555555555, 0x6666666666666666, 0x7777777777777777, 0x8888888888888888}},
		status: RUNNING,
	},
	{
		name: "opAnd_Swap1_Pop_Swap2_Swap1: big numbers, no bites of first data (1)",
		op:   opAnd_Swap1_Pop_Swap2_Swap1,
		code: []Instruction{},
		data: []uint256.Int{
			{0x1111111111111111, 0x2222222222222222, 0x3333333333333333, 0x4444444444444444},
			{0x5555555555555555, 0x6666666666666666, 0x7777777777777777, 0x8888888888888888},
			{0x9999999999999999, 0xAAAAAAAAAAAAAAAA, 0xBBBBBBBBBBBBBBBB, 0xCCCCCCCCCCCCCCCC},
			{0xDDDDDDDDDDDDDDDD, 0xEEEEEEEEEEEEEEEE, 0x0000000000000000, 0xFFFFFFFFFFFFFFFF},
			{0x00, 0x00, 0x00, 0x00}},
		res: []uint256.Int{
			{0x00, 0x00, 0x00, 0x00},
			{0x1111111111111111, 0x2222222222222222, 0x3333333333333333, 0x4444444444444444},
			{0x5555555555555555, 0x6666666666666666, 0x7777777777777777, 0x8888888888888888}},
		status: RUNNING,
	},
	{
		name: "opAnd_Swap1_Pop_Swap2_Swap1: big numbers, no bites of first data (2)",
		op:   opAnd_Swap1_Pop_Swap2_Swap1,
		code: []Instruction{},
		data: []uint256.Int{
			{0x1111111111111111, 0x2222222222222222, 0x3333333333333333, 0x4444444444444444},
			{0x5555555555555555, 0x6666666666666666, 0x7777777777777777, 0x8888888888888888},
			{0x9999999999999999, 0xAAAAAAAAAAAAAAAA, 0xBBBBBBBBBBBBBBBB, 0xCCCCCCCCCCCCCCCC},
			{0x00, 0x00, 0x00, 0x00},
			{0xDDDDDDDDDDDDDDDD, 0xEEEEEEEEEEEEEEEE, 0x0000000000000000, 0xFFFFFFFFFFFFFFFF}},
		res: []uint256.Int{
			{0x00, 0x00, 0x00, 0x00},
			{0x1111111111111111, 0x2222222222222222, 0x3333333333333333, 0x4444444444444444},
			{0x5555555555555555, 0x6666666666666666, 0x7777777777777777, 0x8888888888888888}},
		status: RUNNING,
	},
	{
		name: "opAnd_Swap1_Pop_Swap2_Swap1: big numbers, various bites of first data (1)",
		op:   opAnd_Swap1_Pop_Swap2_Swap1,
		code: []Instruction{},
		data: []uint256.Int{
			{0x1111111111111111, 0x2222222222222222, 0x3333333333333333, 0x4444444444444444},
			{0x5555555555555555, 0x6666666666666666, 0x7777777777777777, 0x8888888888888888},
			{0x9999999999999999, 0xAAAAAAAAAAAAAAAA, 0xBBBBBBBBBBBBBBBB, 0xCCCCCCCCCCCCCCCC},
			{0xDDDDDDDDDDDDDDDD, 0xEEEEEEEEEEEEEEEE, 0x0000000000000000, 0xFFFFFFFFFFFFFFFF},
			{0xFFEEDDCCBBAA9988, 0x7766554433221100, 0x0123456789ABCDEF, 0xFEDCBA9876543210}},
		res: []uint256.Int{
			{0xDDCCDDCC99889988, 0x6666444422220000, 0x0000000000000000, 0xFEDCBA9876543210},
			{0x1111111111111111, 0x2222222222222222, 0x3333333333333333, 0x4444444444444444},
			{0x5555555555555555, 0x6666666666666666, 0x7777777777777777, 0x8888888888888888}},
		status: RUNNING,
	},
	{
		name: "opAnd_Swap1_Pop_Swap2_Swap1: big numbers, various bites of first data (2)",
		op:   opAnd_Swap1_Pop_Swap2_Swap1,
		code: []Instruction{},
		data: []uint256.Int{
			{0x1111111111111111, 0x2222222222222222, 0x3333333333333333, 0x4444444444444444},
			{0x5555555555555555, 0x6666666666666666, 0x7777777777777777, 0x8888888888888888},
			{0x9999999999999999, 0xAAAAAAAAAAAAAAAA, 0xBBBBBBBBBBBBBBBB, 0xCCCCCCCCCCCCCCCC},
			{0xFFEEDDCCBBAA9988, 0x7766554433221100, 0x0123456789ABCDEF, 0xFEDCBA9876543210},
			{0xDDDDDDDDDDDDDDDD, 0xEEEEEEEEEEEEEEEE, 0x0000000000000000, 0xFFFFFFFFFFFFFFFF}},
		res: []uint256.Int{
			{0xDDCCDDCC99889988, 0x6666444422220000, 0x0000000000000000, 0xFEDCBA9876543210},
			{0x1111111111111111, 0x2222222222222222, 0x3333333333333333, 0x4444444444444444},
			{0x5555555555555555, 0x6666666666666666, 0x7777777777777777, 0x8888888888888888}},
		status: RUNNING,
	},
	{
		name: "opAnd_Swap1_Pop_Swap2_Swap1: big numbers (with data in stack and code)",
		op:   opAnd_Swap1_Pop_Swap2_Swap1,
		code: []Instruction{
			{opcode: NOOP, arg: 0xFEDC}},
		data: []uint256.Int{
			{0x123456789ABCDEF0, 0x123456789ABCDEF0, 0x123456789ABCDEF0, 0x123456789ABCDEF0},
			{0x1111111111111111, 0x2222222222222222, 0x3333333333333333, 0x4444444444444444},
			{0x5555555555555555, 0x6666666666666666, 0x7777777777777777, 0x8888888888888888},
			{0x9999999999999999, 0xAAAAAAAAAAAAAAAA, 0xBBBBBBBBBBBBBBBB, 0xCCCCCCCCCCCCCCCC},
			{0xFFEEDDCCBBAA9988, 0x7766554433221100, 0x0123456789ABCDEF, 0xFEDCBA9876543210},
			{0xDDDDDDDDDDDDDDDD, 0xEEEEEEEEEEEEEEEE, 0x0000000000000000, 0xFFFFFFFFFFFFFFFF}},
		res: []uint256.Int{
			{0x123456789ABCDEF0, 0x123456789ABCDEF0, 0x123456789ABCDEF0, 0x123456789ABCDEF0},
			{0xDDCCDDCC99889988, 0x6666444422220000, 0x0000000000000000, 0xFEDCBA9876543210},
			{0x1111111111111111, 0x2222222222222222, 0x3333333333333333, 0x4444444444444444},
			{0x5555555555555555, 0x6666666666666666, 0x7777777777777777, 0x8888888888888888}},
		status: RUNNING,
	},

	// operation Push1_Push1_Push1_Shl_Sub
	// lsh ((code[0].arg & 0xFF), code[1].arg) - (code[0].arg >> 8)
	{
		name: "opPush1_Push1_Push1_Shl_Sub: 0, 0 -> 0",
		op:   opPush1_Push1_Push1_Shl_Sub,
		code: []Instruction{
			{opcode: PUSH1_PUSH1_PUSH1_SHL_SUB, arg: 0x00},
			{opcode: PUSH1_PUSH1_PUSH1_SHL_SUB, arg: 0x00}},
		data: []uint256.Int{},
		res: []uint256.Int{
			{0x00, 0x00, 0x00, 0x00}},
		res_pc: 0x01,
		status: RUNNING,
	},
	{
		name: "opPush1_Push1_Push1_Shl_Sub: 1, 1 -> 2",
		op:   opPush1_Push1_Push1_Shl_Sub,
		code: []Instruction{
			{opcode: NOOP, arg: 0x01},
			{opcode: NOOP, arg: 0x01}},
		data: []uint256.Int{},
		res: []uint256.Int{
			{0x02, 0x00, 0x00, 0x00}},
		res_pc: 0x01,
		status: RUNNING,
	},
	{
		name: "opPush1_Push1_Push1_Shl_Sub: 1, 0 -> 1",
		op:   opPush1_Push1_Push1_Shl_Sub,
		code: []Instruction{
			{opcode: NOOP, arg: 0x01},
			{opcode: NOOP, arg: 0x00}},
		data: []uint256.Int{},
		res: []uint256.Int{
			{0x01, 0x00, 0x00, 0x00}},
		res_pc: 0x01,
		status: RUNNING,
	},
	{
		name: "opPush1_Push1_Push1_Shl_Sub: 0, 1 -> 0",
		op:   opPush1_Push1_Push1_Shl_Sub,
		code: []Instruction{
			{opcode: NOOP, arg: 0x00},
			{opcode: NOOP, arg: 0x01}},
		data: []uint256.Int{},
		res: []uint256.Int{
			{0x00, 0x00, 0x00, 0x00}},
		res_pc: 0x01,
		status: RUNNING,
	},
	{
		name: "opPush1_Push1_Push1_Shl_Sub: 1, 2 -> 4",
		op:   opPush1_Push1_Push1_Shl_Sub,
		code: []Instruction{
			{opcode: PUSH1_PUSH1_PUSH1_SHL_SUB, arg: 0x01},
			{opcode: PUSH1_PUSH1_PUSH1_SHL_SUB, arg: 0x02}},
		data: []uint256.Int{},
		res: []uint256.Int{
			{0x04, 0x00, 0x00, 0x00}},
		res_pc: 0x01,
		status: RUNNING,
	},
	{
		name: "opPush1_Push1_Push1_Shl_Sub: 1, 2 -> 4 (with stack)",
		op:   opPush1_Push1_Push1_Shl_Sub,
		code: []Instruction{
			{opcode: NOOP, arg: 0x01},
			{opcode: NOOP, arg: 0x02}},
		data: []uint256.Int{
			{0x1234, 0x00, 0x00, 0xFEDCBA9876543210},
			{0x123456789ABCDEF0, 0x00, 0x00, 0x9876}},
		res: []uint256.Int{
			{0x1234, 0x00, 0x00, 0xFEDCBA9876543210},
			{0x123456789ABCDEF0, 0x00, 0x00, 0x9876},
			{0x04, 0x00, 0x00, 0x00}},
		res_pc: 0x01,
		status: RUNNING,
	},
	{
		name: "opPush1_Push1_Push1_Shl_Sub: 0x25, 0x04 -> 0x0250",
		op:   opPush1_Push1_Push1_Shl_Sub,
		code: []Instruction{
			{opcode: PUSH1_PUSH1_PUSH1_SHL_SUB, arg: 0x25},
			{opcode: PUSH1_PUSH1_PUSH1_SHL_SUB, arg: 0x04}},
		data: []uint256.Int{},
		res: []uint256.Int{
			{0x0250, 0x00, 0x00, 0x00}},
		res_pc: 0x01,
		status: RUNNING,
	},
	{
		name: "opPush1_Push1_Push1_Shl_Sub: 0x1025, 0x04 -> 0x0240",
		op:   opPush1_Push1_Push1_Shl_Sub,
		code: []Instruction{
			{opcode: PUSH1_PUSH1_PUSH1_SHL_SUB, arg: 0x1025},
			{opcode: PUSH1_PUSH1_PUSH1_SHL_SUB, arg: 0x04}},
		data: []uint256.Int{},
		res: []uint256.Int{
			{0x0240, 0x00, 0x00, 0x00}},
		res_pc: 0x01,
		status: RUNNING,
	},

	// operation Swap1_Pop_Swap2_Swap1
	// res[0] = data[3], res[1] = data[0], res[2] = data[1], data[2] throw away
	{
		name: "opSwap1_Pop_Swap2_Swap1: all zeros",
		op:   opSwap1_Pop_Swap2_Swap1,
		code: []Instruction{},
		data: []uint256.Int{
			{0x00, 0x00, 0x00, 0x00},
			{0x00, 0x00, 0x00, 0x00},
			{0x00, 0x00, 0x00, 0x00},
			{0x00, 0x00, 0x00, 0x00}},
		res: []uint256.Int{
			{0x00, 0x00, 0x00, 0x00},
			{0x00, 0x00, 0x00, 0x00},
			{0x00, 0x00, 0x00, 0x00}},
		status: RUNNING,
	},
	{
		name: "opSwap1_Pop_Swap2_Swap1: small numbers",
		op:   opSwap1_Pop_Swap2_Swap1,
		code: []Instruction{},
		data: []uint256.Int{
			{0x02, 0x00, 0x00, 0x00},
			{0x03, 0x00, 0x00, 0x00},
			{0x04, 0x00, 0x00, 0x00},
			{0x05, 0x00, 0x00, 0x00}},
		res: []uint256.Int{
			{0x05, 0x00, 0x00, 0x00},
			{0x02, 0x00, 0x00, 0x00},
			{0x03, 0x00, 0x00, 0x00}},
		status: RUNNING,
	},
	{
		name: "opSwap1_Pop_Swap2_Swap1: larger numbers",
		op:   opSwap1_Pop_Swap2_Swap1,
		code: []Instruction{},
		data: []uint256.Int{
			{0x5678, 0x0A, 0x0B, 0x0C},
			{0x9ABC, 0x0D, 0x0E, 0x0F},
			{0xDEF0, 0x04, 0x03, 0x02},
			{0x4321, 0x07, 0x06, 0x05}},
		res: []uint256.Int{
			{0x4321, 0x07, 0x06, 0x05},
			{0x5678, 0x0A, 0x0B, 0x0C},
			{0x9ABC, 0x0D, 0x0E, 0x0F}},
		status: RUNNING,
	},
	{
		name: "opSwap1_Pop_Swap2_Swap1: big numbers",
		op:   opSwap1_Pop_Swap2_Swap1,
		code: []Instruction{},
		data: []uint256.Int{
			{0x1111111111111111, 0x2222222222222222, 0x3333333333333333, 0x4444444444444444},
			{0x5555555555555555, 0x6666666666666666, 0x7777777777777777, 0x8888888888888888},
			{0x9999999999999999, 0xAAAAAAAAAAAAAAAA, 0xBBBBBBBBBBBBBBBB, 0xCCCCCCCCCCCCCCCC},
			{0xDDDDDDDDDDDDDDDD, 0xEEEEEEEEEEEEEEEE, 0x0000000000000000, 0xFFFFFFFFFFFFFFFF}},
		res: []uint256.Int{
			{0xDDDDDDDDDDDDDDDD, 0xEEEEEEEEEEEEEEEE, 0x0000000000000000, 0xFFFFFFFFFFFFFFFF},
			{0x1111111111111111, 0x2222222222222222, 0x3333333333333333, 0x4444444444444444},
			{0x5555555555555555, 0x6666666666666666, 0x7777777777777777, 0x8888888888888888}},
		status: RUNNING,
	},
	{
		name: "opSwap1_Pop_Swap2_Swap1: big numbers (with data in stack and code)",
		op:   opSwap1_Pop_Swap2_Swap1,
		code: []Instruction{
			{opcode: NOOP, arg: 0xFEDC}},
		data: []uint256.Int{
			{0x123456789ABCDEF0, 0x123456789ABCDEF0, 0x123456789ABCDEF0, 0x123456789ABCDEF0},
			{0x1111111111111111, 0x2222222222222222, 0x3333333333333333, 0x4444444444444444},
			{0x5555555555555555, 0x6666666666666666, 0x7777777777777777, 0x8888888888888888},
			{0x9999999999999999, 0xAAAAAAAAAAAAAAAA, 0xBBBBBBBBBBBBBBBB, 0xCCCCCCCCCCCCCCCC},
			{0xDDDDDDDDDDDDDDDD, 0xEEEEEEEEEEEEEEEE, 0x0000000000000000, 0xFFFFFFFFFFFFFFFF}},
		res: []uint256.Int{
			{0x123456789ABCDEF0, 0x123456789ABCDEF0, 0x123456789ABCDEF0, 0x123456789ABCDEF0},
			{0xDDDDDDDDDDDDDDDD, 0xEEEEEEEEEEEEEEEE, 0x0000000000000000, 0xFFFFFFFFFFFFFFFF},
			{0x1111111111111111, 0x2222222222222222, 0x3333333333333333, 0x4444444444444444},
			{0x5555555555555555, 0x6666666666666666, 0x7777777777777777, 0x8888888888888888}},
		status: RUNNING,
	},
}

// super operations - runtime error
var testDataSuperOpError = []tTestDataSuperOp{
	/*{
		// runtime error: index out of range [-1]
		name:   "opPop_Jump: no data",
		op:     opPop_Jump,
		code:   []Instruction{},
		data:   []uint256.Int{},
		res:    []uint256.Int{},
		res_pc: 0x00,
		status: ERROR,
	},
	/**/
	/*{
		// runtime error: index out of range [-1]
		name: "opIsZero_Push2_Jumpi: no data",
		op:   opIsZero_Push2_Jumpi,
		code: []Instruction{
			{opcode: JUMPDEST, arg: 0x00}},
		data:   []uint256.Int{},
		res:    []uint256.Int{},
		res_pc: 0x00 - 0x01,
		status: RUNNING,
	}, /**/
}
