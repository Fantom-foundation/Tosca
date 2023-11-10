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
	constOps             []constOpConstraint
	varOps               []varOpConstraint
	varIsCodeConstraints []varIsCodeConstraint
	varIsDataConstraints []varIsDataConstraint
}

type constOpConstraint struct {
	pos int
	op  OpCode
}

type varOpConstraint struct {
	variable Variable
	op       OpCode
}

type varIsCodeConstraint struct {
	variable Variable
}

type varIsDataConstraint struct {
	variable Variable
}

func NewCodeGenerator() *CodeGenerator {
	return &CodeGenerator{}
}

// SetOperation fixes an operation to be placed at a given offset.
func (g *CodeGenerator) SetOperation(pos int, op OpCode) {
	g.constOps = append(g.constOps, constOpConstraint{pos: pos, op: op})
}

// AddOperation adds a constraint placing an operation at a variable position.
func (g *CodeGenerator) AddOperation(v Variable, op OpCode) {
	g.varOps = append(g.varOps, varOpConstraint{variable: v, op: op})
}

// AddIsCode adds a constraint such that the generator will produce a code
// segment where the byte at v is an instruction (not data).
func (g *CodeGenerator) AddIsCode(v Variable) {
	g.varIsCodeConstraints = append(g.varIsCodeConstraints, varIsCodeConstraint{v})
}

// AddIsData adds a constraint such that the generator will produce a code
// segment where the byte at v is data.
func (g *CodeGenerator) AddIsData(v Variable) {
	g.varIsDataConstraints = append(g.varIsDataConstraints, varIsDataConstraint{v})
}

// Generate produces a Code instance satisfying the constraints set on this
// generator or returns ErrUnsatisfiable on conflicting constraints. Updates the
// given assignment along the way.
func (g *CodeGenerator) Generate(assignment Assignment, rnd *rand.Rand) (*st.Code, error) {
	// Pick a random size that is large enough for all const constraints.
	minSize := 0
	for _, constraint := range g.constOps {
		if constraint.pos > minSize {
			minSize = constraint.pos + 1
		}
	}
	// Make extra space for worst-case size requirements of variable operation
	// constraints.
	for _, constraint := range g.varOps {
		size := 1
		if PUSH1 <= constraint.op && constraint.op <= PUSH32 {
			size += int(constraint.op - PUSH1 + 1)
		}
		minSize += size
	}

	size := int(rnd.Int31n(int32(24576+1-minSize))) + minSize

	ops, err := g.solveVarConstraints(assignment, rnd, size)
	if err != nil {
		return nil, err
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
	sort.Slice(ops, func(i, j int) bool { return ops[i].pos < ops[j].pos })
	if last := ops[len(ops)-1].pos; last >= size {
		return nil, fmt.Errorf(
			"%w, operation constraint on position %d cannot be satisfied with a code length of %d",
			ErrUnsatisfiable, last, size,
		)
	}

	// Build random code incrementally.
	for i := 0; i < size; i++ {
		// Pick the next operation.
		op := INVALID
		nextFixedPosition := ops[0].pos
		if nextFixedPosition < i {
			return nil, fmt.Errorf(
				"%w, unable to satisfy op[%d]=%v constraint",
				ErrUnsatisfiable, ops[0].pos, ops[0].op)
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
			end := i + 1 + width
			if end > len(code) {
				end = len(code)
			}
			rnd.Read(code[i+1 : end])
			i += width
		}
	}

	// After filling in the code, we expect all ops to have been processed.
	if len(ops) > 0 {
		return nil, fmt.Errorf(
			"%w, unable to satisfy last %v constraint",
			ErrUnsatisfiable, len(ops))
	}

	return st.NewCode(code), nil
}

// solveVarConstraints converts the variable op-constraints into const
// constraints by fixing their position.
func (g *CodeGenerator) solveVarConstraints(assignment Assignment, rnd *rand.Rand, codeSize int) ([]constOpConstraint, error) {
	ops := slices.Clone(g.constOps)
	sort.Slice(ops, func(i, j int) bool { return ops[i].pos < ops[j].pos })

	if len(g.varOps) == 0 && len(g.varIsCodeConstraints) == 0 && len(g.varIsDataConstraints) == 0 {
		return ops, nil
	}

	// Note: this solver for constraints with variables is sound but not
	// complete. There may be cases where variable positions may be assigned to
	// fit into a given code size which are missed due to a fragmentation of the
	// code into too small code sections, eliminating the possibility to fit in
	// larger push operations. However, this can only happen with sets of
	// constraints with more than one variable, which should not be needed.
	bound := map[Variable]OpCode{}

	// track used code positions
	used := map[int]int{}

	const isUnused = 0
	const isCode = 1
	const isData = 2

	markUsed := func(pos int, op OpCode) {
		used[pos] = isCode
		if PUSH1 <= op && op <= PUSH32 {
			width := int(op - PUSH1 + 1)
			for i := 0; i < width; i++ {
				used[pos+i+1] = isData
			}
		}
	}
	fits := func(pos int, op OpCode) bool {
		if used[pos] != isUnused {
			return false
		}
		if PUSH1 <= op && op <= PUSH32 {
			width := int(op - PUSH1 + 1)
			for i := 0; i < width; i++ {
				if used[pos+i+1] != isUnused {
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
		pos := int(rnd.Int31n(int32(codeSize)))
		startPos := pos
		for !fits(pos, cur.op) {
			pos++
			if pos >= codeSize {
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
			assignment[cur.variable] = NewU256(uint64(pos))
		}
		ops = append(ops, constOpConstraint{pos, cur.op})
	}

	for _, cur := range g.varIsCodeConstraints {
		// All assigned variables already point to code by processing g.varOps.
		if _, isAssigned := assignment[cur.variable]; isAssigned {
			continue
		}

		// For the remaining variables, find a position and either populate an
		// unused slot, or use a slot with code in it. Code position 0 is
		// guaranteed to be unused or contain code.
		pos := int(rnd.Int31n(int32(codeSize)))
		startPos := pos
		for used[pos] == isData {
			pos++
			if pos >= codeSize {
				pos = 0
			}
			if pos == startPos {
				return nil, fmt.Errorf("%w, unable to fit isCode constraint in given code size", ErrUnsatisfiable)
			}
		}

		// Record and enforce the variable position.
		if assignment != nil {
			assignment[cur.variable] = NewU256(uint64(pos))
		}

		if used[pos] == isUnused {
			// Using JUMPDEST as *some* code instruction.
			ops = append(ops, constOpConstraint{pos, JUMPDEST})
			markUsed(pos, JUMPDEST)
		}
	}

	for _, cur := range g.varIsDataConstraints {
		// All assigned variables already point to code!
		if _, isAssigned := assignment[cur.variable]; isAssigned {
			return nil, fmt.Errorf("%w, unable to satisfy isData[%v]", ErrUnsatisfiable, cur.variable)
		}

		// For the remaining variables, find a position and either populate an
		// unused slot, or use a slot with data in it.
		pos := int(rnd.Int31n(int32(codeSize)))
		startPos := pos
		for used[pos] != isData && !(used[pos] == isUnused && fits(pos, PUSH1)) {
			pos++
			if pos >= codeSize {
				pos = 0
			}
			if pos == startPos {
				return nil, fmt.Errorf("%w, unable to fit isData constraint in given code size", ErrUnsatisfiable)
			}
		}

		if used[pos] == isUnused {
			ops = append(ops, constOpConstraint{pos, PUSH1})
			markUsed(pos, PUSH1)
			pos++
		}

		// Record and enforce the variable position.
		if assignment != nil {
			assignment[cur.variable] = NewU256(uint64(pos))
		}
	}

	return ops, nil
}

// Clone creates an independent copy of the generator in its current state.
// Future modifications are isolated from each other.
func (g *CodeGenerator) Clone() *CodeGenerator {
	return &CodeGenerator{
		constOps:             slices.Clone(g.constOps),
		varOps:               slices.Clone(g.varOps),
		varIsCodeConstraints: slices.Clone(g.varIsCodeConstraints),
		varIsDataConstraints: slices.Clone(g.varIsDataConstraints),
	}
}

// Restore copies the state of the provided generator into this generator.
func (g *CodeGenerator) Restore(other *CodeGenerator) {
	if g == other {
		return
	}
	g.constOps = slices.Clone(other.constOps)
	g.varOps = slices.Clone(other.varOps)
	g.varIsCodeConstraints = slices.Clone(other.varIsCodeConstraints)
	g.varIsDataConstraints = slices.Clone(other.varIsDataConstraints)
}

func (g *CodeGenerator) String() string {
	var parts []string

	sort.Slice(g.constOps, func(i, j int) bool { return g.constOps[i].pos < g.constOps[j].pos })
	for _, op := range g.constOps {
		parts = append(parts, fmt.Sprintf("op[%v]=%v", op.pos, op.op))
	}

	sort.Slice(g.varOps, func(i, j int) bool { return g.varOps[i].variable < g.varOps[j].variable })
	for _, op := range g.varOps {
		parts = append(parts, fmt.Sprintf("op[%v]=%v", op.variable, op.op))
	}

	sort.Slice(g.varIsCodeConstraints, func(i, j int) bool { return g.varIsCodeConstraints[i].variable < g.varIsCodeConstraints[j].variable })
	for _, con := range g.varIsCodeConstraints {
		parts = append(parts, fmt.Sprintf("isCode[%v]", con.variable))
	}

	sort.Slice(g.varIsDataConstraints, func(i, j int) bool { return g.varIsDataConstraints[i].variable < g.varIsDataConstraints[j].variable })
	for _, con := range g.varIsDataConstraints {
		parts = append(parts, fmt.Sprintf("isData[%v]", con.variable))
	}

	return "{" + strings.Join(parts, ",") + "}"
}
