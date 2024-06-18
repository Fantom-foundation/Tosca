// Copyright (c) 2024 Fantom Foundation
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at fantom.foundation/bsl11.
//
// Change Date: 2028-4-16
//
// On the date above, in accordance with the Business Source License, use of
// this software will be governed by the GNU Lesser General Public License v3.

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
	enumerateTestCases(rule.Condition, gen.NewStateGenerator(), func(generator *gen.StateGenerator) ConsumerResult {
		state, err := generator.Generate(rnd)
		if errors.Is(err, gen.ErrUnsatisfiable) {
			return ConsumeContinue // ignored
		}
		if err != nil {
			enumError = err
			return ConsumeAbort
		}
		res := enumerateParameters(0, rule.Parameter, state, consume)
		state.Release()
		return res
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
		res.conditions = append(res.conditions, condition.String())
	}
	res.propertyDomains = make(map[Property][]string)
	for property, domain := range getPropertyTestValues(r.Condition) {
		list := make([]string, 0, len(domain))
		for _, cur := range domain {
			list = append(list, cur.String())
		}
		res.propertyDomains[property] = list
	}
	res.parameterDomainSizes = make([]int, 0, len(r.Parameter))
	for _, parameter := range r.Parameter {
		res.parameterDomainSizes = append(res.parameterDomainSizes, len(parameter.Samples())+1)
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

type TestCaseEnumerationInfo struct {
	conditions           []string
	propertyDomains      map[Property][]string
	parameterDomainSizes []int
}

func (i *TestCaseEnumerationInfo) TotalNumberOfCases() int {
	res := 1
	for _, domain := range i.propertyDomains {
		res *= len(domain)
	}
	for _, size := range i.parameterDomainSizes {
		res *= size
	}
	return res
}

func (i *TestCaseEnumerationInfo) String() string {
	builder := strings.Builder{}
	builder.WriteString("Conditions:\n")
	slices.Sort(i.conditions)
	for _, condition := range i.conditions {
		builder.WriteString(fmt.Sprintf("\t%s\n", condition))
	}
	if len(i.conditions) == 0 {
		builder.WriteString("\t-none-\n")
	}
	builder.WriteString("Domains:\n")
	keys := maps.Keys(i.propertyDomains)
	slices.Sort(keys)
	for _, key := range keys {
		builder.WriteString(fmt.Sprintf(
			"\t%s: N=%d, {%s}\n",
			key,
			len(i.propertyDomains[key]),
			strings.Join(i.propertyDomains[key], ", "),
		))
	}
	if len(i.propertyDomains) == 0 {
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
