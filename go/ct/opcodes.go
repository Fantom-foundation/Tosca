package ct

import "fmt"

type OpCode byte

const (
	STOP     OpCode = 0
	ADD      OpCode = 0x01
	LT       OpCode = 0x10
	EQ       OpCode = 0x14
	AND      OpCode = 0x16
	OR       OpCode = 0x17
	NOT      OpCode = 0x19
	POP      OpCode = 0x50
	SLOAD    OpCode = 0x54
	SSTORE   OpCode = 0x55
	JUMP     OpCode = 0x56
	JUMPI    OpCode = 0x57
	JUMPDEST OpCode = 0x5B
	PUSH1    OpCode = 0x60
	PUSH2    OpCode = 0x61
	PUSH16   OpCode = 0x6F
	PUSH32   OpCode = 0x7F
)

func (op OpCode) String() string {
	switch op {
	case STOP:
		return "STOP"
	case ADD:
		return "ADD"
	case LT:
		return "LT"
	case EQ:
		return "EQ"
	case AND:
		return "AND"
	case OR:
		return "OR"
	case NOT:
		return "NOT"
	case POP:
		return "POP"
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
	case PUSH16:
		return "PUSH16"
	case PUSH32:
		return "PUSH32"
	default:
		return fmt.Sprintf("op(%d)", op)
	}
}
