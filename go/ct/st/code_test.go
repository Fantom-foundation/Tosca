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
	"bytes"
	"errors"
	"fmt"
	"slices"
	"testing"

	. "github.com/Fantom-foundation/Tosca/go/ct/common"
)

func TestCode_NewCode(t *testing.T) {
	code := NewCode([]byte{})
	if want, got := 0, code.Length(); want != got {
		t.Errorf("unexpected code length, want %v, got %v", want, got)
	}

	code = NewCode([]byte{byte(ADD), byte(PUSH1), 0, byte(PUSH2)})
	if want, got := 4, code.Length(); want != got {
		t.Errorf("unexpected code length, want %v, got %v", want, got)
	}
}

func TestCode_NewCodeIsIndependent(t *testing.T) {
	src := []byte{byte(ADD), byte(PUSH1), 5, byte(PUSH2)}
	code := NewCode(src)
	if want, got := 4, code.Length(); want != got {
		t.Fatalf("unexpected code length, want %v, got %v", want, got)
	}

	src[0] = byte(PUSH1)
	if want, got := byte(ADD), code.code[0]; want != got {
		t.Errorf("unexpected code, want %v, got %v", want, got)
	}
}

func TestCode_Hash(t *testing.T) {
	empty := NewCode([]byte{})
	if fmt.Sprintf("%x", empty.Hash()) != "c5d2460186f7233c927e7db2dcc703c0e500b653ca82273b7bfad8045d85a470" {
		t.Fatal("invalid code hash for empty")
	}

	add := NewCode([]byte{byte(ADD)})
	if fmt.Sprintf("%x", add.Hash()) != "5fe7f977e71dba2ea1a68e21057beebb9be2ac30c6410aa38d4f3fbe41dcffd2" {
		t.Fatal("invalid code hash for single ADD")
	}
}

func TestCode_IsCode(t *testing.T) {
	code := NewCode([]byte{byte(ADD), byte(PUSH1), 0, byte(PUSH2), 1})
	for i, want := range []bool{true, true, false, true, false, false, true, true} {
		if got := code.IsCode(i); want != got {
			t.Errorf("unexpected result for position %d, want %t, got %t", i, want, got)
		}
	}
}

func TestCode_IsData(t *testing.T) {
	code := NewCode([]byte{byte(ADD), byte(PUSH1), 0, byte(PUSH2)})
	for i, want := range []bool{false, false, true, false, true, true, false, false} {
		if got := code.IsData(i); want != got {
			t.Errorf("unexpected result for position %d, want %t, got %t", i, want, got)
		}
	}
}

func TestCode_GetOperation(t *testing.T) {
	code := NewCode([]byte{byte(ADD), byte(PUSH1), 0, byte(PUSH2)})
	for i, want := range map[int]OpCode{-1: STOP, 0: ADD, 1: PUSH1, 3: PUSH2, 6: STOP} {
		if got, err := code.GetOperation(i); err != nil || want != got {
			t.Errorf("unexpected result for position %d, want %v, got %v, err %v", i, want, got, err)
		}
	}
	for _, i := range []int{2, 4, 5} {
		if _, err := code.GetOperation(i); !errors.Is(err, ErrInvalidPosition) {
			t.Errorf("unexpected error: %v", err)
		}
	}
}

func TestCode_GetData(t *testing.T) {
	code := NewCode([]byte{byte(ADD), byte(PUSH1), 5, byte(PUSH2)})
	for i, want := range map[int]byte{2: 5, 4: 0, 5: 0} {
		if got, err := code.GetData(i); err != nil || want != got {
			t.Errorf("unexpected result for position %d, want %v, got %v, err %v", i, want, got, err)
		}
	}
	for _, i := range []int{0, 1, 3} {
		if _, err := code.GetData(i); !errors.Is(err, ErrInvalidPosition) {
			t.Errorf("unexpected error: %v", err)
		}
	}
}

func TestCode_Copy(t *testing.T) {
	src := []byte{byte(ADD), byte(PUSH1), 5, byte(PUSH2)}
	code := NewCode(src)

	if got, want := code.Length(), len(src); got != want {
		t.Errorf("unexpected code length, wanted %d, got %d", want, got)
	}

	for i := 0; i < len(src); i++ {
		if got := code.Copy(); !bytes.Equal(src, got) {
			t.Errorf("failed to copy data, expected %x, got %x", src, got)
		}
	}
}

func TestCode_Equal(t *testing.T) {
	a := NewCode([]byte{1, 2, 3})
	b := NewCode([]byte{3, 2, 1})
	c := a

	if a.Eq(b) {
		t.Errorf("should not be equal: %v vs. %v", a, b)
	}

	if !a.Eq(c) {
		t.Errorf("should be equal: %v vs. %v", a, c)
	}
}

func TestCode_Printer(t *testing.T) {
	code := NewCode([]byte{byte(ADD), byte(PUSH1), 5, byte(PUSH2)})
	want := "01600561"
	if got := code.String(); want != got {
		t.Errorf("invalid print, wanted %s, got %s", want, got)
	}
}

func TestCode_CopyCodeSlice(t *testing.T) {
	code := NewCode([]byte{byte(ADD), byte(PUSH1), 5, byte(PUSH2)})
	tests := map[string]struct {
		start int
		end   int
		want  []byte
	}{
		"regular":  {1, 4, []byte{byte(PUSH1), 5, byte(PUSH2)}},
		"sizeZero": {1, 1, []byte{}},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			got := make([]byte, test.end-test.start)
			_ = code.CopyCodeSlice(test.start, test.end, got)
			if !slices.Equal(test.want, got) {
				t.Errorf("unexpected code, wanted %v, got %v", test.want, got)
			}
		})
	}
}

func TestCode_CopyCodeSliceInvalid(t *testing.T) {
	code := NewCode([]byte{byte(ADD), byte(PUSH1), 5, byte(PUSH2)})
	tests := map[string]struct {
		start int
		end   int
	}{
		"endBeforeStart":      {2, 0},
		"negativeOffset":      {-2, 2},
		"partiallyOutOfBound": {1, 6},
		"fullyOutOfBound":     {6, 8},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			defer func() {
				if r := recover(); r == nil {
					t.Errorf("expected panic.")
				}
			}()
			buffer := make([]byte, max(test.end-test.start, 4))
			_ = code.CopyCodeSlice(test.start, test.end, buffer)
		})
	}
}
