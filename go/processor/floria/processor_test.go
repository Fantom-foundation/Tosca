// Copyright (c) 2024 Fantom Foundation
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at fantom.foundation/bsl11.
//
// Change Date: 2028-4-16
//
// On the date above, in accordance with the Business Source License, use of
// this software will be governed by the GNU Lesser General Public License v3.

package floria

import (
	"fmt"
	"testing"

	"github.com/Fantom-foundation/Tosca/go/tosca"
	"go.uber.org/mock/gomock"
)

func TestProcessorRegistry_InitProcessor(t *testing.T) {
	processorFactories := tosca.GetAllRegisteredProcessorFactories()
	if len(processorFactories) == 0 {
		t.Errorf("No processor factories found")
	}

	processor := tosca.GetProcessorFactory("floria")
	if processor == nil {
		t.Errorf("Floria processor factory not found")
	}
}

func TestProcessor_HandleNonce(t *testing.T) {
	ctrl := gomock.NewController(t)
	context := tosca.NewMockTransactionContext(ctrl)

	context.EXPECT().GetNonce(tosca.Address{1}).Return(uint64(9))
	context.EXPECT().SetNonce(tosca.Address{1}, uint64(10))
	context.EXPECT().GetNonce(tosca.Address{1}).Return(uint64(10))

	transaction := tosca.Transaction{
		Sender: tosca.Address{1},
		Nonce:  9,
	}

	err := handleNonce(transaction, context)
	if err != nil {
		t.Errorf("handleNonce returned an error: %v", err)
	}
	if context.GetNonce(transaction.Sender) != 10 {
		t.Errorf("Nonce was not incremented")
	}
}

func TestProcessor_NonceMissmatch(t *testing.T) {
	ctrl := gomock.NewController(t)
	context := tosca.NewMockTransactionContext(ctrl)

	context.EXPECT().GetNonce(tosca.Address{1}).Return(uint64(5))

	transaction := tosca.Transaction{
		Sender: tosca.Address{1},
		Nonce:  10,
	}
	err := handleNonce(transaction, context)
	if err == nil {
		t.Errorf("handleNonce did not spot nonce miss match")
	}
}

func TestProcessor_BuyGas(t *testing.T) {
	balance := uint64(1000)
	gasLimit := uint64(100)
	gasPrice := uint64(2)

	transaction := tosca.Transaction{
		Sender:   tosca.Address{1},
		GasLimit: tosca.Gas(gasLimit),
		GasPrice: tosca.NewValue(gasPrice),
	}

	ctrl := gomock.NewController(t)
	context := tosca.NewMockTransactionContext(ctrl)
	context.EXPECT().GetBalance(transaction.Sender).Return(tosca.NewValue(balance))
	context.EXPECT().SetBalance(transaction.Sender, tosca.NewValue(balance-gasLimit*gasPrice))
	context.EXPECT().GetBalance(transaction.Sender).Return(tosca.NewValue(balance - gasLimit*gasPrice))

	err := buyGas(transaction, context)
	if err != nil {
		t.Errorf("buyGas returned an error: %v", err)
	}
	if context.GetBalance(transaction.Sender).Cmp(tosca.NewValue(balance-gasLimit*gasPrice)) != 0 {
		t.Errorf("Sender balance was not decremented correctly")
	}
}

func TestProcessor_BuyGasInsufficientBalance(t *testing.T) {
	balance := uint64(100)
	gasLimit := uint64(100)
	gasPrice := uint64(2)

	transaction := tosca.Transaction{
		Sender:   tosca.Address{1},
		GasLimit: tosca.Gas(gasLimit),
		GasPrice: tosca.NewValue(gasPrice),
	}

	ctrl := gomock.NewController(t)
	context := tosca.NewMockTransactionContext(ctrl)
	context.EXPECT().GetBalance(transaction.Sender).Return(tosca.NewValue(balance))

	err := buyGas(transaction, context)
	if err == nil {
		t.Errorf("buyGas did not fail with insufficient balance")
	}
}

func TestGasUsed(t *testing.T) {
	tests := []struct {
		sender          tosca.Address
		expectedGasUsed tosca.Gas
	}{
		{
			sender:          tosca.Address{},
			expectedGasUsed: 500,
		},
		{
			sender:          tosca.Address{1},
			expectedGasUsed: 550,
		},
		{
			sender:          tosca.Address{42},
			expectedGasUsed: 550,
		},
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("sender%v", test.sender), func(t *testing.T) {
			transaction := tosca.Transaction{
				Sender:   test.sender,
				GasLimit: 1000,
			}

			gasLeft := tosca.Gas(500)
			actualGasUsed := gasUsed(transaction, gasLeft)

			if actualGasUsed != test.expectedGasUsed {
				t.Errorf("gasUsed returned incorrect result, got: %d, want: %d", actualGasUsed, test.expectedGasUsed)
			}
		})
	}
}

func TestProcessor_SetupGasBilling(t *testing.T) {
	tests := map[string]struct {
		recipient       *tosca.Address
		input           []byte
		accessList      []tosca.AccessTuple
		expectedGasUsed tosca.Gas
	}{
		"creation": {
			recipient:       nil,
			input:           []byte{},
			accessList:      nil,
			expectedGasUsed: TxGasContractCreation,
		},
		"call": {
			recipient:       &tosca.Address{1},
			input:           []byte{},
			accessList:      nil,
			expectedGasUsed: TxGas,
		},
		"inputZeros": {
			recipient:       &tosca.Address{1},
			input:           []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
			accessList:      nil,
			expectedGasUsed: TxGas + 10*TxDataZeroGasEIP2028,
		},
		"inputNonZeros": {
			recipient:       &tosca.Address{1},
			input:           []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10},
			accessList:      nil,
			expectedGasUsed: TxGas + 10*TxDataNonZeroGasEIP2028,
		},
		"accessList": {
			recipient: &tosca.Address{1},
			input:     []byte{},
			accessList: []tosca.AccessTuple{
				{
					Address: tosca.Address{1},
					Keys:    []tosca.Key{{1}, {2}, {3}},
				},
			},
			expectedGasUsed: TxGas + TxAccessListAddressGas + 3*TxAccessListStorageKeyGas,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			transaction := tosca.Transaction{
				Recipient:  test.recipient,
				Input:      test.input,
				AccessList: test.accessList,
			}

			actualGasUsed := setupGasBilling(transaction)
			if actualGasUsed != test.expectedGasUsed {
				t.Errorf("setupGasBilling returned incorrect gas used, got: %d, want: %d", actualGasUsed, test.expectedGasUsed)
			}
		})
	}
}
