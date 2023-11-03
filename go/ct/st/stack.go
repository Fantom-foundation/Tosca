package st

import (
	"fmt"

	"golang.org/x/exp/slices"

	. "github.com/Fantom-foundation/Tosca/go/ct/common"
)

// Stack represent's the EVM's execution stack.
type Stack struct {
	stack []U256
}

// NewStack creates a new stack filled with the given values.
func NewStack(values ...U256) *Stack {
	return &Stack{values}
}

// NewStackWithSize creates a new stack with the given size, all elements
// initialized to zero.
func NewStackWithSize(size int) *Stack {
	return &Stack{make([]U256, size)}
}

// Clone creates an independent copy of the stack.
func (s *Stack) Clone() *Stack {
	return &Stack{slices.Clone(s.stack)}
}

func (s *Stack) Size() int {
	return len(s.stack)
}

// Get returns the value located the given index. The index must not be
// out-of-bounds.
func (s *Stack) Get(index int) U256 {
	return s.stack[s.Size()-index-1]
}

// Set places the given value at the given position on the stack. The index must
// not be out-of-bounds.
func (s *Stack) Set(index int, value U256) {
	s.stack[s.Size()-index-1] = value
}

// Push adds the given value to the top of the stack.
func (s *Stack) Push(value U256) {
	s.stack = append(s.stack, value)
}

// Pop removes the top most value from the stack and returns it. The stack must
// not be empty.
func (s *Stack) Pop() U256 {
	value := s.stack[s.Size()-1]
	s.stack = s.stack[:s.Size()-1]
	return value
}

func (a *Stack) Eq(b *Stack) bool {
	return slices.Equal(a.stack, b.stack)
}

func (a *Stack) Diff(b *Stack) (res []string) {
	if a.Size() != b.Size() {
		res = append(res, fmt.Sprintf("Different stack size: %v vs %v", a.Size(), b.Size()))
		return
	}
	for i := 0; i < a.Size(); i++ {
		if aValue, bValue := a.Get(i), b.Get(i); !aValue.Eq(bValue) {
			res = append(res, fmt.Sprintf("Different stack value at position %d:\n    %v\n    vs\n    %v", i, aValue, bValue))
		}
	}
	return
}
