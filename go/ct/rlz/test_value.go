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
	"fmt"
	"slices"
	"sort"

	"github.com/Fantom-foundation/Tosca/go/ct/gen"
	"golang.org/x/exp/maps"
)

// TestValue is a single point in the parameter space of a state
// property that should be tested. An example might be a gas value
// of 67 units.
type TestValue interface {
	fmt.Stringer
	Property() Property           // < the property the value is to assigned to (e.g. "gas")
	Compare(TestValue) int        // < a total order on test values
	Restrict(*gen.StateGenerator) // < restricts the given generator to produce states with the value to be tested
	private()                     // < to make sure nobody outside this package can implement this interface
}

// getPropertyTestValues computes a list of test values to test the given
// condition, grouped by the targeted property.
func getPropertyTestValues(condition Condition) map[Property][]TestValue {
	// Collect and index test cases derived from the condition.
	dimensions := map[Property][]TestValue{}
	for _, cur := range condition.GetTestValues() {
		tests := dimensions[cur.Property()]
		tests = append(tests, cur)
		dimensions[cur.Property()] = tests
	}

	// Remove duplicates in individual dimensions.
	for property, tests := range dimensions {
		dimensions[property] = removeDuplicates(tests)
	}
	return dimensions
}

// enumerateTestCases sets constraints on a copy of the given generator and
// invokes the given consumer function with it.
func enumerateTestCases(
	condition Condition,
	generator *gen.StateGenerator,
	consumer func(*gen.StateGenerator) ConsumerResult,
) ConsumerResult {
	dimensions := getPropertyTestValues(condition)

	// Sort dimensions to have a deterministic execution order.
	properties := maps.Keys(dimensions)
	slices.Sort(properties)
	cases := [][]TestValue{}
	for _, property := range properties {
		cases = append(cases, dimensions[property])
	}

	// Run the actual test-case generation.
	return enumerateTestStates(cases, generator, consumer)
}

func enumerateTestStates(
	values [][]TestValue,
	generator *gen.StateGenerator,
	consumer func(*gen.StateGenerator) ConsumerResult,
) ConsumerResult {
	if len(values) == 0 {
		return consumer(generator)
	}
	head := values[0]
	rest := values[1:]
	for _, cur := range head {
		copy := generator.Clone()
		cur.Restrict(copy)
		if res := enumerateTestStates(rest, copy, consumer); res == ConsumeAbort {
			return res
		}
	}
	return ConsumeContinue
}

func removeDuplicates(values []TestValue) []TestValue {
	if len(values) < 2 {
		return values
	}
	sort.Slice(values, func(i, j int) bool {
		return values[i].Compare(values[j]) < 0
	})
	o := 1
	for i := 1; i < len(values); i++ {
		if values[i].Compare(values[i-1]) != 0 {
			values[o] = values[i]
			o++
		}
	}
	return values[:o]
}

type testValue[T any] struct {
	property Property
	domain   Domain[T]
	restrict func(*gen.StateGenerator, T)
	value    T
}

func NewTestValue[T any](
	property Property,
	domain Domain[T],
	value T,
	restrict func(*gen.StateGenerator, T),
) TestValue {
	return &testValue[T]{property, domain, restrict, value}
}

func (c *testValue[T]) Property() Property {
	return c.property
}

func (c *testValue[T]) Compare(other TestValue) int {
	thisProperty := c.property
	otherProperty := other.Property()
	if thisProperty != otherProperty {
		if thisProperty < otherProperty {
			return -1
		}
		return 1
	}
	otherValue, ok := other.(interface{ Value() T })
	if !ok {
		panic("other test value with same property is not of expected type")
	}
	domain := c.domain
	if domain.Equal(c.value, otherValue.Value()) {
		return 0
	}
	if domain.Less(c.value, otherValue.Value()) {
		return -1
	}
	return 1
}

func (c *testValue[T]) Value() T {
	return c.value
}

func (c *testValue[T]) Restrict(gen *gen.StateGenerator) {
	c.restrict(gen, c.value)
}

func (c *testValue[T]) String() string {
	return fmt.Sprintf("%v", c.value)
}

func (c *testValue[T]) private() {}
