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

import "fmt"

// stackUsage defines the combined effect of an instruction on the stack. Each
// instruction is accessing a range of elements on the stack relative to the
// stack pointer. The range is given by the interval [from, to) where from is
// the lower end and to is the upper end of the accessed interval. The delta
// field represents the change in the stack size caused by the instruction.
type stackUsage struct {
	from, to, delta int
}

// computeStackUsage computes the stack usage of the given opcode. The result
// is a stackUsage struct that defines the combined effect of the instruction
// on the stack. If the opcode is not executable (i.e NOOP, INVALID), an error
// is returned.
func computeStackUsage(op OpCode) (stackUsage, error) {

	// For single instructions it is easiest to define the stack usage based on
	// the opcode's pops and pushes.
	makeUsage := func(pops, pushes int) stackUsage {
		delta := pushes - pops
		to := 0
		if delta > 0 {
			to = delta
		}
		return stackUsage{from: -pops, to: to, delta: delta}
	}

	if PUSH1 <= op && op <= PUSH32 {
		return makeUsage(0, 1), nil
	}
	if DUP1 <= op && op <= DUP16 {
		return makeUsage(int(op-DUP1+1), int(op-DUP1+2)), nil
	}
	if SWAP1 <= op && op <= SWAP16 {
		return makeUsage(int(op-SWAP1+2), int(op-SWAP1+2)), nil
	}
	if LOG0 <= op && op <= LOG4 {
		return makeUsage(int(op-LOG0+2), 0), nil
	}

	switch op {
	case JUMPDEST, JUMP_TO, STOP:
		return makeUsage(0, 0), nil
	case PUSH0, MSIZE, ADDRESS, ORIGIN, CALLER, CALLVALUE, CALLDATASIZE,
		CODESIZE, GASPRICE, COINBASE, TIMESTAMP, NUMBER,
		PREVRANDAO, GASLIMIT, PC, GAS, RETURNDATASIZE,
		SELFBALANCE, CHAINID, BASEFEE, BLOBBASEFEE:
		return makeUsage(0, 1), nil
	case POP, JUMP, SELFDESTRUCT:
		return makeUsage(1, 0), nil
	case ISZERO, NOT, BALANCE, CALLDATALOAD, EXTCODESIZE,
		BLOCKHASH, MLOAD, SLOAD, TLOAD, EXTCODEHASH, BLOBHASH:
		return makeUsage(1, 1), nil
	case MSTORE, MSTORE8, SSTORE, TSTORE, JUMPI, RETURN, REVERT:
		return makeUsage(2, 0), nil
	case ADD, SUB, MUL, DIV, SDIV, MOD, SMOD, EXP, SIGNEXTEND,
		SHA3, LT, GT, SLT, SGT, EQ, AND, XOR, OR, BYTE,
		SHL, SHR, SAR:
		return makeUsage(2, 1), nil
	case CALLDATACOPY, CODECOPY, RETURNDATACOPY, MCOPY:
		return makeUsage(3, 0), nil
	case ADDMOD, MULMOD, CREATE:
		return makeUsage(3, 1), nil
	case EXTCODECOPY:
		return makeUsage(4, 0), nil
	case CREATE2:
		return makeUsage(4, 1), nil
	case STATICCALL, DELEGATECALL:
		return makeUsage(6, 1), nil
	case CALL, CALLCODE:
		return makeUsage(7, 1), nil
	}

	// For super-instructions, we need to decompose the instruction into its
	// sub-instructions and compute the combined stack usage.
	if op.isSuperInstruction() {
		usages := []stackUsage{}
		for _, subOp := range op.decompose() {
			usage, err := computeStackUsage(subOp)
			if err != nil {
				return stackUsage{}, err
			}
			usages = append(usages, usage)
		}
		return combineStackUsage(usages...), nil
	}

	return stackUsage{}, fmt.Errorf("unsupported opcode: %v", op)
}

// combineStackUsage combines the given stack usages into a single stack usage.
func combineStackUsage(usages ...stackUsage) stackUsage {
	// This function simulates the effect of the given stack usages on the stack
	// step by step. The delta of the resulting stack usage tracks the current
	// stack height offset.
	res := stackUsage{}
	for _, usage := range usages {
		from := usage.from + res.delta
		to := usage.to + res.delta

		if from < res.from {
			res.from = from
		}
		if to > res.to {
			res.to = to
		}
		res.delta += usage.delta
	}
	return res
}
