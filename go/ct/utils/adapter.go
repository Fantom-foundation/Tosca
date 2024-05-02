//
// Copyright (c) 2024 Fantom Foundation
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at fantom.foundation/bsl11.
//
// Change Date: 2028-4-16
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by the GNU Lesser General Public Licence v3
//

package utils

import (
	"fmt"

	cc "github.com/Fantom-foundation/Tosca/go/ct/common"
	"github.com/Fantom-foundation/Tosca/go/ct/st"
	"github.com/Fantom-foundation/Tosca/go/vm"
)

// ToVmParameters converts the given state into a set of interpreter Parameters.
// The resulting parameters depends partially on the internal state of the
// provided CT state. It should thus not be modified during the life-time of
// the resulting parameter set. Furthermore, when providing these parameters
// to an Interpreter, effects of state modifications are passed on to the given
// state. Thus, effects of an execution can be observed in the provided state
// after the execution of an interpreter.
func ToVmParameters(state *st.State) vm.Parameters {

	var code []byte
	if state.Code != nil {
		code = state.Code.Copy()
	}

	var revision vm.Revision
	switch state.Revision {
	case cc.R07_Istanbul:
		revision = vm.R07_Istanbul
	case cc.R09_Berlin:
		revision = vm.R09_Berlin
	case cc.R10_London:
		revision = vm.R10_London
	default:
		panic(fmt.Errorf("unsupported revision: %v", state.Revision))
	}

	return vm.Parameters{
		Context:   &ctRunContext{state},
		Revision:  revision,
		Kind:      vm.Call,
		Static:    state.ReadOnly,
		Depth:     0,
		Gas:       vm.Gas(state.Gas),
		Recipient: vm.Address(state.CallContext.AccountAddress),
		Sender:    vm.Address(state.CallContext.CallerAddress),
		Input:     state.CallData.ToBytes(),
		Value:     vm.Value(state.CallContext.Value.Bytes32be()),
		CodeHash:  nil,
		Code:      code,
	}
}

// ctRunContext adapts a st.State to the vm.RunContext interface utilized
// by Tosca Interpreter implementations. In particular, it makes global state
// information like Storage visible and mutable to Interpreters.
type ctRunContext struct {
	state *st.State
}

// TODO: add unit test
func (c *ctRunContext) AccountExists(addr vm.Address) bool {
	return c.state.Accounts.Exist(addr)
}

func (c *ctRunContext) GetStorage(addr vm.Address, key vm.Key) vm.Word {
	k := cc.NewU256FromBytes(key[:]...)
	return c.state.Storage.GetCurrent(k).Bytes32be()
}

// TODO: add unit test
func (c *ctRunContext) SetStorage(addr vm.Address, key vm.Key, value vm.Word) vm.StorageStatus {
	k := cc.NewU256FromBytes(key[:]...)
	v := cc.NewU256FromBytes(value[:]...)
	original := vm.Word(c.state.Storage.GetOriginal(k).Bytes32be())
	current := vm.Word(c.state.Storage.GetCurrent(k).Bytes32be())
	c.state.Storage.SetCurrent(k, v)
	return vm.GetStorageStatus(original, current, value)
}

func (c *ctRunContext) GetBalance(addr vm.Address) vm.Value {
	balance := c.state.Accounts.GetBalance(addr)
	return vm.Value(balance.Bytes32be())
}

func (c *ctRunContext) GetCodeSize(addr vm.Address) int {
	return c.state.Accounts.GetCode(addr).Length()
}

func (c *ctRunContext) GetCodeHash(addr vm.Address) vm.Hash {
	return c.state.Accounts.GetCodeHash(addr)
}

func (c *ctRunContext) GetCode(addr vm.Address) []byte {
	return c.state.Accounts.GetCode(addr).ToBytes()
}

func (c *ctRunContext) GetTransactionContext() vm.TransactionContext {
	return vm.TransactionContext{
		GasPrice:    c.state.BlockContext.GasPrice.Bytes32be(),
		Origin:      vm.Address(c.state.CallContext.OriginAddress),
		Coinbase:    vm.Address(c.state.BlockContext.CoinBase),
		BlockNumber: int64(c.state.BlockContext.BlockNumber),
		Timestamp:   int64(c.state.BlockContext.TimeStamp),
		GasLimit:    vm.Gas(c.state.BlockContext.GasLimit),
		PrevRandao:  vm.Hash(c.state.BlockContext.Difficulty.Bytes32be()),
		ChainID:     c.state.BlockContext.ChainID.Bytes32be(),
		BaseFee:     c.state.BlockContext.BaseFee.Bytes32be(),
	}
}

func (c *ctRunContext) GetBlockHash(number int64) vm.Hash {
	panic("not implemented")
}

// TODO: add unit test
func (c *ctRunContext) EmitLog(addr vm.Address, topics []vm.Hash, data []byte) {
	var ctTopics []cc.U256
	for _, topic := range topics {
		ctTopics = append(ctTopics, cc.NewU256FromBytes(topic[:]...))
	}
	// TODO: also log the address the log was emitted for!
	c.state.Logs.AddLog(data, ctTopics...)
}

// TODO: add unit test
func (c *ctRunContext) Call(kind vm.CallKind, parameter vm.CallParameter) (vm.CallResult, error) {
	res := c.state.CallJournal.Call(kind, parameter)
	return vm.CallResult{
		Success:   res.Success,
		Output:    res.Output,
		GasLeft:   res.GasLeft,
		GasRefund: res.GasRefund,
	}, nil
}

func (c *ctRunContext) SelfDestruct(addr vm.Address, beneficiary vm.Address) bool {
	panic("not implemented")
}

func (c *ctRunContext) AccessAccount(addr vm.Address) vm.AccessStatus {
	warm := c.state.Accounts.IsWarm(addr)
	c.state.Accounts.MarkWarm(addr)
	if warm {
		return vm.WarmAccess
	}
	return vm.ColdAccess
}

// TODO: add unit test
func (c *ctRunContext) AccessStorage(addr vm.Address, key vm.Key) vm.AccessStatus {
	k := cc.NewU256FromBytes(key[:]...)
	isWarm := c.state.Storage.IsWarm(k)
	c.state.Storage.MarkWarm(k)
	if isWarm {
		return vm.WarmAccess
	}
	return vm.ColdAccess
}

// --- legacy API ---

func (c *ctRunContext) GetCommittedStorage(addr vm.Address, key vm.Key) vm.Word {
	k := cc.NewU256FromBytes(key[:]...)
	return c.state.Storage.GetOriginal(k).Bytes32be()
}

func (c *ctRunContext) IsAddressInAccessList(addr vm.Address) bool {
	return c.state.Accounts.IsWarm(addr)
}

func (c *ctRunContext) IsSlotInAccessList(addr vm.Address, key vm.Key) (addressPresent, slotPresent bool) {
	return true, c.state.Storage.IsWarm(cc.NewU256FromBytes(key[:]...))
}

func (c *ctRunContext) HasSelfDestructed(addr vm.Address) bool {
	panic("not implemented")
}
