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
	"sort"

	"golang.org/x/exp/maps"
	"pgregory.net/rand"

	"github.com/Fantom-foundation/Tosca/go/ct/common"
	"github.com/Fantom-foundation/Tosca/go/ct/st"
	"github.com/Fantom-foundation/Tosca/go/tosca"
)

type TransactionContextGenerator struct {
	unsatisfiable bool
	// This map is used to keep track of variables that are required to have a value
	// that can be used as index for the blob hash list.
	blobHashVariables map[Variable]bool
}

func NewTransactionContextGenerator() *TransactionContextGenerator {
	return &TransactionContextGenerator{}
}

func (t *TransactionContextGenerator) Generate(assignment Assignment, rnd *rand.Rand) (*st.TransactionContext, error) {
	if t.unsatisfiable {
		return nil, ErrUnsatisfiable
	}

	maxMustHaveBlobHash := uint64(0)
	minMustNotHaveBlobHash := uint64(math.MaxUint64)

	// process all the bounded variables to find the maximum present and minimum absent values
	for variable, hasBlobHash := range t.blobHashVariables {
		if assignedValue, isBound := assignment[variable]; isBound {
			if hasBlobHash {
				if !assignedValue.IsUint64() {
					return nil, ErrUnsatisfiable
				}
				if assignedValue.Uint64() > maxMustHaveBlobHash {
					maxMustHaveBlobHash = assignedValue.Uint64()
				}
			} else {
				if assignedValue.IsUint64() && assignedValue.Uint64() < minMustNotHaveBlobHash {
					minMustNotHaveBlobHash = assignedValue.Uint64()
				}
			}
		}
	}

	if maxMustHaveBlobHash > minMustNotHaveBlobHash ||
		maxMustHaveBlobHash == math.MaxUint64 {
		return nil, ErrUnsatisfiable
	}

	blobHashesCount := maxMustHaveBlobHash
	if minMustNotHaveBlobHash != 0 {
		blobHashesCount += 1
	}
	for variable, hasBlobHash := range t.blobHashVariables {
		// the bounded variables are dealt with above
		if _, isBound := assignment[variable]; !isBound {
			newValueRangeSolver := NewRangeSolver(0, blobHashesCount-1)
			if !hasBlobHash {
				newValueRangeSolver = NewRangeSolver(blobHashesCount, math.MaxUint64)
			}
			newValue, err := newValueRangeSolver.Generate(rnd)
			if err != nil {
				return nil, err
			}
			assignment[variable] = common.NewU256(newValue)
		}
	}

	blobHashes := []tosca.Hash{}
	for i := uint64(0); i < blobHashesCount; i++ {
		blobHashes = append(blobHashes, common.GetRandomHash(rnd))
	}

	originAddress := common.RandomAddress(rnd)

	return &st.TransactionContext{
		OriginAddress: originAddress,
		BlobHashes:    blobHashes,
	}, nil
}

func (t *TransactionContextGenerator) Clone() *TransactionContextGenerator {
	if t.unsatisfiable {
		return &TransactionContextGenerator{unsatisfiable: true}
	}

	return &TransactionContextGenerator{
		unsatisfiable:     false,
		blobHashVariables: maps.Clone(t.blobHashVariables),
	}
}

func (t *TransactionContextGenerator) Restore(o *TransactionContextGenerator) {
	t.unsatisfiable = o.unsatisfiable
	t.blobHashVariables = maps.Clone(o.blobHashVariables)
}

func (t *TransactionContextGenerator) String() string {
	if t.unsatisfiable {
		return "{false}"
	}
	ret := ""
	if t.blobHashVariables != nil {

		keys := maps.Keys(t.blobHashVariables)
		sort.Slice(keys, func(i, j int) bool { return keys[i] < keys[j] })

		for _, variable := range keys {
			hasBlobHash := t.blobHashVariables[variable]
			if ret != "" {
				ret += " ∧ "
			}
			if hasBlobHash {
				ret += variable.String() + " < len(blobHashes)"
			} else {
				ret += variable.String() + " >= len(blobHashes)"
			}
		}
	}
	if len(ret) == 0 {
		ret = "true"
	}
	return "{" + ret + "}"
}

func (t *TransactionContextGenerator) IsPresentBlobHashIndex(variable Variable) {
	if t.unsatisfiable {
		return
	}
	if t.blobHashVariables == nil {
		t.blobHashVariables = make(map[Variable]bool)
	}
	if val, ok := t.blobHashVariables[variable]; !ok {
		t.blobHashVariables[variable] = true
	} else if !val {
		t.markUnsatisfiable()
	}
}

func (t *TransactionContextGenerator) IsAbsentBlobHashIndex(variable Variable) {
	if t.unsatisfiable {
		return
	}
	if t.blobHashVariables == nil {
		t.blobHashVariables = make(map[Variable]bool)
	}
	if val, ok := t.blobHashVariables[variable]; !ok {
		t.blobHashVariables[variable] = false
	} else if val {
		t.markUnsatisfiable()
	}
}

func (t *TransactionContextGenerator) markUnsatisfiable() {
	t.unsatisfiable = true
	t.blobHashVariables = nil
}
