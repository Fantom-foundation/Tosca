package ct

import "fmt"

type OpCode byte

const (
	STOP   OpCode = 0
	ADD    OpCode = 0x01
	POP    OpCode = 0x50
	PUSH1  OpCode = 0x60
	PUSH2  OpCode = 0x61
	PUSH16 OpCode = 0x6F
	PUSH32 OpCode = 0x7F
)

func (op OpCode) String() string {
	switch op {
	case STOP:
		return "STOP"
	case ADD:
		return "ADD"
	case POP:
		return "POP"
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
