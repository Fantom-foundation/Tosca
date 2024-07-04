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
	"sort"
	"strings"

	"golang.org/x/exp/slices"
	"pgregory.net/rand"

	. "github.com/Fantom-foundation/Tosca/go/ct/common"
	"github.com/Fantom-foundation/Tosca/go/ct/st"
	"github.com/Fantom-foundation/Tosca/go/tosca"
)

type StorageGenerator struct {
	cfg      []storageConfigConstraint
	warmCold []warmColdConstraint
}

type storageConfigConstraint struct {
	config   tosca.StorageStatus
	key      Variable
	newValue Variable
}

func (a *storageConfigConstraint) Less(b *storageConfigConstraint) bool {
	if a.config != b.config {
		return a.config < b.config
	}
	if a.key != b.key {
		return a.key < b.key
	}
	return a.newValue < b.newValue
}

// Check checks if the given storage configuration (org,cur,new) corresponds to
// the wanted config.
func CheckStorageStatusConfig(config tosca.StorageStatus, org, cur, new U256) bool {
	return config == tosca.GetStorageStatus(
		tosca.Word(org.Bytes32be()),
		tosca.Word(cur.Bytes32be()),
		tosca.Word(new.Bytes32be()),
	)
}

func NewValueMustBeZero(config tosca.StorageStatus) bool {
	return config == tosca.StorageAddedDeleted ||
		config == tosca.StorageDeleted ||
		config == tosca.StorageModifiedDeleted
}

func NewValueMustNotBeZero(config tosca.StorageStatus) bool {
	return config == tosca.StorageAssigned ||
		config == tosca.StorageAdded ||
		config == tosca.StorageDeletedRestored ||
		config == tosca.StorageDeletedAdded ||
		config == tosca.StorageModified ||
		config == tosca.StorageModifiedRestored
}

type warmColdConstraint struct {
	key  Variable
	warm bool
}

func (a *warmColdConstraint) Less(b *warmColdConstraint) bool {
	if a.key != b.key {
		return a.key < b.key
	}
	return a.warm != b.warm && a.warm
}

func NewStorageGenerator() *StorageGenerator {
	return &StorageGenerator{}
}

func (g *StorageGenerator) BindConfiguration(config tosca.StorageStatus, key, newValue Variable) {
	v := storageConfigConstraint{config, key, newValue}
	if !slices.Contains(g.cfg, v) {
		g.cfg = append(g.cfg, v)
	}
}

func (g *StorageGenerator) BindWarm(key Variable) {
	v := warmColdConstraint{key, true}
	if !slices.Contains(g.warmCold, v) {
		g.warmCold = append(g.warmCold, v)
	}
}

func (g *StorageGenerator) BindCold(key Variable) {
	v := warmColdConstraint{key, false}
	if !slices.Contains(g.warmCold, v) {
		g.warmCold = append(g.warmCold, v)
	}
}

func (g *StorageGenerator) Generate(assignment Assignment, rnd *rand.Rand) (*st.Storage, error) {
	// Check for conflicts among storage configuration constraints
	sort.Slice(g.cfg, func(i, j int) bool { return g.cfg[i].Less(&g.cfg[j]) })
	for i := 0; i < len(g.cfg)-1; i++ {
		a, b := g.cfg[i], g.cfg[i+1]
		if a.key == b.key && (a.config != b.config || a.newValue != b.newValue) {
			return nil, fmt.Errorf("%w, key %v conflicting storage configuration", ErrUnsatisfiable, a.key)
		}
	}

	// Check for conflicts among warm/cold constraints.
	sort.Slice(g.warmCold, func(i, j int) bool { return g.warmCold[i].Less(&g.warmCold[j]) })
	for i := 0; i < len(g.warmCold)-1; i++ {
		a, b := g.warmCold[i], g.warmCold[i+1]
		if a.key == b.key && a.warm != b.warm {
			return nil, fmt.Errorf("%w, key %v conflicting warm/cold constraints", ErrUnsatisfiable, a.key)
		}
	}

	// When handling unbound variables, we need to generate an unused key for
	// them. We therefore track which keys have already been used.
	keysInUse := map[U256]bool{}
	for _, con := range g.cfg {
		if key, isBound := assignment[con.key]; isBound {
			keysInUse[key] = true
		}
	}
	for _, con := range g.warmCold {
		if key, isBound := assignment[con.key]; isBound {
			keysInUse[key] = true
		}
	}
	getUnusedKey := func() U256 {
		for {
			key := RandU256(rnd)
			if _, isPresent := keysInUse[key]; !isPresent {
				keysInUse[key] = true
				return key
			}
		}
	}

	getKey := func(v Variable) U256 {
		key, isBound := assignment[v]
		if !isBound {
			key = getUnusedKey()
			assignment[v] = key // update assignment
		}
		return key
	}
	randValueButNot := func(exclusive ...U256) U256 {
		for {
			value := RandU256(rnd)
			if !slices.Contains(exclusive, value) {
				return value
			}
		}
	}

	builder := st.NewStorageBuilder()

	// Process storage configuration constraints.
	for _, con := range g.cfg {
		key := getKey(con.key)

		newValue, isBound := assignment[con.newValue]
		if isBound {
			// Check for conflict!
			if (newValue.IsZero() && NewValueMustNotBeZero(con.config)) ||
				(!newValue.IsZero() && NewValueMustBeZero(con.config)) {
				return nil, fmt.Errorf("%w, assignment for %v is incompatible with storage config %v", ErrUnsatisfiable, con.newValue, con.config)
			}
		} else {
			// Pick a suitable newValue.
			if NewValueMustBeZero(con.config) {
				newValue = NewU256(0)
			} else if NewValueMustNotBeZero(con.config) {
				newValue = randValueButNot(NewU256(0))
			} else {
				if rnd.Intn(5) == 0 {
					newValue = NewU256(0)
				} else {
					newValue = RandU256(rnd)
				}
			}
			assignment[con.newValue] = newValue // update assignment
		}

		orgValue, curValue := NewU256(0), NewU256(0)
		switch con.config {
		case tosca.StorageAdded:
			orgValue, curValue = NewU256(0), NewU256(0)
		case tosca.StorageAddedDeleted:
			curValue = randValueButNot(NewU256(0))
		case tosca.StorageDeletedRestored:
			orgValue = newValue
		case tosca.StorageDeletedAdded:
			orgValue = randValueButNot(NewU256(0), newValue)
		case tosca.StorageDeleted:
			orgValue = randValueButNot(NewU256(0))
			curValue = orgValue
		case tosca.StorageModified:
			orgValue = randValueButNot(NewU256(0), newValue)
			curValue = orgValue
		case tosca.StorageModifiedDeleted:
			orgValue = randValueButNot(NewU256(0))
			curValue = randValueButNot(NewU256(0), orgValue)
		case tosca.StorageModifiedRestored:
			orgValue = newValue
			curValue = randValueButNot(NewU256(0), orgValue)
		case tosca.StorageAssigned:
			// Technically, there are more configurations than this one which
			// satisfy StorageAssigned; but this should do for now.
			orgValue = randValueButNot(NewU256(0), newValue)
			curValue = randValueButNot(NewU256(0), orgValue, newValue)
		}

		builder.SetOriginal(key, orgValue)
		builder.SetCurrent(key, curValue)
	}

	// Process warm/cold constraints.
	for _, con := range g.warmCold {
		key := getKey(con.key)
		if !builder.IsInOriginal(key) {
			builder.SetOriginal(key, RandU256(rnd))
			builder.SetCurrent(key, RandU256(rnd))
		}
		builder.SetWarm(key, con.warm)
	}

	// Also, add some random entries.
	for i, max := 0, rnd.Intn(5); i < max; i++ {
		key := getUnusedKey()
		builder.SetOriginal(key, RandU256(rnd))
		builder.SetCurrent(key, RandU256(rnd))
		builder.SetWarm(key, rnd.Intn(2) == 1)
	}

	return builder.Build(), nil
}

func (g *StorageGenerator) Clone() *StorageGenerator {
	return &StorageGenerator{
		cfg:      slices.Clone(g.cfg),
		warmCold: slices.Clone(g.warmCold),
	}
}

func (g *StorageGenerator) Restore(other *StorageGenerator) {
	if g == other {
		return
	}
	g.cfg = slices.Clone(other.cfg)
	g.warmCold = slices.Clone(other.warmCold)
}

func (g *StorageGenerator) String() string {
	var parts []string

	sort.Slice(g.cfg, func(i, j int) bool { return g.cfg[i].Less(&g.cfg[j]) })
	for _, con := range g.cfg {
		parts = append(parts, fmt.Sprintf("cfg[%v]=(%v,%v)", con.key, con.config, con.newValue))
	}

	sort.Slice(g.warmCold, func(i, j int) bool { return g.warmCold[i].Less(&g.warmCold[j]) })
	for _, con := range g.warmCold {
		if con.warm {
			parts = append(parts, fmt.Sprintf("warm(%v)", con.key))
		} else {
			parts = append(parts, fmt.Sprintf("cold(%v)", con.key))
		}
	}

	return "{" + strings.Join(parts, ",") + "}"
}
