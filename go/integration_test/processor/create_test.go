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

	"github.com/Fantom-foundation/Tosca/go/tosca"
	"github.com/Fantom-foundation/Tosca/go/tosca/vm"
)

func TestProcessor_CreateAndCallContract(t *testing.T) {
	gasLimit := big.NewInt(int64(sufficientGas))
	gasPush := vm.PUSH1 + vm.OpCode(len(gasLimit.Bytes())-1)

	for processorName, processor := range getProcessors() {
		if strings.Contains(processorName, "floria") {
			continue // todo implement different create types
		}
		for _, create := range []vm.OpCode{vm.CREATE, vm.CREATE2} {
			t.Run(fmt.Sprintf("%s-%s", processorName, create.String()), func(t *testing.T) {
				sender0 := tosca.Address{1}
				receiver0 := tosca.Address{2}
				toBeCreatedCodeHolder := tosca.Address{3}
				initCodeHolder := tosca.Address{4}

				initCodeOffset := 32

				input := byte(42)
				increment := byte(24)

				// code to be created
				codeToBeCreated := []byte{
					byte(vm.PUSH1), byte(0),
					byte(vm.CALLDATALOAD),
					byte(vm.PUSH1), increment,
					byte(vm.ADD),
					byte(vm.PUSH1), byte(0),
					byte(vm.MSTORE),
					byte(vm.PUSH1), byte(32),
					byte(vm.PUSH1), byte(0),
					byte(vm.RETURN),
				}

				// save code to be created to memory
				initCode := saveCodeFromAccountToMemory(
					toBeCreatedCodeHolder,
					byte(len(codeToBeCreated)),
					byte(0),
				)

				// get code to be created from memory and return it
				initCode = append(initCode, []byte{
					byte(vm.PUSH1), byte(len(codeToBeCreated)),
					byte(vm.PUSH1), byte(0),
					byte(vm.RETURN),
				}...)

				// save input to memory
				baseCode := []byte{
					byte(vm.PUSH1), byte(0),
					byte(vm.CALLDATALOAD),
					byte(vm.PUSH1), byte(0),
					byte(vm.MSTORE),
				}

				// save init code to memory
				baseCode = append(baseCode, saveCodeFromAccountToMemory(
					initCodeHolder,
					byte(len(initCode)),
					byte(initCodeOffset),
				)...)

				// input for the following call
				baseCode = append(baseCode, []byte{
					byte(vm.PUSH1), byte(32), // result size
					byte(vm.PUSH1), byte(0), // result offset
					byte(vm.PUSH1), byte(32), // input size
					byte(vm.PUSH1), byte(0), // input offset
					byte(vm.PUSH1), byte(0), // value
				}...)

				// Add salt for CREATE2
				if create == vm.CREATE2 {
					baseCode = append(baseCode, byte(vm.PUSH1), byte(0))
				}

				// Create the contract
				baseCode = append(baseCode, []byte{
					byte(vm.PUSH1), byte(len(initCode)), // input size
					byte(vm.PUSH1), byte(initCodeOffset), // input offset
					byte(vm.PUSH1), byte(0), // value
					byte(create),
				}...)

				// gas for the call
				baseCode = append(baseCode, byte(gasPush))
				baseCode = append(baseCode, gasLimit.Bytes()...)

				// Call contract and return result
				baseCode = append(baseCode, []byte{
					byte(vm.CALL),
					byte(vm.PUSH1), byte(32),
					byte(vm.PUSH1), byte(0),
					byte(vm.RETURN),
				}...)

				state := WorldState{
					sender0:               Account{},
					receiver0:             Account{Code: baseCode},
					toBeCreatedCodeHolder: Account{Code: codeToBeCreated},
					initCodeHolder:        Account{Code: initCode},
				}
				transaction := tosca.Transaction{
					Sender:    sender0,
					Recipient: &receiver0,
					GasLimit:  sufficientGas,
					Input:     append(bytes.Repeat([]byte{0}, 31), input),
				}

				transactionContext := newScenarioContext(state)

				// Run the processor
				result, err := processor.Run(tosca.BlockParameters{}, transaction, transactionContext)
				if err != nil || !result.Success {
					t.Errorf("execution was not successful or failed with error %v", err)
				}
				if !slices.Equal(result.Output, append(bytes.Repeat([]byte{0}, 31), input+increment)) {
					t.Errorf("creation of contract or its call was not successful, returned %v", result.Output)
				}
			})
		}
	}
}

func TestProcessor_CreateInitCodeIsExecutedInRightContext(t *testing.T) {
	gasLimit := big.NewInt(int64(sufficientGas))
	gasPush := vm.PUSH1 + vm.OpCode(len(gasLimit.Bytes())-1)

	for processorName, processor := range getProcessors() {
		if strings.Contains(processorName, "floria") {
			continue // todo implement different create types
		}
		for _, create := range []vm.OpCode{vm.CREATE, vm.CREATE2} {
			t.Run(fmt.Sprintf("%s-%s", processorName, create.String()), func(t *testing.T) {
				sender0 := tosca.Address{1}
				receiver0 := tosca.Address{2}
				toBeCreatedCodeHolder := tosca.Address{3}
				initCodeHolder := tosca.Address{4}

				initCodeOffset := 64

				input := byte(42)
				increment := byte(24)
				otherValue := byte(5)

				// code to be created
				codeToBeCreated := []byte{
					byte(vm.PUSH1), byte(0),
					byte(vm.SLOAD),
					byte(vm.PUSH1), increment,
					byte(vm.ADD),
					byte(vm.PUSH1), byte(0),
					byte(vm.MSTORE),
					byte(vm.PUSH1), byte(32),
					byte(vm.PUSH1), byte(0),
					byte(vm.RETURN),
				}

				// save code to be created to memory
				initCode := saveCodeFromAccountToMemory(
					toBeCreatedCodeHolder,
					byte(len(codeToBeCreated)),
					byte(0),
				)

				// save input to storage
				initCode = append(initCode, []byte{
					byte(vm.PUSH1), byte(0),
					byte(vm.PUSH1), input,
					byte(vm.PUSH1), byte(0),
					byte(vm.SSTORE),
				}...)

				// get code to be created from memory and return it
				initCode = append(initCode, []byte{
					byte(vm.PUSH1), byte(len(codeToBeCreated)),
					byte(vm.PUSH1), byte(0),
					byte(vm.RETURN),
				}...)

				// save input to memory
				baseCode := []byte{
					byte(vm.PUSH1), byte(0),
					byte(vm.CALLDATALOAD),
					byte(vm.PUSH1), byte(0),
					byte(vm.MSTORE),
				}

				// set same storage location as init code with a different value
				baseCode = append(baseCode, []byte{
					byte(vm.PUSH1), otherValue,
					byte(vm.PUSH1), byte(0),
					byte(vm.SSTORE),
				}...)

				// save init code to memory
				baseCode = append(baseCode, saveCodeFromAccountToMemory(
					initCodeHolder,
					byte(len(initCode)),
					byte(initCodeOffset),
				)...)

				// input for the following call
				baseCode = append(baseCode, []byte{
					byte(vm.PUSH1), byte(32), // result size
					byte(vm.PUSH1), byte(0), // result offset
					byte(vm.PUSH1), byte(32), // input size
					byte(vm.PUSH1), byte(0), // input offset
					byte(vm.PUSH1), byte(0), // value
				}...)

				// Add salt for CREATE2
				if create == vm.CREATE2 {
					baseCode = append(baseCode, byte(vm.PUSH1), byte(0))
				}

				// Create the contract
				baseCode = append(baseCode, []byte{
					byte(vm.PUSH1), byte(len(initCode)), // input size
					byte(vm.PUSH1), byte(initCodeOffset), // input offset
					byte(vm.PUSH1), byte(0), // value
					byte(create),
				}...)

				// gas for the call
				baseCode = append(baseCode, byte(gasPush))
				baseCode = append(baseCode, gasLimit.Bytes()...)

				// Call contract and return result
				baseCode = append(baseCode, []byte{
					byte(vm.CALL),
					byte(vm.PUSH1), byte(0),
					byte(vm.SLOAD),
					byte(vm.PUSH1), byte(32),
					byte(vm.MSTORE),
					byte(vm.PUSH1), byte(64),
					byte(vm.PUSH1), byte(0),
					byte(vm.RETURN),
				}...)

				state := WorldState{
					sender0:               Account{},
					receiver0:             Account{Code: baseCode},
					toBeCreatedCodeHolder: Account{Code: codeToBeCreated},
					initCodeHolder:        Account{Code: initCode},
				}
				transaction := tosca.Transaction{
					Sender:    sender0,
					Recipient: &receiver0,
					GasLimit:  sufficientGas,
				}

				transactionContext := newScenarioContext(state)

				// Run the processor
				result, err := processor.Run(tosca.BlockParameters{}, transaction, transactionContext)
				if err != nil || !result.Success {
					t.Errorf("execution was not successful or failed with error %v", err)
				}
				want := append(bytes.Repeat([]byte{0}, 31), input+increment)
				want = append(want, append(bytes.Repeat([]byte{0}, 31), otherValue)...)
				if !slices.Equal(result.Output, want) {
					t.Errorf("creation of contract or its call was not successful, returned %v", result.Output)
				}
			})
		}
	}
}

func TestProcessor_EmptyReceiverCreatesAccount(t *testing.T) {
	for processorName, processor := range getProcessors() {
		if strings.Contains(processorName, "floria") {
			continue // todo implement different create types
		}

		sender0 := tosca.Address{1}
		checkValue := byte(42)

		initCode := []byte{
			byte(vm.PUSH1), checkValue,
			byte(vm.PUSH1), byte(0),
			byte(vm.MSTORE),
			byte(vm.PUSH1), byte(32),
			byte(vm.PUSH1), byte(0),
			byte(vm.RETURN),
		}

		state := WorldState{
			sender0: Account{},
		}
		transaction := tosca.Transaction{
			Sender:    sender0,
			Recipient: nil,
			GasLimit:  sufficientGas,
			Input:     initCode,
		}

		transactionContext := newScenarioContext(state)

		// Run the processor
		result, err := processor.Run(tosca.BlockParameters{}, transaction, transactionContext)
		if err != nil || !result.Success {
			t.Errorf("execution was not successful or failed with error %v", err)
		}
		if !slices.Equal(result.Output, append(bytes.Repeat([]byte{0}, 31), checkValue)) {
			t.Errorf("creation of account was not successful, returned %v", result.Output)
		}
	}

}

func saveCodeFromAccountToMemory(account tosca.Address, length byte, offset byte) []byte {
	addressPush := vm.PUSH1 + vm.OpCode(len(tosca.Address{0})-1)
	code := []byte{}
	code = append(code, []byte{
		byte(vm.PUSH1), length, // input size
		byte(vm.PUSH1), byte(0), // offset in code
		byte(vm.PUSH1), offset, // memory offset
	}...)
	code = append(code, byte(addressPush))
	code = append(code, account[:]...)
	code = append(code, byte(vm.EXTCODECOPY))

	return code
}
