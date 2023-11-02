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

func Pc() Expression[ct.U256] {
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
