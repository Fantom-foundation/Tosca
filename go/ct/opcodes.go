package ct

import "fmt"

type OpCode byte

const (
	STOP OpCode = 0
	POP         = 0x50
	ADD         = 0x01
)

func (op OpCode) String() string {
	switch op {
	case STOP:
		return "STOP"
	case ADD:
		return "ADD"
	case POP:
		return "POP"
	default:
		return fmt.Sprintf("op(%d)", op)
	}
}
