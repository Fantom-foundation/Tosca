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

	// geth dependencies
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

type runContext struct {
	tosca.TransactionContext
	interpreter           tosca.Interpreter
	blockParameters       tosca.BlockParameters
	transactionParameters tosca.TransactionParameters
	depth                 int
	static                bool
}

func (r runContext) Call(kind tosca.CallKind, parameters tosca.CallParameters) (tosca.CallResult, error) {
	if r.depth > MaxRecursiveDepth {
		return tosca.CallResult{}, nil
	}
	r.depth++
	defer func() { r.depth-- }()

	codeHash := r.GetCodeHash(parameters.Recipient)
	code := r.GetCode(parameters.Recipient)

	if kind == tosca.DelegateCall || kind == tosca.CallCode {
		code = r.GetCode(parameters.CodeAddress)
		codeHash = r.GetCodeHash(parameters.CodeAddress)
	}

	recipient := parameters.Recipient
	var createdAddress tosca.Address
	if kind == tosca.Create || kind == tosca.Create2 {
		if parameters.Recipient == (tosca.Address{}) {
			code = tosca.Code(parameters.Input)
			codeHash = hashCode(code)
		}
		createdAddress = createAddress(
			kind,
			parameters.Sender,
			r.GetNonce(parameters.Sender),
			parameters.Salt,
			codeHash,
		)

		r.SetNonce(parameters.Sender, r.GetNonce(parameters.Sender)+1)
		r.SetNonce(createdAddress, 1)
		recipient = createdAddress
	}

	if kind == tosca.StaticCall {
		r.static = true
	}

	snapshot := r.CreateSnapshot()
	if err := transferValue(r, parameters.Value, parameters.Sender, recipient); err != nil {
		r.RestoreSnapshot(snapshot)
		return tosca.CallResult{}, nil
	}

	output, isPrecompiled := handlePrecompiled(r.blockParameters.Revision, parameters.Input, recipient, parameters.Gas)
	if isPrecompiled {
		return output, nil
	}

	interpreterParameters := tosca.Parameters{
		BlockParameters:       r.blockParameters,
		TransactionParameters: r.transactionParameters,
		Context:               r,
		Kind:                  kind,
		Static:                r.static,
		Depth:                 r.depth - 1, // depth has already been incremented
		Gas:                   parameters.Gas,
		Recipient:             recipient,
		Sender:                parameters.Sender,
		Input:                 parameters.Input,
		Value:                 parameters.Value,
		CodeHash:              &codeHash,
		Code:                  code,
	}

	result, err := r.interpreter.Run(interpreterParameters)
	if err != nil || !result.Success {
		r.RestoreSnapshot(snapshot)
	} else if kind == tosca.Create || kind == tosca.Create2 {
		r.SetCode(createdAddress, tosca.Code(result.Output))
	}

	return tosca.CallResult{
		Output:         result.Output,
		GasLeft:        result.GasLeft,
		GasRefund:      result.GasRefund,
		Success:        result.Success,
		CreatedAddress: createdAddress,
	}, err
}

func hashCode(code tosca.Code) tosca.Hash {
	return tosca.Hash(crypto.Keccak256(code))
}

func createAddress(
	kind tosca.CallKind,
	sender tosca.Address,
	nonce uint64,
	salt tosca.Hash,
	initHash tosca.Hash,
) tosca.Address {
	if kind == tosca.Create {
		return tosca.Address(crypto.CreateAddress(common.Address(sender), nonce))
	}
	return tosca.Address(crypto.CreateAddress2(common.Address(sender), common.Hash(salt), initHash[:]))
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
