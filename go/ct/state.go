package ct

import (
	"fmt"
	"math/rand"
	"strings"
	"time"
)

type State struct {
	Code string // since []byte is not comparable
	Pc   uint16
	Gas  uint64
}

func (s State) String() string {
	builder := strings.Builder{}
	builder.WriteString("{")
	builder.WriteString(fmt.Sprintf("Pc: %d, ", s.Pc))
	builder.WriteString(fmt.Sprintf("Gas: %d, ", s.Gas))
	if len(s.Code) > 20 {
		builder.WriteString(fmt.Sprintf("Code: %x...", s.Code[:20]))
	} else {
		builder.WriteString(fmt.Sprintf("Code: %x", s.Code))
	}
	builder.WriteString("}")
	return builder.String()
}

func GetRandomState() State {
	return GetRandomStateWithSeed(time.Now().UnixNano())
}

func GetRandomStateWithSeed(seed int64) State {
	return NewStateBuilderWithSeed(seed).Build()
}

// StateBuilder can be used to built a random state satisfying a set
// of constraints. Constraints
type StateBuilder struct {
	// The state under construction.
	state State

	// A set of fixed properties that can no longer be altered.
	fixed uint64

	random *rand.Rand
}

type stateProperty int

const (
	sp_CodeLength stateProperty = iota
	sp_Code
	sp_Pc
	sp_Gas
)

func NewStateBuilder() *StateBuilder {
	return NewStateBuilderWithSeed(time.Now().UnixNano())
}

func NewStateBuilderWithSeed(seed int64) *StateBuilder {
	return &StateBuilder{
		random: rand.New(rand.NewSource(seed)),
	}
}

func (b *StateBuilder) Clone() *StateBuilder {
	return &StateBuilder{
		state:  b.state,
		fixed:  b.fixed,
		random: rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

func (b *StateBuilder) SetCodeLength(length uint16) {
	if b.isFixed(sp_CodeLength) {
		panic("can only define code length once")
	}

	// Create the code and fill with random content.
	code := make([]byte, length)
	b.random.Read(code)
	b.state.Code = string(code)

	b.markFixed(sp_CodeLength)
}

func (b *StateBuilder) fixCodeLength(minimumLength uint16) {
	if b.isFixed(sp_CodeLength) {
		return
	}
	size := uint16(b.random.Uint32() % (24576 + 1))
	if size < minimumLength {
		size = minimumLength
	}
	b.SetCodeLength(size)
}

func (b *StateBuilder) SetCode(code string) {
	if b.isFixed(sp_Code) {
		panic("can only define the code once")
	}
	if b.isFixed(sp_CodeLength) && len(b.state.Code) != len(code) {
		panic("can not set code of inconsistent length")
	}
	b.state.Code = string(code)
	b.markFixed(sp_CodeLength)
	b.markFixed(sp_Code)
}

func (b *StateBuilder) GetCode() string {
	b.fixCode()
	return b.state.Code
}

func (b *StateBuilder) fixCode() {
	if b.isFixed(sp_Code) {
		return
	}
	b.fixCodeLength(0) // implicitly randomizes content
	b.markFixed(sp_Code)
}

func (b *StateBuilder) SetOpCode(pos uint16, op OpCode) {
	// TODO: support defining more than one OpCode
	if b.isFixed(sp_Code) {
		panic("can only define the code once")
	}
	b.fixCodeLength(pos + 1)
	if pos >= uint16(len(b.state.Code)) {
		return
	}
	code := []byte(b.state.Code)
	code[pos] = byte(op)
	b.state.Code = string(code)
	b.markFixed(sp_Code)
}

func (b *StateBuilder) GetOpCode(pos uint16) OpCode {
	b.fixCode()
	if pos >= uint16(len(b.state.Code)) {
		return STOP
	}
	return OpCode(b.state.Code[pos])
}

func (b *StateBuilder) SetPc(pc uint16) {
	if b.isFixed(sp_Pc) {
		panic("cannot only define PC once")
	}
	b.markFixed(sp_Pc)
	b.state.Pc = pc
}

func (b *StateBuilder) GetPc() uint16 {
	b.fixPc(false)
	return b.state.Pc
}

func (b *StateBuilder) fixPc(allowInvalid bool) {
	minLength := uint16(0)
	if !allowInvalid {
		minLength = 1
	}
	b.fixCodeLength(minLength)
	if b.isFixed(sp_Pc) {
		return
	}
	// give it a 1% chance to be an out-of-bound PC
	if allowInvalid && b.random.Int31n(100) == 0 {
		pos := uint16(b.random.Uint32())
		if pos < uint16(len(b.state.Code)) {
			pos = uint16(len(b.state.Code))
		}
		b.SetPc(pos)
	} else {
		b.SetPc(uint16(b.random.Int31n(int32(len(b.state.Code)))))
	}
}

func (b *StateBuilder) SetGas(gas uint64) {
	if b.isFixed(sp_Gas) {
		panic("cannot only define gas once")
	}
	b.markFixed(sp_Gas)
	b.state.Gas = gas
}

func (b *StateBuilder) GetGas() uint64 {
	b.fixGas()
	return b.state.Gas
}

func (b *StateBuilder) fixGas() {
	if b.isFixed(sp_Gas) {
		return
	}
	b.SetGas(b.random.Uint64())
}

func (b *StateBuilder) Build() State {
	// Fix everything that is not yet fixed.
	b.fixCodeLength(0)
	b.fixPc(true)
	b.fixGas()
	return b.state
}

func (b *StateBuilder) isFixed(property stateProperty) bool {
	return b.fixed&(1<<int(property)) != 0
}

func (b *StateBuilder) markFixed(property stateProperty) {
	b.fixed = b.fixed | (1 << int(property))
}
