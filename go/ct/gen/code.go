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

// CodeGenerator is a utility class for generating Codes. See StateGenerator for
// more information on generators.
type CodeGenerator struct {
	// Constraints
	ops []opConstraint
}

type opConstraint struct {
	pos int
	op  OpCode
}

func NewCodeGenerator() *CodeGenerator {
	return &CodeGenerator{}
}

// SetOperation fixes an operation to be placed at a given offset.
func (g *CodeGenerator) SetOperation(pos int, op OpCode) {
	g.ops = append(g.ops, opConstraint{pos: pos, op: op})
}

// Generate produces a Code instance satisfying the constraints set on this
// generator or returns ErrUnsatisfiable on conflicting constraints.
func (g *CodeGenerator) Generate(rnd *rand.Rand) (*st.Code, error) {
	// Pick a random size that is large enough for all operation constraints.
	size := 0
	for _, constraint := range g.ops {
		if constraint.pos > size {
			size = constraint.pos + 1
		}
	}
	size = int(rnd.Int31n(int32(24576+1-size))) + size

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
		op := INVALID
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
			limit := PUSH32
			if maxDataSize := nextFixedPosition - i - 1; maxDataSize < 32 {
				limit = OpCode(int(PUSH1) + maxDataSize - 1)
			}

			op = OpCode(rnd.Int())
			if limit < op && op <= PUSH32 {
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
		if PUSH1 <= op && op <= PUSH32 {
			width := int(op - PUSH1 + 1)
			rnd.Read(code[i+1 : i+1+width])
			i += width
		}
	}

	return st.NewCode(code), nil
}

// Clone creates an independent copy of the generator in its current state.
// Future modifications are isolated from each other.
func (g *CodeGenerator) Clone() *CodeGenerator {
	return &CodeGenerator{
		ops: slices.Clone(g.ops),
	}
}

// Restore copies the state of the provided generator into this generator.
func (g *CodeGenerator) Restore(other *CodeGenerator) {
	if g == other {
		return
	}
	g.ops = slices.Clone(other.ops)
}

func (g *CodeGenerator) String() string {
	var parts []string

	sort.Slice(g.ops, func(i, j int) bool { return g.ops[i].pos < g.ops[j].pos })
	for _, op := range g.ops {
		parts = append(parts, fmt.Sprintf("op[%d]=%v", op.pos, op.op))
	}

	return "{" + strings.Join(parts, ",") + "}"
}
