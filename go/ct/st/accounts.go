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
	balance map[vm.Address]U256
	code    map[vm.Address][]byte
	warm    map[vm.Address]struct{}
}

func NewAccounts() *Accounts {
	return &Accounts{
		balance: make(map[vm.Address]U256),
		code:    make(map[vm.Address][]byte),
		warm:    make(map[vm.Address]struct{}),
	}
}

func (a *Accounts) GetCodeHash(address vm.Address) (hash [32]byte) {
	hasher := sha3.NewLegacyKeccak256()
	hasher.Write(a.code[address])
	hasher.Sum(hash[:])
	return
}

func (a *Accounts) IsEmpty(address vm.Address) bool {
	// By definition, an account is empty if it has an empty balance,
	// a nonce that is 0, and an empty code. However, we do not model
	// nonces in this state, so we only check the balance and code.
	return a.balance[address] == U256{} && len(a.code[address]) == 0
}

func (a *Accounts) Clone() *Accounts {
	return &Accounts{
		balance: maps.Clone(a.balance),
		code:    maps.Clone(a.code),
		warm:    maps.Clone(a.warm),
	}
}

func (a *Accounts) Eq(b *Accounts) bool {
	return maps.Equal(a.balance, b.balance) &&
		reflect.DeepEqual(a.code, b.code) &&
		maps.Equal(a.warm, b.warm)
}

func (a *Accounts) Diff(b *Accounts) (res []string) {
	for key, valueA := range a.balance {
		valueB, contained := b.balance[key]
		if !contained {
			res = append(res, fmt.Sprintf("Different balance entry:\n\t[%v]=%v\n\tvs\n\tmissing", key, valueA))
		} else if valueA != valueB {
			res = append(res, fmt.Sprintf("Different balance entry:\n\t[%v]=%v\n\tvs\n\t[%v]=%v", key, valueA, key, valueB))
		}
	}
	for key, valueB := range b.balance {
		if _, contained := a.balance[key]; !contained {
			res = append(res, fmt.Sprintf("Different balance entry:\n\tmissing\n\tvs\n\t[%v]=%v", key, valueB))
		}
	}

	for address, valueA := range a.code {
		valueB, contained := b.code[address]
		if !contained {
			res = append(res, fmt.Sprintf("Different code entry:\n\t[%v]=%v\n\tvs\n\tmissing", address, valueA))
		} else if !bytes.Equal(valueA, valueB) {
			res = append(res, fmt.Sprintf("Different code entry:\n\t[%v]=%v\n\tvs\n\t[%v]=%v", address, valueA, address, valueB))
		}
	}
	for address, valueB := range b.code {
		if _, contained := a.balance[address]; !contained {
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

func (a *Accounts) SetBalance(address vm.Address, val U256) {
	a.balance[address] = val
}

func (a *Accounts) GetBalance(address vm.Address) U256 {
	return a.balance[address]
}

func (a *Accounts) SetCode(address vm.Address, code []byte) {
	a.code[address] = code
}

func (a *Accounts) GetCode(address vm.Address) []byte {
	return a.code[address]
}

func (a *Accounts) Exist(address vm.Address) bool {
	_, existsBalance := a.balance[address]
	_, existsCode := a.code[address]
	return existsBalance || existsCode
}

func AccountBuilder(balance map[vm.Address]U256, code map[vm.Address][]byte) *Accounts {
	newAcc := Accounts{}
	newAcc.balance = balance
	newAcc.code = code
	newAcc.warm = make(map[vm.Address]struct{})
	return &newAcc
}

func AccountBuilderWithWarm(balance map[vm.Address]U256, code map[vm.Address][]byte, warm map[vm.Address]struct{}) *Accounts {
	newAcc := Accounts{}
	newAcc.balance = balance
	newAcc.code = code
	newAcc.warm = warm
	return &newAcc
}

func (a *Accounts) CloneBalance() map[vm.Address]U256 {
	return maps.Clone(a.balance)
}

func (a *Accounts) CloneCode() map[vm.Address][]byte {
	return maps.Clone(a.code)
}

func (a *Accounts) String() string {
	var retString string
	write := func(pattern string, args ...any) {
		retString += fmt.Sprintf(pattern, args...)
	}
	write("\tAccount.Balance:\n")
	for k, v := range a.balance {
		write("\t    [%v]=%v\n", k, v)
	}
	write("\tAccount.Code:\n")
	for k, v := range a.code {
		write("\t    [%v]=%v\n", k, v)
	}
	write("\tAccount.Warm:\n")
	for k, v := range a.warm {
		write("\t    [%v]=%v\n", k, v)
	}

	return retString
}
