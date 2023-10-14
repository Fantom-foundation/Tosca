package ct

import (
	"fmt"
	"math"
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
	SomethingNotEqual(T) T
	Samples(T) []T
	SamplesForAll([]T) []T
}

type Parameter interface {
	Samples(example uint256.Int) []uint256.Int
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
func (rule *Rule) GetSatisfyingState() State {
	builder := NewStateBuilder()
	rule.Condition.restrict(builder)
	return builder.Build()
}

// EnumerateTestCases enumerates a list of states representing relevant test
// cases for the given condition. At least one of those cases is satisfying
// the condition.
func (rule *Rule) EnumerateTestCases(consume func(s State)) {
	builder := NewStateBuilder()
	rule.Condition.enumerateTestCases(builder, func(s *StateBuilder) {
		enumerateParameters(0, rule.Parameter, s, func(s *StateBuilder) {
			consume(s.Build())
		})
	})
}

func enumerateParameters(pos int, params []Parameter, builder *StateBuilder, consume func(s *StateBuilder)) {
	if len(params) == 0 {
		consume(builder)
		return
	}
	current := builder.GetStackValue(pos)
	for _, value := range params[0].Samples(current) {
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

type ne[T any] struct {
	lhs Expression[T]
	rhs T
}

func Ne[T any](lhs Expression[T], rhs T) Condition {
	return &ne[T]{lhs, rhs}
}

func (e *ne[T]) Check(s State) bool {
	domain := e.lhs.Domain()
	return !domain.Equal(e.lhs.Eval(s), e.rhs)
}

func (e *ne[T]) restrict(builder *StateBuilder) {
	domain := e.lhs.Domain()
	e.lhs.set(domain.SomethingNotEqual(e.rhs), builder)
}

func (e *ne[T]) enumerateTestCases(builder *StateBuilder, consumer func(*StateBuilder)) {
	domain := e.lhs.Domain()
	for _, value := range domain.Samples(e.rhs) {
		clone := builder.Clone()
		e.lhs.set(value, clone)
		consumer(clone)
	}
}

func (e *ne[T]) String() string {
	return fmt.Sprintf("%s ≠ %v", e.lhs, e.rhs)
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

type isCode struct {
	position Expression[uint256.Int]
}

func IsCode(position Expression[uint256.Int]) Condition {
	return &isCode{position}
}

func (c *isCode) Check(s State) bool {
	pos := c.position.Eval(s)
	if !pos.IsUint64() {
		return false
	}
	if pos.Uint64() > math.MaxInt {
		return false
	}
	return s.IsCode(int(pos.Uint64()))
}

func (c *isCode) restrict(builder *StateBuilder) {
	// pick a random PC in the code.
	length := builder.GetCodeLength()
	pos := int(builder.random.Int31n(int32(length)))
	pos = builder.state.GetNextCodePosition(pos)

	// For now, only certain types of position parameters are supported
	if _, isPc := c.position.(pc); isPc {
		builder.SetPc(uint16(pos))
	} else if _, isParam := c.position.(param); isParam {
		builder.SetStackValue(c.position.(param).position, *uint256.NewInt(uint64(pos)))
	} else {
		panic("so far only Pc(..) and Param(..) values are supported if isCode constraints")
	}
}

func (c *isCode) enumerateTestCases(builder *StateBuilder, consumer func(*StateBuilder)) {
	positive := builder.Clone()
	c.restrict(positive)
	consumer(positive)

	negative := builder.Clone()
	IsData(c.position).restrict(negative)
	consumer(negative)
}

func (c *isCode) String() string {
	return fmt.Sprintf("isCode[%s]", c.position)
}

type isData struct {
	position Expression[uint256.Int]
}

func IsData(position Expression[uint256.Int]) Condition {
	return &isData{position}
}

func (c *isData) Check(s State) bool {
	return !IsCode(c.position).Check(s)
}

func (c *isData) restrict(builder *StateBuilder) {
	backup := builder.Clone()
	// pick a random PC in the code.
	length := builder.GetCodeLength()
	pos := int(builder.random.Int31n(int32(length)))

	data, found := builder.state.GetNextDataPosition(pos)
	if !found {
		// If there is no data in the code we can restart the generation
		// with some different code.
		if backup.isFixed(sp_CodeLength) {
			// TODO: handle this case better, e.g. by reporting back an issue
			fmt.Printf("WARNING: there is no data section in the program and the program was already fixed\n")
			return
			//panic("there is no data section in the program and can't change the program")
		}
		// TODO: add some check making sure this does not loop forever
		state := builder.Build()
		fmt.Printf("state: %v\n", &state)
		builder.Restore(backup)
		c.restrict(builder)
		return
	}

	// For now, only certain types of position parameters are supported
	if _, isPc := c.position.(pc); isPc {
		builder.SetPc(uint16(data))
	} else if _, isParam := c.position.(param); isParam {
		builder.SetStackValue(c.position.(param).position, *uint256.NewInt(uint64(data)))
	} else {
		panic("so far only Pc(..) and Param(..) values are supported if isData constraints")
	}
}

func (c *isData) enumerateTestCases(builder *StateBuilder, consumer func(*StateBuilder)) {
	positive := builder.Clone()
	c.restrict(positive)
	consumer(positive)

	negative := builder.Clone()
	IsCode(c.position).restrict(negative)
	consumer(negative)
}

func (c *isData) String() string {
	return fmt.Sprintf("isData[%s]", c.position)
}

// ----------------------------------------------------------------------------
//                                   Domains
// ----------------------------------------------------------------------------

type booleanDomain struct{}

func (booleanDomain) Equal(a bool, b bool) bool { return a == b }
func (booleanDomain) Less(a bool, b bool) bool  { panic("not useful") }
func (booleanDomain) Predecessor(a bool) bool   { panic("not useful") }
func (booleanDomain) Successor(a bool) bool     { panic("not useful") }
func (booleanDomain) SomethingNotEqual(a bool) bool {
	return !a
}
func (booleanDomain) Samples(bool) []bool {
	return []bool{false, true}
}
func (booleanDomain) SamplesForAll(_ []bool) []bool {
	return []bool{false, true}
}

type statusCodeDomain struct{}

func (statusCodeDomain) Equal(a StatusCode, b StatusCode) bool { return a == b }
func (statusCodeDomain) Less(a StatusCode, b StatusCode) bool  { panic("not useful") }
func (statusCodeDomain) Predecessor(a StatusCode) StatusCode   { panic("not useful") }
func (statusCodeDomain) Successor(a StatusCode) StatusCode     { panic("not useful") }
func (statusCodeDomain) SomethingNotEqual(a StatusCode) StatusCode {
	if a == Running {
		return Stopped
	}
	return Running
}
func (statusCodeDomain) Samples(a StatusCode) []StatusCode {
	return []StatusCode{Running, Stopped, Returned, Reverted, Failed}
}
func (statusCodeDomain) SamplesForAll(a []StatusCode) []StatusCode {
	return []StatusCode{Running, Stopped, Returned, Reverted, Failed}
}

type uint16Domain struct{}

func (uint16Domain) Equal(a uint16, b uint16) bool     { return a == b }
func (uint16Domain) Less(a uint16, b uint16) bool      { return a < b }
func (uint16Domain) Predecessor(a uint16) uint16       { return a - 1 }
func (uint16Domain) Successor(a uint16) uint16         { return a + 1 }
func (uint16Domain) SomethingNotEqual(a uint16) uint16 { return a + 1 }
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

func (uint64Domain) Equal(a uint64, b uint64) bool     { return a == b }
func (uint64Domain) Less(a uint64, b uint64) bool      { return a < b }
func (uint64Domain) Predecessor(a uint64) uint64       { return a - 1 }
func (uint64Domain) Successor(a uint64) uint64         { return a + 1 }
func (uint64Domain) SomethingNotEqual(a uint64) uint64 { return a + 1 }
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

type uint256Domain struct{}

func (uint256Domain) Equal(a uint256.Int, b uint256.Int) bool { return a == b }
func (uint256Domain) Less(a uint256.Int, b uint256.Int) bool  { return a.Lt(&b) }
func (uint256Domain) Predecessor(a uint256.Int) uint256.Int   { return *a.Sub(&a, uint256.NewInt(1)) }
func (uint256Domain) Successor(a uint256.Int) uint256.Int     { return *a.Add(&a, uint256.NewInt(1)) }
func (uint256Domain) SomethingNotEqual(a uint256.Int) uint256.Int {
	return *a.Add(&a, uint256.NewInt(1))
}
func (d uint256Domain) Samples(a uint256.Int) []uint256.Int {
	return d.SamplesForAll([]uint256.Int{a})
}
func (d uint256Domain) SamplesForAll(as []uint256.Int) []uint256.Int {
	res := []uint256.Int{}

	// Test every element off by one.
	for _, a := range as {
		res = append(res, d.Predecessor(a))
		res = append(res, a)
		res = append(res, d.Successor(a))
	}

	// Add more interesting values.
	res = append(res, NumericParameter{}.SampleValues()...)

	// TODO: consider removing duplicates.

	return res
}

type pcDomain struct{}

func (pcDomain) Equal(a uint256.Int, b uint256.Int) bool     { return a == b }
func (pcDomain) Less(a uint256.Int, b uint256.Int) bool      { return a.Lt(&b) }
func (pcDomain) Predecessor(a uint256.Int) uint256.Int       { return *a.Sub(&a, uint256.NewInt(1)) }
func (pcDomain) Successor(a uint256.Int) uint256.Int         { return *a.Add(&a, uint256.NewInt(1)) }
func (pcDomain) SomethingNotEqual(a uint256.Int) uint256.Int { return *a.Add(&a, uint256.NewInt(1)) }
func (d pcDomain) Samples(a uint256.Int) []uint256.Int {
	return d.SamplesForAll([]uint256.Int{a})
}
func (pcDomain) SamplesForAll(as []uint256.Int) []uint256.Int {
	pcs := []uint16{}
	for _, a := range as {
		if a.IsUint64() && a.Uint64() <= uint64(math.MaxUint16) {
			pcs = append(pcs, uint16(a.Uint64()))
		}
	}

	pcs = uint16Domain{}.SamplesForAll(pcs)

	res := make([]uint256.Int, 0, len(pcs))
	for _, cur := range pcs {
		res = append(res, *uint256.NewInt(uint64(cur)))
	}
	return res
}

type codeDomain struct{}

func (codeDomain) Equal(a []byte, b []byte) bool     { return slices.Equal(a, b) }
func (codeDomain) Less(a []byte, b []byte) bool      { panic("not useful") }
func (codeDomain) Predecessor(a []byte) []byte       { panic("not useful") }
func (codeDomain) Successor(a []byte) []byte         { panic("not useful") }
func (codeDomain) SomethingNotEqual(a []byte) []byte { panic("not implemented") }
func (codeDomain) Samples(a []byte) [][]byte         { return [][]byte{a} }
func (codeDomain) SamplesForAll([][]byte) [][]byte   { panic("not useful") }

type opCodeDomain struct{}

func (opCodeDomain) Equal(a OpCode, b OpCode) bool     { return a == b }
func (opCodeDomain) Less(a OpCode, b OpCode) bool      { panic("not useful") }
func (opCodeDomain) Predecessor(a OpCode) OpCode       { panic("not useful") }
func (opCodeDomain) Successor(a OpCode) OpCode         { panic("not useful") }
func (opCodeDomain) SomethingNotEqual(a OpCode) OpCode { return a + 1 }
func (opCodeDomain) Samples(a OpCode) []OpCode         { return []OpCode{a, a + 1} }
func (opCodeDomain) SamplesForAll([]OpCode) []OpCode {
	res := make([]OpCode, 0, 256)
	for i := 0; i < 256; i++ {
		res = append(res, OpCode(i))
	}
	return res
}

type stackSizeDomain struct{}

func (stackSizeDomain) Equal(a int, b int) bool     { return a == b }
func (stackSizeDomain) Less(a int, b int) bool      { return a < b }
func (stackSizeDomain) Predecessor(a int) int       { return a - 1 }
func (stackSizeDomain) Successor(a int) int         { return a + 1 }
func (stackSizeDomain) SomethingNotEqual(a int) int { return (a + 1) % 1024 }
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

// --- static ---

type static struct{}

func Static() Expression[bool] {
	return static{}
}

func (static) Domain() Domain[bool] { return booleanDomain{} }

func (static) Eval(s State) bool {
	return s.Static
}

func (static) eval(s *StateBuilder) bool {
	return s.GetStatic()
}

func (static) set(static bool, builder *StateBuilder) {
	builder.SetStatic(static)
}

func (static) String() string {
	return "static"
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

func Pc() Expression[uint256.Int] {
	return pc{}
}

func (pc) Domain() Domain[uint256.Int] { return pcDomain{} }

func (pc) Eval(s State) uint256.Int {
	return *uint256.NewInt(uint64(s.Pc))
}

func (pc) eval(s *StateBuilder) uint256.Int {
	return *uint256.NewInt(uint64(s.GetPc()))
}

func (pc) set(pc uint256.Int, builder *StateBuilder) {
	if !pc.IsUint64() || pc.Uint64() > uint64(math.MaxUint16) {
		panic("invalid value for PC")
	}
	builder.SetPc(uint16(pc.Uint64()))
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
	position Expression[uint256.Int]
}

func Op(position Expression[uint256.Int]) Expression[OpCode] {
	return op{position}
}

func (op) Domain() Domain[OpCode] { return opCodeDomain{} }

func (e op) Eval(s State) OpCode {
	pos := e.position.Eval(s)
	if !pos.IsUint64() || pos.Uint64() > uint64(math.MaxUint16) {
		return STOP
	}

	code := []byte(s.Code)
	if int(pos.Uint64()) < len(code) {
		return OpCode(code[pos.Uint64()])
	}
	return STOP
}

func (o op) eval(builder *StateBuilder) OpCode {
	pos := o.position.eval(builder)
	if !pos.IsUint64() || pos.Uint64() > uint64(math.MaxUint16) {
		return STOP
	}

	code := []byte(builder.GetCode())
	if int(pos.Uint64()) < len(code) {
		return OpCode(code[pos.Uint64()])
	}
	return STOP
}

func (o op) set(op OpCode, builder *StateBuilder) {
	pos := o.position.eval(builder)
	if !pos.IsUint64() || pos.Uint64() > uint64(math.MaxUint16) {
		// TODO: provide feedback to the caller that this set was not effective
		fmt.Printf("WARNING failed to set operation %d to %v\n", pos, op)
		//panic("out of range")
		return
	}

	builder.SetOpCode(uint16(pos.Uint64()), op)
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

// --- parameter ---

type param struct {
	position int
}

func Param(pos int) Expression[uint256.Int] {
	return param{pos}
}

func (param) Domain() Domain[uint256.Int] { return uint256Domain{} }

func (p param) Eval(s State) uint256.Int {
	stack := &s.Stack
	if p.position >= stack.Size() {
		return uint256.Int{}
	}
	return stack.Get(p.position)
}

func (p param) eval(builder *StateBuilder) uint256.Int {
	return builder.GetStackValue(p.position)
}

func (p param) set(value uint256.Int, builder *StateBuilder) {
	builder.SetStackValue(p.position, value)
}

func (p param) String() string {
	return fmt.Sprintf("param[%v]", p.position)
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

func (n NumericParameter) Samples(uint256.Int) []uint256.Int {
	return n.SampleValues()
}

func (NumericParameter) SampleValues() []uint256.Int {
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

type AddressParameter struct{}

func (AddressParameter) Samples(example uint256.Int) []uint256.Int {
	return []uint256.Int{example}
}

type GasParameter struct{}

func (GasParameter) Samples(uint256.Int) []uint256.Int {
	return []uint256.Int{
		*uint256.NewInt(0),
		*uint256.NewInt(1),
		*uint256.NewInt(1 << 32),
	}
}

type ValueParameter struct{}

func (ValueParameter) Samples(uint256.Int) []uint256.Int {
	return []uint256.Int{
		*uint256.NewInt(0),
		*uint256.NewInt(120),
		*uint256.NewInt(0).Not(uint256.NewInt(0)),
	}
}

type SizeParameter struct{}

func (SizeParameter) Samples(uint256.Int) []uint256.Int {
	return []uint256.Int{
		*uint256.NewInt(0),
		*uint256.NewInt(32),
		// TODO: expand when generator performance allows to
	}
}

type OffsetParameter struct{}

func (OffsetParameter) Samples(uint256.Int) []uint256.Int {
	return []uint256.Int{
		*uint256.NewInt(0),
		*uint256.NewInt(127),
		// TODO: expand when generator performance allows to
	}
}
