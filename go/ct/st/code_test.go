package st

import (
	"bytes"
	"errors"
	"fmt"
	"slices"
	"testing"

	. "github.com/Fantom-foundation/Tosca/go/ct/common"
)

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

func TestCode_CopyTo(t *testing.T) {
	src := []byte{byte(ADD), byte(PUSH1), 5, byte(PUSH2)}
	code := NewCode(src)

	if got, want := code.Length(), len(src); got != want {
		t.Errorf("unexpected code length, wanted %d, got %d", want, got)
	}

	for i := 0; i < len(src); i++ {
		dst := make([]byte, i)
		if got := code.CopyTo(dst); got != i || !bytes.Equal(src[0:i], dst) {
			t.Errorf("failed to copy data, expected %x, got %x, return %d", src[0:i], dst, i)
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

func TestCode_GetSlice(t *testing.T) {
	code := NewCode([]byte{byte(ADD), byte(PUSH1), 5, byte(PUSH2)})
	sizeOverflowCode := append(code.code, []byte{0, 0}...)

	tests := map[string]struct {
		code   *Code
		offset int
		size   int
		want   []byte
	}{
		"regular":        {code, 1, 4, []byte{byte(PUSH1), 5, byte(PUSH2)}},
		"sizeZero":       {code, 1, 0, []byte{}},
		"offsetOverflow": {code, 5, 1, []byte{}},
		"sizeOverflow":   {code, 0, 6, sizeOverflowCode},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			got := test.code.GetSlice(test.offset, test.size)
			if !slices.Equal(test.want, got) {
				t.Errorf("unexpected code, wanted %v, got %v", test.want, got)
			}
		})
	}
}
