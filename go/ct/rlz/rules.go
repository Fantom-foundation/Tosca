//
// Copyright (c) 2024 Fantom Foundation
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at fantom.foundation/bsl11.
//
// Change Date: 2028-4-16
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by the GNU Lesser General Public Licence v3
//

package rlz

import (
	"errors"
	"fmt"
	"slices"
	"strings"

	"golang.org/x/exp/maps"
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

func (r *Rule) GetTestCaseEnumerationInfo() TestCaseEnumerationInfo {
	res := TestCaseEnumerationInfo{}
	conditions := getConditions(r.Condition)
	for _, condition := range conditions {
		if res.conditionDomainSizes == nil {
			res.conditionDomainSizes = make(map[string]int)
		}
		res.conditionDomainSizes[condition.String()] = estimateTestDomainSize(condition)
	}
	res.parameterDomainSizes = make([]int, 0, len(r.Parameter))
	for _, parameter := range r.Parameter {
		res.parameterDomainSizes = append(res.parameterDomainSizes, len(parameter.Samples()))
	}
	return res
}

func getConditions(condition Condition) []Condition {
	if condition == nil {
		return nil
	}
	conjunction, ok := condition.(*conjunction)
	if !ok {
		return []Condition{condition}
	}
	res := []Condition{}
	for _, cur := range conjunction.conditions {
		res = append(res, getConditions(cur)...)
	}
	return res
}

func estimateTestDomainSize(condition Condition) int {
	counter := 0
	generator := gen.NewStateGenerator()
	condition.EnumerateTestCases(generator, func(*gen.StateGenerator) ConsumerResult {
		counter++
		return ConsumeContinue
	})
	return counter
}

type TestCaseEnumerationInfo struct {
	conditionDomainSizes map[string]int
	parameterDomainSizes []int
}

func (i *TestCaseEnumerationInfo) TotalNumberOfCases() int {
	res := 1
	for _, size := range i.conditionDomainSizes {
		res *= size
	}
	for _, size := range i.parameterDomainSizes {
		res *= size
	}
	return res
}

func (i *TestCaseEnumerationInfo) String() string {
	builder := strings.Builder{}
	builder.WriteString("Conditions:\n")
	keys := maps.Keys(i.conditionDomainSizes)
	slices.Sort(keys)
	for _, key := range keys {
		builder.WriteString(fmt.Sprintf("\t%s: %d\n", key, i.conditionDomainSizes[key]))
	}
	if len(i.conditionDomainSizes) == 0 {
		builder.WriteString("\t-none-\n")
	}
	builder.WriteString("Parameters:\n")
	for i, size := range i.parameterDomainSizes {
		builder.WriteString(fmt.Sprintf("\t%d: %d\n", i, size))
	}
	if len(i.parameterDomainSizes) == 0 {
		builder.WriteString("\t-none-\n")
	}
	builder.WriteString(fmt.Sprintf("Total number of cases: %d\n", i.TotalNumberOfCases()))
	return builder.String()
}
