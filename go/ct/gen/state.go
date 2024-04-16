//
// Copyright (c) 2024 Fantom Foundation
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at fantom.foundation/bsl11.
//
// Change Date: 2028-4-16
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by the GNU Lesser General Public Licence v3
//

package gen

import (
	"fmt"
	"slices"
	"sort"
	"strings"

	"pgregory.net/rand"

	. "github.com/Fantom-foundation/Tosca/go/ct/common"
	"github.com/Fantom-foundation/Tosca/go/ct/st"
	"github.com/Fantom-foundation/Tosca/go/vm"
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
	revisionConstraints   []RevisionBounds
	readOnlyConstraints   []bool
	pcConstantConstraints []uint16
	pcVariableConstraints []Variable
	gasConstraints        []vm.Gas
	gasRefundConstraints  []vm.Gas
	variableBindings      []variableBinding

	// Generators
	codeGen         *CodeGenerator
	stackGen        *StackGenerator
	memoryGen       *MemoryGenerator
	storageGen      *StorageGenerator
	accountsGen     *AccountsGenerator
	callContextGen  *CallContextGenerator
	callJournalGen  *CallJournalGenerator
	BlockContextGen *BlockContextGenerator
}

// NewStateGenerator creates a generator without any initial constraints.
func NewStateGenerator() *StateGenerator {
	return &StateGenerator{
		codeGen:         NewCodeGenerator(),
		stackGen:        NewStackGenerator(),
		memoryGen:       NewMemoryGenerator(),
		storageGen:      NewStorageGenerator(),
		accountsGen:     NewAccountGenerator(),
		callContextGen:  NewCallContextGenerator(),
		callJournalGen:  NewCallJournalGenerator(),
		BlockContextGen: NewBlockContextGenerator(),
	}
}

// variableBinding binds a constant value to a variable.
type variableBinding struct {
	variable Variable
	value    U256
}

func (a *variableBinding) Less(b *variableBinding) bool {
	return a.variable < b.variable || (a.variable == b.variable && a.value.Lt(b.value))
}

// BindValue binds a U256 value to a variable.
func (g *StateGenerator) BindValue(variable Variable, value U256) {
	binding := variableBinding{variable, value}
	if !slices.Contains(g.variableBindings, binding) {
		g.variableBindings = append(g.variableBindings, binding)
	}
}

// SetStatus adds a constraint on the State's status.
func (g *StateGenerator) SetStatus(status st.StatusCode) {
	if !slices.Contains(g.statusConstraints, status) {
		g.statusConstraints = append(g.statusConstraints, status)
	}
}

type RevisionBounds struct{ min, max Revision }

func (a RevisionBounds) Less(b RevisionBounds) bool {
	if a.min != b.min {
		return a.min < b.min
	} else {
		return a.max < b.max
	}
}

func (r RevisionBounds) String() string {
	if r.min == r.max {
		return fmt.Sprintf("%v", r.min)
	}
	return fmt.Sprintf("%v-%v", r.min, r.max)
}

// SetRevision adds a constraint on the State's revision.
func (g *StateGenerator) SetRevision(revision Revision) {
	g.SetRevisionBounds(revision, revision)
}

func (g *StateGenerator) SetRevisionBounds(min, max Revision) {
	if min > max {
		min, max = max, min
	}
	r := RevisionBounds{min, max}
	if !slices.Contains(g.revisionConstraints, r) {
		g.revisionConstraints = append(g.revisionConstraints, r)
	}
}

// SetReadOnly adds a constraint on the states read only mode.
func (g *StateGenerator) SetReadOnly(readOnly bool) {
	if !slices.Contains(g.readOnlyConstraints, readOnly) {
		g.readOnlyConstraints = append(g.readOnlyConstraints, readOnly)
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
func (g *StateGenerator) SetGas(gas vm.Gas) {
	if !slices.Contains(g.gasConstraints, gas) {
		g.gasConstraints = append(g.gasConstraints, gas)
	}
}

// SetGasRefund adds a constraint on the State's gas refund counter.
func (g *StateGenerator) SetGasRefund(gasRefund vm.Gas) {
	if !slices.Contains(g.gasRefundConstraints, gasRefund) {
		g.gasRefundConstraints = append(g.gasRefundConstraints, gasRefund)
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

// AddIsCode wraps CodeGenerator.AddIsCode.
func (g *StateGenerator) AddIsCode(v Variable) {
	g.codeGen.AddIsCode(v)
}

// AddIsData wraps CodeGenerator.AddIsData.
func (g *StateGenerator) AddIsData(v Variable) {
	g.codeGen.AddIsData(v)
}

// SetMinStackSize wraps StackGenerator.SetMinSize.
func (g *StateGenerator) SetMinStackSize(size int) {
	g.stackGen.SetMinSize(size)
}

// SetMaxStackSize wraps StackGenerator.SetMaxSize.
func (g *StateGenerator) SetMaxStackSize(size int) {
	g.stackGen.SetMaxSize(size)
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

// BindStorageConfiguration wraps StorageGenerator.BindConfiguration.
func (g *StateGenerator) BindStorageConfiguration(config StorageCfg, key, newValue Variable) {
	g.storageGen.BindConfiguration(config, key, newValue)
}

// BindIsStorageWarm wraps StorageGenerator.BindWarm.
func (g *StateGenerator) BindIsStorageWarm(key Variable) {
	g.storageGen.BindWarm(key)
}

// BindIsStorageCold wraps StorageGenerator.BindCold.
func (g *StateGenerator) BindIsStorageCold(key Variable) {
	g.storageGen.BindCold(key)
}

// BindToWarmAddress wraps AccountsGenerator.BindWarm.
func (g *StateGenerator) BindToWarmAddress(key Variable) {
	g.accountsGen.BindWarm(key)
}

// BindToColdAddress wraps AccountsGenerator.BindCold.
func (g *StateGenerator) BindToColdAddress(key Variable) {
	g.accountsGen.BindCold(key)
}

func getRandomData(rnd *rand.Rand) ([]byte, error) {
	size := uint(rnd.ExpFloat64() * float64(200))
	if size > st.MaxDataSize {
		size = st.MaxDataSize
	}
	dataBuffer := make([]byte, size)
	_, err := rnd.Read(dataBuffer)
	if err != nil {
		return nil, err
	}
	return dataBuffer, nil
}

// Generate produces a State instance satisfying the constraints set on this
// generator or returns ErrUnsatisfiable on conflicting constraints. Subsequent
// generators are invoked automatically.
func (g *StateGenerator) Generate(rnd *rand.Rand) (*st.State, error) {
	assignment := Assignment{}

	// Enforce variable assignments.
	for _, binding := range g.variableBindings {
		cur, found := assignment[binding.variable]
		if found && cur != binding.value {
			return nil, fmt.Errorf("%w: unsatisfiable variable binding", ErrUnsatisfiable)
		}
		assignment[binding.variable] = binding.value
	}

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

	var resultRevision Revision
	if len(g.revisionConstraints) == 0 {
		resultRevision = Revision(rnd.Int31n(int32(R99_UnknownNextRevision) + 1))
	} else {
		bounds := RevisionBounds{Revision(0), R99_UnknownNextRevision}
		for _, con := range g.revisionConstraints {
			if con.min > bounds.min {
				bounds.min = con.min
			}
			if con.max < bounds.max {
				bounds.max = con.max
			}
		}
		if bounds.min > bounds.max {
			return nil, fmt.Errorf("%w, conflicting revision constraints defined: %v", ErrUnsatisfiable, g.revisionConstraints)
		}
		resultRevision = Revision(rnd.Int31n(int32(bounds.max-bounds.min)+1)) + bounds.min
	}

	// Invoke CodeGenerator
	resultCode, err := g.codeGen.Generate(assignment, rnd)
	if err != nil {
		return nil, err
	}

	// Choose if state is in read only mode
	var resultReadOnly bool
	if len(g.readOnlyConstraints) == 0 {
		resultReadOnly = rnd.Uint32n(2) == 1
	} else if len(g.readOnlyConstraints) == 1 {
		resultReadOnly = g.readOnlyConstraints[0]
	} else {
		return nil, fmt.Errorf("%w, multiple conflicting read only constraints defined: %v", ErrUnsatisfiable, g.readOnlyConstraints)
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
			if resultCode.Length() > 0 {
				resultPc = uint16(rnd.Int31n(int32(resultCode.Length())))
			}
		} else if len(values) == 1 {
			resultPc = values[0]
		} else {
			return nil, fmt.Errorf("%w, multiple conflicting program counter constraints defined: %v", ErrUnsatisfiable, g.statusConstraints)
		}
	}

	// Pick a gas counter.
	var resultGas vm.Gas
	if len(g.gasConstraints) == 0 {
		resultGas = vm.Gas(rnd.Int63n(int64(st.MaxGas)))
	} else if len(g.gasConstraints) == 1 {
		resultGas = g.gasConstraints[0]
		if resultGas < 0 || resultGas > st.MaxGas {
			return nil, fmt.Errorf(
				"%w: gas out of bounds, constraint defined %d not in range [%d,%d]",
				ErrUnsatisfiable, resultGas, 0, st.MaxGas,
			)
		}
	} else {
		return nil, fmt.Errorf("%w, multiple conflicting gas counter constraints defined: %v", ErrUnsatisfiable, g.gasConstraints)
	}

	// Pick a gas refund counter.
	var resultGasRefund vm.Gas
	if len(g.gasRefundConstraints) == 0 {
		// Refunds can be positive or negative, of any value, and should be tracked accordingly.
		resultGasRefund = vm.Gas(rnd.Uint64())
	} else if len(g.gasRefundConstraints) == 1 {
		resultGasRefund = g.gasRefundConstraints[0]
	} else {
		return nil, fmt.Errorf("%w, multiple conflicting gas refund counter constraints defined: %v", ErrUnsatisfiable, g.gasRefundConstraints)
	}

	accountAddress, err := RandAddress(rnd)
	if err != nil {
		return nil, err
	}

	// Invoke CallContextGenerator
	resultCallContext, err := g.callContextGen.Generate(rnd, accountAddress)
	if err != nil {
		return nil, err
	}

	resultCallJournal, err := g.callJournalGen.Generate(rnd)
	if err != nil {
		return nil, err
	}

	// Invoke BlockContextGenerator
	resultBlockContext, err := g.BlockContextGen.Generate(rnd, resultRevision)
	if err != nil {
		return nil, err
	}

	// Pick a random calldata
	resultCallData := RandomBytes(rnd, st.MaxDataSize)

	// Generate return data of last call
	resultLastCallReturnData := RandomBytes(rnd, st.MaxDataSize)

	// Generate return data for terminal states.
	var resultReturnData Bytes
	if resultStatus == st.Stopped || resultStatus == st.Reverted {
		resultReturnData = RandomBytes(rnd, st.MaxDataSize)
	}

	// Sub-generators can modify the assignment when unassigned variables are
	// encountered. The order in which sub-generators are invoked influences
	// this process.

	// Invoke StorageGenerator
	resultStorage, err := g.storageGen.Generate(assignment, rnd)
	if err != nil {
		return nil, err
	}

	// Invoke AccountGenerator
	resultAccounts, err := g.accountsGen.Generate(assignment, rnd, accountAddress)
	if err != nil {
		return nil, err
	}

	// Invoke MemoryGenerator
	resultMemory, err := g.memoryGen.Generate(rnd)
	if err != nil {
		return nil, err
	}

	// Invoke StackGenerator
	resultStack, err := g.stackGen.Generate(assignment, rnd)
	if err != nil {
		return nil, err
	}

	result := st.NewState(resultCode)
	result.Status = resultStatus
	result.Revision = resultRevision
	result.ReadOnly = resultReadOnly
	result.Pc = resultPc
	result.Gas = resultGas
	result.GasRefund = resultGasRefund
	result.Stack = resultStack
	result.Memory = resultMemory
	result.Storage = resultStorage
	result.Accounts = resultAccounts
	result.CallContext = resultCallContext
	result.CallJournal = resultCallJournal
	result.BlockContext = resultBlockContext
	result.CallData = resultCallData
	result.LastCallReturnData = resultLastCallReturnData
	result.ReturnData = resultReturnData

	return result, nil
}

// Clone creates an independent copy of the generator in its current state.
// Future modifications are isolated from each other.
func (g *StateGenerator) Clone() *StateGenerator {
	return &StateGenerator{
		statusConstraints:     slices.Clone(g.statusConstraints),
		revisionConstraints:   slices.Clone(g.revisionConstraints),
		readOnlyConstraints:   slices.Clone(g.readOnlyConstraints),
		pcConstantConstraints: slices.Clone(g.pcConstantConstraints),
		pcVariableConstraints: slices.Clone(g.pcVariableConstraints),
		gasConstraints:        slices.Clone(g.gasConstraints),
		gasRefundConstraints:  slices.Clone(g.gasRefundConstraints),
		variableBindings:      slices.Clone(g.variableBindings),
		codeGen:               g.codeGen.Clone(),
		stackGen:              g.stackGen.Clone(),
		memoryGen:             g.memoryGen.Clone(),
		storageGen:            g.storageGen.Clone(),
		accountsGen:           g.accountsGen.Clone(),
		callContextGen:        g.callContextGen.Clone(),
		callJournalGen:        g.callJournalGen.Clone(),
		BlockContextGen:       g.BlockContextGen.Clone(),
	}
}

// Restore copies the state of the provided generator into this generator.
func (g *StateGenerator) Restore(other *StateGenerator) {
	if g != other {
		g.statusConstraints = slices.Clone(other.statusConstraints)
		g.revisionConstraints = slices.Clone(other.revisionConstraints)
		g.readOnlyConstraints = slices.Clone(other.readOnlyConstraints)
		g.pcConstantConstraints = slices.Clone(other.pcConstantConstraints)
		g.pcVariableConstraints = slices.Clone(other.pcVariableConstraints)
		g.gasConstraints = slices.Clone(other.gasConstraints)
		g.gasRefundConstraints = slices.Clone(other.gasRefundConstraints)
		g.variableBindings = slices.Clone(g.variableBindings)
		g.codeGen.Restore(other.codeGen)
		g.stackGen.Restore(other.stackGen)
		g.memoryGen.Restore(other.memoryGen)
		g.storageGen.Restore(other.storageGen)
		g.accountsGen.Restore(other.accountsGen)
		g.callContextGen.Restore(other.callContextGen)
		g.callJournalGen.Restore(other.callJournalGen)
		g.BlockContextGen.Restore(other.BlockContextGen)
	}
}

func (g *StateGenerator) String() string {
	var parts []string

	sort.Slice(g.variableBindings, func(i, j int) bool { return g.variableBindings[i].Less(&g.variableBindings[j]) })
	for _, binding := range g.variableBindings {
		parts = append(parts, fmt.Sprintf("%v=%v", binding.variable, binding.value))
	}

	sort.Slice(g.statusConstraints, func(i, j int) bool { return g.statusConstraints[i] < g.statusConstraints[j] })
	for _, status := range g.statusConstraints {
		parts = append(parts, fmt.Sprintf("status=%v", status))
	}

	sort.Slice(g.revisionConstraints, func(i, j int) bool { return g.revisionConstraints[i].Less(g.revisionConstraints[j]) })
	for _, revision := range g.revisionConstraints {
		parts = append(parts, fmt.Sprintf("revision=%v", revision))
	}

	for _, mode := range g.readOnlyConstraints {
		parts = append(parts, fmt.Sprintf("readOnly mode=%v", mode))
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

	sort.Slice(g.gasRefundConstraints, func(i, j int) bool { return g.gasRefundConstraints[i] < g.gasRefundConstraints[j] })
	for _, gas := range g.gasRefundConstraints {
		parts = append(parts, fmt.Sprintf("gasRefund=%d", gas))
	}

	parts = append(parts, fmt.Sprintf("code=%v", g.codeGen))
	parts = append(parts, fmt.Sprintf("stack=%v", g.stackGen))
	parts = append(parts, fmt.Sprintf("memory=%v", g.memoryGen))
	parts = append(parts, fmt.Sprintf("storage=%v", g.storageGen))
	parts = append(parts, fmt.Sprintf("accounts=%v", g.accountsGen))
	parts = append(parts, fmt.Sprintf("callContext=%v", g.callContextGen))
	parts = append(parts, fmt.Sprintf("callJournal=%v", g.callJournalGen))
	parts = append(parts, fmt.Sprintf("blockContext=%v", g.BlockContextGen))

	return "{" + strings.Join(parts, ",") + "}"
}
