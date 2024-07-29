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
	"testing"

	"github.com/Fantom-foundation/Tosca/go/tosca"
	"github.com/Fantom-foundation/Tosca/go/tosca/vm"
	op "github.com/ethereum/go-ethereum/core/vm"
	"go.uber.org/mock/gomock"

	_ "github.com/Fantom-foundation/Tosca/go/processor/opera" // < registers opera processor for testing
)

func TestProcessor_GasBillingEndToEnd(t *testing.T) {
	senderBalance := tosca.NewValue(1000000)
	gasLimit := tosca.Gas(100000)
	gasRefund := tosca.Gas(3000)
	gasPrice := tosca.NewValue(5)
	gasLeftSuccess := tosca.Gas(5000)

	tests := map[string]struct {
		result  tosca.Result
		gasUsed tosca.Gas
		success bool
	}{
		"success": {
			result: tosca.Result{
				GasLeft:   gasLeftSuccess,
				Success:   true,
				GasRefund: gasRefund,
			},
			gasUsed: gasLimit - (gasLeftSuccess - gasLeftSuccess/10 + gasRefund),
			success: true,
		},
		"failed": {
			result: tosca.Result{
				GasLeft:   0,
				Success:   false,
				GasRefund: gasRefund,
			},
			gasUsed: gasLimit,
			success: false,
		},
	}

	ctrl := gomock.NewController(t)
	interpreter := tosca.NewMockInterpreter(ctrl)

	sender := tosca.Address{1}
	recipient := tosca.Address{2}
	before := WorldState{
		sender: Account{Balance: senderBalance, Nonce: 4},
		recipient: Account{Balance: tosca.NewValue(0),
			Code: tosca.Code{
				byte(vm.PUSH1), byte(0), // < push 0
				byte(vm.PUSH1), byte(0), // < push 0
				byte(op.RETURN),
			},
		},
	}

	transaction := tosca.Transaction{
		Sender:    sender,
		Recipient: &recipient,
		GasLimit:  gasLimit,
		GasPrice:  gasPrice,
		Nonce:     4,
	}

	for name, test := range tests {
		for processorName, processor := range processorsWithInterpreter("mockInterpreter", interpreter) {
			t.Run(fmt.Sprintf("%s/%s", processorName, name), func(t *testing.T) {

				after := before.Clone()
				afterBalance := tosca.Sub(senderBalance, gasPrice.Scale(uint64(test.gasUsed)))
				after[sender] = Account{Balance: afterBalance, Nonce: after[sender].Nonce + 1}

				receipt := tosca.Receipt{
					Success: test.success,
					GasUsed: test.gasUsed,
				}

				scenario := Scenario{
					Before:      before,
					Transaction: transaction,
					After:       after,
					Receipt:     receipt,
				}

				interpreter.EXPECT().Run(gomock.Any()).Return(test.result, nil)
				scenario.Run(t, processor)
			})
		}
	}

}

func processorsWithInterpreter(name string, interpreter tosca.Interpreter) map[string]tosca.Processor {
	factories := tosca.GetAllRegisteredProcessorFactories()
	res := map[string]tosca.Processor{}
	for processorName, factory := range factories {
		processor := factory(interpreter)
		res[fmt.Sprintf("%s/%s", processorName, name)] = processor
	}

	return res
}
