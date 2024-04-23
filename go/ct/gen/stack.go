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

	"pgregory.net/rand"

	. "github.com/Fantom-foundation/Tosca/go/ct/common"
	"github.com/Fantom-foundation/Tosca/go/ct/st"
)

type StackGenerator struct {
	minSize     int
	maxSize     int
	constValues []constValueConstraint
	varValues   []varValueConstraint
}

type constValueConstraint struct {
	pos   int
	value U256
}

func (c *constValueConstraint) Less(o *constValueConstraint) bool {
	if c.pos != o.pos {
		return c.pos < o.pos
	}
	return c.value.Lt(o.value)
}

type varValueConstraint struct {
	pos      int
	variable Variable
}

func (c *varValueConstraint) Less(o *varValueConstraint) bool {
	if c.pos != o.pos {
		return c.pos < o.pos
	}
	return c.variable < o.variable
}

func NewStackGenerator() *StackGenerator {
	return &StackGenerator{
		minSize: 0,
		maxSize: st.MaxStackSize,
	}
}

func (g *StackGenerator) SetMinSize(size int) {
	if g.minSize < size {
		g.minSize = size
	}
}

func (g *StackGenerator) SetMaxSize(size int) {
	if g.maxSize > size {
		g.maxSize = size
	}
}

func (g *StackGenerator) SetSize(size int) {
	g.SetMinSize(size)
	g.SetMaxSize(size)
}

func (g *StackGenerator) SetValue(pos int, value U256) {
	v := constValueConstraint{pos, value}
	if !slices.Contains(g.constValues, v) {
		g.constValues = append(g.constValues, v)
	}
}

func (g *StackGenerator) BindValue(pos int, variable Variable) {
	v := varValueConstraint{pos, variable}
	if !slices.Contains(g.varValues, v) {
		g.varValues = append(g.varValues, v)
	}
}

func (g *StackGenerator) Generate(assignment Assignment, rnd *rand.Rand) (*st.Stack, error) {
	// convert variable constraints to constant constraints
	constraints := make([]constValueConstraint, len(g.constValues), len(g.constValues)+len(g.varValues))
	copy(constraints, g.constValues)

	for _, cur := range g.varValues {
		value, found := assignment[cur.variable]
		if !found {
			return nil, fmt.Errorf("%w, internal error, variable %v not bound", ErrUnboundVariable, cur.variable)
		}
		constraints = append(constraints, constValueConstraint{
			pos:   cur.pos,
			value: value,
		})
	}

	// Pick a size
	if g.minSize > g.maxSize {
		return nil, fmt.Errorf("%w, stack size constraint: %d <= size <= %d", ErrUnsatisfiable, g.minSize, g.maxSize)
	}
	maxInValues := maxPositionInValues(constraints)
	if maxInValues > 0 && maxInValues >= g.maxSize {
		return nil, fmt.Errorf("%w, stack size range [%d,%d] too small for max position in value constraints %d", ErrUnsatisfiable, g.minSize, g.maxSize, maxInValues)
	}
	if g.minSize <= maxInValues {
		g.minSize = maxInValues + 1
	}
	size := g.minSize
	if g.maxSize > g.minSize {
		size += int(rnd.Int31n(int32(g.maxSize - g.minSize)))
	}

	stack := st.NewStackWithSize(size)
	stackMask := make([]bool, size)

	// Apply value constraints
	for _, value := range constraints {
		if value.pos < 0 {
			return nil, fmt.Errorf("%w, cannot satisfy constraint value[%d]=%v", ErrUnsatisfiable, value.pos, value.value)
		}
		if stackMask[value.pos] {
			return nil, fmt.Errorf("%w, conflicting value constraints at position %d", ErrUnsatisfiable, value.pos)
		}
		stack.Set(value.pos, value.value)
		stackMask[value.pos] = true
	}

	// Fill in remaining slots
	for i, isSet := range stackMask {
		if !isSet {
			stack.Set(i, RandU256(rnd))
		}
	}

	return stack, nil
}

func maxPositionInValues(constraints []constValueConstraint) int {
	max := -1
	for _, value := range constraints {
		if value.pos > max {
			max = value.pos
		}
	}
	return max
}

func (g *StackGenerator) Clone() *StackGenerator {
	return &StackGenerator{
		minSize:     g.minSize,
		maxSize:     g.maxSize,
		constValues: slices.Clone(g.constValues),
		varValues:   slices.Clone(g.varValues),
	}
}

func (g *StackGenerator) Restore(other *StackGenerator) {
	if g == other {
		return
	}
	g.minSize = other.minSize
	g.maxSize = other.maxSize
	g.constValues = slices.Clone(other.constValues)
	g.varValues = slices.Clone(other.varValues)
}

func (g *StackGenerator) String() string {
	var parts []string

	if g.minSize != 0 || g.maxSize != st.MaxStackSize {
		parts = append(parts, fmt.Sprintf("%d≤size≤%d", g.minSize, g.maxSize))
	}

	sort.Slice(g.constValues, func(i, j int) bool { return g.constValues[i].Less(&g.constValues[j]) })
	for _, value := range g.constValues {
		parts = append(parts, fmt.Sprintf("value[%d]=%v", value.pos, value.value))
	}

	sort.Slice(g.varValues, func(i, j int) bool { return g.varValues[i].Less(&g.varValues[j]) })
	for _, value := range g.varValues {
		parts = append(parts, fmt.Sprintf("value[%d]=%v", value.pos, value.variable))
	}

	return "{" + strings.Join(parts, ",") + "}"
}
