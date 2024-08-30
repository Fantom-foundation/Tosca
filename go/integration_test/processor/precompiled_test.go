// Copyright (c) 2024 Fantom Foundation
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at fantom.foundation/bsl11.
//
// Change Date: 2028-4-16
//
// On the date above, in accordance with the Business Source License, use of
// this software will be governed by the GNU Lesser General Public License v3.

package processor

import (
	"bytes"
	"fmt"
	"math/big"
	"slices"
	"strings"
	"testing"

	test_utils "github.com/Fantom-foundation/Tosca/go/processor"
	"github.com/Fantom-foundation/Tosca/go/processor/floria"
	"github.com/Fantom-foundation/Tosca/go/tosca"
	"github.com/Fantom-foundation/Tosca/go/tosca/vm"
)

func TestProcessor_PreCompiledContractsCanBeProcessed(t *testing.T) {
	tests := map[string]struct {
		address tosca.Address
		input   []byte
	}{
		"ecrecover":          {test_utils.NewAddress(0x1), make([]byte, 256)},
		"sha256hash":         {test_utils.NewAddress(0x2), make([]byte, 256)},
		"ripemd160hash":      {test_utils.NewAddress(0x3), make([]byte, 256)},
		"dataCopy":           {test_utils.NewAddress(0x4), make([]byte, 256)},
		"bigModExp":          {test_utils.NewAddress(0x5), make([]byte, 256)},
		"bn256Add":           {test_utils.NewAddress(0x6), make([]byte, 256)},
		"bn256ScalarMul":     {test_utils.NewAddress(0x7), make([]byte, 256)},
		"bn256Pairing":       {test_utils.NewAddress(0x8), make([]byte, 192)},
		"blake2F":            {test_utils.NewAddress(0x9), make([]byte, 213)},
		"kzgPointEvaluation": {test_utils.NewAddress(0xa), test_utils.ValidPointEvaluationInput},
	}

	for processorName, processor := range getProcessors() {
		for contractName, contract := range tests {
			t.Run(fmt.Sprintf("%s-%s", processorName, contractName), func(t *testing.T) {

				code := []byte{}
				// save input to memory
				for i := 0; i < len(contract.input); i += 32 {
					code = append(code,
						byte(vm.PUSH1), byte(i),
						byte(vm.CALLDATALOAD),
						byte(vm.PUSH1), byte(i),
						byte(vm.MSTORE),
					)
				}

				// push call arguments to stack
				code = append(code, pushToStack([]*big.Int{
					big.NewInt(int64(sufficientGas)),           // gas send to nested call
					new(big.Int).SetBytes(contract.address[:]), // call target
					big.NewInt(0),                          // value to transfer
					big.NewInt(0),                          // argument offset
					big.NewInt(int64(len(contract.input))), // argument size
					big.NewInt(0),                          // result offset
					big.NewInt(32),                         // result size
				})...)

				// call and return whether it was successful
				code = append(code, []byte{
					byte(vm.CALL),
					byte(vm.PUSH1), byte(0),
					byte(vm.MSTORE),
					byte(vm.PUSH1), byte(32),
					byte(vm.PUSH1), byte(0),
					byte(vm.RETURN),
				}...)

				sender := tosca.Address{0x42}
				receiver := tosca.Address{0x43}
				state := WorldState{
					sender:   Account{},
					receiver: Account{Code: code},
				}
				transaction := tosca.Transaction{
					Sender:    sender,
					Recipient: &receiver,
					GasLimit:  sufficientGas,
					Input:     contract.input,
				}

				transactionContext := newScenarioContext(state)
				blockParameters := tosca.BlockParameters{Revision: tosca.R13_Cancun}

				// Run the processor
				result, err := processor.Run(blockParameters, transaction, transactionContext)
				if err != nil || !result.Success {
					t.Errorf("execution was not successful or failed with error %v", err)
				}
				if !slices.Equal(result.Output, append(bytes.Repeat([]byte{0}, 31), byte(1))) {
					t.Errorf("call to precompiled contract %s was not successful", contractName)
				}
			})
		}
	}
}

func TestPrecompiled_StatePrecompiledContractSetBalance(t *testing.T) {
	accountToChange := tosca.Address{0x55}
	newBalance := byte(0x05)
	sender := tosca.Address{0x42}
	receiver := floria.DriverAddress()
	stateContractAddress := floria.StateContractAddress()

	setBalancePrefix := []byte{0xe3, 0x4, 0x43, 0xbc}
	setBalanceValidInput := append(make([]byte, 12), accountToChange[:]...)
	setBalanceValidInput = append(setBalanceValidInput, make([]byte, 31)...)
	setBalanceValidInput = append(setBalanceValidInput, newBalance)
	input := append(setBalancePrefix, setBalanceValidInput...)

	for processorName, processor := range getProcessors() {
		// When using geth as the interpreter the state precompiled contract is not set up
		if strings.Contains(processorName, "/geth") {
			continue
		}
		t.Run(processorName, func(t *testing.T) {
			code := []byte{}
			// save input to memory
			for i := 0; i < len(input); i += 32 {
				code = append(code,
					byte(vm.PUSH1), byte(i),
					byte(vm.CALLDATALOAD),
					byte(vm.PUSH1), byte(i),
					byte(vm.MSTORE),
				)
			}
			// push call arguments to stack
			code = append(code, pushToStack([]*big.Int{
				big.NewInt(int64(sufficientGas)),               // gas send to nested call
				new(big.Int).SetBytes(stateContractAddress[:]), // call target
				big.NewInt(0),                 // value to transfer
				big.NewInt(0),                 // argument offset
				big.NewInt(int64(len(input))), // argument size
				big.NewInt(0),                 // result offset
				big.NewInt(32),                // result size
			})...)
			// call state precompiled contract
			code = append(code, []byte{
				byte(vm.CALL),
				byte(vm.PUSH20),
			}...)
			// changed account address
			code = append(code, accountToChange[:]...)
			// return the balance
			code = append(code, []byte{
				byte(vm.BALANCE),
				byte(vm.PUSH1), byte(0),
				byte(vm.MSTORE),
				byte(vm.PUSH1), byte(32),
				byte(vm.PUSH1), byte(0),
				byte(vm.RETURN),
			}...)

			state := WorldState{
				sender:          Account{},
				receiver:        Account{Code: code},
				accountToChange: Account{Balance: tosca.NewValue(42000)},
			}
			transaction := tosca.Transaction{
				Sender:    sender,
				Recipient: &receiver,
				GasLimit:  sufficientGas,
				Input:     input,
			}

			transactionContext := newScenarioContext(state)
			blockParameters := tosca.BlockParameters{Revision: tosca.R13_Cancun}

			// Run the processor
			result, err := processor.Run(blockParameters, transaction, transactionContext)
			if err != nil || !result.Success {
				t.Errorf("execution was not successful or failed with error %v", err)
			}
			if !slices.Equal(result.Output, append(bytes.Repeat([]byte{0}, 31), newBalance)) {
				t.Errorf("call to state precompiled contract was not successful%v", result.Output)
			}
		})
	}
}
