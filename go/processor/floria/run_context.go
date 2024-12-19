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
		return r.executeCreate(kind, parameters)
	}
	return r.executeCall(kind, parameters)
}

func (r runContext) executeCall(kind tosca.CallKind, parameters tosca.CallParameters) (tosca.CallResult, error) {
	errResult := tosca.CallResult{
		Success: false,
		GasLeft: parameters.Gas,
	}
	if r.depth > MaxRecursiveDepth {
		return errResult, nil
	}
	r.depth++
	defer func() { r.depth-- }()

	if kind == tosca.Call || kind == tosca.CallCode {
		if !canTransferValue(r, parameters.Value, parameters.Sender, &parameters.Recipient) {
			return errResult, nil
		}
	}
	snapshot := r.CreateSnapshot()
	recipient := parameters.Recipient

	if kind == tosca.StaticCall {
		r.static = true
	}

	if r.blockParameters.Revision >= tosca.R09_Berlin &&
		!isPrecompiled(recipient, r.blockParameters.Revision) &&
		!isStateContract(recipient) &&
		!r.AccountExists(recipient) &&
		parameters.Value.Cmp(tosca.Value{}) == 0 {
		return tosca.CallResult{Success: true, GasLeft: parameters.Gas}, nil
	}

	if kind == tosca.Call || kind == tosca.CallCode {
		transferValue(r, parameters.Value, parameters.Sender, recipient)
	}

	if kind == tosca.Call {
		result, isStatePrecompiled := handleStateContract(
			r, parameters.Sender, recipient, parameters.Input, parameters.Gas)
		if isStatePrecompiled {
			if !result.Success {
				r.RestoreSnapshot(snapshot)
				result.GasLeft = 0
			}
			return result, nil
		}
	}

	result, isPrecompiled := handlePrecompiledContract(
		r.blockParameters.Revision, parameters.Input, recipient, parameters.Gas)
	if isPrecompiled {
		if !result.Success {
			r.RestoreSnapshot(snapshot)
			result.GasLeft = 0
		}
		return result, nil
	}

	var codeHash tosca.Hash
	var code tosca.Code
	if kind == tosca.Call || kind == tosca.StaticCall {
		codeHash = r.GetCodeHash(recipient)
		code = r.GetCode(recipient)
	} else {
		code = r.GetCode(parameters.CodeAddress)
		codeHash = r.GetCodeHash(parameters.CodeAddress)
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

	callResult, err := r.interpreter.Run(interpreterParameters)
	if err != nil || !callResult.Success {
		r.RestoreSnapshot(snapshot)

		if !isRevert(callResult, err) {
			// if the unsuccessful call was due to a revert, the gas is not consumed
			callResult.GasLeft = 0
		}
	}

	return tosca.CallResult{
		Output:    callResult.Output,
		GasLeft:   callResult.GasLeft,
		GasRefund: callResult.GasRefund,
		Success:   callResult.Success,
	}, err
}

func (r runContext) executeCreate(kind tosca.CallKind, parameters tosca.CallParameters) (tosca.CallResult, error) {
	errResult := tosca.CallResult{
		Success: false,
		GasLeft: parameters.Gas,
	}
	if r.depth > MaxRecursiveDepth {
		return errResult, nil
	}
	r.depth++
	defer func() { r.depth-- }()

	if !canTransferValue(r, parameters.Value, parameters.Sender, &parameters.Recipient) {
		return errResult, nil
	}
	if err := incrementNonce(r, parameters.Sender); err != nil {
		return errResult, nil
	}

	code := tosca.Code(parameters.Input)
	codeHash := hashCode(code)

	createdAddress := createAddress(kind, parameters.Sender, r.GetNonce(parameters.Sender)-1,
		parameters.Salt, codeHash)

	if r.blockParameters.Revision >= tosca.R09_Berlin {
		r.AccessAccount(createdAddress)
	}

	if r.GetNonce(createdAddress) != 0 ||
		(r.GetCodeHash(createdAddress) != (tosca.Hash{}) &&
			r.GetCodeHash(createdAddress) != emptyCodeHash) {
		return tosca.CallResult{}, nil
	}
	snapshot := r.CreateSnapshot()
	r.SetNonce(createdAddress, 1)

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
		Input:                 nil,
		Value:                 parameters.Value,
		CodeHash:              &codeHash,
		Code:                  code,
	}

	result, err := r.interpreter.Run(interpreterParameters)
	if err != nil || !result.Success {
		r.RestoreSnapshot(snapshot)

		if !isRevert(result, err) {
			// if the unsuccessful create was due to a revert, the result is still returned
			return tosca.CallResult{}, err
		}
		return tosca.CallResult{Output: result.Output, GasLeft: result.GasLeft, CreatedAddress: createdAddress}, nil
	}

	outCode := result.Output
	if len(outCode) > maxCodeSize {
		result.Success = false
	}
	if r.blockParameters.Revision >= tosca.R10_London && len(outCode) > 0 && outCode[0] == 0xEF {
		result.Success = false
	}
	createGas := tosca.Gas(len(outCode) * createGasCostPerByte)
	if result.GasLeft < createGas {
		result.Success = false
	}
	result.GasLeft -= createGas

	if result.Success {
		r.SetCode(createdAddress, tosca.Code(outCode))
	} else {
		r.RestoreSnapshot(snapshot)
		result.GasLeft = 0
		result.Output = nil
	}

	return tosca.CallResult{
		Output:         result.Output,
		GasLeft:        result.GasLeft,
		GasRefund:      result.GasRefund,
		Success:        result.Success,
		CreatedAddress: createdAddress,
	}, nil
}

func isRevert(result tosca.Result, err error) bool {
	if err == nil && !result.Success && (result.GasLeft > 0 || len(result.Output) > 0) {
		return true
	}
	return false
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
	recipient *tosca.Address,
) bool {
	if value == (tosca.Value{}) {
		return true
	}

	senderBalance := context.GetBalance(sender)
	if senderBalance.Cmp(value) < 0 {
		return false
	}

	if recipient == nil || sender == *recipient {
		return true
	}

	receiverBalance := context.GetBalance(*recipient)
	updatedBalance := tosca.Add(receiverBalance, value)
	if updatedBalance.Cmp(receiverBalance) < 0 || updatedBalance.Cmp(value) < 0 {
		return false
	}

	return true
}

func incrementNonce(context tosca.TransactionContext, address tosca.Address) error {
	nonce := context.GetNonce(address)
	if nonce+1 < nonce {
		return fmt.Errorf("nonce overflow")
	}
	context.SetNonce(address, nonce+1)
	return nil
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
