package rlz

import (
	"testing"

	"pgregory.net/rand"

	"github.com/Fantom-foundation/Tosca/go/ct"
	"github.com/Fantom-foundation/Tosca/go/ct/st"
)

func TestRule_GenerateSatisfyingState(t *testing.T) {
	tests := []Condition{
		And(), // = anything
		Eq(Status(), st.Failed),
		Eq(Pc(), ct.NewU256(42)),
		And(Eq(Status(), st.Failed), Eq(Pc(), ct.NewU256(42))),
		And(Eq(Op(Pc()), st.ADD)),
		And(Eq(Op(Pc()), st.JUMP), Eq(Op(Param(0)), st.JUMPDEST)),
	}

	rnd := rand.New(0)

	for _, test := range tests {
		rule := Rule{Condition: test}
		state, err := rule.GenerateSatisfyingState(rnd)
		if err != nil {
			t.Fatalf("Failed to generate state: %v", err)
		}
		if !test.Check(state) {
			t.Errorf("Generated state does not satisfy condition %v: %v", test, &state)
		}
	}
}

func TestRule_EnumerateTestCases(t *testing.T) {
	tests := []Condition{
		And(), // = anything
		Eq(Status(), st.Failed),
		Eq(Pc(), ct.NewU256(42)),
		And(Eq(Status(), st.Failed), Eq(Pc(), ct.NewU256(42))),
		And(Eq(Op(Pc()), st.ADD)),
		And(Eq(Op(Pc()), st.JUMP), Eq(Op(Param(0)), st.JUMPDEST)),
	}

	rnd := rand.New(0)

	for _, test := range tests {
		matches := 0
		misses := 0

		rule := Rule{Condition: test}
		err := rule.EnumerateTestCases(rnd, func(sample *st.State) {
			if test.Check(sample) {
				matches++
			} else {
				misses++
			}
		})
		if err != nil {
			t.Errorf("EnumerateTestCases failed %v", err)
		}
		if matches == 0 {
			t.Errorf("none of the %d generated samples for %v is a match", matches+misses, test)
		}
		if matches+misses > 1 && misses == 0 {
			t.Errorf("none of the %d generated samples for %v is a miss", matches+misses, test)
		}
	}
}
