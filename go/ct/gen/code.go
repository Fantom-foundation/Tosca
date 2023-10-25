package gen

import (
	"fmt"
	"slices"
	"sort"
	"strings"

	"github.com/Fantom-foundation/Tosca/go/ct/st"
	"pgregory.net/rand"
)

// CodeGenerator is a utility class for generating Codes. It provides two
// capabilities:
//   - The specification of constraints and the generation of a code satisfying
//     those constraints or the identification of unsatisfiable constraints.
//     This constraint solver is sound and complete.
//   - Support for cloning a generator state and restoring it.
//
// The goal of the CodeGenerator is to produce high entropy results by setting
// all unconstrained degrees of freedom for the final code to random values.
type CodeGenerator struct {
	// Constraints
	sizes []int
	ops   []opConstraint
}

type opConstraint struct {
	pos int
	op  st.OpCode
}

// NewCodeGenerator creates a generator without any initial constraints.
func NewCodeGenerator() *CodeGenerator {
	return &CodeGenerator{}
}

// SetSize adds a constraint on the size of the resulting code. Multiple size
// constraints may be added, though conflicting constraints will result in an
// error signalling unsatisfiability during the generation process.
func (g *CodeGenerator) SetSize(size int) {
	if !slices.Contains(g.sizes, size) {
		g.sizes = append(g.sizes, size)
	}
}

// SetOperation fixes an operation to be placed at a given offset.
func (g *CodeGenerator) SetOperation(pos int, op st.OpCode) {
	g.ops = append(g.ops, opConstraint{pos: pos, op: op})
}

// Generate produces a Code instance satisfying the constraints set on this
// generator or returns an error indicating unsatisfiability.
func (g *CodeGenerator) Generate(rnd *rand.Rand) (*st.Code, error) {
	// Pick a size.
	if len(g.sizes) > 1 {
		return nil, fmt.Errorf("%w, multiple conflicting sizes defined: %v", ErrUnsatisfiable, g.sizes)
	}

	size := 0
	if len(g.sizes) == 1 {
		size = g.sizes[0]
		if size < 0 {
			return nil, fmt.Errorf("%w, can not produce code with negative size %d", ErrUnsatisfiable, size)
		}
	} else {
		// Pick a random size that is large enough for all operation constraints.
		minSize := 0
		for _, constraint := range g.ops {
			if constraint.pos > minSize {
				minSize = constraint.pos + 1
			}
		}
		size = int(rnd.Int31n(int32(24576+1-minSize))) + minSize
	}

	// Create the code and fill in content based on the operation constraints.
	code := make([]byte, size)

	// If there are no operation constraints producing random code is sufficient.
	if len(g.ops) == 0 {
		rnd.Read(code)
		return st.NewCode(code), nil
	}

	// If there are constraints we need to make sure that all operations are
	// indeed operations (not data) and that the operation code is correct.
	sort.Slice(g.ops, func(i, j int) bool { return g.ops[i].pos < g.ops[j].pos })
	if last := g.ops[len(g.ops)-1].pos; last >= size {
		return nil, fmt.Errorf(
			"%w, operation constraint on position %d cannot be satisfied with a code length of %d",
			ErrUnsatisfiable, last, size,
		)
	}

	// Build random code incrementally.
	ops := g.ops
	for i := 0; i < size; i++ {
		// Pick the next operation.
		op := st.INVALID
		nextFixedPosition := ops[0].pos
		if nextFixedPosition < i {
			return nil, fmt.Errorf(
				"%w, unable to satisfy op[%d]=%v constraint",
				ErrUnsatisfiable, ops[0].pos, ops[0].op,
			)
		}

		if i == nextFixedPosition {
			op = ops[0].op
			ops = ops[1:]
		} else {
			// Pick a random operation, but make sure to not overshoot to next position.
			limit := st.PUSH32
			if maxDataSize := nextFixedPosition - i - 1; maxDataSize < 32 {
				limit = st.OpCode(int(st.PUSH1) + maxDataSize - 1)
			}

			op = st.OpCode(rnd.Int())
			if limit < op && op <= st.PUSH32 {
				op = limit
			}
		}

		code[i] = byte(op)

		// If this was the last, fill the rest randomly.
		if len(ops) == 0 {
			rnd.Read(code[i+1:])
			break
		}

		// Fill data if needed and continue with the rest.
		if st.PUSH1 <= op && op <= st.PUSH32 {
			width := int(op - st.PUSH1 + 1)
			rnd.Read(code[i+1 : i+1+width])
			i += width
		}
	}

	return st.NewCode(code), nil
}

// Clone creates an independent copy of a generator in its current state. Future
// modifications are isolated from each other.
func (g *CodeGenerator) Clone() *CodeGenerator {
	return &CodeGenerator{
		sizes: slices.Clone(g.sizes),
		ops:   slices.Clone(g.ops),
	}
}

// Restore copies the state of the provided generator into this generator.
func (g *CodeGenerator) Restore(other *CodeGenerator) {
	if g == other {
		return
	}
	g.sizes = slices.Clone(other.sizes)
	g.ops = slices.Clone(other.ops)
}

func (g *CodeGenerator) String() string {
	var parts []string

	sort.Slice(g.sizes, func(i, j int) bool { return g.sizes[i] < g.sizes[j] })
	for _, size := range g.sizes {
		parts = append(parts, fmt.Sprintf("size=%d", size))
	}

	sort.Slice(g.ops, func(i, j int) bool { return g.ops[i].pos < g.ops[j].pos })
	for _, op := range g.ops {
		parts = append(parts, fmt.Sprintf("op[%d]=%v", op.pos, op.op))
	}

	return "{" + strings.Join(parts, ",") + "}"
}
