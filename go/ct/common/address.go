package common

import (
	"fmt"
	"regexp"
	"strconv"

	"pgregory.net/rand"
)

type Address [20]byte

// Diff returns a list of differences between the two addresses.
func (a *Address) Diff(b Address) (res []string) {
	if *a != b {
		res = append(res, fmt.Sprintf("Different address, want %v, got %v", a, b))
	}
	return
}

func NewAddress(in U256) Address {
	return in.internal.Bytes20()
}

func NewAddressFromInt(in uint64) Address {
	return NewAddress(NewU256(in))
}

func (a *Address) ToU256() U256 {
	return NewU256FromBytes(a[:]...)
}

func RandAddress(rnd *rand.Rand) (Address, error) {
	address := Address{}
	_, err := rnd.Read(address[:])
	if err != nil {
		return Address{}, err
	}
	return address, nil
}

func (a *Address) Clone() Address {
	return *a
}

func (a Address) String() string {
	return fmt.Sprintf("0x%x", ([20]byte)(a))
}

func (a Address) MarshalText() ([]byte, error) {
	return []byte(a.String()), nil
}

func (a *Address) UnmarshalText(text []byte) error {
	r := regexp.MustCompile("^0x([[:xdigit:]]{40})$")
	match := r.FindSubmatch(text)

	if len(match) != 2 {
		return fmt.Errorf("invalid address: %s", text)
	}

	for i := 0; i < 20; i++ {
		b := match[1][i*2 : i*2+2]
		parsed, err := strconv.ParseUint(string(b), 16, 8)
		if err != nil {
			return err
		}
		a[i] = byte(parsed)
	}
	return nil
}
