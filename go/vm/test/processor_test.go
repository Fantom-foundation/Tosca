package vm_test

import (
	"testing"

	"github.com/Fantom-foundation/Tosca/go/vm"
	"github.com/Fantom-foundation/Tosca/go/vm/geth"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"go.uber.org/mock/gomock"

	// This is only imported to get the EVM opcode definitions.
	// TODO: write up our own op-code definition and remove this dependency.
	op "github.com/ethereum/go-ethereum/core/vm"
)

func getProcessors() map[string]vm.Processor {
	return map[string]vm.Processor{
		"geth": geth.NewProcessor(),
	}
}

func TestProcessor_SimpleValueTransfer(t *testing.T) {
	const transferCosts = vm.Gas(21_000)

	// Transfer 3*2^(31*8) tokens from account 1 to account 2.
	transaction := vm.Transaction{
		Sender:    vm.Address{1},
		Recipient: &vm.Address{2},
		Value:     vm.Value{3}, // < TODO: need a better value type!
		Nonce:     4,
		GasLimit:  transferCosts,
	}

	for name, processor := range getProcessors() {
		t.Run(name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			blockInfo := vm.BlockInfo{}
			txContext := vm.TransactionContext{}

			// TODO: clean up expectations
			// TODO: provide a better way to define expectations
			// - use a before/after pattern
			state := vm.NewMockWorldState(ctrl)

			state.EXPECT().GetBalance(vm.Address{1}).Return(vm.Value{10}).AnyTimes()
			state.EXPECT().GetBalance(vm.Address{2}).Return(vm.Value{5}).AnyTimes()

			state.EXPECT().AccountExists(vm.Address{2}).Return(true)
			state.EXPECT().GetCode(vm.Address{2}).Return([]byte{})
			state.EXPECT().GetNonce(vm.Address{1}).Return(uint64(4)).Times(2)
			state.EXPECT().SetNonce(vm.Address{1}, uint64(5)).Return()
			state.EXPECT().GetCodeHash(vm.Address{1}).Return(vm.Hash{})

			gomock.InOrder(
				state.EXPECT().SetBalance(vm.Address{1}, vm.Value{10}), // < charging gas, but price is zero
				state.EXPECT().SetBalance(vm.Address{1}, vm.Value{7}),  // < withdraw 3 tokens
				state.EXPECT().SetBalance(vm.Address{2}, vm.Value{8}),  // < deposit 3 tokens
			)

			state.EXPECT().GetTransactionContext().Return(txContext)

			// Execute the transaction.
			receipt, err := processor.Run(blockInfo, transaction, state)
			if err != nil {
				t.Errorf("error: %v", err)
			}

			// Check the result.
			if got, want := receipt.Success, true; got != want {
				t.Errorf("unexpected success: got %v, want %v", got, want)
			}
			if want, got := transferCosts, receipt.GasUsed; want != got {
				t.Errorf("unexpected gas costs: want %d, got %d", want, got)
			}
		})
	}
}

func TestProcessor_ContractCallThatSucceeds(t *testing.T) {
	const gasCosts = vm.Gas(21_000 + 2*3)

	// Transfer 3*2^(31*8) tokens from account 1 to account 2.
	transaction := vm.Transaction{
		Sender:    vm.Address{1},
		Recipient: &vm.Address{2},
		Nonce:     4,
		GasLimit:  gasCosts,
	}

	for name, processor := range getProcessors() {
		t.Run(name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			blockInfo := vm.BlockInfo{}
			txContext := vm.TransactionContext{}

			state := vm.NewMockWorldState(ctrl)

			state.EXPECT().GetBalance(vm.Address{1}).Return(vm.Value{10}).AnyTimes()
			state.EXPECT().GetBalance(vm.Address{2}).Return(vm.Value{5}).AnyTimes()

			state.EXPECT().GetNonce(vm.Address{1}).Return(uint64(4)).Times(2)
			state.EXPECT().SetNonce(vm.Address{1}, uint64(5)).Return()
			state.EXPECT().GetCodeHash(vm.Address{1}).Return(vm.Hash{})

			state.EXPECT().AccountExists(vm.Address{2}).Return(true)

			code := []byte{
				byte(op.PUSH1), byte(0), // < push 0
				byte(op.PUSH1), byte(0), // < push 0
				byte(op.RETURN),
			}
			state.EXPECT().GetCode(vm.Address{2}).Return(code)
			state.EXPECT().GetCodeHash(vm.Address{2}).Return(keccak256Hash(code))

			state.EXPECT().SetBalance(vm.Address{1}, vm.Value{10}) // < charging gas, but price is zero

			state.EXPECT().GetTransactionContext().Return(txContext)

			// Execute the transaction.
			receipt, err := processor.Run(blockInfo, transaction, state)
			if err != nil {
				t.Errorf("error: %v", err)
			}

			// Check the result.
			if got, want := receipt.Success, true; got != want {
				t.Errorf("unexpected success: got %v, want %v", got, want)
			}
			if want, got := gasCosts, receipt.GasUsed; want != got {
				t.Errorf("unexpected gas costs: want %d, got %d", want, got)
			}
		})
	}
}

func TestProcessor_ContractCallThatReverts(t *testing.T) {
	const gasCosts = vm.Gas(21_000 + 2*3)

	// Transfer 3*2^(31*8) tokens from account 1 to account 2.
	transaction := vm.Transaction{
		Sender:    vm.Address{1},
		Recipient: &vm.Address{2},
		Nonce:     4,
		GasLimit:  gasCosts,
	}

	for name, processor := range getProcessors() {
		t.Run(name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			blockInfo := vm.BlockInfo{}
			txContext := vm.TransactionContext{}

			state := vm.NewMockWorldState(ctrl)

			state.EXPECT().GetBalance(vm.Address{1}).Return(vm.Value{10}).AnyTimes()
			state.EXPECT().GetBalance(vm.Address{2}).Return(vm.Value{5}).AnyTimes()

			state.EXPECT().GetNonce(vm.Address{1}).Return(uint64(4)).Times(2)
			state.EXPECT().SetNonce(vm.Address{1}, uint64(5)).Return()

			state.EXPECT().GetCodeHash(vm.Address{1}).Return(vm.Hash{})

			state.EXPECT().AccountExists(vm.Address{2}).Return(true)

			code := []byte{
				byte(op.PUSH1), byte(0), // < push 0
				byte(op.PUSH1), byte(0), // < push 0
				byte(op.REVERT),
			}
			state.EXPECT().GetCode(vm.Address{2}).Return(code)
			state.EXPECT().GetCodeHash(vm.Address{2}).Return(keccak256Hash(code))

			state.EXPECT().SetBalance(vm.Address{1}, vm.Value{10}) // < charging gas, but price is zero

			state.EXPECT().GetTransactionContext().Return(txContext)

			// Execute the transaction.
			receipt, err := processor.Run(blockInfo, transaction, state)
			if err != nil {
				t.Errorf("error: %v", err)
			}

			// Check the result.
			if got, want := receipt.Success, false; got != want {
				t.Errorf("unexpected success: got %v, want %v", got, want)
			}

			if want, got := gasCosts, receipt.GasUsed; want != got {
				t.Errorf("unexpected gas costs: want %d, got %d", want, got)
			}
		})
	}
}

func TestProcessor_ContractCreation(t *testing.T) {
	const gasCosts = vm.Gas(53_000)

	// Transfer 3*2^(31*8) tokens from account 1 to account 2.
	transaction := vm.Transaction{
		Sender:   vm.Address{1},
		Nonce:    4,
		GasLimit: gasCosts,
	}

	// The new contract address is derived from the sender's address and the nonce.
	newContractAddress := vm.Address(crypto.CreateAddress(common.Address{1}, 4))

	for name, processor := range getProcessors() {
		t.Run(name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			blockInfo := vm.BlockInfo{}
			txContext := vm.TransactionContext{}

			state := vm.NewMockWorldState(ctrl)

			state.EXPECT().GetTransactionContext().Return(txContext)

			state.EXPECT().GetBalance(vm.Address{1}).Return(vm.Value{10}).AnyTimes()
			state.EXPECT().SetBalance(vm.Address{1}, vm.Value{10}) // < charging gas, but price is zero

			state.EXPECT().GetNonce(vm.Address{1}).Return(uint64(4)).Times(3)
			state.EXPECT().SetNonce(vm.Address{1}, uint64(5)).Return()

			//state.EXPECT().AccountExists(newContractAddress).Return(false)
			state.EXPECT().GetNonce(newContractAddress).Return(uint64(0))
			state.EXPECT().SetNonce(newContractAddress, uint64(1)).Return()

			state.EXPECT().GetCodeHash(vm.Address{1}).Return(vm.Hash{})
			state.EXPECT().GetCodeHash(newContractAddress).Return(vm.Hash{})

			state.EXPECT().SetCode(newContractAddress, gomock.Any()).Do(func(address vm.Address, code []byte) {
				if len(code) != 0 {
					t.Fatalf("unexpected code: %x", code)
				}
			})

			// Execute the transaction.
			receipt, err := processor.Run(blockInfo, transaction, state)
			if err != nil {
				t.Errorf("error: %v", err)
			}

			// Check the result.
			if got, want := receipt.Success, true; got != want {
				t.Errorf("unexpected success: got %v, want %v", got, want)
			}

			if want, got := gasCosts, receipt.GasUsed; want != got {
				t.Errorf("unexpected gas costs: want %d, got %d", want, got)
			}
			if receipt.ContractAddress == nil {
				t.Fatalf("created contract address not set in result")
			}
			if *receipt.ContractAddress != newContractAddress {
				t.Errorf("unexpected result for created contract address, wanted %v, got %v", newContractAddress, *receipt.ContractAddress)
			}
		})
	}
}

// TODO:
// - test gas price charging
// - test gas refunding
// - test left-over gas refunding
// - test recursive calls
// - test roll-back on revert
// - improve test setup
// - find better place for those tests
