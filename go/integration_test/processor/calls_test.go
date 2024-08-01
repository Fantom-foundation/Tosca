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

func TestProcessor_MaximalCallDepthIsEnforced(t *testing.T) {
	gasLimit := tosca.Gas(1000000000000)
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
				big.NewInt(int64(gasLimit)),        // gas send to nested call
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
				GasLimit:  gasLimit,
				Nonce:     0,
				Input:     tosca.Data{},
			}
			scenario := getScenarioContext(sender, *receiver, code, gasLimit)
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
	tests := map[string]struct {
		call        vm.OpCode
		sameStorage bool
		hasValue    bool
	}{
		"call": {
			call:        vm.CALL,
			sameStorage: false,
			hasValue:    true,
		},
		"callCode": {
			call:        vm.CALLCODE,
			sameStorage: true,
			hasValue:    true,
		},
		"staticCall": {
			call:        vm.STATICCALL,
			sameStorage: false,
			hasValue:    false,
		},
		"delegateCall": {
			call:        vm.DELEGATECALL,
			sameStorage: true,
			hasValue:    false,
		},
	}

	gasLimit := tosca.Gas(1000000)
	for processorName, processor := range getProcessors() {
		if strings.Contains(processorName, "floria") {
			continue // todo implement different call types
		}
		for testName, test := range tests {
			t.Run(fmt.Sprintf("%s-%s", processorName, testName), func(t *testing.T) {
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
				if test.hasValue {
					code0 = append(code0, pushToStack([]*big.Int{
						big.NewInt(int64(gasLimit)),         // gas send to nested call
						new(big.Int).SetBytes(receiver1[:]), // call target
						big.NewInt(0),                       // value to transfer
						big.NewInt(0),                       // argument offset
						big.NewInt(32),                      // argument size
						big.NewInt(0),                       // result offset
						big.NewInt(32),                      // result size
					})...)
				} else {
					code0 = append(code0, pushToStack([]*big.Int{
						big.NewInt(int64(gasLimit)),         // gas send to nested call
						new(big.Int).SetBytes(receiver1[:]), // call target
						big.NewInt(0),                       // argument offset
						big.NewInt(32),                      // argument size
						big.NewInt(0),                       // result offset
						big.NewInt(32),                      // result size
					})...)
				}
				// perform call and forward the result
				code0 = append(code0, []byte{
					byte(test.call),
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
				scenario := Scenario{
					Before: state,
					Transaction: tosca.Transaction{
						Sender:    sender0,
						Recipient: &receiver0,
						GasLimit:  gasLimit,
					},
					After: state,
				}

				transactionContext := newScenarioContext(state)

				// Run the processor
				result, err := processor.Run(tosca.BlockParameters{}, scenario.Transaction, transactionContext)
				if err != nil || !result.Success {
					t.Errorf("execution was not successful or failed with error %v", err)
				}
				if test.sameStorage {
					if !slices.Equal(result.Output, append(bytes.Repeat([]byte{0}, 31), byte(42))) {
						t.Errorf("%s did not access the same storage, got unexpected output: %v", test.call.String(), result.Output)
					}
				} else {
					if !slices.Equal(result.Output, bytes.Repeat([]byte{0}, 32)) {
						t.Errorf("%s did access the same storage, got unexpected output: %v", test.call.String(), result.Output)
					}
				}
			})
		}
	}
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
