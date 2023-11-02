package gen

import (
	"fmt"
	"slices"
	"sort"
	"strings"

	"github.com/Fantom-foundation/Tosca/go/ct"
	"github.com/Fantom-foundation/Tosca/go/ct/st"
	"pgregory.net/rand"
)

// CodeGenerator is a utility class for generating Codes. See StateGenerator for
// more information on generators.
type CodeGenerator struct {
	// Constraints
	sizes    []int
	constOps []constOpConstraint
	varOps   []varOpConstraint
}

type constOpConstraint struct {
	pos int
	op  st.OpCode
}

type varOpConstraint struct {
	variable Variable
	op       st.OpCode
}

func NewCodeGenerator() *CodeGenerator {
	return &CodeGenerator{}
}

// SetSize adds a constraint on the size of the resulting code.
func (g *CodeGenerator) SetSize(size int) {
	if !slices.Contains(g.sizes, size) {
		g.sizes = append(g.sizes, size)
	}
}

// AddOperation adds a constraint placing an operation at a variable position.
func (g *CodeGenerator) AddOperation(v Variable, op st.OpCode) {
	g.varOps = append(g.varOps, varOpConstraint{variable: v, op: op})
}

// SetOperation fixes an operation to be placed at a given offset.
func (g *CodeGenerator) SetOperation(pos int, op st.OpCode) {
	g.constOps = append(g.constOps, constOpConstraint{pos: pos, op: op})
}

// Generate produces a Code instance satisfying the constraints set on this
// generator or returns ErrUnsatisfiable on conflicting constraints.
func (g *CodeGenerator) Generate(assignment Assignment, rnd *rand.Rand) (*st.Code, error) {
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
		for _, constraint := range g.constOps {
			if constraint.pos > minSize {
				minSize = constraint.pos + 1
			}
		}
		// Make extra space for worst-case size requirements of variable operation constraints.
		for _, constraint := range g.varOps {
			size := 1
			if st.PUSH1 <= constraint.op && constraint.op <= st.PUSH32 {
				size += int(constraint.op - st.PUSH1 + 1)
			}
			minSize += size
		}
		size = int(rnd.Int31n(int32(24576+1-minSize))) + minSize
	}

	// Convert the variable op-constraints into const constraints by fixing their position.
	ops := slices.Clone(g.constOps)
	sort.Slice(ops, func(i, j int) bool { return ops[i].pos < ops[j].pos })

	if len(g.varOps) > 0 {
		// Note: this solver for constraints with variables is sound but not complete.
		// There may be cases where variable positions may be assigned to fit into a
		// given code size which are missed due to a fragmentation of the code into
		// too small code sections, eliminating the possibility to fit in larger push
		// operations. However, this can only happen with sets of constraints with more
		// than one variable, which should not be needed.
		bound := map[Variable]st.OpCode{}

		// track used code positions
		used := map[int]bool{}
		markUsed := func(pos int, op st.OpCode) {
			used[pos] = true
			if st.PUSH1 <= op && op <= st.PUSH32 {
				width := int(op - st.PUSH1 + 1)
				for i := 0; i < width; i++ {
					used[pos+i+1] = true
				}
			}
		}
		fits := func(pos int, op st.OpCode) bool {
			if used[pos] {
				return false
			}
			if st.PUSH1 <= op && op <= st.PUSH32 {
				width := int(op - st.PUSH1 + 1)
				for i := 0; i < width; i++ {
					if used[pos+i+1] {
						return false
					}
				}
			}
			return true
		}

		for _, cur := range g.constOps {
			markUsed(cur.pos, cur.op)
		}
		for _, cur := range g.varOps {
			if op, found := bound[cur.variable]; found {
				if op != cur.op {
					return nil, fmt.Errorf(
						"%w, unable to satisfy conflicting constraint for op[%v]=%v and op[%v]=%v",
						ErrUnsatisfiable, cur.variable, op, cur.variable, cur.op,
					)
				}
				continue
			}
			bound[cur.variable] = cur.op

			// select a suitable code position for the current variable constraint
			pos := int(rnd.Int31n(int32(size)))
			startPos := pos
			for !fits(pos, cur.op) {
				pos++
				if pos >= size {
					pos = 0
				}
				if pos == startPos {
					return nil, fmt.Errorf(
						"%w, unable to fit operations in given code size",
						ErrUnsatisfiable,
					)
				}
			}
			markUsed(pos, cur.op)

			// Record and enforce the variable position.
			if assignment != nil {
				assignment[cur.variable] = ct.NewU256(uint64(pos))
			}
			ops = append(ops, constOpConstraint{pos, cur.op})
		}
		sort.Slice(ops, func(i, j int) bool { return ops[i].pos < ops[j].pos })
	}

	// Create the code and fill in content based on the operation constraints.
	code := make([]byte, size)

	// If there are no operation constraints producing random code is sufficient.
	if len(ops) == 0 {
		rnd.Read(code)
		return st.NewCode(code), nil
	}

	// If there are constraints we need to make sure that all operations are
	// indeed operations (not data) and that the operation code is correct.
	if last := ops[len(ops)-1].pos; last >= size {
		return nil, fmt.Errorf(
			"%w, operation constraint on position %d cannot be satisfied with a code length of %d",
			ErrUnsatisfiable, last, size,
		)
	}

	// Build random code incrementally.
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
			end := i + 1 + width
			if end > len(code) {
				end = len(code)
			}
			rnd.Read(code[i+1 : end])
			i += width
		}
	}

	return st.NewCode(code), nil
}

// Clone creates an independent copy of the generator in its current state.
// Future modifications are isolated from each other.
func (g *CodeGenerator) Clone() *CodeGenerator {
	return &CodeGenerator{
		sizes:    slices.Clone(g.sizes),
		constOps: slices.Clone(g.constOps),
		varOps:   slices.Clone(g.varOps),
	}
}

// Restore copies the state of the provided generator into this generator.
func (g *CodeGenerator) Restore(other *CodeGenerator) {
	if g == other {
		return
	}
	g.sizes = slices.Clone(other.sizes)
	g.constOps = slices.Clone(other.constOps)
}

func (g *CodeGenerator) String() string {
	var parts []string

	sort.Slice(g.sizes, func(i, j int) bool { return g.sizes[i] < g.sizes[j] })
	for _, size := range g.sizes {
		parts = append(parts, fmt.Sprintf("size=%d", size))
	}

	sort.Slice(g.constOps, func(i, j int) bool { return g.constOps[i].pos < g.constOps[j].pos })
	for _, op := range g.constOps {
		parts = append(parts, fmt.Sprintf("op[%v]=%v", op.pos, op.op))
	}

	sort.Slice(g.varOps, func(i, j int) bool { return g.varOps[i].variable < g.varOps[j].variable })
	for _, op := range g.varOps {
		parts = append(parts, fmt.Sprintf("op[%v]=%v", op.variable, op.op))
	}

	return "{" + strings.Join(parts, ",") + "}"
}
