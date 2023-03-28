package lfvm

import (
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/holiman/uint256"
)

type slot struct {
	addr common.Address
	key  common.Hash
}

// A local shadow copy of the slot value state
var shadow_values = map[slot]common.Hash{}

var suicided_contracts = map[common.Address]int{}

func ClearShadowValues() {
	shadow_values = map[slot]common.Hash{}
	suicided_contracts = map[common.Address]int{}
}

type ShadowStateDB struct {
	state vm.StateDB
}

func (s ShadowStateDB) CreateAccount(common.Address) {
	// Ignored
}

func (s ShadowStateDB) SubBalance(common.Address, *big.Int) {
	// Ignored
}
func (s ShadowStateDB) AddBalance(common.Address, *big.Int) {
	// Ignored
}
func (s ShadowStateDB) GetBalance(addr common.Address) *big.Int {
	return s.state.GetBalance(addr)
}

func (s ShadowStateDB) GetNonce(addr common.Address) uint64 {
	return s.state.GetNonce(addr)
}
func (s ShadowStateDB) SetNonce(common.Address, uint64) {
	// Ignored
}

func (s ShadowStateDB) GetCodeHash(addr common.Address) common.Hash {
	return s.state.GetCodeHash(addr)
}
func (s ShadowStateDB) GetCode(addr common.Address) []byte {
	return s.state.GetCode(addr)
}
func (s ShadowStateDB) SetCode(common.Address, []byte) {
	// Ignored
}
func (s ShadowStateDB) GetCodeSize(addr common.Address) int {
	return s.state.GetCodeSize(addr)
}

func (s ShadowStateDB) AddRefund(amount uint64) {
	// Ignored
	fmt.Printf("SHADOW_DB: AddRefund %v\n", amount)
}
func (s ShadowStateDB) SubRefund(amount uint64) {
	// Ignored
	fmt.Printf("SHADOW_DB: SubRefund %v\n", amount)
}
func (s ShadowStateDB) GetRefund() uint64 {
	return s.state.GetRefund()
}

func (s ShadowStateDB) GetCommittedState(addr common.Address, key common.Hash) common.Hash {
	return s.state.GetCommittedState(addr, key)
}
func (s ShadowStateDB) GetState(addr common.Address, key common.Hash) common.Hash {
	fmt.Printf("SHADOW_DB: Reading %v %v\n", addr, key)
	value, present := shadow_values[slot{addr, key}]
	if present {
		return value
	}
	value = s.state.GetCommittedState(addr, key)
	shadow_values[slot{addr, key}] = value
	return value
}

func (s ShadowStateDB) SetState(addr common.Address, key common.Hash, value common.Hash) {
	fmt.Printf("SHADOW_DB: Writing %v %v\n", addr, key)
	if shadow_values == nil {
		shadow_values = make(map[slot]common.Hash)
	}
	shadow_values[slot{addr, key}] = value
}

func (s ShadowStateDB) Suicide(addr common.Address) bool {
	suicided_contracts[addr] = 0
	return true
}

func (s ShadowStateDB) HasSuicided(addr common.Address) bool {
	_, killed := suicided_contracts[addr]
	return killed || s.state.HasSuicided(addr)
}

func (s ShadowStateDB) Exist(addr common.Address) bool {
	return s.state.Exist(addr)
}
func (s ShadowStateDB) Empty(addr common.Address) bool {
	return s.state.Empty(addr)
}

func (s ShadowStateDB) PrepareAccessList(sender common.Address, dest *common.Address, precompiles []common.Address, txAccesses types.AccessList) {
	// Ignored
	fmt.Printf("SHADOW_DB: PrepareAccessList for %v\n", sender)
}
func (s ShadowStateDB) AddressInAccessList(addr common.Address) bool {
	fmt.Printf("SHADOW_DB: AddressInAccessList(%v)\n", addr)
	return s.state.AddressInAccessList(addr)
}
func (s ShadowStateDB) SlotInAccessList(addr common.Address, slot common.Hash) (addressOk bool, slotOk bool) {
	fmt.Printf("SHADOW_DB: SlotInAccessList(%v,%v)\n", addr, slot)
	return s.state.SlotInAccessList(addr, slot)
}
func (s ShadowStateDB) AddAddressToAccessList(addr common.Address) {
	// Ignored
	fmt.Printf("SHADOW_DB: AddAddressToAccessList(%v)\n", addr)
}
func (s ShadowStateDB) AddSlotToAccessList(addr common.Address, slot common.Hash) {
	// Ignored
	fmt.Printf("SHADOW_DB: AddSlotToAccessList(%v,%v)\n", addr, slot)
}

func (s ShadowStateDB) RevertToSnapshot(int) {
	// Ignored
	panic("Snapshot in shadow DB not supported")
}
func (s ShadowStateDB) Snapshot() int {
	panic("Snapshot in shadow DB not supported")
}

func (s ShadowStateDB) AddLog(*types.Log) {
	// Ignored
}
func (s ShadowStateDB) AddPreimage(common.Hash, []byte) {
	// Ignored
}

func (s ShadowStateDB) ForEachStorage(addr common.Address, op func(common.Hash, common.Hash) bool) error {
	return s.state.ForEachStorage(addr, op)
}

type CaptureCallContext struct {
	evm    *vm.EVM
	shadow *ShadowCallContext
}

func (c *CaptureCallContext) Call(env *vm.EVM, me vm.ContractRef, addr common.Address, data []byte, gas uint64, value *big.Int) ([]byte, uint64, error) {
	c.shadow.current_result, c.shadow.current_leftover_gas, c.shadow.current_error = c.evm.Call(me, addr, data, gas, value)
	return c.shadow.current_result, c.shadow.current_leftover_gas, c.shadow.current_error
}

func (c *CaptureCallContext) CallCode(env *vm.EVM, me vm.ContractRef, addr common.Address, data []byte, gas uint64, value *big.Int) ([]byte, uint64, error) {
	c.shadow.current_result, c.shadow.current_leftover_gas, c.shadow.current_error = c.evm.CallCode(me, addr, data, gas, value)
	return c.shadow.current_result, c.shadow.current_leftover_gas, c.shadow.current_error
}

func (c *CaptureCallContext) StaticCall(env *vm.EVM, me vm.ContractRef, addr common.Address, input []byte, gas uint64) ([]byte, uint64, error) {
	c.shadow.current_result, c.shadow.current_leftover_gas, c.shadow.current_error = c.evm.StaticCall(me, addr, input, gas)
	return c.shadow.current_result, c.shadow.current_leftover_gas, c.shadow.current_error
}

func (c *CaptureCallContext) DelegateCall(env *vm.EVM, me vm.ContractRef, addr common.Address, data []byte, gas uint64) ([]byte, uint64, error) {
	c.shadow.current_result, c.shadow.current_leftover_gas, c.shadow.current_error = c.evm.DelegateCall(me, addr, data, gas)
	return c.shadow.current_result, c.shadow.current_leftover_gas, c.shadow.current_error
}

func (c *CaptureCallContext) Create(env *vm.EVM, me vm.ContractRef, data []byte, gas uint64, value *big.Int) ([]byte, common.Address, uint64, error) {
	c.shadow.current_result, c.shadow.current_address, c.shadow.current_leftover_gas, c.shadow.current_error = c.evm.Create(me, data, gas, value)
	return c.shadow.current_result, c.shadow.current_address, c.shadow.current_leftover_gas, c.shadow.current_error
}

func (c *CaptureCallContext) Create2(env *vm.EVM, me vm.ContractRef, data []byte, gas uint64, endowment *big.Int, salt *uint256.Int) ([]byte, common.Address, uint64, error) {
	c.shadow.current_result, c.shadow.current_address, c.shadow.current_leftover_gas, c.shadow.current_error = c.evm.Create2(me, data, gas, endowment, salt)
	return c.shadow.current_result, c.shadow.current_address, c.shadow.current_leftover_gas, c.shadow.current_error
}

type ShadowCallContext struct {
	current_result       []byte
	current_leftover_gas uint64
	current_error        error
	current_address      common.Address
}

func (s *ShadowCallContext) Call(env *vm.EVM, me vm.ContractRef, addr common.Address, data []byte, gas uint64, value *big.Int) ([]byte, uint64, error) {
	return s.current_result, s.current_leftover_gas, s.current_error
}

func (s *ShadowCallContext) CallCode(env *vm.EVM, me vm.ContractRef, addr common.Address, data []byte, gas uint64, value *big.Int) ([]byte, uint64, error) {
	return s.current_result, s.current_leftover_gas, s.current_error
}

func (s *ShadowCallContext) StaticCall(env *vm.EVM, me vm.ContractRef, addr common.Address, input []byte, gas uint64) ([]byte, uint64, error) {
	return s.current_result, s.current_leftover_gas, s.current_error
}

func (s *ShadowCallContext) DelegateCall(env *vm.EVM, me vm.ContractRef, addr common.Address, data []byte, gas uint64) ([]byte, uint64, error) {
	return s.current_result, s.current_leftover_gas, s.current_error
}

func (s *ShadowCallContext) Create(env *vm.EVM, me vm.ContractRef, data []byte, gas uint64, value *big.Int) ([]byte, common.Address, uint64, error) {
	return s.current_result, s.current_address, s.current_leftover_gas, s.current_error
}

func (s *ShadowCallContext) Create2(env *vm.EVM, me vm.ContractRef, data []byte, gas uint64, endowment *big.Int, salt *uint256.Int) ([]byte, common.Address, uint64, error) {
	return s.current_result, s.current_address, s.current_leftover_gas, s.current_error
}
