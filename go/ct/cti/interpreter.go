package cti

import (
	"math"

	"github.com/holiman/uint256"
)

type OpCode byte

const (
	STOP     OpCode = 0
	ADD      OpCode = 0x01
	LT       OpCode = 0x10
	EQ       OpCode = 0x14
	AND      OpCode = 0x16
	OR       OpCode = 0x17
	NOT      OpCode = 0x19
	JUMP     OpCode = 0x56
	JUMPI    OpCode = 0x57
	JUMPDEST OpCode = 0x5B
	POP      OpCode = 0x50
	MLOAD    OpCode = 0x51
	MSTORE   OpCode = 0x52
	MSTORE8  OpCode = 0x53
	PUSH1    OpCode = 0x60
	PUSH2    OpCode = 0x61
	PUSH16   OpCode = 0x6F
	PUSH32   OpCode = 0x7F
	DUP1     OpCode = 0x80
	DUP2     OpCode = 0x81
	DUP16    OpCode = 0x8F
	SWAP1    OpCode = 0x90
	SWAP2    OpCode = 0x91
	SWAP16   OpCode = 0x9F
)

type Status byte

const (
	Running             Status = 0
	Done                Status = 1
	Return              Status = 2
	Revert              Status = 3
	Invalid             Status = 4
	ErrorGas            Status = 5
	ErrorStackUnderflow Status = 6
	ErrorStackOverflow  Status = 7
	ErrorJump           Status = 8
)

type State struct {
	Status  Status
	Pc      int
	GasLeft uint64
	Code    []OpCode
	Stack   []uint256.Int
	Memory  []byte
}

const MaxStackLength = 1024

func (s *State) Run() {
	for s.Status == Running {
		s.Step()
	}
}

func (s *State) StepN(n int) {
	for i := 0; i < n && s.Status == Running; i++ {
		s.Step()
	}
}

func (s *State) Step() {
	if s.Status != Running {
		return
	}

	if s.Pc >= len(s.Code) {
		s.Status = Done
		return
	}

	switch s.Code[s.Pc] {
	case STOP:
		s.Status = Done
	case ADD:
		s.opADD()
	case LT:
		s.opLT()
	case EQ:
		s.opEQ()
		/*
			case AND:
				s.opAND()
			case OR:
				s.opOR()
			case NOT:
				s.opNOT()
		*/
	case JUMP:
		s.opJUMP()
	case JUMPI:
		s.opJUMPI()
	case JUMPDEST:
		s.opJUMPDEST()
	case POP:
		s.opPOP()
	case MLOAD:
		s.opMLOAD()
	case MSTORE:
		s.opMSTORE()
	case MSTORE8:
		s.opMSTORE8()
	case PUSH1:
		s.opPUSH(1)
	case PUSH2:
		s.opPUSH(2)
	case PUSH16:
		s.opPUSH(16)
	case PUSH32:
		s.opPUSH(32)
		/*
			case DUP1:
				s.opDUP(1)
			case DUP2:
				s.opDUP(2)
			case DUP16:
				s.opDUP(16)
			case SWAP1:
				s.opSWAP(1)
			case SWAP2:
				s.opSWAP(2)
			case SWAP16:
				s.opSWAP(16)
		*/
	default:
		s.Status = Invalid
	}
}

func (s *State) applyGasCost(cost uint64) bool {
	if cost > s.GasLeft {
		s.Status = ErrorGas
		return false
	}
	s.GasLeft -= cost
	return true
}

func (s *State) pushStack(i *uint256.Int) {
	s.Stack = append(s.Stack, *i)
}

func (s *State) popStack() uint256.Int {
	i := s.Stack[len(s.Stack)-1]
	s.Stack = s.Stack[:len(s.Stack)-1]
	return i
}

func (s *State) peekStack() *uint256.Int {
	return &s.Stack[len(s.Stack)-1]
}

func (s *State) memoryExpansionCost(offset_u256 uint256.Int, size_u256 uint256.Int) (memCost uint64, offset uint64, size uint64) {
	if offset_u256.GtUint64(math.MaxUint64) || size_u256.GtUint64(math.MaxUint64) {
		return math.MaxUint64, 0, 0
	}

	offset = offset_u256.Uint64()
	size = size_u256.Uint64()

	if size == 0 {
		memCost = 0
		return
	}

	newSize := offset + size
	if newSize <= uint64(len(s.Memory)) {
		memCost = 0
		return
	}

	calcMemoryCost := func(size uint64) uint64 {
		memorySizeWord := (size + 31) / 32
		return (memorySizeWord*memorySizeWord)/512 + (3 * memorySizeWord)
	}
	memCost = calcMemoryCost(newSize) - calcMemoryCost(uint64(len(s.Memory)))
	return
}

func (s *State) readFromMemory(offset uint64, size uint64) []byte {
	s.growMemory(offset, size)
	data := make([]byte, size)
	copy(data, s.Memory[offset:])
	return data
}

func (s *State) writeToMemory(data []byte, offset uint64) {
	s.growMemory(offset, uint64(len(data)))
	copy(s.Memory[offset:], data)
}

func (s *State) growMemory(offset uint64, size uint64) {
	if size != 0 {
		newSize := offset + size
		if newSize > uint64(len(s.Memory)) {
			newSize = ((newSize + 31) / 32) * 32
			s.Memory = append(s.Memory, make([]byte, newSize-uint64(len(s.Memory)))...)
		}
	}
}

func (s *State) checkJumpDest(target uint256.Int) bool {
	if !target.LtUint64(math.MaxInt32) {
		return false
	}
	target_i32 := int(target.Uint64())
	if target_i32 < len(s.Code) {
		for i := 0; i < len(s.Code); i++ {
			instruction := s.Code[i]
			if PUSH1 <= instruction && instruction <= PUSH32 {
				i += int(instruction - PUSH1 + 1) // skip push layload
			}
			if i == target_i32 {
				return instruction == JUMPDEST
			}
		}
	}
	return false
}

func (s *State) opADD() {
	if !s.applyGasCost(3) {
		return
	}
	if len(s.Stack) < 2 {
		s.Status = ErrorStackUnderflow
		return
	}

	a := s.popStack()
	b := s.peekStack()
	b.Add(&a, b)

	s.Pc += 1
}

func (s *State) opLT() {
	if !s.applyGasCost(3) {
		return
	}
	if len(s.Stack) < 2 {
		s.Status = ErrorStackUnderflow
		return
	}

	a := s.popStack()
	b := s.peekStack()
	if a.Lt(b) {
		b.SetOne()
	} else {
		b.Clear()
	}

	s.Pc += 1
}

func (s *State) opEQ() {
	if !s.applyGasCost(3) {
		return
	}
	if len(s.Stack) < 2 {
		s.Status = ErrorStackUnderflow
		return
	}

	a := s.popStack()
	b := s.peekStack()
	if a.Eq(b) {
		b.SetOne()
	} else {
		b.Clear()
	}

	s.Pc += 1
}

func (s *State) opAND() {
	if !s.applyGasCost(3) {
		return
	}
	if len(s.Stack) < 2 {
		s.Status = ErrorStackUnderflow
		return
	}

	a := s.popStack()
	b := s.peekStack()
	b.And(&a, b)

	s.Pc += 1
}

func (s *State) opOR() {
	if !s.applyGasCost(3) {
		return
	}
	if len(s.Stack) < 2 {
		s.Status = ErrorStackUnderflow
		return
	}

	a := s.popStack()
	b := s.peekStack()
	b.Or(&a, b)

	s.Pc += 1
}

func (s *State) opNOT() {
	if !s.applyGasCost(3) {
		return
	}
	if len(s.Stack) < 1 {
		s.Status = ErrorStackUnderflow
		return
	}

	b := s.peekStack()
	b.Not(b)

	s.Pc += 1
}

func (s *State) opJUMP() {
	if !s.applyGasCost(8) {
		return
	}
	if len(s.Stack) < 1 {
		s.Status = ErrorStackUnderflow
		return
	}

	target := s.popStack()
	if !s.checkJumpDest(target) {
		s.Status = ErrorJump
		return
	}

	s.Pc = int(target.Uint64())
}

func (s *State) opJUMPI() {
	if !s.applyGasCost(10) {
		return
	}
	if len(s.Stack) < 2 {
		s.Status = ErrorStackUnderflow
		return
	}

	target := s.popStack()
	condition := s.popStack()

	if condition.IsZero() {
		s.Pc += 1
		return
	} else {
		if !s.checkJumpDest(target) {
			s.Status = ErrorJump
			return
		}

		s.Pc = int(target.Uint64())
	}
}

func (s *State) opJUMPDEST() {
	if !s.applyGasCost(1) {
		return
	}

	s.Pc += 1
}

func (s *State) opPOP() {
	if !s.applyGasCost(2) {
		return
	}
	if len(s.Stack) < 1 {
		s.Status = ErrorStackUnderflow
		return
	}

	s.popStack()

	s.Pc += 1
}

func (s *State) opMLOAD() {
	if !s.applyGasCost(3) {
		return
	}
	if len(s.Stack) < 1 {
		s.Status = ErrorStackUnderflow
		return
	}

	offset_u256 := s.popStack()

	memCost, offset, _ := s.memoryExpansionCost(offset_u256, *uint256.NewInt(32))
	if !s.applyGasCost(memCost) {
		return
	}

	var value uint256.Int
	value.SetBytes32(s.readFromMemory(offset, 32))
	s.pushStack(&value)

	s.Pc += 1
}

func (s *State) opMSTORE() {
	if !s.applyGasCost(3) {
		return
	}
	if len(s.Stack) < 2 {
		s.Status = ErrorStackUnderflow
		return
	}

	offset_u256 := s.popStack()
	value := s.popStack()

	memCost, offset, _ := s.memoryExpansionCost(offset_u256, *uint256.NewInt(32))
	if !s.applyGasCost(memCost) {
		return
	}

	bytes := value.Bytes32()
	s.writeToMemory(bytes[:], offset)

	s.Pc += 1
}

func (s *State) opMSTORE8() {
	if !s.applyGasCost(3) {
		return
	}
	if len(s.Stack) < 2 {
		s.Status = ErrorStackUnderflow
		return
	}

	offset_u256 := s.popStack()
	value_u256 := s.popStack()
	value := value_u256.Bytes32()[31]

	memCost, offset, _ := s.memoryExpansionCost(offset_u256, *uint256.NewInt(1))
	if !s.applyGasCost(memCost) {
		return
	}

	s.writeToMemory([]byte{value}, offset)

	s.Pc += 1
}

func (s *State) opPUSH(n int) {
	if !s.applyGasCost(3) {
		return
	}
	if len(s.Stack)+1 > MaxStackLength {
		s.Status = ErrorStackOverflow
		return
	}

	var value [32]byte
	for i := 0; i < n; i++ {
		if s.Pc+1+i < len(s.Code) {
			value[i] = byte(s.Code[s.Pc+1+i])
		}
	}

	z := uint256.NewInt(0)
	z.SetBytes(value[0:n])
	s.pushStack(z)

	s.Pc += 1 + n
}

func (s *State) opDUP(n int) {
	if !s.applyGasCost(3) {
		return
	}
	if len(s.Stack) < n {
		s.Status = ErrorStackUnderflow
		return
	}
	if len(s.Stack)+1 > MaxStackLength {
		s.Status = ErrorStackOverflow
		return
	}

	s.pushStack(&s.Stack[len(s.Stack)-1-n])

	s.Pc += 1
}

func (s *State) opSWAP(n int) {
	if !s.applyGasCost(3) {
		return
	}
	if len(s.Stack) < n+1 {
		s.Status = ErrorStackUnderflow
		return
	}

	a := s.Stack[len(s.Stack)-1]
	b := s.Stack[len(s.Stack)-1-n]

	s.Stack[len(s.Stack)-1] = b
	s.Stack[len(s.Stack)-1-n] = a

	s.Pc += 1
}
