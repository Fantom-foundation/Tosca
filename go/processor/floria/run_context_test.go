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
	"testing"

	"github.com/Fantom-foundation/Tosca/go/tosca"
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

	context.EXPECT().GetBalance(params.Sender).Return(tosca.NewValue(100))
	context.EXPECT().GetBalance(params.Recipient).Return(tosca.NewValue(0))
	context.EXPECT().SetBalance(params.Sender, tosca.NewValue(90))
	context.EXPECT().SetBalance(params.Recipient, tosca.NewValue(10))

	interpreter.EXPECT().Run(gomock.Any()).Return(tosca.Result{Success: true}, nil)

	_, err := runContext.Call(tosca.Call, params)
	if err != nil {
		t.Errorf("transferValue returned an error: %v", err)
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
	}

	params := tosca.CallParameters{
		Sender:    tosca.Address{1},
		Recipient: tosca.Address{2},
		Value:     tosca.NewValue(10),
		Gas:       1000,
		Input:     []byte{},
	}

	context.EXPECT().CreateSnapshot()
	context.EXPECT().GetBalance(params.Sender).Return(tosca.NewValue(0))
	context.EXPECT().RestoreSnapshot(gomock.Any())

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
				context.EXPECT().SetBalance(transaction.Sender, tosca.Sub(senderBalance, value))
				context.EXPECT().SetBalance(*transaction.Recipient, tosca.Add(recipientBalance, value))
			}

			err := transferValue(context, transaction.Value, transaction.Sender, *transaction.Recipient)
			if err != nil {
				t.Errorf("transferValue returned an error: %v", err)
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

			err := transferValue(context, transfer.value, tosca.Address{1}, tosca.Address{2})
			if err == nil {
				t.Errorf("transferValue should have returned an error")
			}
		})
	}

}
