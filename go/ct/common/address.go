package common

import (
	"fmt"
	"slices"

	"pgregory.net/rand"
)

type Address [20]byte

func NewAddress() *Address {
	return &Address{}
}

func (a *Address) Eq(other *Address) bool {
	return slices.Equal(a[:], other[:])
}

// Diff returns a list of differences between the two addresses.
func (a *Address) Diff(b *Address) (res []string) {
	for i := 0; i < 20; i++ {
		if a[i] != b[i] {
			res = append(res, fmt.Sprintf("Different address value at position %d:\n    %v\n    vs\n    %v", i, a[i], b[i]))
		}
	}
	return
}

func RandAddress(rnd *rand.Rand) *Address {
	addr := Address{}
	rnd.Read(addr[:])
	return &addr
}
