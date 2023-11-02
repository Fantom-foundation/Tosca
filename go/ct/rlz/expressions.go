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
