package st

import (
	"fmt"
	"strings"
	"testing"

	. "github.com/Fantom-foundation/Tosca/go/ct/common"
)

func test_newaddr(t *testing.T, address *Address) {
	if want, got := (Address{}), *address; want != got {
		t.Errorf("Unexpected address, want %v, got %v", want, got)
	}
}

func TestCallContext_NewCallContext(t *testing.T) {
	callContext := NewCallContext()
	test_newaddr(t, &callContext.AccountAddress)
	test_newaddr(t, &callContext.OriginAddress)
}

func TestCallContext_Clone(t *testing.T) {
	callContext1 := NewCallContext()
	callContext2 := callContext1.Clone()

	if !callContext1.Eq(callContext2) {
		t.Errorf("Clone is different from original")
	}

	callContext2.AccountAddress = Address{0xff}
	callContext2.OriginAddress = Address{0xfe}

	if callContext1.AccountAddress == callContext2.AccountAddress ||
		callContext1.OriginAddress == callContext2.OriginAddress {
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

	callContext2 = callContext1.Clone()
	callContext2.OriginAddress = Address{0xff}
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
		t.Errorf("No difference found in different call contexts")
	}

	callContext2 = NewCallContext()
	callContext2.OriginAddress = Address{0xff}
	if diffs := callContext1.Diff(callContext2); len(diffs) == 0 {
		t.Errorf("No difference found in different call contexts")
	}
}

func TestCallContext_String(t *testing.T) {
	s := NewState(NewCode([]byte{}))
	s.CallContext = NewCallContext()
	s.CallContext.AccountAddress = Address{}
	s.CallContext.AccountAddress[19] = 0xff
	s.CallContext.OriginAddress = Address{}
	s.CallContext.OriginAddress[19] = 0xff

	if !strings.Contains(s.String(), fmt.Sprintf("Account Address: %s", s.CallContext.AccountAddress)) {
		t.Errorf("Did not find account address string.")
	}

	if !strings.Contains(s.String(), fmt.Sprintf("Origin Address: %s", s.CallContext.OriginAddress)) {
		t.Errorf("Did not find account address string.")
	}
}
