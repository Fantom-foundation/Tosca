package ct

import "fmt"

type OpCode byte

const (
	STOP OpCode = 0
	POP         = 50
)

func (op OpCode) String() string {
	switch op {
	case STOP:
		return "STOP"
	case POP:
		return "POP"
	default:
		return fmt.Sprintf("op(%d)", op)
	}
}
