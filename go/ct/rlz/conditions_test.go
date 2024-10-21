// Copyright (c) 2024 Fantom Foundation
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at fantom.foundation/bsl11.
//
// Change Date: 2028-4-16
//
// On the date above, in accordance with the Business Source License, use of
// this software will be governed by the GNU Lesser General Public License v3.

package rlz

import (
	"errors"
	"math"
	"testing"

	. "github.com/Fantom-foundation/Tosca/go/ct/common"
	"github.com/Fantom-foundation/Tosca/go/ct/gen"
	"github.com/Fantom-foundation/Tosca/go/ct/st"
	"github.com/Fantom-foundation/Tosca/go/tosca"
	"github.com/Fantom-foundation/Tosca/go/tosca/vm"
	"pgregory.net/rand"
)

func TestCondition_Check(t *testing.T) {
	newStateWithStatusAndPc := func(status st.StatusCode, pc uint16) *st.State {
		state := st.NewState(st.NewCode([]byte{}))
		state.Status = status
		state.Pc = pc
		return state
	}

	newStateWithPc := func(pc uint16) *st.State {
		state := st.NewState(st.NewCode([]byte{}))
		state.Pc = pc
		return state
	}

	newStateWithPcAndCode := func(pc uint16, ops ...byte) *st.State {
		state := st.NewState(st.NewCode(ops))
		state.Pc = pc
		return state
	}

	newStateWithStack := func(stack *st.Stack) *st.State {
		state := st.NewState(st.NewCode([]byte{byte(vm.PUSH1), byte(0)}))
		state.Stack = stack
		return state
	}

	tests := []struct {
		condition Condition
		valid     *st.State
		invalid   *st.State
	}{
		{Eq(Pc(), NewU256(42)), newStateWithPc(42), newStateWithPc(41)},
		{Ne(Pc(), NewU256(42)), newStateWithPc(41), newStateWithPc(42)},
		{Lt(Pc(), NewU256(42)), newStateWithPc(41), newStateWithPc(42)},
		{Lt(Pc(), NewU256(42)), newStateWithPc(41), newStateWithPc(43)},
		{Le(Pc(), NewU256(42)), newStateWithPc(41), newStateWithPc(43)},
		{Le(Pc(), NewU256(42)), newStateWithPc(42), newStateWithPc(43)},
		{Le(Pc(), NewU256(42)), newStateWithPc(42), newStateWithPc(44)},
		{Gt(Pc(), NewU256(42)), newStateWithPc(43), newStateWithPc(42)},
		{Gt(Pc(), NewU256(42)), newStateWithPc(43), newStateWithPc(41)},
		{Ge(Pc(), NewU256(42)), newStateWithPc(42), newStateWithPc(41)},
		{Ge(Pc(), NewU256(42)), newStateWithPc(43), newStateWithPc(41)},
		{Ge(Pc(), NewU256(42)), newStateWithPc(43), newStateWithPc(40)},
		{And(Eq(Status(), st.Reverted), Eq(Pc(), NewU256(42))), newStateWithStatusAndPc(st.Reverted, 42), newStateWithStatusAndPc(st.Stopped, 42)},
		{And(Eq(Status(), st.Reverted), Eq(Pc(), NewU256(42))), newStateWithStatusAndPc(st.Reverted, 42), newStateWithStatusAndPc(st.Reverted, 41)},
		{IsCode(Pc()), newStateWithPcAndCode(1, byte(vm.ADD), byte(vm.ADD)), newStateWithPcAndCode(1, byte(vm.PUSH1), byte(0))},
		{IsCode(Pc()), newStateWithPcAndCode(2, byte(vm.ADD), byte(vm.ADD)), newStateWithPcAndCode(1, byte(vm.PUSH1), byte(0))},
		{IsCode(Param(0)), newStateWithStack(st.NewStack(NewU256(1, 1))), newStateWithStack(st.NewStack(NewU256(1)))},
		{IsData(Pc()), newStateWithPcAndCode(1, byte(vm.PUSH1), byte(0)), newStateWithPcAndCode(1, byte(vm.ADD), byte(vm.ADD))},
		{IsData(Pc()), newStateWithPcAndCode(1, byte(vm.PUSH1), byte(0)), newStateWithPcAndCode(2, byte(vm.ADD), byte(vm.ADD))},
		{IsData(Param(0)), newStateWithStack(st.NewStack(NewU256(1))), newStateWithStack(st.NewStack(NewU256(1, 1)))},
	}

	for _, test := range tests {
		valid, err := test.condition.Check(test.valid)
		if err != nil {
			t.Errorf("Condition check error %v", err)
		}
		if !valid {
			t.Errorf("Condition %v should be valid for\n%v", test.condition, test.valid)
		}

		invalid, err := test.condition.Check(test.invalid)
		if err != nil {
			t.Errorf("Condition check error %v", err)
		}
		if invalid {
			t.Errorf("Condition %v should not be valid for\n%v", test.condition, test.invalid)
		}
	}
}

func TestCondition_CheckRevisions(t *testing.T) {
	state := st.NewState(st.NewCode([]byte{}))
	state.Revision = tosca.R10_London

	validConditions := []Condition{
		AnyKnownRevision(),
		IsRevision(tosca.R10_London),
		RevisionBounds(tosca.R10_London, tosca.R10_London),
		RevisionBounds(tosca.R07_Istanbul, tosca.R10_London),
	}
	for _, cond := range validConditions {
		isValid, err := cond.Check(state)
		if err != nil {
			t.Fatal(err)
		}
		if !isValid {
			t.Fatalf("valid condition check failed %v", cond)
		}
	}

	invalidConditions := []Condition{
		IsRevision(tosca.R09_Berlin),
		IsRevision(R99_UnknownNextRevision),
		RevisionBounds(tosca.R07_Istanbul, tosca.R09_Berlin),
	}
	for _, cond := range invalidConditions {
		isValid, err := cond.Check(state)
		if err != nil {
			t.Fatal(err)
		}
		if isValid {
			t.Fatalf("invalid condition check failed %v", cond)
		}
	}
}

func TestCondition_UnknownNextRevisionIsNotAnyKnownIsRevision(t *testing.T) {
	state := st.NewState(st.NewCode([]byte{}))
	state.Revision = R99_UnknownNextRevision

	isValid, err := AnyKnownRevision().Check(state)
	if err != nil {
		t.Fatal(err)
	}
	if isValid {
		t.Fatal("AnyKnownRevision matches UnknownNextRevision")
	}
}

func TestCondition_CheckStorageConfiguration(t *testing.T) {
	allConfigs := []tosca.StorageStatus{
		tosca.StorageAssigned,
		tosca.StorageAdded,
		tosca.StorageAddedDeleted,
		tosca.StorageDeletedRestored,
		tosca.StorageDeletedAdded,
		tosca.StorageDeleted,
		tosca.StorageModified,
		tosca.StorageModifiedDeleted,
		tosca.StorageModifiedRestored,
	}

	tests := []struct {
		config        tosca.StorageStatus
		org, cur, new U256
	}{
		{tosca.StorageAssigned, NewU256(1), NewU256(2), NewU256(3)},
		{tosca.StorageAdded, NewU256(0), NewU256(0), NewU256(1)},
		{tosca.StorageAddedDeleted, NewU256(0), NewU256(1), NewU256(0)},
		{tosca.StorageDeletedRestored, NewU256(1), NewU256(0), NewU256(1)},
		{tosca.StorageDeletedAdded, NewU256(1), NewU256(0), NewU256(2)},
		{tosca.StorageDeleted, NewU256(1), NewU256(1), NewU256(0)},
		{tosca.StorageModified, NewU256(1), NewU256(1), NewU256(2)},
		{tosca.StorageModifiedDeleted, NewU256(1), NewU256(2), NewU256(0)},
		{tosca.StorageModifiedRestored, NewU256(1), NewU256(2), NewU256(1)},
	}

	for _, test := range tests {
		t.Run(test.config.String(), func(t *testing.T) {
			state := st.NewState(st.NewCode([]byte{}))

			key := NewU256(42)
			state.Storage.SetOriginal(key, test.org)
			state.Storage.SetCurrent(key, test.cur)

			state.Stack.Push(test.new)
			state.Stack.Push(key)

			for _, config := range allConfigs {
				satisfied, err := StorageConfiguration(config, Param(0), Param(1)).Check(state)
				if err != nil {
					t.Fatal(err)
				}
				if config == test.config && !satisfied {
					t.Fatalf("StorageConfiguration %v is not satisfied for %v", config, state)
				}
				if config != test.config && satisfied {
					t.Fatalf("StorageConfiguration %v should not be satisfied for %v", config, state)
				}
			}
		})
	}
}

func TestCondition_CheckForEmptyAccount(t *testing.T) {
	state := st.NewState(st.NewCode([]byte{}))
	state.Stack = st.NewStack(NewU256(42))
	condition := AccountIsEmpty(Param(0))

	empty, err := condition.Check(state)
	if err != nil {
		t.Fatal(err)
	}
	if !empty {
		t.Error("account should be considered empty")
	}

	state.Accounts = st.NewAccountsBuilder().
		SetBalance(NewAddress(NewU256(42)), NewU256(12)).
		Build()

	empty, err = condition.Check(state)
	if err != nil {
		t.Fatal(err)
	}
	if empty {
		t.Error("account should be considered non-empty")
	}
}

func TestCondition_CheckForNoneEmptyAccount(t *testing.T) {
	state := st.NewState(st.NewCode([]byte{}))
	state.Stack = st.NewStack(NewU256(42))
	condition := AccountIsNotEmpty(Param(0))

	noneEmpty, err := condition.Check(state)
	if err != nil {
		t.Fatal(err)
	}
	if noneEmpty {
		t.Error("account should be considered empty")
	}

	state.Accounts = st.NewAccountsBuilder().
		SetBalance(NewAddress(NewU256(42)), NewU256(12)).
		Build()

	noneEmpty, err = condition.Check(state)
	if err != nil {
		t.Fatal(err)
	}
	if !noneEmpty {
		t.Error("account should be considered non-empty")
	}
}

func TestCondition_CheckWarmCold(t *testing.T) {
	state := st.NewState(st.NewCode([]byte{}))
	state.Pc = 42
	state.Storage.MarkWarm(NewU256(42))

	isCold, err := IsStorageCold(Pc()).Check(state)
	if err != nil {
		t.Fatal(err)
	}
	if isCold {
		t.Fatal("Storage key is cold, should be warm")
	}

	isWarm, err := IsStorageWarm(Pc()).Check(state)
	if err != nil {
		t.Fatal(err)
	}
	if !isWarm {
		t.Fatal("Storage key is not warm")
	}
}

func TestCondition_String(t *testing.T) {
	tests := []struct {
		condition Condition
		result    string
	}{
		{And(), "true"},
		{And(And(), And()), "true"},
		{Eq(Pc(), NewU256(42)), "PC = 0000000000000000 0000000000000000 0000000000000000 000000000000002a"},
		{Ne(Pc(), NewU256(42)), "PC ≠ 0000000000000000 0000000000000000 0000000000000000 000000000000002a"},
		{Lt(Pc(), NewU256(42)), "PC < 0000000000000000 0000000000000000 0000000000000000 000000000000002a"},
		{Le(Pc(), NewU256(42)), "PC ≤ 0000000000000000 0000000000000000 0000000000000000 000000000000002a"},
		{Gt(Pc(), NewU256(42)), "PC > 0000000000000000 0000000000000000 0000000000000000 000000000000002a"},
		{Ge(Pc(), NewU256(42)), "PC ≥ 0000000000000000 0000000000000000 0000000000000000 000000000000002a"},
		{IsCode(Pc()), "isCode[PC]"},
		{IsData(Pc()), "isData[PC]"},
		{And(Eq(Status(), st.Running), Eq(Status(), st.Failed)), "status = running ∧ status = failed"},
		{IsRevision(tosca.R10_London), "revision(London)"},
		{RevisionBounds(tosca.R09_Berlin, tosca.R11_Paris), "revision(Berlin-Paris)"},
		{HasNotSelfDestructed(), "hasNotSelfDestructed()"},
	}

	for _, test := range tests {
		if got, want := test.condition.String(), test.result; got != want {
			t.Errorf("unexpected print, wanted %s, got %s", want, got)
		}
	}
}

func TestHasSelfDestructedCondition_CheckSelfDestructed(t *testing.T) {
	state := st.NewState(st.NewCode([]byte{}))
	state.CallContext.AccountAddress = NewAddress(NewU256(0x01))
	state.HasSelfDestructed = true

	hasSelfDestructed, err := HasSelfDestructed().Check(state)
	if err != nil {
		t.Fatal(err)
	}
	if !hasSelfDestructed {
		t.Fatal("hasSelfDestructed check failed")
	}

	state.HasSelfDestructed = false

	hasNotSelfDestructed, err := HasNotSelfDestructed().Check(state)
	if err != nil {
		t.Fatal(err)
	}
	if !hasNotSelfDestructed {
		t.Fatal("hasNotSelfDestructed check failed")
	}
}

func TestCondition_InOutRange256FromCurrentBlock_Check(t *testing.T) {
	gen := gen.NewStateGenerator()
	rnd := rand.New(0)
	state, err := gen.Generate(rnd)
	if err != nil {
		t.Fatalf("%v", err)
	}

	tests := map[string]struct {
		condition Condition
		offset    int64
		want      bool
	}{
		"checkInWantIn-1": {
			condition: InRange256FromCurrentBlock(Param(0)),
			offset:    -1,
			want:      false,
		},
		"checkInWantIn0": {
			condition: InRange256FromCurrentBlock(Param(0)),
			offset:    0,
			want:      false,
		},
		"checkInWantIn1": {
			condition: InRange256FromCurrentBlock(Param(0)),
			offset:    1,
			want:      true,
		},
		"checkInWantIn255": {
			condition: InRange256FromCurrentBlock(Param(0)),
			offset:    255,
			want:      true,
		},
		"checkInWantIn256": {
			condition: InRange256FromCurrentBlock(Param(0)),
			offset:    256,
			want:      true,
		},
		"checkInWantIn257": {
			condition: InRange256FromCurrentBlock(Param(0)),
			offset:    257,
			want:      false,
		},
		"checkOutWantOut-1": {
			condition: OutOfRange256FromCurrentBlock(Param(0)),
			offset:    -1,
			want:      true,
		},
		"checkOutWantOut0": {
			condition: OutOfRange256FromCurrentBlock(Param(0)),
			offset:    0,
			want:      true,
		},
		"checkOutWantIn1": {
			condition: OutOfRange256FromCurrentBlock(Param(0)),
			offset:    1,
			want:      false,
		},
		"checkOutWantIn255": {
			condition: OutOfRange256FromCurrentBlock(Param(0)),
			offset:    255,
			want:      false,
		},
		"checkOutWantIn256": {
			condition: OutOfRange256FromCurrentBlock(Param(0)),
			offset:    256,
			want:      false,
		},
		"checkOutWantOut257": {
			condition: OutOfRange256FromCurrentBlock(Param(0)),
			offset:    257,
			want:      true,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			state.Stack.Push(NewU256(uint64(int64(state.BlockContext.BlockNumber) - test.offset)))
			got, err := test.condition.Check(state)
			if err != nil {
				t.Fatal(err)
			}
			if test.want != got {
				t.Fatal("block number is not within range")
			}
		})
	}

}

func TestCondition_CheckUnsatisfiableTransientStorageBindings(t *testing.T) {
	conditionNonZero := BindTransientStorageToNonZero(Param(0))
	conditionZero := BindTransientStorageToZero(Param(0))

	gen := gen.NewStateGenerator()
	rnd := rand.New(0)
	conditionNonZero.Restrict(gen)
	conditionZero.Restrict(gen)
	_, err := gen.Generate(rnd)
	if err == nil {
		t.Errorf("Expected unsatisfiable condition, but got nil")
	}
}

func TestCondition_BlobHashes_check(t *testing.T) {
	tests := map[string]struct {
		condition Condition
		setup     func(*st.State)
	}{
		"hasBlobHash": {
			condition: HasBlobHash(Param(0)),
			setup: func(state *st.State) {
				state.TransactionContext.BlobHashes = []tosca.Hash{{0}}
				state.Stack.Push(NewU256(0))
			},
		},
		"hasNotBlobHash": {
			condition: HasNoBlobHash(Param(0)),
			setup: func(state *st.State) {
				state.TransactionContext.BlobHashes = []tosca.Hash{{0}}
				state.Stack.Push(NewU256(1))
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			for i := 0; i < 1000; i++ {
				state := st.NewState(st.NewCode([]byte{}))
				test.setup(state)

				if checked, err := test.condition.Check(state); err != nil || !checked {
					t.Errorf("failed to check condition: %v", err)
				}
			}
		})
	}
}

func TestCondition_ConflictingBlobHashesConditionsProduceUnsatisfiableGenerator(t *testing.T) {
	tests := map[string]struct {
		condition Condition
		setup     func(*st.State)
	}{
		"hasBlobHash-first": {
			condition: And(HasBlobHash(Param(0)), HasNoBlobHash(Param(0))),
			setup: func(state *st.State) {
				state.Stack.Push(NewU256(0))
			},
		},
		"hasNoBlobHash-first": {
			condition: And(HasNoBlobHash(Param(0)), HasBlobHash(Param(0))),
			setup: func(state *st.State) {
				state.Stack.Push(NewU256(0))
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			state := st.NewState(st.NewCode([]byte{}))
			test.setup(state)

			gen := gen.NewStateGenerator()
			test.condition.Restrict(gen)
			_, err := gen.Generate(rand.New(0))
			if err == nil {
				t.Errorf("expected unsatisfiable condition")
			}
		})
	}
}

func TestCondition_BlobHashesProducesGetTestValues(t *testing.T) {
	tests := map[string]Condition{
		"hasBlobHash":   HasBlobHash(Param(0)),
		"hasNoBlobHash": HasNoBlobHash(Param(0)),
	}

	for name, condition := range tests {
		t.Run(name, func(t *testing.T) {
			var matchCovered, failCovered bool
			for _, value := range condition.GetTestValues() {
				gen := gen.NewStateGenerator()
				value.Restrict(gen)
				state, err := gen.Generate(rand.New(0))
				if err != nil {
					t.Fatalf("failed to build state: %v", err)
				}
				if matches, err := condition.Check(state); err != nil {
					t.Errorf("failed to check condition: %v", err)
				} else if matches {
					matchCovered = true
				} else {
					failCovered = true
				}
			}
			if !matchCovered {
				t.Errorf("no test value matched the condition")
			}
			if !failCovered {
				t.Errorf("no test value failed the condition")
			}
		})
	}
}

func TestCondition_ConflictingAccountWarmConditionsAreUnsatisfiable(t *testing.T) {
	isWarm := IsAddressWarm(Param(0))
	isCold := IsAddressCold(Param(0))

	generator := gen.NewStateGenerator()
	isWarm.Restrict(generator)
	isCold.Restrict(generator)
	_, err := generator.Generate(rand.New(0))
	if !errors.Is(err, gen.ErrUnsatisfiable) {
		t.Errorf("expected unsatisfiable condition")
	}
}

func TestCondition_RevisionBoundInvalidBounds(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("expected panic")
		}
	}()
	RevisionBounds(tosca.R10_London, tosca.R09_Berlin)
}

func TestCondition_IsRevisionCondition(t *testing.T) {

	tests := map[string]struct {
		condition  Condition
		isRevision bool
	}{
		"IsRevision":       {IsRevision(tosca.R10_London), true},
		"RevisionBounds":   {RevisionBounds(tosca.R10_London, tosca.R10_London), true},
		"AnyKnownRevision": {AnyKnownRevision(), true},
		"IsAddressWarm":    {IsAddressWarm(Param(0)), false},
		"IsCode":           {IsCode(Param(0)), false},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			if got, want := IsRevisionCondition(test.condition), test.isRevision; got != want {
				t.Errorf("condition %v is not a revision condition. got %v, want %v", test.condition, got, want)
			}
		})
	}
}

func TestCondition_GetTestValues(t *testing.T) {

	inOutofRangeTestValues := []any{math.MinInt64, -1, 0, 1, 255, 256, 257, math.MaxInt64}

	tests := []struct {
		condition Condition
		want      []any
	}{
		{IsRevision(tosca.R10_London), []any{tosca.R10_London}},
		{AnyKnownRevision(), []any{tosca.R07_Istanbul, tosca.R09_Berlin, tosca.R10_London, tosca.R11_Paris, tosca.R12_Shanghai, tosca.R13_Cancun, R99_UnknownNextRevision}},
		{InRange256FromCurrentBlock(Param(0)), inOutofRangeTestValues},
		{OutOfRange256FromCurrentBlock(Param(0)), inOutofRangeTestValues},
		{IsCode(Param(0)), []any{true, false}},
		{IsData(Param(0)), []any{true, false}},
		{IsStorageWarm(Param(0)), []any{true, false}},
		{IsStorageCold(Param(0)), []any{true, false}},
		{StorageConfiguration(tosca.StorageAssigned, Param(0), Param(1)), []any{true}},
		{BindTransientStorageToNonZero(Param(0)), []any{true, false}},
		{BindTransientStorageToZero(Param(0)), []any{true, false}},
		{IsAddressWarm(Param(0)), []any{true}},
		{IsAddressCold(Param(0)), []any{false}},
		{HasSelfDestructed(), []any{true, false}},
		{HasNotSelfDestructed(), []any{true, false}},
	}

	for _, test := range tests {
		t.Run(test.condition.String(), func(t *testing.T) {
			if got, want := test.condition.GetTestValues(), test.want; len(got) != len(want) {
				t.Errorf("unexpected test values. wanted %v, got %v", want, got)
			}
		})
	}
}

func TestCondition_RestricAndCheck(t *testing.T) {

	tests := []Condition{
		IsRevision(tosca.R10_London),
		HasSelfDestructed(),
		HasNotSelfDestructed(),
		InRange256FromCurrentBlock(Param(0)),
		OutOfRange256FromCurrentBlock(Param(0)),
		BindTransientStorageToNonZero(Param(0)),
		BindTransientStorageToZero(Param(0)),
		HasBlobHash(Param(0)),
		HasNoBlobHash(Param(0)),
		IsAddressWarm(Param(0)),
		IsAddressCold(Param(0)),
		Ne(Gas(), 1),
		Lt(Gas(), 2),
		Le(Gas(), 2),
		Gt(Gas(), 0),
		Ge(Gas(), 0),
		IsCode(Param(0)),
		IsData(Param(0)),
		IsStorageWarm(Param(0)),
		IsStorageCold(Param(0)),
		StorageConfiguration(tosca.StorageAssigned, Param(0), Param(1)),
	}

	for _, condition := range tests {
		t.Run(condition.String(), func(t *testing.T) {
			gen := gen.NewStateGenerator()
			condition.Restrict(gen)
			state, err := gen.Generate(rand.New(0))
			if err != nil {
				t.Errorf("failed to build state: %v", err)
			}
			if checked, err := condition.Check(state); err != nil || !checked {
				t.Errorf("failed to check condition: %v", err)
			}
		})
	}
}

func TestCondition_CheckFail(t *testing.T) {

	tests := []struct {
		condition  Condition
		breakState func(st.State)
	}{
		// only Param and Op can return errors on eval, which is where the errors from check come from.
		{Eq(Param(1), NewU256(1)), func(s st.State) { s.Stack.Resize(0) }},
		{Ne(Param(1), NewU256(1)), func(s st.State) { s.Stack.Resize(0) }},
		// Op and Param expressions only support equality constraints,
		// hence the inequality constraints cannot produce error on check with the current expressions.
		// {Lt(Param(1), NewU256(1)), func(s st.State) { s.Stack.Resize(0) }},
		// {Le(Param(1), NewU256(1)), func(s st.State) { s.Stack.Resize(0) }},
		// {Gt(Param(1), NewU256(1)), func(s st.State) { s.Stack.Resize(0) }},
		// {Ge(Param(1), NewU256(1)), func(s st.State) { s.Stack.Resize(0) }},
		{IsCode(Param(1)), func(s st.State) { s.Stack.Resize(0) }},
		{IsData(Param(1)), func(s st.State) { s.Stack.Resize(0) }},
		{IsStorageWarm(Param(1)), func(s st.State) { s.Stack.Resize(0) }},
		{IsStorageCold(Param(1)), func(s st.State) { s.Stack.Resize(0) }},
		{StorageConfiguration(tosca.StorageAssigned, Param(1), Param(2)), func(s st.State) { s.Stack.Resize(1) }},
		{StorageConfiguration(tosca.StorageAssigned, Param(1), Param(2)), func(s st.State) { s.Stack.Resize(0) }},
		{BindTransientStorageToNonZero(Param(1)), func(s st.State) { s.Stack.Resize(0) }},
		{BindTransientStorageToZero(Param(1)), func(s st.State) { s.Stack.Resize(0) }},
		{IsAddressWarm(Param(1)), func(s st.State) { s.Stack.Resize(0) }},
		{InRange256FromCurrentBlock(Param(1)), func(s st.State) { s.Stack.Resize(0) }},
		{OutOfRange256FromCurrentBlock(Param(1)), func(s st.State) { s.Stack.Resize(0) }},
		{IsAddressCold(Param(1)), func(s st.State) { s.Stack.Resize(0) }},
		{HasNoBlobHash(Param(1)), func(s st.State) { s.Stack.Resize(0) }},
	}

	for _, test := range tests {
		t.Run(test.condition.String(), func(t *testing.T) {
			gen := gen.NewStateGenerator()
			test.condition.Restrict(gen)
			state, err := gen.Generate(rand.New(0))
			if err != nil {
				t.Errorf("unexpected error generating state. %v", err)
			}
			test.breakState(*state)
			if checked, err := test.condition.Check(state); err == nil {
				t.Errorf("expected error checking condition. err %v. checked: %v", err, checked)
			}
		})
	}
}

////////////////////////////////////////////////////////////////////////////
// Benchmarks

func BenchmarkCondition_IsAddressWarmCheckWarm(b *testing.B) {
	state := st.NewState(st.NewCode([]byte{}))
	state.Accounts.MarkWarm(NewAddress(NewU256(42)))
	state.Stack.Push(NewU256(42))
	condition := IsAddressWarm(Param(0))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := condition.Check(state)
		if err != nil {
			b.Fatalf("invalid benchmark, check returned error %v", err)
		}
	}
}

func BenchmarkCondition_IsAddressWarmCheckCold(b *testing.B) {
	state := st.NewState(st.NewCode([]byte{}))
	state.Accounts.MarkWarm(NewAddress(NewU256(42)))
	state.Stack.Push(NewU256(1))
	condition := IsAddressWarm(Param(0))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := condition.Check(state)
		if err != nil {
			b.Fatalf("invalid benchmark, check returned error %v", err)
		}
	}
}
