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
	BYTE       OpCode = 0x1A
	SHL        OpCode = 0x1B
	SHR        OpCode = 0x1C
	SAR        OpCode = 0x1D
	POP        OpCode = 0x50
	JUMP       OpCode = 0x56
	JUMPI      OpCode = 0x57
	JUMPDEST   OpCode = 0x5B
	PUSH1      OpCode = 0x60
	PUSH2      OpCode = 0x61
	PUSH3      OpCode = 0x62
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
		BYTE,
		SHL,
		SHR,
		SAR,
		POP,
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
	case INVALID:
		return "INVALID"
	default:
		return fmt.Sprintf("op(%d)", op)
	}
}
