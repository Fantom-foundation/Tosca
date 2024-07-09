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

func init() {
	tosca.RegisterProcessorFactory("floria", newProcessor)
}

func newProcessor(interpreter tosca.Interpreter) tosca.Processor {
	return &processor{
		interpreter: interpreter,
	}
}

type processor struct {
	interpreter tosca.Interpreter
}

func (p *processor) Run(
	blockParams tosca.BlockParameters,
	transaction tosca.Transaction,
	context tosca.TransactionContext,
) (tosca.Receipt, error) {

	errorReceipt := tosca.Receipt{
		Success: false,
		GasUsed: transaction.GasLimit,
	}

	if err := buyGas(transaction, context); err != nil {
		return errorReceipt, nil
	}

	if err := handleNonce(transaction, context); err != nil {
		return errorReceipt, nil
	}

	if err := transferValue(transaction, context); err != nil {
		return errorReceipt, nil
	}

	return tosca.Receipt{
		Success:         true,
		GasUsed:         transaction.GasLimit,
		ContractAddress: nil,
		Output:          []byte{},
		Logs:            nil,
	}, nil
}

func handleNonce(transaction tosca.Transaction, context tosca.TransactionContext) error {
	stateNonce := context.GetNonce(transaction.Sender)
	messageNonce := transaction.Nonce
	if messageNonce != stateNonce {
		return fmt.Errorf("nonce mismatch: %v != %v", messageNonce, stateNonce)
	}

	// Increment nonce
	context.SetNonce(tosca.Address(transaction.Sender), stateNonce+1)

	return nil
}

func buyGas(transaction tosca.Transaction, context tosca.TransactionContext) error {
	gas := tosca.Mul(transaction.GasPrice, tosca.ValueFromUint64(uint64(transaction.GasLimit)))

	// Buy gas
	senderBalance := context.GetBalance(transaction.Sender)
	if senderBalance.Cmp(gas) < 0 {
		return fmt.Errorf("insufficient balance: %v < %v", senderBalance, gas)
	}

	senderBalance = tosca.Sub(senderBalance, gas)
	context.SetBalance(transaction.Sender, senderBalance)

	return nil
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
