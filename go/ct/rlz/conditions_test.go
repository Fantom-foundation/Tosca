package rlz

import (
	"testing"

	. "github.com/Fantom-foundation/Tosca/go/ct/common"
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
		{And(Eq(Status(), st.Reverted), Eq(Pc(), NewU256(42))), newStateWithStatusAndPc(st.Reverted, 42), newStateWithStatusAndPc(st.Returned, 42)},
		{And(Eq(Status(), st.Reverted), Eq(Pc(), NewU256(42))), newStateWithStatusAndPc(st.Reverted, 42), newStateWithStatusAndPc(st.Reverted, 41)},
		{IsCode(Pc()), newStateWithPcAndCode(1, byte(ADD), byte(ADD)), newStateWithPcAndCode(1, byte(PUSH1), byte(0))},
		{IsData(Pc()), newStateWithPcAndCode(1, byte(PUSH1), byte(0)), newStateWithPcAndCode(1, byte(ADD), byte(ADD))},
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
