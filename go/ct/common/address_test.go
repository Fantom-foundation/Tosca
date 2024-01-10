package common

import (
	"strings"
	"testing"

	"pgregory.net/rand"
)

func TestAddress_NewAddress(t *testing.T) {
	addr := NewAddress()

	if addr == nil {
		t.Errorf("New address must be allocated.")
	}
}

func TestAddress_Eq(t *testing.T) {
	addr1 := NewAddress()
	addr2 := &Address{0xff}

	if !addr1.Eq(addr1) {
		t.Error("Self-comparison is broken")
	}

	if addr1.Eq(addr2) {
		t.Errorf("Different addresses are not recognized as such")
	}
}

func TestAddress_Diff(t *testing.T) {
	addr1 := NewAddress()
	addr2 := &Address{0xff}
	str := addr1.Diff(addr2)

	if len(str) != 1 {
		t.Errorf("Wrong amount of differences found.")
	}

	if !strings.Contains(str[0], "at position 0:") {
		t.Errorf("Difference reported at wrong position.")
	}
}

func TestAddress_RandAddress(t *testing.T) {
	addr1 := NewAddress()
	rnd := rand.New(0)
	addr2, err := RandAddress(rnd)

	if err != nil {
		t.Errorf("Error generating random %v", err)
	}

	if addr1.Eq(addr2) {
		t.Errorf("Random Address is same as default value")
	}
}

func TestAddress_Clone(t *testing.T) {
	addr1 := NewAddress()
	addr2 := addr1.Clone()

	if !addr1.Eq(addr2) {
		t.Error("Clones are not equal")
	}

	rnd := rand.New(0)
	addr2, err := RandAddress(rnd)
	if err != nil {
		t.Errorf("Error generating random %v", err)
	}
	if addr1.Eq(addr2) {
		t.Errorf("Clones are not independent")
	}
}

func TestAddress_String(t *testing.T) {
	addr1 := NewAddress()
	str := addr1.String()
	if str != "0x0000000000000000000000000000000000000000" {
		t.Errorf("Invalid address string.")
	}

}
