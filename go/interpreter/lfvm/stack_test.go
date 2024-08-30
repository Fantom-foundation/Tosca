// Copyright (c) 2024 Fantom Foundation
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at fantom.foundation/bsl11.
//
// Change Date: 2028-4-16
//
// On the date above, in accordance with the Business Source License, use of
// this software will be governed by the GNU Lesser General Public License v3.

package lfvm

import (
	"fmt"
	"strings"
	"sync"
	"testing"

	"github.com/holiman/uint256"
)

func TestStack_ZeroStackIsEmpty(t *testing.T) {
	var stack stack
	if want, got := 0, stack.len(); want != got {
		t.Errorf("expected stack to be empty, but got %d elements", got)
	}
}

func TestStack_pushAndPop_CanUseFullCapacity(t *testing.T) {
	var stack stack

	for i := 0; i < maxStackSize; i++ {
		if want, got := i, stack.len(); want != got {
			t.Errorf("expected stack to have %d elements, but got %d", want, got)
		}
		val := uint256.NewInt(uint64(i))
		stack.push(val)
	}

	if want, got := maxStackSize, stack.len(); want != got {
		t.Errorf("expected stack to have %d elements, but got %d", want, got)
	}

	for i := maxStackSize - 1; i >= 0; i-- {
		val := stack.pop()
		if want, got := uint256.NewInt(uint64(i)), val; want.Cmp(got) != 0 {
			t.Errorf("expected popped value to be %d, but got %d", want, got)
		}
		if want, got := i, stack.len(); want != got {
			t.Errorf("expected stack to have %d elements, but got %d", want, got)
		}
	}
}

func TestStack_push_AddsProvidedElementToStack(t *testing.T) {
	values := []*uint256.Int{
		uint256.NewInt(0),
		uint256.NewInt(1),
		new(uint256.Int).Lsh(uint256.NewInt(1), 64),
		new(uint256.Int).Lsh(uint256.NewInt(1), 128),
		new(uint256.Int).Lsh(uint256.NewInt(1), 192),
	}

	stack := NewStack()
	defer ReturnStack(stack)

	for _, val := range values {
		stack.push(val)
		if want, got := val, stack.peek(); want.Cmp(got) != 0 {
			t.Errorf("expected top element to be %d, but got %d", want, got)
		}
	}
}

func TestStack_pushUndefined_ResultCanBeUsedToManipulatePeek(t *testing.T) {
	values := []*uint256.Int{
		uint256.NewInt(0),
		uint256.NewInt(1),
		new(uint256.Int).Lsh(uint256.NewInt(1), 64),
		new(uint256.Int).Lsh(uint256.NewInt(1), 128),
		new(uint256.Int).Lsh(uint256.NewInt(1), 192),
	}

	stack := NewStack()
	defer ReturnStack(stack)

	for _, val := range values {
		peek := stack.pushUndefined()
		peek.Set(val)
		if want, got := val, stack.peek(); want.Cmp(got) != 0 {
			t.Errorf("expected top element to be %d, but got %d", want, got)
		}
	}
}

func TestStack_peekN_ObtainsNthElementFromTop(t *testing.T) {
	stack := NewStack()
	defer ReturnStack(stack)

	for i := 0; i < 10; i++ {
		val := uint256.NewInt(uint64(i))
		stack.push(val)
	}

	if want, got := stack.peek(), stack.peekN(0); want != got {
		t.Errorf("expected peekN(0) to be the same as peek(), but got %d and %d", want, got)
	}

	for i := 0; i < 10; i++ {
		want := uint256.NewInt(uint64(9 - i))
		got := stack.peekN(i)
		if want.Cmp(got) != 0 {
			t.Errorf("expected %d-th element from top to be %d, but got %d", i, want, got)
		}
	}
}

func TestStack_swap_ExchangesTopElementWithSelectedElement(t *testing.T) {
	// n => expected order after swap(n)
	tests := map[int][]uint64{
		1: {0, 1, 2, 3, 4},
		2: {1, 0, 2, 3, 4},
		3: {2, 1, 0, 3, 4},
		4: {3, 1, 2, 0, 4},
		5: {4, 1, 2, 3, 0},
	}

	for n, result := range tests {
		t.Run(fmt.Sprintf("swap%d", n), func(t *testing.T) {
			stack := NewStack()
			defer ReturnStack(stack)

			for i := 4; i >= 0; i-- {
				stack.push(uint256.NewInt(uint64(i)))
			}

			stack.swap(n)

			for i, want := range result {
				got := stack.peekN(i).Uint64()
				if want != got {
					t.Errorf("expected %d-th element to be %d, but got %d", i, want, got)
				}
			}
		})
	}
}

func TestStack_swap_WorksForAnyValueBiggerThanOne(t *testing.T) {
	for _, size := range []int{2, 128, maxStackSize - 1} {
		for i := 1; i < size; i++ {
			t.Run(fmt.Sprintf("size=%d_swap%d", size, i), func(t *testing.T) {
				stack := NewStack()
				defer ReturnStack(stack)

				for i := 0; i < size; i++ {
					stack.push(uint256.NewInt(uint64(i)))
				}

				want := stack.peekN(i - 1).Uint64()
				stack.swap(i)
				got := stack.peek().Uint64()

				if want != got {
					t.Errorf("expected top element to be %d, but got %d", want, got)
				}
			})
		}
	}
}

func TestStack_dup_DuplicatesSelectedElementFromStack(t *testing.T) {
	// n => expected content after dup(n)
	tests := map[int][]uint64{
		1: {0, 0, 1, 2, 3, 4},
		2: {1, 0, 1, 2, 3, 4},
		3: {2, 0, 1, 2, 3, 4},
		4: {3, 0, 1, 2, 3, 4},
		5: {4, 0, 1, 2, 3, 4},
	}

	for n, result := range tests {
		t.Run(fmt.Sprintf("dup%d", n), func(t *testing.T) {
			stack := NewStack()
			defer ReturnStack(stack)

			for i := 4; i >= 0; i-- {
				stack.push(uint256.NewInt(uint64(i)))
			}

			stack.dup(n)

			for i, want := range result {
				got := stack.peekN(i).Uint64()
				if want != got {
					t.Errorf("expected %d-th element to be %d, but got %d", i, want, got)
				}
			}
		})
	}
}

func TestStack_dup_WorksForAnyValueBiggerThanOne(t *testing.T) {
	for _, size := range []int{2, 128, maxStackSize - 1} {
		for i := 1; i < size; i++ {
			t.Run(fmt.Sprintf("size=%d_dup%d", size, i), func(t *testing.T) {
				stack := NewStack()
				defer ReturnStack(stack)

				for i := 0; i < size; i++ {
					stack.push(uint256.NewInt(uint64(i)))
				}

				want := stack.peekN(i - 1).Uint64()
				stack.dup(i)
				got := stack.peek().Uint64()

				if want != got {
					t.Errorf("expected top element to be %d, but got %d", want, got)
				}
			})
		}
	}
}

func TestStack_get_IndexesElementsBottomUp(t *testing.T) {
	stack := NewStack()
	defer ReturnStack(stack)

	for i := 0; i < maxStackSize; i++ {
		stack.push(uint256.NewInt(uint64(i)))
	}

	for i := 0; i < maxStackSize; i++ {
		want := uint256.NewInt(uint64(i))
		got := stack.get(i)
		if want.Cmp(got) != 0 {
			t.Errorf("expected %d-th element to be %d, but got %d", i, want, got)
		}
	}
}

func TestStack_String_PrintsContentUsingFormattedHex(t *testing.T) {
	stack := NewStack()
	defer ReturnStack(stack)

	for i := 0; i < 256; i++ {
		top := stack.pushUndefined()
		top.Lsh(uint256.NewInt(1), uint(i))
	}

	print := stack.String()

	wanted := []string{
		"[   0] 0x0000000000000000 0000000000000000 0000000000000000 0000000000000001",
		"[   1] 0x0000000000000000 0000000000000000 0000000000000000 0000000000000002",
		"[   2] 0x0000000000000000 0000000000000000 0000000000000000 0000000000000004",
		"[  16] 0x0000000000000000 0000000000000000 0000000000000000 0000000000010000",
		"[  64] 0x0000000000000000 0000000000000000 0000000000000001 0000000000000000",
		"[ 128] 0x0000000000000000 0000000000000001 0000000000000000 0000000000000000",
		"[ 255] 0x8000000000000000 0000000000000000 0000000000000000 0000000000000000",
	}

	for _, want := range wanted {
		if !strings.Contains(print, want) {
			t.Errorf("expected output to contain %q, but got %q", want, print)
		}
	}
}

func TestStack_NewStack_IsEmpty(t *testing.T) {
	stack := NewStack()
	defer ReturnStack(stack)

	if want, got := 0, stack.len(); want != got {
		t.Errorf("expected stack to be empty, but got %d elements", got)
	}
}

func TestStack_NewStackAndReturnStack_AreThreadSafe(t *testing.T) {
	// this test assumes to be executed using the --race flag.
	const parallelism = 10
	const iterations = 1000

	var wg sync.WaitGroup
	wg.Add(parallelism)
	for i := 0; i < parallelism; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				stack := NewStack()
				defer ReturnStack(stack)
			}
		}()
	}
	wg.Wait()
}
