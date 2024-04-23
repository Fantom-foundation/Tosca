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
	"sort"
	"strings"

	. "github.com/Fantom-foundation/Tosca/go/ct/common"
	"github.com/Fantom-foundation/Tosca/go/vm"
	"pgregory.net/rand"
)

type selfDestructedConstraint struct {
	address   Variable
	destroyed bool
}

func (a *selfDestructedConstraint) Less(b *selfDestructedConstraint) bool {
	if a.address != b.address {
		return a.address < b.address
	}
	return a.destroyed != b.destroyed && a.destroyed
}

type SelfDestructedGenerator struct {
	hasSelfDestructed []selfDestructedConstraint
}

func NewSelfDestructedGenerator() *SelfDestructedGenerator {
	return &SelfDestructedGenerator{}
}

func (g *SelfDestructedGenerator) Clone() *SelfDestructedGenerator {
	return &SelfDestructedGenerator{
		hasSelfDestructed: slices.Clone(g.hasSelfDestructed),
	}
}

func (g *SelfDestructedGenerator) Restore(other *SelfDestructedGenerator) {
	if g == other {
		return
	}
	g.hasSelfDestructed = slices.Clone(other.hasSelfDestructed)
}

func (g *SelfDestructedGenerator) BindHasSelfDestructed(address Variable) {
	v := selfDestructedConstraint{address, true}
	if !slices.Contains(g.hasSelfDestructed, v) {
		g.hasSelfDestructed = append(g.hasSelfDestructed, v)
	}

}

func (g *SelfDestructedGenerator) BindHasNotSelfDestructed(address Variable) {
	v := selfDestructedConstraint{address, false}
	if !slices.Contains(g.hasSelfDestructed, v) {
		g.hasSelfDestructed = append(g.hasSelfDestructed, v)
	}
}

func (g *SelfDestructedGenerator) String() string {
	var parts []string

	sort.Slice(g.hasSelfDestructed, func(i, j int) bool { return g.hasSelfDestructed[i].Less(&g.hasSelfDestructed[j]) })
	for _, con := range g.hasSelfDestructed {
		if con.destroyed {
			parts = append(parts, fmt.Sprintf("destructed(%v)", con.address))
		} else {
			parts = append(parts, fmt.Sprintf("notDestructed(%v)", con.address))
		}
	}

	return "{" + strings.Join(parts, ",") + "}"
}

func (g *SelfDestructedGenerator) Generate(assignment Assignment, rnd *rand.Rand) (map[vm.Address]struct{}, error) {
	// Check for conflicts among has/hasnot self destructed constraints.
	sort.Slice(g.hasSelfDestructed, func(i, j int) bool { return g.hasSelfDestructed[i].Less(&g.hasSelfDestructed[j]) })
	for i := 0; i < len(g.hasSelfDestructed)-1; i++ {
		a, b := g.hasSelfDestructed[i], g.hasSelfDestructed[i+1]
		if a.address == b.address && a.destroyed != b.destroyed {
			return nil, fmt.Errorf("%w, address %v has conflicting selfdestructed constraints", ErrUnsatisfiable, a.address)
		}
	}

	// When handling unbound variables, we need to generate an unused address for
	// them. We therefore track which addresses have already been used.
	addressesInUse := map[vm.Address]bool{}
	for _, con := range g.hasSelfDestructed {
		if pos, isBound := assignment[con.address]; isBound {
			addressesInUse[NewAddress(pos)] = true
		}
	}
	getUnusedAddress := func() vm.Address {
		for {
			address, _ := RandAddress(rnd)

			if _, isPresent := addressesInUse[address]; !isPresent {
				addressesInUse[address] = true
				return address
			}
		}
	}

	getAddress := func(v Variable) vm.Address {
		addr, isBound := assignment[v]
		address := NewAddress(addr)
		if !isBound {
			address = getUnusedAddress()
			assignment[v] = AddressToU256(address)
		}

		return address
	}

	hasSelfDestrBuilder := make(map[vm.Address]struct{})

	// Process has/has not self destructed constraints.
	for _, con := range g.hasSelfDestructed {
		address := getAddress(con.address)
		if con.destroyed {
			hasSelfDestrBuilder[address] = struct{}{}
		} else {
			delete(hasSelfDestrBuilder, address)
		}
	}

	// Some random entries.
	for i, max := 0, rnd.Intn(5); i < max; i++ {
		address := getUnusedAddress()
		hasSelfDestrBuilder[address] = struct{}{}
	}

	return hasSelfDestrBuilder, nil
}
