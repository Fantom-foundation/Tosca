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
	gas := transaction.GasLimit

	if err := buyGas(transaction, context); err != nil {
		return errorReceipt, nil
	}

	intrinsicGas := setupGasBilling(transaction)
	if gas < intrinsicGas {
		return errorReceipt, nil
	}
	gas -= intrinsicGas

	if err := handleNonce(transaction, context); err != nil {
		return errorReceipt, nil
	}

	isCreate := false
	if transaction.Recipient == nil {
		isCreate = true
	}

	var result tosca.Result
	var err error

	if isCreate {
		// Create new contract
	} else {
		// Call existing contract
		result, err = call(p.interpreter, transaction, context, gas)
		if err != nil {
			return errorReceipt, err
		}
	}

	gasUsed := gasUsed(transaction, result.GasLeft)

	return tosca.Receipt{
		Success:         result.Success,
		GasUsed:         gasUsed,
		ContractAddress: nil,
		Output:          result.Output,
		Logs:            nil,
	}, nil
}

func gasUsed(transaction tosca.Transaction, gasLeft tosca.Gas) tosca.Gas {
	// 10% of remaining gas is charged for non-internal transactions
	if transaction.Sender != (tosca.Address{}) {
		gasLeft -= gasLeft / 10
	}

	return transaction.GasLimit - gasLeft
}

func setupGasBilling(transaction tosca.Transaction) tosca.Gas {
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
		gas += zeroBytes * TxDataZeroGasEIP2028
		gas += nonZeroBytes * TxDataNonZeroGasEIP2028
	}

	// No overflow check for the gas computation is required although it is performed in the
	// opera version. The overflow check would be triggered in a worst case with an input
	// greater than 2^64 / 16 - 53000 = ~10^18, which is not possible with real world hardware
	if transaction.AccessList != nil {
		gas += tosca.Gas(len(transaction.AccessList)) * TxAccessListAddressGas

		// charge for each storage key
		for _, accessTuple := range transaction.AccessList {
			gas += tosca.Gas(len(accessTuple.Keys)) * TxAccessListStorageKeyGas
		}
	}

	return tosca.Gas(gas)
}

func handleNonce(transaction tosca.Transaction, context tosca.TransactionContext) error {
	stateNonce := context.GetNonce(transaction.Sender)
	messageNonce := transaction.Nonce
	if messageNonce != stateNonce {
		return fmt.Errorf("nonce mismatch: %v != %v", messageNonce, stateNonce)
	}
	context.SetNonce(transaction.Sender, stateNonce+1)
	return nil
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
