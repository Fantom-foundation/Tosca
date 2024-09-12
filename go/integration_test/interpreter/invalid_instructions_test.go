// Copyright (c) 2024 Fantom Foundation
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at fantom.foundation/bsl11.
//
// Change Date: 2028-4-16
//
// On the date above, in accordance with the Business Source License, use of
// this software will be governed by the GNU Lesser General Public License v3.

package interpreter_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/Fantom-foundation/Tosca/go/tosca/vm"
)

func TestInterpreterDetectsInvalidInstruction(t *testing.T) {
	for _, rev := range revisions {
		for _, variant := range getAllInterpreterVariantsForTests() {
			evm := GetCleanEVM(rev, variant, nil)
			// LFVM currently does not support detection of invalid codes!
			// TODO: fix this
			if strings.Contains(variant, "lfvm") {
				continue
			}
			instructions := getInstructions(rev)
			for i := 0; i < 256; i++ {
				op := vm.OpCode(i)
				_, exits := instructions[op]
				if exits {
					continue
				}
				t.Run(fmt.Sprintf("%s-%s-%s", variant, rev, op), func(t *testing.T) {
					code := []byte{byte(op), byte(vm.STOP)}
					input := []byte{}

					result, err := evm.Run(code, input)
					if err != nil {
						t.Fatalf("unexpected failure of EVM execution: %v", err)
					}
					if result.Success {
						t.Errorf("expected execution to fail, but got %v", result)
					}
				})
			}
		}
	}
}
