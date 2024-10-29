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
	"errors"
	"slices"
	"strings"
	"testing"

	"pgregory.net/rand"

	. "github.com/Fantom-foundation/Tosca/go/ct/common"
	"github.com/Fantom-foundation/Tosca/go/ct/st"
	"github.com/Fantom-foundation/Tosca/go/tosca"
	"github.com/Fantom-foundation/Tosca/go/tosca/vm"
)

func TestStateGenerator_UnconstrainedGeneratorCanProduceState(t *testing.T) {
	rnd := rand.New(0)
	generator := NewStateGenerator()
	if _, err := generator.Generate(rnd); err != nil {
		t.Fatalf("unexpected error during build: %v", err)
	}
}

////////////////////////////////////////////////////////////
// Status

func TestStateGenerator_SetStatusIsEnforced(t *testing.T) {
	statuses := []st.StatusCode{st.Running, st.Failed, st.Reverted}

	rnd := rand.New(0)
	for _, status := range statuses {
		generator := NewStateGenerator()
		generator.SetStatus(status)
		state, err := generator.Generate(rnd)
		if err != nil {
			t.Fatalf("unexpected error during build: %v", err)
		}
		if want, got := status, state.Status; want != got {
			t.Errorf("unexpected status, wanted %d, got %d", want, got)
		}
	}
}

func TestStateGenerator_ConflictingStatusesAreDetected(t *testing.T) {
	generator := NewStateGenerator()
	generator.SetStatus(st.Running)
	generator.SetStatus(st.Failed)
	rnd := rand.New(0)
	if _, err := generator.Generate(rnd); !errors.Is(err, ErrUnsatisfiable) {
		t.Errorf("unsatisfiable constraint not detected, got %v", err)
	}
}

func TestStateGenerator_NegativeStatusesAreDetected(t *testing.T) {
	generator := NewStateGenerator()
	generator.SetStatus(st.StatusCode(-12))
	rnd := rand.New(0)
	if _, err := generator.Generate(rnd); !errors.Is(err, ErrUnsatisfiable) {
		t.Errorf("unsatisfiable constraint not detected, got %v", err)
	}
}

func TestStateGenerator_NonConflictingStatusesAreAccepted(t *testing.T) {
	generator := NewStateGenerator()
	generator.SetStatus(st.Reverted)
	generator.SetStatus(st.Reverted)
	rnd := rand.New(0)
	if _, err := generator.Generate(rnd); err != nil {
		t.Errorf("generation failed: %v", err)
	}
}

////////////////////////////////////////////////////////////
// Revision

func TestStateGenerator_SetRevisionIsEnforced(t *testing.T) {
	revisions := []tosca.Revision{tosca.R07_Istanbul, tosca.R09_Berlin, tosca.R10_London}

	rnd := rand.New(0)
	for _, revision := range revisions {
		generator := NewStateGenerator()
		generator.SetRevision(revision)
		state, err := generator.Generate(rnd)
		if err != nil {
			t.Fatalf("unexpected error during build: %v", err)
		}
		if want, got := revision, state.Revision; want != got {
			t.Errorf("unexpected revision, wanted %d, got %d", want, got)
		}
	}
}

func TestStateGenerator_ConflictingRevisionsAreDetected(t *testing.T) {
	generator := NewStateGenerator()
	generator.SetRevision(tosca.R07_Istanbul)
	generator.SetRevision(tosca.R10_London)
	rnd := rand.New(0)
	if _, err := generator.Generate(rnd); !errors.Is(err, ErrUnsatisfiable) {
		t.Errorf("unsatisfiable constraint not detected, got %v", err)
	}
}

func TestStateGenerator_NegativeRevisionsAreDetected(t *testing.T) {
	generator := NewStateGenerator()
	generator.SetRevision(tosca.Revision(-12))
	rnd := rand.New(0)
	if _, err := generator.Generate(rnd); !errors.Is(err, ErrUnsatisfiable) {
		t.Errorf("unsatisfiable constraint not detected, got %v", err)
	}
}

func TestStateGenerator_NonConflictingRevisionsAreAccepted(t *testing.T) {
	generator := NewStateGenerator()
	generator.SetRevision(tosca.R10_London)
	generator.SetRevision(tosca.R10_London)
	rnd := rand.New(0)
	if _, err := generator.Generate(rnd); err != nil {
		t.Errorf("generation failed: %v", err)
	}
}

func TestStateGenerator_AddRevisionBoundsIsEnforced(t *testing.T) {
	generator := NewStateGenerator()
	generator.AddRevisionBounds(tosca.R07_Istanbul, tosca.R09_Berlin)
	generator.AddRevisionBounds(tosca.R09_Berlin, tosca.R10_London)

	state, err := generator.Generate(rand.New(0))
	if err != nil {
		t.Fatalf("unexpected error during build: %v", err)
	}
	if want, got := tosca.R09_Berlin, state.Revision; want != got {
		t.Fatalf("Revision bounds not working, want %v, got %v", want, got)
	}
}

func TestStateGenerator_ConflictingRevisionBoundsAreDetected(t *testing.T) {
	generator := NewStateGenerator()
	generator.AddRevisionBounds(tosca.R07_Istanbul, tosca.R09_Berlin)
	generator.AddRevisionBounds(tosca.R10_London, R99_UnknownNextRevision)

	if _, err := generator.Generate(rand.New(0)); !errors.Is(err, ErrUnsatisfiable) {
		t.Errorf("unsatisfiable constraint not detected, got %v", err)
	}
}

////////////////////////////////////////////////////////////
// Program Counter

func TestStateGenerator_SetPcIsEnforced(t *testing.T) {
	pcs := []uint16{0, 2, 4}

	rnd := rand.New(0)
	for _, pc := range pcs {
		generator := NewStateGenerator()
		generator.SetPc(pc)
		state, err := generator.Generate(rnd)
		if err != nil {
			t.Fatalf("unexpected error during build: %v", err)
		}
		if want, got := pc, state.Pc; want != got {
			t.Errorf("unexpected program counter, wanted %d, got %d", want, got)
		}
	}
}

func TestStateGenerator_ConflictingPcesAreDetected(t *testing.T) {
	generator := NewStateGenerator()
	generator.SetPc(0)
	generator.SetPc(1)
	rnd := rand.New(0)
	if _, err := generator.Generate(rnd); !errors.Is(err, ErrUnsatisfiable) {
		t.Errorf("unsatisfiable constraint not detected, got %v", err)
	}
}

func TestStateGenerator_NonConflictingPcesAreAccepted(t *testing.T) {
	generator := NewStateGenerator()
	generator.SetPc(1)
	generator.SetPc(1)
	rnd := rand.New(0)
	if _, err := generator.Generate(rnd); err != nil {
		t.Errorf("generation failed: %v", err)
	}
}

////////////////////////////////////////////////////////////
// Gas Counter

func TestStateGenerator_SetGasIsEnforced(t *testing.T) {
	gasCounts := []tosca.Gas{0, 42, st.MaxGasUsedByCt}

	rnd := rand.New(0)
	for _, gas := range gasCounts {
		generator := NewStateGenerator()
		generator.SetGas(gas)
		state, err := generator.Generate(rnd)
		if err != nil {
			t.Fatalf("unexpected error during build: %v", err)
		}
		if want, got := gas, state.Gas; want != got {
			t.Errorf("unexpected amount of gas, wanted %d, got %d", want, got)
		}
	}
}

func TestStateGenerator_ConflictingGasAreDetected(t *testing.T) {
	generator := NewStateGenerator()
	generator.SetGas(0)
	generator.SetGas(42)
	rnd := rand.New(0)
	if _, err := generator.Generate(rnd); !errors.Is(err, ErrUnsatisfiable) {
		t.Errorf("unsatisfiable constraint not detected, got %v", err)
	}
}

func TestStateGenerator_NonConflictingGasAreAccepted(t *testing.T) {
	generator := NewStateGenerator()
	generator.SetGas(42)
	generator.SetGas(42)
	rnd := rand.New(0)
	if _, err := generator.Generate(rnd); err != nil {
		t.Errorf("generation failed: %v", err)
	}
}

func TestStateGenerator_AddGasLowerUpperBoundIsEnforced(t *testing.T) {
	generator := NewStateGenerator()
	generator.AddGasLowerBound(42)
	generator.AddGasUpperBound(44)

	state, err := generator.Generate(rand.New(0))
	if err != nil {
		t.Fatalf("unexpected error during build: %v", err)
	}
	if want, got := tosca.Gas(43), state.Gas; want != got {
		t.Fatalf("Gas bounds not working, want %v, got %v", want, got)
	}
}

////////////////////////////////////////////////////////////
// Gas Refund Counter

func TestStateGenerator_SetGasRefundIsEnforced(t *testing.T) {
	gasRefundCounts := []tosca.Gas{0, 42, st.MaxGasUsedByCt}

	rnd := rand.New(0)
	for _, gasRefund := range gasRefundCounts {
		generator := NewStateGenerator()
		generator.SetGasRefund(gasRefund)
		state, err := generator.Generate(rnd)
		if err != nil {
			t.Fatalf("failed to generate state, unexpected error: %v", err)
		}
		if want, got := gasRefund, state.GasRefund; want != got {
			t.Errorf("unexpected amount of gas refund, wanted %d, got %d", want, got)
		}
	}
}

func TestStateGenerator_ConflictingGasRefundAreDetected(t *testing.T) {
	generator := NewStateGenerator()
	generator.SetGasRefund(0)
	generator.SetGasRefund(42)
	rnd := rand.New(0)
	if _, err := generator.Generate(rnd); !errors.Is(err, ErrUnsatisfiable) {
		t.Errorf("unsatisfiable constraint not detected, got %v", err)
	}
}

func TestStateGenerator_NonConflictingGasRefundAreAccepted(t *testing.T) {
	generator := NewStateGenerator()
	generator.SetGasRefund(42)
	generator.SetGasRefund(42)
	rnd := rand.New(0)
	if _, err := generator.Generate(rnd); err != nil {
		t.Errorf("generation failed: %v", err)
	}
}

func TestStateGenerator_AddGasRefundLowerUpperBoundIsEnforced(t *testing.T) {
	generator := NewStateGenerator()
	generator.AddGasRefundLowerBound(42)
	generator.AddGasRefundUpperBound(44)

	state, err := generator.Generate(rand.New(0))
	if err != nil {
		t.Fatalf("failed to generate state, unexpected error: %v", err)
	}
	if want, got := tosca.Gas(43), state.GasRefund; want != got {
		t.Fatalf("Gas refund bounds not working, want %v, got %v", want, got)
	}
}

////////////////////////////////////////////////////////////
// Self Address

func TestStateGenerator_SetSelfAddress(t *testing.T) {
	addresses := []tosca.Address{{1}, {8}, {72, 14}}

	rnd := rand.New(0)
	for _, address := range addresses {
		generator := NewStateGenerator()
		generator.SetSelfAddress(address)
		state, err := generator.Generate(rnd)
		if err != nil {
			t.Fatalf("unexpected error during build: %v", err)
		}
		if want, got := address, state.CallContext.AccountAddress; want != got {
			t.Errorf("unexpected account address, wanted %d, got %d", want, got)
		}
	}
}

func TestStateGenerator_ConflictingSelfAddressesAreDetected(t *testing.T) {
	generator := NewStateGenerator()
	generator.SetSelfAddress(tosca.Address{1})
	generator.SetSelfAddress(tosca.Address{2})
	rnd := rand.New(0)
	if _, err := generator.Generate(rnd); !errors.Is(err, ErrUnsatisfiable) {
		t.Errorf("unsatisfiable constraint not detected, got %v", err)
	}
}

func TestStateGenerator_NonConflictingSelfAddressesAreAccepted(t *testing.T) {
	generator := NewStateGenerator()
	generator.SetSelfAddress(tosca.Address{1})
	generator.SetSelfAddress(tosca.Address{1})
	rnd := rand.New(0)
	if _, err := generator.Generate(rnd); err != nil {
		t.Errorf("generation failed: %v", err)
	}
}

func TestStateGenerator_SelfAddressCanBeBoundToVariable(t *testing.T) {
	varX := Variable("X")
	varY := Variable("Y")

	rnd := rand.New(0)
	generator := NewStateGenerator()
	generator.BindToSelfAddress(varX)
	generator.BindToSelfAddress(varY)
	assignment := Assignment{}
	state, err := generator.generateWith(rnd, assignment)
	if err != nil {
		t.Fatalf("unexpected error during build: %v", err)
	}
	want := NewU256FromBytes(state.CallContext.AccountAddress[:]...)
	if got := assignment[varX]; want != got {
		t.Errorf("unexpected account address for %s, wanted %d, got %d", varX, want, got)
	}
	if got := assignment[varY]; want != got {
		t.Errorf("unexpected account address for %s, wanted %d, got %d", varY, want, got)
	}
}

func TestStateGenerator_ConflictingPreBoundAddressesAreDetected(t *testing.T) {
	varX := Variable("X")
	varY := Variable("Y")

	rnd := rand.New(0)
	generator := NewStateGenerator()
	generator.BindToSelfAddress(varX)
	generator.BindToSelfAddress(varY)
	assignment := Assignment{
		varX: NewU256(1),
		varY: NewU256(2),
	}
	_, err := generator.generateWith(rnd, assignment)
	if !errors.Is(err, ErrUnsatisfiable) {
		t.Errorf("unsatisfiable constraint not detected, got %v", err)
	}
}

func TestStateGenerator_SelfAddressConstraintsAndVariablesCanBeCombined(t *testing.T) {
	addr := tosca.Address{1, 2, 3}
	varX := Variable("X")

	rnd := rand.New(0)
	generator := NewStateGenerator()
	generator.SetSelfAddress(addr)
	generator.BindToSelfAddress(varX)
	assignment := Assignment{}
	state, err := generator.generateWith(rnd, assignment)
	if err != nil {
		t.Fatalf("unexpected error during build: %v", err)
	}
	if want, got := addr, state.CallContext.AccountAddress; want != got {
		t.Errorf("unexpected account address, wanted %d, got %d", want, got)
	}
	want := NewU256FromBytes(addr[:]...)
	if got := assignment[varX]; want != got {
		t.Errorf("unexpected account address for %s, wanted %d, got %d", varX, want, got)
	}
}

////////////////////////////////////////////////////////////
// Clone / Restore

func TestStateGenerator_CloneCopiesBuilderState(t *testing.T) {
	original := NewStateGenerator()
	original.SetStatus(st.Reverted)
	original.SetRevision(tosca.R10_London)
	original.SetPc(4)
	original.SetGas(5)
	original.SetGasRefund(6)
	original.BindValue(Variable("x"), NewU256(12))
	original.BindToSelfAddress(Variable("Y"))
	original.SetSelfAddress(tosca.Address{1})

	clone := original.Clone()

	if want, got := original.String(), clone.String(); want != got {
		t.Errorf("invalid clone, wanted %s, got %s", want, got)
	}
}

func TestStateGenerator_ClonesAreIndependent(t *testing.T) {
	base := NewStateGenerator()
	base.SetPc(4)

	clone1 := base.Clone()
	clone1.SetStatus(st.Reverted)
	clone1.SetRevision(tosca.R10_London)
	clone1.SetGas(5)
	clone1.SetGasRefund(6)
	clone1.SetCodeOperation(20, vm.ADD)
	clone1.AddStackSizeLowerBound(2)
	clone1.AddStackSizeUpperBound(200)
	clone1.BindValue(Variable("x"), NewU256(12))
	clone1.BindToSelfAddress(Variable("Y"))
	clone1.SetSelfAddress(tosca.Address{1})
	clone1.MustBeSelfDestructed()

	clone2 := base.Clone()
	clone2.SetStatus(st.Running)
	clone2.SetRevision(tosca.R09_Berlin)
	clone2.SetGas(7)
	clone2.SetGasRefund(8)
	clone2.SetCodeOperation(30, vm.ADD)
	clone2.AddStackSizeLowerBound(3)
	clone2.AddStackSizeUpperBound(300)
	clone2.BindTransientStorageToZero("x")
	clone2.BindValue(Variable("y"), NewU256(14))
	clone2.BindToSelfAddress(Variable("w"))
	clone2.SetSelfAddress(tosca.Address{2})
	clone2.MustNotBeSelfDestructed()
	clone2.IsPresentBlobHashIndex(Variable("z"))

	checkPrint := func(clone *StateGenerator, want []string) {
		t.Helper()
		str := clone.String()
		if len(str) < 2 {
			t.Fatalf("unexpected print format: %v", str)
		}
		str = str[1 : len(str)-1]
		if got := strings.Split(str, ","); !slices.Equal(want, got) {
			t.Errorf("invalid clone, wanted %s, got %s", want, got)
		}
	}

	checkPrint(clone1, []string{
		"$x=0000000000000000 0000000000000000 0000000000000000 000000000000000c",
		"status=reverted",
		"pc=4",
		"selfAddress=$Y",
		"selfAddress=0x0100000000000000000000000000000000000000",
		"gas=5",
		"gasRefund=6",
		"code={op[20]=ADD}",
		"stack={2≤size≤200}",
		"memory={}",
		"storage={}",
		"transient={}",
		"accounts={}",
		"callContext={}",
		"callJournal={}",
		"blockContext=2000≤BlockNumber≤2999",
		"selfdestruct={mustBeSelfDestructed}",
		"transactionContext={true}",
	})

	checkPrint(clone2, []string{
		"$y=0000000000000000 0000000000000000 0000000000000000 000000000000000e",
		"status=running",
		"pc=4",
		"selfAddress=$w",
		"selfAddress=0x0200000000000000000000000000000000000000",
		"gas=7",
		"gasRefund=8",
		"code={op[30]=ADD}",
		"stack={3≤size≤300}",
		"memory={}",
		"storage={}",
		"transient={transient[$x]=0}",
		"accounts={}",
		"callContext={}",
		"callJournal={}",
		"blockContext=1000≤BlockNumber≤1999",
		"selfdestruct={mustNotBeSelfDestructed}",
		"transactionContext={$z < len(blobHashes)}",
	})
}

func TestStateGenerator_CloneCanBeUsedToResetBuilder(t *testing.T) {

	tests := map[string]struct {
		modify func(*StateGenerator)
	}{
		"status":            {modify: func(gen *StateGenerator) { gen.SetStatus(st.Reverted) }},
		"read-only":         {modify: func(gen *StateGenerator) { gen.SetReadOnly(false) }},
		"pc":                {modify: func(gen *StateGenerator) { gen.SetPc(4) }},
		"pc-bind":           {modify: func(gen *StateGenerator) { gen.BindPc("PC") }},
		"gas":               {modify: func(gen *StateGenerator) { gen.SetGas(5) }},
		"gasRefund":         {modify: func(gen *StateGenerator) { gen.SetGasRefund(6) }},
		"bind-value":        {modify: func(gen *StateGenerator) { gen.BindValue(Variable("x"), NewU256(12)) }},
		"code":              {modify: func(gen *StateGenerator) { gen.SetCodeOperation(20, vm.ADD) }},
		"stack":             {modify: func(gen *StateGenerator) { gen.AddStackSizeLowerBound(2); gen.AddStackSizeUpperBound(200) }},
		"storage":           {modify: func(gen *StateGenerator) { gen.BindIsStorageWarm("warmStorage") }},
		"accounts":          {modify: func(gen *StateGenerator) { gen.BindToWarmAddress("warmAccount") }},
		"revision":          {modify: func(gen *StateGenerator) { gen.SetRevision(tosca.R10_London) }},
		"selfDestruct":      {modify: func(gen *StateGenerator) { gen.MustBeSelfDestructed() }},
		"selfAddress-var":   {modify: func(gen *StateGenerator) { gen.BindToSelfAddress(Variable("Y")) }},
		"selfAddress-const": {modify: func(gen *StateGenerator) { gen.SetSelfAddress(tosca.Address{1}) }},
		// the following fields can not be tested as they do not have any internal state to be compared
		// "memory": {setup: func(gen *StateGenerator) { gen.memoryGen = nil }},
		// "call": {setup: func(gen *StateGenerator) { gen.callContextGen = &CallContextGenerator{} }},
		// "call-journal": {setup: func(gen *StateGenerator) { gen.callJournalGen = &CallJournalGenerator{} }},
		// "transaction": {setup: func(gen *StateGenerator) { gen.transactionContextGen = &TransactionContextGenerator{} }},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {

			gen := NewStateGenerator()
			backup := gen.Clone()

			test.modify(gen)
			if base, modified := backup.String(), gen.String(); base == modified {
				t.Errorf("clones are not independent")
			}

			gen.Restore(backup)

			if want, got := backup.String(), gen.String(); want != got {
				t.Errorf("restore did not work, wanted %s, got %s", want, got)
			}

		})
	}
}

// //////////////////////////////////////////////////////////

func genRandomState(t *testing.T) *st.State {
	t.Helper()
	gen := NewStateGenerator()
	rnd := rand.New(0)
	state, err := gen.Generate(rnd)
	if err != nil {
		t.Fatalf("error generating new state: %v", err)
	}
	return state
}

func testDataBytes(data Bytes, name string, t *testing.T) {
	t.Helper()
	allzeros := true
	for b := range data.Get(0, uint64(data.Length())) {
		if b != 0 {
			allzeros = false
			break
		}
	}
	if allzeros {
		t.Errorf("failed to generate a non-zero %v", name)
	}
}

// //////////////////////////////////////////////////////////
// Call data
// Last Call Return data

func TestStateGenerator_DataGeneration(t *testing.T) {
	state := genRandomState(t)
	testDataBytes(state.CallData, "call data", t)
	testDataBytes(state.LastCallReturnData, "last call return data", t)
}

// //////////////////////////////////////////////////////////
// Return data should always be empty

func TestStateGenerator_ReturnDataShouldBeEmpty(t *testing.T) {
	state := genRandomState(t)
	if want, got := 0, state.ReturnData.Length(); want != got {
		t.Errorf("unexpected length of generated return data, wanted %d, got %d", want, got)
	}
}

// //////////////////////////////////////////////////////////
// Block number hashes
func TestStateGenerator_BlockNumberHashes(t *testing.T) {
	newHashes := []tosca.Hash{}
	state := genRandomState(t)
	for i := uint64(0); i < 256; i++ {
		hashi := state.RecentBlockHashes.Get(i)
		if slices.Contains(newHashes, hashi) {
			t.Errorf("unexpected hash value, should be unique %v", hashi)
		}
		newHashes = append(newHashes, hashi)
	}
}

func TestStateGenerator_VariousContraints(t *testing.T) {

	tests := map[string]struct {
		setup func(*StateGenerator)
		check func(*StateGenerator, *testing.T) bool
	}{
		"add-code-op": {
			setup: func(gen *StateGenerator) { gen.AddCodeOperation(Variable("a"), vm.ADD) },
			check: func(gen *StateGenerator, t *testing.T) bool {
				got, want := gen.codeGen.varOps, []varOpConstraint{{Variable("a"), vm.ADD}}
				return slices.Equal(want, got)
			}},
		"add-is-code": {
			setup: func(gen *StateGenerator) { gen.AddIsCode(Variable("a")) },
			check: func(gen *StateGenerator, t *testing.T) bool {
				got, want := gen.codeGen.varIsCodeConstraints, []varIsCodeConstraint{{Variable("a")}}
				return slices.Equal(want, got)
			},
		},
		"add-is-data": {
			setup: func(gen *StateGenerator) { gen.AddIsData(Variable("a")) },
			check: func(gen *StateGenerator, t *testing.T) bool {
				got, want := gen.codeGen.varIsDataConstraints, []varIsDataConstraint{{Variable("a")}}
				return slices.Equal(want, got)
			},
		},
		"set-stack-size": {
			setup: func(gen *StateGenerator) { gen.SetStackSize(10) },
			check: func(gen *StateGenerator, t *testing.T) bool {
				return gen.stackGen.size.max == 10 && gen.stackGen.size.min == 10
			},
		},
		"set-stack-value": {
			setup: func(gen *StateGenerator) { gen.SetStackValue(0, NewU256(1)) },
			check: func(gen *StateGenerator, t *testing.T) bool {
				return gen.stackGen.constValues[0] == constValueConstraint{0, NewU256(1)}
			},
		},
		"bind-stack-value": {
			setup: func(gen *StateGenerator) { gen.BindStackValue(0, Variable("a")) },
			check: func(gen *StateGenerator, t *testing.T) bool {
				return gen.stackGen.varValues[0] == varValueConstraint{0, Variable("a")}
			},
		},
		"bind-storage-config": {
			setup: func(gen *StateGenerator) {
				gen.BindStorageConfiguration(tosca.StorageAdded, Variable("a"), Variable("b"))
			},
			check: func(gen *StateGenerator, t *testing.T) bool {
				return slices.Contains(gen.storageGen.cfg, storageConfigConstraint{tosca.StorageAdded, Variable("a"), Variable("b")})
			},
		},
		"bind-is-storage-cold": {
			setup: func(gen *StateGenerator) { gen.BindIsStorageCold(Variable("a")) },
			check: func(gen *StateGenerator, t *testing.T) bool {
				return slices.Contains(gen.storageGen.warmCold, warmColdConstraint{Variable("a"), false})
			},
		},
		"bind-transient-storage-non-zero": {
			setup: func(gen *StateGenerator) { gen.BindTransientStorageToNonZero(Variable("a")) },
			check: func(gen *StateGenerator, t *testing.T) bool {
				return gen.transientStorageGen.nonZeroConstraints[Variable("a")]
			},
		},
		"bind-cold-address": {
			setup: func(gen *StateGenerator) { gen.BindToColdAddress(Variable("a")) },
			check: func(gen *StateGenerator, t *testing.T) bool {
				return slices.Contains(gen.accountsGen.warmCold, warmColdConstraint{Variable("a"), false})
			},
		},
		"one-of-last-256-blocks": {
			setup: func(gen *StateGenerator) { gen.RestrictVariableToOneOfTheLast256Blocks(Variable("a")) },
			check: func(gen *StateGenerator, t *testing.T) bool {
				return gen.blockContextGen.rangeConstraints[Variable("a")]
			},
		},
		"none-of-last-256-blocks": {
			setup: func(gen *StateGenerator) { gen.RestrictVariableToNoneOfTheLast256Blocks(Variable("a")) },
			check: func(gen *StateGenerator, t *testing.T) bool {
				return !gen.blockContextGen.rangeConstraints[Variable("a")]
			},
		},
		"set-block-number-offfset": {
			setup: func(gen *StateGenerator) { gen.SetBlockNumberOffsetValue(Variable("a"), 10) },
			check: func(gen *StateGenerator, t *testing.T) bool {
				return gen.blockContextGen.valueConstraint[Variable("a")] == 10
			},
		},
		"absent-blob-hash-index": {
			setup: func(gen *StateGenerator) { gen.IsAbsentBlobHashIndex(Variable("a")) },
			check: func(gen *StateGenerator, t *testing.T) bool {
				return !gen.transactionContextGen.blobHashVariables[Variable("a")]
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			gen := NewStateGenerator()
			test.setup(gen)
			if !test.check(gen, t) {
				t.Errorf("unexpected state generator")
			}
		})
	}
}
