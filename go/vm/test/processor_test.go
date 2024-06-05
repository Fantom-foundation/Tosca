package vm_test

import (
	"testing"

	"github.com/Fantom-foundation/Tosca/go/vm"
	"github.com/Fantom-foundation/Tosca/go/vm/geth"
	"go.uber.org/mock/gomock"
)

func getProcessors() map[string]vm.Processor {
	return map[string]vm.Processor{
		"geth": geth.NewProcessor(),
	}
}

func TestProcessor_SimpleValueTransfer(t *testing.T) {

	// Transfer 3*2^(31*8) tokens from account 1 to account 2.
	transaction := vm.Transaction{
		Sender:    vm.Address{1},
		Recipient: &vm.Address{2},
		Value:     vm.Value{3}, // < TODO: need a better value type!
		Nonce:     4,
		GasLimit:  21_000, // < the transfer costs 21_000 gas
	}

	for name, processor := range getProcessors() {
		t.Run(name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			txContext := vm.TransactionContext{}

			runContext := vm.NewMockTxContext(ctrl)

			runContext.EXPECT().GetBalance(vm.Address{1}).Return(vm.Value{10}).AnyTimes()
			runContext.EXPECT().GetBalance(vm.Address{2}).Return(vm.Value{5}).AnyTimes()

			runContext.EXPECT().AccountExists(vm.Address{2}).Return(true)
			runContext.EXPECT().GetCode(vm.Address{2}).Return([]byte{})
			runContext.EXPECT().GetNonce(vm.Address{1}).Return(uint64(4))
			runContext.EXPECT().GetCodeHash(vm.Address{1}).Return(vm.Hash{})

			gomock.InOrder(
				runContext.EXPECT().SetBalance(vm.Address{1}, vm.Value{10}), // < charging gas, but price is zero
				runContext.EXPECT().SetBalance(vm.Address{1}, vm.Value{7}),  // < withdraw 3 tokens
				runContext.EXPECT().SetBalance(vm.Address{2}, vm.Value{8}),  // < deposit 3 tokens
			)

			runContext.EXPECT().GetTransactionContext().Return(txContext)

			// Execute the transaction.
			receipt, err := processor.Run(vm.R07_Istanbul, transaction, txContext, runContext)
			if err != nil {
				t.Errorf("error: %v", err)
			}

			// Check the result.
			if !receipt.Success {
				t.Errorf("transaction failed")
			}
		})
	}

}
