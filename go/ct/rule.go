package ct

import (
	"fmt"
	"strings"
)

// ----------------------------------------------------------------------------
//                                  Interfaces
// ----------------------------------------------------------------------------

type Condition interface {
	Check(State) bool
	restrict(*StateBuilder)
	enumerateTestCases(*StateBuilder, func(*StateBuilder))
	fmt.Stringer
}

type Effect interface {
	Apply(State) State
	fmt.Stringer
}

type Rule struct {
	Name      string
	Condition Condition
	Effect    Effect
}

type Domain[T any] interface {
	Equal(T, T) bool
	Less(T, T) bool
	Predecessor(T) T
	Successor(T) T
	Samples(T) []T
}

type Expression[T any] interface {
	Domain() Domain[T]
	Eval(State) T
	eval(*StateBuilder) T
	set(T, *StateBuilder)
	//	enumerateTestCases(*StateBuilder, func(State))
	fmt.Stringer
}

// GetSatisfyingState produces a state satisfying the given condition.
func GetSatisfyingState(condition Condition) State {
	builder := NewStateBuilder()
	condition.restrict(builder)
	return builder.Build()
}

// GetTestSamples produces a list of states representing relevant test
// cases for the given condition. At least one of those cases is satisfying
// the condition.
func GetTestSamples(condition Condition) []State {
	res := []State{}
	builder := NewStateBuilder()
	condition.enumerateTestCases(builder, func(s *StateBuilder) {
		res = append(res, s.Build())
	})
	return res
}

// ----------------------------------------------------------------------------
//                                 Conditions
// ----------------------------------------------------------------------------

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

func (c *conjunction) Check(s State) bool {
	for _, cur := range c.conditions {
		if !cur.Check(s) {
			return false
		}
	}
	return true
}

func (c *conjunction) restrict(builder *StateBuilder) {
	for _, cur := range c.conditions {
		cur.restrict(builder)
	}
}

func (c *conjunction) enumerateTestCases(builder *StateBuilder, consumer func(*StateBuilder)) {
	if len(c.conditions) == 0 {
		consumer(builder)
		return
	}
	rest := And(c.conditions[1:]...)
	c.conditions[0].enumerateTestCases(builder, func(builder *StateBuilder) {
		rest.enumerateTestCases(builder, consumer)
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

type eq[T comparable] struct {
	lhs Expression[T]
	rhs T
}

func Eq[T comparable](lhs Expression[T], rhs T) Condition {
	return &eq[T]{lhs, rhs}
}

func (e *eq[T]) Check(s State) bool {
	domain := e.lhs.Domain()
	return domain.Equal(e.lhs.Eval(s), e.rhs)
}

func (e *eq[T]) restrict(builder *StateBuilder) {
	e.lhs.set(e.rhs, builder)
}

func (e *eq[T]) enumerateTestCases(builder *StateBuilder, consumer func(*StateBuilder)) {
	domain := e.lhs.Domain()
	for _, value := range domain.Samples(e.rhs) {
		clone := builder.Clone()
		e.lhs.set(value, clone)
		consumer(clone)
	}
}

func (e *eq[T]) String() string {
	return fmt.Sprintf("%s = %v", e.lhs, e.rhs)
}

type lt[T any] struct {
	lhs Expression[T]
	rhs T
}

func Lt[T any](lhs Expression[T], rhs T) Condition {
	return &lt[T]{lhs, rhs}
}

func (c *lt[T]) Check(s State) bool {
	domain := c.lhs.Domain()
	return domain.Less(c.lhs.Eval(s), c.rhs)
}

func (c *lt[T]) restrict(builder *StateBuilder) {
	domain := c.lhs.Domain()
	c.lhs.set(domain.Predecessor(c.rhs), builder)
}

func (c *lt[T]) enumerateTestCases(builder *StateBuilder, consumer func(*StateBuilder)) {
	domain := c.lhs.Domain()
	for _, value := range domain.Samples(c.rhs) {
		clone := builder.Clone()
		c.lhs.set(value, clone)
		consumer(clone)
	}
}

func (e *lt[T]) String() string {
	return fmt.Sprintf("%s < %v", e.lhs, e.rhs)
}

// ----------------------------------------------------------------------------
//                                   Domains
// ----------------------------------------------------------------------------

type uint16Domain struct{}

func (uint16Domain) Equal(a uint16, b uint16) bool { return a == b }
func (uint16Domain) Less(a uint16, b uint16) bool  { return a < b }
func (uint16Domain) Predecessor(a uint16) uint16   { return a - 1 }
func (uint16Domain) Successor(a uint16) uint16     { return a + 1 }
func (uint16Domain) Samples(a uint16) []uint16 {
	res := []uint16{0, a - 1, a, a + 1, ^uint16(0)}
	for i := 0; i < 16; i++ {
		res = append(res, uint16(1<<i))
	}
	return res
}

type uint64Domain struct{}

func (uint64Domain) Equal(a uint64, b uint64) bool { return a == b }
func (uint64Domain) Less(a uint64, b uint64) bool  { return a < b }
func (uint64Domain) Predecessor(a uint64) uint64   { return a - 1 }
func (uint64Domain) Successor(a uint64) uint64     { return a + 1 }
func (uint64Domain) Samples(a uint64) []uint64 {
	res := []uint64{0, a - 1, a, a + 1, ^uint64(0)}
	for i := 0; i < 64; i++ {
		res = append(res, uint64(1<<i))
	}
	return res
}

type codeDomain struct{}

func (codeDomain) Equal(a string, b string) bool { return a == b }
func (codeDomain) Less(a string, b string) bool  { panic("not useful") }
func (codeDomain) Predecessor(a string) string   { panic("not useful") }
func (codeDomain) Successor(a string) string     { panic("not useful") }
func (codeDomain) Samples(a string) []string     { return []string{a} }

type opCodeDomain struct{}

func (opCodeDomain) Equal(a OpCode, b OpCode) bool { return a == b }
func (opCodeDomain) Less(a OpCode, b OpCode) bool  { panic("not useful") }
func (opCodeDomain) Predecessor(a OpCode) OpCode   { panic("not useful") }
func (opCodeDomain) Successor(a OpCode) OpCode     { panic("not useful") }
func (opCodeDomain) Samples(a OpCode) []OpCode     { return []OpCode{a} }

// ----------------------------------------------------------------------------
//                                 Expressions
// ----------------------------------------------------------------------------

// --- code ---

type code struct{}

func Code() Expression[string] {
	return code{}
}

func (code) Domain() Domain[string] { return codeDomain{} }

func (code) Eval(s State) string {
	return s.Code
}

func (code) eval(s *StateBuilder) string {
	return s.GetCode()
}

func (code) set(code string, builder *StateBuilder) {
	builder.SetCode(code)
}

func (code) String() string {
	return "code"
}

// --- pc ---

type pc struct{}

func Pc() Expression[uint16] {
	return pc{}
}

func (pc) Domain() Domain[uint16] { return uint16Domain{} }

func (pc) Eval(s State) uint16 {
	return s.Pc
}

func (pc) eval(s *StateBuilder) uint16 {
	return s.GetPc()
}

func (pc) set(pc uint16, builder *StateBuilder) {
	builder.SetPc(pc)
}

func (pc) String() string {
	return "PC"
}

// --- gas ---

type gas struct{}

func Gas() Expression[uint64] {
	return gas{}
}

func (gas) Domain() Domain[uint64] { return uint64Domain{} }

func (gas) Eval(s State) uint64 {
	return s.Gas
}

func (gas) eval(s *StateBuilder) uint64 {
	return s.GetGas()
}

func (gas) set(gas uint64, builder *StateBuilder) {
	builder.SetGas(gas)
}

func (gas) String() string {
	return "gas"
}

// --- Operations ---

type op struct {
	position Expression[uint16]
}

func Op(position Expression[uint16]) Expression[OpCode] {
	return op{position}
}

func (op) Domain() Domain[OpCode] { return opCodeDomain{} }

func (e op) Eval(s State) OpCode {
	pos := e.position.Eval(s)
	code := []byte(s.Code)
	if pos < uint16(len(code)) {
		return OpCode(code[pos])
	}
	return STOP
}

func (o op) eval(builder *StateBuilder) OpCode {
	pos := o.position.eval(builder)
	code := []byte(builder.GetCode())
	if pos < uint16(len(code)) {
		return OpCode(code[pos])
	}
	return STOP
}

func (o op) set(op OpCode, builder *StateBuilder) {
	pos := o.position.eval(builder)
	builder.SetOpCode(pos, op)
}

func (o op) String() string {
	return fmt.Sprintf("code[%v]", o.position)
}
