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
	"testing"

	"github.com/Fantom-foundation/Tosca/go/tosca"
	"github.com/Fantom-foundation/Tosca/go/tosca/vm"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

func TestProcessor_CreateAndCallContract(t *testing.T) {
	gasLimit := big.NewInt(int64(sufficientGas))
	gasPush := vm.PUSH1 + vm.OpCode(len(gasLimit.Bytes())-1)

	for processorName, processor := range getProcessors() {
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

				gasPrice := uint64(2)
				gasLimit := tosca.Gas(100000)
				senderBalance := tosca.NewValue(uint64(gasLimit) * gasPrice)
				state := WorldState{
					sender0:               Account{Balance: senderBalance},
					receiver0:             Account{Code: baseCode},
					toBeCreatedCodeHolder: Account{Code: codeToBeCreated},
					initCodeHolder:        Account{Code: initCode},
				}
				transaction := tosca.Transaction{
					Sender:    sender0,
					Recipient: &receiver0,
					GasLimit:  gasLimit,
					Input:     append(bytes.Repeat([]byte{0}, 31), input),
					GasPrice:  tosca.NewValue(gasPrice),
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

				// Values from geth/reference implementation
				expectedBalance := tosca.NewValue(75308)
				if create == vm.CREATE2 {
					expectedBalance = tosca.NewValue(75280)
				}
				if balance := transactionContext.GetBalance(sender0); balance.Cmp(expectedBalance) != 0 {
					t.Errorf("sender balance was not calculated correctly, wanted %v, got %v",
						balance, expectedBalance)
				}
			})
		}
	}
}

func TestProcessor_CreateInitCodeIsExecutedInRightContext(t *testing.T) {
	gasLimit := big.NewInt(int64(sufficientGas))
	gasPush := vm.PUSH1 + vm.OpCode(len(gasLimit.Bytes())-1)

	for processorName, processor := range getProcessors() {
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
	for _, processor := range getProcessors() {
		checkValue := byte(42)
		sender := tosca.Address{1}
		addressToBeCreated := tosca.Address(crypto.CreateAddress(common.Address(sender), 0))

		initCode := []byte{
			byte(vm.PUSH1), checkValue,
			byte(vm.PUSH1), byte(0),
			byte(vm.MSTORE),
			byte(vm.PUSH1), byte(32),
			byte(vm.PUSH1), byte(0),
			byte(vm.RETURN),
		}
		state := WorldState{
			sender: Account{},
		}
		transaction := tosca.Transaction{
			Sender:    sender,
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
		if *result.ContractAddress != addressToBeCreated {
			t.Errorf("account was created with the wrong address, returned %v", result.ContractAddress)
		}
	}
}

func TestProcessor_CorrectAddressIsCreated(t *testing.T) {
	gasLimit := big.NewInt(int64(sufficientGas))
	gasPush := vm.PUSH1 + vm.OpCode(len(gasLimit.Bytes())-1)

	for processorName, processor := range getProcessors() {
		for _, create := range []vm.OpCode{vm.CREATE, vm.CREATE2} {
			t.Run(fmt.Sprintf("%s-%s", processorName, create.String()), func(t *testing.T) {
				sender := tosca.Address{1}
				receiver := tosca.Address{2}
				toBeCreatedCodeHolder := tosca.Address{3}
				initCodeHolder := tosca.Address{4}

				initCodeOffset := 64
				saltByte := byte(55)

				// code to be created
				codeToBeCreated := []byte{
					byte(vm.ADDRESS),
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

				// save init code to memory
				baseCode := saveCodeFromAccountToMemory(
					initCodeHolder,
					byte(len(initCode)),
					byte(initCodeOffset),
				)

				// Add salt for CREATE2
				if create == vm.CREATE2 {
					baseCode = append(baseCode, byte(vm.PUSH1), saltByte)
				}

				// Create the contract
				baseCode = append(baseCode, []byte{
					byte(vm.PUSH1), byte(len(initCode)), // input size
					byte(vm.PUSH1), byte(initCodeOffset), // input offset
					byte(vm.PUSH1), byte(0), // value
					byte(create),
				}...)

				// Safe created address to memory
				baseCode = append(baseCode, []byte{
					byte(vm.PUSH1), byte(32),
					byte(vm.MSTORE),
				}...)

				// input for the call
				baseCode = append(baseCode, []byte{
					byte(vm.PUSH1), byte(32), // result size
					byte(vm.PUSH1), byte(0), // result offset
					byte(vm.PUSH1), byte(0), // input size
					byte(vm.PUSH1), byte(0), // input offset
					byte(vm.PUSH1), byte(0), // value
					byte(vm.PUSH1), byte(32), // memory offset for address
					byte(vm.MLOAD), // load address
				}...)

				// gas for the call
				baseCode = append(baseCode, byte(gasPush))
				baseCode = append(baseCode, gasLimit.Bytes()...)

				// Call contract and return result
				baseCode = append(baseCode, []byte{
					byte(vm.CALL),
					byte(vm.PUSH1), byte(64),
					byte(vm.PUSH1), byte(0),
					byte(vm.RETURN),
				}...)

				state := WorldState{
					sender:                Account{},
					receiver:              Account{Code: baseCode, Nonce: 44},
					toBeCreatedCodeHolder: Account{Code: codeToBeCreated},
					initCodeHolder:        Account{Code: initCode},
				}
				transaction := tosca.Transaction{
					Sender:    sender,
					Recipient: &receiver,
					GasLimit:  sufficientGas,
				}

				transactionContext := newScenarioContext(state)

				codeHash := [32]byte(crypto.Keccak256Hash(initCode))
				salt := append(bytes.Repeat([]byte{0}, 31), saltByte)
				wantAddress := tosca.Address(crypto.CreateAddress(common.Address(receiver), state[receiver].Nonce))
				if create == vm.CREATE2 {
					wantAddress = tosca.Address(crypto.CreateAddress2(common.Address(receiver), [32]byte(salt), codeHash[:]))
				}

				// Run the processor
				result, err := processor.Run(tosca.BlockParameters{}, transaction, transactionContext)
				if err != nil || !result.Success {
					t.Errorf("execution was not successful or failed with error %v", err)
				}
				if !slices.Equal(wantAddress[:], result.Output[12:32]) {
					t.Errorf("contract address was not created correctly, returned %v vs %v", result.Output[12:32], wantAddress[:])
				}
				if !slices.Equal(wantAddress[:], result.Output[44:64]) {
					t.Errorf("contract address was not created correctly, returned %v vs %v", result.Output[44:64], wantAddress[:])
				}
			})
		}
	}
}

func TestProcessor_CreateExistingAccountFails(t *testing.T) {
	storageWithEntry := Storage{}
	storageWithEntry[tosca.Key{42}] = tosca.Word{42}

	tests := map[string]struct {
		account Account
		success bool
	}{
		"empty": {
			account: Account{},
			success: true,
		},
		"withCode": {
			account: Account{Code: []byte{byte(vm.PUSH1), byte(0), byte(vm.RETURN)}},
			success: false,
		},
		"withNonce": {
			account: Account{Nonce: 1},
			success: false,
		},

		// It is possible to create an account if it already had balance
		"withBalance": {
			account: Account{Balance: tosca.NewValue(42)},
			success: true,
		},

		// Different to ethereum, on Sonic accounts can only have set storage if their nonce != 0
		"withStorage": {
			account: Account{Nonce: 1, Storage: storageWithEntry},
			success: false,
		},
	}

	for processorName, processor := range getProcessors() {
		for testName, test := range tests {
			t.Run(processorName+"/"+testName, func(t *testing.T) {
				checkValue := byte(42)
				sender := tosca.Address{1}
				addressToBeCreated := tosca.Address(crypto.CreateAddress(common.Address(sender), 0))

				initCode := []byte{
					byte(vm.PUSH1), checkValue,
					byte(vm.PUSH1), byte(0),
					byte(vm.MSTORE),
					byte(vm.PUSH1), byte(32),
					byte(vm.PUSH1), byte(0),
					byte(vm.RETURN),
				}
				state := WorldState{
					sender:             Account{},
					addressToBeCreated: test.account,
				}
				originalState := state.Clone()

				transaction := tosca.Transaction{
					Sender:    sender,
					Recipient: nil,
					GasLimit:  sufficientGas,
					Input:     initCode,
				}

				transactionContext := newScenarioContext(state)

				// Run the processor
				result, err := processor.Run(tosca.BlockParameters{}, transaction, transactionContext)
				if err != nil {
					t.Errorf("execution failed with error %v", err)
				}
				if result.Success != test.success {
					t.Errorf("execution success was %v, expected %v", result.Success, test.success)
				}
				if test.success {
					if !slices.Equal(result.Output, append(bytes.Repeat([]byte{0}, 31), checkValue)) {
						t.Errorf("creation returned successful but with wrong output %v", result.Output)
					}
					if state[addressToBeCreated].Code != nil {
						t.Errorf("account was created with code, expected nil")
					}
					if state[addressToBeCreated].Nonce != 0 {
						t.Errorf("account was created with nonce %d, expected 0", state[addressToBeCreated].Nonce)
					}
					if state[addressToBeCreated].Balance.Cmp(test.account.Balance) != 0 {
						t.Errorf("account was created with balance %d, expected %d", state[addressToBeCreated].Balance, test.account.Balance)
					}
				} else {
					if result.GasUsed != sufficientGas {
						t.Errorf("execution failed but gas was not fully used, used %d", result.GasUsed)
					}
					if !state.Equal(originalState) {
						t.Errorf("state was changed although execution failed, state %v", state.Diff(originalState))
					}
				}
			})
		}
	}
}

func TestProcessor_CodeStartingWith0xEFCanNotBeCreated(t *testing.T) {
	tests := map[string]struct {
		revision         tosca.Revision
		firstInstruction byte
		success          bool
	}{
		"preLondon0xEF": {
			revision:         tosca.R09_Berlin,
			firstInstruction: byte(0xEF),
			success:          true,
		},
		"london0xEF": {
			revision:         tosca.R10_London,
			firstInstruction: byte(0xEF),
			success:          false,
		},
		"postLondon0xEF": {
			revision:         tosca.R13_Cancun,
			firstInstruction: byte(0xEF),
			success:          false,
		},
		"preLondonNo0xEF": {
			revision:         tosca.R09_Berlin,
			firstInstruction: byte(vm.STOP),
			success:          true,
		},
		"londonNo0xEF": {
			revision:         tosca.R10_London,
			firstInstruction: byte(vm.STOP),
			success:          true,
		},
		"postLondonNo0xEF": {
			revision:         tosca.R13_Cancun,
			firstInstruction: byte(vm.STOP),
			success:          true,
		},
	}

	for processorName, processor := range getProcessors() {
		for testName, test := range tests {
			t.Run(processorName+"/"+testName, func(t *testing.T) {
				sender := tosca.Address{1}
				addressToBeCreated := tosca.Address(crypto.CreateAddress(common.Address(sender), 0))

				initCode := []byte{
					byte(vm.PUSH1), test.firstInstruction,
					byte(vm.PUSH1), byte(0),
					byte(vm.MSTORE8),
					byte(vm.PUSH1), byte(32),
					byte(vm.PUSH1), byte(0),
					byte(vm.RETURN),
				}
				state := WorldState{
					sender: Account{},
				}
				transaction := tosca.Transaction{
					Sender:    sender,
					Recipient: nil,
					GasLimit:  sufficientGas,
					Input:     initCode,
				}

				blockParameters := tosca.BlockParameters{Revision: test.revision}
				transactionContext := newScenarioContext(state)

				// Run the processor
				result, err := processor.Run(blockParameters, transaction, transactionContext)
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if result.Success != test.success {
					t.Errorf("execution success was %v, expected %v", result.Success, test.success)
				}
				if test.success {
					if len(transactionContext.GetCode(addressToBeCreated)) == 0 {
						t.Errorf("Code has not been set correctly")
					}
				} else {
					if code := transactionContext.GetCode(addressToBeCreated); len(code) != 0 {
						t.Errorf("Code should have not been set but returned %v", code)
					}
					if result.GasUsed != sufficientGas {
						t.Errorf("execution failed but gas was not fully used, used %d", result.GasUsed)
					}
				}
			})
		}
	}
}

func TestProcessor_CodeSizeIsLimited(t *testing.T) {
	maxCodeSize := 24576
	tests := map[string]struct {
		length  int
		success bool
	}{
		"threshold": {
			length:  maxCodeSize,
			success: true,
		},
		"exceedingThreshold": {
			length:  maxCodeSize + 1,
			success: false,
		},
	}

	for processorName, processor := range getProcessors() {
		for testName, test := range tests {
			t.Run(processorName+"/"+testName, func(t *testing.T) {
				sender := tosca.Address{1}
				addressToBeCreated := tosca.Address(crypto.CreateAddress(common.Address(sender), 0))

				initCode := []byte{byte(vm.PUSH2)}
				initCode = append(initCode, big.NewInt(int64(test.length)).Bytes()...)
				initCode = append(initCode, []byte{
					byte(vm.PUSH1), byte(0),
					byte(vm.RETURN),
				}...)
				state := WorldState{
					sender: Account{},
				}
				transaction := tosca.Transaction{
					Sender:    sender,
					Recipient: nil,
					GasLimit:  sufficientGas,
					Input:     initCode,
				}

				blockParameters := tosca.BlockParameters{}
				transactionContext := newScenarioContext(state)

				// Run the processor
				result, err := processor.Run(blockParameters, transaction, transactionContext)
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if result.Success != test.success {
					t.Errorf("execution success was %v, expected %v", result.Success, test.success)
				}
				if test.success {
					if len(transactionContext.GetCode(addressToBeCreated)) == 0 {
						t.Errorf("Code has not been set correctly")
					}
				} else {
					if code := transactionContext.GetCode(addressToBeCreated); len(code) != 0 {
						t.Errorf("Code should have not been set but returned %v", code)
					}
					if result.GasUsed != sufficientGas {
						t.Errorf("execution failed but gas was not fully used, used %d", result.GasUsed)
					}
				}
			})
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
