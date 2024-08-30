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
	"errors"
	"regexp"
	"testing"

	"github.com/Fantom-foundation/Tosca/go/tosca"
	"github.com/holiman/uint256"
	"go.uber.org/mock/gomock"
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

func TestGas_AddressInAccessList(t *testing.T) {

	tests := map[string]struct {
		setup         func(*context)
		accessStatus  bool
		coldCost      tosca.Gas
		expectedError error
	}{
		"istanbul": {
			setup: func(c *context) {
				c.revision = tosca.R07_Istanbul
			},
			accessStatus: true,
		},
		"berlin_warm": {
			setup: func(c *context) {
				c.revision = tosca.R09_Berlin
				c.context.(*tosca.MockRunContext).EXPECT().IsAddressInAccessList(gomock.Any()).Return(true)
				c.stack.push(uint256.NewInt(1))
				c.stack.push(uint256.NewInt(2))
			},
			accessStatus: true,
			coldCost:     ColdAccountAccessCostEIP2929 - WarmStorageReadCostEIP2929,
		},
		"berlin_cold_enough_gas": {
			setup: func(c *context) {
				c.revision = tosca.R09_Berlin
				c.context.(*tosca.MockRunContext).EXPECT().IsAddressInAccessList(gomock.Any()).Return(false)
				c.context.(*tosca.MockRunContext).EXPECT().AccessAccount(gomock.Any()).Return(tosca.ColdAccess)
				c.stack.push(uint256.NewInt(1))
				c.stack.push(uint256.NewInt(2))
				c.gas = ColdAccountAccessCostEIP2929 - WarmStorageReadCostEIP2929
			},
			accessStatus: false,
			coldCost:     ColdAccountAccessCostEIP2929 - WarmStorageReadCostEIP2929,
		},
		"berlin_cold_not_enough_gas": {
			setup: func(c *context) {
				c.revision = tosca.R09_Berlin
				c.context.(*tosca.MockRunContext).EXPECT().IsAddressInAccessList(gomock.Any()).Return(false)
				c.context.(*tosca.MockRunContext).EXPECT().AccessAccount(gomock.Any()).Return(tosca.ColdAccess)
				c.stack.push(uint256.NewInt(1))
				c.stack.push(uint256.NewInt(2))
				c.gas = ColdAccountAccessCostEIP2929 - WarmStorageReadCostEIP2929 - 1
			},
			expectedError: errOutOfGas,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			runContext := tosca.NewMockRunContext(ctrl)

			ctxt := context{
				status:   statusRunning,
				params:   tosca.Parameters{},
				context:  runContext,
				stack:    NewStack(),
				memory:   NewMemory(),
				gas:      1 << 20,
				revision: tosca.R09_Berlin,
			}
			test.setup(&ctxt)

			warmGot, coldCostGot, errGot := addressInAccessList(&ctxt)

			if !errors.Is(errGot, test.expectedError) {
				t.Errorf("unexpected error, wanted %v, got %v", test.expectedError, errGot)
			}

			if warmGot != test.accessStatus {
				t.Errorf("unexpected warm access status, wanted %v, got %v", test.accessStatus, warmGot)
			}

			if coldCostGot != test.coldCost {
				t.Errorf("unexpected cold cost, wanted %d, got %d", test.coldCost, coldCostGot)
			}

		})
	}
}
