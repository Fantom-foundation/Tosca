// Copyright (c) 2024 Fantom Foundation
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at fantom.foundation/bsl11.
//
// Change Date: 2028-4-16
//
// On the date above, in accordance with the Business Source License, use of
// this software will be governed by the GNU Lesser General Public License v3.

package st

import (
	"bytes"
	"fmt"
	"sort"
	"strings"

	. "github.com/Fantom-foundation/Tosca/go/ct/common"
	"github.com/Fantom-foundation/Tosca/go/tosca"
	"golang.org/x/crypto/sha3"
	"golang.org/x/exp/maps"
)

// Account models a single account on the blockchain. For each account, balances
// and codes are stored, as these are the properties that can be accessed by
// transactions. Nonces can not be accessed, and are thus not modeled here.
type Account struct {
	Balance U256
	Code    Bytes
}

// Accounts models the account state of the block chain. It retains information
// on balances of accounts, codes and their existence state during the execution
// of transactions.
type Accounts struct {
	accounts map[tosca.Address]Account

	// TODO: the warm/cold state of accounts is a property of the transaction
	// context, not the block chain state. It should thus be moved to a separate
	// struct within the CT state model.
	warm map[tosca.Address]struct{}
}

func NewAccounts() *Accounts {
	return &Accounts{}
}

func (a *Accounts) Exists(address tosca.Address) bool {
	_, found := a.accounts[address]
	return found
}

func (a *Accounts) IsEmpty(address tosca.Address) bool {
	// By definition, an account is empty if it has an empty balance,
	// a nonce that is 0, and an empty code. However, we do not model
	// nonces in this state, so we only check the balance and code.
	return a.GetAccount(address) == Account{}
}

func (a *Accounts) GetAccount(address tosca.Address) Account {
	return a.accounts[address]
}

func (a *Accounts) GetBalance(address tosca.Address) U256 {
	return a.accounts[address].Balance
}

func (a *Accounts) SetBalance(address tosca.Address, val U256) {
	a.modifyAccount(address, func(account *Account) {
		account.Balance = val
	})
}

func (a *Accounts) GetCode(address tosca.Address) Bytes {
	return a.accounts[address].Code
}

func (a *Accounts) GetCodeHash(address tosca.Address) (hash [32]byte) {
	hasher := sha3.NewLegacyKeccak256()
	_, _ = hasher.Write(a.GetCode(address).ToBytes()) // Hash.Write never returns an error
	hasher.Sum(hash[:])
	return
}

func (a *Accounts) modifyAccount(address tosca.Address, f func(*Account)) {
	if a.accounts == nil {
		a.accounts = make(map[tosca.Address]Account)
	} else {
		a.accounts = maps.Clone(a.accounts)
	}
	account := a.accounts[address]
	f(&account)
	a.accounts[address] = account
}

// -- Warm / Cold Accounts --

func (a *Accounts) IsWarm(key tosca.Address) bool {
	_, contains := a.warm[key]
	return contains
}

func (a *Accounts) MarkWarm(address tosca.Address) {
	if a.warm == nil {
		a.warm = make(map[tosca.Address]struct{})
	} else {
		a.warm = maps.Clone(a.warm)
	}
	a.warm[address] = struct{}{}
}

// -- State Management --

func (a *Accounts) Clone() *Accounts {
	return &Accounts{ // < content is copy-on-write
		accounts: a.accounts,
		warm:     a.warm,
	}
}

func (a *Accounts) Eq(b *Accounts) bool {
	return maps.Equal(a.accounts, b.accounts) && maps.Equal(a.warm, b.warm)
}

func (a *Accounts) Diff(b *Accounts) (res []string) {
	for address, accountA := range a.accounts {
		accountB, contained := b.accounts[address]
		if !contained {
			res = append(res, fmt.Sprintf("Different account entry:\n\t[%v]=%v\n\tvs\n\tmissing", address, accountA))
		} else if accountA != accountB {
			res = append(res, fmt.Sprintf("Different account entry:\n\t[%v]=%v\n\tvs\n\t[%v]=%v", address, accountA, address, accountB))
		}
	}
	for address, accountB := range b.accounts {
		if _, contained := a.accounts[address]; !contained {
			res = append(res, fmt.Sprintf("Different account entry:\n\tmissing\n\tvs\n\t[%v]=%v", address, accountB))
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

func (a *Accounts) String() string {
	res := strings.Builder{}
	write := func(pattern string, args ...any) {
		res.WriteString(fmt.Sprintf(pattern, args...))
	}

	order := func(a, b tosca.Address) bool {
		return bytes.Compare(a[:], b[:]) < 0
	}

	addresses := maps.Keys(a.accounts)
	sort.Slice(addresses, func(i, j int) bool {
		return order(addresses[i], addresses[j])
	})
	write("Accounts:\n")
	for _, address := range addresses {
		account := a.accounts[address]
		write("\t%v:\n", address)
		write("\t\tBalance: %v\n", account.Balance)
		write("\t\tCode: %v\n", account.Code)
	}

	addresses = maps.Keys(a.warm)
	sort.Slice(addresses, func(i, j int) bool {
		return order(addresses[i], addresses[j])
	})
	write("Warm Accounts:\n")
	for _, address := range addresses {
		write("\t\t%v\n", address)
	}

	return res.String()
}

/// --- Accounts Builder

type AccountsBuilder struct {
	accounts Accounts
}

func NewAccountsBuilder() *AccountsBuilder {
	ab := &AccountsBuilder{}
	ab.accounts = *NewAccounts()
	return ab
}

// Build returns the immutable accounts instance and resets it's own internal accounts.
func (ab *AccountsBuilder) Build() *Accounts {
	acc := ab.accounts
	ab.accounts = Accounts{}
	return &acc
}

func (ab *AccountsBuilder) SetBalance(addr tosca.Address, value U256) *AccountsBuilder {
	ab.modifyAccount(addr, func(account *Account) {
		account.Balance = value
	})
	return ab
}

func (ab *AccountsBuilder) SetCode(addr tosca.Address, code Bytes) *AccountsBuilder {
	ab.modifyAccount(addr, func(account *Account) {
		account.Code = code
	})
	return ab
}

func (ab *AccountsBuilder) modifyAccount(address tosca.Address, f func(*Account)) {
	if ab.accounts.accounts == nil {
		ab.accounts.accounts = make(map[tosca.Address]Account)
	}
	account := ab.accounts.accounts[address]
	f(&account)
	ab.accounts.accounts[address] = account
}

func (ab *AccountsBuilder) SetWarm(addr tosca.Address) *AccountsBuilder {
	if ab.accounts.warm == nil {
		ab.accounts.warm = make(map[tosca.Address]struct{})
	}
	ab.accounts.warm[addr] = struct{}{}
	return ab
}

func (ab *AccountsBuilder) Exists(addr tosca.Address) bool {
	return ab.accounts.Exists(addr)
}
