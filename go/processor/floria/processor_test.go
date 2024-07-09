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

func TestProcessor_TransferValue(t *testing.T) {
	transaction := tosca.Transaction{
		Sender:    tosca.Address{1},
		Recipient: &tosca.Address{2},
		Value:     tosca.NewValue(10),
	}

	ctrl := gomock.NewController(t)
	context := tosca.NewMockTransactionContext(ctrl)

	context.EXPECT().GetBalance(transaction.Sender).Return(tosca.NewValue(100))
	context.EXPECT().GetBalance(*transaction.Recipient).Return(tosca.NewValue(0))
	context.EXPECT().SetBalance(transaction.Sender, tosca.NewValue(90))
	context.EXPECT().GetBalance(transaction.Sender).Return(tosca.NewValue(90))
	context.EXPECT().SetBalance(*transaction.Recipient, tosca.NewValue(10))
	context.EXPECT().GetBalance(*transaction.Recipient).Return(tosca.NewValue(10))

	err := transferValue(transaction, context)
	if err != nil {
		t.Errorf("transferValue returned an error: %v", err)
	}
	if context.GetBalance(transaction.Sender).Cmp(tosca.NewValue(90)) != 0 {
		t.Errorf("Sender balance was not decremented")
	}
	if context.GetBalance(*transaction.Recipient).Cmp(tosca.NewValue(10)) != 0 {
		t.Errorf("Recipient balance was not incremented")
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
