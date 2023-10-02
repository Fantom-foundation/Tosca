package ct

import (
	"testing"

	"github.com/holiman/uint256"
)

func TestCondition_Printing(t *testing.T) {
	tests := []struct {
		condition Condition
		result    string
	}{
		{And(), "true"},
		{And(And(), And()), "true"},
		{And(Eq(Gas(), uint64(12)), Eq(Pc(), *uint256.NewInt(14))), "gas = 12 ∧ PC = [14 0 0 0]"},
		{And(Eq(Code(), []byte("abc"))), "code = [97 98 99]"},
		{And(Lt(Gas(), uint64(4))), "gas < 4"},
		{Eq(Op(Pc()), POP), "code[PC] = POP"},
		{Ne(Op(Pc()), POP), "code[PC] ≠ POP"},
		{IsCode(Pc()), "isCode[PC]"},
		{IsData(Pc()), "isData[PC]"},
		{IsCode(Param(1)), "isCode[param[1]]"},
		{IsData(Param(2)), "isData[param[2]]"},
		{Eq(Param(2), *uint256.NewInt(12)), "param[2] = [12 0 0 0]"},
	}

	for _, test := range tests {
		if got, want := test.condition.String(), test.result; got != want {
			t.Errorf("unexpected print, wanted %s, got %s", want, got)
		}
	}
}

func TestCondition_CreateSatisfyingState(t *testing.T) {
	tests := []Condition{
		And(), // = anything
		Eq(Gas(), 1),
		Eq(Pc(), *uint256.NewInt(2)),
		Lt(Gas(), 10),
		And(Eq(Gas(), 1), Eq(Pc(), *uint256.NewInt(2))),
		And(Eq(Pc(), *uint256.NewInt(2)), Eq(Gas(), 1)),
		Eq(Code(), []byte("abc")),
		Eq(Op(Pc()), POP),
		Ne(Op(Pc()), POP),
		And(Eq(Pc(), *uint256.NewInt(4)), Eq(Op(Pc()), POP)),
		IsCode(Pc()),
		IsData(Pc()),
		IsCode(Param(1)),
		IsData(Param(2)),
		Eq(Param(2), *uint256.NewInt(12)),
	}

	for _, test := range tests {
		res := GetSatisfyingState(Rule{Condition: test})
		if !test.Check(res) {
			t.Errorf("Generated state does not satisfy condition %v: %v", test, &res)
		}
	}
}

func TestCondition_CanGenerateTestSamples(t *testing.T) {
	tests := []Condition{
		And(), // = anything
		Eq(Gas(), 1),
		Eq(Pc(), *uint256.NewInt(2)),
		Lt(Gas(), 10),
		And(Eq(Gas(), 1), Eq(Pc(), *uint256.NewInt(2))),
		And(Eq(Pc(), *uint256.NewInt(2)), Eq(Gas(), 1)),
		Eq(Code(), []byte("abc")),
		Eq(Op(Pc()), POP),
		Ne(Op(Pc()), POP),
		And(Eq(Pc(), *uint256.NewInt(4)), Eq(Op(Pc()), POP)),
		IsCode(Pc()),
		IsData(Pc()),
		Eq(Param(2), *uint256.NewInt(12)),
	}

	for _, test := range tests {
		samples := GetTestSamples(Rule{Condition: test})
		matches := 0
		misses := 0
		for _, sample := range samples {
			if test.Check(sample) {
				matches++
			} else {
				misses++
			}
		}
		if matches == 0 {
			t.Errorf("none of the %d generated samples for %v is a match", len(samples), test)
		}
		if len(samples) > 1 && misses == 0 {
			t.Errorf("none of the %d generated samples for %v is a miss", len(samples), test)
		}
	}
}

func TestCondition_CanGenerateTestSamplesForParameters(t *testing.T) {
	tests := []Rule{
		{
			Condition: Eq(StackSize(), 4),
			Parameter: []Parameter{
				NumericParameter{},
				NumericParameter{},
			},
		},
	}

	for _, test := range tests {
		samples := GetTestSamples(test)

		matches := 0
		misses := 0
		for _, sample := range samples {
			if test.Condition.Check(sample) {
				matches++
			} else {
				misses++
			}
		}
		if matches == 0 {
			t.Errorf("none of the %d generated samples is a match", len(samples))
		}
		if len(samples) > 1 && misses == 0 {
			t.Errorf("none of the %d generated samples is a miss", len(samples))
		}
	}
}
