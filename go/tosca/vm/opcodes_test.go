// Copyright (c) 2024 Fantom Foundation
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at fantom.foundation/bsl11.
//
// Change Date: 2028-4-16
//
// On the date above, in accordance with the Business Source License, use of
// this software will be governed by the GNU Lesser General Public License v3.

package vm

import (
	"regexp"
	"slices"
	"testing"
)

func TestOpCode_ValidOpCodes(t *testing.T) {
	noPrettyPrint := regexp.MustCompile(`^OpCode\([0-9]*\)$`)
	for i := 0; i < 256; i++ {
		op := OpCode(i)

		want := !noPrettyPrint.MatchString(op.String())
		if op == INVALID {
			want = false
		}
		got := IsValid(op)
		if want != got {
			t.Errorf("invalid classification of instruction %v, wanted %t, got %t", op, want, got)
		}
	}
}

func TestOpCode_ValidOpCodesNoPush(t *testing.T) {
	validOps := ValidOpCodesNoPush()

	noPrettyPrint := regexp.MustCompile(`^OpCode\([0-9]*\)$`)
	for i := 0; i < 256; i++ {
		op := OpCode(i)

		shouldBePresent := !noPrettyPrint.MatchString(op.String())
		if op == INVALID {
			shouldBePresent = false
		} else if PUSH0 <= op && op <= PUSH32 {
			shouldBePresent = false
		}

		if present := slices.Contains(validOps, op); present && !shouldBePresent {
			t.Errorf("%v should not be in ValidOpCodesNoPush", op)
		} else if !present && shouldBePresent {
			t.Errorf("%v should be in ValidOpCodesNoPush", op)
		}
	}
}

func TestOpCode_CanBePrinted(t *testing.T) {
	validName := regexp.MustCompile(`^OpCode\([0-9]*\)|([A-Z0-9]+)$`)
	for i := 0; i < 256; i++ {
		op := OpCode(i)
		if !validName.MatchString(op.String()) {
			t.Errorf("Invalid print for op %v (%d)", op, i)
		}
	}
}

func TestOpCode_NumberOfOpCodes(t *testing.T) {
	currentOpCodes := []OpCode{
		STOP, ADD, MUL, SUB, DIV, SDIV, MOD, SMOD, ADDMOD, MULMOD, EXP, SIGNEXTEND,
		LT, GT, SLT, SGT, EQ, ISZERO, AND, OR, XOR, NOT, BYTE, SHL, SHR, SAR,
		SHA3,
		ADDRESS, BALANCE, ORIGIN, CALLER, CALLVALUE, CALLDATALOAD, CALLDATASIZE, CALLDATACOPY, CODESIZE, CODECOPY, GASPRICE, EXTCODESIZE, EXTCODECOPY, RETURNDATASIZE, RETURNDATACOPY, EXTCODEHASH,
		BLOCKHASH, COINBASE, TIMESTAMP, NUMBER, PREVRANDAO, GASLIMIT, CHAINID, SELFBALANCE,
		POP, MLOAD, MSTORE, MSTORE8, SLOAD, SSTORE, JUMP, JUMPI, PC, MSIZE, GAS, JUMPDEST,
		PUSH1, PUSH2, PUSH3, PUSH4, PUSH5, PUSH6, PUSH7, PUSH8, PUSH9, PUSH10, PUSH11, PUSH12, PUSH13, PUSH14, PUSH15, PUSH16, PUSH17, PUSH18, PUSH19, PUSH20, PUSH21, PUSH22, PUSH23, PUSH24, PUSH25, PUSH26, PUSH27, PUSH28, PUSH29, PUSH30, PUSH31, PUSH32,
		DUP1, DUP2, DUP3, DUP4, DUP5, DUP6, DUP7, DUP8, DUP9, DUP10, DUP11, DUP12, DUP13, DUP14, DUP15, DUP16,
		SWAP1, SWAP2, SWAP3, SWAP4, SWAP5, SWAP6, SWAP7, SWAP8, SWAP9, SWAP10, SWAP11, SWAP12, SWAP13, SWAP14, SWAP15, SWAP16,
		LOG0, LOG1, LOG2, LOG3, LOG4,
		CREATE, CALL, CALLCODE, RETURN, DELEGATECALL, CREATE2, STATICCALL, REVERT, INVALID, SELFDESTRUCT,
		BASEFEE,                                     // London
		PUSH0,                                       // Shanghai
		BLOBHASH, BLOBBASEFEE, TLOAD, TSTORE, MCOPY, // Cancun
	}

	for i := 0; i < 256; i++ {
		op := OpCode(i)
		if slices.Contains(currentOpCodes, op) && (!IsValid(op) && op != INVALID) {
			t.Errorf("Missing OpCode %v", op)
		}
	}

}
