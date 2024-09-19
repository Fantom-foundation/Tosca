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

// Instruction encodes an instruction for the long-form virtual machine (LFVM).
type Instruction struct {
	// The op-code of this instruction.
	opcode OpCode
	// An argument value for this instruction.
	arg uint16
}

// Code for the LFVM is a slice of instructions.
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
