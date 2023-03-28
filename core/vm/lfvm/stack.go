package lfvm

import (
	"bytes"
	"fmt"
	"sync"

	"github.com/ethereum/go-ethereum/params"
	"github.com/holiman/uint256"
)

var staticStackBoundry = [NUM_OPCODES]InstructionStack{}

type Stack struct {
	data      [1024]uint256.Int
	stack_ptr int
}

func init() {
	for i := 0; i < int(NUM_OPCODES); i++ {
		staticStackBoundry[OpCode(i)] = getStaticStackInternal(OpCode(i))
	}
}

func (s *Stack) Data() []uint256.Int {
	return s.data[:s.stack_ptr]
}

func (s *Stack) push(d *uint256.Int) {
	s.data[s.stack_ptr] = *d
	s.stack_ptr++
}

func (s *Stack) pushEmpty() *uint256.Int {
	s.stack_ptr++
	return &s.data[s.stack_ptr-1]
}

func (s *Stack) pop() *uint256.Int {
	s.stack_ptr--
	return &s.data[s.stack_ptr]
}

func (s *Stack) len() int {
	return s.stack_ptr
}

func (s *Stack) swap(n int) {
	s.data[s.len()-n], s.data[s.len()-1] = s.data[s.len()-1], s.data[s.len()-n]
}

func (s *Stack) dup(n int) {
	s.data[s.stack_ptr] = s.data[s.stack_ptr-n]
	s.stack_ptr++
}

func (s *Stack) peek() *uint256.Int {
	return &s.data[s.len()-1]
}

func (s *Stack) Back(n int) *uint256.Int {
	return &s.data[s.len()-n-1]
}

func ToHex(z *uint256.Int) string {
	var b bytes.Buffer
	b.WriteString("0x")
	bytes := z.Bytes32()
	for i, cur := range bytes {
		b.WriteString(fmt.Sprintf("%02x", cur))
		if (i+1)%8 == 0 {
			b.WriteString(" ")
		}
	}
	return b.String()
}

func (s *Stack) String() string {
	var b bytes.Buffer
	for i := 0; i < s.len(); i++ {
		b.WriteString(fmt.Sprintf("    [%2d] %v\n", s.len()-i-1, ToHex(s.Back(i))))
	}
	return b.String()
}

// ------------------ Stack Pool ------------------

var stackPool = sync.Pool{
	New: func() interface{} {
		return &Stack{}
	},
}

func NewStack() *Stack {
	return stackPool.Get().(*Stack)
}

func ReturnStack(s *Stack) {
	s.stack_ptr = 0
	stackPool.Put(s)
}

// ------------------ Stack Boundry ------------------

// min is number of pop and max is pop - push
func newInstructionStack(min, max, _increase int) InstructionStack {
	return InstructionStack{
		stackMin: min,
		stackMax: int(params.StackLimit) - max,
		increase: _increase,
	}
}

func getStaticStackInternal(op OpCode) InstructionStack {

	if PUSH1 <= op && op <= PUSH32 {
		return newInstructionStack(0, 1, 1)
	}
	if DUP1 <= op && op <= DUP16 {
		return newInstructionStack(int(op)-int(DUP1)+1, 1, 1)
	}
	if SWAP1 <= op && op <= SWAP16 {
		return newInstructionStack(int(op)-int(SWAP1)+1, 0, 0)
	}
	if LOG0 <= op && op <= LOG4 {
		return newInstructionStack(int(op)-int(LOG0)+2, 0, 0)
	}

	switch op {
	case JUMPDEST, JUMP_TO, STOP, INVALID:
		return newInstructionStack(0, 0, 0)
	case ADD, SUB, MUL, DIV, SDIV, MOD, SMOD, EXP, SIGNEXTEND,
		SHA3, LT, GT, SLT, SGT, EQ, AND, XOR, OR, BYTE,
		SHL, SHR, SAR,
		SWAP1_POP, DUP2_MSTORE:
		return newInstructionStack(2, 0, 1)
	case ADDMOD, MULMOD, SWAP2_SWAP1_POP_JUMP:
		return newInstructionStack(3, 0, 1)
	case ISZERO, NOT, BALANCE, CALLDATALOAD, EXTCODESIZE,
		BLOCKHASH, MLOAD, SLOAD, EXTCODEHASH,
		PUSH1_SHL:
		return newInstructionStack(1, 0, 1)
	case MSIZE, ADDRESS, ORIGIN, CALLER, CALLVALUE, CALLDATASIZE,
		CODESIZE, GASPRICE, COINBASE, TIMESTAMP, NUMBER,
		DIFFICULTY, GASLIMIT, PC, GAS, RETURNDATASIZE,
		SELFBALANCE, CHAINID, BASEFEE,
		PUSH1_PUSH1_PUSH1_SHL_SUB:
		return newInstructionStack(0, 1, 1)
	case POP, JUMP, SELFDESTRUCT,
		SWAP2_POP, PUSH1_ADD, PUSH2_JUMPI,
		ISZERO_PUSH2_JUMPI:
		return newInstructionStack(1, 0, 0)
	case MSTORE, MSTORE8, SSTORE, JUMPI, RETURN, REVERT,
		POP_POP, POP_JUMP:
		return newInstructionStack(2, 0, 0)
	case CALLDATACOPY, CODECOPY, RETURNDATACOPY:
		return newInstructionStack(3, 0, 0)
	case EXTCODECOPY:
		return newInstructionStack(4, 0, 0)
	case CREATE:
		return newInstructionStack(3, 0, 1)
	case CREATE2:
		return newInstructionStack(4, 0, 1)
	case CALL, CALLCODE:
		return newInstructionStack(7, 0, 1)
	case STATICCALL, DELEGATECALL:
		return newInstructionStack(6, 0, 1)
	case PUSH1_DUP1, PUSH1_PUSH1:
		return newInstructionStack(0, 2, 2)
	case SWAP2_SWAP1:
		return newInstructionStack(3, 0, 3)
	case DUP2_LT:
		return newInstructionStack(2, 0, 2)
	case SWAP1_POP_SWAP2_SWAP1:
		return newInstructionStack(4, 0, 3)
	case POP_SWAP2_SWAP1_POP:
		return newInstructionStack(4, 0, 2)
	case PUSH1_PUSH4_DUP3:
		return newInstructionStack(0, 3, 3)
	case AND_SWAP1_POP_SWAP2_SWAP1:
		return newInstructionStack(5, 0, 3)
	}
	return newInstructionStack(0, 0, 0)
}
