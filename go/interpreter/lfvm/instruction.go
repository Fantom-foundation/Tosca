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
	"bytes"
	"fmt"
)

// The encoding of each instruction for the MACRO EVM
type Instruction struct {
	// The op-code of this instruction.
	opcode OpCode
	// An argument value for this instruction.
	arg uint16
}

// NewInstruction returns a new instruction with the given opcode and argument.
func NewInstruction(op OpCode, arg uint16) Instruction {
	return Instruction{opcode: op, arg: arg}
}

// Instruction stack boundaries for execution
type InstructionStack struct {
	// Minimum stack height because of pop or peek operations
	stackMin int
	// Maximal stack hight on which this instruction can be executed
	// and not overflow
	stackMax int
	// Increase of stack after instruction execution
	increase int
}

// Code for the macro EVM is a slice of instructions
type Code []Instruction

func (c Code) IsIndexOp(index int, op OpCode) bool {
	return c[index].opcode == op
}

func (c Code) GetArgOf(index int) uint16 {
	return c[index].arg
}

func (i Instruction) String() string {
	if i.opcode.HasArgument() {
		return fmt.Sprintf("%v 0x%04x", i.opcode, i.arg)
	}
	return i.opcode.String()
}

func (c Code) String() string {
	var buffer bytes.Buffer
	for i, instruction := range c {
		buffer.WriteString(fmt.Sprintf("0x%04x: %v\n", i, instruction))
	}
	return buffer.String()
}
