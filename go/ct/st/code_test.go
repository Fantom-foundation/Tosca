package st

import (
	"bytes"
	"errors"
	"testing"
)

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
