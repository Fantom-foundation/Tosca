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
	addr1 := &Address{0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x0e, 0x0f, 0x10, 0x11, 0x12, 0x13}
	str := addr1.String()
	if str != "0x000102030405060708090a0b0c0d0e0f10111213" {
		t.Errorf("Invalid address string.")
	}

}
