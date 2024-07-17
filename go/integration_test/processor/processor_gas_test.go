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
	"strings"
	"testing"

	"github.com/Fantom-foundation/Tosca/go/processor/floria"
	"github.com/Fantom-foundation/Tosca/go/tosca"
	op "github.com/ethereum/go-ethereum/core/vm"
)

func getGasTestScenarios() map[string]Scenario {
	const (
		codePrice = 3 + 3
		excessGas = 100
	)

	allTestCases := map[string]Scenario{
		"ValueTransferExact": {
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
		"ValueTransferSufficient": {
			Before: WorldState{
				{1}: Account{Balance: tosca.NewValue(100), Nonce: 4},
			},
			Transaction: tosca.Transaction{
				Sender:    tosca.Address{1},
				Recipient: &tosca.Address{2},
				GasLimit:  floria.TxGas + excessGas,
				Value:     tosca.NewValue(3),
				Nonce:     4,
			},
			After: WorldState{
				{1}: Account{Balance: tosca.NewValue(97), Nonce: 5},
				{2}: Account{Balance: tosca.NewValue(3)},
			},
			Receipt: tosca.Receipt{
				Success: true,
				GasUsed: floria.TxGas + excessGas/10, // 1/10th of the gas is always consumed
			},
		},
		"ValueTransferInsufficient": {
			Before: WorldState{
				{1}: Account{Balance: tosca.NewValue(100), Nonce: 4},
			},
			Transaction: tosca.Transaction{
				Sender:    tosca.Address{1},
				Recipient: &tosca.Address{2},
				GasLimit:  floria.TxGas - 1,
				Value:     tosca.NewValue(3),
				Nonce:     4,
			},
			After: WorldState{
				{1}: Account{Balance: tosca.NewValue(100), Nonce: 4},
			},
			Receipt: tosca.Receipt{
				Success: false,
				GasUsed: 0,
			},
			OperaError: fmt.Errorf("gas too low"),
		},
		"SimpleCodeExact": {
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
				GasLimit:  floria.TxGas + codePrice,
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
				GasUsed: floria.TxGas + codePrice,
			},
		},
		"SimpleCodeSufficient": {
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
				GasLimit:  floria.TxGas + codePrice + excessGas,
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
				GasUsed: floria.TxGas + codePrice + excessGas/10,
			},
		},
		"SimpleCodeInsufficient": {
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
				GasLimit:  floria.TxGas + codePrice - 1,
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
				GasUsed: floria.TxGas + codePrice - 1,
			},
			OperaError: fmt.Errorf("gas too low"),
		},
		"InputZerosExact": {
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
		"InputZerosSufficient": {
			Before: WorldState{
				{1}: Account{Balance: tosca.NewValue(100), Nonce: 4},
				{2}: Account{Balance: tosca.NewValue(0)},
			},
			Transaction: tosca.Transaction{
				Sender:    tosca.Address{1},
				Recipient: &tosca.Address{2},
				GasLimit:  floria.TxGas + floria.TxDataZeroGasEIP2028*10 + excessGas,
				Nonce:     4,
				Input:     []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
			},
			After: WorldState{
				{1}: Account{Balance: tosca.NewValue(100), Nonce: 5},
				{2}: Account{Balance: tosca.NewValue(0)},
			},
			Receipt: tosca.Receipt{
				Success: true,
				GasUsed: floria.TxGas + floria.TxDataZeroGasEIP2028*10 + excessGas/10,
			},
		},
		"InputZerosInsufficient": {
			Before: WorldState{
				{1}: Account{Balance: tosca.NewValue(100), Nonce: 4},
				{2}: Account{Balance: tosca.NewValue(0)},
			},
			Transaction: tosca.Transaction{
				Sender:    tosca.Address{1},
				Recipient: &tosca.Address{2},
				GasLimit:  floria.TxGas + floria.TxDataZeroGasEIP2028*10 - 1,
				Nonce:     4,
				Input:     []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
			},
			After: WorldState{
				{1}: Account{Balance: tosca.NewValue(100), Nonce: 4},
				{2}: Account{Balance: tosca.NewValue(0)},
			},
			Receipt: tosca.Receipt{
				Success: false,
				GasUsed: 0,
			},
			OperaError: fmt.Errorf("gas too low"),
		},
		"InputNonZerosExact": {
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
		"InputNonZerosSufficient": {
			Before: WorldState{
				{1}: Account{Balance: tosca.NewValue(100), Nonce: 4},
				{2}: Account{Balance: tosca.NewValue(0)},
			},
			Transaction: tosca.Transaction{
				Sender:    tosca.Address{1},
				Recipient: &tosca.Address{2},
				GasLimit:  floria.TxGas + floria.TxDataNonZeroGasEIP2028*10 + excessGas,
				Nonce:     4,
				Input:     []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10},
			},
			After: WorldState{
				{1}: Account{Balance: tosca.NewValue(100), Nonce: 5},
				{2}: Account{Balance: tosca.NewValue(0)},
			},
			Receipt: tosca.Receipt{
				Success: true,
				GasUsed: floria.TxGas + floria.TxDataNonZeroGasEIP2028*10 + excessGas/10,
			},
		},
		"InputNonZerosInsufficient": {
			Before: WorldState{
				{1}: Account{Balance: tosca.NewValue(100), Nonce: 4},
				{2}: Account{Balance: tosca.NewValue(0)},
			},
			Transaction: tosca.Transaction{
				Sender:    tosca.Address{1},
				Recipient: &tosca.Address{2},
				GasLimit:  floria.TxGas + floria.TxDataNonZeroGasEIP2028*10 - 1,
				Nonce:     4,
				Input:     []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10},
			},
			After: WorldState{
				{1}: Account{Balance: tosca.NewValue(100), Nonce: 4},
				{2}: Account{Balance: tosca.NewValue(0)},
			},
			Receipt: tosca.Receipt{
				Success: false,
				GasUsed: 0,
			},
			OperaError: fmt.Errorf("gas too low"),
		},
		"AccessListOnlyAddressesExact": {
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
		"AccessListOnlyAddressesSufficient": {
			Before: WorldState{
				{1}: Account{Balance: tosca.NewValue(100), Nonce: 4},
				{2}: Account{Balance: tosca.NewValue(0)},
			},
			Transaction: tosca.Transaction{
				Sender:    tosca.Address{1},
				Recipient: &tosca.Address{2},
				GasLimit:  floria.TxGas + floria.TxAccessListAddressGas*2 + excessGas,
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
				GasUsed: floria.TxGas + floria.TxAccessListAddressGas*2 + excessGas/10,
			},
		},
		"AccessListOnlyAddressesInSufficient": {
			Before: WorldState{
				{1}: Account{Balance: tosca.NewValue(100), Nonce: 4},
				{2}: Account{Balance: tosca.NewValue(0)},
			},
			Transaction: tosca.Transaction{
				Sender:    tosca.Address{1},
				Recipient: &tosca.Address{2},
				GasLimit:  floria.TxGas + floria.TxAccessListAddressGas*2 - 1,
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
				{1}: Account{Balance: tosca.NewValue(100), Nonce: 4},
				{2}: Account{Balance: tosca.NewValue(0)},
			},
			Receipt: tosca.Receipt{
				Success: false,
				GasUsed: 0,
			},
			OperaError: fmt.Errorf("gas too low"),
		},
		"AccessListExact": {
			Before: WorldState{
				{1}: Account{Balance: tosca.NewValue(100), Nonce: 4},
				{2}: Account{Balance: tosca.NewValue(0)},
			},
			Transaction: tosca.Transaction{
				Sender:    tosca.Address{1},
				Recipient: &tosca.Address{2},
				GasLimit:  floria.TxGas + floria.TxAccessListAddressGas*2 + floria.TxAccessListStorageKeyGas*5 + excessGas,
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
				GasUsed: floria.TxGas + floria.TxAccessListAddressGas*2 + floria.TxAccessListStorageKeyGas*5 + excessGas/10,
			},
		},
		"AccessListSufficient": {
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
		"AccessListInsufficient": {
			Before: WorldState{
				{1}: Account{Balance: tosca.NewValue(100), Nonce: 4},
				{2}: Account{Balance: tosca.NewValue(0)},
			},
			Transaction: tosca.Transaction{
				Sender:    tosca.Address{1},
				Recipient: &tosca.Address{2},
				GasLimit:  floria.TxGas + floria.TxAccessListAddressGas*2 + floria.TxAccessListStorageKeyGas*5 - 1,
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
				{1}: Account{Balance: tosca.NewValue(100), Nonce: 4},
				{2}: Account{Balance: tosca.NewValue(0)},
			},
			Receipt: tosca.Receipt{
				Success: false,
				GasUsed: 0,
			},
			OperaError: fmt.Errorf("gas too low"),
		},
	}

	return allTestCases
}

func TestProcessor_GasSpecificScenarios(t *testing.T) {
	for name, processor := range getProcessors() {
		if strings.Contains(name, "floria") {
			continue // todo implement gas billing in floria
		}
		t.Run(name, func(t *testing.T) {
			for name, s := range getGasTestScenarios() {
				t.Run(name, func(t *testing.T) {
					s.Run(t, processor)
				})
			}
		})
	}
}
