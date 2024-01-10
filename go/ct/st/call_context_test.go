package st

import (
	"regexp"
	"testing"

	. "github.com/Fantom-foundation/Tosca/go/ct/common"
)

func TestCallCtx_NewCallCtx(t *testing.T) {
	callCtx := NewCallCtx()
	if want, got := *NewAddress(), *callCtx.AccountAddr; want != got {
		t.Errorf("Unexpected address, want %v, got %v", want, got)
	}
}

func TestCallCtx_Clone(t *testing.T) {
	callCtx1 := NewCallCtx()
	callCtx2 := callCtx1.Clone()

	if !callCtx1.Eq(callCtx2) {
		t.Errorf("Clone is different from original")
	}

	callCtx2.AccountAddr = &Address{0xff}

	if callCtx1.AccountAddr.Eq(callCtx2.AccountAddr) {
		t.Errorf("Clone is not independent from original")
	}
}

func TestCallCtx_Eq(t *testing.T) {
	callCtx1 := NewCallCtx()
	callCtx2 := callCtx1.Clone()

	if !callCtx1.Eq(callCtx1) {
		t.Error("Self-comparison is broken")
	}

	if !callCtx1.Eq(callCtx2) {
		t.Error("Clones are not equal")
	}

	callCtx2.AccountAddr = &Address{0xff}

	if callCtx1.Eq(callCtx2) {
		t.Error("Different call context considered the same")
	}
}

func TestCallCtx_Diff(t *testing.T) {
	callCtx1 := NewCallCtx()
	callCtx2 := NewCallCtx()

	if diffs := callCtx1.Diff(callCtx1); len(diffs) != 0 {
		t.Errorf("Found differences in same call context.")
	}

	callCtx2.AccountAddr = &Address{0xff}
	if diffs := callCtx1.Diff(callCtx2); len(diffs) == 0 {
		t.Errorf("Different not found in different call contexts")
	}
}

func TestCallCtx_String(t *testing.T) {
	s := NewState(NewCode([]byte{}))
	s.CallCtx = NewCallCtx()
	s.CallCtx.AccountAddr = &Address{}
	s.CallCtx.AccountAddr[19] = 0xff

	r := regexp.MustCompile(`Account Address: 0x([[:xdigit:]]+)`)
	str := s.String()
	match := r.FindStringSubmatch(str)

	if len(match) != 2 {
		t.Fatal("invalid print, did not find Account Address")
	}

	if want, got := "00000000000000000000000000000000000000ff", match[1]; want != got {
		t.Errorf("invalid account address, want %v, got %v", want, got)
	}

}
