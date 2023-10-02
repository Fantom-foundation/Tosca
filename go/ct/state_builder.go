package ct

import (
	"fmt"

	"github.com/holiman/uint256"
	"pgregory.net/rand"
)

func GetRandomState() State {
	return NewStateBuilder().Build()
}

func GetRandomStateWithSeed(seed uint64) State {
	return NewStateBuilderWithSeed(seed).Build()
}

// StateBuilder can be used to built a random state satisfying a set
// of constraints. Constraints
type StateBuilder struct {
	// The state under construction.
	state State

	// A set of fixed properties that can no longer be altered.
	fixed uint64

	// Code positions that are fixed and can no longer be altered.
	fixedOps map[uint16]struct{}

	random *rand.Rand
}

type stateProperty int

const (
	sp_Status stateProperty = iota
	sp_CodeLength
	sp_Code
	sp_Pc
	sp_Gas
	sp_StackSize
	sp_Stack
)

func NewStateBuilder() *StateBuilder {
	return &StateBuilder{
		random: rand.New(),
	}
}

func NewStateBuilderWithSeed(seed uint64) *StateBuilder {
	return &StateBuilder{
		random:   rand.New(seed),
		fixedOps: map[uint16]struct{}{},
	}
}

func (b *StateBuilder) Clone() *StateBuilder {
	var fixedOps map[uint16]struct{}
	if len(b.fixedOps) > 0 {
		fixedOps = map[uint16]struct{}{}
		for k := range b.fixedOps {
			fixedOps[k] = struct{}{}
		}
	}
	return &StateBuilder{
		state:    *b.state.Clone(),
		fixed:    b.fixed,
		fixedOps: fixedOps,
		random:   rand.New(),
	}
}

func (b *StateBuilder) Restore(backup *StateBuilder) {
	b.state = backup.state
	b.fixed = backup.fixed
	b.fixedOps = backup.fixedOps
	b.random = backup.random
}

// --- Status ---

func (b *StateBuilder) SetStatus(status StatusCode) {
	if b.isFixed(sp_Status) {
		panic("can only set status once")
	}
	b.state.Status = status
	b.markFixed(sp_Status)
}

func (b *StateBuilder) GetStatus() StatusCode {
	b.fixStatus()
	return b.state.Status
}

func (b *StateBuilder) fixStatus() {
	if b.isFixed(sp_Status) {
		return
	}
	b.SetStatus(StatusCode(b.random.Int31n(int32(numStatuses))))
}

// --- Code ---

func (b *StateBuilder) SetCodeLength(length uint16) {
	if b.isFixed(sp_CodeLength) {
		panic("can only define code length once")
	}

	// Create the code and fill with random content.
	code := make([]byte, length)
	b.random.Read(code)
	b.state.Code = code

	b.markFixed(sp_CodeLength)
}

func (b *StateBuilder) GetCodeLength() uint16 {
	b.fixCodeLength(0)
	return uint16(len(b.state.Code))
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

func (b *StateBuilder) SetCode(code []byte) {
	if b.isFixed(sp_Code) {
		panic("can only define the code once")
	}
	if b.isFixed(sp_CodeLength) && len(b.state.Code) != len(code) {
		panic("can not set code of inconsistent length")
	}
	b.state.Code = code
	b.markFixed(sp_CodeLength)
	b.markFixed(sp_Code)
}

func (b *StateBuilder) GetCode() []byte {
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
	if b.isFixed(sp_Code) {
		panic("can only define the code once")
	}
	if _, fixed := b.fixedOps[pos]; fixed {
		// TODO: add some warning as a return value that this operation had no effect
		fmt.Printf("WARNING: code position %d has been fixed before\n", pos)
		//panic("code position has been fixed before")
		return
	}
	b.fixCodeLength(pos + 1)
	if pos >= uint16(len(b.state.Code)) {
		return
	}
	b.state.Code[pos] = byte(op)
	if b.fixedOps == nil {
		b.fixedOps = map[uint16]struct{}{}
	}
	b.fixedOps[pos] = struct{}{}
}

func (b *StateBuilder) GetOpCode(pos uint16) OpCode {
	b.fixCode()
	if pos >= uint16(len(b.state.Code)) {
		return STOP
	}
	return OpCode(b.state.Code[pos])
}

// --- PC ---

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
		codeSize := len(b.state.Code)
		if codeSize == 0 {
			b.SetPc(0)
		} else {
			b.SetPc(uint16(b.random.Int31n(int32(codeSize))))
		}
	}
}

// --- Gas ---

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

// --- Stack ---

func (b *StateBuilder) SetStackSize(size int) {
	if b.isFixed(sp_StackSize) {
		panic("cannot only define stack size once")
	}
	b.markFixed(sp_StackSize)
	for i := 0; i < size; i++ {
		var value [32]byte
		b.random.Read(value[:])
		var element uint256.Int
		element.SetBytes(value[:])
		b.state.Stack.Push(element)
	}
}

func (b *StateBuilder) GetStackSize() int {
	b.fixStackSize()
	return b.state.Stack.Size()
}

func (b *StateBuilder) fixStackSize() {
	if b.isFixed(sp_StackSize) {
		return
	}
	b.SetStackSize(int(b.random.Int31n(1025))) // range [0,1024]
}

func (b *StateBuilder) SetStackValue(pos int, value uint256.Int) {
	b.fixStackSize()
	if pos >= b.state.Stack.Size() {
		return
	}
	b.state.Stack.Set(pos, value)
}

func (b *StateBuilder) GetStackValue(pos int) uint256.Int {
	b.fixStackSize()
	if pos >= b.state.Stack.Size() {
		return *uint256.NewInt(0)
	}
	return b.state.Stack.Get(pos)
}

// --- Build ---

func (b *StateBuilder) Build() State {
	// Fix everything that is not yet fixed.
	b.fixStatus()
	b.fixCodeLength(0)
	b.fixPc(true)
	b.fixGas()
	b.fixStackSize()
	return b.state
}

func (b *StateBuilder) isFixed(property stateProperty) bool {
	return b.fixed&(1<<int(property)) != 0
}

func (b *StateBuilder) markFixed(property stateProperty) {
	b.fixed = b.fixed | (1 << int(property))
}
