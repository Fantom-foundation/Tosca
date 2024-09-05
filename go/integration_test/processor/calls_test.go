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
)

const sufficientGas = tosca.Gas(500_000_000_000)

type callProperties struct {
	callType    vm.OpCode
	hasValue    bool
	sameStorage bool
	sameValue   bool
	sameSender  bool
}

func callTypesAndProperties() map[string]callProperties {
	return map[string]callProperties{
		"call": {
			callType:    vm.CALL,
			hasValue:    true,
			sameStorage: false,
			sameValue:   false,
			sameSender:  false,
		},
		"callCode": {
			callType:    vm.CALLCODE,
			hasValue:    true,
			sameStorage: true,
			sameValue:   false,
			sameSender:  false,
		},
		"delegateCall": {
			callType:    vm.DELEGATECALL,
			hasValue:    false,
			sameStorage: true,
			sameValue:   true,
			sameSender:  true,
		},
		"staticCall": {
			callType:    vm.STATICCALL,
			hasValue:    false,
			sameStorage: false,
			sameValue:   false,
			sameSender:  false,
		},
	}
}

func TestProcessor_MaximalCallDepthIsEnforced(t *testing.T) {
	for processorName, processor := range getProcessors() {
		t.Run(fmt.Sprintf("%s-MaxCallDepth", processorName), func(t *testing.T) {
			sender := tosca.Address{1}
			receiver := &tosca.Address{2}

			// put 32byte input value with 0 offset from memory to stack,
			// add 1 to it and put it back to memory with 0 offset
			code := []byte{
				byte(vm.PUSH1), byte(0),
				byte(vm.CALLDATALOAD),
				byte(vm.PUSH1), byte(1),
				byte(vm.ADD),
				byte(vm.PUSH1), byte(0),
				byte(vm.MSTORE)}

			// add stack values for call instruction
			code = append(code, pushToStack([]*big.Int{
				big.NewInt(int64(sufficientGas)),   // gas send to nested call
				new(big.Int).SetBytes(receiver[:]), // call target
				big.NewInt(0),                      // value to transfer
				big.NewInt(0),                      // argument offset
				big.NewInt(32),                     // argument size
				big.NewInt(0),                      // result offset
				big.NewInt(32),                     // result size
			})...)

			// make inner call and return 32byte value with 0 offset from memory
			code = append(code, []byte{
				byte(vm.CALL),
				byte(vm.PUSH1), byte(32),
				byte(vm.PUSH1), byte(0),
				byte(vm.RETURN),
			}...)

			blockParams := tosca.BlockParameters{}
			transaction := tosca.Transaction{
				Sender:    sender,
				Recipient: receiver,
				GasLimit:  sufficientGas,
				Nonce:     0,
				Input:     tosca.Data{},
			}
			scenario := getScenarioContext(sender, *receiver, code, sufficientGas)
			transactionContext := newScenarioContext(scenario.Before)

			// Run the processor
			result, err := processor.Run(blockParams, transaction, transactionContext)

			// Check the result.
			if err != nil || !result.Success {
				t.Errorf("execution failed with error: %v and result %v", err, result)
			} else {
				expectedDepth := uint64(1025)
				depth := big.NewInt(0).SetBytes(result.Output).Uint64()
				if depth != expectedDepth {
					t.Errorf("expected call depth is %v, got %v", expectedDepth, depth)
				}
			}
		})
	}
}

func TestProcessor_DifferentCallTypesAccessStorage(t *testing.T) {
	for processorName, processor := range getProcessors() {
		for callName, call := range callTypesAndProperties() {
			t.Run(fmt.Sprintf("%s-%s", processorName, callName), func(t *testing.T) {
				sender0 := tosca.Address{1}
				receiver0 := tosca.Address{2}
				receiver1 := tosca.Address{3}

				// store 42 at storage slot 24
				code0 := []byte{
					byte(vm.PUSH1), byte(42),
					byte(vm.PUSH1), byte(24),
					byte(vm.SSTORE),
				}
				// set call arguments
				code0 = append(code0, pushCallArguments(call, sufficientGas, tosca.Value{}, receiver1)...)
				// perform call and forward the result
				code0 = append(code0, []byte{
					byte(call.callType),
					byte(vm.PUSH1), byte(32),
					byte(vm.PUSH1), byte(0),
					byte(vm.RETURN),
				}...)

				// inner call, read from storage slot 24 and return its value
				code1 := []byte{
					byte(vm.PUSH1), byte(24),
					byte(vm.SLOAD),
					byte(vm.PUSH1), byte(0),
					byte(vm.MSTORE),
					byte(vm.PUSH1), byte(32),
					byte(vm.PUSH1), byte(0),
					byte(vm.RETURN),
				}

				state := WorldState{
					sender0:   Account{},
					receiver0: Account{Code: code0},
					receiver1: Account{Code: code1},
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
				if call.sameStorage {
					if !slices.Equal(result.Output, append(bytes.Repeat([]byte{0}, 31), byte(42))) {
						t.Errorf("%s did not access the same storage, got unexpected output: %v", callName, result.Output)
					}
				} else {
					if !slices.Equal(result.Output, bytes.Repeat([]byte{0}, 32)) {
						t.Errorf("%s did access the same storage, got unexpected output: %v", callName, result.Output)
					}
				}
			})
		}
	}
}

func TestProcessor_DifferentCallTypesHandleValueCorrectly(t *testing.T) {
	for processorName, processor := range getProcessors() {
		for callName, call := range callTypesAndProperties() {
			t.Run(fmt.Sprintf("%s-%s", processorName, callName), func(t *testing.T) {
				sender0 := tosca.Address{1}
				receiver0 := tosca.Address{2}
				receiver1 := tosca.Address{3}
				senderBalance := tosca.NewValue(100)
				transferValue := tosca.NewValue(42)

				// set call arguments
				code0 := pushCallArguments(call, sufficientGas, tosca.NewValue(24), receiver1)
				// perform call and forward the result
				code0 = append(code0, []byte{
					byte(call.callType),
					byte(vm.PUSH1), byte(32),
					byte(vm.PUSH1), byte(0),
					byte(vm.RETURN),
				}...)

				// inner call, get value from call and return it
				code1 := []byte{
					byte(vm.CALLVALUE),
					byte(vm.PUSH1), byte(0),
					byte(vm.MSTORE),
					byte(vm.PUSH1), byte(32),
					byte(vm.PUSH1), byte(0),
					byte(vm.RETURN),
				}

				state := WorldState{
					sender0:   Account{Balance: senderBalance},
					receiver0: Account{Code: code0},
					receiver1: Account{Code: code1},
				}
				transaction := tosca.Transaction{
					Sender:    sender0,
					Recipient: &receiver0,
					GasLimit:  sufficientGas,
					Value:     transferValue,
				}

				transactionContext := newScenarioContext(state)

				// Run the processor
				result, err := processor.Run(tosca.BlockParameters{}, transaction, transactionContext)
				if err != nil || !result.Success {
					t.Errorf("execution was not successful or failed with error %v", err)
				}
				if call.sameValue {
					if !slices.Equal(result.Output, append(bytes.Repeat([]byte{0}, 31), byte(42))) {
						t.Errorf("%s did not forward value, got unexpected output: %v", callName, result.Output)
					}
				} else if call.hasValue {
					if !slices.Equal(result.Output, append(bytes.Repeat([]byte{0}, 31), byte(24))) {
						t.Errorf("%s did not handle value correctly, got unexpected output: %v", callName, result.Output)
					}
				} else {
					if !slices.Equal(result.Output, bytes.Repeat([]byte{0}, 32)) {
						t.Errorf("%s did forward value, got unexpected output: %v", callName, result.Output)
					}
				}
			})
		}
	}
}

func TestProcessor_DifferentCallTypesSetTheCorrectSender(t *testing.T) {
	for processorName, processor := range getProcessors() {
		for callName, call := range callTypesAndProperties() {
			t.Run(fmt.Sprintf("%s-%s", processorName, callName), func(t *testing.T) {
				sender0 := tosca.Address{1}
				receiver0 := tosca.Address{2}
				receiver1 := tosca.Address{3}

				// set call arguments
				code0 := pushCallArguments(call, sufficientGas, tosca.Value{}, receiver1)
				// perform call and forward the result
				code0 = append(code0, []byte{
					byte(call.callType),
					byte(vm.PUSH1), byte(32),
					byte(vm.PUSH1), byte(0),
					byte(vm.RETURN),
				}...)

				// inner call, get caller and return it
				code1 := []byte{
					byte(vm.CALLER),
					byte(vm.PUSH1), byte(0),
					byte(vm.MSTORE),
					byte(vm.PUSH1), byte(32),
					byte(vm.PUSH1), byte(0),
					byte(vm.RETURN),
				}

				state := WorldState{
					sender0:   Account{},
					receiver0: Account{Code: code0},
					receiver1: Account{Code: code1},
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
				if call.sameSender {
					if !slices.Equal(result.Output, append(bytes.Repeat([]byte{0}, 12), sender0[:]...)) {
						t.Errorf("%s did not set the correct sender, got unexpected output: %v", callName, result.Output)
					}
				} else {
					if !slices.Equal(result.Output, append(bytes.Repeat([]byte{0}, 12), receiver0[:]...)) {
						t.Errorf("%s did not set the correct sender, got unexpected output: %v", callName, result.Output)
					}
				}
			})
		}
	}
}

func TestProcessor_RecursiveCallsAfterAStaticCallAreStatic(t *testing.T) {
	for processorName, processor := range getProcessors() {
		calls := callTypesAndProperties()
		for callName, call := range calls {
			t.Run(fmt.Sprintf("%s-%s", processorName, callName), func(t *testing.T) {

				// Test structure:
				// sender0 --CALL--> receiver0 --STATICCALL--> receiver1
				// --test.CALL--> receiver2 -> SSTORE
				// after the first static calls all following calls have to be static,
				// therefore the SSTORE shall revert. The success boolean of the test.CALL
				// is returned as output.

				sender0 := tosca.Address{1}
				receiver0 := tosca.Address{2}
				receiver1 := tosca.Address{3}
				receiver2 := tosca.Address{4}

				code0 := pushCallArguments(calls["staticCall"], sufficientGas, tosca.Value{}, receiver1)
				// perform static call and forward the result
				code0 = append(code0, []byte{
					byte(vm.STATICCALL),
					byte(vm.PUSH1), byte(32),
					byte(vm.PUSH1), byte(0),
					byte(vm.RETURN),
				}...)

				code1 := pushCallArguments(call, sufficientGas, tosca.Value{}, receiver2)
				// perform call and return whether it was successful
				code1 = append(code1, []byte{
					byte(call.callType),
					byte(vm.PUSH1), byte(0),
					byte(vm.MSTORE),
					byte(vm.PUSH1), byte(32),
					byte(vm.PUSH1), byte(0),
					byte(vm.RETURN),
				}...)

				// perform sstore (should revert in static mode)
				code2 := []byte{
					byte(vm.PUSH1), byte(0),
					byte(vm.PUSH1), byte(0),
					byte(vm.SSTORE),
					byte(vm.PUSH1), byte(32),
					byte(vm.PUSH1), byte(0),
					byte(vm.RETURN),
				}

				state := WorldState{
					sender0:   Account{},
					receiver0: Account{Code: code0},
					receiver1: Account{Code: code1},
					receiver2: Account{Code: code2},
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
				if !slices.Equal(result.Output, bytes.Repeat([]byte{0}, 32)) {
					t.Errorf("%s did not enforce static mode and returned %v", callName, result.Output)
				}
			})
		}
	}
}

func TestProcessor_CallingNonExistentAccountIsHandledCorrectly(t *testing.T) {
	tests := map[string]struct {
		account            Account
		callValue          tosca.Value
		accountExistsAfter bool
	}{
		"empty": {
			account:            Account{},
			callValue:          tosca.Value{},
			accountExistsAfter: false,
		},
		"callWithValue": {
			account:            Account{},
			callValue:          tosca.NewValue(42),
			accountExistsAfter: true,
		},
		"existingAccount": {
			account:            Account{Nonce: 1},
			callValue:          tosca.Value{},
			accountExistsAfter: true,
		},
	}
	for processorName, processor := range getProcessors() {
		for testName, test := range tests {
			t.Run(processorName+"/"+testName, func(t *testing.T) {
				sender := tosca.Address{1}
				receiver := tosca.Address{2}

				state := WorldState{
					sender:   Account{Balance: tosca.NewValue(100)},
					receiver: test.account,
				}
				transaction := tosca.Transaction{
					Sender:    sender,
					Recipient: &receiver,
					GasLimit:  sufficientGas,
					Value:     test.callValue,
				}

				transactionContext := newScenarioContext(state)

				// Run the processor
				result, err := processor.Run(tosca.BlockParameters{}, transaction, transactionContext)
				if err != nil {
					t.Errorf("execution failed with error %v", err)
				}
				if !result.Success {
					t.Errorf("execution was not successful")
				}
				if test.accountExistsAfter != transactionContext.AccountExists(receiver) {
					t.Errorf("account has not been created")
				}
			})
		}
	}
}

func pushCallArguments(
	call callProperties,
	gasLimit tosca.Gas,
	transferValue tosca.Value,
	receiver tosca.Address,
) []byte {
	var code []byte
	if call.hasValue {
		code = pushToStack([]*big.Int{
			big.NewInt(int64(gasLimit)),        // gas send to nested call
			new(big.Int).SetBytes(receiver[:]), // call target
			transferValue.ToBig(),              // value to transfer
			big.NewInt(0),                      // argument offset
			big.NewInt(32),                     // argument size
			big.NewInt(0),                      // result offset
			big.NewInt(32),                     // result size
		})
	} else {
		code = pushToStack([]*big.Int{
			big.NewInt(int64(gasLimit)),        // gas send to nested call
			new(big.Int).SetBytes(receiver[:]), // call target
			big.NewInt(0),                      // argument offset
			big.NewInt(32),                     // argument size
			big.NewInt(0),                      // result offset
			big.NewInt(32),                     // result size
		})
	}
	return code
}

func pushToStack(values []*big.Int) []byte {
	code := []byte{}
	for i := len(values) - 1; i >= 0; i-- {
		valueBytes := values[i].Bytes()
		if len(valueBytes) == 0 {
			valueBytes = []byte{0}
		}
		push := vm.PUSH1 + vm.OpCode(len(valueBytes)-1)
		code = append(code, byte(push))
		code = append(code, valueBytes...)
	}
	return code
}
