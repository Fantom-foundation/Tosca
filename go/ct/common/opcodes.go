package common

import "fmt"

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
	JUMP       OpCode = 0x56
	JUMPI      OpCode = 0x57
	JUMPDEST   OpCode = 0x5B
	PUSH1      OpCode = 0x60
	PUSH2      OpCode = 0x61
	PUSH3      OpCode = 0x62
	PUSH31     OpCode = 0x7E
	PUSH32     OpCode = 0x7F
	INVALID    OpCode = 0xFE
)

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
	case INVALID:
		return "INVALID"
	default:
		return fmt.Sprintf("op(%d)", op)
	}
}
