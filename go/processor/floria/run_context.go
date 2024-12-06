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
	"github.com/Fantom-foundation/Tosca/go/tosca"

	// geth dependencies
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

var emptyCodeHash = tosca.Hash(crypto.Keccak256(nil))

type runContext struct {
	tosca.TransactionContext
	interpreter           tosca.Interpreter
	blockParameters       tosca.BlockParameters
	transactionParameters tosca.TransactionParameters
	depth                 int
	static                bool
}

func (r runContext) Call(kind tosca.CallKind, parameters tosca.CallParameters) (tosca.CallResult, error) {
	if kind == tosca.Create || kind == tosca.Create2 {
		return r.Creates(kind, parameters)
	}
	return r.Calls(kind, parameters)
}

func (r runContext) Calls(kind tosca.CallKind, parameters tosca.CallParameters) (tosca.CallResult, error) {
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

	if kind == tosca.StaticCall {
		r.static = true
	}

	snapshot := r.CreateSnapshot()

	// StaticCall and DelegateCall do not transfer value
	if kind == tosca.Call || kind == tosca.CallCode {
		if !canTransferValue(r, parameters.Value, parameters.Sender, recipient) {
			r.RestoreSnapshot(snapshot)
			return tosca.CallResult{}, nil
		}
		transferValue(r, parameters.Value, parameters.Sender, recipient)
	}

	output, isStatePrecompiled := handleStateContract(
		r, parameters.Sender, parameters.Recipient, parameters.Input, parameters.Gas)
	if isStatePrecompiled {
		return output, nil
	}
	output, isPrecompiled := handlePrecompiledContract(
		r.blockParameters.Revision, parameters.Input, recipient, parameters.Gas)
	if isPrecompiled {
		return output, nil
	}

	if kind == tosca.Call && !r.AccountExists(recipient) {
		return tosca.CallResult{
			Success: true,
			GasLeft: parameters.Gas,
		}, nil
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
	}

	return tosca.CallResult{
		Output:    result.Output,
		GasLeft:   result.GasLeft,
		GasRefund: result.GasRefund,
		Success:   result.Success,
	}, err
}

func (r runContext) Creates(kind tosca.CallKind, parameters tosca.CallParameters) (tosca.CallResult, error) {
	if r.depth > MaxRecursiveDepth {
		return tosca.CallResult{}, nil
	}
	r.depth++
	defer func() { r.depth-- }()

	codeHash := r.GetCodeHash(parameters.Recipient)
	code := r.GetCode(parameters.Recipient)

	if parameters.Recipient == (tosca.Address{}) {
		code = tosca.Code(parameters.Input)
		codeHash = hashCode(code)
	}

	createdAddress := createAddress(
		kind,
		parameters.Sender,
		r.GetNonce(parameters.Sender),
		parameters.Salt,
		codeHash,
	)
	if r.GetNonce(createdAddress) != 0 ||
		(r.GetCodeHash(createdAddress) != (tosca.Hash{}) &&
			r.GetCodeHash(createdAddress) != emptyCodeHash) {
		return tosca.CallResult{}, nil
	}

	r.SetNonce(parameters.Sender, r.GetNonce(parameters.Sender)+1)
	r.SetNonce(createdAddress, 1)

	snapshot := r.CreateSnapshot()
	if !canTransferValue(r, parameters.Value, parameters.Sender, createdAddress) {
		r.RestoreSnapshot(snapshot)
		return tosca.CallResult{}, nil
	}
	transferValue(r, parameters.Value, parameters.Sender, createdAddress)

	interpreterParameters := tosca.Parameters{
		BlockParameters:       r.blockParameters,
		TransactionParameters: r.transactionParameters,
		Context:               r,
		Kind:                  kind,
		Static:                r.static,
		Depth:                 r.depth - 1, // depth has already been incremented
		Gas:                   parameters.Gas,
		Recipient:             createdAddress,
		Sender:                parameters.Sender,
		Input:                 parameters.Input,
		Value:                 parameters.Value,
		CodeHash:              &codeHash,
		Code:                  code,
	}

	result, err := r.interpreter.Run(interpreterParameters)
	if err != nil || !result.Success {
		r.RestoreSnapshot(snapshot)
	} else {
		outCode := result.Output
		if len(outCode) > maxCodeSize {
			return tosca.CallResult{}, nil
		}
		if r.blockParameters.Revision >= tosca.R10_London && len(outCode) > 0 && outCode[0] == 0xEF {
			return tosca.CallResult{}, nil
		}
		createGas := tosca.Gas(len(outCode) * createGasCostPerByte)
		if result.GasLeft < createGas {
			return tosca.CallResult{}, nil
		}
		result.GasLeft -= createGas

		r.SetCode(createdAddress, tosca.Code(outCode))
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

func canTransferValue(
	context tosca.TransactionContext,
	value tosca.Value,
	sender tosca.Address,
	recipient tosca.Address,
) bool {
	if value == (tosca.Value{}) {
		return true
	}

	senderBalance := context.GetBalance(sender)
	if senderBalance.Cmp(value) < 0 {
		return false
	}

	if sender == recipient {
		return true
	}

	receiverBalance := context.GetBalance(recipient)
	updatedBalance := tosca.Add(receiverBalance, value)
	if updatedBalance.Cmp(receiverBalance) < 0 || updatedBalance.Cmp(value) < 0 {
		return false
	}

	return true
}

// Only to be called after canTransferValue
func transferValue(
	context tosca.TransactionContext,
	value tosca.Value,
	sender tosca.Address,
	recipient tosca.Address,
) {
	if value == (tosca.Value{}) {
		return
	}
	if sender == recipient {
		return
	}

	senderBalance := context.GetBalance(sender)
	receiverBalance := context.GetBalance(recipient)
	updatedBalance := tosca.Add(receiverBalance, value)

	senderBalance = tosca.Sub(senderBalance, value)
	context.SetBalance(sender, senderBalance)
	context.SetBalance(recipient, updatedBalance)
}
