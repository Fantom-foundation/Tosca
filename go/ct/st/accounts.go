package st

import (
	"bytes"
	"fmt"
	"reflect"

	. "github.com/Fantom-foundation/Tosca/go/ct/common"
	"github.com/Fantom-foundation/Tosca/go/vm"
	"golang.org/x/crypto/sha3"
	"golang.org/x/exp/maps"
)

type Accounts struct {
	Balance map[vm.Address]U256
	Code    map[vm.Address][]byte
	warm    map[vm.Address]struct{}
}

func NewAccounts() *Accounts {
	return &Accounts{
		Balance: make(map[vm.Address]U256),
		Code:    make(map[vm.Address][]byte),
		warm:    make(map[vm.Address]struct{}),
	}
}

func (a *Accounts) GetCodeHash(address vm.Address) (hash [32]byte) {
	hasher := sha3.NewLegacyKeccak256()
	hasher.Write(a.Code[address])
	hasher.Sum(hash[:])
	return
}

func (a *Accounts) Clone() *Accounts {
	return &Accounts{
		Balance: maps.Clone(a.Balance),
		Code:    maps.Clone(a.Code),
		warm:    maps.Clone(a.warm),
	}
}

func (a *Accounts) Eq(b *Accounts) bool {
	return maps.Equal(a.Balance, b.Balance) &&
		reflect.DeepEqual(a.Code, b.Code) &&
		maps.Equal(a.warm, b.warm)
}

func (a *Accounts) Diff(b *Accounts) (res []string) {
	for key, valueA := range a.Balance {
		valueB, contained := b.Balance[key]
		if !contained {
			res = append(res, fmt.Sprintf("Different balance entry:\n\t[%v]=%v\n\tvs\n\tmissing", key, valueA))
		} else if valueA != valueB {
			res = append(res, fmt.Sprintf("Different balance entry:\n\t[%v]=%v\n\tvs\n\t[%v]=%v", key, valueA, key, valueB))
		}
	}
	for key, valueB := range b.Balance {
		if _, contained := a.Balance[key]; !contained {
			res = append(res, fmt.Sprintf("Different balance entry:\n\tmissing\n\tvs\n\t[%v]=%v", key, valueB))
		}
	}

	for address, valueA := range a.Code {
		valueB, contained := b.Code[address]
		if !contained {
			res = append(res, fmt.Sprintf("Different code entry:\n\t[%v]=%v\n\tvs\n\tmissing", address, valueA))
		} else if !bytes.Equal(valueA, valueB) {
			res = append(res, fmt.Sprintf("Different code entry:\n\t[%v]=%v\n\tvs\n\t[%v]=%v", address, valueA, address, valueB))
		}
	}
	for address, valueB := range b.Code {
		if _, contained := a.Balance[address]; !contained {
			res = append(res, fmt.Sprintf("Different code entry:\n\tmissing\n\tvs\n\t[%v]=%v", address, valueB))
		}
	}

	for key := range a.warm {
		if _, contained := b.warm[key]; !contained {
			res = append(res, fmt.Sprintf("Different account warm entry: %v vs missing", key))
		}
	}
	for key := range b.warm {
		if _, contained := a.warm[key]; !contained {
			res = append(res, fmt.Sprintf("Different account warm entry: missing vs %v", key))
		}
	}

	return
}

func (a *Accounts) IsWarm(key vm.Address) bool {
	_, contains := a.warm[key]
	return contains
}

func (a *Accounts) IsCold(key vm.Address) bool {
	_, contains := a.warm[key]
	return !contains
}

func (a *Accounts) MarkWarm(key vm.Address) {
	a.warm[key] = struct{}{}
}

func (a *Accounts) MarkCold(key vm.Address) {
	delete(a.warm, key)
}

func (a *Accounts) SetWarm(key vm.Address, warm bool) {
	if warm {
		a.MarkWarm(key)
	} else {
		a.MarkCold(key)
	}
}
