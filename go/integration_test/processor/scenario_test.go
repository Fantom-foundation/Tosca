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
	"reflect"
	"testing"

	"github.com/Fantom-foundation/Tosca/go/integration_test"
	"github.com/Fantom-foundation/Tosca/go/tosca"
)

func TestScenarioContext_AccountsAreImplictilyCreated(t *testing.T) {
	addr := tosca.Address{1}
	tests := map[string]func(tosca.WorldState){
		"balance": func(s tosca.WorldState) {
			s.SetBalance(addr, tosca.NewValue(100))
		},
		"nonce": func(s tosca.WorldState) {
			s.SetNonce(addr, 12)
		},
		"code": func(s tosca.WorldState) {
			s.SetCode(addr, tosca.Code{1, 2, 3})
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			context := newScenarioContext(WorldState{})

			if context.AccountExists(addr) {
				t.Errorf("test account should not exist")
			}
			test(context)
			if !context.AccountExists(addr) {
				t.Errorf("account should exist")
			}
		})
	}
}

func TestScenarioContext_BalanceManipulation(t *testing.T) {
	context := newScenarioContext(WorldState{})

	addr := tosca.Address{1}
	if want, got := (tosca.Value{}), context.GetBalance(addr); got != want {
		t.Errorf("unexpected balance, want %v, got %v", want, got)
	}

	snapshot := context.CreateSnapshot()

	context.SetBalance(addr, tosca.NewValue(100))
	if want, got := tosca.NewValue(100), context.GetBalance(addr); got != want {
		t.Errorf("unexpected balance, want %v, got %v", want, got)
	}

	context.RestoreSnapshot(snapshot)

	if want, got := (tosca.Value{}), context.GetBalance(addr); got != want {
		t.Errorf("unexpected balance, want %v, got %v", want, got)
	}
}

func TestScenarioContext_NonceManipulation(t *testing.T) {
	context := newScenarioContext(WorldState{})

	addr := tosca.Address{1}
	if want, got := uint64(0), context.GetNonce(addr); got != want {
		t.Errorf("unexpected nonce, want %v, got %v", want, got)
	}

	snapshot := context.CreateSnapshot()

	context.SetNonce(addr, 12)
	if want, got := uint64(12), context.GetNonce(addr); got != want {
		t.Errorf("unexpected nonce, want %v, got %v", want, got)
	}

	context.RestoreSnapshot(snapshot)

	if want, got := uint64(0), context.GetNonce(addr); got != want {
		t.Errorf("unexpected nonce, want %v, got %v", want, got)
	}
}

func TestScenarioContext_CodeManipulation(t *testing.T) {
	context := newScenarioContext(WorldState{})

	addr := tosca.Address{1}
	if want, got := (tosca.Code{}), context.GetCode(addr); !bytes.Equal(want, got) {
		t.Errorf("unexpected code, want %x, got %x", want, got)
	}

	snapshot := context.CreateSnapshot()

	context.SetCode(addr, tosca.Code{1, 2, 3})
	if want, got := (tosca.Code{1, 2, 3}), context.GetCode(addr); !bytes.Equal(want, got) {
		t.Errorf("unexpected code, want %x, got %x", want, got)
	}

	context.RestoreSnapshot(snapshot)

	if want, got := (tosca.Code{}), context.GetCode(addr); !bytes.Equal(want, got) {
		t.Errorf("unexpected code, want %x, got %x", want, got)
	}
}

func TestScenarioContext_StorageManipulation(t *testing.T) {
	context := newScenarioContext(WorldState{})

	addr := tosca.Address{1}
	key := tosca.Key{2}
	if want, got := (tosca.Word{}), context.GetStorage(addr, key); got != want {
		t.Errorf("unexpected storage, want %v, got %v", want, got)
	}

	snapshot := context.CreateSnapshot()

	if want, got := tosca.StorageAdded, context.SetStorage(addr, key, tosca.Word{12}); want != got {
		t.Errorf("unexpected storage change, want %v, got %v", want, got)
	}

	if want, got := (tosca.Word{12}), context.GetStorage(addr, key); got != want {
		t.Errorf("unexpected storage, want %v, got %v", want, got)
	}

	context.RestoreSnapshot(snapshot)

	if want, got := (tosca.Word{}), context.GetStorage(addr, key); got != want {
		t.Errorf("unexpected storage, want %v, got %v", want, got)
	}
}

func TestScenarioContext_CodeQuery(t *testing.T) {
	context := newScenarioContext(WorldState{})

	addr := tosca.Address{1}

	emptyHash := integration_test.Keccak256Hash(tosca.Code{})

	if want, got := emptyHash, context.GetCodeHash(addr); want != got {
		t.Errorf("unexpected code hash, want %x, got %x", want, got)
	}
	if want, got := 0, context.GetCodeSize(addr); want != got {
		t.Errorf("unexpected code length, want %x, got %x", want, got)
	}

	code := tosca.Code{1, 2, 3}
	codeHash := integration_test.Keccak256Hash(code)
	context.SetCode(addr, code)

	if want, got := codeHash, context.GetCodeHash(addr); want != got {
		t.Errorf("unexpected code hash, want %x, got %x", want, got)
	}
	if want, got := len(code), context.GetCodeSize(addr); want != got {
		t.Errorf("unexpected code length, want %x, got %x", want, got)
	}
}

func TestScenarioContext_LogManipulation(t *testing.T) {
	context := newScenarioContext(WorldState{})

	l1 := tosca.Log{Address: tosca.Address{1}}
	l2 := tosca.Log{Address: tosca.Address{2}}
	if want, got := 0, len(context.GetLogs()); want != got {
		t.Errorf("unexpected length of logs, want %v, got %v", want, got)
	}

	s1 := context.CreateSnapshot()

	context.EmitLog(l1)

	if want, got := []tosca.Log{l1}, context.GetLogs(); !reflect.DeepEqual(want, got) {
		t.Errorf("unexpected logs, want %v, got %v", want, got)
	}

	s2 := context.CreateSnapshot()

	context.EmitLog(l2)

	if want, got := []tosca.Log{l1, l2}, context.GetLogs(); !reflect.DeepEqual(want, got) {
		t.Errorf("unexpected logs, want %v, got %v", want, got)
	}

	context.RestoreSnapshot(s2)

	if want, got := []tosca.Log{l1}, context.GetLogs(); !reflect.DeepEqual(want, got) {
		t.Errorf("unexpected logs, want %v, got %v", want, got)
	}

	context.RestoreSnapshot(s1)

	if want, got := 0, len(context.GetLogs()); want != got {
		t.Errorf("unexpected length of logs, want %v, got %v", want, got)
	}
}
func TestScenario_Clone(t *testing.T) {
	tests := map[string]func(*Scenario){
		"before": func(s *Scenario) {
			s.Before = WorldState{{1}: Account{Balance: tosca.NewValue(100), Nonce: 4}}
		},
		"after": func(s *Scenario) {
			s.After = WorldState{{1}: Account{Balance: tosca.NewValue(100), Nonce: 4}}
		},
		"parameters": func(s *Scenario) {
			s.Parameters = tosca.BlockParameters{Timestamp: 42}
		},
		"transaction": func(s *Scenario) {
			s.Transaction = tosca.Transaction{Sender: tosca.Address{1}, Recipient: &tosca.Address{2}}
		},
		"receipt": func(s *Scenario) {
			s.Receipt = tosca.Receipt{Success: true}
		},
		"opera_error": func(s *Scenario) {
			s.OperaError = fmt.Errorf("test")
		},
	}

	equals := func(a, b Scenario) bool {
		return a.Before.Equal(b.Before) &&
			a.After.Equal(b.After) &&
			reflect.DeepEqual(a.Parameters, b.Parameters) &&
			reflect.DeepEqual(a.Transaction, b.Transaction) &&
			reflect.DeepEqual(a.Receipt, b.Receipt) &&
			a.OperaError == b.OperaError
	}

	scenario := Scenario{
		Before:      WorldState{},
		After:       WorldState{},
		Parameters:  tosca.BlockParameters{},
		Transaction: tosca.Transaction{},
		Receipt:     tosca.Receipt{},
		OperaError:  nil,
	}

	for name, change := range tests {
		t.Run(name, func(t *testing.T) {
			clone := scenario.Clone()

			if !equals(scenario, clone) {
				t.Errorf("clones are not equal")
			}

			change(&clone)

			if equals(scenario, clone) {
				t.Errorf("clones are not independent")
			}
		})
	}
}
