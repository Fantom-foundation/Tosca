package common

import (
	"github.com/Fantom-foundation/Tosca/go/vm"
	"pgregory.net/rand"
)

func NewAddress(in U256) vm.Address {
	return in.internal.Bytes20()
}

func NewAddressFromInt(in uint64) vm.Address {
	return NewAddress(NewU256(in))
}

func AddressToU256(a vm.Address) U256 {
	return NewU256FromBytes(a[:]...)
}

func RandAddress(rnd *rand.Rand) (vm.Address, error) {
	address := vm.Address{}
	_, err := rnd.Read(address[:])
	if err != nil {
		return vm.Address{}, err
	}
	return address, nil
}
