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
	statusConstraints   []st.StatusCode
	revisionConstraints []st.Revision
	pcConstraints       []uint16
	gasConstraints      []uint64

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

// PickStatus provides an st.StatusCode satisfying the constraints (assuming
// generator is satisfiable). Constraints are added if needed.
func (g *StateGenerator) PickStatus(rnd *rand.Rand) st.StatusCode {
	if len(g.statusConstraints) == 0 {
		g.SetStatus(st.StatusCode(rnd.Int31n(int32(st.NumStatusCodes))))
	}
	return g.statusConstraints[0]
}

// SetRevision adds a constraint on the State's revision.
func (g *StateGenerator) SetRevision(revision st.Revision) {
	if !slices.Contains(g.revisionConstraints, revision) {
		g.revisionConstraints = append(g.revisionConstraints, revision)
	}
}

// SetPc adds a constraint on the State's program counter.
func (g *StateGenerator) SetPc(pc uint16) {
	if !slices.Contains(g.pcConstraints, pc) {
		g.pcConstraints = append(g.pcConstraints, pc)
	}
}

// SetGas adds a constraint on the State's gas counter.
func (g *StateGenerator) SetGas(gas uint64) {
	if !slices.Contains(g.gasConstraints, gas) {
		g.gasConstraints = append(g.gasConstraints, gas)
	}
}

// SetCodeSize wraps CodeGenerator.SetSize.
func (g *StateGenerator) SetCodeSize(size int) {
	g.codeGen.SetSize(size)
}

// SetCodeOperation wraps CodeGenerator.SetOperation.
func (g *StateGenerator) SetCodeOperation(pos int, op st.OpCode) {
	g.codeGen.SetOperation(pos, op)
}

func (g *StateGenerator) PickCodeOperation(pos int, rnd *rand.Rand) st.OpCode {
	return g.codeGen.PickOperation(pos, rnd)
}

// SetStackSize wraps StackGenerator.SetSize.
func (g *StateGenerator) SetStackSize(size int) {
	g.stackGen.SetSize(size)
}

// SetStackValue wraps StackGenerator.SetValue.
func (g *StateGenerator) SetStackValue(pos int, value ct.U256) {
	g.stackGen.SetValue(pos, value)
}

// Generate produces a State instance satisfying the constraints set on this
// generator or returns ErrUnsatisfiable on conflicting constraints. Subsequent
// generators are invoked automatically.
func (g *StateGenerator) Generate(rnd *rand.Rand) (*st.State, error) {
	// Pick a status.
	var resultStatus st.StatusCode
	if len(g.statusConstraints) == 0 {
		resultStatus = g.PickStatus(rnd)
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
	resultCode, err := g.codeGen.Generate(rnd)
	if err != nil {
		return nil, err
	}

	// Pick a program counter.
	var resultPc uint16
	if len(g.pcConstraints) == 0 {
		// Generate a random program counter that points into the code slice.
		resultPc = uint16(rnd.Int31n(int32(resultCode.Length())))
	} else if len(g.pcConstraints) == 1 {
		resultPc = g.pcConstraints[0]
	} else {
		return nil, fmt.Errorf("%w, multiple conflicting program counter constraints defined: %v", ErrUnsatisfiable, g.statusConstraints)
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
	resultStack, err := g.stackGen.Generate(rnd)
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
		statusConstraints:   slices.Clone(g.statusConstraints),
		revisionConstraints: slices.Clone(g.revisionConstraints),
		pcConstraints:       slices.Clone(g.pcConstraints),
		gasConstraints:      slices.Clone(g.gasConstraints),
		codeGen:             g.codeGen.Clone(),
		stackGen:            g.stackGen.Clone(),
	}
}

// Restore copies the state of the provided generator into this generator.
func (g *StateGenerator) Restore(other *StateGenerator) {
	if g != other {
		g.statusConstraints = slices.Clone(other.statusConstraints)
		g.revisionConstraints = slices.Clone(other.revisionConstraints)
		g.pcConstraints = slices.Clone(other.pcConstraints)
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

	sort.Slice(g.pcConstraints, func(i, j int) bool { return g.pcConstraints[i] < g.pcConstraints[j] })
	for _, pc := range g.pcConstraints {
		parts = append(parts, fmt.Sprintf("pc=%d", pc))
	}

	sort.Slice(g.gasConstraints, func(i, j int) bool { return g.gasConstraints[i] < g.gasConstraints[j] })
	for _, gas := range g.gasConstraints {
		parts = append(parts, fmt.Sprintf("gas=%d", gas))
	}

	parts = append(parts, fmt.Sprintf("code=%v", g.codeGen))
	parts = append(parts, fmt.Sprintf("stack=%v", g.stackGen))

	return "{" + strings.Join(parts, ",") + "}"
}
