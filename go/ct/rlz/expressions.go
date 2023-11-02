package rlz

import (
	"fmt"
	"math"

	"pgregory.net/rand"

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
	Restrict(value T, generator *gen.StateGenerator, rnd *rand.Rand)

	// Pick returns a specific value for this expression. If the generator is
	// already constrained, the value satisfies the constraints (if possible);
	// otherwise the generator is constraint accordingly.
	Pick(generator *gen.StateGenerator, rnd *rand.Rand) T

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

func (status) Restrict(status st.StatusCode, generator *gen.StateGenerator, rnd *rand.Rand) {
	generator.SetStatus(status)
}

func (status) Pick(generator *gen.StateGenerator, rnd *rand.Rand) st.StatusCode {
	return generator.PickStatus(rnd)
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

func (pc) Restrict(pc ct.U256, generator *gen.StateGenerator, rnd *rand.Rand) {
	if !pc.IsUint64() || pc.Uint64() > uint64(math.MaxUint16) {
		panic("invalid value for PC")
	}
	generator.SetPc(uint16(pc.Uint64()))
}

func (pc) Pick(generator *gen.StateGenerator, rnd *rand.Rand) ct.U256 {
	panic("TODO")
}

func (pc) String() string {
	return "PC"
}

////////////////////////////////////////////////////////////
// Code Operation

type op struct {
	position Expression[ct.U256]
}

func Op(position Expression[ct.U256]) Expression[st.OpCode] {
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

func (e op) Restrict(op st.OpCode, generator *gen.StateGenerator, rnd *rand.Rand) {
	pos := e.position.Pick(generator, rnd)
	if pos.Gt(ct.NewU256(math.MaxUint16)) {
		panic("invalid pos") // TODO
	}

	generator.SetCodeOperation(int(pos.Uint64()), op)
}

func (e op) Pick(generator *gen.StateGenerator, rnd *rand.Rand) st.OpCode {
	pos := e.position.Pick(generator, rnd)
	if pos.Gt(ct.NewU256(math.MaxUint16)) {
		panic("invalid pos") // TODO
	}

	return generator.PickCodeOperation(int(pos.Uint64()), rnd)
}

func (e op) String() string {
	return fmt.Sprintf("code[%v]", e.position)
}
