package rlz

import (
	"errors"
	"slices"

	"pgregory.net/rand"

	"github.com/Fantom-foundation/Tosca/go/ct/gen"
	"github.com/Fantom-foundation/Tosca/go/ct/st"
)

type Rule struct {
	Name      string
	Condition Condition
	Parameter []Parameter
	Effect    Effect
}

// GenerateSatisfyingState produces an st.State satisfying this Rule.
func (rule *Rule) GenerateSatisfyingState(rnd *rand.Rand) (*st.State, error) {
	generator := gen.NewStateGenerator()
	rule.Condition.Restrict(generator)
	states, err := generator.Generate(rnd)
	return states[0], err
}

// EnumerateTestCases generates interesting st.States according to this Rule.
// Each valid st.State is passed to the given consume function. consume must
// *not* modify the provided state. Errors are accumulated and a list of all errors is returned.
func (rule *Rule) EnumerateTestCases(rnd *rand.Rand, consume func(*st.State) error) ([]error, int) {
	var accumulatedErrors []error

	onError := func(err error) {
		if err != ErrSkipped {
			accumulatedErrors = append(accumulatedErrors, err)
		}
	}

	executionCount := 0

	onSuccess := func() {
		executionCount++
	}

	rule.Condition.EnumerateTestCases(gen.NewStateGenerator(), func(generator *gen.StateGenerator) {
		states, err := generator.Generate(rnd)
		if errors.Is(err, gen.ErrUnsatisfiable) {
			return // ignored
		}
		if err != nil {
			onError(err)
			return
		}

		for _, state := range states {
			enumerateParameters(0, rule.Parameter, state, consume, onError, onSuccess)
			if err != nil {
				onError(err)
				return
			}
		}
	})

	return accumulatedErrors, executionCount
}

func enumerateParameters(pos int, params []Parameter, state *st.State, consume func(*st.State) error, onError func(error), onSuccess func()) {
	if len(params) == 0 || pos >= state.Stack.Size() {
		err := consume(state)
		if err == nil {
			onSuccess()
		} else if !slices.Contains(IgnoredErrors, err) {
			onError(err)
		}
		return
	}

	current := state.Stack.Get(pos)

	// Cross product with current parameter value, as set by generator.
	enumerateParameters(pos+1, params[1:], state, consume, onError, onSuccess)

	// Cross product with different samples for parameter.
	for _, value := range params[0].Samples() {
		state.Stack.Set(pos, value)
		enumerateParameters(pos+1, params[1:], state, consume, onError, onSuccess)
	}

	state.Stack.Set(pos, current)
}
