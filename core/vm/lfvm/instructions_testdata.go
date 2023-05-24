package lfvm

import (
	"github.com/holiman/uint256"
)

type tTestDataOp struct {
	name   string         // test description
	op     func(*context) // tested operation
	data   []uint256.Int  // input data (in reverse order)
	res    uint256.Int    // expected result
	status Status         // expected status
	gas    uint64         // required gas
}

// bitwise logic operations (And, Or, Not, Xor, Byte, Shl, Shr, Sar)

var testDataBitwiseLogicOp = []tTestDataOp{

	// operation And
	{
		name: "opAnd: 0xF..F & various bits",
		op:   opAnd,
		data: []uint256.Int{
			{0x123456789ABCDEF0, 0x23456789ABCDEF01, 0x3456789ABCDEF012, 0x456789ABCDEF0123},
			{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF}},
		res:    uint256.Int{0x123456789ABCDEF0, 0x23456789ABCDEF01, 0x3456789ABCDEF012, 0x456789ABCDEF0123},
		status: RUNNING,
	},
	{
		name: "opAnd: various bits & 0xF..F",
		op:   opAnd,
		data: []uint256.Int{
			{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF},
			{0x123456789ABCDEF0, 0x23456789ABCDEF01, 0x3456789ABCDEF012, 0x456789ABCDEF0123}},
		res:    uint256.Int{0x123456789ABCDEF0, 0x23456789ABCDEF01, 0x3456789ABCDEF012, 0x456789ABCDEF0123},
		status: RUNNING,
	},
	{
		name: "opAnd: 0xF..F & 0x0..0",
		op:   opAnd,
		data: []uint256.Int{
			{0x00, 0x00, 0x00, 0x00},
			{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF}},
		res:    uint256.Int{0x00, 0x00, 0x00, 0x00},
		status: RUNNING,
	},
	{
		name: "opAnd: 0x0..0 & 0xF..F",
		op:   opAnd,
		data: []uint256.Int{
			{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF},
			{0x00, 0x00, 0x00, 0x00}},
		res:    uint256.Int{0x00, 0x00, 0x00, 0x00},
		status: RUNNING,
	},
	{
		name: "opAnd: 0xF..F & 0xF..F",
		op:   opAnd,
		data: []uint256.Int{
			{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF},
			{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF}},
		res:    uint256.Int{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF},
		status: RUNNING,
	},
	{
		name: "opAnd: 0x0..0 & 0x0..0",
		op:   opAnd,
		data: []uint256.Int{
			{0x00, 0x00, 0x00, 0x00},
			{0x00, 0x00, 0x00, 0x00}},
		res:    uint256.Int{0x00, 0x00, 0x00, 0x00},
		status: RUNNING,
	},
	{
		name: "opAnd: 0x5..5 & 0xA..A (x & ^x)",
		op:   opAnd,
		data: []uint256.Int{
			{0xAAAAAAAAAAAAAAAA, 0xAAAAAAAAAAAAAAAA, 0xAAAAAAAAAAAAAAAA, 0xAAAAAAAAAAAAAAAA},
			{0x5555555555555555, 0x5555555555555555, 0x5555555555555555, 0x5555555555555555}},
		res:    uint256.Int{0x00, 0x00, 0x00, 0x00},
		status: RUNNING,
	},
	{
		name: "opAnd: 0xA..A & 0x5..5 (x & ^x)",
		op:   opAnd,
		data: []uint256.Int{
			{0x5555555555555555, 0x5555555555555555, 0x5555555555555555, 0x5555555555555555},
			{0xAAAAAAAAAAAAAAAA, 0xAAAAAAAAAAAAAAAA, 0xAAAAAAAAAAAAAAAA, 0xAAAAAAAAAAAAAAAA}},
		res:    uint256.Int{0x00, 0x00, 0x00, 0x00},
		status: RUNNING,
	},
	{
		name: "opAnd: various bytes 0xFF and 0x00",
		op:   opAnd,
		data: []uint256.Int{
			{0xFFFF0000FFFF0000, 0xFFFFFFFF00000000, 0x00000000FFFFFFFF, 0x0000FFFF0000FFFF},
			{0x00FF00FF00FF00FF, 0xFF00FF00FF00FF00, 0x00FF00FF00FF00FF, 0xFF00FF00FF00FF00}},
		res:    uint256.Int{0x00FF000000FF0000, 0xFF00FF0000000000, 0x0000000000FF00FF, 0x0000FF000000FF00},
		status: RUNNING,
	},
	{
		name: "opAnd: various 8*bytes",
		op:   opAnd,
		data: []uint256.Int{
			{0xFFFFFFFFFFFFFFFF, 0x00, 0xFFFFFFFFFFFFFFFF, 0x00},
			{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0x00, 0x00}},
		res:    uint256.Int{0xFFFFFFFFFFFFFFFF, 0x00, 0x00, 0x00},
		status: RUNNING,
	},
	{
		name: "opAnd: first and last bit",
		op:   opAnd,
		data: []uint256.Int{
			{0x01, 0x00, 0x00, 0x8000000000000000},
			{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF}},
		res:    uint256.Int{0x01, 0x00, 0x00, 0x8000000000000000},
		status: RUNNING,
	},
	{
		name: "opAnd: last byte",
		op:   opAnd,
		data: []uint256.Int{
			{0x1234, 0x00, 0x00, 0x00},
			{0x00FF, 0x00, 0x00, 0x00}},
		res:    uint256.Int{0x0034, 0x00, 0x00, 0x00},
		status: RUNNING,
	},
	{
		name: "opAnd: bytes 14 and 15",
		op:   opAnd,
		data: []uint256.Int{
			{0x00, 0x00, 0x1234, 0x00},
			{0x00, 0x00, 0x1234, 0xFFFF}},
		res:    uint256.Int{0x00, 0x00, 0x1234, 0x00},
		status: RUNNING,
	},
	{
		name: "opAnd: last 2 bytes of each 8",
		op:   opAnd,
		data: []uint256.Int{
			{0xFF00, 0x0FF0, 0x00FF, 0xF0F0},
			{0x1234, 0xABCD, 0x5678, 0xEF90}},
		res:    uint256.Int{0x1200, 0x0BC0, 0x0078, 0xE090},
		status: RUNNING,
	},

	// operation Or
	{
		name: "opOr: 0xF..F | various bits",
		op:   opOr,
		data: []uint256.Int{
			{0x123456789ABCDEF0, 0x23456789ABCDEF01, 0x3456789ABCDEF012, 0x456789ABCDEF0123},
			{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF}},
		res:    uint256.Int{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF},
		status: RUNNING,
	},
	{
		name: "opOr: various bits | 0xF..F",
		op:   opOr,
		data: []uint256.Int{
			{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF},
			{0x123456789ABCDEF0, 0x23456789ABCDEF01, 0x3456789ABCDEF012, 0x456789ABCDEF0123}},
		res:    uint256.Int{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF},
		status: RUNNING,
	},
	{
		name: "opOr: 0xF..F | 0x0..0",
		op:   opOr,
		data: []uint256.Int{
			{0x00, 0x00, 0x00, 0x00},
			{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF}},
		res:    uint256.Int{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF},
		status: RUNNING,
	},
	{
		name: "opOr: 0x0..0 | 0xF..F",
		op:   opOr,
		data: []uint256.Int{
			{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF},
			{0x00, 0x00, 0x00, 0x00}},
		res:    uint256.Int{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF},
		status: RUNNING,
	},
	{
		name: "opOr: 0xF..F | 0xF..F",
		op:   opOr,
		data: []uint256.Int{
			{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF},
			{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF}},
		res:    uint256.Int{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF},
		status: RUNNING,
	},
	{
		name: "opOr: 0x0..0 | 0x0..0",
		op:   opOr,
		data: []uint256.Int{
			{0x00, 0x00, 0x00, 0x00},
			{0x00, 0x00, 0x00, 0x00}},
		res:    uint256.Int{0x00, 0x00, 0x00, 0x00},
		status: RUNNING,
	},
	{
		name: "opOr: 0x5..5 | 0xA..A (x | ^x)",
		op:   opOr,
		data: []uint256.Int{
			{0xAAAAAAAAAAAAAAAA, 0xAAAAAAAAAAAAAAAA, 0xAAAAAAAAAAAAAAAA, 0xAAAAAAAAAAAAAAAA},
			{0x5555555555555555, 0x5555555555555555, 0x5555555555555555, 0x5555555555555555}},
		res:    uint256.Int{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF},
		status: RUNNING,
	},
	{
		name: "opOr: 0xA..A | 0x5..5 (x | ^x)",
		op:   opOr,
		data: []uint256.Int{
			{0x5555555555555555, 0x5555555555555555, 0x5555555555555555, 0x5555555555555555},
			{0xAAAAAAAAAAAAAAAA, 0xAAAAAAAAAAAAAAAA, 0xAAAAAAAAAAAAAAAA, 0xAAAAAAAAAAAAAAAA}},
		res:    uint256.Int{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF},
		status: RUNNING,
	},
	{
		name: "opOr: various bytes 0xFF and 0x00",
		op:   opOr,
		data: []uint256.Int{
			{0xFFFF0000FFFF0000, 0xFFFFFFFF00000000, 0x00000000FFFFFFFF, 0x0000FFFF0000FFFF},
			{0x00FF00FF00FF00FF, 0xFF00FF00FF00FF00, 0x00FF00FF00FF00FF, 0xFF00FF00FF00FF00}},
		res:    uint256.Int{0xFFFF00FFFFFF00FF, 0xFFFFFFFFFF00FF00, 0x00FF00FFFFFFFFFF, 0xFF00FFFFFF00FFFF},
		status: RUNNING,
	},
	{
		name: "opOr: various 8*bytes",
		op:   opOr,
		data: []uint256.Int{
			{0xFFFFFFFFFFFFFFFF, 0x00, 0xFFFFFFFFFFFFFFFF, 0x00},
			{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0x00, 0x00}},
		res:    uint256.Int{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0x00},
		status: RUNNING,
	},
	{
		name: "opOr: first and last bit",
		op:   opOr,
		data: []uint256.Int{
			{0x01, 0x00, 0x00, 0x8000000000000000},
			{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF}},
		res:    uint256.Int{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF},
		status: RUNNING,
	},
	{
		name: "opOr: last 2 bytes",
		op:   opOr,
		data: []uint256.Int{
			{0x1234, 0x00, 0x00, 0x00},
			{0x00FF, 0x00, 0x00, 0x00}},
		res:    uint256.Int{0x12FF, 0x00, 0x00, 0x00},
		status: RUNNING,
	},
	{
		name: "opOr: bytes 14 and 15",
		op:   opOr,
		data: []uint256.Int{
			{0x00, 0x00, 0x1234, 0x00},
			{0x00, 0x00, 0x1234, 0xFFFF}},
		res:    uint256.Int{0x00, 0x00, 0x1234, 0xFFFF},
		status: RUNNING,
	},
	{
		name: "opOr: last 2 bytes of each 8",
		op:   opOr,
		data: []uint256.Int{
			{0xFF00, 0x0FF0, 0x00FF, 0xF0F0},
			{0x1234, 0xABCD, 0x5678, 0xEF90}},
		res:    uint256.Int{0xFF34, 0xAFFD, 0x56FF, 0xFFF0},
		status: RUNNING,
	},

	// operation Not
	{
		name:   "opNot: ^0x0..0",
		op:     opNot,
		data:   []uint256.Int{{0x00, 0x00, 0x00, 0x00}},
		res:    uint256.Int{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF},
		status: RUNNING,
	},
	{
		name:   "opNot: ^0xF..F",
		op:     opNot,
		data:   []uint256.Int{{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF}},
		res:    uint256.Int{0x00, 0x00, 0x00, 0x00},
		status: RUNNING,
	},
	{
		name:   "opNot: various bits",
		op:     opNot,
		data:   []uint256.Int{{0x123456789ABCDEF0, 0x23456789ABCDEF01, 0x3456789ABCDEF012, 0x456789ABCDEF0123}},
		res:    uint256.Int{0xEDCBA9876543210F, 0xDCBA9876543210FE, 0xCBA9876543210FED, 0xBA9876543210FEDC},
		status: RUNNING,
	},
	{
		name:   "opNot: ^0xA..A",
		op:     opNot,
		data:   []uint256.Int{{0xAAAAAAAAAAAAAAAA, 0xAAAAAAAAAAAAAAAA, 0xAAAAAAAAAAAAAAAA, 0xAAAAAAAAAAAAAAAA}},
		res:    uint256.Int{0x5555555555555555, 0x5555555555555555, 0x5555555555555555, 0x5555555555555555},
		status: RUNNING,
	},
	{
		name:   "opNot: ^0x5..5",
		op:     opNot,
		data:   []uint256.Int{{0x5555555555555555, 0x5555555555555555, 0x5555555555555555, 0x5555555555555555}},
		res:    uint256.Int{0xAAAAAAAAAAAAAAAA, 0xAAAAAAAAAAAAAAAA, 0xAAAAAAAAAAAAAAAA, 0xAAAAAAAAAAAAAAAA},
		status: RUNNING,
	},
	{
		name:   "opNot: various 2*bytes 0xFFFF and 0x0000",
		op:     opNot,
		data:   []uint256.Int{{0xFFFF0000FFFF0000, 0xFFFFFFFF00000000, 0x00000000FFFFFFFF, 0x0000FFFF0000FFFF}},
		res:    uint256.Int{0x0000FFFF0000FFFF, 0x00000000FFFFFFFF, 0xFFFFFFFF00000000, 0xFFFF0000FFFF0000},
		status: RUNNING,
	},
	{
		name:   "opNot: various bytes 0xFF and 0x00",
		op:     opNot,
		data:   []uint256.Int{{0x00FF00FF00FF00FF, 0xFF00FF00FF00FF00, 0x00FF00FF00FF00FF, 0xFF00FF00FF00FF00}},
		res:    uint256.Int{0xFF00FF00FF00FF00, 0x00FF00FF00FF00FF, 0xFF00FF00FF00FF00, 0x00FF00FF00FF00FF},
		status: RUNNING,
	},
	{
		name:   "opNot: first and last bit",
		op:     opNot,
		data:   []uint256.Int{{0x01, 0x00, 0x00, 0x8000000000000000}},
		res:    uint256.Int{0xFFFFFFFFFFFFFFFE, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0x7FFFFFFFFFFFFFFF},
		status: RUNNING,
	},
	{
		name:   "opNot: bytes 22 and 23",
		op:     opNot,
		data:   []uint256.Int{{0x00, 0x1234, 0x00, 0x00}},
		res:    uint256.Int{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFEDCB, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF},
		status: RUNNING,
	},
	{
		name:   "opNot: bytes 14 and 15",
		op:     opNot,
		data:   []uint256.Int{{0x00, 0x00, 0xFFFF, 0x00}},
		res:    uint256.Int{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFF0000, 0xFFFFFFFFFFFFFFFF},
		status: RUNNING,
	},

	// operation Xor
	{
		name: "opXor: 0xF..F xor various bits",
		op:   opXor,
		data: []uint256.Int{
			{0x123456789ABCDEF0, 0x23456789ABCDEF01, 0x3456789ABCDEF012, 0x456789ABCDEF0123},
			{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF}},
		res:    uint256.Int{0xEDCBA9876543210F, 0xDCBA9876543210FE, 0xCBA9876543210FED, 0xBA9876543210FEDC},
		status: RUNNING,
	},
	{
		name: "opXor: various bits xor 0xF..F",
		op:   opXor,
		data: []uint256.Int{
			{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF},
			{0x123456789ABCDEF0, 0x23456789ABCDEF01, 0x3456789ABCDEF012, 0x456789ABCDEF0123}},
		res:    uint256.Int{0xEDCBA9876543210F, 0xDCBA9876543210FE, 0xCBA9876543210FED, 0xBA9876543210FEDC},
		status: RUNNING,
	},
	{
		name: "opXor: 0xF..F xor 0x0..0",
		op:   opXor,
		data: []uint256.Int{
			{0x00, 0x00, 0x00, 0x00},
			{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF}},
		res:    uint256.Int{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF},
		status: RUNNING,
	},
	{
		name: "opXor: 0x0..0 xor 0xF..F",
		op:   opXor,
		data: []uint256.Int{
			{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF},
			{0x00, 0x00, 0x00, 0x00}},
		res:    uint256.Int{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF},
		status: RUNNING,
	},
	{
		name: "opXor: 0xF..F xor 0xF..F",
		op:   opXor,
		data: []uint256.Int{
			{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF},
			{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF}},
		res:    uint256.Int{0x00, 0x00, 0x00, 0x00},
		status: RUNNING,
	},
	{
		name: "opXor: 0x0..0 xor 0x0..0",
		op:   opXor,
		data: []uint256.Int{
			{0x00, 0x00, 0x00, 0x00},
			{0x00, 0x00, 0x00, 0x00}},
		res:    uint256.Int{0x00, 0x00, 0x00, 0x00},
		status: RUNNING,
	},
	{
		name: "opXor: 0x5..5 xor 0xA..A (x xor ^x)",
		op:   opXor,
		data: []uint256.Int{
			{0xAAAAAAAAAAAAAAAA, 0xAAAAAAAAAAAAAAAA, 0xAAAAAAAAAAAAAAAA, 0xAAAAAAAAAAAAAAAA},
			{0x5555555555555555, 0x5555555555555555, 0x5555555555555555, 0x5555555555555555}},
		res:    uint256.Int{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF},
		status: RUNNING,
	},
	{
		name: "opXor: 0xA..A xor 0x5..5 (x xor ^x)",
		op:   opXor,
		data: []uint256.Int{
			{0x5555555555555555, 0x5555555555555555, 0x5555555555555555, 0x5555555555555555},
			{0xAAAAAAAAAAAAAAAA, 0xAAAAAAAAAAAAAAAA, 0xAAAAAAAAAAAAAAAA, 0xAAAAAAAAAAAAAAAA}},
		res:    uint256.Int{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF},
		status: RUNNING,
	},
	{
		name: "opXor: various bytes 0xFF and 0x00",
		op:   opXor,
		data: []uint256.Int{
			{0xFFFF0000FFFF0000, 0xFFFFFFFF00000000, 0x00000000FFFFFFFF, 0x0000FFFF0000FFFF},
			{0x00FF00FF00FF00FF, 0xFF00FF00FF00FF00, 0x00FF00FF00FF00FF, 0xFF00FF00FF00FF00}},
		res:    uint256.Int{0xFF0000FFFF0000FF, 0x00FF00FFFF00FF00, 0x00FF00FFFF00FF00, 0xFF0000FFFF0000FF},
		status: RUNNING,
	},
	{
		name: "opXor: various 8*bytes",
		op:   opXor,
		data: []uint256.Int{
			{0xFFFFFFFFFFFFFFFF, 0x00, 0xFFFFFFFFFFFFFFFF, 0x00},
			{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0x00, 0x00}},
		res:    uint256.Int{0x00, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0x00},
		status: RUNNING,
	},
	{
		name: "opXor: first and last bit",
		op:   opXor,
		data: []uint256.Int{
			{0x01, 0x00, 0x00, 0x8000000000000000},
			{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF}},
		res:    uint256.Int{0xFFFFFFFFFFFFFFFE, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0x7FFFFFFFFFFFFFFF},
		status: RUNNING,
	},
	{
		name: "opXor: last 2 bytes",
		op:   opXor,
		data: []uint256.Int{
			{0x1234, 0x00, 0x00, 0x00},
			{0x00FF, 0x00, 0x00, 0x00}},
		res:    uint256.Int{0x12CB, 0x00, 0x00, 0x00},
		status: RUNNING,
	},
	{
		name: "opXor: bytes 6, 7, 14 and 15",
		op:   opXor,
		data: []uint256.Int{
			{0x00, 0x00, 0x1234, 0x00},
			{0x00, 0x00, 0x1234, 0xFFFF}},
		res:    uint256.Int{0x00, 0x00, 0x00, 0xFFFF},
		status: RUNNING,
	},
	{
		name: "opXor: last 2 bytes of each 8",
		op:   opXor,
		data: []uint256.Int{
			{0xFF00, 0x0FF0, 0x00FF, 0xF0F0},
			{0x1234, 0xABCD, 0x5678, 0xEF90}},
		res:    uint256.Int{0xED34, 0xA43D, 0x5687, 0x1F60},
		status: RUNNING,
	},

	// operation Byte
	// byte(i, x), data: {x, i}
	{
		name: "opByte: byte 0 of number 0",
		op:   opByte,
		data: []uint256.Int{
			{0x00, 0x00, 0x00, 0x00},
			{0x00, 0x00, 0x00, 0x00}},
		res:    uint256.Int{0x00, 0x00, 0x00, 0x00},
		status: RUNNING,
	},
	{
		name: "opByte: byte 0 of 0xFF0..0",
		op:   opByte,
		data: []uint256.Int{
			{0x00, 0x00, 0x00, 0xFF00000000000000},
			{0x00, 0x00, 0x00, 0x00}},
		res:    uint256.Int{0xFF, 0x00, 0x00, 0x00},
		status: RUNNING,
	},
	{
		name: "opByte: byte 0 of 0x00AB0..0",
		op:   opByte,
		data: []uint256.Int{
			{0x00, 0x00, 0x00, 0x00AB000000000000},
			{0x00, 0x00, 0x00, 0x00}},
		res:    uint256.Int{0x00, 0x00, 0x00, 0x00},
		status: RUNNING,
	},
	{
		name: "opByte: byte 1",
		op:   opByte,
		data: []uint256.Int{
			{0x00, 0x00, 0x00, 0x123456789ABCDEF0},
			{1, 0, 0, 0}},
		res:    uint256.Int{0x34, 0x00, 0x00, 0x00},
		status: RUNNING,
	},
	{
		name: "opByte: byte 0 of 0x63F..F",
		op:   opByte,
		data: []uint256.Int{
			{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0x63FFFFFFFFFFFFFF},
			{0, 0, 0, 0}},
		res:    uint256.Int{0x63, 0x00, 0x00, 0x00},
		status: RUNNING,
	},
	{
		name: "opByte: byte 1 of 0xFF84F..F",
		op:   opByte,
		data: []uint256.Int{
			{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFF84FFFFFFFFFFFF},
			{1, 0, 0, 0}},
		res:    uint256.Int{0x84, 0x00, 0x00, 0x00},
		status: RUNNING,
	},
	{
		name: "opByte: byte 7",
		op:   opByte,
		data: []uint256.Int{
			{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFC5},
			{7, 0, 0, 0}},
		res:    uint256.Int{0xC5, 0x00, 0x00, 0x00},
		status: RUNNING,
	},
	{
		name: "opByte: byte 8",
		op:   opByte,
		data: []uint256.Int{
			{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xB4FFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF},
			{8, 0, 0, 0}},
		res:    uint256.Int{0xB4, 0x00, 0x00, 0x00},
		status: RUNNING,
	},
	{
		name: "opByte: byte 15",
		op:   opByte,
		data: []uint256.Int{
			{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFF32, 0xFFFFFFFFFFFFFFFF},
			{15, 0, 0, 0}},
		res:    uint256.Int{0x32, 0x00, 0x00, 0x00},
		status: RUNNING,
	},
	{
		name: "opByte: byte 16",
		op:   opByte,
		data: []uint256.Int{
			{0xFFFFFFFFFFFFFFFF, 0x74FFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF},
			{16, 0, 0, 0}},
		res:    uint256.Int{0x74, 0x00, 0x00, 0x00},
		status: RUNNING,
	},
	{
		name: "opByte: byte 23",
		op:   opByte,
		data: []uint256.Int{
			{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFF68, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF},
			{23, 0, 0, 0}},
		res:    uint256.Int{0x68, 0x00, 0x00, 0x00},
		status: RUNNING,
	},
	{
		name: "opByte: byte 24",
		op:   opByte,
		data: []uint256.Int{
			{0x91FFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF},
			{24, 0, 0, 0}},
		res:    uint256.Int{0x91, 0x00, 0x00, 0x00},
		status: RUNNING,
	},
	{
		name: "opByte: byte 31",
		op:   opByte,
		data: []uint256.Int{
			{0xFFFFFFFFFFFFFF42, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF},
			{31, 0, 0, 0}},
		res:    uint256.Int{0x42, 0x00, 0x00, 0x00},
		status: RUNNING,
	},
	{
		name: "opByte: byte 32 (out of range)",
		op:   opByte,
		data: []uint256.Int{
			{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF},
			{32, 0, 0, 0}},
		res:    uint256.Int{0x00, 0x00, 0x00, 0x00},
		status: RUNNING,
	},
	{
		name: "opByte: byte 19",
		op:   opByte,
		data: []uint256.Int{
			{0x00, 0x0000002500000000, 0x00, 0x00},
			{19, 0, 0, 0}},
		res:    uint256.Int{0x25, 0x00, 0x00, 0x00},
		status: RUNNING,
	},

	// operation Shl
	// val << shift, data: {val, shift}
	{
		name: "opShl: left shift 0 on number 0",
		op:   opShl,
		data: []uint256.Int{
			{0x00, 0x00, 0x00, 0x00},
			{0x00, 0x00, 0x00, 0x00}},
		res:    uint256.Int{0x00, 0x00, 0x00, 0x00},
		status: RUNNING,
	},
	{
		name: "opShl: left shift 0 on number 0xFF0..0",
		op:   opShl,
		data: []uint256.Int{
			{0x00, 0x00, 0x00, 0xFF00000000000000},
			{0x00, 0x00, 0x00, 0x00}},
		res:    uint256.Int{0x00, 0x00, 0x00, 0xFF00000000000000},
		status: RUNNING,
	},
	{
		name: "opShl: left shift 0 on number 0x00AB0..0",
		op:   opShl,
		data: []uint256.Int{
			{0x00, 0x00, 0x00, 0x00AB000000000000},
			{0x00, 0x00, 0x00, 0x00}},
		res:    uint256.Int{0x00, 0x00, 0x00, 0x00AB000000000000},
		status: RUNNING,
	},
	{
		name: "opShl: left shift 1",
		op:   opShl,
		data: []uint256.Int{
			{0x00, 0x00, 0x00, 0x123456789ABCDEF0},
			{1, 0, 0, 0}},
		res:    uint256.Int{0x00, 0x00, 0x00, 0x2468ACF13579BDE0},
		status: RUNNING,
	},
	{
		name: "opShl: left shift 0 on number 0x63F..F",
		op:   opShl,
		data: []uint256.Int{
			{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0x63FFFFFFFFFFFFFF},
			{0, 0, 0, 0}},
		res:    uint256.Int{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0x63FFFFFFFFFFFFFF},
		status: RUNNING,
	},
	{
		name: "opShl: left shift 1 on number 0xFF84F..F",
		op:   opShl,
		data: []uint256.Int{
			{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFF84FFFFFFFFFFFF},
			{1, 0, 0, 0}},
		res:    uint256.Int{0xFFFFFFFFFFFFFFFE, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFF09FFFFFFFFFFFF},
		status: RUNNING,
	},
	{
		name: "opShl: left shift 8",
		op:   opShl,
		data: []uint256.Int{
			{0x1234000000000000, 0x5678000000000000, 0x9ABC000000000000, 0xDEF0000000000000},
			{8, 0, 0, 0}},
		res:    uint256.Int{0x3400000000000000, 0x7800000000000012, 0xBC00000000000056, 0xF00000000000009A},
		status: RUNNING,
	},
	{
		name: "opShl: left shift 64",
		op:   opShl,
		data: []uint256.Int{
			{0x1234000000000000, 0x5678000000000000, 0x9ABC000000000000, 0xDEF0000000000000},
			{64, 0, 0, 0}},
		res:    uint256.Int{0x00, 0x1234000000000000, 0x5678000000000000, 0x9ABC000000000000},
		status: RUNNING,
	},
	{
		name: "opShl: left shift 256",
		op:   opShl,
		data: []uint256.Int{
			{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFF32, 0xFFFFFFFFFFFFFFFF},
			{256, 0, 0, 0}},
		res:    uint256.Int{0x00, 0x00, 0x00, 0x00},
		status: RUNNING,
	},

	// operation Shr
	// val >> shift, data: {val, shift}
	{
		name: "opShr: right shift 0 on number 0",
		op:   opShr,
		data: []uint256.Int{
			{0x00, 0x00, 0x00, 0x00},
			{0x00, 0x00, 0x00, 0x00}},
		res:    uint256.Int{0x00, 0x00, 0x00, 0x00},
		status: RUNNING,
	},
	{
		name: "opShr: right shift 0 on number 0xFF0..0",
		op:   opShr,
		data: []uint256.Int{
			{0x00, 0x00, 0x00, 0xFF00000000000000},
			{0x00, 0x00, 0x00, 0x00}},
		res:    uint256.Int{0x00, 0x00, 0x00, 0xFF00000000000000},
		status: RUNNING,
	},
	{
		name: "opShr: right shift 0 on number 0x00AB0..0",
		op:   opShr,
		data: []uint256.Int{
			{0x00, 0x00, 0x00, 0x00AB000000000000},
			{0x00, 0x00, 0x00, 0x00}},
		res:    uint256.Int{0x00, 0x00, 0x00, 0x00AB000000000000},
		status: RUNNING,
	},
	{
		name: "opShr: right shift 1",
		op:   opShr,
		data: []uint256.Int{
			{0x00, 0x00, 0x00, 0x123456789ABCDEF0},
			{1, 0, 0, 0}},
		res:    uint256.Int{0x00, 0x00, 0x00, 0x091A2B3C4D5E6F78},
		status: RUNNING,
	},
	{
		name: "opShr: right shift 0 on number 0x63F..F",
		op:   opShr,
		data: []uint256.Int{
			{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0x63FFFFFFFFFFFFFF},
			{0x00, 0x00, 0x00, 0x00}},
		res:    uint256.Int{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0x63FFFFFFFFFFFFFF},
		status: RUNNING,
	},
	{
		name: "opShr: right shift 1 on number 0xFF84F..F",
		op:   opShr,
		data: []uint256.Int{
			{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFF84FFFFFFFFFFFF},
			{1, 0, 0, 0}},
		res:    uint256.Int{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0x7FC27FFFFFFFFFFF},
		status: RUNNING,
	},
	{
		name: "opShr: right shift 8",
		op:   opShr,
		data: []uint256.Int{
			{0x3400000000000000, 0x7800000000000012, 0xBC00000000000056, 0xF00000000000009A},
			{8, 0, 0, 0}},
		res:    uint256.Int{0x1234000000000000, 0x5678000000000000, 0x9ABC000000000000, 0x00F0000000000000},
		status: RUNNING,
	},
	{
		name: "opShr: right shift 64",
		op:   opShr,
		data: []uint256.Int{
			{0x1234000000000000, 0x5678000000000000, 0x9ABC000000000000, 0xDEF0000000000000},
			{64, 0, 0, 0}},
		res:    uint256.Int{0x5678000000000000, 0x9ABC000000000000, 0xDEF0000000000000, 0x00},
		status: RUNNING,
	},
	{
		name: "opShr: right shift 256",
		op:   opShr,
		data: []uint256.Int{
			{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFF32, 0xFFFFFFFFFFFFFFFF},
			{256, 0, 0, 0}},
		res:    uint256.Int{0x00, 0x00, 0x00, 0x00},
		status: RUNNING,
	},

	// operation Sar
	// val >> shift (arithemtic), data: {val, shift}
	{
		name: "opSar: right shift 0 on number 0",
		op:   opSar,
		data: []uint256.Int{
			{0x00, 0x00, 0x00, 0x00},
			{0x00, 0x00, 0x00, 0x00}},
		res:    uint256.Int{0x00, 0x00, 0x00, 0x00},
		status: RUNNING,
	},
	{
		name: "opSar: right shift 0 on number 0xFF0..0",
		op:   opSar,
		data: []uint256.Int{
			{0x00, 0x00, 0x00, 0xFF00000000000000},
			{0x00, 0x00, 0x00, 0x00}},
		res:    uint256.Int{0x00, 0x00, 0x00, 0xFF00000000000000},
		status: RUNNING,
	},
	{
		name: "opSar: right shift 0 on number 0x00AB0..0",
		op:   opSar,
		data: []uint256.Int{
			{0x00, 0x00, 0x00, 0x00AB000000000000},
			{0x00, 0x00, 0x00, 0x00}},
		res:    uint256.Int{0x00, 0x00, 0x00, 0x00AB000000000000},
		status: RUNNING,
	},
	{
		name: "opSar: right shift 1",
		op:   opSar,
		data: []uint256.Int{
			{0x00, 0x00, 0x00, 0x123456789ABCDEF0},
			{1, 0, 0, 0}},
		res:    uint256.Int{0x00, 0x00, 0x00, 0x091A2B3C4D5E6F78},
		status: RUNNING,
	},
	{
		name: "opSar: right shift 0 on number 0x63F..F",
		op:   opSar,
		data: []uint256.Int{
			{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0x63FFFFFFFFFFFFFF},
			{0, 0, 0, 0}},
		res:    uint256.Int{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0x63FFFFFFFFFFFFFF},
		status: RUNNING,
	},
	{
		name: "opSar: right shift 1 on number 0xFF84F..F",
		op:   opSar,
		data: []uint256.Int{
			{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFF84FFFFFFFFFFFF},
			{1, 0, 0, 0}},
		res:    uint256.Int{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFC27FFFFFFFFFFF},
		status: RUNNING,
	},
	{
		name: "opSar: right shift 8",
		op:   opSar,
		data: []uint256.Int{
			{0x3400000000000000, 0x7800000000000012, 0xBC00000000000056, 0xF00000000000009A},
			{8, 0, 0, 0}},
		res:    uint256.Int{0x1234000000000000, 0x5678000000000000, 0x9ABC000000000000, 0xFFF0000000000000},
		status: RUNNING,
	},
	{
		name: "opSar: right shift 64",
		op:   opSar,
		data: []uint256.Int{
			{0x1234000000000000, 0x5678000000000000, 0x9ABC000000000000, 0xDEF0000000000000},
			{64, 0, 0, 0}},
		res:    uint256.Int{0x5678000000000000, 0x9ABC000000000000, 0xDEF0000000000000, 0xFFFFFFFFFFFFFFFF},
		status: RUNNING,
	},
	{
		name: "opSar: right shift 256",
		op:   opSar,
		data: []uint256.Int{
			{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFF32, 0xFFFFFFFFFFFFFFFF},
			{256, 0, 0, 0}},
		res:    uint256.Int{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF},
		status: RUNNING,
	},
}

// arithmetic operations (Add, Sub, Mul, MulMod, Div, SDiv, Mod, AddMod, SMod, Exp, SignExtend)
var testDataArithmeticOp = []tTestDataOp{

	// operation Add
	{
		name: "opAdd: 0 + 0",
		op:   opAdd,
		data: []uint256.Int{
			{0, 0, 0, 0},
			{0, 0, 0, 0}},
		res:    uint256.Int{0, 0, 0, 0},
		status: RUNNING,
	},
	{
		name: "opAdd: -1 + 0",
		op:   opAdd,
		data: []uint256.Int{
			{0x00, 0x00, 0x00, 0x00},
			{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF}},
		res:    uint256.Int{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF},
		status: RUNNING,
	},
	{
		name: "opAdd: 0 + -1",
		op:   opAdd,
		data: []uint256.Int{
			{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF},
			{0x00, 0x00, 0x00, 0x00}},
		res:    uint256.Int{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF},
		status: RUNNING,
	},
	{
		name: "opAdd: 1 + -1",
		op:   opAdd,
		data: []uint256.Int{
			{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF},
			{0x01, 0x00, 0x00, 0x00}},
		res:    uint256.Int{0x00, 0x00, 0x00, 0x00},
		status: RUNNING,
	},
	{
		name: "opAdd: -1 + 1",
		op:   opAdd,
		data: []uint256.Int{
			{0x01, 0x00, 0x00, 0x00},
			{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF}},
		res:    uint256.Int{0x00, 0x00, 0x00, 0x00},
		status: RUNNING,
	},
	{
		name: "opAdd: -1 + -1",
		op:   opAdd,
		data: []uint256.Int{
			{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF},
			{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF}},
		res:    uint256.Int{0xFFFFFFFFFFFFFFFE, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF},
		status: RUNNING,
	},
	{
		name: "opAdd: -1 + x",
		op:   opAdd,
		data: []uint256.Int{
			{0x123456789ABCDEF0, 0x23456789ABCDEF01, 0x3456789ABCDEF012, 0x456789ABCDEF0123},
			{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF}},
		res:    uint256.Int{0x123456789ABCDEEF, 0x23456789ABCDEF01, 0x3456789ABCDEF012, 0x456789ABCDEF0123},
		status: RUNNING,
	},
	{
		name: "opAdd: x + -1",
		op:   opAdd,
		data: []uint256.Int{
			{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF},
			{0x123456789ABCDEF0, 0x23456789ABCDEF01, 0x3456789ABCDEF012, 0x456789ABCDEF0123}},
		res:    uint256.Int{0x123456789ABCDEEF, 0x23456789ABCDEF01, 0x3456789ABCDEF012, 0x456789ABCDEF0123},
		status: RUNNING,
	},
	{
		name: "opAdd: 0x5..5 + 0xA..A (x + ^x)",
		op:   opAdd,
		data: []uint256.Int{
			{0xAAAAAAAAAAAAAAAA, 0xAAAAAAAAAAAAAAAA, 0xAAAAAAAAAAAAAAAA, 0xAAAAAAAAAAAAAAAA},
			{0x5555555555555555, 0x5555555555555555, 0x5555555555555555, 0x5555555555555555}},
		res:    uint256.Int{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF},
		status: RUNNING,
	},
	{
		name: "opAdd: 0xA..A + 0x5..5 (x + ^x)",
		op:   opAdd,
		data: []uint256.Int{
			{0x5555555555555555, 0x5555555555555555, 0x5555555555555555, 0x5555555555555555},
			{0xAAAAAAAAAAAAAAAA, 0xAAAAAAAAAAAAAAAA, 0xAAAAAAAAAAAAAAAA, 0xAAAAAAAAAAAAAAAA}},
		res:    uint256.Int{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF},
		status: RUNNING,
	},
	{
		name: "opAdd: overflow to the highest 8 bytes",
		op:   opAdd,
		data: []uint256.Int{
			{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0x00},
			{0x01, 0x00, 0x00, 0x00}},
		res:    uint256.Int{0x00, 0x00, 0x00, 0x01},
		status: RUNNING,
	},
	{
		name: "opAdd: overflow over 64bit",
		op:   opAdd,
		data: []uint256.Int{
			{0xFFFFFFFFFFFFFFFF, 0x00, 0xFFFFFFFFFFFFFFFF, 0x00},
			{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0x00, 0x00}},
		res:    uint256.Int{0xFFFFFFFFFFFFFFFE, 0x00, 0x00, 0x01},
		status: RUNNING,
	},
	{
		name: "opAdd: first and last bit",
		op:   opAdd,
		data: []uint256.Int{
			{0x01, 0x00, 0x00, 0x8000000000000000},
			{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF}},
		res:    uint256.Int{0x00, 0x00, 0x00, 0x8000000000000000},
		status: RUNNING,
	},
	{
		name: "opAdd: last 2 bytes",
		op:   opAdd,
		data: []uint256.Int{
			{0x1234, 0x00, 0x00, 0x00},
			{0x00FF, 0x00, 0x00, 0x00}},
		res:    uint256.Int{0x1333, 0x00, 0x00, 0x00},
		status: RUNNING,
	},
	{
		name: "opAdd: sum in the upper 16 bytes",
		op:   opAdd,
		data: []uint256.Int{
			{0x00, 0x00, 0x1234, 0x00},
			{0x00, 0x00, 0x1234, 0xFFFF}},
		res:    uint256.Int{0x00, 0x00, 0x2468, 0xFFFF},
		status: RUNNING,
	},
	{
		name: "opAdd: last 2 bytes of each 8",
		op:   opAdd,
		data: []uint256.Int{
			{0xFF00, 0x0FF0, 0x00FF, 0xF0F0},
			{0x1234, 0xABCD, 0x5678, 0xEF90}},
		res:    uint256.Int{0x011134, 0xBBBD, 0x5777, 0x01E080},
		status: RUNNING,
	},
	{
		name: "opAdd: -1 + 0x1234",
		op:   opAdd,
		data: []uint256.Int{
			{0x1234, 0x00, 0x00, 0x00},
			{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF}},
		res:    uint256.Int{0x1233, 0x00, 0x00, 0x00},
		status: RUNNING,
	},
	{
		name: "opAdd: -10 + 0x1234",
		op:   opAdd,
		data: []uint256.Int{
			{0x1234, 0x00, 0x00, 0x00},
			{0xFFFFFFFFFFFFFFF6, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF}},
		res:    uint256.Int{0x122A, 0x00, 0x00, 0x00},
		status: RUNNING,
	},

	// operation Sub
	{
		name: "opSub: 0 - 0",
		op:   opSub,
		data: []uint256.Int{
			{0, 0, 0, 0},
			{0, 0, 0, 0}},
		res:    uint256.Int{0, 0, 0, 0},
		status: RUNNING,
	},
	{
		name: "opSub: -1 - 0",
		op:   opSub,
		data: []uint256.Int{
			{0x00, 0x00, 0x00, 0x00},
			{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF}},
		res:    uint256.Int{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF},
		status: RUNNING,
	},
	{
		name: "opSub: 0 - -1",
		op:   opSub,
		data: []uint256.Int{
			{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF},
			{0x00, 0x00, 0x00, 0x00}},
		res:    uint256.Int{0x01, 0x00, 0x00, 0x00},
		status: RUNNING,
	},
	{
		name: "opSub: 1 - -1",
		op:   opSub,
		data: []uint256.Int{
			{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF},
			{0x01, 0x00, 0x00, 0x00}},
		res:    uint256.Int{0x02, 0x00, 0x00, 0x00},
		status: RUNNING,
	},
	{
		name: "opSub: -1 - 1",
		op:   opSub,
		data: []uint256.Int{
			{0x01, 0x00, 0x00, 0x00},
			{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF}},
		res:    uint256.Int{0xFFFFFFFFFFFFFFFE, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF},
		status: RUNNING,
	},
	{
		name: "opSub: -1 - -1",
		op:   opSub,
		data: []uint256.Int{
			{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF},
			{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF}},
		res:    uint256.Int{0x00, 0x00, 0x00, 0x00},
		status: RUNNING,
	},
	{
		name: "opSub: -1 - x",
		op:   opSub,
		data: []uint256.Int{
			{0x123456789ABCDEF0, 0x23456789ABCDEF01, 0x3456789ABCDEF012, 0x456789ABCDEF0123},
			{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF}},
		res:    uint256.Int{0xEDCBA9876543210F, 0xDCBA9876543210FE, 0xCBA9876543210FED, 0xBA9876543210FEDC},
		status: RUNNING,
	},
	{
		name: "opSub: x - -1",
		op:   opSub,
		data: []uint256.Int{
			{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF},
			{0x123456789ABCDEF0, 0x23456789ABCDEF01, 0x3456789ABCDEF012, 0x456789ABCDEF0123}},
		res:    uint256.Int{0x123456789ABCDEF1, 0x23456789ABCDEF01, 0x3456789ABCDEF012, 0x456789ABCDEF0123},
		status: RUNNING,
	},
	{
		name: "opSub: 0x5..5 - 0xA..A (x - ^x)",
		op:   opSub,
		data: []uint256.Int{
			{0xAAAAAAAAAAAAAAAA, 0xAAAAAAAAAAAAAAAA, 0xAAAAAAAAAAAAAAAA, 0xAAAAAAAAAAAAAAAA},
			{0x5555555555555555, 0x5555555555555555, 0x5555555555555555, 0x5555555555555555}},
		res:    uint256.Int{0xAAAAAAAAAAAAAAAB, 0xAAAAAAAAAAAAAAAA, 0xAAAAAAAAAAAAAAAA, 0xAAAAAAAAAAAAAAAA},
		status: RUNNING,
	},
	{
		name: "opSub: 0xA..A - 0x5..5 (x - ^x)",
		op:   opSub,
		data: []uint256.Int{
			{0x5555555555555555, 0x5555555555555555, 0x5555555555555555, 0x5555555555555555},
			{0xAAAAAAAAAAAAAAAA, 0xAAAAAAAAAAAAAAAA, 0xAAAAAAAAAAAAAAAA, 0xAAAAAAAAAAAAAAAA}},
		res:    uint256.Int{0x5555555555555555, 0x5555555555555555, 0x5555555555555555, 0x5555555555555555},
		status: RUNNING,
	},
	{
		name: "opSub: overflow between 8 bytes",
		op:   opSub,
		data: []uint256.Int{
			{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0x00},
			{0x01, 0x00, 0x00, 0x00}},
		res:    uint256.Int{0x02, 0x00, 0x00, 0xFFFFFFFFFFFFFFFF},
		status: RUNNING,
	},
	{
		name: "opSub: diff in the upper 8 bytes",
		op:   opSub,
		data: []uint256.Int{
			{0x00, 0x00, 0x00, 0x01},
			{0x00, 0x00, 0x00, 0x8000000000000000}},
		res:    uint256.Int{0x00, 0x00, 0x00, 0x7FFFFFFFFFFFFFFF},
		status: RUNNING,
	},
	{
		name: "opSub: 0x8..0 - 1",
		op:   opSub,
		data: []uint256.Int{
			{0x01, 0x00, 0x00, 0x00},
			{0x00, 0x00, 0x00, 0x8000000000000000}},
		res:    uint256.Int{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0x7FFFFFFFFFFFFFFF},
		status: RUNNING,
	},
	{
		name: "opSub: various 8*bytes",
		op:   opSub,
		data: []uint256.Int{
			{0xFFFFFFFFFFFFFFFF, 0x00, 0xFFFFFFFFFFFFFFFF, 0x00},
			{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0x00, 0x00}},
		res:    uint256.Int{0x00, 0xFFFFFFFFFFFFFFFF, 0x01, 0xFFFFFFFFFFFFFFFF},
		status: RUNNING,
	},
	{
		name: "opSub: first and last bit",
		op:   opSub,
		data: []uint256.Int{
			{0x01, 0x00, 0x00, 0x8000000000000000},
			{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF}},
		res:    uint256.Int{0xFFFFFFFFFFFFFFFE, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0x7FFFFFFFFFFFFFFF},
		status: RUNNING,
	},
	{
		name: "opSub: 0x00FF - 0x1234",
		op:   opSub,
		data: []uint256.Int{
			{0x1234, 0x00, 0x00, 0x00},
			{0x00FF, 0x00, 0x00, 0x00}},
		res:    uint256.Int{0xFFFFFFFFFFFFEECB, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF},
		status: RUNNING,
	},
	{
		name: "opSub: diff in the upper 16 bytes",
		op:   opSub,
		data: []uint256.Int{
			{0x00, 0x00, 0x1234, 0x00},
			{0x00, 0x00, 0x1234, 0xFFFF}},
		res:    uint256.Int{0x00, 0x00, 0x00, 0xFFFF},
		status: RUNNING,
	},
	{
		name: "opSub: last 2 bytes of each 8",
		op:   opSub,
		data: []uint256.Int{
			{0xFF00, 0x0FF0, 0x00FF, 0xF0F0},
			{0x1234, 0xABCD, 0x5678, 0xEF90}},
		res:    uint256.Int{0xFFFFFFFFFFFF1334, 0x0000000000009BDC, 0x0000000000005579, 0xFFFFFFFFFFFFFEA0},
		status: RUNNING,
	},
	{
		name: "opSub: -1 - 0x1234",
		op:   opSub,
		data: []uint256.Int{
			{0x1234, 0x00, 0x00, 0x00},
			{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF}},
		res:    uint256.Int{0xFFFFFFFFFFFFEDCB, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF},
		status: RUNNING,
	},
	{
		name: "opSub: -10 - 0x1234",
		op:   opSub,
		data: []uint256.Int{
			{0x1234, 0x00, 0x00, 0x00},
			{0xFFFFFFFFFFFFFFF6, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF}},
		res:    uint256.Int{0xFFFFFFFFFFFFEDC2, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF},
		status: RUNNING,
	},

	// operation Mul
	{
		name: "opMul: 0 * 0",
		op:   opMul,
		data: []uint256.Int{
			{0, 0, 0, 0},
			{0, 0, 0, 0}},
		res:    uint256.Int{0, 0, 0, 0},
		status: RUNNING,
	},
	{
		name: "opMul: -1 * 0",
		op:   opMul,
		data: []uint256.Int{
			{0x00, 0x00, 0x00, 0x00},
			{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF}},
		res:    uint256.Int{0x00, 0x00, 0x00, 0x00},
		status: RUNNING,
	},
	{
		name: "opMul: 0 * -1",
		op:   opMul,
		data: []uint256.Int{
			{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF},
			{0x00, 0x00, 0x00, 0x00}},
		res:    uint256.Int{0x00, 0x00, 0x00, 0x00},
		status: RUNNING,
	},
	{
		name: "opMul: 1 * -1",
		op:   opMul,
		data: []uint256.Int{
			{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF},
			{0x01, 0x00, 0x00, 0x00}},
		res:    uint256.Int{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF},
		status: RUNNING,
	},
	{
		name: "opMul: -1 * 1",
		op:   opMul,
		data: []uint256.Int{
			{0x01, 0x00, 0x00, 0x00},
			{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF}},
		res:    uint256.Int{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF},
		status: RUNNING,
	},
	{
		name: "opMul: -1 * -1",
		op:   opMul,
		data: []uint256.Int{
			{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF},
			{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF}},
		res:    uint256.Int{0x01, 0x00, 0x00, 0x00},
		status: RUNNING,
	},
	{
		name: "opMul: -1 * x",
		op:   opMul,
		data: []uint256.Int{
			{0x123456789ABCDEF0, 0x23456789ABCDEF01, 0x3456789ABCDEF012, 0x456789ABCDEF0123},
			{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF}},
		res:    uint256.Int{0xEDCBA98765432110, 0xDCBA9876543210FE, 0xCBA9876543210FED, 0xBA9876543210FEDC},
		status: RUNNING,
	},
	{
		name: "opMul: x * -1",
		op:   opMul,
		data: []uint256.Int{
			{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF},
			{0x123456789ABCDEF0, 0x23456789ABCDEF01, 0x3456789ABCDEF012, 0x456789ABCDEF0123}},
		res:    uint256.Int{0xEDCBA98765432110, 0xDCBA9876543210FE, 0xCBA9876543210FED, 0xBA9876543210FEDC},
		status: RUNNING,
	},
	{
		name: "opMul: 0x5..5 * 0xA..A (x * ^x)",
		op:   opMul,
		data: []uint256.Int{
			{0xAAAAAAAAAAAAAAAA, 0xAAAAAAAAAAAAAAAA, 0xAAAAAAAAAAAAAAAA, 0xAAAAAAAAAAAAAAAA},
			{0x5555555555555555, 0x5555555555555555, 0x5555555555555555, 0x5555555555555555}},
		res:    uint256.Int{0x1C71C71C71C71C72, 0x71C71C71C71C71C7, 0xC71C71C71C71C71C, 0x1C71C71C71C71C71},
		status: RUNNING,
	},
	{
		name: "opMul: 0xA..A * 0x5..5 (x * ^x)",
		op:   opMul,
		data: []uint256.Int{
			{0x5555555555555555, 0x5555555555555555, 0x5555555555555555, 0x5555555555555555},
			{0xAAAAAAAAAAAAAAAA, 0xAAAAAAAAAAAAAAAA, 0xAAAAAAAAAAAAAAAA, 0xAAAAAAAAAAAAAAAA}},
		res:    uint256.Int{0x1C71C71C71C71C72, 0x71C71C71C71C71C7, 0xC71C71C71C71C71C, 0x1C71C71C71C71C71},
		status: RUNNING,
	},
	{
		name: "opMul: 0x01 * 0x00F..F",
		op:   opMul,
		data: []uint256.Int{
			{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0x00},
			{0x01, 0x00, 0x00, 0x00}},
		res:    uint256.Int{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0x00},
		status: RUNNING,
	},
	{
		name: "opMul: overflow",
		op:   opMul,
		data: []uint256.Int{
			{0xFFFFFFFFFFFFFFFF, 0x00, 0xFFFFFFFFFFFFFFFF, 0x00},
			{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0x00, 0x00}},
		res:    uint256.Int{1, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF},
		status: RUNNING,
	},
	{
		name: "opMul: first and last bit",
		op:   opMul,
		data: []uint256.Int{
			{0x01, 0x00, 0x00, 0x8000000000000000},
			{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF}},
		res:    uint256.Int{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0x7FFFFFFFFFFFFFFF},
		status: RUNNING,
	},
	{
		name: "opMul: last 2 bytes",
		op:   opMul,
		data: []uint256.Int{
			{0x1234, 0x00, 0x00, 0x00},
			{0x00FF, 0x00, 0x00, 0x00}},
		res:    uint256.Int{0x1221CC, 0x00, 0x00, 0x00},
		status: RUNNING,
	},
	{
		name: "opMul: multiplication in the upper 16 bytes",
		op:   opMul,
		data: []uint256.Int{
			{0x00, 0x00, 0x1234, 0x00},
			{0x00, 0x00, 0x1234, 0xFFFF}},
		res:    uint256.Int{0x00, 0x00, 0x00, 0x00},
		status: RUNNING,
	},
	{
		name: "opMul: last 2 bytes of each 8",
		op:   opMul,
		data: []uint256.Int{
			{0xFF00, 0x0FF0, 0x00FF, 0xF0F0},
			{0x1234, 0xABCD, 0x5678, 0xEF90}},
		res:    uint256.Int{0x000000001221CC00, 0x00000000AC434FC0, 0x0000000060E5BCFC, 0x0000000105CF7A73},
		status: RUNNING,
	},
	{
		name: "opMul: -1 * 0x1234",
		op:   opMul,
		data: []uint256.Int{
			{0x1234, 0x00, 0x00, 0x00},
			{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF}},
		res:    uint256.Int{0xFFFFFFFFFFFFEDCC, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF},
		status: RUNNING,
	},
	{
		name: "opMul: -10 * 0x1234",
		op:   opMul,
		data: []uint256.Int{
			{0x1234, 0x00, 0x00, 0x00},
			{0xFFFFFFFFFFFFFFF6, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF}},
		res:    uint256.Int{0xFFFFFFFFFFFF49F8, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF},
		status: RUNNING,
	},
	{
		name: "opMul: 0xF..F * 0x7F..F",
		op:   opMul,
		data: []uint256.Int{
			{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0x7FFFFFFFFFFFFFFF},
			{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF}},
		res:    uint256.Int{0x01, 0x00, 0x00, 0x8000000000000000},
		status: RUNNING,
	},
	{
		name: "opMul: 0xF..F * 0x80..0",
		op:   opMul,
		data: []uint256.Int{
			{0x00, 0x00, 0x00, 0x8000000000000000},
			{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF}},
		res:    uint256.Int{0x00, 0x00, 0x00, 0x8000000000000000},
		status: RUNNING,
	},
	{
		name: "opMul: 0x80..01 * 0xF..F",
		op:   opMul,
		data: []uint256.Int{
			{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF},
			{0x01, 0x00, 0x00, 0x8000000000000000}},
		res:    uint256.Int{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0x7FFFFFFFFFFFFFFF},
		status: RUNNING,
	},

	// operation MulMod
	// (a * b) % N, data: {N, b, a}
	{
		name: "opMulMod: (0 * 0) mod 0",
		op:   opMulMod,
		data: []uint256.Int{
			{0, 0, 0, 0},
			{0, 0, 0, 0},
			{0, 0, 0, 0}},
		res:    uint256.Int{0, 0, 0, 0},
		status: RUNNING,
	},
	{
		name: "opMulMod: (0 * 0) mod 1",
		op:   opMulMod,
		data: []uint256.Int{
			{1, 0, 0, 0},
			{0, 0, 0, 0},
			{0, 0, 0, 0}},
		res:    uint256.Int{0, 0, 0, 0},
		status: RUNNING,
	},
	{
		name: "opMulMod: (0 * -1) mod 1",
		op:   opMulMod,
		data: []uint256.Int{
			{0x01, 0x00, 0x00, 0x00},
			{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF},
			{0x00, 0x00, 0x00, 0x00}},
		res:    uint256.Int{0x00, 0x00, 0x00, 0x00},
		status: RUNNING,
	},
	{
		name: "opMulMod: (-1 * 0) mod -1",
		op:   opMulMod,
		data: []uint256.Int{
			{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF},
			{0x00, 0x00, 0x00, 0x00},
			{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF}},
		res:    uint256.Int{0x00, 0x00, 0x00, 0x00},
		status: RUNNING,
	},
	{
		name: "opMulMod: (0 * 1) mod -1",
		op:   opMulMod,
		data: []uint256.Int{
			{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF},
			{0x01, 0x00, 0x00, 0x00},
			{0x00, 0x00, 0x00, 0x00}},
		res:    uint256.Int{0x00, 0x00, 0x00, 0x00},
		status: RUNNING,
	},
	{ // different results for int and uint
		name: "opMulMod: (1 * -1) mod 0xFF00 // result? int -1, uint 0xFF",
		op:   opMulMod,
		data: []uint256.Int{
			{0xFF00, 0x00, 0x00, 0x00},
			{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF},
			{0x01, 0x00, 0x00, 0x00}},
		//res: uint256.Int{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF}, // int
		res:    uint256.Int{0xFF, 0x00, 0x00, 0x00},
		status: RUNNING,
	},
	{ // different results for int and uint
		name: "opMulMod: (1 * -1) mod (2**128) // result? int -1, uint (2**128)-1",
		op:   opMulMod,
		data: []uint256.Int{
			{0x00, 0x00, 0x01, 0x00},
			{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF},
			{0x01, 0x00, 0x00, 0x00}},
		//res: uint256.Int{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF}, // int
		res:    uint256.Int{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0x00, 0x00},
		status: RUNNING,
	},
	{ // different results for int and uint
		name: "opMulMod: (-1 * -1) mod 15 // result? int 1, uint 0",
		op:   opMulMod,
		data: []uint256.Int{
			{0x0F, 0x00, 0x00, 0x00},
			{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF},
			{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF}},
		//res: uint256.Int{0x01, 0x00, 0x00, 0x00}, // int
		res:    uint256.Int{0x00, 0x00, 0x00, 0x00},
		status: RUNNING,
	},
	{ // different results for int and uint
		name: "opMulMod: (-1 * x) mod (2**64) // result? int 0xF..F123456789ABCDEF0, uint 0xEDCBA98765432110",
		op:   opMulMod,
		data: []uint256.Int{
			{0x00, 0x01, 0x00, 0x00},
			{0x123456789ABCDEF0, 0x23456789ABCDEF01, 0x3456789ABCDEF012, 0x456789ABCDEF0123},
			{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF}},
		//res: uint256.Int{0x123456789ABCDEF0, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF}, // int
		res:    uint256.Int{0xEDCBA98765432110, 0x00, 0x00, 0x00},
		status: RUNNING,
	},
	{ // different results for int and uint
		name: "opMulMod: (x * -1) mod y // result? int 0xF..F1146, uint 0xEEBA",
		op:   opMulMod,
		data: []uint256.Int{
			{0xFFFE, 0x00, 0x00, 0x00},
			{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF},
			{0x123456789ABCDEF0, 0x23456789ABCDEF01, 0x3456789ABCDEF012, 0x456789ABCDEF0123}},
		//res: uint256.Int{0xFFFFFFFFFFFF1146, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF}, // int
		res:    uint256.Int{0xEEBA, 0x00, 0x00, 0x00},
		status: RUNNING,
	},
	{ // different results for int and uint
		name: "opMulMod: (0x5..5 * 0xA..A) mod (2**192) // result? int x uint",
		op:   opMulMod,
		data: []uint256.Int{
			{0x00, 0x00, 0x00, 0x01},
			{0xAAAAAAAAAAAAAAAA, 0xAAAAAAAAAAAAAAAA, 0xAAAAAAAAAAAAAAAA, 0xAAAAAAAAAAAAAAAA},
			{0x5555555555555555, 0x5555555555555555, 0x5555555555555555, 0x5555555555555555}},
		//res: uint256.Int{0x1C71C71C71C71C72, 0x71C71C71C71C71C7, 0xC71C71C71C71C71C, 0xFFFFFFFFFFFFFFFF}, // int
		res:    uint256.Int{0x1C71C71C71C71C72, 0x71C71C71C71C71C7, 0xC71C71C71C71C71C, 0x00},
		status: RUNNING,
	},
	{
		name: "opMulMod: (0xA..A * 0x5..5) mod 0xAAAA",
		op:   opMulMod,
		data: []uint256.Int{
			{0xAAAA, 0x00, 0x00, 0x00},
			{0x5555555555555555, 0x5555555555555555, 0x5555555555555555, 0x5555555555555555},
			{0xAAAAAAAAAAAAAAAA, 0xAAAAAAAAAAAAAAAA, 0xAAAAAAAAAAAAAAAA, 0xAAAAAAAAAAAAAAAA}},
		res:    uint256.Int{0x00, 0x00, 0x00, 0x00},
		status: RUNNING,
	},
	{
		name: "opMulMod: (1 * x) mod y",
		op:   opMulMod,
		data: []uint256.Int{
			{0x123456789ABCDEF0, 0x123456789ABCDEF0, 0x123456789ABCDEF0, 0x123456789ABCDEF0},
			{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0x00},
			{0x01, 0x00, 0x00, 0x00}},
		res:    uint256.Int{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0x00},
		status: RUNNING,
	},
	{
		name: "opMulMod: various 8*bytes",
		op:   opMulMod,
		data: []uint256.Int{
			{0x00, 0x01, 0x00, 0x00},
			{0xFFFFFFFFFFFFFFFF, 0x00, 0xFFFFFFFFFFFFFFFF, 0x00},
			{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0x00, 0x00}},
		res:    uint256.Int{0x01, 0x00, 0x00, 0x00},
		status: RUNNING,
	},
	{ // different results for int and uint
		name: "opMulMod: first and last bit // result? int x uint",
		op:   opMulMod,
		data: []uint256.Int{
			{0xFFFFFFFFFFFFFFFC, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF},
			{0x01, 0x00, 0x00, 0x8000000000000000},
			{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF}},
		//res: uint256.Int{0x03, 0x00, 0x00, 0x00}, // int
		res:    uint256.Int{0x07, 0x00, 0x00, 0x8000000000000000},
		status: RUNNING,
	},
	{
		name: "opMulMod: (0x00FF * 0x1234) mod 0xFF00",
		op:   opMulMod,
		data: []uint256.Int{
			{0xFF00, 0x00, 0x00, 0x00},
			{0x1234, 0x00, 0x00, 0x00},
			{0x00FF, 0x00, 0x00, 0x00}},
		res:    uint256.Int{0x33CC, 0x00, 0x00, 0x00},
		status: RUNNING,
	},
	{
		name: "opMulMod: mul in the upper 16 bytes",
		op:   opMulMod,
		data: []uint256.Int{
			{0x07, 0x00, 0x00, 0x00},
			{0x00, 0x00, 0x1234, 0x00},
			{0x00, 0x00, 0x1234, 0xFFFF}},
		res:    uint256.Int{0x00, 0x00, 0x00, 0x00},
		status: RUNNING,
	},
	{
		name: "opMulMod: last 2 bytes of each 8",
		op:   opMulMod,
		data: []uint256.Int{
			{0x000000001221CC01, 0x00, 0x00, 0x00},
			{0xFF00, 0x0FF0, 0x00FF, 0xF0F0},
			{0x1234, 0xABCD, 0x5678, 0xEF90}},
		res:    uint256.Int{0x0BB60F29, 0x00, 0x00, 0x00},
		status: RUNNING,
	},
	{ // different results for int and uint
		name: "opMulMod: (-1 * 0x1234) mod 0xFF // result? int x uint",
		op:   opMulMod,
		data: []uint256.Int{
			{0xFF, 0x00, 0x00, 0x00},
			{0x1234, 0x00, 0x00, 0x00},
			{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF}},
		//res: uint256.Int{0xFFFFFFFFFFFFFFBA, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF}, // int
		res:    uint256.Int{0x00, 0x00, 0x00, 0x00},
		status: RUNNING,
	},
	{ // different results for int and uint
		name: "opMulMod: (-10 * 0x1234) mod 0xFF // result? int x uint",
		op:   opMulMod,
		data: []uint256.Int{
			{0xFF, 0x00, 0x00, 0x00},
			{0x1234, 0x00, 0x00, 0x00},
			{0xFFFFFFFFFFFFFFF6, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF}},
		//res: uint256.Int{0xFFFFFFFFFFFFFF42, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF}, // int
		res:    uint256.Int{0x87, 0x00, 0x00, 0x00},
		status: RUNNING,
	},
	{ // different results for int and uint
		name: "opMulMod: (1 * -1) mod 0x1234 // result? int x uint",
		op:   opMulMod,
		data: []uint256.Int{
			{0x1234, 0x00, 0x00, 0x00},
			{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF},
			{0x01, 0x00, 0x00, 0x00}},
		//res: uint256.Int{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF}, // int
		res:    uint256.Int{0x04BF, 0x00, 0x00, 0x00},
		status: RUNNING,
	},
	{ // different results for int and uint
		name: "opMulMod: (1 * -10) mod 0x1234 // result? int x uint",
		op:   opMulMod,
		data: []uint256.Int{
			{0x1234, 0x00, 0x00, 0x00},
			{0xFFFFFFFFFFFFFFF6, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF},
			{0x01, 0x00, 0x00, 0x00}},
		//res: uint256.Int{0xFFFFFFFFFFFFFFF6, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF}, // int
		res:    uint256.Int{0x04B6, 0x00, 0x00, 0x00},
		status: RUNNING,
	},
	{ // different results for int and uint
		name: "opMulMod: (1 * (-1)) mod 2 // result? int x uint",
		op:   opMulMod,
		data: []uint256.Int{
			{0x02, 0x00, 0x00, 0x00},
			{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF},
			{0x01, 0x00, 0x00, 0x00}},
		//res: uint256.Int{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF}, // int
		res:    uint256.Int{0x01, 0x00, 0x00, 0x00},
		status: RUNNING,
	},
	{
		name: "opMulMod: (1 * 1) mod -2",
		op:   opMulMod,
		data: []uint256.Int{
			{0xFFFFFFFFFFFFFFFE, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF},
			{0x01, 0x00, 0x00, 0x00},
			{0x01, 0x00, 0x00, 0x00}},
		res:    uint256.Int{0x01, 0x00, 0x00, 0x00},
		status: RUNNING,
	},
	{ // different results for int and uint
		name: "opMulMod: (1 * (-1)) mod -2 // result? int x uint",
		op:   opMulMod,
		data: []uint256.Int{
			{0xFFFFFFFFFFFFFFFE, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF},
			{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF},
			{0x01, 0x00, 0x00, 0x00}},
		//res: uint256.Int{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF}, // int
		res:    uint256.Int{0x01, 0x00, 0x00, 0x00},
		status: RUNNING,
	},
	{ // different results for int and uint
		name: "opMulMod: (2 * (-1)) mod 3 // result? int x uint",
		op:   opMulMod,
		data: []uint256.Int{
			{0x03, 0x00, 0x00, 0x00},
			{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF},
			{0x02, 0x00, 0x00, 0x00}},
		//res: uint256.Int{0xFFFFFFFFFFFFFFFE, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF}, // int
		res:    uint256.Int{0x00, 0x00, 0x00, 0x00},
		status: RUNNING,
	},

	// operation Div
	// a / b (uint), data: {b, a}
	{
		name: "opDiv: 0 / 0",
		op:   opDiv,
		data: []uint256.Int{
			{0, 0, 0, 0},
			{0, 0, 0, 0}},
		res:    uint256.Int{0, 0, 0, 0},
		status: RUNNING,
	},
	{
		name: "opDiv: max / 0",
		op:   opDiv,
		data: []uint256.Int{
			{0x00, 0x00, 0x00, 0x00},
			{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF}},
		res:    uint256.Int{0x00, 0x00, 0x00, 0x00},
		status: RUNNING,
	},
	{
		name: "opDiv: 0 / max",
		op:   opDiv,
		data: []uint256.Int{
			{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF},
			{0x00, 0x00, 0x00, 0x00}},
		res:    uint256.Int{0x00, 0x00, 0x00, 0x00},
		status: RUNNING,
	},
	{
		name: "opDiv: 1 / max",
		op:   opDiv,
		data: []uint256.Int{
			{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF},
			{0x01, 0x00, 0x00, 0x00}},
		res:    uint256.Int{0x00, 0x00, 0x00, 0x00},
		status: RUNNING,
	},
	{
		name: "opDiv: max / 1",
		op:   opDiv,
		data: []uint256.Int{
			{0x01, 0x00, 0x00, 0x00},
			{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF}},
		res:    uint256.Int{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF},
		status: RUNNING,
	},
	{
		name: "opDiv: max / max",
		op:   opDiv,
		data: []uint256.Int{
			{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF},
			{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF}},
		res:    uint256.Int{0x01, 0x00, 0x00, 0x00},
		status: RUNNING,
	},
	{
		name: "opDiv: 3 / x",
		op:   opDiv,
		data: []uint256.Int{
			{0x123456789ABCDEF0, 0x23456789ABCDEF01, 0x3456789ABCDEF012, 0x456789ABCDEF0123},
			{0x03, 0x00, 0x00, 0x00}},
		res:    uint256.Int{0x00, 0x00, 0x00, 0x00},
		status: RUNNING,
	},
	{
		name: "opDiv: x / max",
		op:   opDiv,
		data: []uint256.Int{
			{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF},
			{0x123456789ABCDEF0, 0x23456789ABCDEF01, 0x3456789ABCDEF012, 0x456789ABCDEF0123}},
		res:    uint256.Int{0x00, 0x00, 0x00, 0x00},
		status: RUNNING,
	},
	{
		name: "opDiv: 0x5..5 / 0xA..A (x / ^x)",
		op:   opDiv,
		data: []uint256.Int{
			{0xAAAAAAAAAAAAAAAA, 0xAAAAAAAAAAAAAAAA, 0xAAAAAAAAAAAAAAAA, 0xAAAAAAAAAAAAAAAA},
			{0x5555555555555555, 0x5555555555555555, 0x5555555555555555, 0x5555555555555555}},
		res:    uint256.Int{0x00, 0x00, 0x00, 0x00},
		status: RUNNING,
	},
	{
		name: "opDiv: 0xA..A / 0x5..5 (x / ^x)",
		op:   opDiv,
		data: []uint256.Int{
			{0x5555555555555555, 0x5555555555555555, 0x5555555555555555, 0x5555555555555555},
			{0xAAAAAAAAAAAAAAAA, 0xAAAAAAAAAAAAAAAA, 0xAAAAAAAAAAAAAAAA, 0xAAAAAAAAAAAAAAAA}},
		res:    uint256.Int{0x02, 0x00, 0x00, 0x00},
		status: RUNNING,
	},
	{
		name: "opDiv: 0x01 / 0x0F..F",
		op:   opDiv,
		data: []uint256.Int{
			{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0x00},
			{0x01, 0x00, 0x00, 0x00}},
		res:    uint256.Int{0x00, 0x00, 0x00, 0x00},
		status: RUNNING,
	},
	{
		name: "opDiv: various 8*bytes",
		op:   opDiv,
		data: []uint256.Int{
			{0xFFFFFFFFFFFFFFFF, 0x00, 0xFFFFFFFFFFFFFFFF, 0x00},
			{0x01, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF}},
		res:    uint256.Int{0x00, 0x01, 0x00, 0x00},
		status: RUNNING,
	},
	{
		name: "opDiv: first and last bit",
		op:   opDiv,
		data: []uint256.Int{
			{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0x7FFFFFFFFFFFFFFF},
			{0x01, 0x00, 0x00, 0x8000000000000000}},
		res:    uint256.Int{0x01, 0x00, 0x00, 0x00},
		status: RUNNING,
	},
	{
		name: "opDiv: last 8 bytes",
		op:   opDiv,
		data: []uint256.Int{
			{0x1234, 0x00, 0x00, 0x00},
			{0x1221CC, 0x00, 0x00, 0x00}},
		res:    uint256.Int{0x00FF, 0x00, 0x00, 0x00},
		status: RUNNING,
	},
	{
		name: "opDiv: div in the upper 16 bytes",
		op:   opDiv,
		data: []uint256.Int{
			{0x00, 0x00, 0x1234, 0x00},
			{0x00, 0x00, 0x1234, 0xFFFF}},
		res:    uint256.Int{0x1034BCFA81A5E7D5, 0x000E, 0x00, 0x00},
		status: RUNNING,
	},
	{
		name: "opDiv: each 8*bytes",
		op:   opDiv,
		data: []uint256.Int{
			{0xFF00, 0x0FF0, 0x00FF, 0xF0F0},
			{0x000000001221CC00, 0x00000000AC434FC0, 0x0000000060E5BCFC, 0x0000000105CF7A73}},
		res:    uint256.Int{0x01162D, 0x00, 0x00, 0x00},
		status: RUNNING,
	},
	{
		name: "opDiv: x / 0x1234",
		op:   opDiv,
		data: []uint256.Int{
			{0x1234, 0x00, 0x00, 0x00},
			{0xFFFFFFFFFFFFEDCC, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF}},
		res:    uint256.Int{0x743E68286EC00E0F, 0xE1042CD3D4EE336B, 0x36B743E68286EC00, 0x000E1042CD3D4EE3},
		status: RUNNING,
	},
	{
		name: "opDiv: y / 0x1234",
		op:   opDiv,
		data: []uint256.Int{
			{0x1234, 0x00, 0x00, 0x00},
			{0xFFFFFFFFFFFF49F8, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF}},
		res:    uint256.Int{0x743E68286EC00E06, 0xE1042CD3D4EE336B, 0x36B743E68286EC00, 0x000E1042CD3D4EE3},
		status: RUNNING,
	},

	// operation SDiv
	// a / b (int), data: {b, a}
	{
		name: "opSDiv: 0 / 0",
		op:   opSDiv,
		data: []uint256.Int{
			{0, 0, 0, 0},
			{0, 0, 0, 0}},
		res:    uint256.Int{0, 0, 0, 0},
		status: RUNNING,
	},
	{
		name: "opSDiv: -1 / 0",
		op:   opSDiv,
		data: []uint256.Int{
			{0x00, 0x00, 0x00, 0x00},
			{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF}},
		res:    uint256.Int{0x00, 0x00, 0x00, 0x00},
		status: RUNNING,
	},
	{
		name: "opSDiv: 0 / -1",
		op:   opSDiv,
		data: []uint256.Int{
			{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF},
			{0x00, 0x00, 0x00, 0x00}},
		res:    uint256.Int{0x00, 0x00, 0x00, 0x00},
		status: RUNNING,
	},
	{
		name: "opSDiv: 1 / -1",
		op:   opSDiv,
		data: []uint256.Int{
			{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF},
			{0x01, 0x00, 0x00, 0x00}},
		res:    uint256.Int{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF},
		status: RUNNING,
	},
	{
		name: "opSDiv: -1 / 1",
		op:   opSDiv,
		data: []uint256.Int{
			{0x01, 0x00, 0x00, 0x00},
			{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF}},
		res:    uint256.Int{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF},
		status: RUNNING,
	},
	{
		name: "opSDiv: -1 / -1",
		op:   opSDiv,
		data: []uint256.Int{
			{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF},
			{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF}},
		res:    uint256.Int{0x01, 0x00, 0x00, 0x00},
		status: RUNNING,
	},
	{
		name: "opSDiv: min / -1",
		op:   opSDiv,
		data: []uint256.Int{
			{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF},
			{0x00, 0x00, 0x00, 0x8000000000000000}},
		res:    uint256.Int{0x00, 0x00, 0x00, 0x8000000000000000},
		status: RUNNING,
	},
	{
		name: "opSDiv: -1 / min",
		op:   opSDiv,
		data: []uint256.Int{
			{0x00, 0x00, 0x00, 0x8000000000000000},
			{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF}},
		res:    uint256.Int{0x00, 0x00, 0x00, 0x00},
		status: RUNNING,
	},
	{
		name: "opSDiv: -1 / x",
		op:   opSDiv,
		data: []uint256.Int{
			{0x123456789ABCDEF0, 0x23456789ABCDEF01, 0x3456789ABCDEF012, 0x456789ABCDEF0123},
			{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF}},
		res:    uint256.Int{0x00, 0x00, 0x00, 0x00},
		status: RUNNING,
	},
	{
		name: "opSDiv: x / -1",
		op:   opSDiv,
		data: []uint256.Int{
			{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF},
			{0x123456789ABCDEF0, 0x23456789ABCDEF01, 0x3456789ABCDEF012, 0x456789ABCDEF0123}},
		res:    uint256.Int{0xEDCBA98765432110, 0xDCBA9876543210FE, 0xCBA9876543210FED, 0xBA9876543210FEDC},
		status: RUNNING,
	},
	{
		name: "opSDiv: 0x5..5 / 0xA..A (x / ^x)",
		op:   opSDiv,
		data: []uint256.Int{
			{0xAAAAAAAAAAAAAAAA, 0xAAAAAAAAAAAAAAAA, 0xAAAAAAAAAAAAAAAA, 0xAAAAAAAAAAAAAAAA},
			{0x5555555555555555, 0x5555555555555555, 0x5555555555555555, 0x5555555555555555}},
		res:    uint256.Int{0x00, 0x00, 0x00, 0x00},
		status: RUNNING,
	},
	{
		name: "opSDiv: 0xA..A / 0x5..5 (x / ^x)",
		op:   opSDiv,
		data: []uint256.Int{
			{0x5555555555555555, 0x5555555555555555, 0x5555555555555555, 0x5555555555555555},
			{0xAAAAAAAAAAAAAAAA, 0xAAAAAAAAAAAAAAAA, 0xAAAAAAAAAAAAAAAA, 0xAAAAAAAAAAAAAAAA}},
		res:    uint256.Int{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF},
		status: RUNNING,
	},
	{
		name: "opSDiv: 0x01 / 0x0F..F",
		op:   opSDiv,
		data: []uint256.Int{
			{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0x00},
			{0x01, 0x00, 0x00, 0x00}},
		res:    uint256.Int{0x00, 0x00, 0x00, 0x00},
		status: RUNNING,
	},
	{
		name: "opSDiv: various 8*bytes",
		op:   opSDiv,
		data: []uint256.Int{
			{0xFFFFFFFFFFFFFFFF, 0x00, 0xFFFFFFFFFFFFFFFF, 0x00},
			{0x01, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF}},
		res:    uint256.Int{0x00, 0x00, 0x00, 0x00},
		status: RUNNING,
	},
	{
		name: "opSDiv: first and last bit",
		op:   opSDiv,
		data: []uint256.Int{
			{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0x7FFFFFFFFFFFFFFF},
			{0x01, 0x00, 0x00, 0x8000000000000000}},
		res:    uint256.Int{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF},
		status: RUNNING,
	},
	{
		name: "opSDiv: last 8 bytes",
		op:   opSDiv,
		data: []uint256.Int{
			{0x1234, 0x00, 0x00, 0x00},
			{0x1221CC, 0x00, 0x00, 0x00}},
		res:    uint256.Int{0xFF, 0x00, 0x00, 0x00},
		status: RUNNING,
	},
	{
		name: "opSDiv: div in the upper 16 bytes",
		op:   opSDiv,
		data: []uint256.Int{
			{0x00, 0x00, 0x1234, 0x00},
			{0x00, 0x00, 0x1234, 0xFFFF}},
		res:    uint256.Int{0x1034BCFA81A5E7D5, 0x000E, 0x00, 0x00},
		status: RUNNING,
	},
	{
		name: "opSDiv: each 8*bytes",
		op:   opSDiv,
		data: []uint256.Int{
			{0xFF00, 0x0FF0, 0x00FF, 0xF0F0},
			{0x000000001221CC00, 0x00000000AC434FC0, 0x0000000060E5BCFC, 0x0000000105CF7A73}},
		res:    uint256.Int{0x01162D, 0x00, 0x00, 0x00},
		status: RUNNING,
	},
	{
		name: "opSDiv: x / 0x1234 = -1",
		op:   opSDiv,
		data: []uint256.Int{
			{0x1234, 0x00, 0x00, 0x00},
			{0xFFFFFFFFFFFFEDCC, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF}},
		res:    uint256.Int{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF},
		status: RUNNING,
	},
	{
		name: "opSDiv: y / 0x1234 = -10",
		op:   opSDiv,
		data: []uint256.Int{
			{0x1234, 0x00, 0x00, 0x00},
			{0xFFFFFFFFFFFF49F8, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF}},
		res:    uint256.Int{0xFFFFFFFFFFFFFFF6, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF},
		status: RUNNING,
	},

	// operation Mod
	// a % b (uint), data: {b, a}
	{
		name: "opMod: 0 mod 0",
		op:   opMod,
		data: []uint256.Int{
			{0, 0, 0, 0},
			{0, 0, 0, 0}},
		res:    uint256.Int{0, 0, 0, 0},
		status: RUNNING,
	},
	{
		name: "opMod: max mod 0",
		op:   opMod,
		data: []uint256.Int{
			{0x00, 0x00, 0x00, 0x00},
			{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF}},
		res:    uint256.Int{0x00, 0x00, 0x00, 0x00},
		status: RUNNING,
	},
	{
		name: "opMod: 0 mod max",
		op:   opMod,
		data: []uint256.Int{
			{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF},
			{0x00, 0x00, 0x00, 0x00}},
		res:    uint256.Int{0x00, 0x00, 0x00, 0x00},
		status: RUNNING,
	},
	{
		name: "opMod: 1 mod max",
		op:   opMod,
		data: []uint256.Int{
			{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF},
			{0x01, 0x00, 0x00, 0x00}},
		res:    uint256.Int{0x01, 0x00, 0x00, 0x00},
		status: RUNNING,
	},
	{
		name: "opMod: max mod 1",
		op:   opMod,
		data: []uint256.Int{
			{0x01, 0x00, 0x00, 0x00},
			{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF}},
		res:    uint256.Int{0x00, 0x00, 0x00, 0x00},
		status: RUNNING,
	},
	{
		name: "opMod: max mod max",
		op:   opMod,
		data: []uint256.Int{
			{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF},
			{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF}},
		res:    uint256.Int{0x00, 0x00, 0x00, 0x00},
		status: RUNNING,
	},
	{
		name: "opMod: 0x8..0 mod max",
		op:   opMod,
		data: []uint256.Int{
			{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF},
			{0x00, 0x00, 0x00, 0x8000000000000000}},
		res:    uint256.Int{0x00, 0x00, 0x00, 0x8000000000000000},
		status: RUNNING,
	},
	{
		name: "opMod: max mod 0x8..0",
		op:   opMod,
		data: []uint256.Int{
			{0x00, 0x00, 0x00, 0x8000000000000000},
			{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF}},
		res:    uint256.Int{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0x7FFFFFFFFFFFFFFF},
		status: RUNNING,
	},
	{
		name: "opMod: max mod x",
		op:   opMod,
		data: []uint256.Int{
			{0x123456789ABCDEF0, 0x23456789ABCDEF01, 0x3456789ABCDEF012, 0x456789ABCDEF0123},
			{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF}},
		res:    uint256.Int{0xC962FC962FC9632F, 0x962FC962FC9632FC, 0x62FC962FC9632FC9, 0x2FC962FC9632FC96},
		status: RUNNING,
	},
	{
		name: "opMod: x mod max",
		op:   opMod,
		data: []uint256.Int{
			{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF},
			{0x123456789ABCDEF0, 0x23456789ABCDEF01, 0x3456789ABCDEF012, 0x456789ABCDEF0123}},
		res:    uint256.Int{0x123456789ABCDEF0, 0x23456789ABCDEF01, 0x3456789ABCDEF012, 0x456789ABCDEF0123},
		status: RUNNING,
	},
	{
		name: "opMod: 0x5..5 mod 0xA..A (x mod ^x)",
		op:   opMod,
		data: []uint256.Int{
			{0xAAAAAAAAAAAAAAAA, 0xAAAAAAAAAAAAAAAA, 0xAAAAAAAAAAAAAAAA, 0xAAAAAAAAAAAAAAAA},
			{0x5555555555555555, 0x5555555555555555, 0x5555555555555555, 0x5555555555555555}},
		res:    uint256.Int{0x5555555555555555, 0x5555555555555555, 0x5555555555555555, 0x5555555555555555},
		status: RUNNING,
	},
	{
		name: "opMod: 0xA..A mod 0x5..5 (x mod ^x)",
		op:   opMod,
		data: []uint256.Int{
			{0x5555555555555555, 0x5555555555555555, 0x5555555555555555, 0x5555555555555555},
			{0xAAAAAAAAAAAAAAAA, 0xAAAAAAAAAAAAAAAA, 0xAAAAAAAAAAAAAAAA, 0xAAAAAAAAAAAAAAAA}},
		res:    uint256.Int{0x00, 0x00, 0x00, 0x00},
		status: RUNNING,
	},
	{
		name: "opMod: 0x01 mod 0x0F..F",
		op:   opMod,
		data: []uint256.Int{
			{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0x00},
			{0x01, 0x00, 0x00, 0x00}},
		res:    uint256.Int{0x01, 0x00, 0x00, 0x00},
		status: RUNNING,
	},
	{
		name: "opMod: various 8*bytes",
		op:   opMod,
		data: []uint256.Int{
			{0xFFFFFFFFFFFFFFFF, 0x00, 0xFFFFFFFFFFFFFFFF, 0x00},
			{0x01, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF}},
		res:    uint256.Int{0x01, 0x00, 0xFFFFFFFFFFFFFFFF, 0x00},
		status: RUNNING,
	},
	{
		name: "opMod: first and last bit",
		op:   opMod,
		data: []uint256.Int{
			{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0x7FFFFFFFFFFFFFFF},
			{0x01, 0x00, 0x00, 0x8000000000000000}},
		res:    uint256.Int{0x02, 0x00, 0x00, 0x00},
		status: RUNNING,
	},
	{
		name: "opMod: last 8 bytes",
		op:   opMod,
		data: []uint256.Int{
			{0x1234, 0x00, 0x00, 0x00},
			{0x1221CC, 0x00, 0x00, 0x00}},
		res:    uint256.Int{0x00, 0x00, 0x00, 0x00},
		status: RUNNING,
	},
	{
		name: "opMod: modulo in the upper 16 bytes",
		op:   opMod,
		data: []uint256.Int{
			{0x00, 0x00, 0x1234, 0x00},
			{0x00, 0x00, 0x1234, 0xFFFF}},
		res:    uint256.Int{0x00, 0x00, 0xF0, 0x00},
		status: RUNNING,
	},
	{
		name: "opMod: each 8*bytes",
		op:   opMod,
		data: []uint256.Int{
			{0xFF00, 0x0FF0, 0x00FF, 0xF0F0},
			{0x000000001221CC00, 0x00000000AC434FC0, 0x0000000060E5BCFC, 0x0000000105CF7A73}},
		res:    uint256.Int{0xFFFFFFFEFD0AF900, 0x9AF1E28F, 0x5FD0A629, 0x8043},
		status: RUNNING,
	},
	{
		name: "opMod: x mod 0x1234",
		op:   opMod,
		data: []uint256.Int{
			{0x1234, 0x00, 0x00, 0x00},
			{0xFFFFFFFFFFFFEDCC, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF}},
		res:    uint256.Int{0x04C0, 0x00, 0x00, 0x00},
		status: RUNNING,
	},
	{
		name: "opMod: y mod 0x1234",
		op:   opMod,
		data: []uint256.Int{
			{0x1234, 0x00, 0x00, 0x00},
			{0xFFFFFFFFFFFF49F8, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF}},
		res:    uint256.Int{0x04C0, 0x00, 0x00, 0x00},
		status: RUNNING,
	},
	{
		name: "opMod: max mod 0x1234",
		op:   opMod,
		data: []uint256.Int{
			{0x1234, 0x00, 0x00, 0x00},
			{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF}},
		res:    uint256.Int{0x04BF, 0x00, 0x00, 0x00},
		status: RUNNING,
	},

	{
		name: "opMod: max mod 2",
		op:   opMod,
		data: []uint256.Int{
			{0x02, 0x00, 0x00, 0x00},
			{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF}},
		res:    uint256.Int{0x01, 0x00, 0x00, 0x00},
		status: RUNNING,
	},
	{
		name: "opMod: max mod (max-1)",
		op:   opMod,
		data: []uint256.Int{
			{0xFFFFFFFFFFFFFFFE, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF},
			{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF}},
		res:    uint256.Int{0x01, 0x00, 0x00, 0x00},
		status: RUNNING,
	},

	// operation AddMod
	// (a + b) % N, data: {N, b, a}
	{
		name: "opAddMod: (0 + 0) mod 0",
		op:   opAddMod,
		data: []uint256.Int{
			{0, 0, 0, 0},
			{0, 0, 0, 0},
			{0, 0, 0, 0}},
		res:    uint256.Int{0, 0, 0, 0},
		status: RUNNING,
	},
	{
		name: "opAddMod: (0 + 0) mod 1",
		op:   opAddMod,
		data: []uint256.Int{
			{1, 0, 0, 0},
			{0, 0, 0, 0},
			{0, 0, 0, 0}},
		res:    uint256.Int{0, 0, 0, 0},
		status: RUNNING,
	},
	{
		name: "opAddMod: (0 + -1) mod 1",
		op:   opAddMod,
		data: []uint256.Int{
			{0x01, 0x00, 0x00, 0x00},
			{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF},
			{0x00, 0x00, 0x00, 0x00}},
		res:    uint256.Int{0x00, 0x00, 0x00, 0x00},
		status: RUNNING,
	},
	{
		name: "opAddMod: (-1 + 0) mod -1",
		op:   opAddMod,
		data: []uint256.Int{
			{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF},
			{0x00, 0x00, 0x00, 0x00},
			{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF}},
		res:    uint256.Int{0x00, 0x00, 0x00, 0x00},
		status: RUNNING,
	},
	{
		name: "opAddMod: (0 + 1) mod -1",
		op:   opAddMod,
		data: []uint256.Int{
			{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF},
			{0x01, 0x00, 0x00, 0x00},
			{0x00, 0x00, 0x00, 0x00}},
		res:    uint256.Int{0x01, 0x00, 0x00, 0x00},
		status: RUNNING,
	},
	{ // different results for int and uint
		name: "opAddMod: (1 + -1) mod 0xFF00 // result? int x uint",
		op:   opAddMod,
		data: []uint256.Int{
			{0xFF00, 0x00, 0x00, 0x00},
			{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF},
			{0x01, 0x00, 0x00, 0x00}},
		//res: uint256.Int{0x00, 0x00, 0x00, 0x00}, // int
		res:    uint256.Int{0x0100, 0x00, 0x00, 0x00},
		status: RUNNING,
	},
	{
		name: "opAddMod: (1 + -1) mod 0x01..0",
		op:   opAddMod,
		data: []uint256.Int{
			{0x00, 0x00, 0x01, 0x00},
			{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF},
			{0x01, 0x00, 0x00, 0x00}},
		res:    uint256.Int{0x00, 0x00, 0x00, 0x00},
		status: RUNNING,
	},
	{ // different results for int and uint
		name: "opAddMod: (-1 + -1) mod 15 // result? int x uint",
		op:   opAddMod,
		data: []uint256.Int{
			{0x0F, 0x00, 0x00, 0x00},
			{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF},
			{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF}},
		//res: uint256.Int{0xFFFFFFFFFFFFFFFE, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF}, // int
		res:    uint256.Int{0x00, 0x00, 0x00, 0x00},
		status: RUNNING,
	},
	{
		name: "opAddMod: (-1 + x) mod y",
		op:   opAddMod,
		data: []uint256.Int{
			{0x00, 0x01, 0x00, 0x00},
			{0x123456789ABCDEF0, 0x23456789ABCDEF01, 0x3456789ABCDEF012, 0x456789ABCDEF0123},
			{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF}},
		res:    uint256.Int{0x123456789ABCDEEF, 0x00, 0x00, 0x00},
		status: RUNNING,
	},
	{ // different results for int and uint
		name: "opAddMod: (x + -1) mod y // result? int x uint",
		op:   opAddMod,
		data: []uint256.Int{
			{0xFFFE, 0x00, 0x00, 0x00},
			{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF},
			{0x123456789ABCDEF0, 0x23456789ABCDEF01, 0x3456789ABCDEF012, 0x456789ABCDEF0123}},
		//res: uint256.Int{0xEEB9, 0x00, 0x00, 0x00}, // int
		res:    uint256.Int{0xEEBB, 0x00, 0x00, 0x00},
		status: RUNNING,
	},
	{ // different results for int and uint
		name: "opAddMod: (x + ^x) mod y // result? int x uint",
		op:   opAddMod,
		data: []uint256.Int{
			{0x00, 0x00, 0x00, 0x01},
			{0xAAAAAAAAAAAAAAAA, 0xAAAAAAAAAAAAAAAA, 0xAAAAAAAAAAAAAAAA, 0xAAAAAAAAAAAAAAAA},
			{0x5555555555555555, 0x5555555555555555, 0x5555555555555555, 0x5555555555555555}},
		//res: uint256.Int{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF}, // int
		res:    uint256.Int{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0x00},
		status: RUNNING,
	},
	{ // different results for int and uint
		name: "opAddMod: (x + ^x) mod 0xA..A // result? int x uint",
		op:   opAddMod,
		data: []uint256.Int{
			{0xAAAA, 0x00, 0x00, 0x00},
			{0x5555555555555555, 0x5555555555555555, 0x5555555555555555, 0x5555555555555555},
			{0xAAAAAAAAAAAAAAAA, 0xAAAAAAAAAAAAAAAA, 0xAAAAAAAAAAAAAAAA, 0xAAAAAAAAAAAAAAAA}},
		//res: uint256.Int{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF}, // int
		res:    uint256.Int{0x5555, 0x00, 0x00, 0x00},
		status: RUNNING,
	},
	{
		name: "opAddMod: (1 + x) mod y",
		op:   opAddMod,
		data: []uint256.Int{
			{0x123456789ABCDEF0, 0x123456789ABCDEF0, 0x123456789ABCDEF0, 0x123456789ABCDEF0},
			{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0x00},
			{0x01, 0x00, 0x00, 0x00}},
		res:    uint256.Int{0x00, 0x00, 0x00, 0x01},
		status: RUNNING,
	},
	{
		name: "opAddMod: various 8*bytes",
		op:   opAddMod,
		data: []uint256.Int{
			{0x00, 0x01, 0x00, 0x00},
			{0xFFFFFFFFFFFFFFFF, 0x00, 0xFFFFFFFFFFFFFFFF, 0x00},
			{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0x00, 0x00}},
		res:    uint256.Int{0xFFFFFFFFFFFFFFFE, 0x00, 0x00, 0x00},
		status: RUNNING,
	},
	{ // different results for int and uint
		name: "opAddMod: first and last bit // result? int x uint",
		op:   opAddMod,
		data: []uint256.Int{
			{0xFFFFFFFFFFFFFFFC, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF},
			{0x01, 0x00, 0x00, 0x8000000000000000},
			{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF}},
		//res: uint256.Int{0x00, 0x00, 0x00, 0x00}, // int
		res:    uint256.Int{0x04, 0x00, 0x00, 0x8000000000000000},
		status: RUNNING,
	},
	{
		name: "opAddMod: (0x00FF + 0x1234) mod 0xFF00",
		op:   opAddMod,
		data: []uint256.Int{
			{0xFF00, 0x00, 0x00, 0x00},
			{0x1234, 0x00, 0x00, 0x00},
			{0x00FF, 0x00, 0x00, 0x00}},
		res:    uint256.Int{0x1333, 0x00, 0x00, 0x00},
		status: RUNNING,
	},
	{
		name: "opAddMod: mul in the upper 16 bytes",
		op:   opAddMod,
		data: []uint256.Int{
			{0x07, 0x00, 0x00, 0x00},
			{0x00, 0x00, 0x1234, 0x00},
			{0x00, 0x00, 0x1234, 0xFFFF}},
		res:    uint256.Int{0x06, 0x00, 0x00, 0x00},
		status: RUNNING,
	},
	{
		name: "opAddMod: last 2 bytes of each 8",
		op:   opAddMod,
		data: []uint256.Int{
			{0x000000001221CC01, 0x00, 0x00, 0x00},
			{0xFF00, 0x0FF0, 0x00FF, 0xF0F0},
			{0x1234, 0xABCD, 0x5678, 0xEF90}},
		res:    uint256.Int{0x090F5D93, 0x00, 0x00, 0x00},
		status: RUNNING,
	},
	{ // different results for int and uint
		name: "opAddMod: (-1 + 0x1234) mod 255 // result? int x uint",
		op:   opAddMod,
		data: []uint256.Int{
			{0xFF, 0x00, 0x00, 0x00},
			{0x1234, 0x00, 0x00, 0x00},
			{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF}},
		//res: uint256.Int{0x45, 0x00, 0x00, 0x00}, // int
		res:    uint256.Int{0x46, 0x00, 0x00, 0x00},
		status: RUNNING,
	},
	{ // different results for int and uint
		name: "opAddMod: (-10 + 0x1234) mod 255 // result? int x uint",
		op:   opAddMod,
		data: []uint256.Int{
			{0xFF, 0x00, 0x00, 0x00},
			{0x1234, 0x00, 0x00, 0x00},
			{0xFFFFFFFFFFFFFFF6, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF}},
		//res: uint256.Int{0x3C, 0x00, 0x00, 0x00}, // int
		res:    uint256.Int{0x3D, 0x00, 0x00, 0x00},
		status: RUNNING,
	},
	{ // different results for int and uint
		name: "opAddMod: (1 + -1) mod 0x1234 // result? int x uint",
		op:   opAddMod,
		data: []uint256.Int{
			{0x1234, 0x00, 0x00, 0x00},
			{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF},
			{0x01, 0x00, 0x00, 0x00}},
		//res: uint256.Int{0x00, 0x00, 0x00, 0x00}, // int
		res:    uint256.Int{0x04C0, 0x00, 0x00, 0x00},
		status: RUNNING,
	},
	{ // different results for int and uint
		name: "opAddMod: (1 + -10) mod 0x1234 // result? int x uint",
		op:   opAddMod,
		data: []uint256.Int{
			{0x1234, 0x00, 0x00, 0x00},
			{0xFFFFFFFFFFFFFFF6, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF},
			{0x01, 0x00, 0x00, 0x00}},
		//res: uint256.Int{0xFFFFFFFFFFFFFFF7, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF}, // int
		res:    uint256.Int{0x04B7, 0x00, 0x00, 0x00},
		status: RUNNING,
	},
	{ // different results for int and uint
		name: "opAddMod: (0 + (-1)) mod 2 // result? int -1, uint 1",
		op:   opAddMod,
		data: []uint256.Int{
			{0x02, 0x00, 0x00, 0x00},
			{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF},
			{0x00, 0x00, 0x00, 0x00}},
		//res: uint256.Int{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF}, // int
		res:    uint256.Int{0x01, 0x00, 0x00, 0x00},
		status: RUNNING,
	},
	{
		name: "opAddMod: (0 + 1) mod -2",
		op:   opAddMod,
		data: []uint256.Int{
			{0xFFFFFFFFFFFFFFFE, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF},
			{0x01, 0x00, 0x00, 0x00},
			{0x00, 0x00, 0x00, 0x00}},
		res:    uint256.Int{0x01, 0x00, 0x00, 0x00},
		status: RUNNING,
	},
	{
		name: "opAddMod: ((-1) + (-1)) mod 2",
		op:   opAddMod,
		data: []uint256.Int{
			{0x02, 0x00, 0x00, 0x00},
			{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF},
			{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF}},
		res:    uint256.Int{0x00, 0x00, 0x00, 0x00},
		status: RUNNING,
	},
	{ // different results for int and uint
		name: "opAddMod: ((-1) + (-1)) mod -2 // result? int 0, uint 2",
		op:   opAddMod,
		data: []uint256.Int{
			{0xFFFFFFFFFFFFFFFE, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF},
			{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF},
			{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF}},
		//res: uint256.Int{0x00, 0x00, 0x00, 0x00}, // int
		res:    uint256.Int{0x02, 0x00, 0x00, 0x00},
		status: RUNNING,
	},
	{ // different results for int and uint
		name: "opAddMod: ((-1) + (-1)) mod 3 // result? int -2, uint 0",
		op:   opAddMod,
		data: []uint256.Int{
			{0x03, 0x00, 0x00, 0x00},
			{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF},
			{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF}},
		//res: uint256.Int{0xFFFFFFFFFFFFFFFE, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF}, // int
		res:    uint256.Int{0x00, 0x00, 0x00, 0x00},
		status: RUNNING,
	},

	// operation SMod
	// a % b (int), data: {b, a}
	{
		name: "opSMod: 0 mod 0",
		op:   opSMod,
		data: []uint256.Int{
			{0, 0, 0, 0},
			{0, 0, 0, 0}},
		res:    uint256.Int{0, 0, 0, 0},
		status: RUNNING,
	},
	{
		name: "opSMod: -1 mod 0",
		op:   opSMod,
		data: []uint256.Int{
			{0x00, 0x00, 0x00, 0x00},
			{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF}},
		res:    uint256.Int{0x00, 0x00, 0x00, 0x00},
		status: RUNNING,
	},
	{
		name: "opSMod: 0 mod -1",
		op:   opSMod,
		data: []uint256.Int{
			{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF},
			{0x00, 0x00, 0x00, 0x00}},
		res:    uint256.Int{0x00, 0x00, 0x00, 0x00},
		status: RUNNING,
	},
	{
		name: "opSMod: 1 mod -1",
		op:   opSMod,
		data: []uint256.Int{
			{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF},
			{0x01, 0x00, 0x00, 0x00}},
		res:    uint256.Int{0x00, 0x00, 0x00, 0x00},
		status: RUNNING,
	},
	{
		name: "opSMod: -1 mod 1",
		op:   opSMod,
		data: []uint256.Int{
			{0x01, 0x00, 0x00, 0x00},
			{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF}},
		res:    uint256.Int{0x00, 0x00, 0x00, 0x00},
		status: RUNNING,
	},
	{
		name: "opSMod: -1 mod -1",
		op:   opSMod,
		data: []uint256.Int{
			{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF},
			{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF}},
		res:    uint256.Int{0x00, 0x00, 0x00, 0x00},
		status: RUNNING,
	},
	{
		name: "opSMod: min mod -1",
		op:   opSMod,
		data: []uint256.Int{
			{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF},
			{0x00, 0x00, 0x00, 0x8000000000000000}},
		res:    uint256.Int{0x00, 0x00, 0x00, 0x00},
		status: RUNNING,
	},
	{
		name: "opSMod: -1 mod min",
		op:   opSMod,
		data: []uint256.Int{
			{0x00, 0x00, 0x00, 0x8000000000000000},
			{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF}},
		res:    uint256.Int{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF},
		status: RUNNING,
	},
	{
		name: "opSMod: -1 mod x",
		op:   opSMod,
		data: []uint256.Int{
			{0x123456789ABCDEF0, 0x23456789ABCDEF01, 0x3456789ABCDEF012, 0x456789ABCDEF0123},
			{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF}},
		res:    uint256.Int{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF},
		status: RUNNING,
	},
	{
		name: "opSMod: x mod -1",
		op:   opSMod,
		data: []uint256.Int{
			{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF},
			{0x123456789ABCDEF0, 0x23456789ABCDEF01, 0x3456789ABCDEF012, 0x456789ABCDEF0123}},
		res:    uint256.Int{0x00, 0x00, 0x00, 0x00},
		status: RUNNING,
	},
	{
		name: "opSMod: 0x5..5 mod 0xA..A (x mod ^x)",
		op:   opSMod,
		data: []uint256.Int{
			{0xAAAAAAAAAAAAAAAA, 0xAAAAAAAAAAAAAAAA, 0xAAAAAAAAAAAAAAAA, 0xAAAAAAAAAAAAAAAA},
			{0x5555555555555555, 0x5555555555555555, 0x5555555555555555, 0x5555555555555555}},
		res:    uint256.Int{0x5555555555555555, 0x5555555555555555, 0x5555555555555555, 0x5555555555555555},
		status: RUNNING,
	},
	{
		name: "opSMod: 0xA..A mod 0x5..5 (x mod ^x)",
		op:   opSMod,
		data: []uint256.Int{
			{0x5555555555555555, 0x5555555555555555, 0x5555555555555555, 0x5555555555555555},
			{0xAAAAAAAAAAAAAAAA, 0xAAAAAAAAAAAAAAAA, 0xAAAAAAAAAAAAAAAA, 0xAAAAAAAAAAAAAAAA}},
		res:    uint256.Int{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF},
		status: RUNNING,
	},
	{
		name: "opSMod: 0x01 mod 0x0F..F",
		op:   opSMod,
		data: []uint256.Int{
			{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0x00},
			{0x01, 0x00, 0x00, 0x00}},
		res:    uint256.Int{0x01, 0x00, 0x00, 0x00},
		status: RUNNING,
	},
	{
		name: "opSMod: various 8*bytes",
		op:   opSMod,
		data: []uint256.Int{
			{0xFFFFFFFFFFFFFFFF, 0x00, 0xFFFFFFFFFFFFFFFF, 0x00},
			{0x01, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF}},
		res:    uint256.Int{0x01, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF},
		status: RUNNING,
	},
	{
		name: "opSMod: first and last bit",
		op:   opSMod,
		data: []uint256.Int{
			{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0x7FFFFFFFFFFFFFFF},
			{0x01, 0x00, 0x00, 0x8000000000000000}},
		res:    uint256.Int{0x00, 0x00, 0x00, 0x00},
		status: RUNNING,
	},
	{
		name: "opSMod: last 8 bytes",
		op:   opSMod,
		data: []uint256.Int{
			{0x1234, 0x00, 0x00, 0x00},
			{0x1221CC, 0x00, 0x00, 0x00}},
		res:    uint256.Int{0x00, 0x00, 0x00, 0x00},
		status: RUNNING,
	},
	{
		name: "opSMod: modulo in the upper 16 bytes",
		op:   opSMod,
		data: []uint256.Int{
			{0x00, 0x00, 0x1234, 0x00},
			{0x00, 0x00, 0x1234, 0xFFFF}},
		res:    uint256.Int{0x00, 0x00, 0xF0, 0x00},
		status: RUNNING,
	},
	{
		name: "opSMod: each 8*bytes",
		op:   opSMod,
		data: []uint256.Int{
			{0xFF00, 0x0FF0, 0x00FF, 0xF0F0},
			{0x000000001221CC00, 0x00000000AC434FC0, 0x0000000060E5BCFC, 0x0000000105CF7A73}},
		res:    uint256.Int{0xFFFFFFFEFD0AF900, 0x9AF1E28F, 0x5FD0A629, 0x8043},
		status: RUNNING,
	},
	{
		name: "opSMod: x mod 0x1234",
		op:   opSMod,
		data: []uint256.Int{
			{0x1234, 0x00, 0x00, 0x00},
			{0xFFFFFFFFFFFFEDCC, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF}},
		res:    uint256.Int{0x00, 0x00, 0x00, 0x00},
		status: RUNNING,
	},
	{
		name: "opSMod: y mod 0x1234",
		op:   opSMod,
		data: []uint256.Int{
			{0x1234, 0x00, 0x00, 0x00},
			{0xFFFFFFFFFFFF49FF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF}},
		res:    uint256.Int{0xFFFFFFFFFFFFEDD3, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF},
		status: RUNNING,
	},
	{
		name: "opSMod: -1 mod 0x1234",
		op:   opSMod,
		data: []uint256.Int{
			{0x1234, 0x00, 0x00, 0x00},
			{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF}},
		res:    uint256.Int{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF},
		status: RUNNING,
	},
	{
		name: "opSMod: -1 mod 2 = -1",
		op:   opSMod,
		data: []uint256.Int{
			{0x02, 0x00, 0x00, 0x00},
			{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF}},
		res:    uint256.Int{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF},
		status: RUNNING,
	},
	{
		name: "opSMod: -1 mod -2 = -1",
		op:   opSMod,
		data: []uint256.Int{
			{0xFFFFFFFFFFFFFFFE, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF},
			{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF}},
		res:    uint256.Int{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF},
		status: RUNNING,
	},

	// operation SignExtend
	// singextend(x, b), data: {x, b}
	{
		name: "opSignExtend: (0, 0)",
		op:   opSignExtend,
		data: []uint256.Int{
			{0, 0, 0, 0},
			{0, 0, 0, 0}},
		res:    uint256.Int{0, 0, 0, 0},
		status: RUNNING,
	},
	{
		name: "opSignExtend: (2, -1)",
		op:   opSignExtend,
		data: []uint256.Int{
			{0x02, 0x00, 0x00, 0x00},
			{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF}},
		res:    uint256.Int{0x02, 0x00, 0x00, 0x00},
		status: RUNNING,
	},
	{
		name: "opSignExtend: (-1, 2)",
		op:   opSignExtend,
		data: []uint256.Int{
			{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF},
			{0x02, 0x00, 0x00, 0x00}},
		res:    uint256.Int{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF},
		status: RUNNING,
	},
	{
		name: "opSignExtend: (0xFFFF, 0)",
		op:   opSignExtend,
		data: []uint256.Int{
			{0xFFFF, 0x00, 0x00, 0x00},
			{0x00, 0x00, 0x00, 0x00}},
		res:    uint256.Int{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF},
		status: RUNNING,
	},
	{
		name: "opSignExtend: (0xFFFF, 1)",
		op:   opSignExtend,
		data: []uint256.Int{
			{0xFFFF, 0x00, 0x00, 0x00},
			{0x01, 0x00, 0x00, 0x00}},
		res:    uint256.Int{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF},
		status: RUNNING,
	},
	{
		name: "opSignExtend: (0xFFFF, 2)",
		op:   opSignExtend,
		data: []uint256.Int{
			{0xFFFF, 0x00, 0x00, 0x00},
			{0x02, 0x00, 0x00, 0x00}},
		res:    uint256.Int{0xFFFF, 0x00, 0x00, 0x00},
		status: RUNNING,
	},
	{
		name: "opSignExtend: (0xFFFF, 3)",
		op:   opSignExtend,
		data: []uint256.Int{
			{0xFFFF, 0x00, 0x00, 0x00},
			{0x03, 0x00, 0x00, 0x00}},
		res:    uint256.Int{0xFFFF, 0x00, 0x00, 0x00},
		status: RUNNING,
	},
	{
		name: "opSignExtend: (-238, 2)",
		op:   opSignExtend,
		data: []uint256.Int{
			{0xFFFFFFFFFFFFFF12, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF},
			{0x02, 0x00, 0x00, 0x00}},
		res:    uint256.Int{0xFFFFFFFFFFFFFF12, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF},
		status: RUNNING,
	},
	{
		name: "opSignExtend: (0xFF12, 1)",
		op:   opSignExtend,
		data: []uint256.Int{
			{0xFF12, 0x00, 0x00, 0x00},
			{0x01, 0x00, 0x00, 0x00}},
		res:    uint256.Int{0xFFFFFFFFFFFFFF12, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF},
		status: RUNNING,
	},
	{
		name: "opSignExtend: (x, 31)",
		op:   opSignExtend,
		data: []uint256.Int{
			{0x1234, 0x00, 0x00, 0x123456789ABCDEF0},
			{31, 0, 0, 0}},
		res:    uint256.Int{0x1234, 0x00, 0x00, 0x123456789ABCDEF0},
		status: RUNNING,
	},
	{
		name: "opSignExtend: (x, 30)",
		op:   opSignExtend,
		data: []uint256.Int{
			{0x1234, 0x00, 0x00, 0x123456789ABCDEF0},
			{30, 0, 0, 0}},
		res:    uint256.Int{0x1234, 0x00, 0x00, 0x003456789ABCDEF0},
		status: RUNNING,
	},
	{
		name: "opSignExtend: (0x1234, 0)",
		op:   opSignExtend,
		data: []uint256.Int{
			{0x1234, 0x00, 0x00, 0x00},
			{0x00, 0x00, 0x00, 0x00}},
		res:    uint256.Int{0x34, 0x00, 0x00, 0x00},
		status: RUNNING,
	},
	{
		name: "opSignExtend: (0x0A00, 0)",
		op:   opSignExtend,
		data: []uint256.Int{
			{0x0A00, 0x00, 0x00, 0x00},
			{0x00, 0x00, 0x00, 0x00}},
		res:    uint256.Int{0x00, 0x00, 0x00, 0x00},
		status: RUNNING,
	},
	{
		name: "opSignExtend: (0x8100, 1)",
		op:   opSignExtend,
		data: []uint256.Int{
			{0x8100, 0x00, 0x00, 0x00},
			{0x01, 0x00, 0x00, 0x00}},
		res:    uint256.Int{0xFFFFFFFFFFFF8100, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF},
		status: RUNNING,
	},

	// operation Exp
	// a ** b (uint), data: {b, a}
	{
		name: "opExp: 0 ** 0",
		op:   opExp,
		data: []uint256.Int{
			{0, 0, 0, 0},
			{0, 0, 0, 0}},
		res:    uint256.Int{1, 0, 0, 0},
		status: RUNNING,
		gas:    0,
	},
	{
		name: "opExp: 2 ** 2",
		op:   opExp,
		data: []uint256.Int{
			{2, 0, 0, 0},
			{2, 0, 0, 0}},
		res:    uint256.Int{4, 0, 0, 0},
		status: RUNNING,
		gas:    50,
	},
	{
		name: "opExp: 0x1234 ** 0",
		op:   opExp,
		data: []uint256.Int{
			{0x00, 0x00, 0x00, 0x00},
			{0x1234, 0x00, 0x00, 0x00}},
		res:    uint256.Int{0x01, 0x00, 0x00, 0x00},
		status: RUNNING,
		gas:    0,
	},
	{
		name: "opExp: 0x1234 ** 1",
		op:   opExp,
		data: []uint256.Int{
			{0x0001, 0x00, 0x00, 0x00},
			{0x1234, 0x00, 0x00, 0x00}},
		res:    uint256.Int{0x1234, 0x00, 0x00, 0x00},
		status: RUNNING,
		gas:    50,
	},
	{
		name: "opExp: 0x1234 ** 2",
		op:   opExp,
		data: []uint256.Int{
			{0x0002, 0x00, 0x00, 0x00},
			{0x1234, 0x00, 0x00, 0x00}},
		res:    uint256.Int{0x014B5A90, 0x00, 0x00, 0x00},
		status: RUNNING,
		gas:    50,
	},
	{
		name: "opExp: 0x1234 ** 3",
		op:   opExp,
		data: []uint256.Int{
			{0x0003, 0x00, 0x00, 0x00},
			{0x1234, 0x00, 0x00, 0x00}},
		res:    uint256.Int{0x00178FAC8540, 0x00, 0x00, 0x00},
		status: RUNNING,
		gas:    50,
	},
	{
		name: "opExp: 0x1234 ** 4",
		op:   opExp,
		data: []uint256.Int{
			{0x0004, 0x00, 0x00, 0x00},
			{0x1234, 0x00, 0x00, 0x00}},
		res:    uint256.Int{0x0001ACE350699100, 0x00, 0x00, 0x00},
		status: RUNNING,
		gas:    50,
	},
	{
		name: "opExp: 0x1234 ** 5",
		op:   opExp,
		data: []uint256.Int{
			{0x0005, 0x00, 0x00, 0x00},
			{0x1234, 0x00, 0x00, 0x00}},
		res:    uint256.Int{0x1E7F19D3C1A37400, 0x00, 0x00, 0x00},
		status: RUNNING,
		gas:    50,
	},
	{
		name: "opExp: 0x1234 ** 6",
		op:   opExp,
		data: []uint256.Int{
			{0x0006, 0x00, 0x00, 0x00},
			{0x1234, 0x00, 0x00, 0x00}},
		res:    uint256.Int{0x21A222A0D35B9000, 0x022B, 0x00, 0x00},
		status: RUNNING,
		gas:    50,
	},
	{
		name: "opExp: 1 ** 10",
		op:   opExp,
		data: []uint256.Int{
			{0x0A00, 0x00, 0x00, 0x00},
			{0x0001, 0x00, 0x00, 0x00}},
		res:    uint256.Int{0x0001, 0x00, 0x00, 0x00},
		status: RUNNING,
		gas:    100,
	},
	{
		name: "opExp: -1 ** 2",
		op:   opExp,
		data: []uint256.Int{
			{0x02, 0x00, 0x00, 0x00},
			{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF}},
		res:    uint256.Int{0x01, 0x00, 0x00, 0x00},
		status: RUNNING,
		gas:    50,
	},
	{
		name: "opExp: 2 ** -1",
		op:   opExp,
		data: []uint256.Int{
			{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF},
			{0x02, 0x00, 0x00, 0x00}},
		res:    uint256.Int{0x00, 0x00, 0x00, 0x00},
		status: RUNNING,
		gas:    1600,
	},
	{
		name: "opExp: 2 ** 256",
		op:   opExp,
		data: []uint256.Int{
			{0x0100, 0x00, 0x00, 0x00},
			{0x02, 0x00, 0x00, 0x00}},
		res:    uint256.Int{0x00, 0x00, 0x00, 0x00},
		status: RUNNING,
		gas:    100,
	},
	{
		name: "opExp: 2 ** 255",
		op:   opExp,
		data: []uint256.Int{
			{0xFF, 0x00, 0x00, 0x00},
			{0x02, 0x00, 0x00, 0x00}},
		res:    uint256.Int{0x00, 0x00, 0x00, 0x8000000000000000},
		status: RUNNING,
		gas:    50,
	},
	{
		name: "opExp: out of gas",
		op:   opExp,
		data: []uint256.Int{
			{0x02, 0x00, 0x00, 0x00},
			{0x0200, 0x00, 0x00, 0x00}},
		res:    uint256.Int{0x02, 0x00, 0x00, 0x00},
		status: OUT_OF_GAS,
		gas:    0,
	},
}

// comparison operations (IsZero, Eq, Lt, Gt, Slt, Sgt)

type tTestDataCompOp struct {
	name   string         // test description
	op     func(*context) // tested operation
	data   []uint256.Int  // input data (in reverse order)
	res    bool           // expected result
	status Status         // expected status
}

var testDataComparsionOp = []tTestDataCompOp{

	// operation Iszero
	{
		name:   "opIszero: 0x1234",
		op:     opIszero,
		data:   []uint256.Int{{0x1234, 0x00, 0x00, 0x00}},
		res:    false,
		status: RUNNING,
	},
	{
		name:   "opIszero: 0",
		op:     opIszero,
		data:   []uint256.Int{{0x00, 0x00, 0x00, 0x00}},
		res:    true,
		status: RUNNING,
	},
	{
		name:   "opIszero: 0xFFFF",
		op:     opIszero,
		data:   []uint256.Int{{0xFFFF, 0x00, 0x00, 0x00}},
		res:    false,
		status: RUNNING,
	},
	{
		name:   "opIszero: 0xFF00",
		op:     opIszero,
		data:   []uint256.Int{{0xFF00, 0x00, 0x00, 0x00}},
		res:    false,
		status: RUNNING,
	},
	{
		name:   "opIszero: -16",
		op:     opIszero,
		data:   []uint256.Int{{0xFFFFFFFFFFFFFFF0, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF}},
		res:    false,
		status: RUNNING,
	},
	{
		name:   "opIszero: max",
		op:     opIszero,
		data:   []uint256.Int{{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0x7FFFFFFFFFFFFFFF}},
		res:    false,
		status: RUNNING,
	},
	{
		name:   "opIszero: -1",
		op:     opIszero,
		data:   []uint256.Int{{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF}},
		res:    false,
		status: RUNNING,
	},
	{
		name:   "opIszero: min",
		op:     opIszero,
		data:   []uint256.Int{{0x00, 0x00, 0x00, 0x8000000000000000}},
		res:    false,
		status: RUNNING,
	},
	{
		name:   "opIszero: 1",
		op:     opIszero,
		data:   []uint256.Int{{0x01, 0x00, 0x00, 0x00}},
		res:    false,
		status: RUNNING,
	},

	// operation Eq
	{
		name: "opEq: 0x00 == 0x1234",
		op:   opEq,
		data: []uint256.Int{
			{0x1234, 0x00, 0x00, 0x00},
			{0x0000, 0x00, 0x00, 0x00}},
		res:    false,
		status: RUNNING,
	},
	{
		name: "opEq: 0xFFFF == 0x1234",
		op:   opEq,
		data: []uint256.Int{
			{0x1234, 0x00, 0x00, 0x00},
			{0xFFFF, 0x00, 0x00, 0x00}},
		res:    false,
		status: RUNNING,
	},
	{
		name: "opEq: 0xFF00 == 0x1234",
		op:   opEq,
		data: []uint256.Int{
			{0x1234, 0x00, 0x00, 0x00},
			{0xFF00, 0x00, 0x00, 0x00}},
		res:    false,
		status: RUNNING,
	},
	{
		name: "opEq: 0x1234 == 0x00",
		op:   opEq,
		data: []uint256.Int{
			{0x0000, 0x00, 0x00, 0x00},
			{0x1234, 0x00, 0x00, 0x00}},
		res:    false,
		status: RUNNING,
	},
	{
		name: "opEq: 0x1234 == 0xFFFF",
		op:   opEq,
		data: []uint256.Int{
			{0xFFFF, 0x00, 0x00, 0x00},
			{0x1234, 0x00, 0x00, 0x00}},
		res:    false,
		status: RUNNING,
	},
	{
		name: "opEq: 0x1234 == 0xFF",
		op:   opEq,
		data: []uint256.Int{
			{0x00FF, 0x00, 0x00, 0x00},
			{0x1234, 0x00, 0x00, 0x00}},
		res:    false,
		status: RUNNING,
	},
	{
		name: "opEq: 0x1234 == 0x1234",
		op:   opEq,
		data: []uint256.Int{
			{0x1234, 0x00, 0x00, 0x00},
			{0x1234, 0x00, 0x00, 0x00}},
		res:    true,
		status: RUNNING,
	},
	{
		name: "opEq: 0 == 0",
		op:   opEq,
		data: []uint256.Int{
			{0x00, 0x00, 0x00, 0x00},
			{0x00, 0x00, 0x00, 0x00}},
		res:    true,
		status: RUNNING,
	},
	{
		name: "opEq: -1 == 0",
		op:   opEq,
		data: []uint256.Int{
			{0x00, 0x00, 0x00, 0x00},
			{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF}},
		res:    false,
		status: RUNNING,
	},
	{
		name: "opEq: -1 == -16",
		op:   opEq,
		data: []uint256.Int{
			{0xFFFFFFFFFFFFFFF0, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF},
			{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF}},
		res:    false,
		status: RUNNING,
	},
	{
		name: "opEq: -1 == max",
		op:   opEq,
		data: []uint256.Int{
			{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0x7FFFFFFFFFFFFFFF},
			{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF}},
		res:    false,
		status: RUNNING,
	},
	{
		name: "opEq: min == 1",
		op:   opEq,
		data: []uint256.Int{
			{0x01, 0x00, 0x00, 0x00},
			{0x00, 0x00, 0x00, 0x8000000000000000}},
		res:    false,
		status: RUNNING,
	},
	{
		name: "opEq: -1 == -1",
		op:   opEq,
		data: []uint256.Int{
			{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF},
			{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF}},
		res:    true,
		status: RUNNING,
	},
	{
		name: "opEq: -1 == 1",
		op:   opEq,
		data: []uint256.Int{
			{0x01, 0x00, 0x00, 0x00},
			{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF}},
		res:    false,
		status: RUNNING,
	},
	{
		name: "opEq: 1 == -1",
		op:   opEq,
		data: []uint256.Int{
			{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF},
			{0x01, 0x00, 0x00, 0x00}},
		res:    false,
		status: RUNNING,
	},
	{
		name: "opEq: max == max",
		op:   opEq,
		data: []uint256.Int{
			{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0x7FFFFFFFFFFFFFFF},
			{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0x7FFFFFFFFFFFFFFF}},
		res:    true,
		status: RUNNING,
	},
	{
		name: "opEq: min == min",
		op:   opEq,
		data: []uint256.Int{
			{0x00, 0x00, 0x00, 0x8000000000000000},
			{0x00, 0x00, 0x00, 0x8000000000000000}},
		res:    true,
		status: RUNNING,
	},

	// operation Lt
	{
		name: "opLt: 0x00 < 0x1234",
		op:   opLt,
		data: []uint256.Int{
			{0x1234, 0x00, 0x00, 0x00},
			{0x0000, 0x00, 0x00, 0x00}},
		res:    true,
		status: RUNNING,
	},
	{
		name: "opLt: 0xFFFF < 0x1234",
		op:   opLt,
		data: []uint256.Int{
			{0x1234, 0x00, 0x00, 0x00},
			{0xFFFF, 0x00, 0x00, 0x00}},
		res:    false,
		status: RUNNING,
	},
	{
		name: "opLt: 0xFF00 < 0x1234",
		op:   opLt,
		data: []uint256.Int{
			{0x1234, 0x00, 0x00, 0x00},
			{0xFF00, 0x00, 0x00, 0x00}},
		res:    false,
		status: RUNNING,
	},
	{
		name: "opLt: 0x1234 < 0x00",
		op:   opLt,
		data: []uint256.Int{
			{0x0000, 0x00, 0x00, 0x00},
			{0x1234, 0x00, 0x00, 0x00}},
		res:    false,
		status: RUNNING,
	},
	{
		name: "opLt: 0x1234 < 0xFFFF",
		op:   opLt,
		data: []uint256.Int{
			{0xFFFF, 0x00, 0x00, 0x00},
			{0x1234, 0x00, 0x00, 0x00}},
		res:    true,
		status: RUNNING,
	},
	{
		name: "opLt: 0x1234 < 0xFF00",
		op:   opLt,
		data: []uint256.Int{
			{0xFF00, 0x00, 0x00, 0x00},
			{0x1234, 0x00, 0x00, 0x00}},
		res:    true,
		status: RUNNING,
	},
	{
		name: "opLt: 0x1234 < 0x1234",
		op:   opLt,
		data: []uint256.Int{
			{0x1234, 0x00, 0x00, 0x00},
			{0x1234, 0x00, 0x00, 0x00}},
		res:    false,
		status: RUNNING,
	},
	{
		name: "opLt: 0 < 0",
		op:   opLt,
		data: []uint256.Int{
			{0x00, 0x00, 0x00, 0x00},
			{0x00, 0x00, 0x00, 0x00}},
		res:    false,
		status: RUNNING,
	},
	{
		name: "opLt: max < 0",
		op:   opLt,
		data: []uint256.Int{
			{0x00, 0x00, 0x00, 0x00},
			{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF}},
		res:    false,
		status: RUNNING,
	},
	{
		name: "opLt: max < (max-15)",
		op:   opLt,
		data: []uint256.Int{
			{0xFFFFFFFFFFFFFFF0, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF},
			{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF}},
		res:    false,
		status: RUNNING,
	},
	{
		name: "opLt: max < (max/2)",
		op:   opLt,
		data: []uint256.Int{
			{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0x7FFFFFFFFFFFFFFF},
			{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF}},
		res:    false,
		status: RUNNING,
	},
	{
		name: "opLt: (max/2+1) < 1",
		op:   opLt,
		data: []uint256.Int{
			{0x01, 0x00, 0x00, 0x00},
			{0x00, 0x00, 0x00, 0x8000000000000000}},
		res:    false,
		status: RUNNING,
	},
	{
		name: "opLt: max < max",
		op:   opLt,
		data: []uint256.Int{
			{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF},
			{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF}},
		res:    false,
		status: RUNNING,
	},
	{
		name: "opLt: max < 1",
		op:   opLt,
		data: []uint256.Int{
			{0x01, 0x00, 0x00, 0x00},
			{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF}},
		res:    false,
		status: RUNNING,
	},
	{
		name: "opLt: 1 < max",
		op:   opLt,
		data: []uint256.Int{
			{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF},
			{0x01, 0x00, 0x00, 0x00}},
		res:    true,
		status: RUNNING,
	},
	{
		name: "opLt: max/2 < max/2",
		op:   opLt,
		data: []uint256.Int{
			{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0x7FFFFFFFFFFFFFFF},
			{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0x7FFFFFFFFFFFFFFF}},
		res:    false,
		status: RUNNING,
	},
	{
		name: "opLt: (max/2+1) < (max/2+1)",
		op:   opLt,
		data: []uint256.Int{
			{0x00, 0x00, 0x00, 0x8000000000000000},
			{0x00, 0x00, 0x00, 0x8000000000000000}},
		res:    false,
		status: RUNNING,
	},

	// operation Gt
	{
		name: "opGt: 0x00 > 0x1234",
		op:   opGt,
		data: []uint256.Int{
			{0x1234, 0x00, 0x00, 0x00},
			{0x0000, 0x00, 0x00, 0x00}},
		res:    false,
		status: RUNNING,
	},
	{
		name: "opGt: 0xFFFF > 0x1234",
		op:   opGt,
		data: []uint256.Int{
			{0x1234, 0x00, 0x00, 0x00},
			{0xFFFF, 0x00, 0x00, 0x00}},
		res:    true,
		status: RUNNING,
	},
	{
		name: "opGt: 0xFF00 > 0x1234",
		op:   opGt,
		data: []uint256.Int{
			{0x1234, 0x00, 0x00, 0x00},
			{0xFF00, 0x00, 0x00, 0x00}},
		res:    true,
		status: RUNNING,
	},
	{
		name: "opGt: 0x1234 > 0x00",
		op:   opGt,
		data: []uint256.Int{
			{0x0000, 0x00, 0x00, 0x00},
			{0x1234, 0x00, 0x00, 0x00}},
		res:    true,
		status: RUNNING,
	},
	{
		name: "opGt: 0x1234 > 0xFFFF",
		op:   opGt,
		data: []uint256.Int{
			{0xFFFF, 0x00, 0x00, 0x00},
			{0x1234, 0x00, 0x00, 0x00}},
		res:    false,
		status: RUNNING,
	},
	{
		name: "opGt: 0x1234 > 0xFF00",
		op:   opGt,
		data: []uint256.Int{
			{0xFF00, 0x00, 0x00, 0x00},
			{0x1234, 0x00, 0x00, 0x00}},
		res:    false,
		status: RUNNING,
	},
	{
		name: "opGt: 0xFFFF > 0xFFFF",
		op:   opGt,
		data: []uint256.Int{
			{0xFFFF, 0x00, 0x00, 0x00},
			{0xFFFF, 0x00, 0x00, 0x00}},
		res:    false,
		status: RUNNING,
	},
	{
		name: "opGt: 0 > 0",
		op:   opGt,
		data: []uint256.Int{
			{0x00, 0x00, 0x00, 0x00},
			{0x00, 0x00, 0x00, 0x00}},
		res:    false,
		status: RUNNING,
	},
	{
		name: "opGt: max > 0",
		op:   opGt,
		data: []uint256.Int{
			{0x00, 0x00, 0x00, 0x00},
			{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF}},
		res:    true,
		status: RUNNING,
	},
	{
		name: "opGt: max > (max-15)",
		op:   opGt,
		data: []uint256.Int{
			{0xFFFFFFFFFFFFFFF0, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF},
			{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF}},
		res:    true,
		status: RUNNING,
	},
	{
		name: "opGt: max > (max/2)",
		op:   opGt,
		data: []uint256.Int{
			{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0x7FFFFFFFFFFFFFFF},
			{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF}},
		res:    true,
		status: RUNNING,
	},
	{
		name: "opGt: (max/2+1) > 1",
		op:   opGt,
		data: []uint256.Int{
			{0x01, 0x00, 0x00, 0x00},
			{0x00, 0x00, 0x00, 0x8000000000000000}},
		res:    true,
		status: RUNNING,
	},
	{
		name: "opGt: max > max",
		op:   opGt,
		data: []uint256.Int{
			{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF},
			{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF}},
		res:    false,
		status: RUNNING,
	},
	{
		name: "opGt: max > 1",
		op:   opGt,
		data: []uint256.Int{
			{0x01, 0x00, 0x00, 0x00},
			{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF}},
		res:    true,
		status: RUNNING,
	},
	{
		name: "opGt: 1 > max",
		op:   opGt,
		data: []uint256.Int{
			{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF},
			{0x01, 0x00, 0x00, 0x00}},
		res:    false,
		status: RUNNING,
	},
	{
		name: "opGt: max/2 > max/2",
		op:   opGt,
		data: []uint256.Int{
			{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0x7FFFFFFFFFFFFFFF},
			{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0x7FFFFFFFFFFFFFFF}},
		res:    false,
		status: RUNNING,
	},
	{
		name: "opGt: (max/2+1) > (max/2+1)",
		op:   opGt,
		data: []uint256.Int{
			{0x00, 0x00, 0x00, 0x8000000000000000},
			{0x00, 0x00, 0x00, 0x8000000000000000}},
		res:    false,
		status: RUNNING,
	},

	// operation Slt
	{
		name: "opSlt: 0x00 < 0x1234",
		op:   opSlt,
		data: []uint256.Int{
			{0x1234, 0x00, 0x00, 0x00},
			{0x0000, 0x00, 0x00, 0x00}},
		res:    true,
		status: RUNNING,
	},
	{
		name: "opSlt: 0xFFFF < 0x1234",
		op:   opSlt,
		data: []uint256.Int{
			{0x1234, 0x00, 0x00, 0x00},
			{0xFFFF, 0x00, 0x00, 0x00}},
		res:    false,
		status: RUNNING,
	},
	{
		name: "opSlt: 0xFF00 < 0x1234",
		op:   opSlt,
		data: []uint256.Int{
			{0x1234, 0x00, 0x00, 0x00},
			{0xFF00, 0x00, 0x00, 0x00}},
		res:    false,
		status: RUNNING,
	},
	{
		name: "opSlt: 0x1234 < 0x00",
		op:   opSlt,
		data: []uint256.Int{
			{0x0000, 0x00, 0x00, 0x00},
			{0x1234, 0x00, 0x00, 0x00}},
		res:    false,
		status: RUNNING,
	},
	{
		name: "opSlt: 0x1234 < 0xFFFF",
		op:   opSlt,
		data: []uint256.Int{
			{0xFFFF, 0x00, 0x00, 0x00},
			{0x1234, 0x00, 0x00, 0x00}},
		res:    true,
		status: RUNNING,
	},
	{
		name: "opSlt: 0x1234 < 0xFF00",
		op:   opSlt,
		data: []uint256.Int{
			{0xFF00, 0x00, 0x00, 0x00},
			{0x1234, 0x00, 0x00, 0x00}},
		res:    true,
		status: RUNNING,
	},
	{
		name: "opSlt: 0x1234 < 0x1234",
		op:   opSlt,
		data: []uint256.Int{
			{0x1234, 0x00, 0x00, 0x00},
			{0x1234, 0x00, 0x00, 0x00}},
		res:    false,
		status: RUNNING,
	},
	{
		name: "opSlt: 0 < 0",
		op:   opSlt,
		data: []uint256.Int{
			{0x00, 0x00, 0x00, 0x00},
			{0x00, 0x00, 0x00, 0x00}},
		res:    false,
		status: RUNNING,
	},
	{
		name: "opSlt: -1 < 0",
		op:   opSlt,
		data: []uint256.Int{
			{0x00, 0x00, 0x00, 0x00},
			{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF}},
		res:    true,
		status: RUNNING,
	},
	{
		name: "opSlt: -1 < -16",
		op:   opSlt,
		data: []uint256.Int{
			{0xFFFFFFFFFFFFFFF0, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF},
			{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF}},
		res:    false,
		status: RUNNING,
	},
	{
		name: "opSlt: -1 < max",
		op:   opSlt,
		data: []uint256.Int{
			{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0x7FFFFFFFFFFFFFFF},
			{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF}},
		res:    true,
		status: RUNNING,
	},
	{
		name: "opSlt: min < 1",
		op:   opSlt,
		data: []uint256.Int{
			{0x01, 0x00, 0x00, 0x00},
			{0x00, 0x00, 0x00, 0x8000000000000000}},
		res:    true,
		status: RUNNING,
	},
	{
		name: "opSlt: -1 < -1",
		op:   opSlt,
		data: []uint256.Int{
			{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF},
			{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF}},
		res:    false,
		status: RUNNING,
	},
	{
		name: "opSlt: -1 < 1",
		op:   opSlt,
		data: []uint256.Int{
			{0x01, 0x00, 0x00, 0x00},
			{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF}},
		res:    true,
		status: RUNNING,
	},
	{
		name: "opSlt: 1 < -1",
		op:   opSlt,
		data: []uint256.Int{
			{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF},
			{0x01, 0x00, 0x00, 0x00}},
		res:    false,
		status: RUNNING,
	},
	{
		name: "opSlt: max < max",
		op:   opSlt,
		data: []uint256.Int{
			{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0x7FFFFFFFFFFFFFFF},
			{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0x7FFFFFFFFFFFFFFF}},
		res:    false,
		status: RUNNING,
	},
	{
		name: "opSlt: min < min",
		op:   opSlt,
		data: []uint256.Int{
			{0x00, 0x00, 0x00, 0x8000000000000000},
			{0x00, 0x00, 0x00, 0x8000000000000000}},
		res:    false,
		status: RUNNING,
	},

	// operation Sgt
	{
		name: "opSgt: 0x00 > 0x1234",
		op:   opSgt,
		data: []uint256.Int{
			{0x1234, 0x00, 0x00, 0x00},
			{0x0000, 0x00, 0x00, 0x00}},
		res:    false,
		status: RUNNING,
	},
	{
		name: "opSgt: 0xFFFF > 0x1234",
		op:   opSgt,
		data: []uint256.Int{
			{0x1234, 0x00, 0x00, 0x00},
			{0xFFFF, 0x00, 0x00, 0x00}},
		res:    true,
		status: RUNNING,
	},
	{
		name: "opSgt: 0xFF00 > 0x1234",
		op:   opSgt,
		data: []uint256.Int{
			{0x1234, 0x00, 0x00, 0x00},
			{0xFF00, 0x00, 0x00, 0x00}},
		res:    true,
		status: RUNNING,
	},
	{
		name: "opSgt: 0x1234 > 0x00",
		op:   opSgt,
		data: []uint256.Int{
			{0x0000, 0x00, 0x00, 0x00},
			{0x1234, 0x00, 0x00, 0x00}},
		res:    true,
		status: RUNNING,
	},
	{
		name: "opSgt: 0x1234 > 0xFFFF",
		op:   opSgt,
		data: []uint256.Int{
			{0xFFFF, 0x00, 0x00, 0x00},
			{0x1234, 0x00, 0x00, 0x00}},
		res:    false,
		status: RUNNING,
	},
	{
		name: "opSgt: 0x1234 > 0xFF00",
		op:   opSgt,
		data: []uint256.Int{
			{0xFF00, 0x00, 0x00, 0x00},
			{0x1234, 0x00, 0x00, 0x00}},
		res:    false,
		status: RUNNING,
	},
	{
		name: "opSgt: 0xFFFF > 0xFFFF",
		op:   opSgt,
		data: []uint256.Int{
			{0xFFFF, 0x00, 0x00, 0x00},
			{0xFFFF, 0x00, 0x00, 0x00}},
		res:    false,
		status: RUNNING,
	},
	{
		name: "opSgt: 0 > 0",
		op:   opSgt,
		data: []uint256.Int{
			{0x00, 0x00, 0x00, 0x00},
			{0x00, 0x00, 0x00, 0x00}},
		res:    false,
		status: RUNNING,
	},
	{
		name: "opSgt: -1 > 0",
		op:   opSgt,
		data: []uint256.Int{
			{0x00, 0x00, 0x00, 0x00},
			{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF}},
		res:    false,
		status: RUNNING,
	},
	{
		name: "opSgt: -1 > -16",
		op:   opSgt,
		data: []uint256.Int{
			{0xFFFFFFFFFFFFFFF0, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF},
			{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF}},
		res:    true,
		status: RUNNING,
	},
	{
		name: "opSgt: -1 > max",
		op:   opSgt,
		data: []uint256.Int{
			{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0x7FFFFFFFFFFFFFFF},
			{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF}},
		res:    false,
		status: RUNNING,
	},
	{
		name: "opSgt: min > 1",
		op:   opSgt,
		data: []uint256.Int{
			{0x01, 0x00, 0x00, 0x00},
			{0x00, 0x00, 0x00, 0x8000000000000000}},
		res:    false,
		status: RUNNING,
	},
	{
		name: "opSgt: -1 > -1",
		op:   opSgt,
		data: []uint256.Int{
			{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF},
			{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF}},
		res:    false,
		status: RUNNING,
	},
	{
		name: "opSgt: -1 > 1",
		op:   opSgt,
		data: []uint256.Int{
			{0x01, 0x00, 0x00, 0x00},
			{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF}},
		res:    false,
		status: RUNNING,
	},
	{
		name: "opSgt: 1 > -1",
		op:   opSgt,
		data: []uint256.Int{
			{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF},
			{0x01, 0x00, 0x00, 0x00}},
		res:    true,
		status: RUNNING,
	},
	{
		name: "opSgt: max > max",
		op:   opSgt,
		data: []uint256.Int{
			{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0x7FFFFFFFFFFFFFFF},
			{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0x7FFFFFFFFFFFFFFF}},
		res:    false,
		status: RUNNING,
	},
	{
		name: "opSgt: min > min",
		op:   opSgt,
		data: []uint256.Int{
			{0x00, 0x00, 0x00, 0x8000000000000000},
			{0x00, 0x00, 0x00, 0x8000000000000000}},
		res:    false,
		status: RUNNING,
	},
}

type tTestDataStackOp struct {
	name   string              // test description
	op     func(*context, int) // tested operation
	data   []uint256.Int       // input data
	res    []uint256.Int       // expected result
	pos    int                 // position in stack
	status Status              // expected status
}

var testDataStackOp = []tTestDataStackOp{

	// operation Swap
	/*{
		name:   "opSwap: no item in stack",
		op:   opSwap,
		data:   []uint256.Int{},
		res:    []uint256.Int{},
		pos:    0,
		status: RUNNING,
	},*/
	{
		name: "opSwap: pos 0",
		op:   opSwap,
		data: []uint256.Int{
			{0x1234567887654321, 0xFEDCBA9889ABCDEF, 0x123456789ABCDEF0, 0x0FEDCBA987654321}},
		res: []uint256.Int{
			{0x1234567887654321, 0xFEDCBA9889ABCDEF, 0x123456789ABCDEF0, 0x0FEDCBA987654321}},
		pos:    0,
		status: RUNNING,
	},
	{
		name: "opSwap: pos 1",
		op:   opSwap,
		data: []uint256.Int{
			{0x1234567887654321, 0xFEDCBA9889ABCDEF, 0x123456789ABCDEF0, 0x0FEDCBA987654321},
			{0x00, 0x00, 0x00, 0x00}},
		res: []uint256.Int{
			{0x00, 0x00, 0x00, 0x00},
			{0x1234567887654321, 0xFEDCBA9889ABCDEF, 0x123456789ABCDEF0, 0x0FEDCBA987654321}},
		pos:    1,
		status: RUNNING,
	},
	{
		name: "opSwap: pos 2",
		op:   opSwap,
		data: []uint256.Int{
			{0x0301, 0x0302, 0x0303, 0x0304},
			{0x0201, 0x0202, 0x0203, 0x0204},
			{0x0101, 0x0102, 0x0103, 0x0104}},
		res: []uint256.Int{
			{0x0101, 0x0102, 0x0103, 0x0104},
			{0x0201, 0x0202, 0x0203, 0x0204},
			{0x0301, 0x0302, 0x0303, 0x0304}},
		pos:    2,
		status: RUNNING,
	},
	{
		name: "opSwap: pos 3",
		op:   opSwap,
		data: []uint256.Int{
			{0x0401, 0x0402, 0x0403, 0x0404},
			{0x0301, 0x0302, 0x0303, 0x0304},
			{0x0201, 0x0202, 0x0203, 0x0204},
			{0x0101, 0x0102, 0x0103, 0x0104}},
		res: []uint256.Int{
			{0x0101, 0x0102, 0x0103, 0x0104},
			{0x0301, 0x0302, 0x0303, 0x0304},
			{0x0201, 0x0202, 0x0203, 0x0204},
			{0x0401, 0x0402, 0x0403, 0x0404}},
		pos:    3,
		status: RUNNING,
	},
	{
		name: "opSwap: pos 2, 4 items",
		op:   opSwap,
		data: []uint256.Int{
			{0x0401, 0x0402, 0x0403, 0x0404},
			{0x0301, 0x0302, 0x0303, 0x0304},
			{0x0201, 0x0202, 0x0203, 0x0204},
			{0x0101, 0x0102, 0x0103, 0x0104}},
		res: []uint256.Int{
			{0x0401, 0x0402, 0x0403, 0x0404},
			{0x0101, 0x0102, 0x0103, 0x0104},
			{0x0201, 0x0202, 0x0203, 0x0204},
			{0x0301, 0x0302, 0x0303, 0x0304}},
		pos:    2,
		status: RUNNING,
	},
	/*{
		name: "opSwap: pos 4, 4 items",
		op:   opSwap,
		data: []uint256.Int{
			{0x0401, 0x0402, 0x0403, 0x0404},
			{0x0301, 0x0302, 0x0303, 0x0304},
			{0x0201, 0x0202, 0x0203, 0x0204},
			{0x0101, 0x0102, 0x0103, 0x0104}},
		res: []uint256.Int{
			{0x0401, 0x0402, 0x0403, 0x0404},
			{0x0301, 0x0302, 0x0303, 0x0304},
			{0x0201, 0x0202, 0x0203, 0x0204},
			{0x0101, 0x0102, 0x0103, 0x0104}},
		pos:    4,
		status: ERROR,
	},*/
	{
		name: "opSwap: pos 16",
		op:   opSwap,
		data: []uint256.Int{
			{0x1101, 0x1102, 0x1103, 0x1104},
			{0x1001, 0x1002, 0x1003, 0x1004},
			{0x0F01, 0x0F02, 0x0F03, 0x0F04},
			{0x0E01, 0x0E02, 0x0E03, 0x0E04},
			{0x0D01, 0x0D02, 0x0D03, 0x0D04},
			{0x0C01, 0x0C02, 0x0C03, 0x0C04},
			{0x0B01, 0x0B02, 0x0B03, 0x0B04},
			{0x0A01, 0x0A02, 0x0A03, 0x0A04},
			{0x0901, 0x0902, 0x0903, 0x0904},
			{0x0801, 0x0802, 0x0803, 0x0804},
			{0x0701, 0x0702, 0x0703, 0x0704},
			{0x0601, 0x0602, 0x0603, 0x0604},
			{0x0501, 0x0502, 0x0503, 0x0504},
			{0x0401, 0x0402, 0x0403, 0x0404},
			{0x0301, 0x0302, 0x0303, 0x0304},
			{0x0201, 0x0202, 0x0203, 0x0204},
			{0x0101, 0x0102, 0x0103, 0x0104}},
		res: []uint256.Int{
			{0x0101, 0x0102, 0x0103, 0x0104},
			{0x1001, 0x1002, 0x1003, 0x1004},
			{0x0F01, 0x0F02, 0x0F03, 0x0F04},
			{0x0E01, 0x0E02, 0x0E03, 0x0E04},
			{0x0D01, 0x0D02, 0x0D03, 0x0D04},
			{0x0C01, 0x0C02, 0x0C03, 0x0C04},
			{0x0B01, 0x0B02, 0x0B03, 0x0B04},
			{0x0A01, 0x0A02, 0x0A03, 0x0A04},
			{0x0901, 0x0902, 0x0903, 0x0904},
			{0x0801, 0x0802, 0x0803, 0x0804},
			{0x0701, 0x0702, 0x0703, 0x0704},
			{0x0601, 0x0602, 0x0603, 0x0604},
			{0x0501, 0x0502, 0x0503, 0x0504},
			{0x0401, 0x0402, 0x0403, 0x0404},
			{0x0301, 0x0302, 0x0303, 0x0304},
			{0x0201, 0x0202, 0x0203, 0x0204},
			{0x1101, 0x1102, 0x1103, 0x1104}},
		pos:    16,
		status: RUNNING,
	},
	{
		name: "opSwap: pos 17",
		op:   opSwap,
		data: []uint256.Int{
			{0x1201, 0x1202, 0x1203, 0x1204},
			{0x1101, 0x1102, 0x1103, 0x1104},
			{0x1001, 0x1002, 0x1003, 0x1004},
			{0x0F01, 0x0F02, 0x0F03, 0x0F04},
			{0x0E01, 0x0E02, 0x0E03, 0x0E04},
			{0x0D01, 0x0D02, 0x0D03, 0x0D04},
			{0x0C01, 0x0C02, 0x0C03, 0x0C04},
			{0x0B01, 0x0B02, 0x0B03, 0x0B04},
			{0x0A01, 0x0A02, 0x0A03, 0x0A04},
			{0x0901, 0x0902, 0x0903, 0x0904},
			{0x0801, 0x0802, 0x0803, 0x0804},
			{0x0701, 0x0702, 0x0703, 0x0704},
			{0x0601, 0x0602, 0x0603, 0x0604},
			{0x0501, 0x0502, 0x0503, 0x0504},
			{0x0401, 0x0402, 0x0403, 0x0404},
			{0x0301, 0x0302, 0x0303, 0x0304},
			{0x0201, 0x0202, 0x0203, 0x0204},
			{0x0101, 0x0102, 0x0103, 0x0104}},
		res: []uint256.Int{
			{0x0101, 0x0102, 0x0103, 0x0104},
			{0x1101, 0x1102, 0x1103, 0x1104},
			{0x1001, 0x1002, 0x1003, 0x1004},
			{0x0F01, 0x0F02, 0x0F03, 0x0F04},
			{0x0E01, 0x0E02, 0x0E03, 0x0E04},
			{0x0D01, 0x0D02, 0x0D03, 0x0D04},
			{0x0C01, 0x0C02, 0x0C03, 0x0C04},
			{0x0B01, 0x0B02, 0x0B03, 0x0B04},
			{0x0A01, 0x0A02, 0x0A03, 0x0A04},
			{0x0901, 0x0902, 0x0903, 0x0904},
			{0x0801, 0x0802, 0x0803, 0x0804},
			{0x0701, 0x0702, 0x0703, 0x0704},
			{0x0601, 0x0602, 0x0603, 0x0604},
			{0x0501, 0x0502, 0x0503, 0x0504},
			{0x0401, 0x0402, 0x0403, 0x0404},
			{0x0301, 0x0302, 0x0303, 0x0304},
			{0x0201, 0x0202, 0x0203, 0x0204},
			{0x1201, 0x1202, 0x1203, 0x1204}},
		pos:    17,
		status: RUNNING, //???
	},

	// operation Dup
	{
		name: "opDup: pos 0",
		op:   opDup,
		data: []uint256.Int{},
		res: []uint256.Int{
			{0x00, 0x00, 0x00, 0x00}},
		pos:    0x00,
		status: RUNNING,
	},
	{
		name: "opDup: pos 1",
		op:   opDup,
		data: []uint256.Int{
			{0x1234567887654321, 0xFEDCBA9889ABCDEF, 0x123456789ABCDEF0, 0x0FEDCBA987654321}},
		res: []uint256.Int{
			{0x1234567887654321, 0xFEDCBA9889ABCDEF, 0x123456789ABCDEF0, 0x0FEDCBA987654321},
			{0x1234567887654321, 0xFEDCBA9889ABCDEF, 0x123456789ABCDEF0, 0x0FEDCBA987654321}},
		pos:    0x01,
		status: RUNNING,
	},
	{
		name: "opDup: pos 2",
		op:   opDup,
		data: []uint256.Int{
			{0x0201, 0x0202, 0x0203, 0x0204},
			{0x0101, 0x0102, 0x0103, 0x0104}},
		res: []uint256.Int{
			{0x0201, 0x0202, 0x0203, 0x0204},
			{0x0101, 0x0102, 0x0103, 0x0104},
			{0x0201, 0x0202, 0x0203, 0x0204}},
		pos:    0x02,
		status: RUNNING,
	},
	{
		name: "opDup: pos 3",
		op:   opDup,
		data: []uint256.Int{
			{0x0301, 0x0302, 0x0303, 0x0304},
			{0x0201, 0x0202, 0x0203, 0x0204},
			{0x0101, 0x0102, 0x0103, 0x0104}},
		res: []uint256.Int{
			{0x0301, 0x0302, 0x0303, 0x0304},
			{0x0201, 0x0202, 0x0203, 0x0204},
			{0x0101, 0x0102, 0x0103, 0x0104},
			{0x0301, 0x0302, 0x0303, 0x0304}},
		pos:    0x03,
		status: RUNNING,
	},
	{
		name: "opDup: pos 3, 4 items",
		op:   opDup,
		data: []uint256.Int{
			{0x0401, 0x0402, 0x0403, 0x0404},
			{0x0301, 0x0302, 0x0303, 0x0304},
			{0x0201, 0x0202, 0x0203, 0x0204},
			{0x0101, 0x0102, 0x0103, 0x0104}},
		res: []uint256.Int{
			{0x0401, 0x0402, 0x0403, 0x0404},
			{0x0301, 0x0302, 0x0303, 0x0304},
			{0x0201, 0x0202, 0x0203, 0x0204},
			{0x0101, 0x0102, 0x0103, 0x0104},
			{0x0301, 0x0302, 0x0303, 0x0304}},
		pos:    0x03,
		status: RUNNING,
	},
	/*{
		name: "opDup: pos 4, 3 items",
		op:   opDup,
		data: []uint256.Int{
			{0x0301, 0x0302, 0x0303, 0x0304},
			{0x0201, 0x0202, 0x0203, 0x0204},
			{0x0101, 0x0102, 0x0103, 0x0104}},
		res: []uint256.Int{
			{0x0301, 0x0302, 0x0303, 0x0304},
			{0x0201, 0x0202, 0x0203, 0x0204},
			{0x0101, 0x0102, 0x0103, 0x0104}},
		pos:    4,
		status: ERROR,
	},*/
	{
		name: "opDup: pos 16",
		op:   opDup,
		data: []uint256.Int{
			{0x1001, 0x1002, 0x1003, 0x1004},
			{0x0F01, 0x0F02, 0x0F03, 0x0F04},
			{0x0E01, 0x0E02, 0x0E03, 0x0E04},
			{0x0D01, 0x0D02, 0x0D03, 0x0D04},
			{0x0C01, 0x0C02, 0x0C03, 0x0C04},
			{0x0B01, 0x0B02, 0x0B03, 0x0B04},
			{0x0A01, 0x0A02, 0x0A03, 0x0A04},
			{0x0901, 0x0902, 0x0903, 0x0904},
			{0x0801, 0x0802, 0x0803, 0x0804},
			{0x0701, 0x0702, 0x0703, 0x0704},
			{0x0601, 0x0602, 0x0603, 0x0604},
			{0x0501, 0x0502, 0x0503, 0x0504},
			{0x0401, 0x0402, 0x0403, 0x0404},
			{0x0301, 0x0302, 0x0303, 0x0304},
			{0x0201, 0x0202, 0x0203, 0x0204},
			{0x0101, 0x0102, 0x0103, 0x0104}},
		res: []uint256.Int{
			{0x1001, 0x1002, 0x1003, 0x1004},
			{0x0F01, 0x0F02, 0x0F03, 0x0F04},
			{0x0E01, 0x0E02, 0x0E03, 0x0E04},
			{0x0D01, 0x0D02, 0x0D03, 0x0D04},
			{0x0C01, 0x0C02, 0x0C03, 0x0C04},
			{0x0B01, 0x0B02, 0x0B03, 0x0B04},
			{0x0A01, 0x0A02, 0x0A03, 0x0A04},
			{0x0901, 0x0902, 0x0903, 0x0904},
			{0x0801, 0x0802, 0x0803, 0x0804},
			{0x0701, 0x0702, 0x0703, 0x0704},
			{0x0601, 0x0602, 0x0603, 0x0604},
			{0x0501, 0x0502, 0x0503, 0x0504},
			{0x0401, 0x0402, 0x0403, 0x0404},
			{0x0301, 0x0302, 0x0303, 0x0304},
			{0x0201, 0x0202, 0x0203, 0x0204},
			{0x0101, 0x0102, 0x0103, 0x0104},
			{0x1001, 0x1002, 0x1003, 0x1004}},
		pos:    0x10,
		status: RUNNING,
	},
	{
		name: "opDup: pos 17",
		op:   opDup,
		data: []uint256.Int{
			{0x1101, 0x1102, 0x1103, 0x1104},
			{0x1001, 0x1002, 0x1003, 0x1004},
			{0x0F01, 0x0F02, 0x0F03, 0x0F04},
			{0x0E01, 0x0E02, 0x0E03, 0x0E04},
			{0x0D01, 0x0D02, 0x0D03, 0x0D04},
			{0x0C01, 0x0C02, 0x0C03, 0x0C04},
			{0x0B01, 0x0B02, 0x0B03, 0x0B04},
			{0x0A01, 0x0A02, 0x0A03, 0x0A04},
			{0x0901, 0x0902, 0x0903, 0x0904},
			{0x0801, 0x0802, 0x0803, 0x0804},
			{0x0701, 0x0702, 0x0703, 0x0704},
			{0x0601, 0x0602, 0x0603, 0x0604},
			{0x0501, 0x0502, 0x0503, 0x0504},
			{0x0401, 0x0402, 0x0403, 0x0404},
			{0x0301, 0x0302, 0x0303, 0x0304},
			{0x0201, 0x0202, 0x0203, 0x0204},
			{0x0101, 0x0102, 0x0103, 0x0104}},
		res: []uint256.Int{
			{0x1101, 0x1102, 0x1103, 0x1104},
			{0x1001, 0x1002, 0x1003, 0x1004},
			{0x0F01, 0x0F02, 0x0F03, 0x0F04},
			{0x0E01, 0x0E02, 0x0E03, 0x0E04},
			{0x0D01, 0x0D02, 0x0D03, 0x0D04},
			{0x0C01, 0x0C02, 0x0C03, 0x0C04},
			{0x0B01, 0x0B02, 0x0B03, 0x0B04},
			{0x0A01, 0x0A02, 0x0A03, 0x0A04},
			{0x0901, 0x0902, 0x0903, 0x0904},
			{0x0801, 0x0802, 0x0803, 0x0804},
			{0x0701, 0x0702, 0x0703, 0x0704},
			{0x0601, 0x0602, 0x0603, 0x0604},
			{0x0501, 0x0502, 0x0503, 0x0504},
			{0x0401, 0x0402, 0x0403, 0x0404},
			{0x0301, 0x0302, 0x0303, 0x0304},
			{0x0201, 0x0202, 0x0203, 0x0204},
			{0x0101, 0x0102, 0x0103, 0x0104},
			{0x1101, 0x1102, 0x1103, 0x1104}},
		pos:    0x11,
		status: RUNNING, //???
	},
}

type tTestDataMemOp struct {
	name     string         // test description
	op       func(*context) // tested operation
	data     []uint256.Int  // input data
	res      []uint256.Int  // expected result
	status   Status         // expected status
	gasStore uint64         // required gas for store
	gasLoad  uint64         // required gas for load
}

var testDataMemOp = []tTestDataMemOp{

	// operation Mstore, Mload
	{
		name: "Mstore, Mload: store to addr 0x00, load from addr 0x00",
		op:   opMstore,
		data: []uint256.Int{
			{0xEF0123456789ABCD, 0xF0123456789ABCDE, 0x0123456789ABCDEF, 0x123456789ABCDEF0},
			{0x00, 0x00, 0x00, 0x00}},
		res: []uint256.Int{
			{0x00, 0x00, 0x00, 0x00},
			{0xEF0123456789ABCD, 0xF0123456789ABCDE, 0x0123456789ABCDEF, 0x123456789ABCDEF0}},
		status: RUNNING,
		// new_mem_size_words = ((0x00+0x1F) + 0x1F) / 0x20 = 0x01
		// gas_cost = (0x01 * 0x01 / 0x0200) + (0x03 * 0x01) = 0x03
		gasStore: 3,
		gasLoad:  0,
	},
	{
		name: "Mstore, Mload: store to addr 0x00, load from addr 0x08",
		op:   opMstore,
		data: []uint256.Int{
			{0xEF0123456789ABCD, 0xF0123456789ABCDE, 0x0123456789ABCDEF, 0x123456789ABCDEF0},
			{0x00, 0x00, 0x00, 0x00}},
		res: []uint256.Int{
			{0x08, 0x00, 0x00, 0x00},
			{0x00, 0xEF0123456789ABCD, 0xF0123456789ABCDE, 0x0123456789ABCDEF}},
		status:   RUNNING,
		gasStore: 3,
		// new_mem_size_words = ((0x08+0x1F) + 0x1F) / 0x20 = 0x02
		// gas_cost = (0x02 * 0x02 / 0x0200) + (0x03 * 0x02) - 0x03 = 0x03
		gasLoad: 3,
	},
	{
		name: "Mstore, Mload: store to addr 0x10, load from addr 0x10",
		op:   opMstore,
		data: []uint256.Int{
			{0xEF0123456789ABCD, 0xF0123456789ABCDE, 0x0123456789ABCDEF, 0x123456789ABCDEF0},
			{0x10, 0x00, 0x00, 0x00}},
		res: []uint256.Int{
			{0x10, 0x00, 0x00, 0x00},
			{0xEF0123456789ABCD, 0xF0123456789ABCDE, 0x0123456789ABCDEF, 0x123456789ABCDEF0}},
		status: RUNNING,
		// new_mem_size_words = ((0x10+0x1F) + 0x1F) / 0x20 = 0x02
		// gas_cost = (0x02 * 0x02 / 0x0200) + (0x03 * 0x02) = 0x06
		gasStore: 6,
		gasLoad:  0,
	},
	{
		name: "Mstore, Mload: store to addr 0x10, load from addr 0x08",
		op:   opMstore,
		data: []uint256.Int{
			{0xEF0123456789ABCD, 0xF0123456789ABCDE, 0x0123456789ABCDEF, 0x123456789ABCDEF0},
			{0x10, 0x00, 0x00, 0x00}},
		res: []uint256.Int{
			{0x08, 0x00, 0x00, 0x00},
			{0xF0123456789ABCDE, 0x0123456789ABCDEF, 0x123456789ABCDEF0, 0x00}},
		status:   RUNNING,
		gasStore: 6,
		gasLoad:  0,
	},
	{
		name: "Mstore, Mload: store to addr 0x00 and 0x10, load from addr 0x08",
		op:   opMstore,
		data: []uint256.Int{
			{0x1111111111111111, 0x2222222222222222, 0x3333333333333333, 0x4444444444444444},
			{0x00, 0x00, 0x00, 0x00},
			{0xEF0123456789ABCD, 0xF0123456789ABCDE, 0x0123456789ABCDEF, 0x123456789ABCDEF0},
			{0x10, 0x00, 0x00, 0x00}},
		res: []uint256.Int{
			{0x08, 0x00, 0x00, 0x00},
			{0xF0123456789ABCDE, 0x1111111111111111, 0x2222222222222222, 0x3333333333333333}},
		status:   RUNNING,
		gasStore: 6,
		gasLoad:  0,
	},
	{
		name: "Mstore, Mload: store to addr 0x18, 0x10, 0x08 and 0x00, read form addr 0x00",
		op:   opMstore,
		data: []uint256.Int{
			{0x1111111111111111, 0x2222222222222222, 0x3333333333333333, 0x4444444444444444},
			{0x18, 0x00, 0x00, 0x00},
			{0x5555555555555555, 0x6666666666666666, 0x7777777777777777, 0x8888888888888888},
			{0x10, 0x00, 0x00, 0x00},
			{0x9999999999999999, 0xAAAAAAAAAAAAAAAA, 0xBBBBBBBBBBBBBBBB, 0xCCCCCCCCCCCCCCCC},
			{0x08, 0x00, 0x00, 0x00},
			{0xDDDDDDDDDDDDDDDD, 0xEEEEEEEEEEEEEEEE, 0xFFFFFFFFFFFFFFFF, 0x123456789ABCDEF0},
			{0x00, 0x00, 0x00, 0x00}},
		res: []uint256.Int{
			{0x00, 0x00, 0x00, 0x00},
			{0x4444444444444444, 0x8888888888888888, 0xCCCCCCCCCCCCCCCC, 0x123456789ABCDEF0}},
		status:   RUNNING,
		gasStore: 6,
		gasLoad:  0,
	},
	{
		name: "Mstore, Mload: store to addr 0x18, 0x10, 0x08 and 0x00, load from addr 0x00 and 0x20",
		op:   opMstore,
		data: []uint256.Int{
			{0x1111111111111111, 0x2222222222222222, 0x3333333333333333, 0x4444444444444444},
			{0x18, 0x00, 0x00, 0x00},
			{0x5555555555555555, 0x6666666666666666, 0x7777777777777777, 0x8888888888888888},
			{0x10, 0x00, 0x00, 0x00},
			{0x9999999999999999, 0xAAAAAAAAAAAAAAAA, 0xBBBBBBBBBBBBBBBB, 0xCCCCCCCCCCCCCCCC},
			{0x08, 0x00, 0x00, 0x00},
			{0xDDDDDDDDDDDDDDDD, 0xEEEEEEEEEEEEEEEE, 0xFFFFFFFFFFFFFFFF, 0x123456789ABCDEF0},
			{0x00, 0x00, 0x00, 0x00}},
		res: []uint256.Int{
			{0x00, 0x00, 0x00, 0x00},
			{0x4444444444444444, 0x8888888888888888, 0xCCCCCCCCCCCCCCCC, 0x123456789ABCDEF0},
			{0x20, 0x00, 0x00, 0x00},
			{0x0000000000000000, 0x1111111111111111, 0x2222222222222222, 0x3333333333333333}},
		status:   RUNNING,
		gasStore: 6,
		gasLoad:  0,
	},
	{
		name: "Mstore, Mload: store to addr 2^64, load from addr 2^64, status ERROR",
		op:   opMstore,
		data: []uint256.Int{
			{0xEF0123456789ABCD, 0xF0123456789ABCDE, 0x0123456789ABCDEF, 0x123456789ABCDEF0},
			{0x00, 0x01, 0x00, 0x00}},
		res: []uint256.Int{
			{0x00, 0x01, 0x00, 0x00},
			{0x00, 0x00, 0x00, 0x00}},
		status:   ERROR,
		gasStore: 0,
		gasLoad:  0,
	},
	/*{
		// runtime error: slice bounds out of range
		name: "Mstore, Mload: store to addr 2^64-1, status ERROR",
		op:   opMstore,
		data: []uint256.Int{
			{0xEF0123456789ABCD, 0xF0123456789ABCDE, 0x0123456789ABCDEF, 0x123456789ABCDEF0},
			{0xFFFFFFFFFFFFFFFF, 0x00, 0x00, 0x00}},
		res: []uint256.Int{
			{0xFFFFFFFFFFFFFFFF, 0x00, 0x00, 0x00},
			{0xEF0123456789ABCD, 0xF0123456789ABCDE, 0x0123456789ABCDEF, 0x123456789ABCDEF0}},
		status:   RUNNING,
		gasStore: 1000,
		gasLoad:  0,
	},*/
	{
		name: "Mstore, Mload: store to addr 0xFF00, load from addr 0xFF00",
		op:   opMstore,
		data: []uint256.Int{
			{0xEF0123456789ABCD, 0xF0123456789ABCDE, 0x0123456789ABCDEF, 0x123456789ABCDEF0},
			{0x000000000000FF00, 0x00, 0x00, 0x00}},
		res: []uint256.Int{
			{0x000000000000FF00, 0x00, 0x00, 0x00},
			{0xEF0123456789ABCD, 0xF0123456789ABCDE, 0x0123456789ABCDEF, 0x123456789ABCDEF0}},
		status: RUNNING,
		// new_mem_size_words = ((0xFF00+0x1F) + 0x1F) / 0x20 = 0x07F9
		// gas_cost = (0x07F9 * 0x07F9 / 0x0200) + (3 * 0x07F9) = 0x37B3,
		gasStore: 0x37B3,
		gasLoad:  0,
	},
	{
		name: "Mstore, Mload: store to addr 0xFFFFFFFF, load from addr 0xFFFFFFFF",
		op:   opMstore,
		data: []uint256.Int{
			{0xEF0123456789ABCD, 0xF0123456789ABCDE, 0x0123456789ABCDEF, 0x123456789ABCDEF0},
			{0x00000000FFFFFFFF, 0x00, 0x00, 0x00}},
		res: []uint256.Int{
			{0x00000000FFFFFFFF, 0x00, 0x00, 0x00},
			{0xEF0123456789ABCD, 0xF0123456789ABCDE, 0x0123456789ABCDEF, 0x123456789ABCDEF0}},
		status: RUNNING,
		// new_mem_size_words = ((0xFFFFFFFF+0x1F) + 0x1F) / 0x20 = 0x08000001
		// gas_cost = (0x08000001 * 0x08000001 / 0x0200) + (3 * 0x08000001) = 0x200018080003,
		gasStore: 0x200018080003,
		gasLoad:  0,
	},
	{
		name: "Mstore, Mload: store to addr 0x00 and 0xFFFFFFFF, load from addr 0x00",
		op:   opMstore,
		data: []uint256.Int{
			{0xEF0123456789ABCD, 0xF0123456789ABCDE, 0x0123456789ABCDEF, 0x123456789ABCDEF0},
			{0x00000000FFFFFFFF, 0x00, 0x00, 0x00},
			{0xDDDDDDDDDDDDDDDD, 0xEEEEEEEEEEEEEEEE, 0xFFFFFFFFFFFFFFFF, 0x123456789ABCDEF0},
			{0x00, 0x00, 0x00, 0x00}},
		res: []uint256.Int{
			{0x00, 0x00, 0x00, 0x00},
			{0xDDDDDDDDDDDDDDDD, 0xEEEEEEEEEEEEEEEE, 0xFFFFFFFFFFFFFFFF, 0x123456789ABCDEF0}},
		status:   RUNNING,
		gasStore: 0x200018080003,
		gasLoad:  0,
	},
	/*{
		// runtime error, it kill VS Code
		name: "Mstore, Mload: store to addr 0xFFFFFFFFFF, load from addr 0xFFFFFFFF",
		op:   opMstore,
		data: []uint256.Int{
			{0xEF0123456789ABCD, 0xF0123456789ABCDE, 0x0123456789ABCDEF, 0x123456789ABCDEF0},
			{0x000000FFFFFFFFFF, 0x00, 0x00, 0x00}},
		res: []uint256.Int{
			{0x000000FFFFFFFFFF, 0x00, 0x00, 0x00},
			{0xEF0123456789ABCD, 0xF0123456789ABCDE, 0x0123456789ABCDEF, 0x123456789ABCDEF0}},
		status: RUNNING,
		gasStore: 0x200018080003,
		gasLoad:  0,
	},*/

	// operation Mstore8
	{
		name: "Mstore8, Mload: store byte to addr 0x00, load from addr 0x00",
		op:   opMstore8,
		data: []uint256.Int{
			{0xEF0123456789ABCD, 0xF0123456789ABCDE, 0x0123456789ABCDEF, 0x123456789ABCDEF0},
			{0x00, 0x00, 0x00, 0x00}},
		res: []uint256.Int{
			{0x00, 0x00, 0x00, 0x00},
			{0x00, 0x00, 0x00, 0xCD00000000000000}},
		status:   RUNNING,
		gasStore: 3,
		gasLoad:  0,
	},
	{
		name: "Mstore8, Mload: store byte to addr 0x01, load from addr 0x00",
		op:   opMstore8,
		data: []uint256.Int{
			{0xEF0123456789ABCD, 0xF0123456789ABCDE, 0x0123456789ABCDEF, 0x123456789ABCDEF0},
			{0x01, 0x00, 0x00, 0x00}},
		res: []uint256.Int{
			{0x00, 0x00, 0x00, 0x00},
			{0x00, 0x00, 0x00, 0x00CD000000000000}},
		status:   RUNNING,
		gasStore: 3,
		gasLoad:  0,
	},
	{
		name: "Mstore8, Mload: store byte to addr 0x02, load from addr 0x00",
		op:   opMstore8,
		data: []uint256.Int{
			{0xEF0123456789ABCD, 0xF0123456789ABCDE, 0x0123456789ABCDEF, 0x123456789ABCDEF0},
			{0x02, 0x00, 0x00, 0x00}},
		res: []uint256.Int{
			{0x00, 0x00, 0x00, 0x00},
			{0x00, 0x00, 0x00, 0x0000CD0000000000}},
		status:   RUNNING,
		gasStore: 3,
		gasLoad:  0,
	},
	{
		name: "Mstore8, Mload: store byte to addr 0x02, load from addr 0x02",
		op:   opMstore8,
		data: []uint256.Int{
			{0xEF0123456789ABCD, 0xF0123456789ABCDE, 0x0123456789ABCDEF, 0x123456789ABCDEF0},
			{0x02, 0x00, 0x00, 0x00}},
		res: []uint256.Int{
			{0x02, 0x00, 0x00, 0x00},
			{0x00, 0x00, 0x00, 0xCD00000000000000}},
		status:   RUNNING,
		gasStore: 3,
		gasLoad:  3,
	},
	{
		name: "Mstore8, Mload: store byte to addr 0x10, load from addr 0x00",
		op:   opMstore8,
		data: []uint256.Int{
			{0x0000000000000098, 0xF0123456789ABCDE, 0x0123456789ABCDEF, 0x123456789ABCDEF0},
			{0x10, 0x00, 0x00, 0x00}},
		res: []uint256.Int{
			{0x00, 0x00, 0x00, 0x00},
			{0x00, 0x9800000000000000, 0x00, 0x00}},
		status:   RUNNING,
		gasStore: 3,
		gasLoad:  0,
	},
	{
		name: "Mstore8: store byte to addr 0x00 and 0x10, load from addr 0x00",
		op:   opMstore8,
		data: []uint256.Int{
			{0xEF0123456789ABCD, 0xF0123456789ABCDE, 0x0123456789ABCDEF, 0x123456789ABCDEF0},
			{0x00, 0x00, 0x00, 0x00},
			{0x0000000000000098, 0xF0123456789ABCDE, 0x0123456789ABCDEF, 0x123456789ABCDEF0},
			{0x10, 0x00, 0x00, 0x00}},
		res: []uint256.Int{
			{0x00, 0x00, 0x00, 0x00},
			{0x00, 0x9800000000000000, 0x00, 0xCD00000000000000}},
		status:   RUNNING,
		gasStore: 3,
		gasLoad:  0,
	},
	{
		name: "Mstore8, Mload: store to addr 0x00 and 0xFFFFFFFF, load from addr 0x00",
		op:   opMstore8,
		data: []uint256.Int{
			{0xEF0123456789ABCD, 0xF0123456789ABCDE, 0x0123456789ABCDEF, 0x123456789ABCDEF0},
			{0x00000000FFFFFFFF, 0x00, 0x00, 0x00},
			{0xAAAAAAAAAAAAAADD, 0xEEEEEEEEEEEEEEEE, 0xFFFFFFFFFFFFFFFF, 0x123456789ABCDEF0},
			{0x00, 0x00, 0x00, 0x00}},
		res: []uint256.Int{
			{0x00, 0x00, 0x00, 0x00},
			{0x00, 0x00, 0x00, 0xDD00000000000000},
			{0x00000000FFFFFFFF, 0x00, 0x00, 0x00},
			{0x00, 0x00, 0x00, 0xCD00000000000000}},
		status: RUNNING,
		// new_mem_size_words = (0xFFFFFFFF + 0x1F) / 0x20 = 0x08000000
		// gas_cost = (0x08000000 * 0x08000000 / 0x0200) + (3 * 0x08000000) = 0x200018000000,
		gasStore: 0x200018000000,
		gasLoad:  0x000000080003,
	},
}

// operation Msize
var testDataMsizeOp = []tTestDataOp{
	{
		name: "Mstore, Msize: store to addr 0x00, mem size 0x20",
		op:   opMsize,
		data: []uint256.Int{
			{0xEF0123456789ABCD, 0xF0123456789ABCDE, 0x0123456789ABCDEF, 0x123456789ABCDEF0},
			{0, 0, 0, 0}},
		res:    uint256.Int{0x20, 0, 0, 0},
		status: RUNNING,
		gas:    3,
	},
	{
		name: "Mstore, Msize: store to addr 0x08, mem size 0x40",
		op:   opMsize,
		data: []uint256.Int{
			{0xEF0123456789ABCD, 0xF0123456789ABCDE, 0x0123456789ABCDEF, 0x123456789ABCDEF0},
			{0x08, 0, 0, 0}},
		res:    uint256.Int{0x40, 0, 0, 0},
		status: RUNNING,
		gas:    6,
	},
	{
		name: "Mstore, Msize: store to addr 0x10, mem size 0x40",
		op:   opMsize,
		data: []uint256.Int{
			{0xEF0123456789ABCD, 0xF0123456789ABCDE, 0x0123456789ABCDEF, 0x123456789ABCDEF0},
			{0x10, 0, 0, 0}},
		res:    uint256.Int{0x40, 0, 0, 0},
		status: RUNNING,
		gas:    6,
	},
	{
		name: "Mstore, Msize: store to addr 0x20, mem size 0x40",
		op:   opMsize,
		data: []uint256.Int{
			{0xEF0123456789ABCD, 0xF0123456789ABCDE, 0x0123456789ABCDEF, 0x123456789ABCDEF0},
			{0x20, 0, 0, 0}},
		res:    uint256.Int{0x40, 0, 0, 0},
		status: RUNNING,
		gas:    6,
	},
	{
		name: "Mstore, Msize: store to addr 0x24, mem size 0x60",
		op:   opMsize,
		data: []uint256.Int{
			{0xEF0123456789ABCD, 0xF0123456789ABCDE, 0x0123456789ABCDEF, 0x123456789ABCDEF0},
			{0x24, 0, 0, 0}},
		res:    uint256.Int{0x60, 0, 0, 0},
		status: RUNNING,
		gas:    9,
	},
	{
		name: "Mstore, Msize: store to addr 0xFF00, mem size 0xFF20",
		op:   opMsize,
		data: []uint256.Int{
			{0xEF0123456789ABCD, 0xF0123456789ABCDE, 0x0123456789ABCDEF, 0x123456789ABCDEF0},
			{0xFF00, 0, 0, 0}},
		res:    uint256.Int{0xFF20, 0, 0, 0},
		status: RUNNING,
		gas:    0x37B3,
	},
}
