// Copyright (c) 2024 Fantom Foundation
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at fantom.foundation/bsl11.
//
// Change Date: 2028-4-16
//
// On the date above, in accordance with the Business Source License, use of
// this software will be governed by the GNU Lesser General Public License v3.

package gen

import (
	"maps"
	"math"
	"testing"

	"github.com/Fantom-foundation/Tosca/go/ct/common"
	"github.com/Fantom-foundation/Tosca/go/tosca"
	"pgregory.net/rand"
)

func TestNewTransactionContextGenerator_String(t *testing.T) {

	tests := map[string]struct {
		setup    func(*TransactionContextGenerator)
		expected string
	}{
		"empty": {
			setup:    func(_ *TransactionContextGenerator) {},
			expected: "{true}",
		},
		"has-blob-hash": {
			setup: func(txCtxGen *TransactionContextGenerator) {
				txCtxGen.IsPresentBlobHashIndex(Variable("v1"))
			},
			expected: "{$v1 < len(blobHashes)}",
		},
		"has-not-blob-hash": {
			setup: func(txCtxGen *TransactionContextGenerator) {
				txCtxGen.IsAbsentBlobHashIndex(Variable("v1"))
			},
			expected: "{$v1 >= len(blobHashes)}",
		},
		"has-blob-hash-and-not-blob-hash": {
			setup: func(txCtxGen *TransactionContextGenerator) {
				txCtxGen.IsPresentBlobHashIndex(Variable("v1"))
				txCtxGen.IsAbsentBlobHashIndex(Variable("v2"))
			},
			expected: "{$v1 < len(blobHashes) ∧ $v2 >= len(blobHashes)}",
		},
		"unsatisfiable": {
			setup: func(txCtxGen *TransactionContextGenerator) {
				txCtxGen.unsatisfiable = true
			},
			expected: "{false}",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			txCtxGen := NewTransactionContextGenerator()
			test.setup(txCtxGen)
			if txCtxGen.String() != test.expected {
				t.Errorf("unexpected string representation. wanted %v, got %v.", test.expected, txCtxGen.String())
			}
		})
	}

}

func TestNewTransactionContextGenerator_Clone(t *testing.T) {
	txCtx := NewTransactionContextGenerator()
	txCtx.IsPresentBlobHashIndex(Variable("v1"))
	clone := txCtx.Clone()
	if clone.String() != txCtx.String() {
		t.Errorf("Clone should be equal to the original.")
	}
}

func TestTransactionContext_GenerateUnconstrained(t *testing.T) {
	rnd := rand.New(0)
	txCtxGen := NewTransactionContextGenerator()
	txCtx, err := txCtxGen.Generate(Assignment{}, rnd)
	if err != nil {
		t.Errorf("Error generating transaction context: %v", err)
	}

	if txCtx.BlobHashes == nil {
		t.Errorf("Generated blob hashes has default value.")
	}

	if txCtx.OriginAddress == (tosca.Address{}) {
		t.Errorf("Generated origin address has default value.")
	}
}

func TestTransactionContextGenerator_GenerateConstrained(t *testing.T) {

	tests := map[string]struct {
		setup           func(*TransactionContextGenerator, Assignment)
		shouldBePresent []Variable
		shouldBeAbsent  []Variable
	}{
		"present": {
			setup: func(txCtxGen *TransactionContextGenerator, _ Assignment) {
				txCtxGen.IsPresentBlobHashIndex(Variable("v1"))
			},
			shouldBePresent: []Variable{"v1"},
		},
		"present-and-assigned": {
			setup: func(txCtxGen *TransactionContextGenerator, assignment Assignment) {
				txCtxGen.IsPresentBlobHashIndex(Variable("v1"))
				(assignment)[Variable("v1")] = common.NewU256(5)
			},
			shouldBePresent: []Variable{"v1"},
		},
		"absent": {
			setup: func(txCtxGen *TransactionContextGenerator, _ Assignment) {
				txCtxGen.IsAbsentBlobHashIndex(Variable("v1"))
			},
			shouldBeAbsent: []Variable{"v1"},
		},
		"absent-and-assigned": {
			setup: func(txCtxGen *TransactionContextGenerator, assignment Assignment) {
				txCtxGen.IsAbsentBlobHashIndex(Variable("v1"))
				(assignment)[Variable("v1")] = common.NewU256(5)
			},
			shouldBeAbsent: []Variable{"v1"},
		},
		"absent-and-present": {
			setup: func(txCtxGen *TransactionContextGenerator, _ Assignment) {
				txCtxGen.IsAbsentBlobHashIndex(Variable("v2"))
				txCtxGen.IsPresentBlobHashIndex(Variable("v1"))
			},
			shouldBePresent: []Variable{"v1"},
			shouldBeAbsent:  []Variable{"v2"},
		},
		"present-and-absent": {
			setup: func(txCtxGen *TransactionContextGenerator, _ Assignment) {
				txCtxGen.IsPresentBlobHashIndex(Variable("v1"))
				txCtxGen.IsAbsentBlobHashIndex(Variable("v2"))
			},
			shouldBePresent: []Variable{"v1"},
			shouldBeAbsent:  []Variable{"v2"},
		},
		"present-assigned-and-absent": {
			setup: func(txCtxGen *TransactionContextGenerator, assignment Assignment) {
				txCtxGen.IsPresentBlobHashIndex(Variable("v1"))
				(assignment)[Variable("v1")] = common.NewU256(5)
				txCtxGen.IsAbsentBlobHashIndex(Variable("v2"))
			},
			shouldBePresent: []Variable{"v1"},
			shouldBeAbsent:  []Variable{"v2"},
		},
		"absent-assigned-and-present": {
			setup: func(txCtxGen *TransactionContextGenerator, assignment Assignment) {
				txCtxGen.IsAbsentBlobHashIndex(Variable("v2"))
				txCtxGen.IsPresentBlobHashIndex(Variable("v1"))
				(assignment)[Variable("v2")] = common.NewU256(5)
			},
			shouldBePresent: []Variable{"v1"},
			shouldBeAbsent:  []Variable{"v2"},
		},
		"assigned-too-big-absent": {
			setup: func(txCtxGen *TransactionContextGenerator, assignment Assignment) {
				(assignment)[Variable("v1")] = common.NewU256(1, 1)
				txCtxGen.IsAbsentBlobHashIndex(Variable("v1"))
			},
			shouldBeAbsent: []Variable{"v1"},
		},
		"zero-assigned-absent": {
			setup: func(txCtxGen *TransactionContextGenerator, assignment Assignment) {
				(assignment)[Variable("v1")] = common.NewU256(0)
				txCtxGen.IsAbsentBlobHashIndex(Variable("v1"))
			},
			shouldBeAbsent: []Variable{"v1"},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			rnd := rand.New(0)
			txCtxGen := NewTransactionContextGenerator()
			assignment := Assignment{}
			test.setup(txCtxGen, assignment)
			assignmentBackup := maps.Clone(assignment)
			txCtx, err := txCtxGen.Generate(assignment, rnd)
			for variable, value := range assignmentBackup {
				if !assignment[variable].Eq(value) {
					t.Errorf("Assignment should not be modified.")
				}
			}
			if err != nil {
				t.Errorf("Error generating transaction context: %v", err)
			}
			for _, variable := range test.shouldBePresent {
				assignedValue, ok := assignment[variable]
				if !ok {
					t.Errorf("Variable %v should be in assignment.", variable.String())
				}
				if !assignedValue.IsUint64() {
					t.Errorf("Variable %v should be assigned a uint64 value.", variable.String())
				}
				if assignedValue.Uint64() >= uint64(len(txCtx.BlobHashes)) {
					t.Errorf("Assigned value for %v %v is out of range.", variable.String(), assignedValue.Uint64())
				}
			}
			for _, variable := range test.shouldBeAbsent {
				assignedValue, ok := assignment[variable]
				if !ok {
					t.Errorf("Variable %v should be in assignment.", variable.String())
				}
				if assignedValue.IsUint64() && assignedValue.Uint64() < uint64(len(txCtx.BlobHashes)) {
					t.Errorf("Assigned value for %v %v is not out of range.", variable.String(), assignedValue.Uint64())
				}
			}
		})
	}
}

func TestTransactionContextGenerator_GenerateUnsatisfiable(t *testing.T) {

	tests := map[string]struct {
		setup func(*TransactionContextGenerator, Assignment)
	}{
		"present-and-absent": {
			setup: func(txCtxGen *TransactionContextGenerator, _ Assignment) {
				txCtxGen.IsPresentBlobHashIndex(Variable("v1"))
				txCtxGen.IsAbsentBlobHashIndex(Variable("v1"))
			},
		},
		"absent-and-present": {
			setup: func(txCtxGen *TransactionContextGenerator, _ Assignment) {
				txCtxGen.IsAbsentBlobHashIndex(Variable("v1"))
				txCtxGen.IsPresentBlobHashIndex(Variable("v1"))
			},
		},
		"assigned-absent-less-than-assigned-present": {
			setup: func(txCtxGen *TransactionContextGenerator, assignment Assignment) {
				assignment[Variable("v1")] = common.NewU256(3)
				txCtxGen.IsPresentBlobHashIndex(Variable("v1"))
				assignment[Variable("v2")] = common.NewU256(2)
				txCtxGen.IsAbsentBlobHashIndex(Variable("v2"))
			},
		},
		"assigned-too-big-present": {
			setup: func(txCtxGen *TransactionContextGenerator, assignment Assignment) {
				assignment[Variable("v1")] = common.NewU256(1, 1)
				txCtxGen.IsPresentBlobHashIndex(Variable("v1"))
			},
		},
		"assigned-max-present": {
			setup: func(txCtxGen *TransactionContextGenerator, assignment Assignment) {
				assignment[Variable("v1")] = common.NewU256(math.MaxUint64)
				txCtxGen.IsPresentBlobHashIndex(Variable("v1"))
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			rnd := rand.New(0)
			txCtxGen := NewTransactionContextGenerator()
			assignment := Assignment{}
			test.setup(txCtxGen, assignment)
			assignmentBackup := maps.Clone(assignment)
			_, err := txCtxGen.Generate(assignment, rnd)
			for variable, value := range assignmentBackup {
				if !assignment[variable].Eq(value) {
					t.Errorf("Assignment should not be modified.")
				}
			}
			if err != ErrUnsatisfiable {
				t.Errorf("Expected an error, but got none.")
			}
		})
	}
}
