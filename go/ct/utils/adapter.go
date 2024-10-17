// Copyright (c) 2024 Fantom Foundation
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at fantom.foundation/bsl11.
//
// Change Date: 2028-4-16
//
// On the date above, in accordance with the Business Source License, use of
// this software will be governed by the GNU Lesser General Public License v3.

package utils

import (
	cc "github.com/Fantom-foundation/Tosca/go/ct/common"
	"github.com/Fantom-foundation/Tosca/go/ct/st"
	"github.com/Fantom-foundation/Tosca/go/tosca"
)

// ToVmParameters converts the given state into a set of interpreter Parameters.
// The resulting parameters depends partially on the internal state of the
// provided CT state. It should thus not be modified during the life-time of
// the resulting parameter set. Furthermore, when providing these parameters
// to an Interpreter, effects of state modifications are passed on to the given
// state. Thus, effects of an execution can be observed in the provided state
// after the execution of an interpreter.
func ToVmParameters(state *st.State) tosca.Parameters {

	var code []byte
	var codeHash tosca.Hash
	if state.Code != nil {
		code = state.Code.Copy()
		codeHash = state.Code.Hash()
	}

	transactionContext := state.TransactionContext
	if transactionContext == nil {
		transactionContext = &st.TransactionContext{}
	}

	return tosca.Parameters{
		BlockParameters: tosca.BlockParameters{
			ChainID:     state.BlockContext.ChainID.Bytes32be(),
			BlockNumber: int64(state.BlockContext.BlockNumber),
			Timestamp:   int64(state.BlockContext.TimeStamp),
			Coinbase:    state.BlockContext.CoinBase,
			GasLimit:    tosca.Gas(state.BlockContext.GasLimit),
			PrevRandao:  state.BlockContext.PrevRandao.Bytes32be(),
			BaseFee:     state.BlockContext.BaseFee.Bytes32be(),
			BlobBaseFee: state.BlockContext.BlobBaseFee.Bytes32be(),
			Revision:    state.Revision,
		},
		TransactionParameters: tosca.TransactionParameters{
			Origin:     transactionContext.OriginAddress,
			GasPrice:   tosca.Value(state.BlockContext.GasPrice.Bytes32be()),
			BlobHashes: transactionContext.BlobHashes,
		},
		Context:   &ctRunContext{state},
		Kind:      tosca.Call,
		Static:    state.ReadOnly,
		Depth:     0,
		Gas:       tosca.Gas(state.Gas),
		Recipient: tosca.Address(state.CallContext.AccountAddress),
		Sender:    tosca.Address(state.CallContext.CallerAddress),
		Input:     state.CallData.ToBytes(),
		Value:     tosca.Value(state.CallContext.Value.Bytes32be()),
		Code:      code,
		CodeHash:  &codeHash,
	}
}

// ctRunContext adapts a st.State to the tosca.RunContext interface utilized
// by Tosca Interpreter implementations. In particular, it makes global state
// information like Storage visible and mutable to Interpreters.
type ctRunContext struct {
	state *st.State
}

// TODO: add unit test
func (c *ctRunContext) AccountExists(addr tosca.Address) bool {
	return c.state.Accounts.Exists(addr)
}

func (c *ctRunContext) GetStorage(addr tosca.Address, key tosca.Key) tosca.Word {
	k := cc.NewU256FromBytes(key[:]...)
	return c.state.Storage.GetCurrent(k).Bytes32be()
}

// TODO: add unit test
func (c *ctRunContext) SetStorage(addr tosca.Address, key tosca.Key, value tosca.Word) tosca.StorageStatus {
	k := cc.NewU256FromBytes(key[:]...)
	v := cc.NewU256FromBytes(value[:]...)
	original := tosca.Word(c.state.Storage.GetOriginal(k).Bytes32be())
	current := tosca.Word(c.state.Storage.GetCurrent(k).Bytes32be())
	c.state.Storage.SetCurrent(k, v)
	return tosca.GetStorageStatus(original, current, value)
}

func (c *ctRunContext) GetTransientStorage(addr tosca.Address, key tosca.Key) tosca.Word {
	return c.state.TransientStorage.Get(cc.NewU256FromBytes(key[:]...)).Bytes32be()
}

func (c *ctRunContext) SetTransientStorage(addr tosca.Address, key tosca.Key, value tosca.Word) {
	c.state.TransientStorage.Set(cc.NewU256FromBytes(key[:]...), cc.NewU256FromBytes(value[:]...))
}

func (c *ctRunContext) GetBalance(addr tosca.Address) tosca.Value {
	balance := c.state.Accounts.GetBalance(addr)
	return tosca.Value(balance.Bytes32be())
}

func (c *ctRunContext) GetCodeSize(addr tosca.Address) int {
	return c.state.Accounts.GetCode(addr).Length()
}

func (c *ctRunContext) GetCodeHash(addr tosca.Address) tosca.Hash {
	return c.state.Accounts.GetCodeHash(addr)
}

func (c *ctRunContext) GetCode(addr tosca.Address) tosca.Code {
	return c.state.Accounts.GetCode(addr).ToBytes()
}

func (c *ctRunContext) SetCode(addr tosca.Address, code tosca.Code) {
	panic("not implemented")
}

func (c *ctRunContext) GetBlockHash(number int64) tosca.Hash {
	min := int64(0)
	max := int64(c.state.BlockContext.BlockNumber)
	if max > 256 {
		min = max - 256
	}
	if min > number || number >= max {
		return tosca.Hash{0x0}
	}
	return c.state.RecentBlockHashes.Get(uint64(max - number - 1))
}

// TODO: add unit test
func (c *ctRunContext) EmitLog(log tosca.Log) {
	var ctTopics []cc.U256
	for _, topic := range log.Topics {
		ctTopics = append(ctTopics, cc.NewU256FromBytes(topic[:]...))
	}
	// TODO: also log the address the log was emitted for!
	c.state.Logs.AddLog(log.Data, ctTopics...)
}

// TODO: add unit test
func (c *ctRunContext) Call(kind tosca.CallKind, parameter tosca.CallParameters) (tosca.CallResult, error) {
	return c.state.CallJournal.Call(kind, parameter), nil
}

func (c *ctRunContext) SelfDestruct(address tosca.Address, beneficiary tosca.Address) bool {
	c.state.SelfDestructedJournal = append(c.state.SelfDestructedJournal, st.NewSelfDestructEntry(address, beneficiary))
	if c.state.HasSelfDestructed {
		return false
	}
	c.state.HasSelfDestructed = true
	return true
}

func (c *ctRunContext) AccessAccount(addr tosca.Address) tosca.AccessStatus {
	warm := c.state.Accounts.IsWarm(addr)
	c.state.Accounts.MarkWarm(addr)
	if warm {
		return tosca.WarmAccess
	}
	return tosca.ColdAccess
}

// TODO: add unit test
func (c *ctRunContext) AccessStorage(addr tosca.Address, key tosca.Key) tosca.AccessStatus {
	k := cc.NewU256FromBytes(key[:]...)
	isWarm := c.state.Storage.IsWarm(k)
	c.state.Storage.MarkWarm(k)
	if isWarm {
		return tosca.WarmAccess
	}
	return tosca.ColdAccess
}

// --- legacy API ---

func (c *ctRunContext) GetCommittedStorage(addr tosca.Address, key tosca.Key) tosca.Word {
	k := cc.NewU256FromBytes(key[:]...)
	return c.state.Storage.GetOriginal(k).Bytes32be()
}

func (c *ctRunContext) IsAddressInAccessList(addr tosca.Address) bool {
	return c.state.Accounts.IsWarm(addr)
}

func (c *ctRunContext) IsSlotInAccessList(addr tosca.Address, key tosca.Key) (addressPresent, slotPresent bool) {
	return true, c.state.Storage.IsWarm(cc.NewU256FromBytes(key[:]...))
}

func (c *ctRunContext) HasSelfDestructed(addr tosca.Address) bool {
	return c.state.HasSelfDestructed
}

func (c *ctRunContext) SetBalance(tosca.Address, tosca.Value) {
	// -- ignored, since balances are not tracked in the context of a CT run --
}

func (c *ctRunContext) GetNonce(tosca.Address) uint64 {
	// Required to identify empty accounts. Nonces are not explicitly modeled
	// by the CT state, so they are always considered to be 0. Any other value
	// would make it impossible to have empty accounts.
	return 0
}

// --- API only needed in the context of a full transaction, which is not covered by CT ---

func (c *ctRunContext) CreateAccount(tosca.Address, tosca.Code) bool {
	panic("should not be needed")
}

func (c *ctRunContext) SetNonce(tosca.Address, uint64) {
	panic("should not be needed")
}

func (c *ctRunContext) CreateSnapshot() tosca.Snapshot {
	panic("should not be needed")
}

func (c *ctRunContext) RestoreSnapshot(tosca.Snapshot) {
	panic("should not be needed")
}

func (c *ctRunContext) GetLogs() []tosca.Log {
	panic("should not be needed")
}
