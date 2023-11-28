package utils

import (
	"math/big"

	. "github.com/Fantom-foundation/Tosca/go/ct/common"
	"github.com/Fantom-foundation/Tosca/go/ct/st"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

// ConformanceTestStateDb is an adapter between the CT framework's storage
// representation and the StateDB interface expected by an evm interpreter.
type ConformanceTestStateDb struct {
	Storage   *st.Storage
	revision  Revision
	gasRefund uint64
}

// NewConformanceTestStateDb creates a new ConformanceTestStateDb.
func NewConformanceTestStateDb(storage *st.Storage, revision Revision) *ConformanceTestStateDb {
	return &ConformanceTestStateDb{
		Storage:  storage,
		revision: revision,
	}
}

func (db *ConformanceTestStateDb) GetCommittedState(_ common.Address, key common.Hash) common.Hash {
	k := NewU256FromBytes(key[:]...)
	return db.Storage.Original[k].Bytes32be()
}

func (db *ConformanceTestStateDb) GetState(_ common.Address, key common.Hash) common.Hash {
	k := NewU256FromBytes(key[:]...)
	return db.Storage.Current[k].Bytes32be()
}

func (db *ConformanceTestStateDb) SetState(_ common.Address, key common.Hash, value common.Hash) {
	k := NewU256FromBytes(key[:]...)
	v := NewU256FromBytes(value[:]...)
	db.Storage.Current[k] = v
}

func (db *ConformanceTestStateDb) SlotInAccessList(_ common.Address, key common.Hash) (addressOk bool, slotOk bool) {
	k := NewU256FromBytes(key[:]...)
	return true, db.Storage.IsWarm(k)
}

func (db *ConformanceTestStateDb) AddSlotToAccessList(_ common.Address, key common.Hash) {
	k := NewU256FromBytes(key[:]...)
	if db.revision != R07_Istanbul {
		db.Storage.MarkWarm(k)
	}
}

func (db *ConformanceTestStateDb) GetCode(common.Address) []byte {
	panic("not implemented yet")
}

func (db *ConformanceTestStateDb) GetCodeHash(common.Address) common.Hash {
	panic("not implemented yet")
}

func (db *ConformanceTestStateDb) GetCodeSize(common.Address) int {
	panic("not implemented yet")
}

func (db *ConformanceTestStateDb) AddRefund(gas uint64) {
	db.gasRefund += gas
}

func (db *ConformanceTestStateDb) SubRefund(gas uint64) {
	db.gasRefund -= gas
}

func (db *ConformanceTestStateDb) GetRefund() uint64 {
	return db.gasRefund
}

func (db *ConformanceTestStateDb) CreateAccount(common.Address) {
	panic("not implemented yet")
}

func (db *ConformanceTestStateDb) Suicide(common.Address) bool {
	panic("not implemented yet")
}

func (db *ConformanceTestStateDb) AddLog(*types.Log) {
	panic("not implemented yet")
}

// -- StateDB interface methods that should not be needed ---

// The remaining methods of the ConformanceTestStateDb are needed to satisfy
// the interface definition of a StateDB but are not relevant for testing the
// interpreter. These functions are used by the enclosing EVM implementation.

func (db *ConformanceTestStateDb) SubBalance(common.Address, *big.Int) {
	panic("should not be needed")
}

func (db *ConformanceTestStateDb) AddBalance(common.Address, *big.Int) {
	panic("should not be needed")
}

func (db *ConformanceTestStateDb) GetBalance(common.Address) *big.Int {
	panic("should not be needed")
}

func (db *ConformanceTestStateDb) GetNonce(common.Address) uint64 {
	panic("should not be needed")
}

func (db *ConformanceTestStateDb) SetNonce(common.Address, uint64) {
	panic("should not be needed")
}

func (db *ConformanceTestStateDb) SetCode(common.Address, []byte) {
	panic("should not be needed")
}

func (db *ConformanceTestStateDb) AddressInAccessList(addr common.Address) bool {
	panic("should not be needed")
}

func (db *ConformanceTestStateDb) AddAddressToAccessList(addr common.Address) {
	panic("should not be needed")
}

func (db *ConformanceTestStateDb) HasSuicided(common.Address) bool {
	panic("should not be needed")
}

func (db *ConformanceTestStateDb) Exist(common.Address) bool {
	panic("should not be needed")
}

func (db *ConformanceTestStateDb) Empty(common.Address) bool {
	panic("should not be needed")
}

func (db *ConformanceTestStateDb) PrepareAccessList(sender common.Address, dest *common.Address, precompiles []common.Address, txAccesses types.AccessList) {
	panic("should not be needed")
}

func (db *ConformanceTestStateDb) RevertToSnapshot(int) {
	panic("should not be needed")
}

func (db *ConformanceTestStateDb) Snapshot() int {
	panic("should not be needed")
}

func (db *ConformanceTestStateDb) AddPreimage(common.Hash, []byte) {
	panic("should not be needed")
}

func (db *ConformanceTestStateDb) ForEachStorage(common.Address, func(common.Hash, common.Hash) bool) error {
	panic("should not be needed")
}
