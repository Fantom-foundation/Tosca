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
	return generator.Generate(rnd)
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
		state, err := generator.Generate(rnd)
		if errors.Is(err, gen.ErrUnsatisfiable) {
			return // ignored
		}
		if err != nil {
			onError(err)
			return
		}

		// in case the Program Counter has no constraints we want to make sure we are testing
		// both possible cases, where it points to data and when it points to code.
		if !generator.HasPcConstraints {
			alternateState := state.Clone()
			findFirst := func(check func(int) bool) {
				for i := 0; i < state.Code.Length(); i++ {
					if check(i) {
						alternateState.Pc = uint16(i)
						return
					}
				}
			}
			if state.Code.IsCode(int(state.Pc)) {
				isNotCode := func(i int) bool { return !state.Code.IsCode(i) }
				findFirst(isNotCode)
			} else {
				findFirst(state.Code.IsCode)
			}
			enumerateParameters(0, rule.Parameter, alternateState, consume, onError, onSuccess)
		}
		enumerateParameters(0, rule.Parameter, state, consume, onError, onSuccess)
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
