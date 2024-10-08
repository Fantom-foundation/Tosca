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
	"fmt"

	"github.com/Fantom-foundation/Tosca/go/tosca/vm"
)

type OpCode uint16

// opCodeMask defines the relevant trailing bits of an OpCode. Any two OpCodes
// with the same value when masked with opCodeMask are considered equal.
//
// The motivation for this is that the long-form EVM has a number of OpCodes
// that are not part of the original EVM. For those, values beyond the range
// [0-255] of the EVM's single-byte OpCodes are used. To that end, the OpCode
// data type in the LFVM is increased to 16 bits. However, in several places
// maps from LFVM OpCodes to properties are required to provide efficient
// lookup tables for properties. To avoid the need to maintain tables of
// 2^16 entries, the number of relevant bits is reduced to 9. Any leading bits
// are ignored when comparing OpCodes.
const opCodeMask = 0x1ff

// numOpCodes is the maximum number of OpCodes that can be defined.
const numOpCodes = opCodeMask + 1

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
	// long-form EVM special instructions

	// JUMP_TO is a special instruction that is used to jump to the end of the
	// current basic block.
	//
	// Since due to the usage of immediate arguments in instructions like PUSH2
	// the code size of basic blocks can shrink compared to the original EVM,
	// gaps can appear between the end of a basic block and the beginning of the
	// next one indicated by a JUMPDEST instruction. Since all JUMPDEST
	// instructions have to remain at the same position in the code as in the
	// original EVM code, since jump-destinations of JUMP and JUMPI  operations
	// are computed dynamically, these gaps have to be filled with NOOP
	// instructions. To avoid having to process long sequences of NOOPs,
	// JUMP_TO instructions are used to skip them in a single step.
	//
	// The following restrictions are imposed on JUMP_TO instructions:
	//  - they must target the immediate succeeding JUMPDEST instruction
	//  - all instructions between the JUMP_TO and the JUMPDEST must be NOOPs
	//
	// These restrictions are enforced during the EVM to LFVM code conversion.
	JUMP_TO OpCode = iota + 0x100

	// NOOP is a special instruction that does nothing. It is used as a filler
	// instruction to pad basic blocks to the correct size.
	NOOP

	// DATA is a special instruction that is used to extend the size of OpCodes
	// that require more than the available 2-byte immediate arguments.
	// For instance, [PUSH4, 1, 2, 3, 4] in the original EVM code gets converted
	// to [(PUSH4, 1<<8 | 2),(DATA, 3<<8 | 4)].
	// Since DATA is marked explicitly as such, jump-destination checks can be
	// conducted in O(1) by checking the OpCode of an instruction. In the
	// implicit data encoding of EVM byte code, this would require a linear
	// search (which could be cached to amortize costs).
	DATA

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

	// _highestOpCode is an alias for the OpCode with the highest defined
	// numeric value. It is only intended to be used in the unit tests
	// associated to this OpCode definition file to verify that the OpCode
	// bit mask limit has not been exceeded.
	_highestOpCode = PUSH1_PUSH1_PUSH1_SHL_SUB
)

var toString = map[OpCode]string{
	DATA:    "DATA",
	NOOP:    "NOOP",
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
}

// String returns the string representation of the OpCode.
func (o OpCode) String() string {
	if o <= 0xFF {
		return vm.OpCode(o).String()
	}

	if str, ok := toString[o]; ok {
		return str
	}
	return fmt.Sprintf("op(0x%04X)", int16(o))
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
	}
	if o.isSuperInstruction() {
		for _, subOp := range o.decompose() {
			if subOp.HasArgument() {
				return true
			}
		}
	}
	return false
}

func (o OpCode) isBaseInstruction() bool {
	return o < 0x100
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

// opCodePropertyMap is a generic property map for precomputed values.
// Its purpose is to provide a precomputed lookup table for OpCode properties
// that can be generated from a function that takes an OpCode as input.
// Using this type hides internal details of the opcode implementation.
type opCodePropertyMap[T any] struct {
	lookup [numOpCodes]T
}

// newOpCodePropertyMap creates a new OpCode property map.
// The property function shall be resilient to undefined OpCode values, and not
// panic. The zero values or a sentinel value shall be used in such cases.
func newOpCodePropertyMap[T any](property func(op OpCode) T) opCodePropertyMap[T] {
	lookup := [numOpCodes]T{}
	for i := 0; i < numOpCodes; i++ {
		lookup[i] = property(OpCode(i))
	}
	return opCodePropertyMap[T]{lookup}
}

func (p *opCodePropertyMap[T]) get(op OpCode) T {
	// Index may be out of bounds. Nevertheless, bounds check carry a performance
	// penalty. If the property map is initialized correctly, the index will be
	// within bounds.
	return p.lookup[op&opCodeMask]
}
