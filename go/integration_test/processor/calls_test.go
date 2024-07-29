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
	"fmt"
	"math/big"
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
