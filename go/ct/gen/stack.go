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
	sizes       []int
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
	return &StackGenerator{}
}

func (g *StackGenerator) SetSize(size int) {
	if !slices.Contains(g.sizes, size) {
		g.sizes = append(g.sizes, size)
	}
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
	size := 0
	if len(g.sizes) > 1 {
		return nil, fmt.Errorf("%w, multiple conflicting sizes defined: %v", ErrUnsatisfiable, g.sizes)
	} else if len(g.sizes) == 1 {
		size = g.sizes[0]
		if size < 0 {
			return nil, fmt.Errorf("%w, can not produce stack with negative size %d", ErrUnsatisfiable, size)
		}
		if maxInValues := maxPositionInValues(constraints); size <= maxInValues {
			return nil, fmt.Errorf("%w, set stack size %d too small for max position in value constraints %d", ErrUnsatisfiable, size, maxInValues)
		}
	} else {
		size = int(rnd.Int31n(5)) + maxPositionInValues(constraints) + 1
	}
	if size > 1024 {
		return nil, fmt.Errorf("%w, can not produce stack larger than 1024 elements %d", ErrUnsatisfiable, size)
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
		sizes:       slices.Clone(g.sizes),
		constValues: slices.Clone(g.constValues),
		varValues:   slices.Clone(g.varValues),
	}
}

func (g *StackGenerator) Restore(other *StackGenerator) {
	if g == other {
		return
	}
	g.sizes = slices.Clone(other.sizes)
	g.constValues = slices.Clone(other.constValues)
	g.varValues = slices.Clone(other.varValues)
}

func (g *StackGenerator) String() string {
	var parts []string

	sort.Slice(g.sizes, func(i, j int) bool { return g.sizes[i] < g.sizes[j] })
	for _, size := range g.sizes {
		parts = append(parts, fmt.Sprintf("size=%d", size))
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
