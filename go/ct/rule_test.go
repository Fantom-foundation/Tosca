package ct

import (
	"testing"
)

func TestCondition_Printing(t *testing.T) {
	tests := []struct {
		condition Condition
		result    string
	}{
		{And(), "true"},
		{And(And(), And()), "true"},
		{And(Eq(Gas(), uint64(12)), Eq(Pc(), uint16(14))), "gas = 12 ∧ PC = 14"},
		{And(Eq(Code(), []byte("abc"))), "code = [97 98 99]"},
		{And(Lt(Gas(), uint64(4))), "gas < 4"},
		{Eq(Op(Pc()), POP), "code[PC] = POP"},
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
		Eq(Pc(), 2),
		Lt(Gas(), 10),
		And(Eq(Gas(), 1), Eq(Pc(), 2)),
		And(Eq(Pc(), 2), Eq(Gas(), 1)),
		Eq(Code(), []byte("abc")),
		Eq(Op(Pc()), POP),
		And(Eq(Pc(), 4), Eq(Op(Pc()), POP)),
	}

	for _, test := range tests {
		res := GetSatisfyingState(Rule{Condition: test})
		if !test.Check(res) {
			t.Errorf("Generated state does not satisfy condition %v: %v", test, res)
		}
	}
}

func TestCondition_CanGenerateTestSamples(t *testing.T) {
	tests := []Condition{
		And(), // = anything
		Eq(Gas(), 1),
		Eq(Pc(), 2),
		Lt(Gas(), 10),
		And(Eq(Gas(), 1), Eq(Pc(), 2)),
		And(Eq(Pc(), 2), Eq(Gas(), 1)),
		Eq(Code(), []byte("abc")),
		Eq(Op(Pc()), POP),
		And(Eq(Pc(), 4), Eq(Op(Pc()), POP)),
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
			t.Errorf("none of the %d generated samples is a match", len(samples))
		}
		if len(samples) > 1 && misses == 0 {
			t.Errorf("none of the %d generated samples is a miss", len(samples))
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
