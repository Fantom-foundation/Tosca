package processor

import (
	"fmt"
	"testing"

	"github.com/Fantom-foundation/Tosca/go/tosca"
	"github.com/ethereum/go-ethereum/common"
	op "github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"

	_ "github.com/Fantom-foundation/Tosca/go/processor/opera" // < registers opera processor for testing
)

// This file contains a few initial shake-down tests or a Processor implementation.
// Right now, the tested features are minimal. Follow-up work is needed to systematically
// establish a set of test cases for Processor features.
//
// TODO:
// - test gas price charging
// - test gas refunding
// - test left-over gas refunding
// - test recursive calls
// - test roll-back on revert

func getScenarios() map[string]Scenario {

	// TODO: improve organization of test scenarios

	createdAddress := tosca.Address(crypto.CreateAddress(common.Address{1}, 4))
	return map[string]Scenario{
		"SuccessfulValueTransfer": {
			Before: WorldState{
				{1}: Account{Balance: tosca.ValueFromUint64(100), Nonce: 4},
			},
			Transaction: tosca.Transaction{
				Sender:    tosca.Address{1},
				Recipient: &tosca.Address{2},
				GasLimit:  21_000,
				Value:     tosca.ValueFromUint64(3),
				Nonce:     4,
			},
			After: WorldState{
				{1}: Account{Balance: tosca.ValueFromUint64(97), Nonce: 5},
				{2}: Account{Balance: tosca.ValueFromUint64(3)},
			},
			Receipt: tosca.Receipt{
				Success: true,
				GasUsed: 21_000,
			},
		},
		"FailedValueTransfer": {
			Before: WorldState{
				{1}: Account{Balance: tosca.ValueFromUint64(10), Nonce: 4},
			},
			Transaction: tosca.Transaction{
				Sender:    tosca.Address{1},
				Recipient: &tosca.Address{2},
				GasLimit:  21_000,
				Value:     tosca.ValueFromUint64(20),
				Nonce:     4,
			},
			After: WorldState{
				{1}: Account{Balance: tosca.ValueFromUint64(10), Nonce: 5},
			},
			Receipt: tosca.Receipt{
				Success: false,
				GasUsed: 21_000,
			},
		},
		"SuccessfulContractCall": {
			Before: WorldState{
				{1}: Account{Balance: tosca.ValueFromUint64(100), Nonce: 4},
				{2}: Account{Balance: tosca.ValueFromUint64(0),
					Code: tosca.Code{
						byte(op.PUSH1), byte(0), // < push 0
						byte(op.PUSH1), byte(0), // < push 0
						byte(op.RETURN),
					},
				},
			},
			Transaction: tosca.Transaction{
				Sender:    tosca.Address{1},
				Recipient: &tosca.Address{2},
				GasLimit:  21_000 + 2*3, // < value transfer + 2 push instructions (return is free)
				Value:     tosca.ValueFromUint64(3),
				Nonce:     4,
			},
			After: WorldState{
				{1}: Account{Balance: tosca.ValueFromUint64(97), Nonce: 5},
				{2}: Account{Balance: tosca.ValueFromUint64(3),
					Code: tosca.Code{
						byte(op.PUSH1), byte(0), // < push 0
						byte(op.PUSH1), byte(0), // < push 0
						byte(op.RETURN),
					},
				},
			},
			Receipt: tosca.Receipt{
				Success: true,
				GasUsed: 21_000 + 2*3,
			},
		},
		"RevertingContractCall": {
			Before: WorldState{
				{1}: Account{Balance: tosca.ValueFromUint64(100), Nonce: 4},
				{2}: Account{Balance: tosca.ValueFromUint64(0),
					Code: tosca.Code{
						byte(op.PUSH1), byte(0), // < push 0
						byte(op.PUSH1), byte(0), // < push 0
						byte(op.REVERT),
					},
				},
			},
			Transaction: tosca.Transaction{
				Sender:    tosca.Address{1},
				Recipient: &tosca.Address{2},
				GasLimit:  21_000 + 2*3, // < value transfer + 2 push instructions (return is free)
				Value:     tosca.ValueFromUint64(3),
				Nonce:     4,
			},
			After: WorldState{
				{1}: Account{Balance: tosca.ValueFromUint64(100), Nonce: 5},
				{2}: Account{Balance: tosca.ValueFromUint64(0),
					Code: tosca.Code{
						byte(op.PUSH1), byte(0), // < push 0
						byte(op.PUSH1), byte(0), // < push 0
						byte(op.REVERT),
					},
				},
			},
			Receipt: tosca.Receipt{
				Success: false,
				GasUsed: 21_000 + 2*3,
			},
		},

		"SuccessfulContractCreation": {
			Before: WorldState{
				{1}: Account{Balance: tosca.ValueFromUint64(100), Nonce: 4},
			},
			Transaction: tosca.Transaction{
				Sender:   tosca.Address{1},
				GasLimit: 53_000,
				Value:    tosca.ValueFromUint64(3),
				Nonce:    4,
			},
			After: WorldState{
				{1}: Account{Balance: tosca.ValueFromUint64(97), Nonce: 5},
				createdAddress: Account{
					Balance: tosca.ValueFromUint64(3),
					Nonce:   1,
					Code:    tosca.Code{},
				},
			},
			Receipt: tosca.Receipt{
				Success:         true,
				GasUsed:         53_000,
				ContractAddress: &createdAddress,
			},
		},
	}
}

func RunProcessorTests(t *testing.T, processor tosca.Processor) {
	for name, s := range getScenarios() {
		t.Run(name, func(t *testing.T) {
			s.Run(t, processor)
		})
	}
}

func TestProcessor_Scenarios(t *testing.T) {
	for name, processor := range getProcessors() {
		t.Run(name, func(t *testing.T) {
			RunProcessorTests(t, processor)
		})
	}
}

// getProcessors returns a map containing all registered processors instantiated
// with all registered interpreters.
func getProcessors() map[string]tosca.Processor {
	interpreter := tosca.GetAllRegisteredInterpreters()
	factories := tosca.GetAllRegisteredProcessorFactories()

	res := map[string]tosca.Processor{}
	for processorName, factory := range factories {
		for interpreterName, interpreter := range interpreter {
			processor := factory(interpreter)
			res[fmt.Sprintf("%s/%s", processorName, interpreterName)] = processor
		}
	}
	return res
}
