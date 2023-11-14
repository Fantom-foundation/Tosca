package rlz

import (
	"fmt"
	"math"
	"strings"

	. "github.com/Fantom-foundation/Tosca/go/ct/common"
	"github.com/Fantom-foundation/Tosca/go/ct/gen"
	"github.com/Fantom-foundation/Tosca/go/ct/st"
)

// Condition represents a state property.
type Condition interface {
	// Check evaluates the Condition.
	Check(*st.State) (bool, error)

	// Restrict sets constraints on the given StateGenerator such that this
	// Condition holds.
	Restrict(*gen.StateGenerator)

	// EnumerateTestCases sets constraints on a copy of the given generator and
	// invokes the given consumer function with it.
	EnumerateTestCases(generator *gen.StateGenerator, consumer func(*gen.StateGenerator))

	fmt.Stringer
}

////////////////////////////////////////////////////////////
// Conjunction

type conjunction struct {
	conditions []Condition
}

func And(conditions ...Condition) Condition {
	if len(conditions) == 1 {
		return conditions[0]
	}
	// Merge nested conjunctions into a single conjunction.
	res := []Condition{}
	for _, cur := range conditions {
		if c, ok := cur.(*conjunction); ok {
			res = append(res, c.conditions...)
		} else {
			res = append(res, cur)
		}
	}
	return &conjunction{conditions: res}
}

func (c *conjunction) Check(s *st.State) (bool, error) {
	for _, cur := range c.conditions {
		r, err := cur.Check(s)
		if !r || err != nil {
			return false, err
		}
	}
	return true, nil
}

func (c *conjunction) Restrict(generator *gen.StateGenerator) {
	for _, cur := range c.conditions {
		cur.Restrict(generator)
	}
}

func (c *conjunction) EnumerateTestCases(generator *gen.StateGenerator, consumer func(*gen.StateGenerator)) {
	if len(c.conditions) == 0 {
		consumer(generator)
		return
	}
	rest := And(c.conditions[1:]...)
	c.conditions[0].EnumerateTestCases(generator, func(generator *gen.StateGenerator) {
		rest.EnumerateTestCases(generator, consumer)
	})
}

func (c *conjunction) String() string {
	if len(c.conditions) == 0 {
		return "true"
	}
	first := true
	var builder strings.Builder
	for _, cur := range c.conditions {
		if !first {
			builder.WriteString(" ∧ ")
		} else {
			first = false
		}
		builder.WriteString(cur.String())
	}
	return builder.String()
}

////////////////////////////////////////////////////////////
// Equal

type eq[T any] struct {
	lhs Expression[T]
	rhs T
}

func Eq[T any](lhs Expression[T], rhs T) Condition {
	return &eq[T]{lhs, rhs}
}

func (e *eq[T]) Check(s *st.State) (bool, error) {
	domain := e.lhs.Domain()
	lhs, err := e.lhs.Eval(s)
	if err != nil {
		return false, err
	}
	return domain.Equal(lhs, e.rhs), nil
}

func (e *eq[T]) Restrict(generator *gen.StateGenerator) {
	e.lhs.Restrict(e.rhs, generator)
}

func (e *eq[T]) EnumerateTestCases(generator *gen.StateGenerator, consumer func(*gen.StateGenerator)) {
	domain := e.lhs.Domain()
	for _, value := range domain.Samples(e.rhs) {
		clone := generator.Clone()
		e.lhs.Restrict(value, clone)
		consumer(clone)
	}
}

func (e *eq[T]) String() string {
	return fmt.Sprintf("%s = %v", e.lhs, e.rhs)
}

////////////////////////////////////////////////////////////
// Not Equal

type ne[T any] struct {
	lhs Expression[T]
	rhs T
}

func Ne[T any](lhs Expression[T], rhs T) Condition {
	return &ne[T]{lhs, rhs}
}

func (e *ne[T]) Check(s *st.State) (bool, error) {
	domain := e.lhs.Domain()
	lhs, err := e.lhs.Eval(s)
	if err != nil {
		return false, err
	}
	return !domain.Equal(lhs, e.rhs), nil
}

func (e *ne[T]) Restrict(generator *gen.StateGenerator) {
	domain := e.lhs.Domain()
	e.lhs.Restrict(domain.SomethingNotEqual(e.rhs), generator)
}

func (e *ne[T]) EnumerateTestCases(generator *gen.StateGenerator, consumer func(*gen.StateGenerator)) {
	domain := e.lhs.Domain()
	for _, value := range domain.Samples(e.rhs) {
		clone := generator.Clone()
		e.lhs.Restrict(value, clone)
		consumer(clone)
	}
}

func (e *ne[T]) String() string {
	return fmt.Sprintf("%s ≠ %v", e.lhs, e.rhs)
}

////////////////////////////////////////////////////////////
// Less Than

type lt[T any] struct {
	lhs Expression[T]
	rhs T
}

func Lt[T any](lhs Expression[T], rhs T) Condition {
	return &lt[T]{lhs, rhs}
}

func (c *lt[T]) Check(s *st.State) (bool, error) {
	domain := c.lhs.Domain()
	lhs, err := c.lhs.Eval(s)
	if err != nil {
		return false, err
	}
	return domain.Less(lhs, c.rhs), nil
}

func (c *lt[T]) Restrict(generator *gen.StateGenerator) {
	domain := c.lhs.Domain()
	c.lhs.Restrict(domain.Predecessor(c.rhs), generator)
}

func (c *lt[T]) EnumerateTestCases(generator *gen.StateGenerator, consumer func(*gen.StateGenerator)) {
	domain := c.lhs.Domain()
	for _, value := range domain.Samples(c.rhs) {
		clone := generator.Clone()
		c.lhs.Restrict(value, clone)
		consumer(clone)
	}
}

func (c *lt[T]) String() string {
	return fmt.Sprintf("%s < %v", c.lhs, c.rhs)
}

////////////////////////////////////////////////////////////
// Less Equal

type le[T any] struct {
	lhs Expression[T]
	rhs T
}

func Le[T any](lhs Expression[T], rhs T) Condition {
	return &le[T]{lhs, rhs}
}

func (c *le[T]) Check(s *st.State) (bool, error) {
	domain := c.lhs.Domain()
	lhs, err := c.lhs.Eval(s)
	if err != nil {
		return false, err
	}
	return domain.Less(lhs, c.rhs) || domain.Equal(lhs, c.rhs), nil
}

func (c *le[T]) Restrict(generator *gen.StateGenerator) {
	c.lhs.Restrict(c.rhs, generator)
}

func (c *le[T]) EnumerateTestCases(generator *gen.StateGenerator, consumer func(*gen.StateGenerator)) {
	domain := c.lhs.Domain()
	for _, value := range domain.Samples(c.rhs) {
		clone := generator.Clone()
		c.lhs.Restrict(value, clone)
		consumer(clone)
	}
}

func (c *le[T]) String() string {
	return fmt.Sprintf("%s ≤ %v", c.lhs, c.rhs)
}

////////////////////////////////////////////////////////////
// Greater Than

type gt[T any] struct {
	lhs Expression[T]
	rhs T
}

func Gt[T any](lhs Expression[T], rhs T) Condition {
	return &gt[T]{lhs, rhs}
}

func (c *gt[T]) Check(s *st.State) (bool, error) {
	domain := c.lhs.Domain()
	lhs, err := c.lhs.Eval(s)
	if err != nil {
		return false, err
	}
	return !(domain.Less(lhs, c.rhs) || domain.Equal(lhs, c.rhs)), nil
}

func (c *gt[T]) Restrict(generator *gen.StateGenerator) {
	domain := c.lhs.Domain()
	c.lhs.Restrict(domain.Successor(c.rhs), generator)
}

func (c *gt[T]) EnumerateTestCases(generator *gen.StateGenerator, consumer func(*gen.StateGenerator)) {
	domain := c.lhs.Domain()
	for _, value := range domain.Samples(c.rhs) {
		clone := generator.Clone()
		c.lhs.Restrict(value, clone)
		consumer(clone)
	}
}

func (c *gt[T]) String() string {
	return fmt.Sprintf("%s > %v", c.lhs, c.rhs)
}

////////////////////////////////////////////////////////////
// Greater Equal

type ge[T any] struct {
	lhs Expression[T]
	rhs T
}

func Ge[T any](lhs Expression[T], rhs T) Condition {
	return &ge[T]{lhs, rhs}
}

func (c *ge[T]) Check(s *st.State) (bool, error) {
	domain := c.lhs.Domain()
	lhs, err := c.lhs.Eval(s)
	if err != nil {
		return false, err
	}
	return !domain.Less(lhs, c.rhs), nil
}

func (c *ge[T]) Restrict(generator *gen.StateGenerator) {
	c.lhs.Restrict(c.rhs, generator)
}

func (c *ge[T]) EnumerateTestCases(generator *gen.StateGenerator, consumer func(*gen.StateGenerator)) {
	domain := c.lhs.Domain()
	for _, value := range domain.Samples(c.rhs) {
		clone := generator.Clone()
		c.lhs.Restrict(value, clone)
		consumer(clone)
	}
}

func (c *ge[T]) String() string {
	return fmt.Sprintf("%s ≥ %v", c.lhs, c.rhs)
}

////////////////////////////////////////////////////////////
// Is Code

type isCode struct {
	position BindableExpression[U256]
}

func IsCode(position BindableExpression[U256]) Condition {
	return &isCode{position}
}

func (c *isCode) Check(s *st.State) (bool, error) {
	pos, err := c.position.Eval(s)
	if err != nil {
		return false, err
	}
	if pos.Gt(NewU256(math.MaxInt)) {
		return false, nil
	}
	return s.Code.IsCode(int(pos.Uint64())), nil
}

func (c *isCode) Restrict(generator *gen.StateGenerator) {
	variable := c.position.GetVariable()
	c.position.BindTo(generator)
	generator.AddIsCode(variable)
}

func (c *isCode) EnumerateTestCases(generator *gen.StateGenerator, consume func(*gen.StateGenerator)) {
	positive := generator.Clone()
	c.Restrict(positive)
	consume(positive)

	negative := generator.Clone()
	IsData(c.position).Restrict(negative)
	consume(negative)
}

func (c *isCode) String() string {
	return fmt.Sprintf("isCode[%s]", c.position)
}

////////////////////////////////////////////////////////////
// Is Data

type isData struct {
	position BindableExpression[U256]
}

func IsData(position BindableExpression[U256]) Condition {
	return &isData{position}
}

func (c *isData) Check(s *st.State) (bool, error) {
	pos, err := c.position.Eval(s)
	if err != nil {
		return false, err
	}
	if pos.Gt(NewU256(math.MaxInt)) {
		return false, nil
	}
	return s.Code.IsData(int(pos.Uint64())), nil
}

func (c *isData) Restrict(generator *gen.StateGenerator) {
	variable := c.position.GetVariable()
	c.position.BindTo(generator)
	generator.AddIsData(variable)
}

func (c *isData) EnumerateTestCases(generator *gen.StateGenerator, consume func(*gen.StateGenerator)) {
	positive := generator.Clone()
	c.Restrict(positive)
	consume(positive)

	negative := generator.Clone()
	IsCode(c.position).Restrict(negative)
	consume(negative)
}

func (c *isData) String() string {
	return fmt.Sprintf("isData[%s]", c.position)
}
