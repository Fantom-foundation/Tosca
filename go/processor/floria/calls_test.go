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

	transaction := tosca.Transaction{
		Sender:    tosca.Address{1},
		Recipient: &tosca.Address{2},
	}

	ctrl := gomock.NewController(t)
	context := tosca.NewMockTransactionContext(ctrl)
	interpreter := tosca.NewMockInterpreter(ctrl)

	senderBalance := tosca.NewValue(0)
	recipientBalance := tosca.NewValue(0)

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			context.EXPECT().GetCodeHash(*transaction.Recipient).Return(tosca.Hash{})
			context.EXPECT().GetCode(*transaction.Recipient).Return([]byte{})
			context.EXPECT().CreateSnapshot()
			context.EXPECT().GetBalance(transaction.Sender).Return(senderBalance)
			context.EXPECT().SetBalance(transaction.Sender, senderBalance)
			context.EXPECT().GetBalance(*transaction.Recipient).Return(recipientBalance)
			context.EXPECT().SetBalance(*transaction.Recipient, recipientBalance)
			context.EXPECT().RestoreSnapshot(gomock.Any()).AnyTimes()

			test.setup(interpreter)

			result, err := call(interpreter, transaction, context)
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

func TestTransferValue_SuccessfulValueTransfer(t *testing.T) {
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
	context.EXPECT().SetBalance(*transaction.Recipient, tosca.NewValue(10))

	err := transferValue(transaction, context)
	if err != nil {
		t.Errorf("transferValue returned an error: %v", err)
	}
}

func TestCall_TransferValueInCall(t *testing.T) {
	transaction := tosca.Transaction{
		Sender:    tosca.Address{1},
		Recipient: &tosca.Address{2},
		Value:     tosca.NewValue(10),
	}

	ctrl := gomock.NewController(t)
	context := tosca.NewMockTransactionContext(ctrl)
	interpreter := tosca.NewMockInterpreter(ctrl)

	context.EXPECT().GetCodeHash(*transaction.Recipient).Return(tosca.Hash{})
	context.EXPECT().GetCode(*transaction.Recipient).Return([]byte{})
	context.EXPECT().CreateSnapshot()

	context.EXPECT().GetBalance(transaction.Sender).Return(tosca.NewValue(100))
	context.EXPECT().GetBalance(*transaction.Recipient).Return(tosca.NewValue(0))
	context.EXPECT().SetBalance(transaction.Sender, tosca.NewValue(90))
	context.EXPECT().SetBalance(*transaction.Recipient, tosca.NewValue(10))

	interpreter.EXPECT().Run(gomock.Any()).Return(tosca.Result{Success: true}, nil)

	_, err := call(interpreter, transaction, context)
	if err != nil {
		t.Errorf("transferValue returned an error: %v", err)
	}
}

func TestProcessor_TransferValueInCallRestoreFailed(t *testing.T) {
	transaction := tosca.Transaction{
		Sender:    tosca.Address{1},
		Recipient: &tosca.Address{2},
		Value:     tosca.NewValue(10),
	}

	ctrl := gomock.NewController(t)
	context := tosca.NewMockTransactionContext(ctrl)
	interpreter := tosca.NewMockInterpreter(ctrl)

	context.EXPECT().GetCodeHash(*transaction.Recipient).Return(tosca.Hash{})
	context.EXPECT().GetCode(*transaction.Recipient).Return([]byte{})
	context.EXPECT().CreateSnapshot()
	context.EXPECT().GetBalance(transaction.Sender).Return(tosca.NewValue(0))
	context.EXPECT().RestoreSnapshot(gomock.Any())

	_, err := call(interpreter, transaction, context)
	if err == nil {
		t.Errorf("Failed transferValue returned no error")
	}
}
