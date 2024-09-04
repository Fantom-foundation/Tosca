// Copyright (c) 2024 Fantom Foundation
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at fantom.foundation/bsl11.
//
// Change Date: 2028-4-16
//
// On the date above, in accordance with the Business Source License, use of
// this software will be governed by the GNU Lesser General Public License v3.

package lfvm

import (
	"fmt"
	"strings"
	"sync"

	"github.com/holiman/uint256"
)

const maxStackSize = 1024 // Maximum size of VM stack allowed.

// stack is the 1024-element 256-bit word-wide stack used by the VM.
// It is a fixed-size stack to prevent memory reallocation during execution.
// Check boundaries are not checked. Users of the stack must prevent over- and
// underflow situations.
//
// Each stack consumes 1024 * 32 bytes = 32KB of memory. Thus, creating and
// destroying stacks could incur significant overhead. To mitigate this, a
// stack pool is provided to reuse stack instances. To obtain an empty stack
// from the pool, use NewStack(). To return a stack to the pool, use
// ReturnStack(s).
//
// Example usage:
//
//	s := NewStack()
//	defer ReturnStack(s)
//	<use the stack in your local scope>
//
// The stack is not thread-safe. NewStack() and ReturnStack() are thread-safe.
type stack struct {
	data         [maxStackSize]uint256.Int
	stackPointer int
}

// push adds a copy of the given value to the top of the stack.
func (s *stack) push(d *uint256.Int) {
	s.data[s.stackPointer] = *d
	s.stackPointer++
}

// push adds a value with an undefined value to the top of the stack and returns
// a pointer to this element. Use this function if the element on the top stack
// should be modified directly using the returned pointer.
func (s *stack) pushUndefined() *uint256.Int {
	s.stackPointer++
	return &s.data[s.stackPointer-1]
}

// pop removes the top element from the stack and returns a pointer to it. The
// obtained pointer is only valid until the next push operation. The pointer
// can be used to obtain the popped element without the need to copy it.
func (s *stack) pop() *uint256.Int {
	s.stackPointer--
	return &s.data[s.stackPointer]
}

// peek returns a pointer to the top element of the stack without removing it.
// The returned pointer is only valid until the next operation on the stack.
func (s *stack) peek() *uint256.Int {
	return &s.data[s.len()-1]
}

// peekN returns a pointer to the n-th element from the top of the stack without
// removing it. The top element is at index 0 Thus, peekN(0) is equivalent to
// peek().
func (s *stack) peekN(n int) *uint256.Int {
	return &s.data[s.len()-n-1]
}

// len returns the number of elements on the stack.
func (s *stack) len() int {
	return s.stackPointer
}

// swap exchanges the top element with the n-th element from the top. The top
// element is at index 0. Thus, swap(0) is a no-op.
func (s *stack) swap(n int) {
	s.data[s.len()-n-1], s.data[s.len()-1] = s.data[s.len()-1], s.data[s.len()-n-1]
}

// dup duplicates the n-th element from the top and pushes it to the top of the
// stack. The top element is at index 0. Thus, dup(0) duplicates the top element.
func (s *stack) dup(n int) {
	s.data[s.stackPointer] = s.data[s.stackPointer-n-1]
	s.stackPointer++
}

// get returns the element at the given index. The bottom element is at index 0.
func (s *stack) get(i int) *uint256.Int {
	return &s.data[i]
}

func (s *stack) String() string {
	toHex := func(z *uint256.Int) string {
		b := strings.Builder{}
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

	b := strings.Builder{}
	for i := 0; i < s.len(); i++ {
		b.WriteString(fmt.Sprintf("    [%4d] %v\n", s.len()-i-1, toHex(s.peekN(i))))
	}
	return b.String()
}

// ------------------ Stack Pool ------------------

var stackPool = sync.Pool{
	New: func() interface{} {
		return &stack{}
	},
}

// NewStack returns a new stack instance from the a reuse pool. Heavy stack
// users should use this function to prevent memory reallocation overhead.
// This function is thread-safe.
func NewStack() *stack {
	return stackPool.Get().(*stack)
}

// ReturnStack returns the stack to the reuse pool. Any stack may only be
// returned once to avoid concurrent re-use. This is not checked internally.
// This function is thread-safe.
func ReturnStack(s *stack) {
	s.stackPointer = 0
	stackPool.Put(s)
}

// ------------------ Stack Boundary ------------------

var staticStackBoundary = [NUM_OPCODES]InstructionStack{}

func init() {
	for i := 0; i < int(NUM_OPCODES); i++ {
		staticStackBoundary[OpCode(i)] = getStaticStackInternal(OpCode(i))
	}
}

// min is number of pop and max is pop - push
func newInstructionStack(min, max, _increase int) InstructionStack {
	return InstructionStack{
		stackMin: min,
		stackMax: maxStackSize - max,
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
		return newInstructionStack(int(op)-int(SWAP1)+2, 0, 0)
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
		BLOCKHASH, MLOAD, SLOAD, TLOAD, EXTCODEHASH, BLOBHASH,
		PUSH1_SHL:
		return newInstructionStack(1, 0, 1)
	case PUSH0, MSIZE, ADDRESS, ORIGIN, CALLER, CALLVALUE, CALLDATASIZE,
		CODESIZE, GASPRICE, COINBASE, TIMESTAMP, NUMBER,
		PREVRANDAO, GASLIMIT, PC, GAS, RETURNDATASIZE,
		SELFBALANCE, CHAINID, BASEFEE, BLOBBASEFEE,
		PUSH1_PUSH1_PUSH1_SHL_SUB:
		return newInstructionStack(0, 1, 1)
	case POP, JUMP, SELFDESTRUCT,
		SWAP2_POP, PUSH1_ADD, PUSH2_JUMPI,
		ISZERO_PUSH2_JUMPI:
		return newInstructionStack(1, 0, 0)
	case MSTORE, MSTORE8, SSTORE, TSTORE, JUMPI, RETURN, REVERT,
		POP_POP, POP_JUMP:
		return newInstructionStack(2, 0, 0)
	case CALLDATACOPY, CODECOPY, RETURNDATACOPY, MCOPY:
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
