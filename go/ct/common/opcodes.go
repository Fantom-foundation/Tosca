package common

import (
	"fmt"

	"golang.org/x/exp/slices"
)

type OpCode byte

const (
	STOP       OpCode = 0x00
	ADD        OpCode = 0x01
	MUL        OpCode = 0x02
	SUB        OpCode = 0x03
	DIV        OpCode = 0x04
	SDIV       OpCode = 0x05
	MOD        OpCode = 0x06
	SMOD       OpCode = 0x07
	ADDMOD     OpCode = 0x08
	MULMOD     OpCode = 0x09
	EXP        OpCode = 0x0A
	SIGNEXTEND OpCode = 0x0B
	LT         OpCode = 0x10
	GT         OpCode = 0x11
	SLT        OpCode = 0x12
	SGT        OpCode = 0x13
	EQ         OpCode = 0x14
	ISZERO     OpCode = 0x15
	AND        OpCode = 0x16
	OR         OpCode = 0x17
	XOR        OpCode = 0x18
	NOT        OpCode = 0x19
	SHA3       OpCode = 0x20
	BYTE       OpCode = 0x1A
	SHL        OpCode = 0x1B
	SHR        OpCode = 0x1C
	SAR        OpCode = 0x1D
	POP        OpCode = 0x50
	MLOAD      OpCode = 0x51
	MSTORE     OpCode = 0x52
	MSTORE8    OpCode = 0x53
	SLOAD      OpCode = 0x54
	SSTORE     OpCode = 0x55
	JUMP       OpCode = 0x56
	JUMPI      OpCode = 0x57
	JUMPDEST   OpCode = 0x5B
	PUSH1      OpCode = 0x60
	PUSH2      OpCode = 0x61
	PUSH3      OpCode = 0x62
	PUSH4      OpCode = 0x63
	PUSH5      OpCode = 0x64
	PUSH6      OpCode = 0x65
	PUSH7      OpCode = 0x66
	PUSH8      OpCode = 0x67
	PUSH9      OpCode = 0x68
	PUSH10     OpCode = 0x69
	PUSH11     OpCode = 0x6A
	PUSH12     OpCode = 0x6B
	PUSH13     OpCode = 0x6C
	PUSH14     OpCode = 0x6D
	PUSH15     OpCode = 0x6E
	PUSH16     OpCode = 0x6F
	PUSH17     OpCode = 0x70
	PUSH18     OpCode = 0x71
	PUSH19     OpCode = 0x72
	PUSH20     OpCode = 0x73
	PUSH21     OpCode = 0x74
	PUSH22     OpCode = 0x75
	PUSH23     OpCode = 0x76
	PUSH24     OpCode = 0x77
	PUSH25     OpCode = 0x78
	PUSH26     OpCode = 0x79
	PUSH27     OpCode = 0x7A
	PUSH28     OpCode = 0x7B
	PUSH29     OpCode = 0x7C
	PUSH30     OpCode = 0x7D
	PUSH31     OpCode = 0x7E
	PUSH32     OpCode = 0x7F
	DUP1       OpCode = 0x80
	DUP2       OpCode = 0x81
	DUP3       OpCode = 0x82
	DUP4       OpCode = 0x83
	DUP5       OpCode = 0x84
	DUP6       OpCode = 0x85
	DUP7       OpCode = 0x86
	DUP8       OpCode = 0x87
	DUP9       OpCode = 0x88
	DUP10      OpCode = 0x89
	DUP11      OpCode = 0x8A
	DUP12      OpCode = 0x8B
	DUP13      OpCode = 0x8C
	DUP14      OpCode = 0x8D
	DUP15      OpCode = 0x8E
	DUP16      OpCode = 0x8F
	SWAP1      OpCode = 0x90
	SWAP2      OpCode = 0x91
	SWAP3      OpCode = 0x92
	SWAP4      OpCode = 0x93
	SWAP5      OpCode = 0x94
	SWAP6      OpCode = 0x95
	SWAP7      OpCode = 0x96
	SWAP8      OpCode = 0x97
	SWAP9      OpCode = 0x98
	SWAP10     OpCode = 0x99
	SWAP11     OpCode = 0x9A
	SWAP12     OpCode = 0x9B
	SWAP13     OpCode = 0x9C
	SWAP14     OpCode = 0x9D
	SWAP15     OpCode = 0x9E
	SWAP16     OpCode = 0x9F
	LOG0       OpCode = 0xA0
	LOG1       OpCode = 0xA1
	LOG2       OpCode = 0xA2
	LOG3       OpCode = 0xA3
	LOG4       OpCode = 0xA4
	INVALID    OpCode = 0xFE
)

func (op OpCode) Width() int {
	if PUSH1 <= op && op <= PUSH32 {
		return int(op-PUSH1) + 2
	} else {
		return 1
	}
}

// OpCodesNoPush returns a slice of valid op codes, but no PUSH instruction.
func ValidOpCodesNoPush() []OpCode {
	return slices.Clone([]OpCode{
		STOP,
		ADD,
		MUL,
		SUB,
		DIV,
		SDIV,
		MOD,
		SMOD,
		ADDMOD,
		MULMOD,
		EXP,
		SIGNEXTEND,
		LT,
		GT,
		SLT,
		SGT,
		EQ,
		ISZERO,
		AND,
		OR,
		XOR,
		NOT,
		SHA3,
		BYTE,
		SHL,
		SHR,
		SAR,
		POP,
		MLOAD,
		MSTORE,
		MSTORE8,
		SLOAD,
		SSTORE,
		JUMP,
		JUMPI,
		JUMPDEST,
		DUP1,
		DUP2,
		DUP3,
		DUP4,
		DUP5,
		DUP6,
		DUP7,
		DUP8,
		DUP9,
		DUP10,
		DUP11,
		DUP12,
		DUP13,
		DUP14,
		DUP15,
		DUP16,
		SWAP1,
		SWAP2,
		SWAP3,
		SWAP4,
		SWAP5,
		SWAP6,
		SWAP7,
		SWAP8,
		SWAP9,
		SWAP10,
		SWAP11,
		SWAP12,
		SWAP13,
		SWAP14,
		SWAP15,
		SWAP16,
		LOG0,
		LOG1,
		LOG2,
		LOG3,
		LOG4,
	})
}

func (op OpCode) String() string {
	switch op {
	case STOP:
		return "STOP"
	case ADD:
		return "ADD"
	case MUL:
		return "MUL"
	case SUB:
		return "SUB"
	case DIV:
		return "DIV"
	case SDIV:
		return "SDIV"
	case MOD:
		return "MOD"
	case SMOD:
		return "SMOD"
	case ADDMOD:
		return "ADDMOD"
	case MULMOD:
		return "MULMOD"
	case EXP:
		return "EXP"
	case SIGNEXTEND:
		return "SIGNEXTEND"
	case LT:
		return "LT"
	case GT:
		return "GT"
	case SLT:
		return "SLT"
	case SGT:
		return "SGT"
	case EQ:
		return "EQ"
	case ISZERO:
		return "ISZERO"
	case AND:
		return "AND"
	case OR:
		return "OR"
	case XOR:
		return "XOR"
	case NOT:
		return "NOT"
	case SHA3:
		return "SHA3"
	case BYTE:
		return "BYTE"
	case SHL:
		return "SHL"
	case SHR:
		return "SHR"
	case SAR:
		return "SAR"
	case POP:
		return "POP"
	case MLOAD:
		return "MLOAD"
	case MSTORE:
		return "MSTORE"
	case MSTORE8:
		return "MSTORE8"
	case SLOAD:
		return "SLOAD"
	case SSTORE:
		return "SSTORE"
	case JUMP:
		return "JUMP"
	case JUMPI:
		return "JUMPI"
	case JUMPDEST:
		return "JUMPDEST"
	case PUSH1:
		return "PUSH1"
	case PUSH2:
		return "PUSH2"
	case PUSH3:
		return "PUSH3"
	case PUSH4:
		return "PUSH4"
	case PUSH5:
		return "PUSH5"
	case PUSH6:
		return "PUSH6"
	case PUSH7:
		return "PUSH7"
	case PUSH8:
		return "PUSH8"
	case PUSH9:
		return "PUSH9"
	case PUSH10:
		return "PUSH10"
	case PUSH11:
		return "PUSH11"
	case PUSH12:
		return "PUSH12"
	case PUSH13:
		return "PUSH13"
	case PUSH14:
		return "PUSH14"
	case PUSH15:
		return "PUSH15"
	case PUSH16:
		return "PUSH16"
	case PUSH17:
		return "PUSH17"
	case PUSH18:
		return "PUSH18"
	case PUSH19:
		return "PUSH19"
	case PUSH20:
		return "PUSH20"
	case PUSH21:
		return "PUSH21"
	case PUSH22:
		return "PUSH22"
	case PUSH23:
		return "PUSH23"
	case PUSH24:
		return "PUSH24"
	case PUSH25:
		return "PUSH25"
	case PUSH26:
		return "PUSH26"
	case PUSH27:
		return "PUSH27"
	case PUSH28:
		return "PUSH28"
	case PUSH29:
		return "PUSH29"
	case PUSH30:
		return "PUSH30"
	case PUSH31:
		return "PUSH31"
	case PUSH32:
		return "PUSH32"
	case DUP1:
		return "DUP1"
	case DUP2:
		return "DUP2"
	case DUP3:
		return "DUP3"
	case DUP4:
		return "DUP4"
	case DUP5:
		return "DUP5"
	case DUP6:
		return "DUP6"
	case DUP7:
		return "DUP7"
	case DUP8:
		return "DUP8"
	case DUP9:
		return "DUP9"
	case DUP10:
		return "DUP10"
	case DUP11:
		return "DUP11"
	case DUP12:
		return "DUP12"
	case DUP13:
		return "DUP13"
	case DUP14:
		return "DUP14"
	case DUP15:
		return "DUP15"
	case DUP16:
		return "DUP16"
	case SWAP1:
		return "SWAP1"
	case SWAP2:
		return "SWAP2"
	case SWAP3:
		return "SWAP3"
	case SWAP4:
		return "SWAP4"
	case SWAP5:
		return "SWAP5"
	case SWAP6:
		return "SWAP6"
	case SWAP7:
		return "SWAP7"
	case SWAP8:
		return "SWAP8"
	case SWAP9:
		return "SWAP9"
	case SWAP10:
		return "SWAP10"
	case SWAP11:
		return "SWAP11"
	case SWAP12:
		return "SWAP12"
	case SWAP13:
		return "SWAP13"
	case SWAP14:
		return "SWAP14"
	case SWAP15:
		return "SWAP15"
	case SWAP16:
		return "SWAP16"
	case LOG0:
		return "LOG0"
	case LOG1:
		return "LOG1"
	case LOG2:
		return "LOG2"
	case LOG3:
		return "LOG3"
	case LOG4:
		return "LOG4"
	case INVALID:
		return "INVALID"
	default:
		return fmt.Sprintf("op(%d)", op)
	}
}
