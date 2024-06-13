//
// Copyright (c) 2024 Fantom Foundation
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at fantom.foundation/bsl11.
//
// Change Date: 2028-4-16
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by the GNU Lesser General Public Licence v3
//

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

type TransientGenerator struct {
	set           map[Variable]bool
	unsatisfiable bool
}

func NewTransientGenerator() *TransientGenerator {
	return &TransientGenerator{set: map[Variable]bool{}}
}

func (t *TransientGenerator) BindSet(key Variable) {
	if slices.Contains(maps.Keys(t.set), key) {
		if !t.set[key] {
			t.unsatisfiable = true
		}
	} else {
		t.set[key] = true
	}
}

func (t *TransientGenerator) BindNotSet(key Variable) {
	if slices.Contains(maps.Keys(t.set), key) {
		if t.set[key] {
			t.unsatisfiable = true
		}
	} else {
		t.set[key] = false
	}
}

func (t *TransientGenerator) Generate(assignment Assignment, rnd *rand.Rand) (*st.Transient, error) {
	if t.unsatisfiable {
		return nil, fmt.Errorf("%w, conflicting set/unset constraints", ErrUnsatisfiable)
	}

	keysInUse := map[common.U256]bool{}
	for variable := range t.set {
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

	transient := &st.Transient{}

	// Process set/unset constraints.
	for variable, set := range t.set {
		var key common.U256
		if pos, isBound := assignment[variable]; isBound {
			key = pos
		} else {
			key = getUnusedKey()
		}

		if set {
			transient.SetStorage(key, common.RandU256(rnd))
			assignment[variable] = key
		} else {
			transient.DeleteStorage(key)
			assignment[variable] = key
		}
	}

	// Random entries
	for i := 0; i < rnd.Intn(42); i++ {
		key := getUnusedKey()
		value := common.RandU256(rnd)

		transient.SetStorage(key, value)
	}

	return transient, nil
}

func (t *TransientGenerator) Clone() *TransientGenerator {
	return &TransientGenerator{
		set:           maps.Clone(t.set),
		unsatisfiable: t.unsatisfiable,
	}
}

func (t *TransientGenerator) Restore(other *TransientGenerator) {
	if t == other {
		return
	}
	t.set = maps.Clone(other.set)
	t.unsatisfiable = other.unsatisfiable
}

func (t *TransientGenerator) String() string {
	var parts []string

	for key, set := range t.set {
		if set {
			parts = append(parts, fmt.Sprintf("set(%v)", key))
		} else {
			parts = append(parts, fmt.Sprintf("notSet(%v)", key))
		}
	}
	return "{" + strings.Join(parts, ",") + "}"
}
