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

func TestGas_AddressInAccessList(t *testing.T) {

	tests := map[string]struct {
		setup        func(*context)
		accessStatus bool
		coldCost     tosca.Gas
	}{
		"istanbul": {
			setup: func(c *context) {
				c.params.Revision = tosca.R07_Istanbul
			},
			accessStatus: true,
		},
		"berlin_warm": {
			setup: func(c *context) {
				c.params.Revision = tosca.R09_Berlin
				c.context.(*tosca.MockRunContext).EXPECT().IsAddressInAccessList(gomock.Any()).Return(true)
				c.stack.push(uint256.NewInt(1)) // value
				c.stack.push(uint256.NewInt(2)) // key
			},
			accessStatus: true,
			coldCost:     ColdAccountAccessCostEIP2929 - WarmStorageReadCostEIP2929,
		},
		"berlin_cold_enough_gas": {
			setup: func(c *context) {
				c.params.Revision = tosca.R09_Berlin
				c.context.(*tosca.MockRunContext).EXPECT().IsAddressInAccessList(gomock.Any()).Return(false)
				c.context.(*tosca.MockRunContext).EXPECT().AccessAccount(gomock.Any()).Return(tosca.ColdAccess)
				c.stack.push(uint256.NewInt(1)) // value
				c.stack.push(uint256.NewInt(2)) // key
				c.gas = ColdAccountAccessCostEIP2929 - WarmStorageReadCostEIP2929
			},
			accessStatus: false,
			coldCost:     ColdAccountAccessCostEIP2929 - WarmStorageReadCostEIP2929,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			runContext := tosca.NewMockRunContext(ctrl)

			ctxt := context{
				status:  statusRunning,
				params:  tosca.Parameters{},
				context: runContext,
				stack:   NewStack(),
				memory:  NewMemory(),
				gas:     1 << 20,
			}
			test.setup(&ctxt)

			warmGot, coldCostGot, errGot := addressInAccessList(&ctxt)

			if errGot != nil {
				t.Errorf("unexpected error, wanted %v, got %v", nil, errGot)
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

func TestGas_AddressInAccessListReportsOutOfGas(t *testing.T) {

	ctrl := gomock.NewController(t)
	runContext := tosca.NewMockRunContext(ctrl)
	ctxt := context{
		status: statusRunning,
		params: tosca.Parameters{
			BlockParameters: tosca.BlockParameters{
				Revision: tosca.R09_Berlin,
			},
		},
		context: runContext,
		stack:   NewStack(),
		memory:  NewMemory(),
		gas:     ColdAccountAccessCostEIP2929 - WarmStorageReadCostEIP2929 - 1,
	}

	ctxt.context.(*tosca.MockRunContext).EXPECT().IsAddressInAccessList(gomock.Any()).Return(false)
	ctxt.context.(*tosca.MockRunContext).EXPECT().AccessAccount(gomock.Any()).Return(tosca.ColdAccess)
	ctxt.stack.push(uint256.NewInt(1)) // value
	ctxt.stack.push(uint256.NewInt(2)) // key

	_, _, errGot := addressInAccessList(&ctxt)
	if !errors.Is(errGot, errOutOfGas) {
		t.Errorf("unexpected error, wanted %v, got %v", errOutOfGas, errGot)
	}
}

func TestGas_gasSStoreEIP2200_Successful(t *testing.T) {
	tests := map[string]struct {
		setup  func(*context)
		gas    tosca.Gas
		refund tosca.Gas
	}{
		"noop": {
			setup: func(c *context) {
				c.context.(*tosca.MockRunContext).EXPECT().
					GetStorage(gomock.Any(), gomock.Any()).Return(tosca.Word{}) // current value
				c.stack.push(uint256.NewInt(0)) // value
				c.stack.push(uint256.NewInt(0)) // key
			},
			gas: SloadGasEIP2200,
		},
		"create slot": {
			setup: func(c *context) {
				c.context.(*tosca.MockRunContext).EXPECT().
					GetCommittedStorage(gomock.Any(), gomock.Any()).Return(tosca.Word{}) // original value
				c.context.(*tosca.MockRunContext).EXPECT().
					GetStorage(gomock.Any(), gomock.Any()).Return(tosca.Word{}) // current value
				c.stack.push(uint256.NewInt(1)) // value
				c.stack.push(uint256.NewInt(1)) // key

			},
			gas: SstoreSetGasEIP2200,
		},
		"delete slot current same as original": {
			setup: func(c *context) {
				c.context.(*tosca.MockRunContext).EXPECT().
					GetCommittedStorage(gomock.Any(), gomock.Any()).Return(tosca.Word{1}) // original value
				c.context.(*tosca.MockRunContext).EXPECT().
					GetStorage(gomock.Any(), gomock.Any()).Return(tosca.Word{1}) // current value
				c.stack.push(uint256.NewInt(0)) // value
				c.stack.push(uint256.NewInt(0)) // key
			},
			gas:    SstoreResetGasEIP2200,
			refund: SstoreClearsScheduleRefundEIP2200,
		},
		"write existing slot": {
			setup: func(c *context) {
				c.context.(*tosca.MockRunContext).EXPECT().
					GetCommittedStorage(gomock.Any(), gomock.Any()).Return(tosca.Word{1}) // original value
				c.context.(*tosca.MockRunContext).EXPECT().
					GetStorage(gomock.Any(), gomock.Any()).Return(tosca.Word{1}) // current value
				c.stack.push(uint256.NewInt(1)) // value
				c.stack.push(uint256.NewInt(0)) // key

			},
			gas: SstoreResetGasEIP2200,
		},
		"recreate slot": {
			setup: func(c *context) {
				c.context.(*tosca.MockRunContext).EXPECT().
					GetCommittedStorage(gomock.Any(), gomock.Any()).Return(tosca.Word{1}) // original value
				c.context.(*tosca.MockRunContext).EXPECT().
					GetStorage(gomock.Any(), gomock.Any()).Return(tosca.Word{0}) // current value
				c.stack.push(uint256.NewInt(1)) // value
				c.stack.push(uint256.NewInt(0)) // key
			},
			gas:    SloadGasEIP2200,
			refund: -SstoreClearsScheduleRefundEIP2200,
		},
		"delete slot different current and original": {
			setup: func(c *context) {
				c.context.(*tosca.MockRunContext).EXPECT().
					GetCommittedStorage(gomock.Any(), gomock.Any()).Return(tosca.Word{1}) // original value
				c.context.(*tosca.MockRunContext).EXPECT().
					GetStorage(gomock.Any(), gomock.Any()).Return(tosca.Word{2}) // current value
				c.stack.push(uint256.NewInt(0)) // value
				c.stack.push(uint256.NewInt(0)) // key
			},
			gas:    SloadGasEIP2200,
			refund: SstoreClearsScheduleRefundEIP2200,
		},
		"reset to original inexistent slot": {
			setup: func(c *context) {
				c.context.(*tosca.MockRunContext).EXPECT().
					GetCommittedStorage(gomock.Any(), gomock.Any()).Return(tosca.Word{0}) // original value
				c.context.(*tosca.MockRunContext).EXPECT().
					GetStorage(gomock.Any(), gomock.Any()).Return(tosca.Word{2}) // current value
				c.stack.push(uint256.NewInt(0)) // value
				c.stack.push(uint256.NewInt(0)) // key
			},
			gas:    SloadGasEIP2200,
			refund: SstoreSetGasEIP2200 - SloadGasEIP2200,
		},
		"reset to original existent slot": {
			setup: func(c *context) {
				c.context.(*tosca.MockRunContext).EXPECT().
					GetCommittedStorage(gomock.Any(), gomock.Any()).Return(
					tosca.Word(uint256.NewInt(1).Bytes32())) // original value
				c.context.(*tosca.MockRunContext).EXPECT().
					GetStorage(gomock.Any(), gomock.Any()).Return(tosca.Word{2}) // current value
				c.stack.push(uint256.NewInt(1)) // value
				c.stack.push(uint256.NewInt(0)) // key
			},
			gas:    SloadGasEIP2200,
			refund: SstoreResetGasEIP2200 - SloadGasEIP2200,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			runContext := tosca.NewMockRunContext(ctrl)

			ctxt := context{
				status:  statusRunning,
				stack:   NewStack(),
				memory:  NewMemory(),
				context: runContext,
			}
			ctxt.gas = SstoreSentryGasEIP2200 + 1
			test.setup(&ctxt)

			gas, err := gasSStore(&ctxt)

			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			if gas != test.gas {
				t.Errorf("unexpected gas cost, wanted %v, got %v", test.gas, gas)
			}
			if ctxt.refund != test.refund {
				t.Errorf("unexpected refund, wanted %v, got %v", test.refund, ctxt.refund)
			}

		})
	}
}

func TestGas_gasSStoreEIP2200_NotEnoughGas(t *testing.T) {
	ctrl := gomock.NewController(t)
	runContext := tosca.NewMockRunContext(ctrl)

	ctxt := context{
		status:  statusRunning,
		stack:   NewStack(),
		memory:  NewMemory(),
		context: runContext,
		gas:     SstoreSentryGasEIP2200,
	}

	// Prepare stack arguments.
	ctxt.stack.push(uint256.NewInt(1)) // value
	ctxt.stack.push(uint256.NewInt(1)) // key

	gas, err := gasSStore(&ctxt)

	if err != errNotEnoughGasReentrancy {
		t.Errorf("unexpected error: %v", err)
	}

	if gas != 0 {
		t.Errorf("unexpected gas cost, wanted %v, got %v", 0, gas)
	}
}

func TestGas_gasSStoreEIP2929_ReportsErrors(t *testing.T) {

	tests := map[string]struct {
		setup         func(*context)
		expectedError error
	}{
		"london slot and address absent": {
			setup: func(c *context) {
				c.context.(*tosca.MockRunContext).EXPECT().
					GetStorage(gomock.Any(), gomock.Any()).Return(tosca.Word{})
				c.context.(*tosca.MockRunContext).EXPECT().
					IsSlotInAccessList(gomock.Any(), gomock.Any()).Return(false, false)
				c.stack.push(uint256.NewInt(0)) // value
				c.stack.push(uint256.NewInt(0)) // key
				c.params.Revision = tosca.R10_London
			},
			expectedError: errAddressNotFoundInSstore,
		},
		"berlin not enough gas": {
			setup: func(c *context) {
				c.stack.push(uint256.NewInt(1)) // value
				c.stack.push(uint256.NewInt(1)) // key
				c.params.Revision = tosca.R09_Berlin
				c.gas = SstoreSentryGasEIP2200
			},
			expectedError: errNotEnoughGasReentrancy,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			runContext := tosca.NewMockRunContext(ctrl)

			ctxt := context{
				status:  statusRunning,
				stack:   NewStack(),
				memory:  NewMemory(),
				context: runContext,
				gas:     1 << 20,
			}
			test.setup(&ctxt)

			_, err := gasSStoreEIP2929(&ctxt)
			if !errors.Is(err, test.expectedError) {
				t.Errorf("unexpected error, wanted %v, got %v", nil, err)
			}

		})
	}
}

func TestGas_gasSStoreEIP2929_Successful(t *testing.T) {

	tests := map[string]struct {
		setup  func(*context)
		gas    tosca.Gas
		refund tosca.Gas
	}{
		"noop with slot present": {
			setup: func(c *context) {
				c.context.(*tosca.MockRunContext).EXPECT().
					GetStorage(gomock.Any(), gomock.Any()).Return(tosca.Word{}) // current value
				c.context.(*tosca.MockRunContext).EXPECT().
					IsSlotInAccessList(gomock.Any(), gomock.Any()).Return(true, false)
				c.context.(*tosca.MockRunContext).EXPECT().
					AccessStorage(gomock.Any(), gomock.Any()).Return(tosca.WarmAccess)
				c.stack.push(uint256.NewInt(0)) // value
				c.stack.push(uint256.NewInt(0)) // key
			},
			gas: ColdSloadCostEIP2929 + WarmStorageReadCostEIP2929,
		},
		"create slot": {
			setup: func(c *context) {

				c.context.(*tosca.MockRunContext).EXPECT().
					GetStorage(gomock.Any(), gomock.Any()).Return(tosca.Word{}) // current value
				c.context.(*tosca.MockRunContext).EXPECT().
					IsSlotInAccessList(gomock.Any(), gomock.Any()).Return(true, true)
				c.context.(*tosca.MockRunContext).EXPECT().
					GetCommittedStorage(gomock.Any(), gomock.Any()).Return(tosca.Word{0}) // original value
				c.stack.push(uint256.NewInt(1)) // value
				c.stack.push(uint256.NewInt(1)) // key
			},
			gas: SstoreSetGasEIP2200,
		},
		"delete slot current same as original": {
			setup: func(c *context) {
				c.context.(*tosca.MockRunContext).EXPECT().
					GetCommittedStorage(gomock.Any(), gomock.Any()).Return(tosca.Word{1}) // original value
				c.context.(*tosca.MockRunContext).EXPECT().
					IsSlotInAccessList(gomock.Any(), gomock.Any()).Return(true, true)
				c.context.(*tosca.MockRunContext).EXPECT().
					GetStorage(gomock.Any(), gomock.Any()).Return(tosca.Word{1}) // current value
				c.stack.push(uint256.NewInt(0)) // value
				c.stack.push(uint256.NewInt(0)) // key
			},
			gas:    SstoreResetGasEIP2200 - ColdSloadCostEIP2929,
			refund: SstoreClearsScheduleRefundEIP2200,
		},
		"write existing slot": {
			setup: func(c *context) {
				c.context.(*tosca.MockRunContext).EXPECT().
					GetCommittedStorage(gomock.Any(), gomock.Any()).Return(tosca.Word{1}) // original value
				c.context.(*tosca.MockRunContext).EXPECT().
					IsSlotInAccessList(gomock.Any(), gomock.Any()).Return(true, true)
				c.context.(*tosca.MockRunContext).EXPECT().
					GetStorage(gomock.Any(), gomock.Any()).Return(tosca.Word{1}) // current value
				c.stack.push(uint256.NewInt(1)) // value
				c.stack.push(uint256.NewInt(0)) // key
			},
			gas: SstoreResetGasEIP2200 - ColdSloadCostEIP2929,
		},
		"recreate slot": {
			setup: func(c *context) {
				c.context.(*tosca.MockRunContext).EXPECT().
					GetCommittedStorage(gomock.Any(), gomock.Any()).Return(tosca.Word{1}) // original value
				c.context.(*tosca.MockRunContext).EXPECT().
					IsSlotInAccessList(gomock.Any(), gomock.Any()).Return(true, true)
				c.context.(*tosca.MockRunContext).EXPECT().
					GetStorage(gomock.Any(), gomock.Any()).Return(tosca.Word{0}) // current value
				c.stack.push(uint256.NewInt(1)) // value
				c.stack.push(uint256.NewInt(0)) // key
			},
			gas:    WarmStorageReadCostEIP2929,
			refund: -SstoreClearsScheduleRefundEIP2200,
		},
		"delete slot different current and original": {
			setup: func(c *context) {
				c.context.(*tosca.MockRunContext).EXPECT().
					GetCommittedStorage(gomock.Any(), gomock.Any()).Return(tosca.Word{1}) // original value
				c.context.(*tosca.MockRunContext).EXPECT().
					IsSlotInAccessList(gomock.Any(), gomock.Any()).Return(true, true)
				c.context.(*tosca.MockRunContext).EXPECT().
					GetStorage(gomock.Any(), gomock.Any()).Return(tosca.Word{2}) // current value
				c.stack.push(uint256.NewInt(0)) // value
				c.stack.push(uint256.NewInt(0)) // key
			},
			gas:    WarmStorageReadCostEIP2929,
			refund: SstoreClearsScheduleRefundEIP2200,
		},
		"reset to original inexistent slot": {
			setup: func(c *context) {
				c.context.(*tosca.MockRunContext).EXPECT().
					GetCommittedStorage(gomock.Any(), gomock.Any()).Return(tosca.Word{0}) // original value
				c.context.(*tosca.MockRunContext).EXPECT().
					IsSlotInAccessList(gomock.Any(), gomock.Any()).Return(true, true)
				c.context.(*tosca.MockRunContext).EXPECT().
					GetStorage(gomock.Any(), gomock.Any()).Return(tosca.Word{2}) // current value
				c.stack.push(uint256.NewInt(0)) // value
				c.stack.push(uint256.NewInt(0)) // key
			},
			gas:    WarmStorageReadCostEIP2929,
			refund: SstoreSetGasEIP2200 - WarmStorageReadCostEIP2929,
		},
		"reset to original existent slot": {
			setup: func(c *context) {
				c.context.(*tosca.MockRunContext).EXPECT().
					GetCommittedStorage(gomock.Any(), gomock.Any()).Return(
					tosca.Word(uint256.NewInt(1).Bytes32())) // original value
				c.context.(*tosca.MockRunContext).EXPECT().
					IsSlotInAccessList(gomock.Any(), gomock.Any()).Return(true, true)
				c.context.(*tosca.MockRunContext).EXPECT().
					GetStorage(gomock.Any(), gomock.Any()).Return(tosca.Word{2}) // current value
				c.stack.push(uint256.NewInt(1)) // value
				c.stack.push(uint256.NewInt(0)) // key
			},
			gas:    WarmStorageReadCostEIP2929,
			refund: (SstoreResetGasEIP2200 - ColdSloadCostEIP2929) - WarmStorageReadCostEIP2929,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			runContext := tosca.NewMockRunContext(ctrl)

			ctxt := context{
				status:  statusRunning,
				stack:   NewStack(),
				memory:  NewMemory(),
				context: runContext,
				gas:     1 << 20,
			}
			test.setup(&ctxt)

			gas, err := gasSStoreEIP2929(&ctxt)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			if gas != test.gas {
				t.Errorf("unexpected gas cost, wanted %v, got %v", test.gas, gas)
			}
			if ctxt.refund != test.refund {
				t.Errorf("unexpected refund, wanted %v, got %v", test.refund, ctxt.refund)
			}

		})
	}

}
