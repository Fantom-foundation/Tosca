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

// StateGenerator is a utility class for generating States. It provides two
// capabilities:
//
//   - The specification of constraints and the generation of a State instance
//     satisfying those constraints or the identification of unsatisfiable constraints.
//     This constraint solver is sound and complete.
//
//   - Support for cloning a generator state and restoring it.
//
// The goal of the StateGenerator is to produce high entropy results by setting
// all unconstrained degrees of freedom for the final code to random values.
//
// Multiple constraints can be set; Generate will return ErrUnsatisfiable when
// the given constraints result in a conflict.
//
// The same applies to subsequent generators.
type StateGenerator struct {
	// Constraints
	statusConstraints     []st.StatusCode
	revisionConstraints   []st.Revision
	pcConstantConstraints []uint16
	pcVariableConstraints []Variable
	gasConstraints        []uint64

	// Generators
	codeGen  *CodeGenerator
	stackGen *StackGenerator
}

// NewStateGenerator creates a generator without any initial constraints.
func NewStateGenerator() *StateGenerator {
	return &StateGenerator{
		codeGen:  NewCodeGenerator(),
		stackGen: NewStackGenerator(),
	}
}

// SetStatus adds a constraint on the State's status.
func (g *StateGenerator) SetStatus(status st.StatusCode) {
	if !slices.Contains(g.statusConstraints, status) {
		g.statusConstraints = append(g.statusConstraints, status)
	}
}

// SetRevision adds a constraint on the State's revision.
func (g *StateGenerator) SetRevision(revision st.Revision) {
	if !slices.Contains(g.revisionConstraints, revision) {
		g.revisionConstraints = append(g.revisionConstraints, revision)
	}
}

// SetPc adds a constraint on the State's program counter.
func (g *StateGenerator) SetPc(pc uint16) {
	if !slices.Contains(g.pcConstantConstraints, pc) {
		g.pcConstantConstraints = append(g.pcConstantConstraints, pc)
	}
}

// BindPc adds a constraint on the State's program counter to match the given
// variable.
func (g *StateGenerator) BindPc(pc Variable) {
	if !slices.Contains(g.pcVariableConstraints, pc) {
		g.pcVariableConstraints = append(g.pcVariableConstraints, pc)
	}
}

// SetGas adds a constraint on the State's gas counter.
func (g *StateGenerator) SetGas(gas uint64) {
	if !slices.Contains(g.gasConstraints, gas) {
		g.gasConstraints = append(g.gasConstraints, gas)
	}
}

// SetCodeOperation wraps CodeGenerator.SetOperation.
func (g *StateGenerator) SetCodeOperation(pos int, op OpCode) {
	g.codeGen.SetOperation(pos, op)
}

// AddCodeOperation add a constraint to ensure the existence of an operation in
// the generated code at a variable position.
func (g *StateGenerator) AddCodeOperation(v Variable, op OpCode) {
	g.codeGen.AddOperation(v, op)
}

// SetStackSize wraps StackGenerator.SetSize.
func (g *StateGenerator) SetStackSize(size int) {
	g.stackGen.SetSize(size)
}

// SetStackValue wraps StackGenerator.SetValue.
func (g *StateGenerator) SetStackValue(pos int, value U256) {
	g.stackGen.SetValue(pos, value)
}

// BindStackValue wraps StackGenerator.BindValue.
func (g *StateGenerator) BindStackValue(pos int, v Variable) {
	g.stackGen.BindValue(pos, v)
}

// Generate produces a State instance satisfying the constraints set on this
// generator or returns ErrUnsatisfiable on conflicting constraints. Subsequent
// generators are invoked automatically.
func (g *StateGenerator) Generate(rnd *rand.Rand) (*st.State, error) {
	assignment := Assignment{}

	// Pick a status.
	var resultStatus st.StatusCode
	if len(g.statusConstraints) == 0 {
		resultStatus = st.StatusCode(rnd.Int31n(int32(st.NumStatusCodes)))
	} else if len(g.statusConstraints) == 1 {
		resultStatus = g.statusConstraints[0]
		if resultStatus < 0 || resultStatus >= st.NumStatusCodes {
			return nil, fmt.Errorf("%w, invalid StatusCode provided %v", ErrUnsatisfiable, resultStatus)
		}
	} else {
		return nil, fmt.Errorf("%w, multiple conflicting status constraints defined: %v", ErrUnsatisfiable, g.statusConstraints)
	}

	// Pick a revision.
	var resultRevision st.Revision
	if len(g.revisionConstraints) == 0 {
		resultRevision = st.Revision(rnd.Int31n(int32(st.NumRevisions)))
	} else if len(g.revisionConstraints) == 1 {
		resultRevision = g.revisionConstraints[0]
		if resultRevision < 0 || resultRevision >= st.NumRevisions {
			return nil, fmt.Errorf("%w, invalid Revision provided %v", ErrUnsatisfiable, resultRevision)
		}
	} else {
		return nil, fmt.Errorf("%w, multiple conflicting revision constraints defined: %v", ErrUnsatisfiable, g.statusConstraints)
	}

	// Invoke CodeGenerator
	resultCode, err := g.codeGen.Generate(assignment, rnd)
	if err != nil {
		return nil, err
	}

	// Pick a program counter.
	var resultPc uint16
	{
		values := slices.Clone(g.pcConstantConstraints)
		for _, cur := range g.pcVariableConstraints {
			pc, found := assignment[cur]
			if !found {
				return nil, fmt.Errorf("%w, variable %v not bound to value", ErrUnboundVariable, cur)
			}
			value := uint16(pc.Uint64())
			if !slices.Contains(values, value) {
				values = append(values, value)
			}
		}
		if len(values) == 0 {
			// Generate a random program counter that points into the code slice.
			resultPc = uint16(rnd.Int31n(int32(resultCode.Length())))
		} else if len(values) == 1 {
			resultPc = values[0]
		} else {
			return nil, fmt.Errorf("%w, multiple conflicting program counter constraints defined: %v", ErrUnsatisfiable, g.statusConstraints)
		}
	}

	// Pick a gas counter.
	var resultGas uint64
	if len(g.gasConstraints) == 0 {
		resultGas = rnd.Uint64()
	} else if len(g.gasConstraints) == 1 {
		resultGas = g.gasConstraints[0]
	} else {
		return nil, fmt.Errorf("%w, multiple conflicting gas counter constraints defined: %v", ErrUnsatisfiable, g.gasConstraints)
	}

	// Invoke StackGenerator
	resultStack, err := g.stackGen.Generate(assignment, rnd)
	if err != nil {
		return nil, err
	}

	result := st.NewState(resultCode)
	result.Status = resultStatus
	result.Revision = resultRevision
	result.Pc = resultPc
	result.Gas = resultGas
	result.Stack = resultStack
	return result, nil
}

// Clone creates an independent copy of the generator in its current state.
// Future modifications are isolated from each other.
func (g *StateGenerator) Clone() *StateGenerator {
	return &StateGenerator{
		statusConstraints:     slices.Clone(g.statusConstraints),
		revisionConstraints:   slices.Clone(g.revisionConstraints),
		pcConstantConstraints: slices.Clone(g.pcConstantConstraints),
		pcVariableConstraints: slices.Clone(g.pcVariableConstraints),
		gasConstraints:        slices.Clone(g.gasConstraints),
		codeGen:               g.codeGen.Clone(),
		stackGen:              g.stackGen.Clone(),
	}
}

// Restore copies the state of the provided generator into this generator.
func (g *StateGenerator) Restore(other *StateGenerator) {
	if g != other {
		g.statusConstraints = slices.Clone(other.statusConstraints)
		g.revisionConstraints = slices.Clone(other.revisionConstraints)
		g.pcConstantConstraints = slices.Clone(other.pcConstantConstraints)
		g.pcVariableConstraints = slices.Clone(other.pcVariableConstraints)
		g.gasConstraints = slices.Clone(other.gasConstraints)
		g.codeGen.Restore(other.codeGen)
		g.stackGen.Restore(other.stackGen)
	}
}

func (g *StateGenerator) String() string {
	var parts []string

	sort.Slice(g.statusConstraints, func(i, j int) bool { return g.statusConstraints[i] < g.statusConstraints[j] })
	for _, status := range g.statusConstraints {
		parts = append(parts, fmt.Sprintf("status=%v", status))
	}

	sort.Slice(g.revisionConstraints, func(i, j int) bool { return g.revisionConstraints[i] < g.revisionConstraints[j] })
	for _, revision := range g.revisionConstraints {
		parts = append(parts, fmt.Sprintf("revision=%v", revision))
	}

	sort.Slice(g.pcConstantConstraints, func(i, j int) bool { return g.pcConstantConstraints[i] < g.pcConstantConstraints[j] })
	for _, pc := range g.pcConstantConstraints {
		parts = append(parts, fmt.Sprintf("pc=%d", pc))
	}

	sort.Slice(g.pcVariableConstraints, func(i, j int) bool { return g.pcVariableConstraints[i] < g.pcVariableConstraints[j] })
	for _, pc := range g.pcVariableConstraints {
		parts = append(parts, fmt.Sprintf("pc=%v", pc))
	}

	sort.Slice(g.gasConstraints, func(i, j int) bool { return g.gasConstraints[i] < g.gasConstraints[j] })
	for _, gas := range g.gasConstraints {
		parts = append(parts, fmt.Sprintf("gas=%d", gas))
	}

	parts = append(parts, fmt.Sprintf("code=%v", g.codeGen))
	parts = append(parts, fmt.Sprintf("stack=%v", g.stackGen))

	return "{" + strings.Join(parts, ",") + "}"
}
