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
	"strings"

	"pgregory.net/rand"
)

type SelfDestructedGenerator struct {
	mustSelfDestructed    bool
	mustNotSelfDestructed bool
}

func NewSelfDestructedGenerator() *SelfDestructedGenerator {
	return &SelfDestructedGenerator{}
}

func (g *SelfDestructedGenerator) Clone() *SelfDestructedGenerator {
	return &SelfDestructedGenerator{
		mustSelfDestructed:    g.mustSelfDestructed,
		mustNotSelfDestructed: g.mustNotSelfDestructed,
	}
}

func (g *SelfDestructedGenerator) Restore(other *SelfDestructedGenerator) {
	if g == other {
		return
	}
	g.mustNotSelfDestructed = other.mustNotSelfDestructed
	g.mustSelfDestructed = other.mustSelfDestructed
}

func (g *SelfDestructedGenerator) MarkAsSelfDestructed() {
	g.mustSelfDestructed = true
}

func (g *SelfDestructedGenerator) MarkAsNotSelfDestructed() {
	g.mustNotSelfDestructed = true
}

func (g *SelfDestructedGenerator) String() string {
	var parts []string

	parts = append(parts, fmt.Sprintf("mustDestroy(%v)", g.mustSelfDestructed))
	parts = append(parts, fmt.Sprintf("mustNotDestroy(%v)", g.mustNotSelfDestructed))

	return "{" + strings.Join(parts, " ") + "}"
}

func (g *SelfDestructedGenerator) Generate(rnd *rand.Rand) (bool, error) {

	var hasSelfDestroyed bool
	if !g.mustSelfDestructed && !g.mustNotSelfDestructed {
		// random true/false
		hasSelfDestroyed = rnd.Int()%2 == 0
	} else if g.mustSelfDestructed && g.mustNotSelfDestructed {
		return false, ErrUnsatisfiable
	} else if g.mustSelfDestructed && !g.mustNotSelfDestructed {
		hasSelfDestroyed = true
	} else {
		hasSelfDestroyed = false
	}

	return hasSelfDestroyed, nil
}
