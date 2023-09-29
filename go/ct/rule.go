package ct

import (
	"fmt"
	"strings"

	"github.com/holiman/uint256"
	"golang.org/x/exp/slices"
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
	Parameter []Parameter
	Effect    Effect
}

type Domain[T any] interface {
	Equal(T, T) bool
	Less(T, T) bool
	Predecessor(T) T
	Successor(T) T
	Samples(T) []T
	SamplesForAll([]T) []T
}

type Parameter interface {
	Samples() []uint256.Int
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
func GetSatisfyingState(rule Rule) State {
	builder := NewStateBuilder()
	rule.Condition.restrict(builder)
	return builder.Build()
}

// GetTestSamples produces a list of states representing relevant test
// cases for the given condition. At least one of those cases is satisfying
// the condition.
func GetTestSamples(rule Rule) []State {
	res := []State{}
	builder := NewStateBuilder()
	rule.Condition.enumerateTestCases(builder, func(s *StateBuilder) {
		enumerateParameters(0, rule.Parameter, s, func(s *StateBuilder) {
			res = append(res, s.Build())
		})
	})
	return res
}

func enumerateParameters(pos int, params []Parameter, builder *StateBuilder, consume func(s *StateBuilder)) {
	if len(params) == 0 {
		consume(builder)
		return
	}
	for _, value := range params[0].Samples() {
		clone := builder.Clone()
		clone.SetStackValue(pos, value)
		enumerateParameters(pos+1, params[1:], clone, consume)
	}
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

type eq[T any] struct {
	lhs Expression[T]
	rhs T
}

func Eq[T any](lhs Expression[T], rhs T) Condition {
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

func (c *lt[T]) String() string {
	return fmt.Sprintf("%s < %v", c.lhs, c.rhs)
}

type ge[T any] struct {
	lhs Expression[T]
	rhs T
}

func Ge[T any](lhs Expression[T], rhs T) Condition {
	return &ge[T]{lhs, rhs}
}

func (c *ge[T]) Check(s State) bool {
	domain := c.lhs.Domain()
	return !domain.Less(c.lhs.Eval(s), c.rhs)
}

func (c *ge[T]) restrict(builder *StateBuilder) {
	c.lhs.set(c.rhs, builder)
}

func (c *ge[T]) enumerateTestCases(builder *StateBuilder, consumer func(*StateBuilder)) {
	domain := c.lhs.Domain()
	for _, value := range domain.Samples(c.rhs) {
		clone := builder.Clone()
		c.lhs.set(value, clone)
		consumer(clone)
	}
}

func (c *ge[T]) String() string {
	return fmt.Sprintf("%s ≥ %v", c.lhs, c.rhs)
}

type in[T comparable] struct {
	lhs Expression[T]
	rhs []T
}

func In[T comparable](lhs Expression[T], rhs []T) Condition {
	if len(rhs) == 0 {
		panic("in condition must have a non-empty list of options")
	}
	return &in[T]{lhs, rhs}
}

func (c *in[T]) Check(s State) bool {
	domain := c.lhs.Domain()
	value := c.lhs.Eval(s)
	for i := 0; i < len(c.rhs); i++ {
		if domain.Equal(value, c.rhs[i]) {
			return true
		}
	}
	return false
}

func (c *in[T]) restrict(builder *StateBuilder) {
	c.lhs.set(c.rhs[0], builder)
}

func (c *in[T]) enumerateTestCases(builder *StateBuilder, consumer func(*StateBuilder)) {
	domain := c.lhs.Domain()
	for _, value := range domain.SamplesForAll(c.rhs) {
		clone := builder.Clone()
		c.lhs.set(value, clone)
		consumer(clone)
	}
}

func (c *in[T]) String() string {
	return fmt.Sprintf("%s ∈ %v", c.lhs, c.rhs)
}

// ----------------------------------------------------------------------------
//                                   Domains
// ----------------------------------------------------------------------------

type statusCodeDomain struct{}

func (statusCodeDomain) Equal(a StatusCode, b StatusCode) bool { return a == b }
func (statusCodeDomain) Less(a StatusCode, b StatusCode) bool  { panic("not useful") }
func (statusCodeDomain) Predecessor(a StatusCode) StatusCode   { panic("not useful") }
func (statusCodeDomain) Successor(a StatusCode) StatusCode     { panic("not useful") }
func (statusCodeDomain) Samples(a StatusCode) []StatusCode {
	return []StatusCode{Running, Stopped, Returned, Reverted, Failed}
}
func (statusCodeDomain) SamplesForAll(a []StatusCode) []StatusCode {
	return []StatusCode{Running, Stopped, Returned, Reverted, Failed}
}

type uint16Domain struct{}

func (uint16Domain) Equal(a uint16, b uint16) bool { return a == b }
func (uint16Domain) Less(a uint16, b uint16) bool  { return a < b }
func (uint16Domain) Predecessor(a uint16) uint16   { return a - 1 }
func (uint16Domain) Successor(a uint16) uint16     { return a + 1 }
func (d uint16Domain) Samples(a uint16) []uint16 {
	return d.SamplesForAll([]uint16{a})
}
func (uint16Domain) SamplesForAll(as []uint16) []uint16 {
	res := []uint16{0, ^uint16(0)}

	// Test every element off by one.
	for _, a := range as {
		res = append(res, a-1)
		res = append(res, a)
		res = append(res, a+1)
	}

	// Add all powers of 2.
	for i := 0; i < 16; i++ {
		res = append(res, uint16(1<<i))
	}

	// TODO: consider removing duplicates.

	return res
}

type uint64Domain struct{}

func (uint64Domain) Equal(a uint64, b uint64) bool { return a == b }
func (uint64Domain) Less(a uint64, b uint64) bool  { return a < b }
func (uint64Domain) Predecessor(a uint64) uint64   { return a - 1 }
func (uint64Domain) Successor(a uint64) uint64     { return a + 1 }
func (d uint64Domain) Samples(a uint64) []uint64 {
	return d.SamplesForAll([]uint64{a})
}
func (uint64Domain) SamplesForAll(as []uint64) []uint64 {
	res := []uint64{0, ^uint64(0)}

	// Test every element off by one.
	for _, a := range as {
		res = append(res, a-1)
		res = append(res, a)
		res = append(res, a+1)
	}

	// Add all powers of 2.
	for i := 0; i < 64; i++ {
		res = append(res, uint64(1<<i))
	}

	// TODO: consider removing duplicates.

	return res
}

type codeDomain struct{}

func (codeDomain) Equal(a []byte, b []byte) bool   { return slices.Equal(a, b) }
func (codeDomain) Less(a []byte, b []byte) bool    { panic("not useful") }
func (codeDomain) Predecessor(a []byte) []byte     { panic("not useful") }
func (codeDomain) Successor(a []byte) []byte       { panic("not useful") }
func (codeDomain) Samples(a []byte) [][]byte       { return [][]byte{a} }
func (codeDomain) SamplesForAll([][]byte) [][]byte { panic("not useful") }

type opCodeDomain struct{}

func (opCodeDomain) Equal(a OpCode, b OpCode) bool { return a == b }
func (opCodeDomain) Less(a OpCode, b OpCode) bool  { panic("not useful") }
func (opCodeDomain) Predecessor(a OpCode) OpCode   { panic("not useful") }
func (opCodeDomain) Successor(a OpCode) OpCode     { panic("not useful") }
func (opCodeDomain) Samples(a OpCode) []OpCode     { return []OpCode{a} }
func (opCodeDomain) SamplesForAll([]OpCode) []OpCode {
	res := make([]OpCode, 0, 256)
	for i := 0; i < 256; i++ {
		res = append(res, OpCode(i))
	}
	return res
}

type stackSizeDomain struct{}

func (stackSizeDomain) Equal(a int, b int) bool { return a == b }
func (stackSizeDomain) Less(a int, b int) bool  { return a < b }
func (stackSizeDomain) Predecessor(a int) int   { return a - 1 }
func (stackSizeDomain) Successor(a int) int     { return a + 1 }
func (d stackSizeDomain) Samples(a int) []int {
	return d.SamplesForAll([]int{a})
}
func (stackSizeDomain) SamplesForAll(as []int) []int {
	res := []int{0, 1024} // extreme values

	// Test every element off by one.
	for _, a := range as {
		if 0 <= a && a <= 1024 {
			if a != 0 {
				res = append(res, a-1)
			}
			res = append(res, a)
			if a != 1024 {
				res = append(res, a+1)
			}
		}
	}

	// TODO: consider removing duplicates.

	return res
}

// ----------------------------------------------------------------------------
//                                 Expressions
// ----------------------------------------------------------------------------

// --- status ---

type status struct{}

func Status() Expression[StatusCode] {
	return status{}
}

func (status) Domain() Domain[StatusCode] { return statusCodeDomain{} }

func (status) Eval(s State) StatusCode {
	return s.Status
}

func (status) eval(s *StateBuilder) StatusCode {
	return s.GetStatus()
}

func (status) set(status StatusCode, builder *StateBuilder) {
	builder.SetStatus(status)
}

func (status) String() string {
	return "status"
}

// --- code ---

type code struct{}

func Code() Expression[[]byte] {
	return code{}
}

func (code) Domain() Domain[[]byte] { return codeDomain{} }

func (code) Eval(s State) []byte {
	return s.Code
}

func (code) eval(s *StateBuilder) []byte {
	return s.GetCode()
}

func (code) set(code []byte, builder *StateBuilder) {
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

// --- stack size ---

type stackSize struct{}

func StackSize() Expression[int] {
	return stackSize{}
}

func (stackSize) Domain() Domain[int] { return stackSizeDomain{} }

func (stackSize) Eval(s State) int {
	return s.Stack.Size()
}

func (stackSize) eval(s *StateBuilder) int {
	return s.GetStackSize()
}

func (stackSize) set(stackSize int, builder *StateBuilder) {
	builder.SetStackSize(stackSize)
}

func (stackSize) String() string {
	return "stackSize"
}

// ----------------------------------------------------------------------------
//                                  Effects
// ----------------------------------------------------------------------------

// TODO: have more structured effects

func Update(change func(State) State) Effect {
	return &effect{change}
}

type effect struct {
	change func(State) State
}

func (e *effect) Apply(state State) State {
	return e.change(state)
}

func (e *effect) String() string {
	return "change"
}

// ----------------------------------------------------------------------------
//                                   Parameter
// ----------------------------------------------------------------------------

type NumericParameter struct{}

func (NumericParameter) Samples() []uint256.Int {
	return []uint256.Int{
		*uint256.NewInt(0),
		*uint256.NewInt(1),
		*uint256.NewInt(1 << 8),
		*uint256.NewInt(1 << 16),
		*uint256.NewInt(1 << 32),
		*uint256.NewInt(1 << 48),
		*uint256.NewInt(1).Lsh(uint256.NewInt(1), 64),
		*uint256.NewInt(1).Lsh(uint256.NewInt(1), 128),
		*uint256.NewInt(1).Lsh(uint256.NewInt(1), 192),
		*uint256.NewInt(1).Lsh(uint256.NewInt(1), 255),
		*uint256.NewInt(0).Not(uint256.NewInt(0)),
	}
}
