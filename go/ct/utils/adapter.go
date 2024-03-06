package utils

import (
	"fmt"

	cc "github.com/Fantom-foundation/Tosca/go/ct/common"
	"github.com/Fantom-foundation/Tosca/go/ct/st"
	"github.com/Fantom-foundation/Tosca/go/vm"
)

func ToVmParameters(state *st.State) vm.Parameters {

	code := make([]byte, state.Code.Length())
	state.Code.CopyTo(code)

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
		Input:     nil, // TODO: add call context information
		Value:     vm.Value(state.CallContext.Value.Bytes32be()),
		CodeHash:  nil,
		Code:      code,
	}
}

type ctRunContext struct {
	state *st.State
}

func (c *ctRunContext) AccountExists(addr vm.Address) bool {
	panic("not implemented")
}

func (c *ctRunContext) GetStorage(addr vm.Address, key vm.Key) vm.Word {
	k := cc.NewU256FromBytes(key[:]...)
	return c.state.Storage.Current[k].Bytes32be()
}

func (c *ctRunContext) SetStorage(addr vm.Address, key vm.Key, value vm.Word) vm.StorageStatus {
	k := cc.NewU256FromBytes(key[:]...)
	v := cc.NewU256FromBytes(value[:]...)
	original := vm.Word(c.state.Storage.Original[k].Bytes32be())
	current := vm.Word(c.state.Storage.Current[k].Bytes32be())
	c.state.Storage.Current[k] = v
	return vm.GetStorageStatus(original, current, value)
}

func (c *ctRunContext) GetBalance(addr vm.Address) vm.Value {
	panic("not implemented")
}

func (c *ctRunContext) GetCodeSize(addr vm.Address) int {
	panic("not implemented")
}

func (c *ctRunContext) GetCodeHash(addr vm.Address) vm.Hash {
	panic("not implemented")
}

func (c *ctRunContext) GetCode(addr vm.Address) []byte {
	panic("not implemented")
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

func (c *ctRunContext) EmitLog(addr vm.Address, topics []vm.Hash, data []byte) {
	var ctTopics []cc.U256
	for _, topic := range topics {
		ctTopics = append(ctTopics, cc.NewU256FromBytes(topic[:]...))
	}
	// TODO: also log the address the log was emitted for!
	c.state.Logs.AddLog(data, ctTopics...)
}

func (c *ctRunContext) Call(kind vm.CallKind, parameter vm.CallParameter) (vm.CallResult, error) {
	panic("not implemented")
}

func (c *ctRunContext) SelfDestruct(addr vm.Address, beneficiary vm.Address) bool {
	panic("not implemented")
}

func (c *ctRunContext) AccessAccount(addr vm.Address) vm.AccessStatus {
	panic("not implemented")
}

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
	return c.state.Storage.Original[k].Bytes32be()
}

func (c *ctRunContext) IsAddressInAccessList(addr vm.Address) bool {
	panic("not implemented")
}

func (c *ctRunContext) IsSlotInAccessList(addr vm.Address, key vm.Key) (addressPresent, slotPresent bool) {
	return true, c.state.Storage.IsWarm(cc.NewU256FromBytes(key[:]...))
}

func (c *ctRunContext) HasSelfDestructed(addr vm.Address) bool {
	panic("not implemented")
}
