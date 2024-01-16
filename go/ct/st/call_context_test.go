package st

import (
	"fmt"
	"strings"
	"testing"

	. "github.com/Fantom-foundation/Tosca/go/ct/common"
)

func TestCallContext_NewCallContext(t *testing.T) {
	callContext := NewCallContext()

	if want, got := (Address{}), callContext.AccountAddress; want != got {
		t.Errorf("Unexpected account address, want %v, got %v", want, got)
	}

	if want, got := (Address{}), callContext.OriginAddress; want != got {
		t.Errorf("Unexpected origin address, want %v, got %v", want, got)
	}

	if want, got := (Address{}), callContext.CallerAddress; want != got {
		t.Errorf("Unexpected caller address, want %v, got %v", want, got)
	}

	if want, got := NewU256(), callContext.Value; !want.Eq(got) {
		t.Errorf("Unexpected call value, want %v got %v", want, got)
	}

}

func TestCallContext_Clone(t *testing.T) {
	callContext1 := NewCallContext()
	callContext2 := callContext1.Clone()

	if !callContext1.Eq(callContext2) {
		t.Errorf("Clone is different from original")
	}

	callContext2.AccountAddress = Address{0xff}
	callContext2.OriginAddress = Address{0xfe}
	callContext2.CallerAddress = Address{0xfd}
	callContext2.Value = NewU256(1)

	if callContext1.AccountAddress == callContext2.AccountAddress ||
		callContext1.OriginAddress == callContext2.OriginAddress ||
		callContext1.CallerAddress == callContext2.CallerAddress ||
		callContext1.Value.Eq(callContext2.Value) {
		t.Errorf("Clone is not independent from original")
	}
}

func TestCallContext_Eq(t *testing.T) {
	callContext1 := NewCallContext()
	callContext2 := callContext1.Clone()

	if !callContext1.Eq(callContext1) {
		t.Error("Self-comparison is broken")
	}

	if !callContext1.Eq(callContext2) {
		t.Error("Clones are not equal")
	}

	callContext2.AccountAddress = Address{0xff}
	if callContext1.Eq(callContext2) {
		t.Error("Different call context considered the same")
	}
	callContext2.AccountAddress = Address{}

	callContext2.OriginAddress = Address{0xff}
	if callContext1.Eq(callContext2) {
		t.Error("Different call context considered the same")
	}
	callContext2.OriginAddress = Address{}

	callContext2.CallerAddress = Address{0xff}
	if callContext1.Eq(callContext2) {
		t.Error("Different call context considered the same")
	}
	callContext2.CallerAddress = Address{}

	callContext2.Value = NewU256(2)
	if callContext1.Eq(callContext2) {
		t.Error("Different call context considered the same")
	}
}

func TestCallContext_Diff(t *testing.T) {
	callContext1 := NewCallContext()
	callContext2 := NewCallContext()

	if diffs := callContext1.Diff(callContext1); len(diffs) != 0 {
		t.Errorf("Found differences in same call context.")
	}

	callContext2.AccountAddress = Address{0xff}
	if diffs := callContext1.Diff(callContext2); len(diffs) == 0 {
		t.Errorf("No difference found in different call contexts account address")
	}
	callContext2.AccountAddress = Address{}

	callContext2.OriginAddress = Address{0xff}
	if diffs := callContext1.Diff(callContext2); len(diffs) == 0 {
		t.Errorf("No difference found in different call contexts origin address")
	}
	callContext2.OriginAddress = Address{}

	callContext2.CallerAddress = Address{0xff}
	if diffs := callContext1.Diff(callContext2); len(diffs) == 0 {
		t.Errorf("No difference found in different call contexts caller address")
	}
	callContext2.CallerAddress = Address{}

	callContext2.Value = NewU256(2)
	if diffs := callContext1.Diff(callContext2); len(diffs) == 0 {
		t.Errorf("No difference found in different call contexts value")
	}
	callContext2.Value = NewU256()
}

func TestCallContext_String(t *testing.T) {
	s := NewState(NewCode([]byte{}))
	s.CallContext = NewCallContext()
	s.CallContext.AccountAddress[19] = 0xff
	s.CallContext.OriginAddress[19] = 0xfe
	s.CallContext.CallerAddress[19] = 0xfd
	s.CallContext.Value = NewU256(1)

	if !strings.Contains(s.String(), fmt.Sprintf("Account Address: %s", s.CallContext.AccountAddress)) {
		t.Errorf("Did not find account address string.")
	}
	if !strings.Contains(s.String(), fmt.Sprintf("Origin Address: %s", s.CallContext.OriginAddress)) {
		t.Errorf("Did not find origin address string.")
	}
	if !strings.Contains(s.String(), fmt.Sprintf("Caller Address: %s", s.CallContext.CallerAddress)) {
		t.Errorf("Did not find caller address string.")
	}
	if !strings.Contains(s.String(), fmt.Sprintf("Value: %s", s.CallContext.Value)) {
		t.Errorf("Did not find value string.")
	}
}
