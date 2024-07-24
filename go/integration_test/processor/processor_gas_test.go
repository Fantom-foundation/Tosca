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
	"fmt"
	"maps"
	"testing"

	"github.com/Fantom-foundation/Tosca/go/processor/floria"
	"github.com/Fantom-foundation/Tosca/go/tosca"
	op "github.com/ethereum/go-ethereum/core/vm"
)

func getGasTestScenarios() map[string]Scenario {
	// cost for 2 PUSH1 operations
	const executionGasCost = 3 + 3

	exactTestCases := map[string]Scenario{
		"ValueTransfer": {
			Before: WorldState{
				{1}: Account{Balance: tosca.NewValue(100), Nonce: 4},
			},
			Transaction: tosca.Transaction{
				Sender:    tosca.Address{1},
				Recipient: &tosca.Address{2},
				GasLimit:  floria.TxGas,
				Value:     tosca.NewValue(3),
				Nonce:     4,
			},
			After: WorldState{
				{1}: Account{Balance: tosca.NewValue(97), Nonce: 5},
				{2}: Account{Balance: tosca.NewValue(3)},
			},
			Receipt: tosca.Receipt{
				Success: true,
				GasUsed: floria.TxGas,
			},
		},
		"InputZeros": {
			Before: WorldState{
				{1}: Account{Balance: tosca.NewValue(100), Nonce: 4},
				{2}: Account{Balance: tosca.NewValue(0)},
			},
			Transaction: tosca.Transaction{
				Sender:    tosca.Address{1},
				Recipient: &tosca.Address{2},
				GasLimit:  floria.TxGas + floria.TxDataZeroGasEIP2028*10,
				Nonce:     4,
				Input:     []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
			},
			After: WorldState{
				{1}: Account{Balance: tosca.NewValue(100), Nonce: 5},
				{2}: Account{Balance: tosca.NewValue(0)},
			},
			Receipt: tosca.Receipt{
				Success: true,
				GasUsed: floria.TxGas + floria.TxDataZeroGasEIP2028*10,
			},
		},
		"InputNonZeros": {
			Before: WorldState{
				{1}: Account{Balance: tosca.NewValue(100), Nonce: 4},
				{2}: Account{Balance: tosca.NewValue(0)},
			},
			Transaction: tosca.Transaction{
				Sender:    tosca.Address{1},
				Recipient: &tosca.Address{2},
				GasLimit:  floria.TxGas + floria.TxDataNonZeroGasEIP2028*10,
				Nonce:     4,
				Input:     []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10},
			},
			After: WorldState{
				{1}: Account{Balance: tosca.NewValue(100), Nonce: 5},
				{2}: Account{Balance: tosca.NewValue(0)},
			},
			Receipt: tosca.Receipt{
				Success: true,
				GasUsed: floria.TxGas + floria.TxDataNonZeroGasEIP2028*10,
			},
		},
		"AccessListOnlyAddresses": {
			Before: WorldState{
				{1}: Account{Balance: tosca.NewValue(100), Nonce: 4},
				{2}: Account{Balance: tosca.NewValue(0)},
			},
			Transaction: tosca.Transaction{
				Sender:    tosca.Address{1},
				Recipient: &tosca.Address{2},
				GasLimit:  floria.TxGas + floria.TxAccessListAddressGas*2,
				Nonce:     4,
				AccessList: []tosca.AccessTuple{
					{Address: tosca.Address{1},
						Keys: []tosca.Key{},
					},
					{Address: tosca.Address{2},
						Keys: []tosca.Key{},
					},
				},
			},
			After: WorldState{
				{1}: Account{Balance: tosca.NewValue(100), Nonce: 5},
				{2}: Account{Balance: tosca.NewValue(0)},
			},
			Receipt: tosca.Receipt{
				Success: true,
				GasUsed: floria.TxGas + floria.TxAccessListAddressGas*2,
			},
		},
		"AccessList": {
			Before: WorldState{
				{1}: Account{Balance: tosca.NewValue(100), Nonce: 4},
				{2}: Account{Balance: tosca.NewValue(0)},
			},
			Transaction: tosca.Transaction{
				Sender:    tosca.Address{1},
				Recipient: &tosca.Address{2},
				GasLimit:  floria.TxGas + floria.TxAccessListAddressGas*2 + floria.TxAccessListStorageKeyGas*5,
				Nonce:     4,
				AccessList: []tosca.AccessTuple{
					{Address: tosca.Address{1},
						Keys: []tosca.Key{{1}, {2}},
					},
					{Address: tosca.Address{2},
						Keys: []tosca.Key{{1}, {2}, {3}},
					},
				},
			},
			After: WorldState{
				{1}: Account{Balance: tosca.NewValue(100), Nonce: 5},
				{2}: Account{Balance: tosca.NewValue(0)},
			},
			Receipt: tosca.Receipt{
				Success: true,
				GasUsed: floria.TxGas + floria.TxAccessListAddressGas*2 + floria.TxAccessListStorageKeyGas*5,
			},
		},
	}

	allTestCases := make(map[string]Scenario)
	for name, exactScenario := range exactTestCases {
		gasTests := exactSufficientAndInsufficientScenarios(exactScenario, name)
		maps.Copy(allTestCases, gasTests)
	}

	allTestCases["SimpleCodeExact"] = Scenario{
		Before: WorldState{
			{1}: Account{Balance: tosca.NewValue(100), Nonce: 4},
			{2}: Account{Balance: tosca.NewValue(0),
				Code: tosca.Code{
					byte(op.PUSH1), byte(0), // < PUSH 0
					byte(op.PUSH1), byte(0), // < PUSH 0
					byte(op.RETURN),
				},
			},
		},
		Transaction: tosca.Transaction{
			Sender:    tosca.Address{1},
			Recipient: &tosca.Address{2},
			GasLimit:  floria.TxGas + executionGasCost,
			Nonce:     4,
		},
		After: WorldState{
			{1}: Account{Balance: tosca.NewValue(100), Nonce: 5},
			{2}: Account{Balance: tosca.NewValue(0),
				Code: tosca.Code{
					byte(op.PUSH1), byte(0), // < PUSH 0
					byte(op.PUSH1), byte(0), // < PUSH 0
					byte(op.RETURN),
				},
			},
		},
		Receipt: tosca.Receipt{
			Success: true,
			GasUsed: floria.TxGas + executionGasCost,
		},
	}
	allTestCases["SimpleCodeSufficient"] = Scenario{
		Before: WorldState{
			{1}: Account{Balance: tosca.NewValue(100), Nonce: 4},
			{2}: Account{Balance: tosca.NewValue(0),
				Code: tosca.Code{
					byte(op.PUSH1), byte(0), // < PUSH 0
					byte(op.PUSH1), byte(0), // < PUSH 0
					byte(op.RETURN),
				},
			},
		},
		Transaction: tosca.Transaction{
			Sender:    tosca.Address{1},
			Recipient: &tosca.Address{2},
			GasLimit:  floria.TxGas + executionGasCost + 100,
			Nonce:     4,
		},
		After: WorldState{
			{1}: Account{Balance: tosca.NewValue(100), Nonce: 5},
			{2}: Account{Balance: tosca.NewValue(0),
				Code: tosca.Code{
					byte(op.PUSH1), byte(0), // < PUSH 0
					byte(op.PUSH1), byte(0), // < PUSH 0
					byte(op.RETURN),
				},
			},
		},
		Receipt: tosca.Receipt{
			Success: true,
			GasUsed: floria.TxGas + executionGasCost + 100/10,
		},
	}
	allTestCases["SimpleCodeInsufficient"] = Scenario{
		Before: WorldState{
			{1}: Account{Balance: tosca.NewValue(100), Nonce: 4},
			{2}: Account{Balance: tosca.NewValue(0),
				Code: tosca.Code{
					byte(op.PUSH1), byte(0), // < PUSH 0
					byte(op.PUSH1), byte(0), // < PUSH 0
					byte(op.RETURN),
				},
			},
		},
		Transaction: tosca.Transaction{
			Sender:    tosca.Address{1},
			Recipient: &tosca.Address{2},
			GasLimit:  floria.TxGas + executionGasCost - 1,
			Nonce:     4,
		},
		After: WorldState{
			{1}: Account{Balance: tosca.NewValue(100), Nonce: 5},
			{2}: Account{Balance: tosca.NewValue(0),
				Code: tosca.Code{
					byte(op.PUSH1), byte(0), // < PUSH 0
					byte(op.PUSH1), byte(0), // < PUSH 0
					byte(op.RETURN),
				},
			},
		},
		Receipt: tosca.Receipt{
			Success: false,
			GasUsed: floria.TxGas + executionGasCost - 1,
		},
		OperaError: fmt.Errorf("gas too low"),
	}

	allTestCases["InternalCallDoesNotConsume10RemainingPercentGas"] = Scenario{
		Before: WorldState{
			{}:  Account{Balance: tosca.NewValue(100), Nonce: 4},
			{2}: Account{Balance: tosca.NewValue(0)},
		},
		Transaction: tosca.Transaction{
			Sender:    tosca.Address{},
			Recipient: &tosca.Address{2},
			GasLimit:  floria.TxGas + 100,
			Nonce:     4,
		},
		After: WorldState{
			{}:  Account{Balance: tosca.NewValue(100), Nonce: 5},
			{2}: Account{Balance: tosca.NewValue(0)},
		},
		Receipt: tosca.Receipt{
			Success: true,
			GasUsed: floria.TxGas,
		},
	}

	return allTestCases
}

func exactSufficientAndInsufficientScenarios(exactScenario Scenario, name string) map[string]Scenario {
	const excessGas = 100

	sufficient := exactScenario.Clone()
	sufficient.Transaction.GasLimit += excessGas
	sufficient.Receipt.GasUsed += excessGas / 10 // 1/10th of any excess gas is always consumed

	insufficient := exactScenario.Clone()
	insufficient.Transaction.GasLimit -= 1
	insufficient.Receipt.Success = false
	insufficient.Receipt.GasUsed = insufficient.Transaction.GasLimit
	insufficient.OperaError = fmt.Errorf("gas too low")
	// Reset world state in case of failure
	beforeSender := insufficient.Before[insufficient.Transaction.Sender]
	insufficient.After[insufficient.Transaction.Sender] = beforeSender
	beforeReceiver := insufficient.Before[*insufficient.Transaction.Recipient]
	insufficient.After[*insufficient.Transaction.Recipient] = beforeReceiver

	return map[string]Scenario{
		name + "Exact":        exactScenario,
		name + "Sufficient":   sufficient,
		name + "Insufficient": insufficient,
	}
}

func TestProcessor_GasSpecificScenarios(t *testing.T) {
	for name, processor := range getProcessors() {
		t.Run(name, func(t *testing.T) {
			for name, s := range getGasTestScenarios() {
				t.Run(name, func(t *testing.T) {
					s.Run(t, processor)
				})
			}
		})
	}
}
