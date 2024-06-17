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
	"fmt"
	"slices"
	"strings"

	"golang.org/x/exp/maps"
	"pgregory.net/rand"

	"github.com/Fantom-foundation/Tosca/go/ct/common"
	"github.com/Fantom-foundation/Tosca/go/ct/st"
)

type TransientStorageGenerator struct {
	nonZeroConstraints map[Variable]bool
	unsatisfiable      bool
}

func NewTransientStorageGenerator() *TransientStorageGenerator {
	return &TransientStorageGenerator{nonZeroConstraints: map[Variable]bool{}}
}

// BindToNonZero binds a location in transient storage to a non-zero value.
func (t *TransientStorageGenerator) BindToNonZero(key Variable) {
	if value, exits := t.nonZeroConstraints[key]; exits {
		if !value {
			t.unsatisfiable = true
			return
		}
	} else {
		t.nonZeroConstraints[key] = true
	}
}

// BindToZero binds a location in transient storage to a zero value.
func (t *TransientStorageGenerator) BindToZero(key Variable) {
	if value, exits := t.nonZeroConstraints[key]; exits {
		if value {
			t.unsatisfiable = true
			return
		}
	} else {
		t.nonZeroConstraints[key] = false
	}
}

func (t *TransientStorageGenerator) Generate(assignment Assignment, rnd *rand.Rand) (*st.TransientStorage, error) {
	if t.unsatisfiable {
		return nil, fmt.Errorf("%w, conflicting set/unset constraints", ErrUnsatisfiable)
	}

	keysInUse := map[common.U256]bool{}
	for variable := range t.nonZeroConstraints {
		if pos, isBound := assignment[variable]; isBound {
			keysInUse[pos] = true
		}
	}

	getUnusedKey := func() common.U256 {
		for {
			key := common.RandU256(rnd)
			if _, isPresent := keysInUse[key]; !isPresent {
				keysInUse[key] = true
				return key
			}
		}
	}

	transientStorage := &st.TransientStorage{}

	// Process zero/nonZero constraints.
	for variable, nonZero := range t.nonZeroConstraints {

		// Check different assignments with the same key whether they conflict
		if _, isBound := assignment[variable]; isBound {
			for other, otherNonZero := range t.nonZeroConstraints {
				if assignment[variable] == assignment[other] && nonZero != otherNonZero {
					return nil, fmt.Errorf("%w, conflicting constraints %v and %v on the same location", ErrUnsatisfiable, nonZero, otherNonZero)
				}
			}
		}

		var key common.U256
		if pos, isBound := assignment[variable]; isBound {
			key = pos
			keysInUse[key] = true
		} else {
			key = getUnusedKey()
		}

		if nonZero {
			transientStorage.Set(key, common.RandU256(rnd))
		}
		assignment[variable] = key
	}

	// Random entries
	for i := 0; i < rnd.Intn(10); i++ {
		key := getUnusedKey()
		value := common.RandU256(rnd)
		transientStorage.Set(key, value)
	}

	return transientStorage, nil
}

func (t *TransientStorageGenerator) Clone() *TransientStorageGenerator {
	return &TransientStorageGenerator{
		nonZeroConstraints: maps.Clone(t.nonZeroConstraints),
		unsatisfiable:      t.unsatisfiable,
	}
}

func (t *TransientStorageGenerator) Restore(other *TransientStorageGenerator) {
	if t == other {
		return
	}
	t.nonZeroConstraints = maps.Clone(other.nonZeroConstraints)
	t.unsatisfiable = other.unsatisfiable
}

func (t *TransientStorageGenerator) String() string {
	if t.unsatisfiable {
		return "false"
	}

	var parts []string
	keys := maps.Keys(t.nonZeroConstraints)
	slices.Sort(keys)
	for _, key := range keys {
		if t.nonZeroConstraints[key] {
			parts = append(parts, fmt.Sprintf("transient[%v]≠0", key))
		} else {
			parts = append(parts, fmt.Sprintf("transient[%v]=0", key))
		}
	}
	return "{" + strings.Join(parts, "∧") + "}"
}
