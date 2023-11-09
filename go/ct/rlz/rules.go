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
// *not* modify the provided state.
func (rule *Rule) EnumerateTestCases(rnd *rand.Rand, consume func(*st.State)) error {
	var generatorErrors []error

	rule.Condition.EnumerateTestCases(gen.NewStateGenerator(), func(generator *gen.StateGenerator) {
		state, err := generator.Generate(rnd)
		if errors.Is(err, gen.ErrUnsatisfiable) {
			return // ignored
		}
		if err != nil {
			generatorErrors = append(generatorErrors, err)
			return
		}

		enumerateParameters(0, rule.Parameter, state, consume)
	})

	return errors.Join(generatorErrors...)
}

func enumerateParameters(pos int, params []Parameter, state *st.State, consume func(*st.State)) {
	if len(params) == 0 || pos >= state.Stack.Size() {
		consume(state)
		return
	}

	current := state.Stack.Get(pos)

	// Cross product with current parameter value, as set by generator.
	enumerateParameters(pos+1, params[1:], state, consume)

	// Cross product with different samples for parameter.
	for _, value := range params[0].Samples() {
		state.Stack.Set(pos, value)
		enumerateParameters(pos+1, params[1:], state, consume)
	}

	state.Stack.Set(pos, current)
}
