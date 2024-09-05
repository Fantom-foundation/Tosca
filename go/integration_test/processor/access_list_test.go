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
	"slices"
	"testing"

	"github.com/Fantom-foundation/Tosca/go/tosca"
	"github.com/Fantom-foundation/Tosca/go/tosca/vm"
)

func TestProcessor_AccessListIsHandledCorrectly(t *testing.T) {
	sender := tosca.Address{1}
	receiver := &tosca.Address{2}
	checkValue := byte(0x42)
	gas := tosca.Gas(1000000)
	accessedKey := tosca.Key{0x55}

	tests := map[string]struct {
		accessList      []tosca.AccessTuple
		expectedGasUsed tosca.Gas
	}{
		"empty_access_list": {
			accessList:      []tosca.AccessTuple{},
			expectedGasUsed: tosca.Gas(140704),
		},
		"account_access": {
			accessList: []tosca.AccessTuple{
				{
					Address: *receiver,
					Keys:    nil,
				},
			},
			expectedGasUsed: tosca.Gas(142864),
		},
		"storage_access": {
			accessList: []tosca.AccessTuple{
				{
					Address: *receiver,
					Keys:    []tosca.Key{accessedKey},
				},
			},
			expectedGasUsed: tosca.Gas(144574),
		},
	}

	for processorName, processor := range getProcessors() {
		for testName, test := range tests {
			t.Run(processorName+"/"+testName, func(t *testing.T) {

				code := []byte{
					byte(vm.PUSH1), checkValue,
					byte(vm.PUSH32),
				}
				code = append(code, accessedKey[:]...)
				code = append(code, []byte{
					byte(vm.SSTORE),
					byte(vm.PUSH32),
				}...)
				code = append(code, accessedKey[:]...)
				code = append(code, []byte{
					byte(vm.SLOAD),
					byte(vm.PUSH1), checkValue,
					byte(vm.PUSH1), byte(0),
					byte(vm.MSTORE),
					byte(vm.PUSH1), byte(32),
					byte(vm.PUSH1), byte(0),
					byte(vm.RETURN),
				}...)

				blockParams := tosca.BlockParameters{Revision: tosca.R09_Berlin}

				transaction := tosca.Transaction{
					Sender:     sender,
					Recipient:  receiver,
					GasLimit:   gas,
					Nonce:      0,
					AccessList: test.accessList,
				}
				scenario := getScenarioContext(sender, *receiver, code, gas)
				transactionContext := newScenarioContext(scenario.Before)

				// Run the processor
				result, err := processor.Run(blockParams, transaction, transactionContext)
				if err != nil || !result.Success {
					t.Errorf("execution failed with error: %v and success %v", err, result.Success)
				}
				if result.GasUsed != test.expectedGasUsed {
					t.Errorf("expected gas used %v, got %v", test.expectedGasUsed, result.GasUsed)
				}
				if !slices.Equal(result.Output, append(bytes.Repeat([]byte{0}, 31), checkValue)) {
					t.Errorf("value was not stored and loaded correctly, got %v", result.Output)
				}
			})
		}
	}
}
