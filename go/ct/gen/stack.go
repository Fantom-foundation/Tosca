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
	sizeConstraints []sizeBounds
	constValues     []constValueConstraint
	varValues       []varValueConstraint
}

type sizeBounds struct{ min, max int }

func (a sizeBounds) Less(b sizeBounds) bool {
	if a.min != b.min {
		return a.min < b.min
	} else {
		return a.max < b.max
	}
}

func (s sizeBounds) String() string {
	if s.min == s.max {
		return fmt.Sprintf("%v", s.min)
	}
	return fmt.Sprintf("%v-%v", s.min, s.max)
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
	g.SetSizeBounds(size, size)
}

func (g *StackGenerator) SetSizeBounds(min, max int) {
	if min > max {
		min, max = max, min
	}
	s := sizeBounds{min, max}
	if !slices.Contains(g.sizeConstraints, s) {
		g.sizeConstraints = append(g.sizeConstraints, s)
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

	// Pick a size.
	var resultSize int
	if len(g.sizeConstraints) == 0 {
		resultSize = int(rnd.Int31n(5)) + maxPositionInValues(constraints) + 1
	} else {
		bounds := sizeBounds{maxPositionInValues(constraints) + 1, st.MaxStackSize + 1}
		for _, con := range g.sizeConstraints {
			if con.min > bounds.min {
				bounds.min = con.min
			}
			if con.max < bounds.max {
				bounds.max = con.max
			}
		}
		if bounds.min > bounds.max {
			return nil, fmt.Errorf("%w, conflicting stack size constraints defined: %v", ErrUnsatisfiable, g.sizeConstraints)
		}
		resultSize = int(rnd.Int31n(int32(bounds.max-bounds.min)+1)) + bounds.min
	}
	if resultSize < 0 {
		return nil, fmt.Errorf("%w, can not produce stack with negative size %d", ErrUnsatisfiable, resultSize)
	}
	if resultSize > st.MaxStackSize {
		return nil, fmt.Errorf("%w, can not produce stack larger than %d elements %d", ErrUnsatisfiable, st.MaxStackSize, resultSize)
	}

	stack := st.NewStackWithSize(resultSize)
	stackMask := make([]bool, resultSize)

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
		sizeConstraints: slices.Clone(g.sizeConstraints),
		constValues:     slices.Clone(g.constValues),
		varValues:       slices.Clone(g.varValues),
	}
}

func (g *StackGenerator) Restore(other *StackGenerator) {
	if g == other {
		return
	}
	g.sizeConstraints = slices.Clone(other.sizeConstraints)
	g.constValues = slices.Clone(other.constValues)
	g.varValues = slices.Clone(other.varValues)
}

func (g *StackGenerator) String() string {
	var parts []string

	sort.Slice(g.sizeConstraints, func(i, j int) bool { return g.sizeConstraints[i].Less(g.sizeConstraints[j]) })
	for _, bounds := range g.sizeConstraints {
		parts = append(parts, fmt.Sprintf("size=%v", bounds))
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
