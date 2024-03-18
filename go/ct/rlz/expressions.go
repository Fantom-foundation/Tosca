package rlz

import (
	"fmt"
	"math"

	. "github.com/Fantom-foundation/Tosca/go/ct/common"
	"github.com/Fantom-foundation/Tosca/go/ct/gen"
	"github.com/Fantom-foundation/Tosca/go/ct/st"
	"github.com/Fantom-foundation/Tosca/go/vm"
)

type RestrictionKind int

const (
	RestrictLess RestrictionKind = iota
	RestrictLessEqual
	RestrictEqual
	RestrictGreaterEqual
	RestrictGreater
)

type Expression[T any] interface {
	Domain() Domain[T]

	// Eval evaluates this expression on the given state.
	Eval(*st.State) (T, error)

	// Restrict applies constraints to the given generator such that this
	// expression evaluates to the given value when invoked on the generated
	// states.
	Restrict(kind RestrictionKind, value T, generator *gen.StateGenerator)

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

func (status) Restrict(kind RestrictionKind, status st.StatusCode, generator *gen.StateGenerator) {
	if kind != RestrictEqual {
		panic("Status can only support equality constraints")
	}
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

func (pc) Restrict(kind RestrictionKind, pc U256, generator *gen.StateGenerator) {
	if kind != RestrictEqual {
		panic("PC can only support equality constraints")
	}
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

func Gas() Expression[vm.Gas] {
	return gas{}
}

func (gas) Domain() Domain[vm.Gas] { return gasDomain{} }

func (gas) Eval(s *st.State) (vm.Gas, error) {
	return s.Gas, nil
}

func (gas) Restrict(kind RestrictionKind, amount vm.Gas, generator *gen.StateGenerator) {
	switch kind {
	case RestrictLess:
		if amount == 0 {
			// TODO: offer different way of marking constraints unsatisfiable
			generator.SetGas(0)
			generator.SetGas(1)
		} else {
			generator.SetGas(amount - 1)
		}
	case RestrictLessEqual, RestrictEqual, RestrictGreaterEqual:
		generator.SetGas(amount)
	case RestrictGreater:
		generator.SetGas(amount + 1)
	}
}

func (gas) String() string {
	return "Gas"
}

////////////////////////////////////////////////////////////
// Gas Refund Counter

type gasRefund struct{}

func GasRefund() Expression[vm.Gas] {
	return gasRefund{}
}

func (gasRefund) Domain() Domain[vm.Gas] { return gasDomain{} }

func (gasRefund) Eval(s *st.State) (vm.Gas, error) {
	return s.GasRefund, nil
}

func (gasRefund) Restrict(kind RestrictionKind, amount vm.Gas, generator *gen.StateGenerator) {
	if kind != RestrictEqual {
		panic("GasRefund only supports equality constraints")
	}
	generator.SetGasRefund(amount)
}

func (gasRefund) String() string {
	return "GasRefund"
}

// //////////////////////////////////////////////////////////
// Read Only Mode
type readOnly struct{}

func ReadOnly() Expression[bool] {
	return readOnly{}
}

func (readOnly) Domain() Domain[bool] { return boolDomain{} }

func (readOnly) Eval(s *st.State) (bool, error) {
	return s.ReadOnly, nil
}

func (readOnly) Restrict(kind RestrictionKind, isSet bool, generator *gen.StateGenerator) {
	if kind != RestrictEqual {
		panic("ReadOnly only supports equality constraints")
	}
	generator.SetReadOnly(isSet)
}

func (readOnly) String() string {
	return "readOnly"
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

	if !pos.IsUint64() || pos.Uint64() > math.MaxInt {
		return STOP, nil
	}

	op, err := s.Code.GetOperation(int(pos.Uint64()))
	if err != nil {
		return INVALID, err
	}
	return op, nil
}

func (e op) Restrict(kind RestrictionKind, op OpCode, generator *gen.StateGenerator) {
	if kind != RestrictEqual {
		panic("Operation codes only support equality constraints")
	}
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

func (stackSize) Restrict(kind RestrictionKind, size int, generator *gen.StateGenerator) {
	switch kind {
	case RestrictLess:
		generator.SetMaxStackSize(size - 1)
	case RestrictLessEqual:
		generator.SetMaxStackSize(size)
	case RestrictEqual:
		generator.SetStackSize(size)
	case RestrictGreaterEqual:
		generator.SetMinStackSize(size)
	case RestrictGreater:
		generator.SetMinStackSize(size + 1)
	}
}

func (stackSize) String() string {
	return "stackSize"
}

////////////////////////////////////////////////////////////
// Instruction Parameter

type param struct {
	position int
}

const ErrStackOutOfBoundsAccess = ConstErr("out-of-bounds stack access")

func Param(pos int) BindableExpression[U256] {
	return param{pos}
}

func (param) Domain() Domain[U256] { return u256Domain{} }

func (p param) Eval(s *st.State) (U256, error) {
	stack := s.Stack
	if p.position >= stack.Size() {
		return NewU256(0), ErrStackOutOfBoundsAccess
	}
	return stack.Get(p.position), nil
}

func (p param) Restrict(kind RestrictionKind, value U256, generator *gen.StateGenerator) {
	if kind != RestrictEqual {
		panic("Parameters only support equality constraints")
	}
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
