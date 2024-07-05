// Copyright (c) 2024 Fantom Foundation
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at fantom.foundation/bsl11.
//
// Change Date: 2028-4-16
//
// On the date above, in accordance with the Business Source License, use of
// this software will be governed by the GNU Lesser General Public License v3.

package processor

import (
	"bytes"
	"fmt"
	"maps"

	"github.com/Fantom-foundation/Tosca/go/tosca"
)

// ----------------------------------------------------------------------------
// WorldState
// ----------------------------------------------------------------------------

// WorldState provides a utility function to model the world state of a chain
// for testing. It is mainly intended to be used to define pre/post states of
// test scenarios for transaction processors.
type WorldState map[tosca.Address]Account

func (s WorldState) Equal(other WorldState) bool {
	return equalMapsIgnoringZero(s, other, func(a, b Account) bool {
		return a.Equal(&b)
	})
}

func (s WorldState) Clone() WorldState {
	if s == nil {
		return nil
	}
	res := make(WorldState, len(s))
	for k, v := range s {
		res[k] = v.Clone()
	}
	return res
}

func (s WorldState) Diff(other WorldState) []string {
	return diffMaps("", s, other, func(address tosca.Address, a, b Account) []string {
		if a.Equal(&b) {
			return nil
		}
		return a.Diff(fmt.Sprintf("%v/", address), &b)
	})
}

// ----------------------------------------------------------------------------
// Account
// ----------------------------------------------------------------------------

// Account represents an account in the world state. The default account is
// an empty account, that is ignored by the world state.
type Account struct {
	Balance tosca.Value
	Nonce   uint64
	Code    tosca.Code
	Storage Storage
}

func (a *Account) Equal(other *Account) bool {
	return a.Balance == other.Balance &&
		a.Nonce == other.Nonce &&
		bytes.Equal(a.Code, other.Code) &&
		a.Storage.Equal(other.Storage)
}

func (a *Account) Clone() Account {
	return Account{
		Balance: a.Balance,
		Nonce:   a.Nonce,
		Code:    append(tosca.Code(nil), a.Code...),
		Storage: a.Storage.Clone(),
	}
}

func (a *Account) Diff(prefix string, other *Account) []string {
	var res []string
	if a.Balance != other.Balance {
		res = append(res, fmt.Sprintf("different balance: %v != %v", a.Balance, other.Balance))
	}
	if a.Nonce != other.Nonce {
		res = append(res, fmt.Sprintf("different nonce: %v != %v", a.Nonce, other.Nonce))
	}
	if !bytes.Equal(a.Code, other.Code) {
		res = append(res, fmt.Sprintf("different code: 0x%x != 0x%x", a.Code, other.Code))
	}
	res = append(res, a.Storage.Diff(prefix+"Storage/", other.Storage)...)
	for i, diff := range res {
		res[i] = prefix + diff
	}
	return res
}

// ----------------------------------------------------------------------------
// Storage
// ----------------------------------------------------------------------------

// Storage represents the storage of an account in the world state. Zero-valued
// entries are ignored in the storage.
type Storage map[tosca.Key]tosca.Word

func (s Storage) Equal(other Storage) bool {
	return equalMapsIgnoringZero(s, other, func(a, b tosca.Word) bool {
		return a == b
	})
}

func (s Storage) Clone() Storage {
	return maps.Clone(s)
}

func (s Storage) Diff(prefix string, other Storage) []string {
	return diffMaps(prefix, s, other, func(k tosca.Key, a, b tosca.Word) []string {
		if a == b {
			return nil
		}
		return []string{
			fmt.Sprintf("different value for key %v: %v != %v", k, a, b),
		}
	})
}

// ----------------------------------------------------------------------------
// Helpers
// ----------------------------------------------------------------------------

// equalMapsIgnoringZero compares two maps, ignoring zero-valued entries.
func equalMapsIgnoringZero[K comparable, V any](a, b map[K]V, equal func(V, V) bool) bool {
	for k, v := range a {
		if !equal(v, b[k]) {
			return false
		}
	}
	for k, v := range b {
		if !equal(v, a[k]) {
			return false
		}
	}
	return true
}

// diffMaps compares two maps and returns a list of differences.
func diffMaps[K comparable, V any](prefix string, a, b map[K]V, diff func(K, V, V) []string) []string {
	var diffs []string
	for k, v := range a {
		diffs = append(diffs, diff(k, v, b[k])...)
	}
	for k, v := range b {
		if _, overlap := a[k]; !overlap {
			diffs = append(diffs, diff(k, a[k], v)...)
		}
	}
	for i, diff := range diffs {
		diffs[i] = prefix + diff
	}
	return diffs
}
