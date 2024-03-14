package common

import (
	"bytes"
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

func TestAddress_NewAddressFrom(t *testing.T) {
	addressU256 := NewAddress(NewU256(42))
	addressInt := NewAddressFromInt(42)

	if addressU256 != addressInt {
		t.Errorf("Address from U256 and int should be the same: %v vs %v", addressU256, addressInt)
	}

	if addressU256.String() != "0x000000000000000000000000000000000000002a" {
		t.Errorf("Address from U256 has the wrong value")
	}

	if addressInt.String() != "0x000000000000000000000000000000000000002a" {
		t.Errorf("Address from int has the wrong value")
	}
}

func TestAddress_ToU256(t *testing.T) {
	want := NewU256(42)
	address := NewAddress(want)

	if got := address.ToU256(); want != got {
		t.Errorf("Conversion from U256 is broken: got %v, want %v", got, want)
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

func TestAddress_Marshalling(t *testing.T) {
	tests := []struct {
		address    Address
		marshalled []byte
	}{
		{Address{}, []byte("0x0000000000000000000000000000000000000000")},
		{Address{0x00}, []byte("0x0000000000000000000000000000000000000000")},
		{Address{0x01}, []byte("0x0100000000000000000000000000000000000000")},
		{Address{0x02}, []byte("0x0200000000000000000000000000000000000000")},
		{Address{0x01, 0x02}, []byte("0x0102000000000000000000000000000000000000")},
		{Address{0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x0e, 0x0f, 0x10, 0x11, 0x12, 0x13}, []byte("0x000102030405060708090a0b0c0d0e0f10111213")},
	}

	for _, test := range tests {
		marshalled, err := test.address.MarshalText()
		if err != nil {
			t.Fatalf("Unexpected error when marshalling address: %v", err)
		}
		if !bytes.Equal(marshalled, test.marshalled) {
			t.Errorf("Unexpected marshalled value: want %v, got %v", test.marshalled, marshalled)
		}
	}
}

func TestAddress_Unmarshalling(t *testing.T) {
	tests := []struct {
		marshalled []byte
		want       Address
	}{
		{[]byte("0x0000000000000000000000000000000000000000"), Address{}},
		{[]byte("0x0000000000000000000000000000000000000000"), Address{0x00}},
		{[]byte("0x0100000000000000000000000000000000000000"), Address{0x01}},
		{[]byte("0x0200000000000000000000000000000000000000"), Address{0x02}},
		{[]byte("0x0102000000000000000000000000000000000000"), Address{0x01, 0x02}},
		{[]byte("0x000102030405060708090a0b0c0d0e0f10111213"), Address{0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x0e, 0x0f, 0x10, 0x11, 0x12, 0x13}},
	}

	for _, test := range tests {
		var a Address
		err := a.UnmarshalText(test.marshalled)
		if err != nil {
			t.Fatalf("Unexpected error when unmarshalling address: %v", err)
		}
		if a != test.want {
			t.Errorf("Unexpected unmarshalled value: want %v, got %v", test.want, a)
		}
	}
}

func TestAddress_UnmarshallingError(t *testing.T) {
	testCases := map[string][]byte{
		"empty":                 []byte(""),
		"empty with hex prefix": []byte("0x"),
		"no hex prefix":         []byte("0000000000000000000000000000000000000000"),
		"too short":             []byte("0x00000000000000000000000000000000000000"),
		"too long":              []byte("0x000000000000000000000000000000000000000000"),
		"invalid hex":           []byte("0x0g00000000000000000000000000000000000000"),
	}

	for name, input := range testCases {
		t.Run(name, func(t *testing.T) {
			var a Address
			err := a.UnmarshalText(input)
			if err == nil {
				t.Fatalf("Expected error when unmarshalling input with: %s", name)
			}
		})
	}
}

func TestAddress_MarshallingRoundTrip(t *testing.T) {
	tests := []struct {
		address Address
	}{
		{Address{}},
		{Address{0x00}},
		{Address{0x01}},
		{Address{0x02}},
		{Address{0x01, 0x02}},
		{Address{0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x0e, 0x0f, 0x10, 0x11, 0x12, 0x13}},
	}

	for _, test := range tests {
		marshalled, err := test.address.MarshalText()
		if err != nil {
			t.Fatalf("Unexpected error when marshalling address: %v", err)
		}

		var a Address
		err = a.UnmarshalText(marshalled)
		if err != nil {
			t.Fatalf("Unexpected error when unmarshalling address: %v", err)
		}
		if a != test.address {
			t.Errorf("Unexpected unmarshalled value: want %v, got %v", test.address, a)
		}
	}
}
