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

	"github.com/Fantom-foundation/Tosca/go/tosca"
)

func call(interpreter tosca.Interpreter, transaction tosca.Transaction, context tosca.TransactionContext, gas tosca.Gas) (tosca.Result, error) {

	blockParameters := tosca.BlockParameters{}

	transactionParameters := tosca.TransactionParameters{
		Origin:     transaction.Sender,
		GasPrice:   transaction.GasPrice,
		BlobHashes: []tosca.Hash{},
	}

	codeHash := context.GetCodeHash(*transaction.Recipient)
	code := context.GetCode(*transaction.Recipient)

	params := tosca.Parameters{
		BlockParameters:       blockParameters,
		TransactionParameters: transactionParameters,
		//Context:               runContext, todo implement
		Kind:      tosca.Call,
		Static:    false,
		Depth:     0, // todo add depth check
		Gas:       gas,
		Recipient: *transaction.Recipient,
		Sender:    transaction.Sender,
		Input:     transaction.Input,
		Value:     transaction.Value,
		CodeHash:  &codeHash,
		Code:      code,
	}

	snapshot := context.CreateSnapshot()
	if err := transferValue(transaction, context); err != nil {
		context.RestoreSnapshot(snapshot)
		return tosca.Result{}, nil
	}

	result, err := interpreter.Run(params)
	if err != nil || !result.Success {
		context.RestoreSnapshot(snapshot)
	}

	return result, err
}

func transferValue(transaction tosca.Transaction, context tosca.TransactionContext) error {
	senderBalance := context.GetBalance(transaction.Sender)
	transferValue := transaction.Value
	if senderBalance.Cmp(transferValue) < 0 {
		return fmt.Errorf("insufficient balance: %v < %v", senderBalance, transferValue)
	}

	senderBalance = tosca.Sub(senderBalance, transferValue)
	context.SetBalance(transaction.Sender, senderBalance)

	receiverBalance := context.GetBalance(*transaction.Recipient)
	receiverBalance = tosca.Add(receiverBalance, transferValue)
	context.SetBalance(*transaction.Recipient, receiverBalance)

	return nil
}
