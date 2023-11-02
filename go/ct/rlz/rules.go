package rlz

import (
	"errors"

	"github.com/Fantom-foundation/Tosca/go/ct/gen"
	"github.com/Fantom-foundation/Tosca/go/ct/st"
)

type Rule struct {
	Name      string
	Condition Condition
	Effect    Effect
}

// GenerateSatisfyingState produces an st.State satisfying this Rule.
func (rule *Rule) GenerateSatisfyingState(seed uint64) (*st.State, error) {
	generator := gen.NewStateGeneratorWithSeed(seed)
	rule.Condition.Restrict(generator)
	return generator.Generate()
}

// EnumerateTestCases generates interesting st.States according to this Rule.
// Each valid st.State is passed to the given consume function.
func (rule *Rule) EnumerateTestCases(seed uint64, consume func(s *st.State)) error {
	var generatorErrors []error

	generator := gen.NewStateGeneratorWithSeed(seed)
	rule.Condition.EnumerateTestCases(generator, func(g *gen.StateGenerator) {
		state, err := g.Generate()
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
