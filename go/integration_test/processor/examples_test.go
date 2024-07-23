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

	"github.com/Fantom-foundation/Tosca/go/examples"
	"github.com/Fantom-foundation/Tosca/go/tosca"
)

var (
	processorExamples = []examples.Example{
		examples.GetIncrementExample(),
		examples.GetFibExample(),
		examples.GetSha3Example(),
		examples.GetArithmeticExample(),
		examples.GetMemoryExample(),
		examples.GetJumpdestAnalysisExample(),
		examples.GetStopAnalysisExample(),
		examples.GetPush1AnalysisExample(),
		examples.GetPush32AnalysisExample(),
	}
)

func TestProcessor_Examples(t *testing.T) {
	for _, example := range processorExamples {
		for processorName, processor := range getProcessors() {
			for i := 0; i < 10; i++ {
				t.Run(fmt.Sprintf("%s-%s-%d", example.Name, processorName, i), func(t *testing.T) {
					want := example.RunReference(i)
					scenario := getScenarioContext(example)
					transactionContext := newScenarioContext(scenario.Before)

					got, err := example.RunOnProcessor(processor, i, scenario.Transaction, transactionContext)
					if err != nil {
						t.Fatalf("error processing contract: %v", err)
					}
					if want != got.Result {
						t.Fatalf("incorrect result, wanted %d, got %d", want, got.Result)
					}
				})
			}
		}
	}
}

func getScenarioContext(example examples.Example) Scenario {
	scenario := Scenario{
		Before: WorldState{
			{1}: Account{},
			{2}: Account{Code: example.Code},
		},
		Transaction: tosca.Transaction{
			Sender:    tosca.Address{1},
			Recipient: &tosca.Address{2},
			GasLimit:  1000000,
		},
		After: WorldState{
			{1}: Account{},
			{2}: Account{Code: example.Code},
		},
	}

	return scenario
}
