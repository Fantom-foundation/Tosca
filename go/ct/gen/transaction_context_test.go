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
	"math"
	"testing"

	"github.com/Fantom-foundation/Tosca/go/ct/common"
	"github.com/Fantom-foundation/Tosca/go/ct/st"
	"github.com/Fantom-foundation/Tosca/go/vm"
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
				txCtxGen.PresentBlobHashIndex(Variable("v1"))
			},
			expected: "{$v1 < len(blobHashes)}",
		},
		"has-not-blob-hash": {
			setup: func(txCtxGen *TransactionContextGenerator) {
				txCtxGen.AbsentBlobHashIndex(Variable("v1"))
			},
			expected: "{$v1 >= len(blobHashes)}",
		},
		"has-blob-hash-and-not-blob-hash": {
			setup: func(txCtxGen *TransactionContextGenerator) {
				txCtxGen.PresentBlobHashIndex(Variable("v1"))
				txCtxGen.AbsentBlobHashIndex(Variable("v2"))
			},
			expected: "{$v1 < len(blobHashes) && $v2 >= len(blobHashes)}",
		},
		"unsatifiable": {
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
	txCtx.unsatisfiable = true
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

	if txCtx.OriginAddress == (vm.Address{}) {
		t.Errorf("Generated origin address has default value.")
	}
}

func TestTransactionContextGenerator_GenerateConstrained(t *testing.T) {

	tests := map[string]struct {
		setup func(*TransactionContextGenerator, *Assignment)
		check func(st.TransactionContext, Assignment, *testing.T)
	}{
		"hash-blob-hash": {
			setup: func(txCtxGen *TransactionContextGenerator, _ *Assignment) {
				txCtxGen.PresentBlobHashIndex(Variable("v1"))
			},
			check: func(txCtx st.TransactionContext, assignment Assignment, t *testing.T) {
				value, ok := assignment[Variable("v1")]
				if !ok {
					t.Errorf("Variable v1 should have been assigned.")
				}
				if !value.IsUint64() {
					t.Errorf("Variable v1 should have been assigned a uint64 value.")
				}
				if value.Uint64() >= uint64(len(txCtx.BlobHashes)) {
					t.Errorf("Assigned value for v1 is out of range.")
				}
			},
		},
		"hash-blob-hash-and-assigned": {
			setup: func(txCtxGen *TransactionContextGenerator, assignment *Assignment) {
				txCtxGen.PresentBlobHashIndex(Variable("v1"))
				(*assignment)[Variable("v1")] = common.NewU256(5)
			},
			check: func(txCtx st.TransactionContext, assignment Assignment, t *testing.T) {
				value, ok := assignment[Variable("v1")]
				if !ok {
					t.Errorf("Variable v1 should have been assigned.")
				}
				if value.Uint64() != 5 {
					t.Errorf("Assigned value for v1 is not the expected value.")
				}
				if value.Uint64() >= uint64(len(txCtx.BlobHashes)) {
					t.Errorf("Assigned value for v1 is out of range.")
				}
			},
		},
		"hash-not-blob-hash": {
			setup: func(txCtxGen *TransactionContextGenerator, _ *Assignment) {
				txCtxGen.AbsentBlobHashIndex(Variable("v1"))
			},
			check: func(txCtx st.TransactionContext, assignment Assignment, t *testing.T) {
				value, ok := assignment[Variable("v1")]
				if !ok {
					t.Errorf("Variable v1 should have been assigned.")
				}
				if value.Uint64() < uint64(len(txCtx.BlobHashes)) {
					t.Errorf("Assigned value for v1 is out of range.")
				}
			},
		},
		"hash-not-blob-hash-and-assigned": {
			setup: func(txCtxGen *TransactionContextGenerator, assignment *Assignment) {
				txCtxGen.AbsentBlobHashIndex(Variable("v1"))
				(*assignment)[Variable("v1")] = common.NewU256(5)
			},
			check: func(txCtx st.TransactionContext, assignment Assignment, t *testing.T) {
				value, ok := assignment[Variable("v1")]
				if !ok {
					t.Errorf("Variable v1 should have been assigned.")
				}
				if value.Uint64() != 5 {
					t.Errorf("Assigned value for v1 is not the expected value.")
				}
				if value.Uint64() < uint64(len(txCtx.BlobHashes)) {
					t.Errorf("Assigned value for v1 should be out of range.")
				}
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			rnd := rand.New(0)
			txCtxGen := NewTransactionContextGenerator()
			assignment := Assignment{}
			test.setup(txCtxGen, &assignment)
			txCtx, err := txCtxGen.Generate(assignment, rnd)
			if err != nil {
				t.Errorf("Error generating transaction context: %v", err)
			}
			test.check(*txCtx, assignment, t)
		})
	}
}

func TestTransactionContextGenerator_GenerateUnsatisfiable(t *testing.T) {

	tests := map[string]struct {
		setup func(*TransactionContextGenerator, *Assignment)
	}{
		"hash-and-has-not-blob-hash": {
			setup: func(txCtxGen *TransactionContextGenerator, _ *Assignment) {
				txCtxGen.PresentBlobHashIndex(Variable("v1"))
				txCtxGen.AbsentBlobHashIndex(Variable("v1"))
			},
		},
		"hash-not-and-has-blob-hash": {
			setup: func(txCtxGen *TransactionContextGenerator, _ *Assignment) {
				txCtxGen.AbsentBlobHashIndex(Variable("v1"))
				txCtxGen.PresentBlobHashIndex(Variable("v1"))
			},
		},
		"assigned-and-has-blob-hash": {
			setup: func(txCtxGen *TransactionContextGenerator, assignment *Assignment) {
				(*assignment)[Variable("v1")] = common.NewU256(math.MaxUint64)
				txCtxGen.PresentBlobHashIndex(Variable("v1"))
			},
		},
		"assigned-and-has-not-blob-hash": {
			setup: func(txCtxGen *TransactionContextGenerator, assignment *Assignment) {
				(*assignment)[Variable("v1")] = common.NewU256(0)
				txCtxGen.AbsentBlobHashIndex(Variable("v1"))
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			rnd := rand.New(0)
			txCtxGen := NewTransactionContextGenerator()
			assignment := Assignment{}
			test.setup(txCtxGen, &assignment)
			_, err := txCtxGen.Generate(assignment, rnd)
			if err != ErrUnsatisfiable {
				t.Errorf("Expected an error, but got none.")
			}
		})
	}
}
