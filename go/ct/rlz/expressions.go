package rlz

import (
	"fmt"
	"math"

	. "github.com/Fantom-foundation/Tosca/go/ct/common"
	"github.com/Fantom-foundation/Tosca/go/ct/gen"
	"github.com/Fantom-foundation/Tosca/go/ct/st"
)

type Expression[T any] interface {
	Domain() Domain[T]

	// Eval evaluates this expression on the given state.
	Eval(*st.State) (T, error)

	// Restrict applies constraints to the given generator such that this
	// expression evaluates to the given value when invoked on the generated
	// states.
	Restrict(value T, generator *gen.StateGenerator)

	fmt.Stringer
}

// Bindable is an Expression that can be referenced as a Variable.
type BindableExpression[T any] interface {
	// GetVariable returns the variable referring to this Expression.
	GetVariable() gen.Variable

	// BindTo adds constraints to the given generator modelling this Expression.
	BindTo(generator *gen.StateGenerator)

	Expression[T]
}

////////////////////////////////////////////////////////////
// st.Status

type status struct{}

func Status() Expression[st.StatusCode] {
	return status{}
}

func (status) Domain() Domain[st.StatusCode] { return statusCodeDomain{} }

func (status) Eval(s *st.State) (st.StatusCode, error) {
	return s.Status, nil
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

func Pc() BindableExpression[U256] {
	return pc{}
}

func (pc) Domain() Domain[U256] { return pcDomain{} }

func (pc) Eval(s *st.State) (U256, error) {
	return NewU256(uint64(s.Pc)), nil
}

func (pc) Restrict(pc U256, generator *gen.StateGenerator) {
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

func (e pc) BindTo(generator *gen.StateGenerator) {
	generator.BindPc(e.GetVariable())
}

////////////////////////////////////////////////////////////
// Gas Counter

type gas struct{}

func Gas() Expression[uint64] {
	return gas{}
}

func (gas) Domain() Domain[uint64] { return uint64Domain{} }

func (gas) Eval(s *st.State) (uint64, error) {
	return s.Gas, nil
}

func (gas) Restrict(amount uint64, generator *gen.StateGenerator) {
	generator.SetGas(amount)
}

func (gas) String() string {
	return "Gas"
}

////////////////////////////////////////////////////////////
// Code Operation

type op struct {
	position BindableExpression[U256]
}

func Op(position BindableExpression[U256]) Expression[OpCode] {
	return op{position}
}

func (op) Domain() Domain[OpCode] { return opCodeDomain{} }

func (e op) Eval(s *st.State) (OpCode, error) {
	pos, err := e.position.Eval(s)
	if err != nil {
		return INVALID, err
	}

	if pos.Gt(NewU256(math.MaxInt)) {
		return STOP, nil
	}

	op, err := s.Code.GetOperation(int(pos.Uint64()))
	if err != nil {
		return INVALID, err
	}
	return op, nil
}

func (e op) Restrict(op OpCode, generator *gen.StateGenerator) {
	variable := e.position.GetVariable()
	e.position.BindTo(generator)
	generator.AddCodeOperation(variable, op)
}

func (e op) String() string {
	return fmt.Sprintf("code[%v]", e.position)
}

////////////////////////////////////////////////////////////
// Stack Size

type stackSize struct{}

func StackSize() Expression[int] {
	return stackSize{}
}

func (stackSize) Domain() Domain[int] { return stackSizeDomain{} }

func (stackSize) Eval(s *st.State) (int, error) {
	return s.Stack.Size(), nil
}

func (stackSize) Restrict(size int, generator *gen.StateGenerator) {
	generator.SetStackSize(size)
}

func (stackSize) String() string {
	return "stackSize"
}
