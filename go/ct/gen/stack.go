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
	sizes  []int
	values []valueConstraint
}

type valueConstraint struct {
	pos   int
	value U256
}

func (c *valueConstraint) Less(o *valueConstraint) bool {
	if c.pos != o.pos {
		return c.pos < o.pos
	}
	return c.value.Lt(o.value)
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
	v := valueConstraint{pos, value}
	if !slices.Contains(g.values, v) {
		g.values = append(g.values, v)
	}
}

func (g *StackGenerator) Generate(rnd *rand.Rand) (*st.Stack, error) {
	// Pick a size
	size := 0
	if len(g.sizes) > 1 {
		return nil, fmt.Errorf("%w, multiple conflicting sizes defined: %v", ErrUnsatisfiable, g.sizes)
	} else if len(g.sizes) == 1 {
		size = g.sizes[0]
		if size < 0 {
			return nil, fmt.Errorf("%w, can not produce stack with negative size %d", ErrUnsatisfiable, size)
		}
		if maxInValues := g.maxPositionInValues(); size <= maxInValues {
			return nil, fmt.Errorf("%w, set stack size %d too small for max position in value constraints %d", ErrUnsatisfiable, size, maxInValues)
		}
	} else {
		size = int(rnd.Int31n(5)) + g.maxPositionInValues() + 1
	}
	if size > 1024 {
		return nil, fmt.Errorf("%w, can not produce stack larger than 1024 elements %d", ErrUnsatisfiable, size)
	}

	stack := st.NewStackWithSize(size)
	stackMask := make([]bool, size)

	// Apply value constraints
	for _, value := range g.values {
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

func (g *StackGenerator) maxPositionInValues() int {
	max := -1
	for _, value := range g.values {
		if value.pos > max {
			max = value.pos
		}
	}
	return max
}

func (g *StackGenerator) Clone() *StackGenerator {
	return &StackGenerator{
		sizes:  slices.Clone(g.sizes),
		values: slices.Clone(g.values),
	}
}

func (g *StackGenerator) Restore(other *StackGenerator) {
	if g == other {
		return
	}
	g.sizes = slices.Clone(other.sizes)
	g.values = slices.Clone(other.values)
}

func (g *StackGenerator) String() string {
	var parts []string

	sort.Slice(g.sizes, func(i, j int) bool { return g.sizes[i] < g.sizes[j] })
	for _, size := range g.sizes {
		parts = append(parts, fmt.Sprintf("size=%d", size))
	}

	sort.Slice(g.values, func(i, j int) bool { return g.values[i].Less(&g.values[j]) })
	for _, value := range g.values {
		parts = append(parts, fmt.Sprintf("value[%d]=%v", value.pos, value.value))
	}

	return "{" + strings.Join(parts, ",") + "}"
}
