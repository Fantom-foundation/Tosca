package rlz

import (
	"errors"
	"testing"

	"pgregory.net/rand"

	. "github.com/Fantom-foundation/Tosca/go/ct/common"
	"github.com/Fantom-foundation/Tosca/go/ct/st"
)

func TestRule_GenerateSatisfyingState(t *testing.T) {
	tests := []Condition{
		And(), // = anything
		Eq(Status(), st.Failed),
		Eq(Pc(), NewU256(42)),
		And(Eq(Status(), st.Failed), Eq(Pc(), NewU256(42))),
		And(Eq(Op(Pc()), ADD)),
		And(Eq(Op(Pc()), JUMP), Eq(Op(Param(0)), JUMPDEST)),
	}

	rnd := rand.New(0)

	for _, test := range tests {
		rule := Rule{Condition: test}
		state, err := rule.GenerateSatisfyingState(rnd)
		if err != nil {
			t.Errorf("Failed to generate state: %v", err)
		}

		satisfied, err := test.Check(state)
		if err != nil {
			t.Errorf("Condition check error %v", err)
		}
		if !satisfied {
			t.Errorf("Generated state does not satisfy condition %v: %v", test, &state)
		}
	}
}

func TestRule_EnumerateTestCases(t *testing.T) {
	tests := []Condition{
		And(), // = anything
		Eq(Status(), st.Failed),
		Eq(Pc(), NewU256(42)),
		And(Eq(Status(), st.Failed), Eq(Pc(), NewU256(42))),
		And(Eq(Op(Pc()), ADD)),
		And(Eq(Op(Pc()), JUMP), Eq(Op(Param(0)), JUMPDEST)),
	}

	rnd := rand.New(0)

	for _, test := range tests {
		matches := 0
		misses := 0

		rule := Rule{Condition: test}
		errs, executedCount := rule.EnumerateTestCases(rnd, func(sample *st.State) error {
			match, err := test.Check(sample)
			if err != nil {
				t.Errorf("Condition check error %v", err)
			}
			if match {
				matches++
			} else {
				misses++
			}
			return nil
		})
		err := errors.Join(errs...)
		if err != nil {
			t.Errorf("EnumerateTestCases failed %v", err)
		}
		if executedCount == 0 {
			t.Errorf("no state executed, count: %v", executedCount)
		}
		if matches == 0 {
			t.Errorf("none of the %d generated samples for %v is a match", matches+misses, test)
		}
		if matches+misses > 1 && misses == 0 {
			t.Errorf("none of the %d generated samples for %v is a miss", matches+misses, test)
		}
	}
}

func TestRule_EnumerateTestCasesUnapplicable(t *testing.T) {
	rnd := rand.New(0)
	condition := And(Eq(Status(), st.Failed), Eq(Status(), st.Running))
	rule := Rule{Condition: condition}
	errs, executedCount := rule.EnumerateTestCases(rnd, func(sample *st.State) error {
		if applies, err := rule.Condition.Check(sample); !applies || err != nil {
			if err == nil {
				err = ErrUnapplicable
			}
			return err
		}
		return nil
	})

	if errs != nil {
		t.Errorf("unexpected amount of errors. wanted: %v, got: %v", ErrUnapplicable, errs)
	}
	if executedCount != 0 {
		t.Errorf("unexpected execution of imposible state. executed: %v", executedCount)
	}
}
