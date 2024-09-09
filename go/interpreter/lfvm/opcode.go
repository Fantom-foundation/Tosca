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
	"github.com/Fantom-foundation/Tosca/go/tosca/vm"
)

type OpCode uint16

// The following constants define the original EVM OpCodes, in the lfvm OpCode space.
const (
	// Stack operations
	POP    = OpCode(vm.POP)
	PUSH0  = OpCode(vm.PUSH0)
	PUSH1  = OpCode(vm.PUSH1)
	PUSH2  = OpCode(vm.PUSH2)
	PUSH3  = OpCode(vm.PUSH3)
	PUSH4  = OpCode(vm.PUSH4)
	PUSH5  = OpCode(vm.PUSH5)
	PUSH6  = OpCode(vm.PUSH6)
	PUSH7  = OpCode(vm.PUSH7)
	PUSH8  = OpCode(vm.PUSH8)
	PUSH9  = OpCode(vm.PUSH9)
	PUSH10 = OpCode(vm.PUSH10)
	PUSH11 = OpCode(vm.PUSH11)
	PUSH12 = OpCode(vm.PUSH12)
	PUSH13 = OpCode(vm.PUSH13)
	PUSH14 = OpCode(vm.PUSH14)
	PUSH15 = OpCode(vm.PUSH15)
	PUSH16 = OpCode(vm.PUSH16)
	PUSH17 = OpCode(vm.PUSH17)
	PUSH18 = OpCode(vm.PUSH18)
	PUSH19 = OpCode(vm.PUSH19)
	PUSH20 = OpCode(vm.PUSH20)
	PUSH21 = OpCode(vm.PUSH21)
	PUSH22 = OpCode(vm.PUSH22)
	PUSH23 = OpCode(vm.PUSH23)
	PUSH24 = OpCode(vm.PUSH24)
	PUSH25 = OpCode(vm.PUSH25)
	PUSH26 = OpCode(vm.PUSH26)
	PUSH27 = OpCode(vm.PUSH27)
	PUSH28 = OpCode(vm.PUSH28)
	PUSH29 = OpCode(vm.PUSH29)
	PUSH30 = OpCode(vm.PUSH30)
	PUSH31 = OpCode(vm.PUSH31)
	PUSH32 = OpCode(vm.PUSH32)

	DUP1  = OpCode(vm.DUP1)
	DUP2  = OpCode(vm.DUP2)
	DUP3  = OpCode(vm.DUP3)
	DUP4  = OpCode(vm.DUP4)
	DUP5  = OpCode(vm.DUP5)
	DUP6  = OpCode(vm.DUP6)
	DUP7  = OpCode(vm.DUP7)
	DUP8  = OpCode(vm.DUP8)
	DUP9  = OpCode(vm.DUP9)
	DUP10 = OpCode(vm.DUP10)
	DUP11 = OpCode(vm.DUP11)
	DUP12 = OpCode(vm.DUP12)
	DUP13 = OpCode(vm.DUP13)
	DUP14 = OpCode(vm.DUP14)
	DUP15 = OpCode(vm.DUP15)
	DUP16 = OpCode(vm.DUP16)

	SWAP1  = OpCode(vm.SWAP1)
	SWAP2  = OpCode(vm.SWAP2)
	SWAP3  = OpCode(vm.SWAP3)
	SWAP4  = OpCode(vm.SWAP4)
	SWAP5  = OpCode(vm.SWAP5)
	SWAP6  = OpCode(vm.SWAP6)
	SWAP7  = OpCode(vm.SWAP7)
	SWAP8  = OpCode(vm.SWAP8)
	SWAP9  = OpCode(vm.SWAP9)
	SWAP10 = OpCode(vm.SWAP10)
	SWAP11 = OpCode(vm.SWAP11)
	SWAP12 = OpCode(vm.SWAP12)
	SWAP13 = OpCode(vm.SWAP13)
	SWAP14 = OpCode(vm.SWAP14)
	SWAP15 = OpCode(vm.SWAP15)
	SWAP16 = OpCode(vm.SWAP16)

	// Control flow
	JUMP     = OpCode(vm.JUMP)
	JUMPI    = OpCode(vm.JUMPI)
	JUMPDEST = OpCode(vm.JUMPDEST)
	RETURN   = OpCode(vm.RETURN)
	REVERT   = OpCode(vm.REVERT)
	PC       = OpCode(vm.PC)
	STOP     = OpCode(vm.STOP)

	// Arithmetic
	ADD        = OpCode(vm.ADD)
	SUB        = OpCode(vm.SUB)
	MUL        = OpCode(vm.MUL)
	DIV        = OpCode(vm.DIV)
	SDIV       = OpCode(vm.SDIV)
	MOD        = OpCode(vm.MOD)
	SMOD       = OpCode(vm.SMOD)
	ADDMOD     = OpCode(vm.ADDMOD)
	MULMOD     = OpCode(vm.MULMOD)
	EXP        = OpCode(vm.EXP)
	SIGNEXTEND = OpCode(vm.SIGNEXTEND)

	// Complex function
	SHA3 = OpCode(vm.SHA3)

	// Comparison operations
	LT     = OpCode(vm.LT)
	GT     = OpCode(vm.GT)
	SLT    = OpCode(vm.SLT)
	SGT    = OpCode(vm.SGT)
	EQ     = OpCode(vm.EQ)
	ISZERO = OpCode(vm.ISZERO)

	// Bit-pattern operations
	AND  = OpCode(vm.AND)
	OR   = OpCode(vm.OR)
	XOR  = OpCode(vm.XOR)
	NOT  = OpCode(vm.NOT)
	BYTE = OpCode(vm.BYTE)
	SHL  = OpCode(vm.SHL)
	SHR  = OpCode(vm.SHR)
	SAR  = OpCode(vm.SAR)

	// Memory
	MSTORE  = OpCode(vm.MSTORE)
	MSTORE8 = OpCode(vm.MSTORE8)
	MLOAD   = OpCode(vm.MLOAD)
	MSIZE   = OpCode(vm.MSIZE)
	MCOPY   = OpCode(vm.MCOPY)

	// Storage
	SLOAD  = OpCode(vm.SLOAD)
	SSTORE = OpCode(vm.SSTORE)
	TLOAD  = OpCode(vm.TLOAD)
	TSTORE = OpCode(vm.TSTORE)

	// LOG
	LOG0 = OpCode(vm.LOG0)
	LOG1 = OpCode(vm.LOG1)
	LOG2 = OpCode(vm.LOG2)
	LOG3 = OpCode(vm.LOG3)
	LOG4 = OpCode(vm.LOG4)

	// System level instructions.
	ADDRESS        = OpCode(vm.ADDRESS)
	BALANCE        = OpCode(vm.BALANCE)
	ORIGIN         = OpCode(vm.ORIGIN)
	CALLER         = OpCode(vm.CALLER)
	CALLVALUE      = OpCode(vm.CALLVALUE)
	CALLDATALOAD   = OpCode(vm.CALLDATALOAD)
	CALLDATASIZE   = OpCode(vm.CALLDATASIZE)
	CALLDATACOPY   = OpCode(vm.CALLDATACOPY)
	CODESIZE       = OpCode(vm.CODESIZE)
	CODECOPY       = OpCode(vm.CODECOPY)
	GASPRICE       = OpCode(vm.GASPRICE)
	EXTCODESIZE    = OpCode(vm.EXTCODESIZE)
	EXTCODECOPY    = OpCode(vm.EXTCODECOPY)
	RETURNDATASIZE = OpCode(vm.RETURNDATASIZE)
	RETURNDATACOPY = OpCode(vm.RETURNDATACOPY)
	EXTCODEHASH    = OpCode(vm.EXTCODEHASH)
	CREATE         = OpCode(vm.CREATE)
	CALL           = OpCode(vm.CALL)
	CALLCODE       = OpCode(vm.CALLCODE)
	DELEGATECALL   = OpCode(vm.DELEGATECALL)
	CREATE2        = OpCode(vm.CREATE2)
	STATICCALL     = OpCode(vm.STATICCALL)
	SELFDESTRUCT   = OpCode(vm.SELFDESTRUCT)

	// Blockchain instructions
	BLOCKHASH   = OpCode(vm.BLOCKHASH)
	COINBASE    = OpCode(vm.COINBASE)
	TIMESTAMP   = OpCode(vm.TIMESTAMP)
	NUMBER      = OpCode(vm.NUMBER)
	PREVRANDAO  = OpCode(vm.PREVRANDAO)
	GAS         = OpCode(vm.GAS)
	GASLIMIT    = OpCode(vm.GASLIMIT)
	CHAINID     = OpCode(vm.CHAINID)
	SELFBALANCE = OpCode(vm.SELFBALANCE)
	BASEFEE     = OpCode(vm.BASEFEE)
	BLOBHASH    = OpCode(vm.BLOBHASH)
	BLOBBASEFEE = OpCode(vm.BLOBBASEFEE)

	// Invalid instruction
	INVALID = OpCode(vm.INVALID)
)

// The following constants define the extended set of OpCodes for the long-form
// EVM.
// These opcodes are specific to the long-form EVM and are not part of the
// original EVM.
const (
	FIRST_LFVM_EXTENDED_OPCODE = 0x100

	// long-form EVM special instructions
	JUMP_TO OpCode = FIRST_LFVM_EXTENDED_OPCODE

	// Super-instructions
	SWAP2_SWAP1_POP_JUMP  OpCode = 0x101
	SWAP1_POP_SWAP2_SWAP1 OpCode = 0x102
	POP_SWAP2_SWAP1_POP   OpCode = 0x103
	POP_POP               OpCode = 0x104
	PUSH1_SHL             OpCode = 0x105
	PUSH1_ADD             OpCode = 0x106
	PUSH1_DUP1            OpCode = 0x107
	PUSH2_JUMP            OpCode = 0x108
	PUSH2_JUMPI           OpCode = 0x109

	PUSH1_PUSH1 OpCode = 0x10A
	SWAP1_POP   OpCode = 0x10B
	POP_JUMP    OpCode = 0x10C
	SWAP2_SWAP1 OpCode = 0x10D
	SWAP2_POP   OpCode = 0x10E
	DUP2_MSTORE OpCode = 0x10F
	DUP2_LT     OpCode = 0x110

	ISZERO_PUSH2_JUMPI        OpCode = 0x111
	PUSH1_PUSH4_DUP3          OpCode = 0x112
	AND_SWAP1_POP_SWAP2_SWAP1 OpCode = 0x113
	PUSH1_PUSH1_PUSH1_SHL_SUB OpCode = 0x114

	END_LFVM_EXECUTABLE_OPCODES = 0x115

	// Special no-instructions op codes
	DATA OpCode = END_LFVM_EXECUTABLE_OPCODES
	NOOP OpCode = 0x120

	END_LFVM_EXTENDED_OPCODES = 0x121
	NUM_OPCODES               = END_LFVM_EXTENDED_OPCODES
)

var to_string = map[OpCode]string{
	JUMP_TO: "JUMP_TO",

	SWAP2_SWAP1_POP_JUMP:  "SWAP2_SWAP1_POP_JUMP",
	SWAP1_POP_SWAP2_SWAP1: "SWAP1_POP_SWAP2_SWAP1",
	POP_SWAP2_SWAP1_POP:   "POP_SWAP2_SWAP1_POP",
	PUSH2_JUMP:            "PUSH2_JUMP",
	PUSH2_JUMPI:           "PUSH2_JUMPI",
	DUP2_MSTORE:           "DUP2_MSTORE",
	DUP2_LT:               "DUP2_LT",

	SWAP1_POP:   "SWAP1_POP",
	POP_JUMP:    "POP_JUMP",
	SWAP2_SWAP1: "SWAP2_SWAP1",
	SWAP2_POP:   "SWAP2_POP",
	PUSH1_PUSH1: "PUSH1_PUSH1",
	PUSH1_ADD:   "PUSH1_ADD",
	PUSH1_DUP1:  "PUSH1_DUP1",
	POP_POP:     "POP_POP",
	PUSH1_SHL:   "PUSH1_SHL",

	ISZERO_PUSH2_JUMPI:        "ISZERO_PUSH2_JUMPI",
	PUSH1_PUSH4_DUP3:          "PUSH1_PUSH4_DUP3",
	AND_SWAP1_POP_SWAP2_SWAP1: "AND_SWAP1_POP_SWAP2_SWAP1",
	PUSH1_PUSH1_PUSH1_SHL_SUB: "PUSH1_PUSH1_PUSH1_SHL_SUB",

	DATA: "DATA",
	NOOP: "NOOP",
}

// String returns the string representation of the OpCode.
// For undefined values the string "UNKNOWN" is returned,
// instead of INVALID, which is a defined value.
func (o OpCode) String() string {
	if o < FIRST_LFVM_EXTENDED_OPCODE {
		return vm.OpCode(o).String()
	}

	if str, ok := to_string[o]; ok {
		return str
	}
	return "UNKNOWN"
}

// HasArgument returns true if the second 16-bit word of the instruction is
// argument data.
func (o OpCode) HasArgument() bool {
	if PUSH1 <= o && o <= PUSH32 {
		return true
	}
	switch o {
	case DATA:
		return true
	case JUMP_TO:
		return true
	case PUSH2_JUMP:
		return true
	case PUSH2_JUMPI:
		return true
	case PUSH1_PUSH4_DUP3:
		return true
	}
	return false
}

func (o OpCode) isSuperInstruction() bool {
	return o.decompose() != nil
}

func (o OpCode) decompose() []OpCode {
	switch o {
	case SWAP2_SWAP1_POP_JUMP:
		return []OpCode{SWAP2, SWAP1, POP, JUMP}
	case SWAP1_POP_SWAP2_SWAP1:
		return []OpCode{SWAP1, POP, SWAP2, SWAP1}
	case POP_SWAP2_SWAP1_POP:
		return []OpCode{POP, SWAP2, SWAP1, POP}
	case POP_POP:
		return []OpCode{POP, POP}
	case PUSH1_SHL:
		return []OpCode{PUSH1, SHL}
	case PUSH1_ADD:
		return []OpCode{PUSH1, ADD}
	case PUSH1_DUP1:
		return []OpCode{PUSH1, DUP1}
	case PUSH2_JUMP:
		return []OpCode{PUSH2, JUMP}
	case PUSH2_JUMPI:
		return []OpCode{PUSH2, JUMPI}
	case PUSH1_PUSH1:
		return []OpCode{PUSH1, PUSH1}
	case SWAP1_POP:
		return []OpCode{SWAP1, POP}
	case POP_JUMP:
		return []OpCode{POP, JUMP}
	case SWAP2_SWAP1:
		return []OpCode{SWAP2, SWAP1}
	case SWAP2_POP:
		return []OpCode{SWAP2, POP}
	case DUP2_MSTORE:
		return []OpCode{DUP2, MSTORE}
	case DUP2_LT:
		return []OpCode{DUP2, LT}
	case ISZERO_PUSH2_JUMPI:
		return []OpCode{ISZERO, PUSH2, JUMPI}
	case PUSH1_PUSH4_DUP3:
		return []OpCode{PUSH1, PUSH4, DUP3}
	case AND_SWAP1_POP_SWAP2_SWAP1:
		return []OpCode{AND, SWAP1, POP, SWAP2, SWAP1}
	case PUSH1_PUSH1_PUSH1_SHL_SUB:
		return []OpCode{PUSH1, PUSH1, PUSH1, SHL, SUB}
	}
	return nil
}

// IsValid returns true if the OpCode is a valid OpCode.
// A valid opcode shall be defined in the OpCode space, with the exception of
// INVALID, which is a defined value.
func (op OpCode) isValid() bool {
	if op < FIRST_LFVM_EXTENDED_OPCODE {
		return vm.IsValid(vm.OpCode(op))
	}
	return op >= FIRST_LFVM_EXTENDED_OPCODE && op < END_LFVM_EXTENDED_OPCODES
}

// isExecutable returns true if the OpCode is an executable OpCode.
// An executable OpCode is a valid OpCode that executes an operation; all valid
// OpCodes with the exception of DATA, and NOOP are executable.
// - JUMPDEST is an executable OpCode.
// - INVALID is not an executable OpCode, since it is neither Valid
func (op OpCode) isExecutable() bool {
	if op < FIRST_LFVM_EXTENDED_OPCODE {
		return op.isValid()
	}
	return op < END_LFVM_EXECUTABLE_OPCODES
}
