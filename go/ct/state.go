package ct

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/holiman/uint256"
)

type StatusCode int

const (
	Running     StatusCode = iota // still running
	Stopped                       // stopped execution successfully
	Returned                      // finished successfully
	Reverted                      // finished with revert signal
	Failed                        // failed (for any reason)
	numStatuses                   // not an actual status
)

func (s StatusCode) String() string {
	switch s {
	case Running:
		return "running"
	case Stopped:
		return "stopped"
	case Returned:
		return "returned"
	case Reverted:
		return "reverted"
	case Failed:
		return "failed"
	}
	return "?"
}

type State struct {
	Status StatusCode
	Code   []byte
	Pc     uint16
	Gas    uint64
	Stack  Stack
}

// TODO: test this
func (s *State) IsCode(position int) bool {
	if position >= len(s.Code) {
		return false
	}
	return s.GetNextCodePosition(position) == position
}

func (s *State) GetNextCodePosition(start int) int {
	if start >= len(s.Code) {
		return 0
	}
	i := 0
	for ; i < start; i++ {
		cur := s.Code[i]
		if byte(PUSH1) <= cur && cur <= byte(PUSH32) {
			i = i + int(cur-byte(PUSH1)+1)
		}
	}
	return i
}

func (s *State) GetNextDataPosition(start int) (position int, found bool) {
	if start >= len(s.Code) {
		start = 0
	}
	i := 0
	for ; i < start; i++ {
		cur := s.Code[i]
		if byte(PUSH1) <= cur && cur <= byte(PUSH32) {
			i = i + int(cur-byte(PUSH1)+1)
		}
	}
	if i > start {
		return start, true
	}
	// Keep searching for next data section.
	for ; i < len(s.Code); i++ {
		cur := s.Code[i]
		if byte(PUSH1) <= cur && cur <= byte(PUSH32) {
			return i + 1, true
		}
	}
	return 0, false
}

func (s *State) Equal(other *State) bool {
	return reflect.DeepEqual(s, other)
}

func (s *State) Clone() *State {
	res := *s
	res.Code = make([]byte, len(s.Code))
	copy(res.Code, s.Code)
	res.Stack = s.Stack.Clone()
	return &res
}

func (s *State) String() string {
	builder := strings.Builder{}
	builder.WriteString("{\n")
	builder.WriteString(fmt.Sprintf("\tStatus: %v,\n", s.Status))
	builder.WriteString(fmt.Sprintf("\tPc: %d", s.Pc))
	if !s.IsCode(int(s.Pc)) {
		builder.WriteString(" (points to data)\n")
	} else if s.Pc < uint16(len(s.Code)) {
		builder.WriteString(fmt.Sprintf(" (operation: %v)\n", OpCode(s.Code[s.Pc])))
	} else {
		builder.WriteString(" (out of bound)\n")

	}
	builder.WriteString(fmt.Sprintf("\tGas: %d,\n", s.Gas))
	if len(s.Code) > 20 {
		builder.WriteString(fmt.Sprintf("\tCode: %x...\n", s.Code[:20]))
	} else {
		builder.WriteString(fmt.Sprintf("\tCode: %x\n", s.Code))
	}

	size := s.Stack.Size()
	builder.WriteString(fmt.Sprintf("\tStack: %d elements\n", size))
	for i := 0; i < size && i < 5; i++ {
		value := s.Stack.Get(i)
		builder.WriteString(fmt.Sprintf("\t\t%5d: [%016x %016x %016x %016x]\n", i, value[3], value[2], value[1], value[0]))
	}
	if size > 5 {
		builder.WriteString("\t\t    ...\n")
	}
	builder.WriteString("}")
	return builder.String()
}

type Stack struct {
	stack []uint256.Int
}

func NewStack(values []uint256.Int) Stack {
	return Stack{values}
}

func (s *Stack) Clone() Stack {
	res := make([]uint256.Int, len(s.stack))
	copy(res, s.stack)
	return Stack{res}
}

func (s *Stack) Size() int {
	return len(s.stack)
}

func (s *Stack) Get(i int) uint256.Int {
	return s.stack[len(s.stack)-i-1]
}

func (s *Stack) Set(i int, value uint256.Int) {
	s.stack[len(s.stack)-i-1] = value
}

func (s *Stack) Pop() uint256.Int {
	res := s.stack[len(s.stack)-1]
	s.stack = s.stack[:len(s.stack)-1]
	return res
}

func (s *Stack) Push(value uint256.Int) {
	s.stack = append(s.stack, value)
}
