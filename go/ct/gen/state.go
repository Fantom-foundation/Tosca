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
	"bytes"
	"fmt"
	"slices"
	"sort"
	"strings"

	"pgregory.net/rand"

	. "github.com/Fantom-foundation/Tosca/go/ct/common"
	"github.com/Fantom-foundation/Tosca/go/ct/st"
	"github.com/Fantom-foundation/Tosca/go/tosca"
	"github.com/Fantom-foundation/Tosca/go/tosca/vm"
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
	statusConstraints      []st.StatusCode
	readOnlyConstraints    []bool
	pcConstantConstraints  []uint16
	pcVariableConstraints  []Variable
	gasConstraints         *RangeSolver[tosca.Gas]
	gasRefundConstraints   *RangeSolver[tosca.Gas]
	variableBindings       []variableBinding
	selfAddressConstraints []tosca.Address
	selfAddressBindings    []Variable

	// Generators
	codeGen               *CodeGenerator
	stackGen              *StackGenerator
	memoryGen             *MemoryGenerator
	storageGen            *StorageGenerator
	transientStorageGen   *TransientStorageGenerator
	accountsGen           *AccountsGenerator
	callContextGen        *CallContextGenerator
	callJournalGen        *CallJournalGenerator
	blockContextGen       *BlockContextGenerator
	hasSelfDestructedGen  *SelfDestructedGenerator
	transactionContextGen *TransactionContextGenerator
}

// NewStateGenerator creates a generator without any initial constraints.
func NewStateGenerator() *StateGenerator {
	return &StateGenerator{
		codeGen:               NewCodeGenerator(),
		stackGen:              NewStackGenerator(),
		memoryGen:             NewMemoryGenerator(),
		storageGen:            NewStorageGenerator(),
		transientStorageGen:   NewTransientStorageGenerator(),
		accountsGen:           NewAccountGenerator(),
		callContextGen:        NewCallContextGenerator(),
		callJournalGen:        NewCallJournalGenerator(),
		blockContextGen:       NewBlockContextGenerator(),
		gasConstraints:        NewRangeSolver[tosca.Gas](0, st.MaxGasUsedByCt),
		gasRefundConstraints:  NewRangeSolver[tosca.Gas](-st.MaxGasUsedByCt, st.MaxGasUsedByCt),
		hasSelfDestructedGen:  NewSelfDestructedGenerator(),
		transactionContextGen: NewTransactionContextGenerator(),
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

// SetRevision adds a constraint on the State's revision.
func (g *StateGenerator) SetRevision(revision tosca.Revision) {
	g.blockContextGen.SetRevision(revision)
}

func (g *StateGenerator) AddRevisionBounds(min, max tosca.Revision) {
	g.blockContextGen.AddRevisionBounds(min, max)
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

// BindToAccountAddress constraints the given variable to be mapped to the
// address of the account that is executing the code.
func (g *StateGenerator) BindToSelfAddress(address Variable) {
	if !slices.Contains(g.selfAddressBindings, address) {
		g.selfAddressBindings = append(g.selfAddressBindings, address)
	}
}

// SetSelfAddress adds a constraint on the State's self address to match the
// given address.
func (g *StateGenerator) SetSelfAddress(address tosca.Address) {
	if !slices.Contains(g.selfAddressConstraints, address) {
		g.selfAddressConstraints = append(g.selfAddressConstraints, address)
	}
}

// SetGas adds a constraint on the State's gas counter.
func (g *StateGenerator) SetGas(gas tosca.Gas) {
	g.gasConstraints.AddEqualityConstraint(gas)
}

// AddGasLowerBound adds a constraint on the lower bound of the gas value.
func (g *StateGenerator) AddGasLowerBound(gas tosca.Gas) {
	g.gasConstraints.AddLowerBoundary(gas)
}

// AddGasUpperBound adds a constraint on the upper bound of the gas value.
func (g *StateGenerator) AddGasUpperBound(gas tosca.Gas) {
	g.gasConstraints.AddUpperBoundary(gas)
}

// SetGasRefund adds a constraint on the State's gas refund counter.
func (g *StateGenerator) SetGasRefund(gasRefund tosca.Gas) {
	g.gasRefundConstraints.AddEqualityConstraint(gasRefund)
}

// AddGasRefundLowerBound adds a constraint on the lower bound of the gas refund value.
func (g *StateGenerator) AddGasRefundLowerBound(gas tosca.Gas) {
	g.gasRefundConstraints.AddLowerBoundary(gas)
}

// AddGasRefundUpperBound adds a constraint on the upper bound of the gas refund value.
func (g *StateGenerator) AddGasRefundUpperBound(gas tosca.Gas) {
	g.gasRefundConstraints.AddUpperBoundary(gas)
}

// SetCodeOperation wraps CodeGenerator.SetOperation.
func (g *StateGenerator) SetCodeOperation(pos int, op vm.OpCode) {
	g.codeGen.SetOperation(pos, op)
}

// AddCodeOperation add a constraint to ensure the existence of an operation in
// the generated code at a variable position.
func (g *StateGenerator) AddCodeOperation(v Variable, op vm.OpCode) {
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

// AddStackSizeLowerBound wraps StackGenerator.SetMinSize.
func (g *StateGenerator) AddStackSizeLowerBound(size int) {
	g.stackGen.AddMinSize(size)
}

// AddStackSizeUpperBound wraps StackGenerator.SetMaxSize.
func (g *StateGenerator) AddStackSizeUpperBound(size int) {
	g.stackGen.AddMaxSize(size)
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
func (g *StateGenerator) BindStorageConfiguration(config tosca.StorageStatus, key, newValue Variable) {
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

// BindTransientStorageToNonZero wraps TransientGenerator.BindToNonZero.
func (g *StateGenerator) BindTransientStorageToNonZero(key Variable) {
	g.transientStorageGen.BindToNonZero(key)
}

// BindTransientStorageToZero wraps TransientGenerator.BindToZero.
func (g *StateGenerator) BindTransientStorageToZero(key Variable) {
	g.transientStorageGen.BindToZero(key)
}

// BindToAddressOfEmptyAccount constraints the given variable to be mapped to
// an address of an account that is empty.
func (g *StateGenerator) BindToAddressOfEmptyAccount(address Variable) {
	g.accountsGen.BindToAddressOfEmptyAccount(address)
}

// BindToAddressOfNonEmptyAccount constraints the given variable to be mapped
// to an address of an account that is not empty.
func (g *StateGenerator) BindToAddressOfNonEmptyAccount(address Variable) {
	g.accountsGen.BindToAddressOfNonEmptyAccount(address)
}

// AddMinimumBalance adds a constraint restricting the balance of the account
// identified by the given variable to be at least the given value.
func (g *StateGenerator) AddBalanceLowerBound(address Variable, value U256) {
	g.accountsGen.AddBalanceLowerBound(address, value)
}

// AddMinimumBalance adds a constraint restricting the balance of the account
// identified by the given variable to be at most the given value.
func (g *StateGenerator) AddBalanceUpperBound(address Variable, value U256) {
	g.accountsGen.AddBalanceUpperBound(address, value)
}

// BindToWarmAddress wraps AccountsGenerator.BindWarm.
func (g *StateGenerator) BindToWarmAddress(key Variable) {
	g.accountsGen.BindWarm(key)
}

// BindToColdAddress wraps AccountsGenerator.BindCold.
func (g *StateGenerator) BindToColdAddress(key Variable) {
	g.accountsGen.BindCold(key)
}

func (g *StateGenerator) MustBeSelfDestructed() {
	g.hasSelfDestructedGen.MarkAsSelfDestructed()
}

func (g *StateGenerator) MustNotBeSelfDestructed() {
	g.hasSelfDestructedGen.MarkAsNotSelfDestructed()
}

func (g *StateGenerator) RestrictVariableToOneOfTheLast256Blocks(variable Variable) {
	g.blockContextGen.RestrictVariableToOneOfTheLast256Blocks(variable)
}

func (g *StateGenerator) RestrictVariableToNoneOfTheLast256Blocks(variable Variable) {
	g.blockContextGen.RestrictVariableToNoneOfTheLast256Blocks(variable)
}

func (g *StateGenerator) SetBlockNumberOffsetValue(variable Variable, value int64) {
	g.blockContextGen.SetBlockNumberOffsetValue(variable, value)
}

func (g *StateGenerator) IsPresentBlobHashIndex(variable Variable) {
	g.transactionContextGen.IsPresentBlobHashIndex(variable)
}

func (g *StateGenerator) IsAbsentBlobHashIndex(variable Variable) {
	g.transactionContextGen.IsAbsentBlobHashIndex(variable)
}

// Generate produces a State instance satisfying the constraints set on this
// generator or returns ErrUnsatisfiable on conflicting constraints. Subsequent
// generators are invoked automatically.
func (g *StateGenerator) Generate(rnd *rand.Rand) (*st.State, error) {
	assignment := Assignment{}
	return g.generateWith(rnd, assignment)
}

func (g *StateGenerator) generateWith(rnd *rand.Rand, assignment Assignment) (*st.State, error) {

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

	// Pick a gas level.
	resultGas, err := g.gasConstraints.Generate(rnd)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve gas constraints: %w", err)
	}

	// Pick a gas refund counter.
	resultGasRefund, err := g.gasRefundConstraints.Generate(rnd)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve gas refund constraints: %w", err)
	}

	// --- Self Address ---

	// Pick the address of the account executing the code.
	var address *tosca.Address

	// Consume fixed constraints on the self-address.
	if len(g.selfAddressConstraints) > 1 {
		return nil, fmt.Errorf(
			"%w, multiple conflicting self address constraints defined: %v",
			ErrUnsatisfiable, g.selfAddressConstraints,
		)
	}
	if len(g.selfAddressConstraints) == 1 {
		address = &g.selfAddressConstraints[0]
	}

	// Check variable constraints and pre-assigned values.
	for _, v := range g.selfAddressBindings {
		value, found := assignment[v]
		if !found {
			continue
		}
		should := NewAddress(value)
		if address == nil {
			address = &should
			continue
		}
		if should != *address {
			return nil, fmt.Errorf("%w, conflicting address bindings", ErrUnsatisfiable)
		}
	}

	// If there are no constraints, generate a random address.
	if address == nil {
		// Generate a random address.
		accountAddress := RandomAddress(rnd)
		address = &accountAddress
	}

	// Update all variables to be bound to the self address to match that value.
	accountAddress := *address
	for _, v := range g.selfAddressBindings {
		assignment[v] = NewU256FromBytes(accountAddress[:]...)
	}

	// --- Call Context ---

	// Invoke CallContextGenerator
	resultCallContext, err := g.callContextGen.Generate(rnd, accountAddress)
	if err != nil {
		return nil, err
	}

	resultCallJournal, err := g.callJournalGen.Generate(rnd)
	if err != nil {
		return nil, err
	}

	// Pick a random calldata
	resultCallData := RandomBytes(rnd, st.MaxDataSize)

	// Generate return data of last call
	resultLastCallReturnData := RandomBytes(rnd, st.MaxDataSize)

	// Invoke SelfDestructedGenerator
	resultHasSelfdestructed, err := g.hasSelfDestructedGen.Generate(rnd)
	if err != nil {
		return nil, err
	}

	// generate recent block hashes
	resultRecentBlockHashes := NewRandomImmutableHashArray(rnd)

	// Sub-generators can modify the assignment when unassigned variables are
	// encountered. The order in which sub-generators are invoked influences
	// this process.

	// Invoke StorageGenerator
	resultStorage, err := g.storageGen.Generate(assignment, rnd)
	if err != nil {
		return nil, err
	}

	// Invoke TransientStorageGenerator
	resultTransient, err := g.transientStorageGen.Generate(assignment, rnd)
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

	// Invoke TransactionContextGenerator
	resultTransactionContext, err := g.transactionContextGen.Generate(assignment, rnd)
	if err != nil {
		return nil, err
	}

	// Invoke BlockContextGenerator
	resultBlockContext, err := g.blockContextGen.Generate(assignment, rnd)
	if err != nil {
		return nil, err
	}

	// Invoke StackGenerator
	resultStack, err := g.stackGen.Generate(assignment, rnd)
	if err != nil {
		return nil, err
	}

	resultRevision := GetRevisionForBlock(resultBlockContext.BlockNumber)

	// Return data is not set as it should only be set by RETURN/REVERT opcodes.
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
	result.TransientStorage = resultTransient
	result.Accounts = resultAccounts
	result.CallContext = resultCallContext
	result.CallJournal = resultCallJournal
	result.BlockContext = resultBlockContext
	result.TransactionContext = resultTransactionContext
	result.CallData = resultCallData
	result.LastCallReturnData = resultLastCallReturnData
	result.HasSelfDestructed = resultHasSelfdestructed
	result.RecentBlockHashes = resultRecentBlockHashes

	return result, nil
}

// Clone creates an independent copy of the generator in its current state.
// Future modifications are isolated from each other.
func (g *StateGenerator) Clone() *StateGenerator {
	return &StateGenerator{
		statusConstraints:      slices.Clone(g.statusConstraints),
		readOnlyConstraints:    slices.Clone(g.readOnlyConstraints),
		pcConstantConstraints:  slices.Clone(g.pcConstantConstraints),
		pcVariableConstraints:  slices.Clone(g.pcVariableConstraints),
		gasConstraints:         g.gasConstraints.Clone(),
		gasRefundConstraints:   g.gasRefundConstraints.Clone(),
		variableBindings:       slices.Clone(g.variableBindings),
		selfAddressConstraints: slices.Clone(g.selfAddressConstraints),
		selfAddressBindings:    slices.Clone(g.selfAddressBindings),
		codeGen:                g.codeGen.Clone(),
		stackGen:               g.stackGen.Clone(),
		memoryGen:              g.memoryGen.Clone(),
		storageGen:             g.storageGen.Clone(),
		transientStorageGen:    g.transientStorageGen.Clone(),
		accountsGen:            g.accountsGen.Clone(),
		callContextGen:         g.callContextGen.Clone(),
		callJournalGen:         g.callJournalGen.Clone(),
		blockContextGen:        g.blockContextGen.Clone(),
		hasSelfDestructedGen:   g.hasSelfDestructedGen.Clone(),
		transactionContextGen:  g.transactionContextGen.Clone(),
	}
}

// Restore copies the state of the provided generator into this generator.
func (g *StateGenerator) Restore(other *StateGenerator) {
	if g != other {
		g.statusConstraints = slices.Clone(other.statusConstraints)
		g.readOnlyConstraints = slices.Clone(other.readOnlyConstraints)
		g.pcConstantConstraints = slices.Clone(other.pcConstantConstraints)
		g.pcVariableConstraints = slices.Clone(other.pcVariableConstraints)
		g.gasConstraints.Restore(other.gasConstraints)
		g.gasRefundConstraints.Restore(other.gasRefundConstraints)
		g.variableBindings = slices.Clone(other.variableBindings)
		g.selfAddressConstraints = slices.Clone(other.selfAddressConstraints)
		g.selfAddressBindings = slices.Clone(other.selfAddressBindings)
		g.codeGen.Restore(other.codeGen)
		g.stackGen.Restore(other.stackGen)
		g.memoryGen.Restore(other.memoryGen)
		g.storageGen.Restore(other.storageGen)
		g.transientStorageGen.Restore(other.transientStorageGen)
		g.accountsGen.Restore(other.accountsGen)
		g.callContextGen.Restore(other.callContextGen)
		g.callJournalGen.Restore(other.callJournalGen)
		g.blockContextGen.Restore(other.blockContextGen)
		g.hasSelfDestructedGen.Restore(other.hasSelfDestructedGen)
		g.transactionContextGen.Restore(other.transactionContextGen)
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

	sort.Slice(g.selfAddressBindings, func(i, j int) bool { return g.selfAddressBindings[i] < g.selfAddressBindings[j] })
	for _, v := range g.selfAddressBindings {
		parts = append(parts, fmt.Sprintf("selfAddress=%v", v))
	}

	sort.Slice(g.selfAddressConstraints, func(i, j int) bool {
		return bytes.Compare(g.selfAddressConstraints[i][:], g.selfAddressConstraints[j][:]) < 0
	})
	for _, address := range g.selfAddressConstraints {
		parts = append(parts, fmt.Sprintf("selfAddress=%v", address))
	}

	parts = append(parts, g.gasConstraints.Print("gas"))
	parts = append(parts, g.gasRefundConstraints.Print("gasRefund"))

	parts = append(parts, fmt.Sprintf("code=%v", g.codeGen))
	parts = append(parts, fmt.Sprintf("stack=%v", g.stackGen))
	parts = append(parts, fmt.Sprintf("memory=%v", g.memoryGen))
	parts = append(parts, fmt.Sprintf("storage=%v", g.storageGen))
	parts = append(parts, fmt.Sprintf("transient=%v", g.transientStorageGen))
	parts = append(parts, fmt.Sprintf("accounts=%v", g.accountsGen))
	parts = append(parts, fmt.Sprintf("callContext=%v", g.callContextGen))
	parts = append(parts, fmt.Sprintf("callJournal=%v", g.callJournalGen))
	parts = append(parts, fmt.Sprintf("blockContext=%v", g.blockContextGen))
	parts = append(parts, fmt.Sprintf("selfdestruct=%v", g.hasSelfDestructedGen))
	parts = append(parts, fmt.Sprintf("transactionContext=%v", g.transactionContextGen))

	return "{" + strings.Join(parts, ",") + "}"
}
