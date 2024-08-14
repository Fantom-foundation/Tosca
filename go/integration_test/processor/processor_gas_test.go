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

func gasTestScenarios() map[string]Scenario {
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

	testCases := make(map[string]Scenario)
	for name, exactScenario := range exactTestCases {
		gasTests := exactSufficientAndInsufficientScenarios(exactScenario, name)
		maps.Copy(testCases, gasTests)
	}
	return testCases
}

func gasLimitTestCases() map[string]Scenario {
	// cost for 2 PUSH1 operations
	const executionGasCost = 3 + 3

	cases := map[string]struct {
		gasLimit   tosca.Gas
		receipt    tosca.Receipt
		OperaError error
	}{
		"SimpleCodeExact": {
			gasLimit: floria.TxGas + executionGasCost,
			receipt: tosca.Receipt{
				Success: true,
				GasUsed: floria.TxGas + executionGasCost,
			},
		},
		"SimpleCodeSufficient": {
			gasLimit: floria.TxGas + executionGasCost + 100,
			receipt: tosca.Receipt{
				Success: true,
				GasUsed: floria.TxGas + executionGasCost + 100/10,
			},
		},
		"SimpleCodeInsufficient": {
			gasLimit: floria.TxGas + executionGasCost - 1,
			receipt: tosca.Receipt{
				Success: false,
				GasUsed: floria.TxGas + executionGasCost - 1,
			},
			OperaError: fmt.Errorf("gas too low"),
		},
	}

	code := tosca.Code{
		byte(op.PUSH1), byte(0), // < PUSH 0
		byte(op.PUSH1), byte(0), // < PUSH 0
		byte(op.RETURN),
	}
	before := WorldState{
		{1}: Account{Balance: tosca.NewValue(100), Nonce: 4},
		{2}: Account{Balance: tosca.NewValue(0),
			Code: code,
		},
	}
	after := WorldState{
		{1}: Account{Balance: tosca.NewValue(100), Nonce: 5},
		{2}: Account{Balance: tosca.NewValue(0),
			Code: code,
		},
	}

	testCases := make(map[string]Scenario, len(cases))
	for name, test := range cases {
		transaction := tosca.Transaction{
			Sender:    tosca.Address{1},
			Recipient: &tosca.Address{2},
			GasLimit:  test.gasLimit,
			Nonce:     4,
		}

		testCases[name] = Scenario{
			Before:      before,
			Transaction: transaction,
			After:       after,
			Receipt:     test.receipt,
			OperaError:  test.OperaError,
		}
	}

	return testCases
}

func gasPricingTestCases() map[string]Scenario {
	gasPrice := uint64(10)
	sender := tosca.Address{1}
	tests := map[string]struct {
		Before     Account
		After      Account
		Receipt    tosca.Receipt
		OperaError error
	}{
		"GasPriceCalculation": {
			Before: Account{Balance: tosca.NewValue(floria.TxGas * gasPrice), Nonce: 4},
			After:  Account{Balance: tosca.NewValue(0), Nonce: 5},
			Receipt: tosca.Receipt{
				Success: true,
				GasUsed: floria.TxGas,
			},
		},
		"GasPriceCalculationExcessBalance": {
			Before: Account{Balance: tosca.NewValue(floria.TxGas*gasPrice + 100), Nonce: 4},
			After:  Account{Balance: tosca.NewValue(100), Nonce: 5},
			Receipt: tosca.Receipt{
				Success: true,
				GasUsed: floria.TxGas,
			},
		},
		"GasPriceCalculationInsufficientBalance": {
			Before: Account{Balance: tosca.NewValue(floria.TxGas*gasPrice - 1), Nonce: 4},
			After:  Account{Balance: tosca.NewValue(floria.TxGas*gasPrice - 1), Nonce: 4},
			Receipt: tosca.Receipt{
				Success: false,
				GasUsed: 0,
			},
			OperaError: fmt.Errorf("insufficient balance"),
		},
	}

	transaction := tosca.Transaction{
		Sender:    sender,
		Recipient: &tosca.Address{2},
		GasLimit:  floria.TxGas,
		GasPrice:  tosca.NewValue(gasPrice),
		Nonce:     4,
	}

	testCases := make(map[string]Scenario, len(tests))
	for name, test := range tests {
		testCases[name] = Scenario{
			Before:      WorldState{sender: test.Before},
			Transaction: transaction,
			After:       WorldState{sender: test.After},
			Receipt:     test.Receipt,
			OperaError:  test.OperaError,
		}
	}

	return testCases
}

func gasSpecificTestCases() map[string]Scenario {
	cases := map[string]Scenario{
		"InternalCallDoesNotConsume10PercentOfRemainingGas": {
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
		},
	}
	return cases
}

func getGasTestScenarios() map[string]Scenario {
	testCases := gasTestScenarios()

	specificCases := gasLimitTestCases()
	maps.Copy(testCases, specificCases)

	refundCases := gasPricingTestCases()
	maps.Copy(testCases, refundCases)

	specificCases = gasSpecificTestCases()
	maps.Copy(testCases, specificCases)

	return testCases
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
