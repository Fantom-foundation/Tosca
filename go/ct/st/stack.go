package st

import (
	"fmt"
	"sync"

	"golang.org/x/exp/slices"

	. "github.com/Fantom-foundation/Tosca/go/ct/common"
)

const MaxStackSize = 1024

// https://blog.mike.norgate.xyz/unlocking-go-slice-performance-navigating-sync-pool-for-enhanced-efficiency-7cb63b0b453e
var stackPool = sync.Pool{
	New: func() interface{} {
		s := make([]U256, 0, MaxStackSize)
		return &s
	},
}

func getStackData() []U256 {
	ptr := stackPool.Get().(*[]U256)
	return *ptr
}

func returnStack(data []U256) {
	data = data[0:0]
	stackPool.Put(&data)
}

// Stack represents the EVM's execution stack.
type Stack struct {
	stack []U256
}

// NewStack creates a new stack filled with the given values.
func NewStack(values ...U256) *Stack {
	data := getStackData()
	data = append(data, values...)
	return &Stack{data}
}

// NewStackWithSize creates a new stack with the given size, all elements
// initialized to zero.
func NewStackWithSize(size int) *Stack {
	data := getStackData()
	data = data[0:size]
	return &Stack{data}
}

func (s *Stack) Release() {
	returnStack(s.stack)
	s.stack = nil
}

// Clone creates an independent copy of the stack.
func (s *Stack) Clone() *Stack {
	return &Stack{slices.Clone(s.stack)}
}

// Size returns the number of elements on the stack.
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

// Eq returns true if the two stacks are equal.
func (a *Stack) Eq(b *Stack) bool {
	return slices.Equal(a.stack, b.stack)
}

// Diff returns a list of differences between the two stacks.
func (a *Stack) Diff(b *Stack) (res []string) {
	if a.Size() != b.Size() {
		res = append(res, fmt.Sprintf("Different stack size: %v vs %v", a.Size(), b.Size()))
		return
	}
	for i := 0; i < a.Size(); i++ {
		if aValue, bValue := a.Get(i), b.Get(i); !aValue.Eq(bValue) {
			res = append(res, fmt.Sprintf("Different stack value at position %d:\n    %v\n    vs\n    %v\n", i, aValue, bValue))
		}
	}
	return
}
