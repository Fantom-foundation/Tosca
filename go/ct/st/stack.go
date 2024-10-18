// Copyright (c) 2024 Fantom Foundation
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at fantom.foundation/bsl11.
//
// Change Date: 2028-4-16
//
// On the date above, in accordance with the Business Source License, use of
// this software will be governed by the GNU Lesser General Public License v3.

package st

import (
	"fmt"
	"sync"

	"golang.org/x/exp/slices"

	. "github.com/Fantom-foundation/Tosca/go/ct/common"
)

const MaxStackSize = 1024

// Stack represents the EVM's execution stack.
type Stack struct {
	stack []U256
}

var stackPool = sync.Pool{
	New: func() interface{} {
		stack := &Stack{
			stack: make([]U256, MaxStackSize),
		}
		return stack
	},
}

// NewStack returns a stack from the stack pool, all stacks are allocated with the maximal capacity
func NewStack(values ...U256) *Stack {
	stack := NewStackWithSize(len(values))
	stack.stack = stack.stack[:copy(stack.stack, values)]
	return stack
}

// NewStackWithSize returns a stack with the given size and unspecified content
func NewStackWithSize(size int) *Stack {
	stack := stackPool.Get().(*Stack)
	if MaxStackSize < size {
		panic("Warning: maximal stack size exceeded")
	}
	stack.stack = stack.stack[:size]
	return stack
}

// Clone creates an independent copy of the stack.
func (s *Stack) Clone() *Stack {
	clone := stackPool.Get().(*Stack)
	if cap(clone.stack) < s.Size() {
		clone.stack = make([]U256, s.Size())
	} else {
		clone.stack = clone.stack[:s.Size()]
	}
	copy(clone.stack, s.stack)
	return clone
}

func (s *Stack) Release() {
	stackPool.Put(s)
}

func (s *Stack) Resize(size int) {
	if MaxStackSize < size {
		panic("Warning: maximal stack size exceeded")
	}
	s.stack = s.stack[:size]
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
