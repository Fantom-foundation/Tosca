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

// Instruction stack boundries for execution
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
