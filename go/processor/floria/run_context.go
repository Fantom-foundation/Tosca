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

type runContext struct {
	tosca.TransactionContext
	interpreter           tosca.Interpreter
	blockParameters       tosca.BlockParameters
	transactionParameters tosca.TransactionParameters
	depth                 int
}

func (r runContext) Call(kind tosca.CallKind, parameters tosca.CallParameters) (tosca.CallResult, error) {

	if r.depth > MaxRecursiveDepth {
		return tosca.CallResult{}, nil
	}
	r.depth++
	defer func() { r.depth-- }()

	snapshot := r.CreateSnapshot()
	if err := transferValue(r, parameters.Value, parameters.Sender, parameters.Recipient); err != nil {
		r.RestoreSnapshot(snapshot)
		return tosca.CallResult{}, nil
	}

	codeHash := r.GetCodeHash(parameters.Recipient)
	code := r.GetCode(parameters.Recipient)

	interpreterParameters := tosca.Parameters{
		BlockParameters:       r.blockParameters,
		TransactionParameters: r.transactionParameters,
		Context:               r,
		Kind:                  kind,
		Static:                kind == tosca.StaticCall,
		Depth:                 r.depth - 1, // depth is already incremented
		Gas:                   parameters.Gas,
		Recipient:             parameters.Recipient,
		Sender:                parameters.Sender,
		Input:                 parameters.Input,
		Value:                 parameters.Value,
		CodeHash:              &codeHash,
		Code:                  code,
	}

	result, err := r.interpreter.Run(interpreterParameters)
	if err != nil || !result.Success {
		r.RestoreSnapshot(snapshot)
	}

	return tosca.CallResult{
		Output:    result.Output,
		GasLeft:   result.GasLeft,
		GasRefund: result.GasRefund,
		Success:   result.Success,
	}, err
}

func transferValue(
	context tosca.TransactionContext,
	value tosca.Value,
	sender tosca.Address,
	recipient tosca.Address,
) error {
	if value == (tosca.Value{}) {
		return nil
	}

	senderBalance := context.GetBalance(sender)
	if senderBalance.Cmp(value) < 0 {
		return fmt.Errorf("insufficient balance: %v < %v", senderBalance, value)
	}

	receiverBalance := context.GetBalance(recipient)
	updatedBalance := tosca.Add(receiverBalance, value)
	if updatedBalance.Cmp(receiverBalance) < 0 || updatedBalance.Cmp(value) < 0 {
		return fmt.Errorf("overflow: %v + %v", receiverBalance, value)
	}

	senderBalance = tosca.Sub(senderBalance, value)
	context.SetBalance(sender, senderBalance)
	context.SetBalance(recipient, updatedBalance)

	return nil
}
