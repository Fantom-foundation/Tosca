//
// Copyright (c) 2024 Fantom Foundation
//
// Use of this software is governed by the Business Source License included
// in the LICENSE.TXT file and at fantom.foundation/bsl11.
//
// Change Date: 2028-4-16
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by the GNU Lesser General Public Licence v3
//

package st

import (
	"fmt"
	"reflect"

	. "github.com/Fantom-foundation/Tosca/go/ct/common"
	"github.com/Fantom-foundation/Tosca/go/vm"
	"golang.org/x/crypto/sha3"
	"golang.org/x/exp/maps"
)

type Accounts struct {
	balance map[vm.Address]U256
	code    map[vm.Address]Bytes
	warm    map[vm.Address]struct{}
}

func NewAccounts() *Accounts {
	return &Accounts{}
}

func (a *Accounts) GetCodeHash(address vm.Address) (hash [32]byte) {
	hasher := sha3.NewLegacyKeccak256()
	hasher.Write(a.code[address].ToBytes())
	hasher.Sum(hash[:])
	return
}

func (a *Accounts) IsEmpty(address vm.Address) bool {
	// By definition, an account is empty if it has an empty balance,
	// a nonce that is 0, and an empty code. However, we do not model
	// nonces in this state, so we only check the balance and code.
	return a.balance[address] == U256{} && a.code[address].Length() == 0
}

func (a *Accounts) Clone() *Accounts {
	return &Accounts{
		balance: a.balance,
		code:    a.code,
		warm:    a.warm,
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
		} else if valueA != valueB {
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
	if a.warm == nil {
		return false
	}
	_, contains := a.warm[key]
	return contains
}

func (a *Accounts) IsCold(key vm.Address) bool {
	if a.warm == nil {
		return true
	}
	_, contains := a.warm[key]
	return !contains
}

func (a *Accounts) MarkWarm(key vm.Address) {
	if a.warm == nil {
		a.warm = make(map[vm.Address]struct{})
	} else {
		a.warm = maps.Clone(a.warm)
	}
	a.warm[key] = struct{}{}
}

func (a *Accounts) MarkCold(key vm.Address) {
	if a.IsCold(key) {
		return
	}
	a.warm = maps.Clone(a.warm)
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
	if a.balance == nil {
		a.balance = make(map[vm.Address]U256)
	} else {
		a.balance = maps.Clone(a.balance)
	}
	a.balance[address] = val
}

func (a *Accounts) GetBalance(address vm.Address) U256 {
	return a.balance[address]
}

func (a *Accounts) RemoveBalance(address vm.Address) {
	if _, exists := a.balance[address]; !exists {
		return
	}
	a.balance = maps.Clone(a.balance)
	delete(a.balance, address)
}

func (a *Accounts) SetCode(address vm.Address, code Bytes) {
	if a.code == nil {
		a.code = make(map[vm.Address]Bytes)
	} else {
		a.code = maps.Clone(a.code)
	}
	a.code[address] = code
}

func (a *Accounts) GetCode(address vm.Address) Bytes {
	if a.code == nil {
		return NewBytes([]byte{})
	}
	return a.code[address]
}

func (a *Accounts) RemoveCode(address vm.Address) {
	if _, exists := a.code[address]; !exists {
		return
	}
	a.code = maps.Clone(a.code)
	delete(a.code, address)
}

func (a *Accounts) Exist(address vm.Address) bool {
	existsBalance := false
	existsCode := false
	bal := NewU256()
	cod := NewBytes([]byte{})
	if a.balance != nil {
		bal, existsBalance = a.balance[address]
	}
	if a.code != nil {
		cod, existsCode = a.code[address]
	}
	return (existsBalance && bal.Gt(NewU256(0))) ||
		(existsCode && cod.Length() > 0)
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

/// --- Accounts Builder

type AccountsBuilder struct {
	accounts Accounts
}

func NewAccountsBuilder() *AccountsBuilder {
	ab := AccountsBuilder{}
	ab.accounts = *NewAccounts()
	return &ab
}

// Build returns the immutable accounts instance and resets it's own internal accounts.
func (ab *AccountsBuilder) Build() *Accounts {
	acc := ab.accounts
	ab.accounts = Accounts{}
	return &acc
}

func (ab *AccountsBuilder) SetBalance(addr vm.Address, value U256) {
	if ab.accounts.balance == nil {
		ab.accounts.balance = make(map[vm.Address]U256)
	}
	ab.accounts.balance[addr] = value
}

func (ab *AccountsBuilder) SetCode(addr vm.Address, code Bytes) {
	if ab.accounts.code == nil {
		ab.accounts.code = make(map[vm.Address]Bytes)
	}
	ab.accounts.code[addr] = code
}

func (ab *AccountsBuilder) SetWarm(addr vm.Address) {
	if ab.accounts.warm == nil {
		ab.accounts.warm = make(map[vm.Address]struct{})
	}
	ab.accounts.warm[addr] = struct{}{}
}

func (ab *AccountsBuilder) Exists(addr vm.Address) bool {
	return ab.accounts.Exist(addr)
}
