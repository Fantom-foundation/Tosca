package rlz

import (
	"fmt"
	"math"

	"github.com/Fantom-foundation/Tosca/go/ct"
	"github.com/Fantom-foundation/Tosca/go/ct/gen"
	"github.com/Fantom-foundation/Tosca/go/ct/st"
)

type Expression[T any] interface {
	Domain() Domain[T]

	// Eval evaluates this expression on the given state.
	Eval(*st.State) T

	// Restrict applies constraints to the given generator such that this
	// expression evaluates to the given value when invoked on the generated
	// states.
	Restrict(value T, generator *gen.StateGenerator)

	fmt.Stringer
}

type Bindable[T any] interface {
	Expression[T]
	GetVariable() gen.Variable
	BindTo(generator *gen.StateGenerator)
}

////////////////////////////////////////////////////////////
// st.Status

type status struct{}

func Status() Expression[st.StatusCode] {
	return status{}
}

func (status) Domain() Domain[st.StatusCode] { return statusCodeDomain{} }

func (status) Eval(s *st.State) st.StatusCode {
	return s.Status
}

func (status) Restrict(status st.StatusCode, generator *gen.StateGenerator) {
	generator.SetStatus(status)
}

func (status) String() string {
	return "status"
}

////////////////////////////////////////////////////////////
// Program Counter

type pc struct{}

func Pc() Bindable[ct.U256] {
	return pc{}
}

func (pc) Domain() Domain[ct.U256] { return pcDomain{} }

func (pc) Eval(s *st.State) ct.U256 {
	return ct.NewU256(uint64(s.Pc))
}

func (pc) Restrict(pc ct.U256, generator *gen.StateGenerator) {
	if !pc.IsUint64() || pc.Uint64() > uint64(math.MaxUint16) {
		panic("invalid value for PC")
	}
	generator.SetPc(uint16(pc.Uint64()))
}

func (pc) String() string {
	return "PC"
}

func (pc) GetVariable() gen.Variable {
	return gen.Variable("PC")
}

func (p pc) BindTo(generator *gen.StateGenerator) {
	generator.BindPc(p.GetVariable())
}

////////////////////////////////////////////////////////////
// Gas Counter

type gas struct{}

func Gas() Expression[uint64] {
	return gas{}
}

func (gas) Domain() Domain[uint64] { return uint64Domain{} }

func (gas) Eval(s *st.State) uint64 {
	return s.Gas
}

func (gas) Restrict(amount uint64, generator *gen.StateGenerator) {
	generator.SetGas(amount)
}

func (gas) String() string {
	return "Gas"
}

////////////////////////////////////////////////////////////
// Stack Size

type stackSize struct{}

func StackSize() Expression[int] {
	return stackSize{}
}

func (stackSize) Domain() Domain[int] { return stackSizeDomain{} }

func (stackSize) Eval(s *st.State) int {
	return s.Stack.Size()
}

func (stackSize) Restrict(size int, generator *gen.StateGenerator) {
	generator.SetStackSize(size)
}

func (stackSize) String() string {
	return "stackSize"
}

////////////////////////////////////////////////////////////
// Code Operation

type op struct {
	position Bindable[ct.U256]
}

func Op(position Bindable[ct.U256]) Expression[st.OpCode] {
	return op{position}
}

func (op) Domain() Domain[st.OpCode] { return opCodeDomain{} }

func (e op) Eval(s *st.State) st.OpCode {
	pos := e.position.Eval(s)
	if pos.Gt(ct.NewU256(math.MaxInt)) {
		return st.STOP
	}

	op, err := s.Code.GetOperation(int(pos.Uint64()))
	if err != nil {
		panic(err) // TODO
	}
	return op
}

func (e op) Restrict(op st.OpCode, generator *gen.StateGenerator) {
	variable := e.position.GetVariable()
	e.position.BindTo(generator)
	generator.AddCodeOperation(variable, op)
}

func (e op) String() string {
	return fmt.Sprintf("code[%v]", e.position)
}

////////////////////////////////////////////////////////////
// Instruction Parameter

type param struct {
	position int
}

func Param(pos int) Bindable[ct.U256] {
	return param{pos}
}

func (param) Domain() Domain[ct.U256] { return u256Domain{} }

func (p param) Eval(s *st.State) ct.U256 {
	stack := s.Stack
	if p.position >= stack.Size() {
		return ct.U256{}
	}
	return stack.Get(p.position)
}

func (p param) Restrict(value ct.U256, generator *gen.StateGenerator) {
	generator.SetStackValue(p.position, value)
}

func (p param) String() string {
	return fmt.Sprintf("param[%v]", p.position)
}

func (p param) GetVariable() gen.Variable {
	return gen.Variable(fmt.Sprintf("param_%d", p.position))
}

func (p param) BindTo(generator *gen.StateGenerator) {
	generator.BindStackValue(p.position, p.GetVariable())
}
