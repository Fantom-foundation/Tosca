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
	Account   *st.Account
	Logs      *st.Logs
	revision  Revision
	gasRefund uint64
}

// NewConformanceTestStateDb creates a new ConformanceTestStateDb.
func NewConformanceTestStateDb(storage *st.Storage, account *st.Account, logs *st.Logs, revision Revision) *ConformanceTestStateDb {
	return &ConformanceTestStateDb{
		Storage:  storage,
		Account:  account,
		Logs:     logs,
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

func (db *ConformanceTestStateDb) GetBalance(addr common.Address) *big.Int {
	return db.Account.Balance[Address(addr)].ToBigInt()
}

func (db *ConformanceTestStateDb) AddressInAccessList(addr common.Address) bool {
	return db.Account.IsWarm(Address(addr))
}

func (db *ConformanceTestStateDb) AddAddressToAccessList(addr common.Address) {
	db.Account.MarkWarm(Address(addr))
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

func (db *ConformanceTestStateDb) AddLog(log *types.Log) {
	var topics []U256
	for _, topic := range log.Topics {
		topics = append(topics, NewU256FromBytes(topic[:]...))
	}
	db.Logs.AddLog(log.Data, topics...)
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

func (db *ConformanceTestStateDb) GetNonce(common.Address) uint64 {
	panic("should not be needed")
}

func (db *ConformanceTestStateDb) SetNonce(common.Address, uint64) {
	panic("should not be needed")
}

func (db *ConformanceTestStateDb) SetCode(common.Address, []byte) {
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
