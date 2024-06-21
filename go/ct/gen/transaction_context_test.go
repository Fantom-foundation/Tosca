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

	variableCheck := func(txCtx st.TransactionContext, assignment Assignment, t *testing.T,
		variable Variable, shouldBePresent, shouldBeAssigned bool, value common.U256) {
		t.Helper()
		assignedValue, ok := assignment[variable]
		if !ok {
			t.Errorf("Variable %v should be in assignment.", variable.String())
		}
		if shouldBePresent {
			if !assignedValue.IsUint64() {
				t.Errorf("Variable %v should be assigned a uint64 value.", variable.String())
			}
			if assignedValue.Uint64() >= uint64(len(txCtx.BlobHashes)) {
				t.Errorf("Assigned value for %v %v is out of range.", variable.String(), assignedValue.Uint64())
			}
		} else {
			if assignedValue.Uint64() < uint64(len(txCtx.BlobHashes)) {
				t.Errorf("Assigned value for %v %v is not out of range.", variable.String(), assignedValue.Uint64())
			}
		}
		if shouldBeAssigned {
			if !assignedValue.Eq(value) {
				t.Errorf("Assigned value for %v %vis not the expected value.", variable.String(), assignedValue.Uint64())
			}
		}
	}

	tests := map[string]struct {
		setup func(*TransactionContextGenerator, *Assignment)
		check func(st.TransactionContext, Assignment, *testing.T)
	}{
		"present": {
			setup: func(txCtxGen *TransactionContextGenerator, _ *Assignment) {
				txCtxGen.PresentBlobHashIndex(Variable("v1"))
			},
			check: func(txCtx st.TransactionContext, assignment Assignment, t *testing.T) {
				variableCheck(txCtx, assignment, t, Variable("v1"), true, false, common.NewU256(0))
			},
		},
		"present-and-assigned": {
			setup: func(txCtxGen *TransactionContextGenerator, assignment *Assignment) {
				txCtxGen.PresentBlobHashIndex(Variable("v1"))
				(*assignment)[Variable("v1")] = common.NewU256(5)
			},
			check: func(txCtx st.TransactionContext, assignment Assignment, t *testing.T) {
				variableCheck(txCtx, assignment, t, Variable("v1"), true, true, common.NewU256(5))
			},
		},
		"absent": {
			setup: func(txCtxGen *TransactionContextGenerator, _ *Assignment) {
				txCtxGen.AbsentBlobHashIndex(Variable("v1"))
			},
			check: func(txCtx st.TransactionContext, assignment Assignment, t *testing.T) {
				variableCheck(txCtx, assignment, t, Variable("v1"), false, false, common.NewU256(0))
			},
		},
		"absent-and-assigned": {
			setup: func(txCtxGen *TransactionContextGenerator, assignment *Assignment) {
				txCtxGen.AbsentBlobHashIndex(Variable("v1"))
				(*assignment)[Variable("v1")] = common.NewU256(5)
			},
			check: func(txCtx st.TransactionContext, assignment Assignment, t *testing.T) {
				variableCheck(txCtx, assignment, t, Variable("v1"), false, true, common.NewU256(5))
			},
		},
		"absent-and-present": {
			setup: func(txCtxGen *TransactionContextGenerator, _ *Assignment) {
				txCtxGen.AbsentBlobHashIndex(Variable("v2"))
				txCtxGen.PresentBlobHashIndex(Variable("v1"))
			},
			check: func(txCtx st.TransactionContext, assignment Assignment, t *testing.T) {
				variableCheck(txCtx, assignment, t, Variable("v1"), true, false, common.NewU256(0))
				variableCheck(txCtx, assignment, t, Variable("v2"), false, false, common.NewU256(0))
			},
		},
		"present-and-absent": {
			setup: func(txCtxGen *TransactionContextGenerator, _ *Assignment) {
				txCtxGen.PresentBlobHashIndex(Variable("v1"))
				txCtxGen.AbsentBlobHashIndex(Variable("v2"))
			},
			check: func(txCtx st.TransactionContext, assignment Assignment, t *testing.T) {
				variableCheck(txCtx, assignment, t, Variable("v1"), true, false, common.NewU256(0))
				variableCheck(txCtx, assignment, t, Variable("v2"), false, false, common.NewU256(0))
			},
		},
		"present-assigned-and-absent": {
			setup: func(txCtxGen *TransactionContextGenerator, assignment *Assignment) {
				txCtxGen.PresentBlobHashIndex(Variable("v1"))
				(*assignment)[Variable("v1")] = common.NewU256(5)
				txCtxGen.AbsentBlobHashIndex(Variable("v2"))
			},
			check: func(txCtx st.TransactionContext, assignment Assignment, t *testing.T) {
				variableCheck(txCtx, assignment, t, Variable("v1"), true, true, common.NewU256(5))
				variableCheck(txCtx, assignment, t, Variable("v2"), false, false, common.NewU256(0))
			},
		},
		"absent-assigned-and-present": {
			setup: func(txCtxGen *TransactionContextGenerator, assignment *Assignment) {
				txCtxGen.AbsentBlobHashIndex(Variable("v2"))
				txCtxGen.PresentBlobHashIndex(Variable("v1"))
				(*assignment)[Variable("v2")] = common.NewU256(5)
			},
			check: func(txCtx st.TransactionContext, assignment Assignment, t *testing.T) {
				variableCheck(txCtx, assignment, t, Variable("v1"), true, false, common.NewU256(0))
				variableCheck(txCtx, assignment, t, Variable("v2"), false, true, common.NewU256(5))
			},
		},
		"assigned-too-big-absent": {
			setup: func(txCtxGen *TransactionContextGenerator, assignment *Assignment) {
				(*assignment)[Variable("v1")] = common.NewU256(1, 1)
				txCtxGen.AbsentBlobHashIndex(Variable("v1"))
			},
			check: func(txCtx st.TransactionContext, assignment Assignment, t *testing.T) {
				variableCheck(txCtx, assignment, t, Variable("v1"), false, true, common.NewU256(1, 1))
			},
		},
		"zero-assigned-absent": {
			setup: func(txCtxGen *TransactionContextGenerator, assignment *Assignment) {
				(*assignment)[Variable("v1")] = common.NewU256(0)
				txCtxGen.AbsentBlobHashIndex(Variable("v1"))
			},
			check: func(txCtx st.TransactionContext, assignment Assignment, t *testing.T) {
				variableCheck(txCtx, assignment, t, Variable("v1"), false, true, common.NewU256(0))
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
		"present-and-absent": {
			setup: func(txCtxGen *TransactionContextGenerator, _ *Assignment) {
				txCtxGen.PresentBlobHashIndex(Variable("v1"))
				txCtxGen.AbsentBlobHashIndex(Variable("v1"))
			},
		},
		"absent-and-present": {
			setup: func(txCtxGen *TransactionContextGenerator, _ *Assignment) {
				txCtxGen.AbsentBlobHashIndex(Variable("v1"))
				txCtxGen.PresentBlobHashIndex(Variable("v1"))
			},
		},
		"assigned-and-present": {
			setup: func(txCtxGen *TransactionContextGenerator, assignment *Assignment) {
				(*assignment)[Variable("v1")] = common.NewU256(math.MaxUint64)
				txCtxGen.PresentBlobHashIndex(Variable("v1"))
			},
		},
		"assigned-present-and-assigned-absent": {
			setup: func(txCtxGen *TransactionContextGenerator, assignment *Assignment) {
				(*assignment)[Variable("v1")] = common.NewU256(3)
				txCtxGen.PresentBlobHashIndex(Variable("v1"))
				(*assignment)[Variable("v2")] = common.NewU256(2)
				txCtxGen.AbsentBlobHashIndex(Variable("v2"))
			},
		},
		"assigned-too-big-present": {
			setup: func(txCtxGen *TransactionContextGenerator, assignment *Assignment) {
				(*assignment)[Variable("v1")] = common.NewU256(1, 1)
				txCtxGen.PresentBlobHashIndex(Variable("v1"))
			},
		},
		"assigned-max-present": {
			setup: func(txCtxGen *TransactionContextGenerator, assignment *Assignment) {
				(*assignment)[Variable("v1")] = common.NewU256(math.MaxUint64)
				txCtxGen.PresentBlobHashIndex(Variable("v1"))
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
