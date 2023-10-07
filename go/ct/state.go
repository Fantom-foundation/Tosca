package ct

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/holiman/uint256"
	"golang.org/x/exp/slices"
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
	isCode []bool
	Pc     uint16
	Gas    uint64
	Stack  Stack
}

func (s *State) setCodeMask() {
	if len(s.isCode) == len(s.Code) {
		return
	}
	s.isCode = make([]bool, len(s.Code))
	for i := 0; i < len(s.Code); i++ {
		s.isCode[i] = true
		op := s.Code[i]
		if byte(PUSH1) <= op && op <= byte(PUSH32) {
			i = i + int(op-byte(PUSH1)+1)
		}
	}
}

// TODO: test this
func (s *State) IsCode(position int) bool {
	s.setCodeMask()
	return position >= 0 && position < len(s.isCode) && s.isCode[position]
}

func (s *State) GetNextCodePosition(start int) int {
	if start >= len(s.Code) {
		return 0
	}
	s.setCodeMask()
	for i := start; i < len(s.isCode); i++ {
		if s.isCode[i] {
			return i
		}
	}
	return 0
}

func (s *State) GetNextDataPosition(start int) (position int, found bool) {
	if start >= len(s.Code) {
		start = 0
	}
	s.setCodeMask()
	for i := start; i < len(s.isCode); i++ {
		if !s.isCode[i] {
			return i, true
		}
	}
	for i := 0; i < start; i++ {
		if !s.isCode[i] {
			return i, true
		}
	}
	return 0, false
}

func (s *State) Equal(other *State) bool {
	if s.Status != other.Status {
		return false
	}
	// All failed states are the same.
	if s.Status == Failed {
		return true
	}
	if s.Gas != other.Gas {
		return false
	}
	if s.Pc != other.Pc {
		return false
	}
	if !s.Stack.Equal(&other.Stack) {
		return false
	}
	return bytes.Equal(s.Code, other.Code)
}

func (s *State) Clone() *State {
	res := *s
	res.Code = make([]byte, len(s.Code))
	copy(res.Code, s.Code)
	res.isCode = make([]bool, len(s.isCode))
	copy(res.isCode, s.isCode)
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
		builder.WriteString(fmt.Sprintf("\tCode: %x... (size: %d)\n", s.Code[:20], len(s.Code)))
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

func (s *Stack) Equal(other *Stack) bool {
	return slices.Equal(s.stack, other.stack)
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

func Diff(a *State, b *State) []string {
	res := []string{}

	if a.Status != b.Status {
		res = append(res, fmt.Sprintf("Different status: %v vs %v", a.Status, b.Status))
	}

	if a.Gas != b.Gas {
		res = append(res, fmt.Sprintf("Different gas: %v vs %v", a.Gas, b.Gas))
	}

	if a.Pc != b.Pc {
		res = append(res, fmt.Sprintf("Different pc: %v vs %v", a.Pc, b.Pc))
	}

	if !bytes.Equal(a.Code, b.Code) {
		res = append(res, "Different code!")
	}

	if as, bs := a.Stack.Size(), b.Stack.Size(); as != bs {
		res = append(res, fmt.Sprintf("Different stack size: %v vs %v", as, bs))
	} else {
		for i := 0; i < as; i++ {
			if av, bv := a.Stack.Get(i), b.Stack.Get(i); !av.Eq(&bv) {
				res = append(res, fmt.Sprintf("Different stack value at position %d: %x vs %x", i, av, bv))
			}
		}
	}

	return res
}
