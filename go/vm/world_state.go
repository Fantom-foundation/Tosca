// Copyright (c) 2024 Fantom Foundation
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at fantom.foundation/bsl11.
//
// Change Date: 2028-4-16
//
// On the date above, in accordance with the Business Source License, use of
// this software will be governed by the GNU Lesser General Public License v3.

package vm

//go:generate mockgen -source world_state.go -destination world_state_mock.go -package vm

// WorldState is an interface to access and manipulate the state of the block chain.
// The state of the chain is a collection of accounts, each with a balance, a nonce,
// optional code and storage.
type WorldState interface {
	AccountExists(Address) bool
	CreateAccount(Address, Code) bool

	GetBalance(Address) Value
	SetBalance(Address, Value)

	GetNonce(Address) uint64
	SetNonce(Address, uint64)

	GetCode(Address) Code
	GetCodeHash(Address) Hash
	GetCodeSize(Address) int

	GetStorage(Address, Key) Word
	SetStorage(Address, Key, Word) StorageStatus

	SelfDestruct(addr Address, beneficiary Address) bool
}

// Address represents the 160-bit (20 bytes) address of an account.
type Address [20]byte

// Key represents the 256-bit (32 bytes) key of a storage slot.
type Key [32]byte

// Word represents an arbitrary 256-bit (32 byte) word in the EVM.
type Word [32]byte

// Value represents an amount of chain currency, typically wei.
type Value [32]byte

// Hash represents the 256-bit (32 bytes) hash of a code, a block, a topic
// or similar sequence of cryptographic summary information.
type Hash [32]byte

// Code represents the byte-code of a contract.
type Code []byte

// StorageStatus is an enum utilized to indicate the effect of a storage
// slot update on the respective slot in the context of the current
// transaction. It is needed to perform proper gas price calculations of
// SSTORE operations.
type StorageStatus int

// See t.ly/b5HPf for the definition of these values.
const (
	StorageAssigned StorageStatus = iota
	StorageAdded
	StorageDeleted
	StorageModified
	StorageDeletedAdded
	StorageModifiedDeleted
	StorageDeletedRestored
	StorageAddedDeleted
	StorageModifiedRestored
)
