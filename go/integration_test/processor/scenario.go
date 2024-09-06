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
	"slices"
	"strings"
	"testing"

	"github.com/Fantom-foundation/Tosca/go/integration_test"
	"github.com/Fantom-foundation/Tosca/go/tosca"
)

// Scenario represents a test scenario for a transaction processor. A scenario
// consists of a world state before and after the operation, a transaction to
// be executed, block chain parameters, and the expected receipt.
type Scenario struct {
	Before      WorldState
	After       WorldState
	Parameters  tosca.BlockParameters
	Transaction tosca.Transaction
	Receipt     tosca.Receipt
	OperaError  error
}

func (s *Scenario) Run(t *testing.T, processor tosca.Processor) {

	context := newScenarioContext(s.Before)
	receipt, err := processor.Run(s.Parameters, s.Transaction, context)
	if err != nil && s.OperaError == nil {
		t.Fatalf("failed to run transaction: %v", err)
	}
	if s.OperaError != nil && receipt.Success {
		t.Fatalf("expected error, got success")
	}

	// check the world state after the operation
	if want, got := s.After, context.current; !want.Equal(got) {
		diff := strings.Join(got.Diff(want), "\n\t")
		t.Fatalf("unexpected world state after the operation: \n\t%v", diff)
	}

	// check the receipt
	if want, got := s.Receipt.Success, receipt.Success; want != got {
		t.Errorf("unexpected success, want %v, got %v", want, got)
	}
	if want, got := s.Receipt.GasUsed, receipt.GasUsed; want != got {
		t.Errorf("unexpected gas used, want %v, got %v", want, got)
	}
	if want, got := s.Receipt.BlobGasUsed, receipt.BlobGasUsed; want != got {
		t.Errorf("unexpected blob gas used, want %v, got %v", want, got)
	}
	if want, got := s.Receipt.Output, receipt.Output; !bytes.Equal(want, got) {
		t.Errorf("unexpected output used, want %x, got %x", want, got)
	}

	wantedCreatedContract := s.Receipt.ContractAddress
	gotCreatedContract := receipt.ContractAddress
	if wantedCreatedContract == nil && gotCreatedContract != nil {
		t.Errorf("unexpected created contract address, want nil, got %v", gotCreatedContract)
	}
	if wantedCreatedContract != nil && gotCreatedContract == nil {
		t.Errorf("unexpected created contract address, want %v, got nil", wantedCreatedContract)
	}
	if wantedCreatedContract != nil && gotCreatedContract != nil {
		if want, got := *wantedCreatedContract, *gotCreatedContract; want != got {
			t.Errorf("unexpected created contract address, want %v, got %v", want, got)
		}
	}

	if len(receipt.Logs) != len(s.Receipt.Logs) {
		t.Fatalf("unexpected receipt logs: %v", receipt.Logs)
	} else {
		for i, want := range s.Receipt.Logs {
			got := receipt.Logs[i]
			if want, got := want.Address, got.Address; want != got {
				t.Errorf("unexpected receipt log address, want %v, got %v", want, got)
			}
			if want, got := want.Topics, got.Topics; !slices.Equal(want, got) {
				t.Errorf("unexpected receipt log topics, want %v, got %v", want, got)
			}
			if want, got := want.Data, got.Data; !bytes.Equal(want, got) {
				t.Errorf("unexpected receipt data, want %x, got %x", want, got)
			}
		}
	}
}

func (s *Scenario) Clone() Scenario {
	return Scenario{
		Before:      s.Before.Clone(),
		After:       s.After.Clone(),
		Parameters:  s.Parameters,
		Transaction: s.Transaction,
		Receipt:     s.Receipt,
		OperaError:  s.OperaError,
	}
}

// ----------------------------------------------------------------------------

// scenarioContext implements the tosca.WorldState interface facilitating the
// interaction with a test-case specific context.
type scenarioContext struct {
	original   WorldState
	current    WorldState
	logs       []tosca.Log
	undo       []func()
	accessList []tosca.AccessTuple
}

func NewScenarioContext() *scenarioContext {
	return &scenarioContext{
		original: WorldState{},
		current:  WorldState{},
	}
}

func newScenarioContext(initial WorldState) *scenarioContext {
	return &scenarioContext{
		original: initial,
		current:  initial.Clone(),
	}
}

func (c *scenarioContext) AccountExists(addr tosca.Address) bool {
	return c.GetBalance(addr) != tosca.Value{} || c.GetNonce(addr) != 0 || c.GetCodeSize(addr) != 0
}

func (c *scenarioContext) GetBalance(addr tosca.Address) tosca.Value {
	return c.current[addr].Balance
}

func (c *scenarioContext) SetBalance(addr tosca.Address, value tosca.Value) {
	original := c.current[addr]
	modified := original
	modified.Balance = value
	c.current[addr] = modified
	c.undo = append(c.undo, func() { c.current[addr] = original })
}

func (c *scenarioContext) GetNonce(addr tosca.Address) uint64 {
	return c.current[addr].Nonce
}

func (c *scenarioContext) SetNonce(addr tosca.Address, value uint64) {
	original := c.current[addr]
	modified := original
	modified.Nonce = value
	c.current[addr] = modified
	c.undo = append(c.undo, func() { c.current[addr] = original })
}

func (c *scenarioContext) GetCode(addr tosca.Address) tosca.Code {
	return tosca.Code(bytes.Clone(c.current[addr].Code))
}

func (c *scenarioContext) GetCodeHash(addr tosca.Address) tosca.Hash {
	return integration_test.Keccak256Hash(c.GetCode(addr))
}

func (c *scenarioContext) GetCodeSize(addr tosca.Address) int {
	return len(c.GetCode(addr))
}

func (c *scenarioContext) SetCode(addr tosca.Address, code tosca.Code) {
	original := c.current[addr]
	modified := original
	modified.Code = tosca.Code(bytes.Clone(code))
	c.current[addr] = modified
	c.undo = append(c.undo, func() { c.current[addr] = original })
}

func (c *scenarioContext) GetStorage(addr tosca.Address, key tosca.Key) tosca.Word {
	return c.current[addr].Storage[key]
}

func (c *scenarioContext) SetStorage(addr tosca.Address, key tosca.Key, new tosca.Word) tosca.StorageStatus {
	original := c.original[addr].Storage[key]
	current := c.current[addr].Storage[key]

	account := c.current[addr]
	if account.Storage == nil {
		account.Storage = Storage{}
		c.current[addr] = account
	}

	c.current[addr].Storage[key] = new
	c.undo = append(c.undo, func() { c.current[addr].Storage[key] = current })
	return tosca.GetStorageStatus(original, current, new)
}

func (c *scenarioContext) SelfDestruct(addr tosca.Address, beneficiary tosca.Address) bool {
	panic("implement me")
}

func (c *scenarioContext) CreateSnapshot() tosca.Snapshot {
	return tosca.Snapshot(len(c.undo))
}

func (c *scenarioContext) RestoreSnapshot(snapshot tosca.Snapshot) {
	for len(c.undo) > int(snapshot) {
		c.undo[len(c.undo)-1]()
		c.undo = c.undo[:len(c.undo)-1]
	}
}

func (c *scenarioContext) GetTransientStorage(tosca.Address, tosca.Key) tosca.Word {
	panic("implement me")
}

func (c *scenarioContext) SetTransientStorage(tosca.Address, tosca.Key, tosca.Word) {
	panic("implement me")
}

func (c *scenarioContext) AccessAccount(address tosca.Address) tosca.AccessStatus {
	for _, tuple := range c.accessList {
		if tuple.Address == address {
			return tosca.WarmAccess
		}
	}
	c.accessList = append(c.accessList, tosca.AccessTuple{Address: address})
	return tosca.ColdAccess
}

func (c *scenarioContext) AccessStorage(addr tosca.Address, key tosca.Key) tosca.AccessStatus {
	for _, tuple := range c.accessList {
		if tuple.Address == addr {
			for _, k := range tuple.Keys {
				if k == key {
					return tosca.WarmAccess
				}
			}
			tuple.Keys = append(tuple.Keys, key)
			return tosca.ColdAccess
		}
	}
	c.accessList = append(c.accessList, tosca.AccessTuple{Address: addr, Keys: []tosca.Key{key}})
	return tosca.ColdAccess
}

func (c *scenarioContext) EmitLog(log tosca.Log) {
	len := len(c.logs)
	c.logs = append(c.logs, log)
	c.undo = append(c.undo, func() { c.logs = c.logs[:len] })
}

func (c *scenarioContext) GetLogs() []tosca.Log {
	return slices.Clone(c.logs)
}

func (c *scenarioContext) GetBlockHash(number int64) tosca.Hash {
	panic("implement me")
}

func (c *scenarioContext) GetCommittedStorage(addr tosca.Address, key tosca.Key) tosca.Word {
	return c.original[addr].Storage[key]
}

func (c *scenarioContext) IsAddressInAccessList(address tosca.Address) bool {
	for _, tuple := range c.accessList {
		if tuple.Address == address {
			return true
		}
	}
	return false
}

func (c *scenarioContext) IsSlotInAccessList(addr tosca.Address, key tosca.Key) (addressPresent, slotPresent bool) {
	for _, tuple := range c.accessList {
		if tuple.Address == addr {
			for _, k := range tuple.Keys {
				if k == key {
					return true, true
				}
			}
			return true, false
		}
	}
	return false, false
}

func (c *scenarioContext) HasSelfDestructed(addr tosca.Address) bool {
	panic("implement me")
}
