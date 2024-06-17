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
	"testing"

	. "github.com/Fantom-foundation/Tosca/go/ct/common"
	"pgregory.net/rand"
)

func TestStack_NewStack(t *testing.T) {
	stack := NewStack()
	if want, got := 0, stack.Size(); want != got {
		t.Errorf("unexpected stack size, want %v, got %v", want, got)
	}

	stack = NewStack(NewU256(1))
	if want, got := 1, stack.Size(); want != got {
		t.Errorf("unexpected stack size, want %v, got %v", want, got)
	}
}

func TestStack_NewStackWithSize(t *testing.T) {
	stack := NewStackWithSize(5)
	if want, got := 5, stack.Size(); want != got {
		t.Errorf("unexpected stack size, want %v, got %v", want, got)
	}
	for i := 0; i < stack.Size(); i++ {
		if !stack.Get(i).Eq(NewU256(0)) {
			t.Errorf("unexpected non-zero value at index %d", i)
		}
	}
}

func TestStack_Clone(t *testing.T) {
	stack := NewStack(NewU256(42))
	clone := stack.Clone()

	if stack.Size() != clone.Size() {
		t.Error("Clone does not have the same size")
	}

	stack.Push(NewU256(21))
	if clone.Size() != 1 {
		t.Error("Clone is not independent from original")
	}

	stack.Set(1, NewU256(43))
	if !clone.Get(0).Eq(NewU256(42)) {
		t.Error("Clone is not independent from original")
	}
}

func TestStack_Get(t *testing.T) {
	stack := NewStack(NewU256(1), NewU256(2), NewU256(3))
	if want, got := uint64(3), stack.Get(0).Uint64(); want != got {
		t.Errorf("unexpected stack value at position 0, want %v, got %v", want, got)
	}
	if want, got := uint64(2), stack.Get(1).Uint64(); want != got {
		t.Errorf("unexpected stack value at position 1, want %v, got %v", want, got)
	}
	if want, got := uint64(1), stack.Get(2).Uint64(); want != got {
		t.Errorf("unexpected stack value at position 2, want %v, got %v", want, got)
	}
}

func TestStack_Set(t *testing.T) {
	stack := NewStack(NewU256(2))
	stack.Set(0, NewU256(4))
	if want, got := uint64(4), stack.Get(0).Uint64(); want != got {
		t.Errorf("unexpected stack value at position 0, want %v, got %v", want, got)
	}
}

func TestStack_Push(t *testing.T) {
	stack := NewStack()

	stack.Push(NewU256(42))
	if want, got := 1, stack.Size(); want != got {
		t.Errorf("unexpected stack size, want %v, got %v", want, got)
	}
	if want, got := uint64(42), stack.Get(0).Uint64(); want != got {
		t.Errorf("unexpected stack value at position 0, want %v, got %v", want, got)
	}

	stack.Push(NewU256(16))
	if want, got := 2, stack.Size(); want != got {
		t.Errorf("unexpected stack size, want %v, got %v", want, got)
	}
	if want, got := uint64(16), stack.Get(0).Uint64(); want != got {
		t.Errorf("unexpected stack value at position 0, want %v, got %v", want, got)
	}
	if want, got := uint64(42), stack.Get(1).Uint64(); want != got {
		t.Errorf("unexpected stack value at position 1, want %v, got %v", want, got)
	}
}

func TestStack_Pop(t *testing.T) {
	stack := NewStack(NewU256(1), NewU256(2))

	value := stack.Pop().Uint64()
	if value != 2 {
		t.Errorf("unexpected value popped, want 2, got %v", value)
	}
	if want, got := 1, stack.Size(); want != got {
		t.Errorf("unexpected stack size, want %v, got %v", want, got)
	}

	value = stack.Pop().Uint64()
	if value != 1 {
		t.Errorf("unexpected value popped, want 1, got %v", value)
	}
	if want, got := 0, stack.Size(); want != got {
		t.Errorf("unexpected stack size, want %v, got %v", want, got)
	}
}

func TestStack_Eq(t *testing.T) {
	stack1 := NewStack(NewU256(1), NewU256(2))
	stack2 := NewStack(NewU256(1), NewU256(2))
	if !stack1.Eq(stack2) {
		t.Errorf("unexpected stack inequality %v vs. %v", stack1.stack, stack2.stack)
	}

	stack2.Set(0, NewU256(42))
	if stack1.Eq(stack2) {
		t.Errorf("unexpected stack equality %v vs. %v", stack1.stack, stack2.stack)
	}

	stack2.Pop()
	if stack1.Eq(stack2) {
		t.Errorf("unexpected stack equality %v vs. %v", stack1.stack, stack2.stack)
	}
}

func TestStack_NewStackFactoriesReturnStacksWithCorrectSize(t *testing.T) {
	stacks := []struct {
		stack *Stack
		size  int
	}{
		{NewStack(), 0},
		{NewStackWithSize(42), 42},
		{NewStack(NewU256(42), NewU256(42), NewU256(42), NewU256(42)), 4},
	}

	for _, input := range stacks {
		stack := input.stack
		if stack == nil {
			t.Fatal("No stack was returned by function")
		}

		if stack.Size() != input.size {
			t.Errorf("Mismatching stack size, wanted %d but got %d", input.size, stack.Size())
		}

		stack.Push(NewU256(42))
		stack.Push(NewU256(42))
		stack.Release()

		new := NewStack()
		if new.Size() != 0 {
			t.Errorf("Mismatching stack size, wanted 0 but got %d", stack.Size())
		}
	}
}

func BenchmarkStack_StackCreation(b *testing.B) {

	rnd := rand.New(0)
	for i := 0; i < b.N; i++ {
		stack := NewStackWithSize(1 + rnd.Intn(1023))
		if i%2 == 0 {
			stack.Pop()
		} else {
			stack.Push(NewU256(42))
		}
		stack.Release()
	}
}
