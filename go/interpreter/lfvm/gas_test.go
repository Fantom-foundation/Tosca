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

// --- SStore ---

func TestGas_getDynamicCostsForSstore_exhaustive(t *testing.T) {
	// This test exhaustively checks the computation of the dynamic gas costs for
	// the SSTORE instruction by enumerating every possible input combination
	// and comparing the result with the specification found on
	// https://www.evm.codes.

	// The specification found on https://www.evm.codes is provided in the form
	// of a python code snippet. The following function is a direct translation
	// of the python code to Go, parameterized to allow for easy testing of
	// different revisions.
	makeSpec := func(create, update, touch tosca.Gas) func(s storageStateExample) tosca.Gas {
		return func(s storageStateExample) tosca.Gas {
			// source: https://www.evm.codes
			gas := tosca.Gas(0)
			if s.value == s.current {
				gas = touch
			} else if s.current == s.original {
				if s.original == 0 {
					gas = create
				} else {
					gas = update
				}
			} else {
				gas = touch
			}
			return gas
		}
	}

	specs := map[tosca.Revision]func(storageStateExample) tosca.Gas{
		// source: https://www.evm.codes/?fork=istanbul
		tosca.R07_Istanbul: makeSpec(20000, 5000, 800),
		// source: https://www.evm.codes/?fork=berlin
		tosca.R09_Berlin: makeSpec(20000, 2900, 100),
	}

	// All other revisions inherit the definition from their predecessor.
	specs[tosca.R10_London] = specs[tosca.R09_Berlin]
	specs[tosca.R11_Paris] = specs[tosca.R10_London]
	specs[tosca.R12_Shanghai] = specs[tosca.R11_Paris]
	specs[tosca.R13_Cancun] = specs[tosca.R12_Shanghai]

	// Check that gas prices are computed correctly.
	for _, revision := range tosca.GetAllKnownRevisions() {
		spec, found := specs[revision]
		if !found {
			t.Errorf("missing specification for revision %v", revision)
			continue
		}
		for storageStatus, example := range getStorageStateExamples() {
			want := spec(example)
			got := getDynamicCostsForSstore(revision, storageStatus)
			if got != want {
				t.Errorf(
					"unexpected result for (%v,%v), wanted %d, got %d",
					revision,
					storageStatus,
					want,
					got,
				)
			}
		}
	}
}

func TestGas_getRefundForSstore_exhaustive(t *testing.T) {
	// This test exhaustively checks the computation of the refunds granted by
	// the SSTORE instruction by enumerating every possible input combination
	// and comparing the result with the specification found on
	// https://www.evm.codes.

	// The specification found on https://www.evm.codes is provided in the form
	// of a python code snippet. The following function is a direct translation
	// of the python code to Go, parameterized to allow for easy testing of
	// different revisions.
	makeSpec := func(delete, resetToZero, resetToNonZero tosca.Gas) func(s storageStateExample) tosca.Gas {
		return func(s storageStateExample) tosca.Gas {
			// source: https://www.evm.codes
			refund := tosca.Gas(0)
			if s.value != s.current {
				if s.current == s.original {
					if s.original != 0 && s.value == 0 {
						refund += delete
					}
				} else {
					if s.original != 0 {
						if s.current == 0 {
							refund -= delete
						} else if s.value == 0 {
							refund += delete
						}
					}
					if s.value == s.original {
						if s.original == 0 {
							refund += resetToZero
						} else {
							refund += resetToNonZero
						}
					}
				}
			}
			return refund
		}
	}

	specs := map[tosca.Revision]func(storageStateExample) tosca.Gas{
		// source: https://www.evm.codes/?fork=istanbul
		tosca.R07_Istanbul: makeSpec(15000, 19200, 4200),
		// source: https://www.evm.codes/?fork=berlin
		tosca.R09_Berlin: makeSpec(15000, 20000-100, 5000-2100-100),
		// source: https://www.evm.codes/?fork=london
		tosca.R10_London: makeSpec(4800, 20000-100, 5000-2100-100),
	}
	// All other revisions inherit the definition from their predecessor.
	specs[tosca.R11_Paris] = specs[tosca.R10_London]
	specs[tosca.R12_Shanghai] = specs[tosca.R11_Paris]
	specs[tosca.R13_Cancun] = specs[tosca.R12_Shanghai]

	// Check that gas prices are computed correctly.
	for _, revision := range tosca.GetAllKnownRevisions() {
		spec, found := specs[revision]
		if !found {
			t.Errorf("missing specification for revision %v", revision)
			continue
		}
		for storageStatus, example := range getStorageStateExamples() {
			want := spec(example)
			got := getRefundForSstore(revision, storageStatus)
			if got != want {
				t.Errorf(
					"unexpected result for (%v,%v), wanted %d, got %d",
					revision,
					storageStatus,
					want,
					got,
				)
			}
		}
	}
}

// getStorageStateExamples returns a map enumerating all possible storage state
// and mapping them to a tipple if original, current and new values of a storage
// slot that constitutes the associated storage state.
//
// This function is intended for testing storage related gas costs and refunds
// functions by providing a complete set of test-case inputs.
func getStorageStateExamples() map[tosca.StorageStatus]storageStateExample {
	X, Y, Z := 1, 2, 3
	return map[tosca.StorageStatus]storageStateExample{
		tosca.StorageAssigned:         {X, Y, Z},
		tosca.StorageAdded:            {0, 0, Z},
		tosca.StorageDeleted:          {X, X, 0},
		tosca.StorageModified:         {X, X, Z},
		tosca.StorageDeletedAdded:     {X, 0, Z},
		tosca.StorageModifiedDeleted:  {X, Y, 0},
		tosca.StorageDeletedRestored:  {X, 0, X},
		tosca.StorageAddedDeleted:     {0, X, 0},
		tosca.StorageModifiedRestored: {X, Y, X},
	}
}

type storageStateExample struct {
	original, current, value int
}

func TestGas_gasSelfDestruct(t *testing.T) {

	tests := map[string]struct {
		setup  func(*tosca.MockRunContext)
		gas    tosca.Gas
		refund tosca.Gas
	}{
		"istanbul has not selfdestructed": {
			setup: func(runContext *tosca.MockRunContext) {
				runContext.EXPECT().AccountExists(gomock.Any()).Return(true)
				runContext.EXPECT().HasSelfDestructed(gomock.Any()).Return(false)
			},
			refund: SelfdestructRefundGas,
			gas:    SelfdestructGasEIP150,
		},
		"istanbul address not present": {
			setup: func(runContext *tosca.MockRunContext) {
				runContext.EXPECT().AccountExists(gomock.Any()).Return(false)
				runContext.EXPECT().GetBalance(gomock.Any()).Return(tosca.Value{1})
				runContext.EXPECT().HasSelfDestructed(gomock.Any()).Return(true)
			},
			gas: SelfdestructGasEIP150 + CreateBySelfdestructGas,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			runContext := tosca.NewMockRunContext(ctrl)
			test.setup(runContext)

			ctxt := context{
				stack:   NewStack(),
				memory:  NewMemory(),
				context: runContext,
			}

			// Prepare stack arguments.
			ctxt.stack.push(uint256.NewInt(1))

			gotGas := gasSelfdestruct(&ctxt)

			if gotGas != test.gas {
				t.Errorf("unexpected gas costs, wanted %d, got %d", test.gas, gotGas)
			}
			if ctxt.refund != test.refund {
				t.Errorf("unexpected refund, wanted %d, got %d", test.refund, ctxt.refund)
			}
		})
	}
}

func TestGas_gasSelfdestructEIP2929(t *testing.T) {

	tests := map[string]struct {
		setup    func(*tosca.MockRunContext)
		revision tosca.Revision
		gas      tosca.Gas
		refund   tosca.Gas
	}{
		"berlin regular": {
			setup: func(runContext *tosca.MockRunContext) {
				runContext.EXPECT().IsAddressInAccessList(gomock.Any()).Return(true)
				runContext.EXPECT().AccountExists(gomock.Any()).Return(true)
				runContext.EXPECT().HasSelfDestructed(gomock.Any()).Return(false)
			},
			refund:   SelfdestructRefundGas,
			revision: tosca.R09_Berlin,
		},
		"london address not in list": {
			setup: func(runContext *tosca.MockRunContext) {
				runContext.EXPECT().IsAddressInAccessList(gomock.Any()).Return(false)
				runContext.EXPECT().AccessAccount(gomock.Any()).Return(tosca.ColdAccess)
				runContext.EXPECT().AccountExists(gomock.Any()).Return(true)
			},
			revision: tosca.R10_London,
			gas:      ColdAccountAccessCostEIP2929,
		},
		"london create new account": {
			setup: func(runContext *tosca.MockRunContext) {
				runContext.EXPECT().IsAddressInAccessList(gomock.Any()).Return(true)
				runContext.EXPECT().AccountExists(gomock.Any()).Return(false)
				runContext.EXPECT().GetBalance(gomock.Any()).Return(tosca.Value{1})
			},
			revision: tosca.R10_London,
			gas:      CreateBySelfdestructGas,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			runContext := tosca.NewMockRunContext(ctrl)
			test.setup(runContext)

			ctxt := context{
				params: tosca.Parameters{
					BlockParameters: tosca.BlockParameters{
						Revision: test.revision,
					},
				},
				stack:   NewStack(),
				memory:  NewMemory(),
				context: runContext,
			}

			// Prepare stack arguments.
			ctxt.stack.push(uint256.NewInt(1))

			gotGas := gasSelfdestructEIP2929(&ctxt)

			if gotGas != test.gas {
				t.Errorf("unexpected gas costs, wanted %d, got %d", test.gas, gotGas)
			}
			if ctxt.refund != test.refund {
				t.Errorf("unexpected refund, wanted %d, got %d", test.refund, ctxt.refund)
			}
		})
	}
}
