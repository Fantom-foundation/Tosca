package common

import (
	"fmt"

	"pgregory.net/rand"
)

type Address [20]byte

func NewAddress() Address {
	return Address{}
}

// Diff returns a list of differences between the two addresses.
func (a *Address) Diff(b *Address) (res []string) {
	if *a != *b {
		res = append(res, fmt.Sprintf("Different address, want %v, got %v", a, b))
	}
	return
}

func RandAddress(rnd *rand.Rand) (*Address, error) {
	address := Address{}
	_, err := rnd.Read(address[:])
	if err != nil {
		return nil, err
	}
	return &address, nil
}

func (a *Address) Clone() *Address {
	newAddress := Address{}
	copy(newAddress[:], a[:])
	return &newAddress
}

func (a Address) String() string {
	return fmt.Sprintf("0x%x", ([20]byte)(a))
}
