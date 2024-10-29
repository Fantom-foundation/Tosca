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
	"math"

	. "github.com/Fantom-foundation/Tosca/go/ct/common"
	"github.com/Fantom-foundation/Tosca/go/ct/gen"
	"github.com/Fantom-foundation/Tosca/go/ct/st"
	"github.com/Fantom-foundation/Tosca/go/tosca"
	"github.com/Fantom-foundation/Tosca/go/tosca/vm"
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
	Property() Property
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

func (status) Property() Property { return Property("status") }

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

func (pc) Property() Property { return Property("pc") }

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

func Gas() Expression[tosca.Gas] {
	return gas{}
}

func (gas) Property() Property { return Property("gas") }

func (gas) Domain() Domain[tosca.Gas] { return gasDomain{} }

func (gas) Eval(s *st.State) (tosca.Gas, error) {
	return s.Gas, nil
}

func (gas) Restrict(kind RestrictionKind, amount tosca.Gas, generator *gen.StateGenerator) {
	switch kind {
	case RestrictLess:
		generator.AddGasUpperBound(amount - 1)
	case RestrictLessEqual:
		generator.AddGasUpperBound(amount)
	case RestrictEqual:
		generator.SetGas(amount)
	case RestrictGreaterEqual:
		generator.AddGasLowerBound(amount)
	case RestrictGreater:
		generator.AddGasLowerBound(amount + 1)
	}
}

func (gas) String() string {
	return "Gas"
}

////////////////////////////////////////////////////////////
// SelfAddress - the address of the called contract

type selfAddress struct{}

func SelfAddress() BindableExpression[tosca.Address] {
	return selfAddress{}
}

func (selfAddress) Property() Property { return Property("selfAddress") }

func (selfAddress) Domain() Domain[tosca.Address] { return addressDomain{} }

func (selfAddress) Eval(s *st.State) (tosca.Address, error) {
	return s.CallContext.AccountAddress, nil
}

func (selfAddress) Restrict(kind RestrictionKind, address tosca.Address, generator *gen.StateGenerator) {
	if kind != RestrictEqual {
		panic("Self can only support equality constraints")
	}
	generator.SetSelfAddress(address)
}

func (selfAddress) String() string {
	return "Self"
}

func (selfAddress) GetVariable() gen.Variable {
	return gen.Variable("self")
}

func (s selfAddress) BindTo(generator *gen.StateGenerator) {
	generator.BindToSelfAddress(s.GetVariable())
}

// //////////////////////////////////////////////////////////
// Read Only Mode
type readOnly struct{}

func ReadOnly() Expression[bool] {
	return readOnly{}
}

func (readOnly) Property() Property { return Property("readOnly") }

func (readOnly) Domain() Domain[bool] { return readOnlyDomain{} }

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
// Balance

type balance struct {
	account BindableExpression[tosca.Address]
}

func Balance(account BindableExpression[tosca.Address]) Expression[U256] {
	return balance{account}
}

func (b balance) Property() Property { return Property(b.String()) }

func (balance) Domain() Domain[U256] { return u256Domain{} }

func (b balance) Eval(s *st.State) (U256, error) {
	address, err := b.account.Eval(s)
	if err != nil {
		return U256{}, err
	}
	return s.Accounts.GetBalance(address), nil
}

func (b balance) Restrict(kind RestrictionKind, value U256, generator *gen.StateGenerator) {
	variable := b.account.GetVariable()
	b.account.BindTo(generator)

	switch kind {
	case RestrictLess:
		generator.AddBalanceUpperBound(variable, value.Sub(NewU256(1)))
	case RestrictLessEqual:
		generator.AddBalanceUpperBound(variable, value)
	case RestrictEqual:
		generator.AddBalanceLowerBound(variable, value)
		generator.AddBalanceUpperBound(variable, value)
	case RestrictGreaterEqual:
		generator.AddBalanceLowerBound(variable, value)
	case RestrictGreater:
		generator.AddBalanceLowerBound(variable, value.Add(NewU256(1)))
	}
}

func (b balance) String() string {
	return fmt.Sprintf("balance(%v)", b.account)
}

////////////////////////////////////////////////////////////
// Code Operation

type op struct {
	position BindableExpression[U256]
}

func Op(position BindableExpression[U256]) Expression[vm.OpCode] {
	return op{position}
}

func (e op) Property() Property { return Property(e.String()) }

func (op) Domain() Domain[vm.OpCode] { return opCodeDomain{} }

func (e op) Eval(s *st.State) (vm.OpCode, error) {
	pos, err := e.position.Eval(s)
	if err != nil {
		return vm.INVALID, err
	}

	if !pos.IsUint64() || pos.Uint64() > math.MaxInt {
		return vm.STOP, nil
	}

	op, err := s.Code.GetOperation(int(pos.Uint64()))
	if err != nil {
		return vm.INVALID, err
	}
	return op, nil
}

func (e op) Restrict(kind RestrictionKind, op vm.OpCode, generator *gen.StateGenerator) {
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

func (stackSize) Property() Property { return Property("stackSize") }

func (stackSize) Domain() Domain[int] { return stackSizeDomain{} }

func (stackSize) Eval(s *st.State) (int, error) {
	return s.Stack.Size(), nil
}

func (stackSize) Restrict(kind RestrictionKind, size int, generator *gen.StateGenerator) {
	switch kind {
	case RestrictLess:
		generator.AddStackSizeUpperBound(size - 1)
	case RestrictLessEqual:
		generator.AddStackSizeUpperBound(size)
	case RestrictEqual:
		generator.SetStackSize(size)
	case RestrictGreaterEqual:
		generator.AddStackSizeLowerBound(size)
	case RestrictGreater:
		generator.AddStackSizeLowerBound(size + 1)
	}
}

func (stackSize) String() string {
	return "stackSize"
}

////////////////////////////////////////////////////////////
// Instruction Parameter

type param struct {
	position int
	domain   Domain[U256]
}

const ErrStackOutOfBoundsAccess = ConstErr("out-of-bounds stack access")

func Param(pos int) BindableExpression[U256] {
	return param{pos, u256Domain{}}
}

func ValueParam(pos int) BindableExpression[U256] {
	return param{pos, valueDomain{}}
}

func (p param) Property() Property { return Property(p.String()) }

func (p param) Domain() Domain[U256] { return p.domain }

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

////////////////////////////////////////////////////////////
// Constants

type constant struct {
	value U256
}

// Constant creates a bindable expression that can only be bounded to the
// provided value. It can, for instance, be used to fix the operation at
// a fixed position in the code.
func Constant(value U256) BindableExpression[U256] {
	return constant{value}
}

func (c constant) Property() Property {
	return Property(fmt.Sprintf("constant(%v)", c.value.ToBigInt()))
}

func (constant) Domain() Domain[U256] { return u256Domain{} }

func (c constant) Eval(*st.State) (U256, error) {
	return c.value, nil
}

func (c constant) Restrict(kind RestrictionKind, value U256, generator *gen.StateGenerator) {
	panic("not implemented")
}

func (c constant) String() string {
	if c.value.IsUint64() {
		return fmt.Sprintf("%d", c.value.Uint64())
	}
	return fmt.Sprintf("%v", c.value)
}

func (c constant) GetVariable() gen.Variable {
	return gen.Variable(fmt.Sprintf("constant_%s", c.String()))
}

func (c constant) BindTo(generator *gen.StateGenerator) {
	generator.BindValue(c.GetVariable(), c.value)
}

////////////////////////////////////////////////////////////
// ToAddress

type toAddress struct {
	expr BindableExpression[U256]
}

func ToAddress(expr BindableExpression[U256]) BindableExpression[tosca.Address] {
	return toAddress{expr}
}

func (a toAddress) Property() Property { return a.expr.Property() }

func (a toAddress) Domain() Domain[tosca.Address] { return addressDomain{} }

func (a toAddress) Eval(s *st.State) (tosca.Address, error) {
	value, err := a.expr.Eval(s)
	if err != nil {
		return tosca.Address{}, err
	}
	return NewAddress(value), nil
}

func (a toAddress) Restrict(RestrictionKind, tosca.Address, *gen.StateGenerator) {
	panic("should not be needed")
}

func (a toAddress) String() string {
	return fmt.Sprintf("toAddress(%v)", a.expr)
}

func (a toAddress) GetVariable() gen.Variable {
	return a.expr.GetVariable()
}

func (a toAddress) BindTo(generator *gen.StateGenerator) {
	a.expr.BindTo(generator)
}
