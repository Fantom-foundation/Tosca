package rlz

import (
	"errors"

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
// *not* modify the provided state.  An error is returned if the enumeration
// process fails due to an unexpected internal state generation error.
func (rule *Rule) EnumerateTestCases(rnd *rand.Rand, consume func(*st.State) ConsumerResult) error {
	var enumError error
	rule.Condition.EnumerateTestCases(gen.NewStateGenerator(), func(generator *gen.StateGenerator) ConsumerResult {
		state, err := generator.Generate(rnd)
		if errors.Is(err, gen.ErrUnsatisfiable) {
			return ConsumeContinue // ignored
		}
		if err != nil {
			enumError = err
			return ConsumeAbort
		}
		return enumerateParameters(0, rule.Parameter, state, consume)
	})

	return enumError
}

func enumerateParameters(pos int, params []Parameter, state *st.State, consume func(*st.State) ConsumerResult) ConsumerResult {
	if len(params) == 0 || pos >= state.Stack.Size() {
		return consume(state)
	}

	current := state.Stack.Get(pos)

	// Cross product with current parameter value, as set by generator.
	if enumerateParameters(pos+1, params[1:], state, consume) == ConsumeAbort {
		return ConsumeAbort
	}

	// Cross product with different samples for parameter.
	for _, value := range params[0].Samples() {
		state.Stack.Set(pos, value)
		if enumerateParameters(pos+1, params[1:], state, consume) == ConsumeAbort {
			return ConsumeAbort
		}
	}

	state.Stack.Set(pos, current)
	return ConsumeContinue
}
