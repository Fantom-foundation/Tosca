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
	"fmt"
	"slices"
	"sort"
	"strings"

	"pgregory.net/rand"

	. "github.com/Fantom-foundation/Tosca/go/ct/common"
	"github.com/Fantom-foundation/Tosca/go/ct/st"
	"github.com/Fantom-foundation/Tosca/go/tosca/vm"
)

// CodeGenerator is a utility class for generating Codes. See StateGenerator for
// more information on generators.
type CodeGenerator struct {
	constOps             []constOpConstraint
	varOps               []varOpConstraint
	varIsCodeConstraints []varIsCodeConstraint
	varIsDataConstraints []varIsDataConstraint

	// testing only
	codeSize *int
}

type constOpConstraint struct {
	pos int
	op  vm.OpCode
}

type varOpConstraint struct {
	variable Variable
	op       vm.OpCode
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
func (g *CodeGenerator) SetOperation(pos int, op vm.OpCode) {
	g.constOps = append(g.constOps, constOpConstraint{pos: pos, op: op})
}

// AddOperation adds a constraint placing an operation at a variable position.
func (g *CodeGenerator) AddOperation(v Variable, op vm.OpCode) {
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
	var err error

	// Convert operation constraints referencing bound variables to constant constraints.
	ops := slices.Clone(g.constOps)
	varOps := make([]varOpConstraint, 0, len(g.varOps))
	for _, cur := range g.varOps {
		if value, found := assignment[cur.variable]; found {
			if !value.IsUint64() || value.Uint64() > st.MaxCodeSize {
				return nil, fmt.Errorf("%w: unable to constrain code at position %v", ErrUnsatisfiable, value)
			}
			ops = append(ops, constOpConstraint{pos: int(value.Uint64()), op: cur.op})
		} else {
			varOps = append(varOps, cur)
		}
	}

	// Pick a random size that is large enough for all const constraints.
	minSize := 0
	for _, constraint := range ops {
		if constraint.pos > minSize {
			minSize = constraint.pos + 1
		}
	}

	// Make extra space for worst-case size requirements of variable operation
	// constraints.
	for _, constraint := range varOps {
		size := 1
		if vm.PUSH1 <= constraint.op && constraint.op <= vm.PUSH32 {
			size += int(constraint.op - vm.PUSH1 + 1)
		}
		minSize += size
	}

	// Make enough space to host all the different opCodes used in condition.
	opCount := make(map[vm.OpCode]bool)
	for _, constraint := range varOps {
		opCount[constraint.op] = true
	}
	for _, constraint := range ops {
		opCount[constraint.op] = true
	}
	minSize = max(minSize, len(opCount))

	// If there are any variables that need to be bound to code, there must be at
	// least one instruction in the resulting code.
	if minSize == 0 && len(g.varIsCodeConstraints) > 0 {
		minSize = 1
	}

	// if there are data constraints, we need at least 2 bytes,
	// one for an op, and the other for data.
	if len(g.varIsDataConstraints) > 0 && minSize < 2 {
		minSize = 2
	}

	var size int
	if g.codeSize != nil {
		if *g.codeSize < minSize {
			return nil, fmt.Errorf("%w, fixed code size %d is too small for constraints", ErrUnsatisfiable, *g.codeSize)
		}
		size = *g.codeSize
	} else {
		// We use an exponential distribution for the code size here since long codes
		// extend the runtime but are expected to reveal limited extra code coverage.
		const expectedSize float64 = 200
		size = int(rnd.ExpFloat64()/(1/expectedSize)) + minSize
		if size > st.MaxCodeSize {
			size = st.MaxCodeSize
		}
	}

	// Solve variable constraints. constOpConstraints are generated, the
	// assignment is updated.
	if len(varOps)+len(g.varIsCodeConstraints)+len(g.varIsDataConstraints) != 0 {
		solver := newVarCodeConstraintSolver(size, ops, assignment, rnd)
		ops, err = solver.solve(varOps, g.varIsCodeConstraints, g.varIsDataConstraints)
		if err != nil {
			return nil, err
		}
	}

	// Create the code and fill in content based on the operation constraints.
	code := make([]byte, size)

	// If there are no operation constraints producing random code is sufficient.
	if len(ops) == 0 {
		_, _ = rnd.Read(code) // rnd.Read never returns an error
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
		op := vm.INVALID
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
			limit := vm.PUSH32
			if maxDataSize := nextFixedPosition - i - 1; maxDataSize < 32 {
				limit = vm.OpCode(int(vm.PUSH1) + maxDataSize - 1)
			}

			op = vm.OpCode(rnd.Int())
			if limit < op && op <= vm.PUSH32 {
				op = limit
			}
		}

		code[i] = byte(op)

		// If this was the last, fill the rest randomly.
		if len(ops) == 0 {
			_, _ = rnd.Read(code[i+1:]) // rnd.Read never returns an error
			break
		}

		// Fill data if needed and continue with the rest.
		if vm.PUSH1 <= op && op <= vm.PUSH32 {
			width := int(op - vm.PUSH1 + 1)
			end := i + 1 + width
			if end > len(code) {
				end = len(code)
			}
			_, _ = rnd.Read(code[i+1 : end]) // rnd.Read never returns an error
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

////////////////////////////////////////////////////////////
// Variable Constraint Solver

// varCodeConstraintSolver is a solver for the CodeGenerator's variable
// constraints (including isCode and isData constraints).
//
// Note: this solver for constraints with variables is sound but not complete.
// There may be cases where variable positions may be assigned to fit into a
// given code size which are missed due to a fragmentation of the code into too
// small code sections, eliminating the possibility to fit in larger push
// operations. However, this can only happen with sets of constraints with more
// than one variable, which should not be needed.
type varCodeConstraintSolver struct {
	codeSize      int
	ops           []constOpConstraint
	assignment    Assignment
	usedPositions map[int]Used
	rnd           *rand.Rand
}

type Used int

const (
	isUnused = 0
	isCode   = 1
	isData   = 2
)

// newVarCodeConstraintSolver creates a solver for the given codeSize. The
// provided constOps are honored. The given assignment is updated in-place.
func newVarCodeConstraintSolver(codeSize int, constOps []constOpConstraint, assignment Assignment, rnd *rand.Rand) varCodeConstraintSolver {
	solver := varCodeConstraintSolver{
		codeSize:      codeSize,
		ops:           constOps,
		assignment:    assignment,
		usedPositions: make(map[int]Used),
		rnd:           rnd,
	}
	for _, con := range constOps {
		solver.markUsed(con.pos, con.op)
	}
	return solver
}

// solve is the entry point for varCodeConstraintSolver, other functions are
// considered internal.
func (s *varCodeConstraintSolver) solve(
	varOps []varOpConstraint,
	varIsCodeConstraints []varIsCodeConstraint,
	varIsDataConstraints []varIsDataConstraint) ([]constOpConstraint, error) {

	err := s.solveVarOps(varOps)
	if err != nil {
		return nil, err
	}

	err = s.solveIsCode(varIsCodeConstraints)
	if err != nil {
		return nil, err
	}

	err = s.solveIsData(varIsDataConstraints)
	if err != nil {
		return nil, err
	}

	return s.ops, nil
}

func (s *varCodeConstraintSolver) markUsed(pos int, op vm.OpCode) {
	s.usedPositions[pos] = isCode
	for i := 1; i <= op.Width()-1; i++ {
		s.usedPositions[pos+i] = isData
	}
}

// fits returns true iff the op can be placed at pos.
func (s *varCodeConstraintSolver) fits(pos int, op vm.OpCode) bool {
	if op.Width() > s.codeSize-pos {
		return false
	}
	for i := 0; i <= op.Width()-1; i++ {
		if s.usedPositions[pos+i] != isUnused {
			return false
		}
	}
	return true
}

// largestFit returns the number of subsequent unused slots starting at pos. The
// maximum is 33 since this is the largest instruction we have.
func (s *varCodeConstraintSolver) largestFit(pos int) int {
	n := 0
	for ; n < 33 && pos+n < s.codeSize; n++ {
		if s.usedPositions[pos+n] != isUnused {
			break
		}
	}
	return n
}

func (s *varCodeConstraintSolver) assign(v Variable, pos int) {
	if s.assignment != nil {
		s.assignment[v] = NewU256(uint64(pos))
	}
}

func (solver *varCodeConstraintSolver) solveVarOps(varOps []varOpConstraint) error {
	boundVariables := make(map[Variable]vm.OpCode)
	for _, cur := range varOps {
		if op, found := boundVariables[cur.variable]; found {
			if op != cur.op {
				return fmt.Errorf("%w, unable to satisfy conflicting constraint for op[%v]=%v and op[%v]=%v", ErrUnsatisfiable, cur.variable, op, cur.variable, cur.op)
			}
			continue
		}

		// Select a suitable code position for the current variable constraint.
		pos := int(solver.rnd.Int31n(int32(solver.codeSize)))
		startPos := pos
		for !solver.fits(pos, cur.op) {
			pos++
			if pos >= solver.codeSize {
				pos = 0
			}
			if pos == startPos {
				return fmt.Errorf("%w, unable to fit operations in given code size %d, (%d=%d)", ErrUnsatisfiable, solver.codeSize, pos, startPos)
			}
		}

		boundVariables[cur.variable] = cur.op

		solver.markUsed(pos, cur.op)
		solver.ops = append(solver.ops, constOpConstraint{pos, cur.op})

		solver.assign(cur.variable, pos)
	}
	return nil
}

func (solver *varCodeConstraintSolver) solveIsCode(varIsCodeConstraints []varIsCodeConstraint) error {
	for _, cur := range varIsCodeConstraints {
		// Check if the variable is already assigned and points to a slot marked
		// as code.
		if pos, isAssigned := solver.assignment[cur.variable]; isAssigned {
			if !pos.Lt(NewU256(uint64(solver.codeSize))) {
				return fmt.Errorf("%w, unable to satisfy isCode[%v], out-of-bounds", ErrUnsatisfiable, cur.variable)
			}
			if solver.usedPositions[int(pos.Uint64())] == isCode {
				continue // already satisfied
			}
		}

		// For the remaining variables, find a position and either populate an
		// unused slot, or use a slot with code in it. Code position 0 is
		// guaranteed to be either unused or contain code.
		pos := int(solver.rnd.Int31n(int32(solver.codeSize)))
		startPos := pos
		for solver.usedPositions[pos] == isData {
			pos++
			if pos >= solver.codeSize {
				pos = 0
			}
			if pos == startPos {
				return fmt.Errorf("%w, unable to fit isCode constraint in given code size", ErrUnsatisfiable)
			}
		}

		if solver.usedPositions[pos] == isUnused {
			// Pick a random op and lock it in.
			randomOps := vm.ValidOpCodesNoPush()
			op := randomOps[solver.rnd.Intn(len(randomOps))]
			solver.markUsed(pos, op)
			solver.ops = append(solver.ops, constOpConstraint{pos, op})
		}

		solver.assign(cur.variable, pos)
	}
	return nil
}

func (solver *varCodeConstraintSolver) solveIsData(varIsDataConstraints []varIsDataConstraint) error {
	for _, cur := range varIsDataConstraints {
		// Check if the variable is already assigned and points to a slot marked
		// as code. If so, we cannot satisfy this constraint!
		if pos, isAssigned := solver.assignment[cur.variable]; isAssigned {
			if pos.Lt(NewU256(uint64(solver.codeSize))) && solver.usedPositions[int(pos.Uint64())] == isCode {
				return fmt.Errorf("%w, unable to satisfy isData[%v]", ErrUnsatisfiable, cur.variable)
			}
		}

		// For the remaining variables, find a position and either populate an
		// unused slot, or use a slot with data in it.
		pos := int(solver.rnd.Int31n(int32(solver.codeSize)))
		startPos := pos
		pushOp := vm.PUSH1

		for {
			if solver.usedPositions[pos] == isData {
				break // using this pos
			}

			if solver.usedPositions[pos] == isUnused {
				// Pick a random PUSH op that fits here, if one fits at all.
				if largestFit := solver.largestFit(pos); largestFit >= 2 {
					pushOp = vm.OpCode(int(vm.PUSH1) + solver.rnd.Intn(largestFit-1))
					break // picked one
				}
			}

			pos++
			if pos >= solver.codeSize {
				pos = 0
			}
			if pos == startPos {
				return fmt.Errorf("%w, unable to fit isData constraint in given code size", ErrUnsatisfiable)
			}
		}

		if solver.usedPositions[pos] == isUnused {
			// set PUSH op
			solver.markUsed(pos, pushOp)
			solver.ops = append(solver.ops, constOpConstraint{pos, pushOp})

			// pick some data byte for the variable's value
			pos = pos + 1 + solver.rnd.Intn(int(pushOp-vm.PUSH1)+1)

		}

		solver.assign(cur.variable, pos)
	}
	return nil
}
