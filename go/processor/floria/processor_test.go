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
	"math"
	"reflect"
	"testing"

	"github.com/Fantom-foundation/Tosca/go/tosca"
	"go.uber.org/mock/gomock"
)

func TestProcessor_NewProcessorReturnsProcessor(t *testing.T) {
	interpreter := tosca.NewMockInterpreter(gomock.NewController(t))
	processor := newProcessor(interpreter)
	if processor == nil {
		t.Errorf("newProcessor returned nil")
	}
}

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

	transaction := tosca.Transaction{
		Sender: tosca.Address{1},
		Nonce:  9,
	}

	err := nonceCheck(transaction.Nonce, context.GetNonce(transaction.Sender))
	if err != nil {
		t.Errorf("nonceCheck returned an error: %v", err)
	}
}

func TestProcessor_NonceOverflowIsDetected(t *testing.T) {
	err := nonceCheck(math.MaxUint64, math.MaxUint64)
	if err == nil {
		t.Errorf("nonceCheck did not spot nonce overflow")
	}
}

func TestProcessor_NonceMissMatch(t *testing.T) {
	err := nonceCheck(uint64(10), uint64(42))
	if err == nil {
		t.Errorf("nonceCheck did not spot nonce miss match")
	}
}

func TestProcessor_EoaCheck(t *testing.T) {
	tests := map[string]struct {
		codeHash tosca.Hash
		isEOA    bool
	}{
		"empty": {
			tosca.Hash{},
			false,
		},
		"emptyHash": {
			emptyCodeHash,
			false,
		},
		"nonEmpty": {
			tosca.Hash{1, 2, 3},
			true,
		},
	}

	for testName, test := range tests {
		t.Run(testName, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			context := tosca.NewMockTransactionContext(ctrl)

			context.EXPECT().GetCodeHash(tosca.Address{1}).Return(test.codeHash)

			err := eoaCheck(tosca.Address{1}, context)
			if test.isEOA && err == nil {
				t.Errorf("eoaCheck returned wrong result: %v", err)
			}
		})
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
	tests := map[string]struct {
		transaction     tosca.Transaction
		result          tosca.CallResult
		revision        tosca.Revision
		expectedGasLeft tosca.Gas
	}{
		"InternalTransaction": {
			transaction: tosca.Transaction{
				Sender:   tosca.Address{},
				GasLimit: 1000,
			},
			result: tosca.CallResult{
				GasLeft:   500,
				Success:   true,
				GasRefund: 0,
			},
			revision:        tosca.R10_London,
			expectedGasLeft: 500,
		},
		"NonInternalTransaction": {
			transaction: tosca.Transaction{
				Sender:   tosca.Address{1},
				GasLimit: 1000,
			},
			result: tosca.CallResult{
				GasLeft:   500,
				Success:   true,
				GasRefund: 0,
			},
			revision:        tosca.R10_London,
			expectedGasLeft: 450,
		},
		"RefundPreLondon": {
			transaction: tosca.Transaction{
				Sender:   tosca.Address{},
				GasLimit: 1000,
			},
			result: tosca.CallResult{
				GasLeft:   500,
				Success:   true,
				GasRefund: 300,
			},
			revision:        tosca.R09_Berlin,
			expectedGasLeft: 750,
		},
		"RefundLondon": {
			transaction: tosca.Transaction{
				Sender:   tosca.Address{},
				GasLimit: 1000,
			},
			result: tosca.CallResult{
				GasLeft:   500,
				Success:   true,
				GasRefund: 300,
			},
			revision:        tosca.R10_London,
			expectedGasLeft: 600,
		},
		"RefundPostLondon": {
			transaction: tosca.Transaction{
				Sender:   tosca.Address{},
				GasLimit: 1000,
			},
			result: tosca.CallResult{
				GasLeft:   500,
				Success:   true,
				GasRefund: 300,
			},
			revision:        tosca.R13_Cancun,
			expectedGasLeft: 600,
		},
		"smallRefund": {
			transaction: tosca.Transaction{
				Sender:   tosca.Address{},
				GasLimit: 1000,
			},
			result: tosca.CallResult{
				GasLeft:   500,
				Success:   true,
				GasRefund: 5,
			},
			revision:        tosca.R10_London,
			expectedGasLeft: 505,
		},
		"UnsuccessfulResult": {
			transaction: tosca.Transaction{
				Sender:   tosca.Address{},
				GasLimit: 1000,
			},
			result: tosca.CallResult{
				GasLeft:   0,
				Success:   false,
				GasRefund: 500,
			},
			revision:        tosca.R10_London,
			expectedGasLeft: 0,
		},
	}

	for testName, test := range tests {
		t.Run(testName, func(t *testing.T) {
			actualGasLeft := calculateGasLeft(test.transaction, test.result, test.revision)

			if actualGasLeft != test.expectedGasLeft {
				t.Errorf("gasUsed returned incorrect result, got: %d, want: %d", actualGasLeft, test.expectedGasLeft)
			}
		})
	}
}

func TestProcessor_RefundGas(t *testing.T) {
	gasPrice := 5
	gasLeft := 50
	senderBalance := 1000

	sender := tosca.Address{1}

	ctrl := gomock.NewController(t)
	context := tosca.NewMockTransactionContext(ctrl)

	context.EXPECT().GetBalance(sender).Return(tosca.NewValue(uint64(senderBalance)))
	context.EXPECT().SetBalance(sender, tosca.NewValue(uint64(senderBalance+gasLeft*gasPrice)))

	transaction := tosca.Transaction{
		Sender:   sender,
		GasPrice: tosca.NewValue(uint64(gasPrice)),
	}

	refundGas(transaction, context, tosca.Gas(gasLeft))

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

			actualGasUsed := calculateSetupGas(transaction)
			if actualGasUsed != test.expectedGasUsed {
				t.Errorf("setupGasBilling returned incorrect gas used, got: %d, want: %d", actualGasUsed, test.expectedGasUsed)
			}
		})
	}
}

func TestProcessor_CallKind(t *testing.T) {
	tests := map[string]struct {
		recipient *tosca.Address
		kind      tosca.CallKind
	}{
		"call": {
			recipient: &tosca.Address{2},
			kind:      tosca.Call,
		},
		"create": {
			recipient: nil,
			kind:      tosca.Create,
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			transaction := tosca.Transaction{
				Sender:    tosca.Address{1},
				Recipient: test.recipient,
			}
			if callKind(transaction) != test.kind {
				t.Errorf("callKind returned incorrect result: %v", callKind(transaction))
			}
		})
	}
}

func TestProcessor_CallParameters(t *testing.T) {
	transaction := tosca.Transaction{
		Sender: tosca.Address{1},
		Input:  []byte{1, 2, 3},
		Value:  tosca.NewValue(100),
	}
	gas := tosca.Gas(1000)

	want := tosca.CallParameters{
		Sender: transaction.Sender,
		Input:  transaction.Input,
		Value:  transaction.Value,
		Gas:    gas,
	}

	got := callParameters(transaction, gas)
	if !reflect.DeepEqual(got, want) {
		t.Errorf("callParameters returned incorrect result: %v", got)

	}

	transaction.Recipient = &tosca.Address{2}
	want.Recipient = *transaction.Recipient

	got = callParameters(transaction, gas)
	if !reflect.DeepEqual(got, want) {
		t.Errorf("callParameters returned incorrect result: %v", got)

	}
}

func TestProcessor_SetUpAccessList(t *testing.T) {
	ctrl := gomock.NewController(t)
	context := tosca.NewMockTransactionContext(ctrl)

	sender := tosca.Address{1}
	recipient := tosca.Address{2}
	accessListAddress := tosca.Address{3}

	transaction := tosca.Transaction{
		Sender:    sender,
		Recipient: &recipient,
		AccessList: []tosca.AccessTuple{
			{
				Address: accessListAddress,
				Keys:    []tosca.Key{{1}, {2}},
			},
		},
	}

	for _, contract := range getPrecompiledAddresses(tosca.R09_Berlin) {
		context.EXPECT().AccessAccount(contract)
	}
	context.EXPECT().AccessAccount(sender)
	context.EXPECT().AccessAccount(recipient)
	context.EXPECT().AccessAccount(accessListAddress)
	context.EXPECT().AccessStorage(accessListAddress, tosca.Key{1})
	context.EXPECT().AccessStorage(accessListAddress, tosca.Key{2})

	setUpAccessList(transaction, context, tosca.R09_Berlin)
}

func TestProcessor_AccessListIsNotCreatedIfTransactionHasNone(t *testing.T) {
	ctrl := gomock.NewController(t)
	context := tosca.NewMockTransactionContext(ctrl)
	// No calls to context

	sender := tosca.Address{1}
	recipient := tosca.Address{2}

	transaction := tosca.Transaction{
		Sender:    sender,
		Recipient: &recipient,
	}

	setUpAccessList(transaction, context, tosca.R09_Berlin)
}
