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
	"pgregory.net/rand"
)

type SelfDestructedGenerator struct {
	mustBeSelfDestructed    bool
	mustNotBeSelfDestructed bool
}

func NewSelfDestructedGenerator() *SelfDestructedGenerator {
	return &SelfDestructedGenerator{}
}

func (g *SelfDestructedGenerator) Clone() *SelfDestructedGenerator {
	return &SelfDestructedGenerator{
		mustBeSelfDestructed:    g.mustBeSelfDestructed,
		mustNotBeSelfDestructed: g.mustNotBeSelfDestructed,
	}
}

func (g *SelfDestructedGenerator) Restore(other *SelfDestructedGenerator) {
	if g == other {
		return
	}
	*g = *other
}

func (g *SelfDestructedGenerator) MarkAsSelfDestructed() {
	g.mustBeSelfDestructed = true
}

func (g *SelfDestructedGenerator) MarkAsNotSelfDestructed() {
	g.mustNotBeSelfDestructed = true
}

func (g *SelfDestructedGenerator) String() string {
	if g.mustBeSelfDestructed && g.mustNotBeSelfDestructed {
		return "{false}" // unsatisfiable
	} else if !g.mustBeSelfDestructed && !g.mustNotBeSelfDestructed {
		return "{true}" // everything is valid
	} else if g.mustBeSelfDestructed && !g.mustNotBeSelfDestructed {
		return "{mustBeSelfDestructed}"
	}
	return "{mustNotBeSelfDestructed}"
}

func (g *SelfDestructedGenerator) Generate(rnd *rand.Rand) (bool, error) {

	var hasSelfDestroyed bool
	if !g.mustBeSelfDestructed && !g.mustNotBeSelfDestructed {
		// random true/false
		hasSelfDestroyed = rnd.Int()%2 == 0
	} else if g.mustBeSelfDestructed && g.mustNotBeSelfDestructed {
		return false, ErrUnsatisfiable
	} else if g.mustBeSelfDestructed && !g.mustNotBeSelfDestructed {
		hasSelfDestroyed = true
	} else {
		hasSelfDestroyed = false
	}

	return hasSelfDestroyed, nil
}
