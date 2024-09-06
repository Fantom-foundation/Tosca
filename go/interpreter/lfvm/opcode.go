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

import "fmt"

type OpCode uint16

// The following constants define OpCodes for the long-form EVM version.
// It avoids reusing the opcodes of the EVM to allow rearranging the numeric
// values of instructions if required.
// The order is currently exploited for computing e.g. the gas cost of operations.
const (
	// Stack operations
	POP OpCode = iota
	PUSH0
	PUSH1
	PUSH2
	PUSH3
	PUSH4
	PUSH5
	PUSH6
	PUSH7
	PUSH8
	PUSH9
	PUSH10
	PUSH11
	PUSH12
	PUSH13
	PUSH14
	PUSH15
	PUSH16
	PUSH17
	PUSH18
	PUSH19
	PUSH20
	PUSH21
	PUSH22
	PUSH23
	PUSH24
	PUSH25
	PUSH26
	PUSH27
	PUSH28
	PUSH29
	PUSH30
	PUSH31
	PUSH32

	DUP1
	DUP2
	DUP3
	DUP4
	DUP5
	DUP6
	DUP7
	DUP8
	DUP9
	DUP10
	DUP11
	DUP12
	DUP13
	DUP14
	DUP15
	DUP16

	SWAP1
	SWAP2
	SWAP3
	SWAP4
	SWAP5
	SWAP6
	SWAP7
	SWAP8
	SWAP9
	SWAP10
	SWAP11
	SWAP12
	SWAP13
	SWAP14
	SWAP15
	SWAP16

	// Control flow
	JUMP
	JUMPI
	JUMPDEST
	RETURN
	REVERT
	PC
	STOP

	// Arithmetic
	ADD
	SUB
	MUL
	DIV
	SDIV
	MOD
	SMOD
	ADDMOD
	MULMOD
	EXP
	SIGNEXTEND

	// Complex function
	SHA3

	// Comparison operations
	LT
	GT
	SLT
	SGT
	EQ
	ISZERO

	// Bit-pattern operations
	AND
	OR
	XOR
	NOT
	BYTE
	SHL
	SHR
	SAR

	// Memory
	MSTORE
	MSTORE8
	MLOAD
	MSIZE
	MCOPY

	// Storage
	SLOAD
	SSTORE
	TLOAD
	TSTORE

	// LOG
	LOG0
	LOG1
	LOG2
	LOG3
	LOG4

	// System level instructions.
	ADDRESS
	BALANCE
	ORIGIN
	CALLER
	CALLVALUE
	CALLDATALOAD
	CALLDATASIZE
	CALLDATACOPY
	CODESIZE
	CODECOPY
	GASPRICE
	EXTCODESIZE
	EXTCODECOPY
	RETURNDATASIZE
	RETURNDATACOPY
	EXTCODEHASH
	CREATE
	CALL
	CALLCODE
	DELEGATECALL
	CREATE2
	STATICCALL
	SELFDESTRUCT

	// Blockchain instructions
	BLOCKHASH
	COINBASE
	TIMESTAMP
	NUMBER
	PREVRANDAO
	GAS
	GASLIMIT
	CHAINID
	SELFBALANCE
	BASEFEE
	BLOBHASH
	BLOBBASEFEE

	// long-form EVM special instructions
	JUMP_TO

	// Super-instructions
	SWAP2_SWAP1_POP_JUMP
	SWAP1_POP_SWAP2_SWAP1
	POP_SWAP2_SWAP1_POP
	POP_POP
	PUSH1_SHL
	PUSH1_ADD
	PUSH1_DUP1
	PUSH2_JUMP
	PUSH2_JUMPI

	PUSH1_PUSH1
	SWAP1_POP
	POP_JUMP
	SWAP2_SWAP1
	SWAP2_POP
	DUP2_MSTORE
	DUP2_LT

	ISZERO_PUSH2_JUMPI
	PUSH1_PUSH4_DUP3
	AND_SWAP1_POP_SWAP2_SWAP1
	PUSH1_PUSH1_PUSH1_SHL_SUB

	// Not really an Op-code but used to get the number of executable opcodes.
	NUM_EXECUTABLE_OPCODES

	// Special non-instruction op codes
	DATA
	NOOP
	INVALID

	// Not really an Op-code but used to get the number of supported op codes.
	NUM_OPCODES
)

var op_to_string = map[OpCode]string{
	POP:      "POP",
	PUSH2:    "PUSH2",
	JUMP:     "JUMP",
	SWAP1:    "SWAP1",
	SWAP2:    "SWAP2",
	DUP3:     "DUP3",
	PUSH1:    "PUSH1",
	PUSH4:    "PUSH4",
	AND:      "AND",
	SWAP3:    "SWAP3",
	JUMPI:    "JUMPI",
	JUMPDEST: "JUMPDEST",
	GT:       "GT",
	DUP4:     "DUP4",
	DUP2:     "DUP2",
	ISZERO:   "ISZERO",
	SUB:      "SUB",
	ADD:      "ADD",
	DUP5:     "DUP5",
	DUP1:     "DUP1",
	EQ:       "EQ",
	LT:       "LT",
	SLT:      "SLT",
	SHR:      "SHR",
	DUP6:     "DUP6",
	RETURN:   "RETURN",
	REVERT:   "REVERT",
	PUSH32:   "PUSH32",

	PUSH0:  "PUSH0",
	PUSH3:  "PUSH3",
	PUSH5:  "PUSH5",
	PUSH6:  "PUSH6",
	PUSH7:  "PUSH7",
	PUSH8:  "PUSH8",
	PUSH9:  "PUSH9",
	PUSH10: "PUSH10",
	PUSH11: "PUSH11",
	PUSH12: "PUSH12",
	PUSH13: "PUSH13",
	PUSH14: "PUSH14",
	PUSH15: "PUSH15",
	PUSH16: "PUSH16",
	PUSH17: "PUSH17",
	PUSH18: "PUSH18",
	PUSH19: "PUSH19",
	PUSH20: "PUSH20",
	PUSH21: "PUSH21",
	PUSH22: "PUSH22",
	PUSH23: "PUSH23",
	PUSH24: "PUSH24",
	PUSH25: "PUSH25",
	PUSH26: "PUSH26",
	PUSH27: "PUSH27",
	PUSH28: "PUSH28",
	PUSH29: "PUSH29",
	PUSH30: "PUSH30",
	PUSH31: "PUSH31",
	DUP7:   "DUP7",
	DUP8:   "DUP8",
	DUP9:   "DUP9",
	DUP10:  "DUP10",
	DUP11:  "DUP11",
	DUP12:  "DUP12",
	DUP13:  "DUP13",
	DUP14:  "DUP14",
	DUP15:  "DUP15",
	DUP16:  "DUP16",
	SWAP4:  "SWAP4",
	SWAP5:  "SWAP5",
	SWAP6:  "SWAP6",
	SWAP7:  "SWAP7",
	SWAP8:  "SWAP8",
	SWAP9:  "SWAP9",
	SWAP10: "SWAP10",
	SWAP11: "SWAP11",
	SWAP12: "SWAP12",
	SWAP13: "SWAP13",
	SWAP14: "SWAP14",
	SWAP15: "SWAP15",
	SWAP16: "SWAP16",

	STOP: "STOP",
	PC:   "PC",

	MUL:        "MUL",
	DIV:        "DIV",
	SDIV:       "SDIV",
	MOD:        "MOD",
	SMOD:       "SMOD",
	ADDMOD:     "ADDMOD",
	MULMOD:     "MULMOD",
	EXP:        "EXP",
	SIGNEXTEND: "SIGNEXTEND",

	SHA3: "SHA3",

	SGT: "SGT",

	OR:   "OR",
	XOR:  "XOR",
	NOT:  "NOT",
	BYTE: "BYTE",
	SHL:  "SHL",
	SAR:  "SAR",

	MSTORE:  "MSTORE",
	MSTORE8: "MSTORE8",
	MLOAD:   "MLOAD",
	MSIZE:   "MSIZE",
	MCOPY:   "MCOPY",

	SLOAD:  "SLOAD",
	SSTORE: "SSTORE",
	TLOAD:  "TLOAD",
	TSTORE: "TSTORE",

	LOG0: "LOG0",
	LOG1: "LOG1",
	LOG2: "LOG2",
	LOG3: "LOG3",
	LOG4: "LOG4",

	ADDRESS:        "ADDRESS",
	BALANCE:        "BALANCE",
	ORIGIN:         "ORIGIN",
	CALLER:         "CALLER",
	CALLVALUE:      "CALLVALUE",
	CALLDATALOAD:   "CALLDATALOAD",
	CALLDATASIZE:   "CALLDATASIZE",
	CALLDATACOPY:   "CALLDATACOPY",
	CODESIZE:       "CODESIZE",
	CODECOPY:       "CODECOPY",
	GASPRICE:       "GASPRICE",
	EXTCODESIZE:    "EXTCODESIZE",
	EXTCODECOPY:    "EXTCODECOPY",
	RETURNDATASIZE: "RETURNDATASIZE",
	RETURNDATACOPY: "RETURNDATACOPY",
	EXTCODEHASH:    "EXTCODEHASH",
	CREATE:         "CREATE",
	CALL:           "CALL",
	CALLCODE:       "CALLCODE",
	DELEGATECALL:   "DELEGATECALL",
	CREATE2:        "CREATE2",
	STATICCALL:     "STATICCALL",
	SELFDESTRUCT:   "SELFDESTRUCT",

	BLOCKHASH:   "BLOCKHASH",
	COINBASE:    "COINBASE",
	TIMESTAMP:   "TIMESTAMP",
	NUMBER:      "NUMBER",
	PREVRANDAO:  "PREVRANDAO",
	GAS:         "GAS",
	GASLIMIT:    "GASLIMIT",
	CHAINID:     "CHAINID",
	SELFBALANCE: "SELFBALANCE",
	BASEFEE:     "BASEFEE",
	BLOBHASH:    "BLOBHASH",
	BLOBBASEFEE: "BLOBBASEFEE",

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

	DATA:    "DATA",
	NOOP:    "NOOP",
	INVALID: "INVALID",
}

func (o OpCode) String() string {
	str, found := op_to_string[o]
	if !found {
		return fmt.Sprintf("0x%04x", byte(o))
	}
	return str
}

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
