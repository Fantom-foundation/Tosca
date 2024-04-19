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

package rlz

import (
	"testing"

	. "github.com/Fantom-foundation/Tosca/go/ct/common"
	"github.com/Fantom-foundation/Tosca/go/ct/gen"
	"github.com/Fantom-foundation/Tosca/go/ct/st"
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
		state := st.NewState(st.NewCode([]byte{byte(PUSH1), byte(0)}))
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
		{IsCode(Pc()), newStateWithPcAndCode(1, byte(ADD), byte(ADD)), newStateWithPcAndCode(1, byte(PUSH1), byte(0))},
		{IsCode(Pc()), newStateWithPcAndCode(2, byte(ADD), byte(ADD)), newStateWithPcAndCode(1, byte(PUSH1), byte(0))},
		{IsCode(Param(0)), newStateWithStack(st.NewStack(NewU256(1, 1))), newStateWithStack(st.NewStack(NewU256(1)))},
		{IsData(Pc()), newStateWithPcAndCode(1, byte(PUSH1), byte(0)), newStateWithPcAndCode(1, byte(ADD), byte(ADD))},
		{IsData(Pc()), newStateWithPcAndCode(1, byte(PUSH1), byte(0)), newStateWithPcAndCode(2, byte(ADD), byte(ADD))},
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
	state.Revision = R10_London

	validConditions := []Condition{
		AnyKnownRevision(),
		IsRevision(R10_London),
		RevisionBounds(R10_London, R10_London),
		RevisionBounds(R07_Istanbul, R10_London),
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
		IsRevision(R09_Berlin),
		IsRevision(R99_UnknownNextRevision),
		RevisionBounds(R07_Istanbul, R09_Berlin),
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
	allConfigs := []gen.StorageCfg{
		gen.StorageAssigned,
		gen.StorageAdded,
		gen.StorageAddedDeleted,
		gen.StorageDeletedRestored,
		gen.StorageDeletedAdded,
		gen.StorageDeleted,
		gen.StorageModified,
		gen.StorageModifiedDeleted,
		gen.StorageModifiedRestored,
	}

	tests := []struct {
		config        gen.StorageCfg
		org, cur, new U256
	}{
		{gen.StorageAssigned, NewU256(1), NewU256(2), NewU256(3)},
		{gen.StorageAdded, NewU256(0), NewU256(0), NewU256(1)},
		{gen.StorageAddedDeleted, NewU256(0), NewU256(1), NewU256(0)},
		{gen.StorageDeletedRestored, NewU256(1), NewU256(0), NewU256(1)},
		{gen.StorageDeletedAdded, NewU256(1), NewU256(0), NewU256(2)},
		{gen.StorageDeleted, NewU256(1), NewU256(1), NewU256(0)},
		{gen.StorageModified, NewU256(1), NewU256(1), NewU256(2)},
		{gen.StorageModifiedDeleted, NewU256(1), NewU256(2), NewU256(0)},
		{gen.StorageModifiedRestored, NewU256(1), NewU256(2), NewU256(1)},
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
	}

	for _, test := range tests {
		if got, want := test.condition.String(), test.result; got != want {
			t.Errorf("unexpected print, wanted %s, got %s", want, got)
		}
	}
}

func TestCondition_CheckSelfDestructed(t *testing.T) {
	state := st.NewState(st.NewCode([]byte{}))
	state.Pc = 42
	state.HasSelfDestructed[NewAddress(NewU256(42))] = struct{}{}

	hasSelfDestructed, err := HasSelfDestructed(Pc()).Check(state)
	if err != nil {
		t.Fatal(err)
	}
	if !hasSelfDestructed {
		t.Fatal("account not set as selfdestructed, when it should be")
	}

	delete(state.HasSelfDestructed, NewAddress(NewU256(42)))

	hasNotSelfDestructed, err := HasNotSelfDestructed(Pc()).Check(state)
	if err != nil {
		t.Fatal(err)
	}
	if !hasNotSelfDestructed {
		t.Fatal("account set as selfdestructed, when it should not be")
	}

}
