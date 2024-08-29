// Copyright (c) 2024 Fantom Foundation
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at fantom.foundation/bsl11.
//
// Change Date: 2028-4-16
//
// On the date above, in accordance with the Business Source License, use of
// this software will be governed by the GNU Lesser General Public License v3.

package lfvm

import (
	"regexp"
	"testing"

	"github.com/Fantom-foundation/Tosca/go/tosca"
	"github.com/holiman/uint256"
)

func TestGas_CallGasCalculation(t *testing.T) {
	tests := map[string]struct {
		available tosca.Gas    // < the gas available in the current context
		baseCosts tosca.Gas    // < the costs for setting up the call
		provided  *uint256.Int // < the gas to be provided to the nested call
		want      tosca.Gas    // < the gas costs for the call
	}{
		"available_is_more_than_provided": {
			available: tosca.Gas(200),
			baseCosts: tosca.Gas(20),
			provided:  uint256.NewInt(30),
			want:      30, // < limited by gas to be provided to nested call
		},
		"available_is_less_than_provided": {
			available: tosca.Gas(200),
			baseCosts: tosca.Gas(20),
			provided:  uint256.NewInt(300),
			want:      (200 - 20) - (200-20)/64, //  < limited by 63/64 of the available gas after the base costs
		},
		"available_is_less_than_provided_exceeding_maxUint64": {
			available: tosca.Gas(200),
			baseCosts: tosca.Gas(20),
			provided:  new(uint256.Int).Lsh(uint256.NewInt(1), 64),
			want:      (200 - 20) - (200-20)/64, //  < limited by 63/64 of the available gas after the base costs
		},
		"base_costs_higher_than_available": {
			available: tosca.Gas(20),
			baseCosts: tosca.Gas(200),
			provided:  uint256.NewInt(300),
			want:      200, //  < the base costs
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			got := callGas(test.available, test.baseCosts, test.provided)
			if want := test.want; want != got {
				t.Errorf("unexpected result, wanted %d, got %d", want, got)
			}
		})
	}
}

func TestGas_CheckGasCostsForAllExecutableOpcodes(t *testing.T) {
	validName := regexp.MustCompile(`^0x00[0-9A-Fa-f]{2}$`)
	for i := 0; i < int(NUM_OPCODES); i++ {
		op := OpCode(i)
		if static_gas_prices[i] == UNKNOWN_GAS_PRICE &&
			!validName.MatchString(op.String()) {
			t.Errorf("gas price for %v is unknown", OpCode(i))
		}
		if static_gas_prices_berlin[i] == UNKNOWN_GAS_PRICE &&
			!validName.MatchString(op.String()) {
			t.Errorf("berlin gas price for %v is unknown", OpCode(i))
		}
	}
}
