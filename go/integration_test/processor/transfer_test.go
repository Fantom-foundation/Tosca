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
	"slices"
	"testing"

	"github.com/Fantom-foundation/Tosca/go/tosca"
	"github.com/Fantom-foundation/Tosca/go/tosca/vm"
)

func TestProcessor_CallValueTransfersAreHandledCorrectly(t *testing.T) {
	for processorName, processor := range getProcessors() {
		for callName, call := range callTypesAndProperties() {
			t.Run(fmt.Sprintf("%s-%s", processorName, callName), func(t *testing.T) {
				sender0 := tosca.Address{1}
				receiver0 := tosca.Address{2}
				receiver1 := tosca.Address{3}
				value := tosca.NewValue(42)

				code0 := pushCallArguments(call, sufficientGas, value, receiver1)
				code0 = append(code0, []byte{
					byte(call.callType),
					byte(vm.PUSH1), byte(32),
					byte(vm.PUSH1), byte(0),
					byte(vm.RETURN),
				}...)

				code1 := []byte{
					byte(vm.CALLVALUE),
					byte(vm.PUSH1), byte(0),
					byte(vm.MSTORE),
					byte(vm.PUSH1), byte(32),
					byte(vm.PUSH1), byte(0),
					byte(vm.RETURN),
				}

				senderBalance := tosca.NewValue(100)
				state := WorldState{
					sender0:   Account{Balance: senderBalance},
					receiver0: Account{Code: code0},
					receiver1: Account{Code: code1},
				}
				transaction := tosca.Transaction{
					Sender:    sender0,
					Recipient: &receiver0,
					GasLimit:  sufficientGas,
					Value:     value,
				}

				transactionContext := newScenarioContext(state)

				// Run the processor
				result, err := processor.Run(tosca.BlockParameters{}, transaction, transactionContext)
				if err != nil || !result.Success {
					t.Fatalf("execution was not successful or failed with error %v", err)
				}

				value0 := value
				value1 := tosca.NewValue(0)
				callValue := append(bytes.Repeat([]byte{0}, 31), byte(42))
				if call.callType == vm.CALL {
					value0 = tosca.NewValue(0)
					value1 = value
				}
				if call.callType == vm.STATICCALL {
					callValue = bytes.Repeat([]byte{0}, 32)
				}

				if !slices.Equal(result.Output, callValue) {
					t.Errorf("call value was not transferred correctly, want %v, got %v", callValue, result.Output)
				}
				if balance := transactionContext.GetBalance(sender0); balance.Cmp(tosca.Sub(senderBalance, value)) != 0 {
					t.Errorf("sender balance was not updated correctly, want %v, got %v", balance, value)
				}
				if balance := transactionContext.GetBalance(receiver0); balance.Cmp(value0) != 0 {
					t.Errorf("receiver balance was not updated correctly, want %v, got %v", value0, balance)
				}
				if balance := transactionContext.GetBalance(receiver1); balance.Cmp(value1) != 0 {
					t.Errorf("receiver balance was not updated correctly, want %v, got %v", value1, balance)
				}
			})
		}
	}
}

func TestProcessor_CallsWithInsufficientBalanceAreHandledCorrectly(t *testing.T) {
	for processorName, processor := range getProcessors() {
		for callName, call := range callTypesAndProperties() {
			t.Run(fmt.Sprintf("%s-%s", processorName, callName), func(t *testing.T) {
				sender0 := tosca.Address{1}
				receiver0 := tosca.Address{2}
				receiver1 := tosca.Address{3}
				checkValue := byte(55)
				value := tosca.NewValue(42)
				receiverBalance := tosca.NewValue(24)

				code0 := pushCallArguments(call, sufficientGas, value, receiver1)
				code0 = append(code0, []byte{
					byte(call.callType),
					byte(vm.PUSH1), byte(32),
					byte(vm.PUSH1), byte(0),
					byte(vm.RETURN),
				}...)

				code1 := []byte{
					byte(vm.PUSH1), checkValue,
					byte(vm.PUSH1), byte(0),
					byte(vm.MSTORE),
					byte(vm.PUSH1), byte(32),
					byte(vm.PUSH1), byte(0),
					byte(vm.RETURN),
				}

				state := WorldState{
					sender0:   Account{},
					receiver0: Account{Balance: receiverBalance, Code: code0},
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
					t.Fatalf("execution was not successful or failed with error %v", err)
				}

				// static call and delegate call do not transfer value,
				// therefore the transaction does not fail with insufficient balance
				if call.callType == vm.STATICCALL || call.callType == vm.DELEGATECALL {
					if !slices.Equal(result.Output, append(bytes.Repeat([]byte{0}, 31), checkValue)) {
						t.Errorf("transaction has not been handled correctly, code was not executed")
					}
				} else {
					if !slices.Equal(result.Output, bytes.Repeat([]byte{0}, 32)) {
						t.Errorf("transaction has not been handled correctly, code was executed")
					}
				}

				zero := tosca.NewValue(0)
				if balance := transactionContext.GetBalance(sender0); balance.Cmp(zero) != 0 {
					t.Errorf("sender balance was not updated correctly, want %v, got %v", zero, balance)
				}
				if balance := transactionContext.GetBalance(receiver0); balance.Cmp(receiverBalance) != 0 {
					t.Errorf("receiver balance should have not been updated, want %v, got %v", receiverBalance, balance)
				}
				if balance := transactionContext.GetBalance(receiver1); balance.Cmp(zero) != 0 {
					t.Errorf("receiver balance should have not been updated, want %v, got %v", zero, balance)
				}
			})
		}
	}
}

func TestProcessor_TransferToSelf(t *testing.T) {
	tests := map[string]struct {
		value   tosca.Value
		success bool
	}{
		"sufficient balance": {
			tosca.NewValue(100),
			true,
		},
		"insufficient balance": {
			tosca.NewValue(10000),
			false,
		},
	}
	for processorName, processor := range getProcessors() {
		for testName, test := range tests {
			t.Run(fmt.Sprintf("%s-%s", processorName, testName), func(t *testing.T) {
				sender := tosca.Address{1}
				senderBalance := tosca.NewValue(1000)

				state := WorldState{
					sender: Account{Balance: senderBalance},
				}
				transaction := tosca.Transaction{
					Sender:    sender,
					Recipient: &sender,
					GasLimit:  sufficientGas,
					Value:     test.value,
				}

				transactionContext := newScenarioContext(state)

				// Run the processor
				result, err := processor.Run(tosca.BlockParameters{}, transaction, transactionContext)
				if err != nil {
					t.Fatalf("execution failed with error %v", err)
				}
				if result.Success != test.success {
					t.Errorf("expected success flag to be %v, got %v", test.success, result.Success)
				}
				if balance := transactionContext.GetBalance(sender); balance.Cmp(senderBalance) != 0 {
					t.Errorf("sender balance should have not been updated, want %v, got %v", senderBalance, balance)
				}
			})
		}
	}
}
