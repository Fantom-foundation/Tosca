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
)

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
