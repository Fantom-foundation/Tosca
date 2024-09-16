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
	"slices"
	"testing"

	"github.com/Fantom-foundation/Tosca/go/tosca"
	"github.com/Fantom-foundation/Tosca/go/tosca/vm"
	"github.com/ethereum/go-ethereum/common"
	op "github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
	"golang.org/x/exp/maps"

	_ "github.com/Fantom-foundation/Tosca/go/processor/floria" // < registers floria processor for testing
	_ "github.com/Fantom-foundation/Tosca/go/processor/opera"  // < registers opera processor for testing

	_ "github.com/Fantom-foundation/Tosca/go/interpreter/evmzero" // < registers evmzero interpreter for testing
	_ "github.com/Fantom-foundation/Tosca/go/interpreter/lfvm"    // < registers lfvm interpreter for testing
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
				{1}: Account{Balance: tosca.NewValue(100), Nonce: 4},
			},
			Transaction: tosca.Transaction{
				Sender:    tosca.Address{1},
				Recipient: &tosca.Address{2},
				GasLimit:  21_000,
				Value:     tosca.NewValue(3),
				Nonce:     4,
			},
			After: WorldState{
				{1}: Account{Balance: tosca.NewValue(97), Nonce: 5},
				{2}: Account{Balance: tosca.NewValue(3)},
			},
			Receipt: tosca.Receipt{
				Success: true,
				GasUsed: 21_000,
			},
		},
		"FailedValueTransfer": {
			Before: WorldState{
				{1}: Account{Balance: tosca.NewValue(10), Nonce: 4},
			},
			Transaction: tosca.Transaction{
				Sender:    tosca.Address{1},
				Recipient: &tosca.Address{2},
				GasLimit:  21_000,
				Value:     tosca.NewValue(20),
				Nonce:     4,
			},
			After: WorldState{
				{1}: Account{Balance: tosca.NewValue(10), Nonce: 5},
			},
			Receipt: tosca.Receipt{
				Success: false,
				GasUsed: 21_000,
			},
		},
		"SuccessfulContractCall": {
			Before: WorldState{
				{1}: Account{Balance: tosca.NewValue(100), Nonce: 4},
				{2}: Account{Balance: tosca.NewValue(0),
					Code: tosca.Code{
						byte(vm.PUSH1), byte(0), // < push 0
						byte(vm.PUSH1), byte(0), // < push 0
						byte(op.RETURN),
					},
				},
			},
			Transaction: tosca.Transaction{
				Sender:    tosca.Address{1},
				Recipient: &tosca.Address{2},
				GasLimit:  21_000 + 2*3, // < value transfer + 2 push instructions (return is free)
				Value:     tosca.NewValue(3),
				Nonce:     4,
			},
			After: WorldState{
				{1}: Account{Balance: tosca.NewValue(97), Nonce: 5},
				{2}: Account{Balance: tosca.NewValue(3),
					Code: tosca.Code{
						byte(vm.PUSH1), byte(0), // < push 0
						byte(vm.PUSH1), byte(0), // < push 0
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
				{1}: Account{Balance: tosca.NewValue(100), Nonce: 4},
				{2}: Account{Balance: tosca.NewValue(0),
					Code: tosca.Code{
						byte(vm.PUSH1), byte(0), // < push 0
						byte(vm.PUSH1), byte(0), // < push 0
						byte(op.REVERT),
					},
				},
			},
			Transaction: tosca.Transaction{
				Sender:    tosca.Address{1},
				Recipient: &tosca.Address{2},
				GasLimit:  21_000 + 2*3, // < value transfer + 2 push instructions (return is free)
				Value:     tosca.NewValue(3),
				Nonce:     4,
			},
			After: WorldState{
				{1}: Account{Balance: tosca.NewValue(100), Nonce: 5},
				{2}: Account{Balance: tosca.NewValue(0),
					Code: tosca.Code{
						byte(vm.PUSH1), byte(0), // < push 0
						byte(vm.PUSH1), byte(0), // < push 0
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
				{1}: Account{Balance: tosca.NewValue(100), Nonce: 4},
			},
			Transaction: tosca.Transaction{
				Sender:   tosca.Address{1},
				GasLimit: 53_000,
				Value:    tosca.NewValue(3),
				Nonce:    4,
			},
			After: WorldState{
				{1}: Account{Balance: tosca.NewValue(97), Nonce: 5},
				createdAddress: Account{
					Balance: tosca.NewValue(3),
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
		for interpreterName, interpreterFactory := range interpreter {
			interpreter, err := interpreterFactory(nil)
			if err != nil {
				panic(fmt.Sprintf("failed to load interpreter %s: %v", interpreterName, err))
			}
			processor := factory(interpreter)
			res[fmt.Sprintf("%s/%s", processorName, interpreterName)] = processor
		}
	}
	return res
}

func TestGetProcessors_ContainsMainConfigurations(t *testing.T) {
	// The main task of this job is to make sure that the essential processors
	// and interpreters are registered and available for testing.
	all := maps.Keys(getProcessors())
	wanted := []string{
		"opera/geth", "opera/lfvm", "opera/evmzero",
		"floria/geth", "floria/lfvm", "floria/evmzero",
	}
	for _, n := range wanted {
		if !slices.Contains(all, n) {
			t.Errorf("Configuration %q is not registered, got %v", n, all)
		}
	}
}
