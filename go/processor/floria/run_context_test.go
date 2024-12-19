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
	"math"
	"testing"

	"github.com/Fantom-foundation/Tosca/go/tosca"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"go.uber.org/mock/gomock"
)

func TestCalls_InterpreterResultIsHandledCorrectly(t *testing.T) {
	tests := map[string]struct {
		setup   func(interpreter *tosca.MockInterpreter)
		success bool
		output  []byte
	}{
		"successful": {
			setup: func(interpreter *tosca.MockInterpreter) {
				interpreter.EXPECT().Run(gomock.Any()).Return(tosca.Result{Success: true}, nil)
			},
			success: true,
		},
		"failed": {
			setup: func(interpreter *tosca.MockInterpreter) {
				interpreter.EXPECT().Run(gomock.Any()).Return(tosca.Result{Success: false}, nil)
			},
			success: false,
		},
		"output": {
			setup: func(interpreter *tosca.MockInterpreter) {
				interpreter.EXPECT().Run(gomock.Any()).Return(tosca.Result{Success: true, Output: []byte("some output")}, nil)
			},
			success: true,
			output:  []byte("some output"),
		},
	}

	ctrl := gomock.NewController(t)
	context := tosca.NewMockTransactionContext(ctrl)
	interpreter := tosca.NewMockInterpreter(ctrl)

	runContext := runContext{
		context,
		interpreter,
		tosca.BlockParameters{},
		tosca.TransactionParameters{},
		0,
		false,
	}

	params := tosca.CallParameters{
		Sender:    tosca.Address{1},
		Recipient: tosca.Address{2},
		Value:     tosca.NewValue(0),
		Gas:       1000,
		Input:     []byte{},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			context.EXPECT().GetCodeHash(params.Recipient).Return(tosca.Hash{})
			context.EXPECT().GetCode(params.Recipient).Return([]byte{})
			context.EXPECT().CreateSnapshot()
			context.EXPECT().RestoreSnapshot(gomock.Any()).AnyTimes()

			test.setup(interpreter)

			result, err := runContext.Call(tosca.Call, params)
			if err != nil {
				t.Errorf("Call returned an unexpected error: %v", err)
			}
			if result.Success != test.success {
				t.Errorf("Unexpected success value from interpreter call")
			}
			if string(result.Output) != string(test.output) {
				t.Errorf("Unexpected output value from interpreter call")
			}
		})
	}
}

func TestCall_TransferValueInCall(t *testing.T) {
	ctrl := gomock.NewController(t)
	context := tosca.NewMockTransactionContext(ctrl)
	interpreter := tosca.NewMockInterpreter(ctrl)
	runContext := runContext{
		context,
		interpreter,
		tosca.BlockParameters{},
		tosca.TransactionParameters{},
		0,
		false,
	}

	params := tosca.CallParameters{
		Sender:    tosca.Address{1},
		Recipient: tosca.Address{2},
		Value:     tosca.NewValue(10),
		Gas:       1000,
		Input:     []byte{},
	}

	context.EXPECT().GetCodeHash(params.Recipient).Return(tosca.Hash{})
	context.EXPECT().GetCode(params.Recipient).Return([]byte{})
	context.EXPECT().CreateSnapshot()

	context.EXPECT().GetBalance(params.Sender).Return(tosca.NewValue(100)).Times(2)
	context.EXPECT().GetBalance(params.Recipient).Return(tosca.NewValue(0)).Times(2)
	context.EXPECT().SetBalance(params.Sender, tosca.NewValue(90))
	context.EXPECT().SetBalance(params.Recipient, tosca.NewValue(10))

	interpreter.EXPECT().Run(gomock.Any()).Return(tosca.Result{Success: true}, nil)

	_, err := runContext.Call(tosca.Call, params)
	if err != nil {
		t.Errorf("transferValue returned an error: %v", err)
	}
}

func TestCall_TransferValueInCreate(t *testing.T) {
	ctrl := gomock.NewController(t)
	context := tosca.NewMockTransactionContext(ctrl)
	interpreter := tosca.NewMockInterpreter(ctrl)
	runContext := runContext{
		context,
		interpreter,
		tosca.BlockParameters{},
		tosca.TransactionParameters{},
		0,
		false,
	}

	params := tosca.CallParameters{
		Sender: tosca.Address{1},
		Value:  tosca.NewValue(10),
		Gas:    1000,
		Input:  []byte{},
	}
	code := tosca.Code{}
	createdAddress := tosca.Address(crypto.CreateAddress(common.Address(params.Sender), 0))

	context.EXPECT().GetBalance(params.Sender).Return(tosca.NewValue(100))
	context.EXPECT().GetBalance(params.Recipient).Return(tosca.NewValue(0))
	context.EXPECT().GetNonce(params.Sender).Return(uint64(0))
	context.EXPECT().SetNonce(params.Sender, uint64(1))
	context.EXPECT().GetNonce((params.Sender)).Return(uint64(1))
	context.EXPECT().GetNonce(createdAddress).Return(uint64(0))
	context.EXPECT().GetCodeHash(createdAddress).Return(tosca.Hash{})
	context.EXPECT().CreateSnapshot()
	context.EXPECT().SetNonce(createdAddress, uint64(1))
	context.EXPECT().GetBalance(params.Sender).Return(tosca.NewValue(100))
	context.EXPECT().GetBalance(createdAddress).Return(tosca.NewValue(0))
	context.EXPECT().SetBalance(params.Sender, tosca.NewValue(90))
	context.EXPECT().SetBalance(createdAddress, tosca.NewValue(10))
	context.EXPECT().SetCode(createdAddress, code)

	interpreter.EXPECT().Run(gomock.Any()).Return(tosca.Result{Success: true, Output: tosca.Data(code)}, nil)

	result, err := runContext.Call(tosca.Create, params)
	if err != nil {
		t.Errorf("transferValue returned an error: %v", err)
	}
	if !result.Success {
		t.Errorf("transferValue was not successful")
	}
}

func TestTransferValue_InCallRestoreFailed(t *testing.T) {
	ctrl := gomock.NewController(t)
	context := tosca.NewMockTransactionContext(ctrl)
	interpreter := tosca.NewMockInterpreter(ctrl)
	runContext := runContext{
		context,
		interpreter,
		tosca.BlockParameters{},
		tosca.TransactionParameters{},
		0,
		false,
	}

	params := tosca.CallParameters{
		Sender:    tosca.Address{1},
		Recipient: tosca.Address{2},
		Value:     tosca.NewValue(10),
		Gas:       1000,
		Input:     []byte{},
	}
	context.EXPECT().GetBalance(params.Sender).Return(tosca.NewValue(0))

	result, err := runContext.Call(tosca.Call, params)
	if err != nil {
		t.Errorf("Correct execution of the transaction should not return an error")
	}

	if result.Success {
		t.Errorf("The transaction should have failed")
	}
}

func TestTransferValue_SuccessfulValueTransfer(t *testing.T) {
	values := map[string]tosca.Value{
		"zeroValue":     tosca.NewValue(0),
		"smallValue":    tosca.NewValue(10),
		"senderBalance": tosca.NewValue(100),
	}

	senderBalance := tosca.NewValue(100)
	recipientBalance := tosca.NewValue(0)

	ctrl := gomock.NewController(t)
	context := tosca.NewMockTransactionContext(ctrl)

	for name, value := range values {
		t.Run(name, func(t *testing.T) {
			transaction := tosca.Transaction{
				Sender:    tosca.Address{1},
				Recipient: &tosca.Address{2},
				Value:     value,
			}

			if name != "zeroValue" {
				context.EXPECT().GetBalance(transaction.Sender).Return(senderBalance)
				context.EXPECT().GetBalance(*transaction.Recipient).Return(recipientBalance)
			}

			if !canTransferValue(context, transaction.Value, transaction.Sender, transaction.Recipient) {
				t.Errorf("Value should be possible but was not")
			}
		})
	}
}

func TestTransferValue_FailedValueTransfer(t *testing.T) {
	transfers := map[string]struct {
		value           tosca.Value
		senderBalance   tosca.Value
		receiverBalance tosca.Value
	}{
		"insufficientBalance": {
			tosca.NewValue(100),
			tosca.NewValue(50),
			tosca.NewValue(0),
		},
		"overflow": {
			tosca.NewValue(100),
			tosca.NewValue(1000),
			tosca.NewValue(math.MaxUint64, math.MaxUint64, math.MaxUint64, math.MaxUint64-10),
		},
	}

	for name, transfer := range transfers {
		t.Run(name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			context := tosca.NewMockTransactionContext(ctrl)

			context.EXPECT().GetBalance(tosca.Address{1}).Return(transfer.senderBalance).AnyTimes()
			context.EXPECT().GetBalance(tosca.Address{2}).Return(transfer.receiverBalance).AnyTimes()

			if canTransferValue(context, transfer.value, tosca.Address{1}, &tosca.Address{2}) {
				t.Errorf("value transfer should have returned an error")
			}
		})
	}
}

func TestCanTransferValue_SameSenderAndReceiver(t *testing.T) {
	tests := map[string]struct {
		value         tosca.Value
		expectedError bool
	}{
		"sufficientBalance":   {tosca.NewValue(10), false},
		"insufficientBalance": {tosca.NewValue(1000), true},
	}

	for _, test := range tests {
		ctrl := gomock.NewController(t)
		context := tosca.NewMockTransactionContext(ctrl)
		context.EXPECT().GetBalance(gomock.Any()).Return(tosca.NewValue(100))

		canTransfer := canTransferValue(context, test.value, tosca.Address{1}, &tosca.Address{1})
		if test.expectedError {
			if canTransfer {
				t.Errorf("transfer value should have not been possible")
			}
		} else {
			if !canTransfer {
				t.Errorf("transfer value should have been possible")
			}
		}
	}
}

func TestTransferValue_BalanceIsNotChangedWhenValueIsTransferredToTheSameAccount(t *testing.T) {
	ctrl := gomock.NewController(t)
	context := tosca.NewMockTransactionContext(ctrl)

	address := tosca.Address{1}
	value := tosca.NewValue(10)

	transferValue(context, value, address, address)
}

func TestCreateAddress(t *testing.T) {
	tests := map[string]struct {
		kind     tosca.CallKind
		sender   tosca.Address
		nonce    uint64
		salt     tosca.Hash
		initHash tosca.Hash
	}{
		"create": {
			kind:     tosca.Create,
			sender:   tosca.Address{1},
			nonce:    42,
			salt:     tosca.Hash{},
			initHash: tosca.Hash{},
		},
		"create2": {
			kind:     tosca.Create2,
			sender:   tosca.Address{1},
			nonce:    0,
			salt:     tosca.Hash{16, 32, 64},
			initHash: tosca.Hash{0x01, 0x02, 0x03, 0x04, 0x05},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			var want tosca.Address
			if test.kind == tosca.Create {
				want = tosca.Address(crypto.CreateAddress(common.Address(test.sender), test.nonce))
			} else {
				want = tosca.Address(crypto.CreateAddress2(common.Address(test.sender), common.Hash(test.salt), test.initHash[:]))
			}
			result := createAddress(test.kind, test.sender, test.nonce, test.salt, test.initHash)
			if result != want {
				t.Errorf("Unexpected address, got: %v, want: %v", result, want)
			}
		})
	}
}

func TestIncrementNonce(t *testing.T) {
	tests := map[string]struct {
		nonce uint64
		err   error
	}{
		"zero": {
			nonce: 0,
			err:   nil,
		},
		"max": {
			nonce: math.MaxUint64,
			err:   fmt.Errorf("nonce overflow"),
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			context := tosca.NewMockTransactionContext(ctrl)
			context.EXPECT().GetNonce(gomock.Any()).Return(test.nonce)
			context.EXPECT().SetNonce(gomock.Any(), test.nonce+1).AnyTimes()

			err := incrementNonce(context, tosca.Address{})
			if test.err != nil && err == nil {
				t.Errorf("incrementNonce returned an unexpected error: %v", err)
			}
		})
	}
}
