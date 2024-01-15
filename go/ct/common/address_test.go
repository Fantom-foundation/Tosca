package common

import (
	"strings"
	"testing"

	"pgregory.net/rand"
)

func TestAddress_NewAddress(t *testing.T) {
	address := Address{}

	if address != [20]byte{} {
		t.Errorf("New address must be default value.")
	}
}

func TestAddress_Eq(t *testing.T) {
	address1 := Address{}
	address2 := Address{0xff}

	if address1 != address1 {
		t.Error("Self-comparison is broken")
	}

	if address1 == address2 {
		t.Errorf("Different addresses are not recognized as such")
	}
}

func TestAddress_Diff(t *testing.T) {
	address1 := Address{}
	address2 := Address{0xff}
	str := address1.Diff(address2)

	if len(str) != 1 {
		t.Errorf("Wrong amount of differences found.")
	}

	if !strings.Contains(str[0], "Different address") {
		t.Errorf("Difference reported at wrong position.")
	}
}

func TestAddress_RandAddress(t *testing.T) {
	address1 := Address{}
	rnd := rand.New(0)
	address2, err := RandAddress(rnd)

	if err != nil {
		t.Errorf("Error generating random %v", err)
	}

	if address1 == address2 {
		t.Errorf("Random Address is same as default value")
	}
}

func TestAddress_Clone(t *testing.T) {
	address1 := Address{}
	address2 := address1.Clone()

	if address1 != address2 {
		t.Error("Clones are not equal")
	}

	address2[0] = 0xff
	if address1 == address2 {
		t.Errorf("Clones are not independent")
	}
}

func TestAddress_String(t *testing.T) {
	address1 := Address{0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x0e, 0x0f, 0x10, 0x11, 0x12, 0x13}
	str := address1.String()
	if str != "0x000102030405060708090a0b0c0d0e0f10111213" {
		t.Errorf("Invalid address string.")
	}

}
