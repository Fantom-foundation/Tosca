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

const (
	TxGas                     = 21_000
	TxGasContractCreation     = 53_000
	TxDataNonZeroGasEIP2028   = 16
	TxDataZeroGasEIP2028      = 4
	TxAccessListAddressGas    = 2400
	TxAccessListStorageKeyGas = 1900

	createGasCostPerByte = 200
	maxCodeSize          = 24576
	maxInitCodeSize      = 2 * maxCodeSize

	MaxRecursiveDepth = 1024 // Maximum depth of call/create stack.
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
	blockParameters tosca.BlockParameters,
	transaction tosca.Transaction,
	context tosca.TransactionContext,
) (tosca.Receipt, error) {
	errorReceipt := tosca.Receipt{
		Success: false,
		GasUsed: transaction.GasLimit,
	}
	gas := transaction.GasLimit

	if nonceCheck(transaction.Nonce, context.GetNonce(transaction.Sender)) != nil {
		return tosca.Receipt{}, nil
	}

	if eoaCheck(transaction.Sender, context) != nil {
		return tosca.Receipt{}, nil
	}

	if err := buyGas(transaction, context); err != nil {
		return tosca.Receipt{}, nil
	}

	setupGas := calculateSetupGas(transaction)
	if gas < setupGas {
		return errorReceipt, nil
	}
	gas -= setupGas

	if blockParameters.Revision >= tosca.R12_Shanghai && transaction.Recipient == nil &&
		len(transaction.Input) > maxInitCodeSize {
		return tosca.Receipt{}, nil
	}

	transactionParameters := tosca.TransactionParameters{
		Origin:     transaction.Sender,
		GasPrice:   transaction.GasPrice,
		BlobHashes: []tosca.Hash{}, // ?
	}

	runContext := runContext{
		context,
		p.interpreter,
		blockParameters,
		transactionParameters,
		0,
		false,
	}

	if blockParameters.Revision >= tosca.R09_Berlin {
		setUpAccessList(transaction, &runContext, blockParameters.Revision)
	}

	callParameters := callParameters(transaction, gas)
	kind := callKind(transaction)

	if kind == tosca.Call {
		context.SetNonce(transaction.Sender, context.GetNonce(transaction.Sender)+1)
	}

	result, err := runContext.Call(kind, callParameters)
	if err != nil {
		return errorReceipt, err
	}

	var createdAddress *tosca.Address
	if kind == tosca.Create {
		createdAddress = &result.CreatedAddress
	}

	gasLeft := calculateGasLeft(transaction, result, blockParameters.Revision)
	refundGas(transaction, context, gasLeft)

	logs := context.GetLogs()

	return tosca.Receipt{
		Success:         result.Success,
		GasUsed:         transaction.GasLimit - gasLeft,
		ContractAddress: createdAddress,
		Output:          result.Output,
		Logs:            logs,
	}, nil
}

func nonceCheck(transactionNonce uint64, stateNonce uint64) error {
	if transactionNonce != stateNonce {
		return fmt.Errorf("nonce mismatch: %v != %v", transactionNonce, stateNonce)
	}
	if stateNonce+1 < stateNonce {
		return fmt.Errorf("nonce overflow")
	}
	return nil
}

// Only accept transactions from externally owned accounts (EOAs) and not from contracts
func eoaCheck(sender tosca.Address, context tosca.TransactionContext) error {
	codehash := context.GetCodeHash(sender)
	if codehash != (tosca.Hash{}) && codehash != emptyCodeHash {
		return fmt.Errorf("sender is not an EOA")
	}
	return nil
}

func setUpAccessList(transaction tosca.Transaction, context tosca.TransactionContext, revision tosca.Revision) {
	if transaction.AccessList == nil {
		return
	}

	context.AccessAccount(transaction.Sender)
	if transaction.Recipient != nil {
		context.AccessAccount(*transaction.Recipient)
	}

	precompiles := getPrecompiledAddresses(revision)
	for _, address := range precompiles {
		context.AccessAccount(address)
	}

	for _, accessTuple := range transaction.AccessList {
		context.AccessAccount(accessTuple.Address)
		for _, key := range accessTuple.Keys {
			context.AccessStorage(accessTuple.Address, key)
		}
	}
}

func callKind(transaction tosca.Transaction) tosca.CallKind {
	if transaction.Recipient == nil {
		return tosca.Create
	}
	return tosca.Call
}

func callParameters(transaction tosca.Transaction, gas tosca.Gas) tosca.CallParameters {
	callParameters := tosca.CallParameters{
		Sender: transaction.Sender,
		Input:  transaction.Input,
		Value:  transaction.Value,
		Gas:    gas,
	}
	if transaction.Recipient != nil {
		callParameters.Recipient = *transaction.Recipient
	}
	return callParameters
}

func calculateGasLeft(transaction tosca.Transaction, result tosca.CallResult, revision tosca.Revision) tosca.Gas {
	gasLeft := result.GasLeft

	// 10% of remaining gas is charged for non-internal transactions
	if transaction.Sender != (tosca.Address{}) {
		gasLeft -= gasLeft / 10
	}

	if result.Success {
		gasUsed := transaction.GasLimit - gasLeft
		refund := result.GasRefund

		maxRefund := tosca.Gas(0)
		if revision < tosca.R10_London {
			// Before EIP-3529: refunds were capped to gasUsed / 2
			maxRefund = gasUsed / 2
		} else {
			// After EIP-3529: refunds are capped to gasUsed / 5
			maxRefund = gasUsed / 5
		}

		if refund > maxRefund {
			refund = maxRefund
		}
		gasLeft += refund
	}

	return gasLeft
}

func refundGas(transaction tosca.Transaction, context tosca.TransactionContext, gasLeft tosca.Gas) {
	refundValue := transaction.GasPrice.Scale(uint64(gasLeft))
	senderBalance := context.GetBalance(transaction.Sender)
	senderBalance = tosca.Add(senderBalance, refundValue)
	context.SetBalance(transaction.Sender, senderBalance)
}

func calculateSetupGas(transaction tosca.Transaction) tosca.Gas {
	var gas tosca.Gas
	if transaction.Recipient == nil {
		gas = TxGasContractCreation
	} else {
		gas = TxGas
	}

	if len(transaction.Input) > 0 {
		nonZeroBytes := tosca.Gas(0)
		for _, inputByte := range transaction.Input {
			if inputByte != 0 {
				nonZeroBytes++
			}
		}
		zeroBytes := tosca.Gas(len(transaction.Input)) - nonZeroBytes

		// No overflow check for the gas computation is required although it is performed in the
		// opera version. The overflow check would be triggered in a worst case with an input
		// greater than 2^64 / 16 - 53000 = ~10^18, which is not possible with real world hardware
		gas += zeroBytes * TxDataZeroGasEIP2028
		gas += nonZeroBytes * TxDataNonZeroGasEIP2028
	}

	if transaction.AccessList != nil {
		gas += tosca.Gas(len(transaction.AccessList)) * TxAccessListAddressGas

		// charge for each storage key
		for _, accessTuple := range transaction.AccessList {
			gas += tosca.Gas(len(accessTuple.Keys)) * TxAccessListStorageKeyGas
		}
	}

	return tosca.Gas(gas)
}

func buyGas(transaction tosca.Transaction, context tosca.TransactionContext) error {
	gas := transaction.GasPrice.Scale(uint64(transaction.GasLimit))

	// Buy gas
	senderBalance := context.GetBalance(transaction.Sender)
	if senderBalance.Cmp(gas) < 0 {
		return fmt.Errorf("insufficient balance: %v < %v", senderBalance, gas)
	}

	senderBalance = tosca.Sub(senderBalance, gas)
	context.SetBalance(transaction.Sender, senderBalance)

	return nil
}
