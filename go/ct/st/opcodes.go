package st

import "fmt"

type OpCode byte

const (
	STOP     OpCode = 0x00
	ADD      OpCode = 0x01
	JUMP     OpCode = 0x56
	JUMPDEST OpCode = 0x5B
	PUSH1    OpCode = 0x60
	PUSH2    OpCode = 0x61
	PUSH3    OpCode = 0x62
	PUSH31   OpCode = 0x7E
	PUSH32   OpCode = 0x7F
	INVALID  OpCode = 0xFE
)

func (op OpCode) String() string {
	switch op {
	case STOP:
		return "STOP"
	case ADD:
		return "ADD"
	case JUMP:
		return "JUMP"
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
