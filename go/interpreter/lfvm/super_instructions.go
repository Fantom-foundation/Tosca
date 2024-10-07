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
	"math/bits"

	"github.com/holiman/uint256"
)

// ----------------------------- Super Instructions -----------------------------

func opSwap1_Pop(c *context) {
	a1 := c.stack.pop()
	a2 := c.stack.peek()
	*a2 = *a1
}

func opSwap2_Pop(c *context) {
	a1 := c.stack.pop()
	*c.stack.peekN(1) = *a1
}

func opPush1_Push1(c *context) {
	arg := c.code[c.pc].arg
	c.stack.stackPointer += 2
	c.stack.peekN(0).SetUint64(uint64(arg & 0xFF))
	c.stack.peekN(1).SetUint64(uint64(arg >> 8))
}

func opPush1_Add(c *context) {
	arg := c.code[c.pc].arg
	trg := c.stack.peek()
	var carry uint64
	trg[0], carry = bits.Add64(trg[0], uint64(arg), 0)
	trg[1], carry = bits.Add64(trg[1], 0, carry)
	trg[2], carry = bits.Add64(trg[2], 0, carry)
	trg[3], _ = bits.Add64(trg[3], 0, carry)
}

func opPush1_Shl(c *context) {
	arg := c.code[c.pc].arg
	trg := c.stack.peek()
	trg.Lsh(trg, uint(arg))
}

func opPush1_Dup1(c *context) {
	arg := c.code[c.pc].arg
	c.stack.stackPointer += 2
	c.stack.peekN(0).SetUint64(uint64(arg))
	c.stack.peekN(1).SetUint64(uint64(arg))
}

func opPush2_Jump(c *context) error {
	// Directly take pushed value and jump to destination.
	c.pc = int32(c.code[c.pc].arg) - 1
	return checkJumpDest(c)
}

func opPush2_Jumpi(c *context) error {
	// Directly take pushed value and jump to destination.
	condition := c.stack.pop()
	if !condition.IsZero() {
		c.pc = int32(c.code[c.pc].arg) - 1
		return checkJumpDest(c)
	}
	return nil
}

func opSwap2_Swap1(c *context) {
	a1 := c.stack.peekN(0)
	a2 := c.stack.peekN(1)
	a3 := c.stack.peekN(2)
	*a1, *a2, *a3 = *a2, *a3, *a1
}

func opDup2_Mstore(c *context) error {
	var value = c.stack.pop()
	var addr = c.stack.peek()
	v := value.Bytes32()
	return c.memory.set(addr, v[:], c)
}

func opDup2_Lt(c *context) {
	b := c.stack.peekN(0)
	a := c.stack.peekN(1)
	if a.Lt(b) {
		b.SetOne()
	} else {
		b.Clear()
	}
}

func opPopPop(c *context) {
	c.stack.stackPointer -= 2
}

func opPop_Jump(c *context) error {
	opPop(c)
	return opJump(c)
}

func opIsZero_Push2_Jumpi(c *context) error {
	condition := c.stack.pop()
	if condition.IsZero() {
		c.pc = int32(c.code[c.pc].arg) - 1
		return checkJumpDest(c)
	}
	return nil
}

func opSwap2_Swap1_Pop_Jump(c *context) error {
	top := c.stack.pop()
	c.stack.pop()
	trg := c.stack.peek()
	c.pc = int32(trg.Uint64()) - 1
	*trg = *top
	return checkJumpDest(c)
}

func opSwap1_Pop_Swap2_Swap1(c *context) {
	a1 := c.stack.pop()
	a2 := c.stack.peekN(0)
	a3 := c.stack.peekN(1)
	a4 := c.stack.peekN(2)
	*a2, *a3, *a4 = *a3, *a4, *a1
}

func opPop_Swap2_Swap1_Pop(c *context) {
	c.stack.pop()
	a2 := c.stack.pop()
	a3 := c.stack.peekN(0)
	a4 := c.stack.peekN(1)
	*a3, *a4 = *a4, *a2
}

func opPush1_Push4_Dup3(c *context) {
	opPush1(c)
	c.pc++
	opPush4(c)
	opDup(c, 3)
}

func opAnd_Swap1_Pop_Swap2_Swap1(c *context) {
	opAnd(c)
	opSwap1_Pop_Swap2_Swap1(c)
}

func opPush1_Push1_Push1_Shl_Sub(c *context) {
	arg1 := c.code[c.pc].arg
	arg2 := c.code[c.pc+1].arg
	shift := uint8(arg2)
	value := uint8(arg1 & 0xFF)
	delta := uint8(arg1 >> 8)
	trg := c.stack.pushUndefined()
	trg.SetUint64(uint64(value))
	trg.Lsh(trg, uint(shift))
	trg.Sub(trg, uint256.NewInt(uint64(delta)))
	c.pc++
}
