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
}

// GenerateSatisfyingState produces an st.State satisfying this Rule.
func (rule *Rule) GenerateSatisfyingState(rnd *rand.Rand) (*st.State, error) {
	generator := gen.NewStateGenerator()
	rule.Condition.Restrict(generator)
	return generator.Generate(rnd)
}

// EnumerateTestCases generates interesting st.States according to this Rule.
// Each valid st.State is passed to the given consume function.
func (rule *Rule) EnumerateTestCases(rnd *rand.Rand, consume func(s *st.State)) error {
	var generatorErrors []error

	generator := gen.NewStateGenerator()
	rule.Condition.EnumerateTestCases(generator, func(g *gen.StateGenerator) {
		state, err := g.Generate(rnd)
		if errors.Is(err, gen.ErrUnsatisfiable) {
			return // ignored
		}
		if err != nil {
			generatorErrors = append(generatorErrors, err)
			return
		}
		consume(state)
	})

	return errors.Join(generatorErrors...)
}
