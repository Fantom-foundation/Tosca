package vm_test

import "github.com/Fantom-foundation/Tosca/go/vm"

type WorldState map[vm.Address]Account

//var _ vm.WorldState = &WorldState{}

type Account struct {
	Nonce   uint64
	Balance vm.Value
	Code    []byte
	Storage map[vm.Key]vm.Word
}

func (ws *WorldState) AccountExists(addr vm.Address) bool {
	_, ok := (*ws)[addr]
	return ok
}

func (ws *WorldState) GetNonce(addr vm.Address) uint64 {
	return (*ws)[addr].Nonce
}

