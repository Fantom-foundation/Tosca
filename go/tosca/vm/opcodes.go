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
	"strings"
)

//go:generate stringer -type=OpCode

type OpCode byte

const (
	STOP           OpCode = 0x00
	ADD            OpCode = 0x01
	MUL            OpCode = 0x02
	SUB            OpCode = 0x03
	DIV            OpCode = 0x04
	SDIV           OpCode = 0x05
	MOD            OpCode = 0x06
	SMOD           OpCode = 0x07
	ADDMOD         OpCode = 0x08
	MULMOD         OpCode = 0x09
	EXP            OpCode = 0x0A
	SIGNEXTEND     OpCode = 0x0B
	LT             OpCode = 0x10
	GT             OpCode = 0x11
	SLT            OpCode = 0x12
	SGT            OpCode = 0x13
	EQ             OpCode = 0x14
	ISZERO         OpCode = 0x15
	AND            OpCode = 0x16
	OR             OpCode = 0x17
	XOR            OpCode = 0x18
	NOT            OpCode = 0x19
	BYTE           OpCode = 0x1A
	SHL            OpCode = 0x1B
	SHR            OpCode = 0x1C
	SAR            OpCode = 0x1D
	SHA3           OpCode = 0x20
	ADDRESS        OpCode = 0x30
	BALANCE        OpCode = 0x31
	ORIGIN         OpCode = 0x32
	CALLER         OpCode = 0x33
	CALLVALUE      OpCode = 0x34
	CALLDATALOAD   OpCode = 0x35
	CALLDATASIZE   OpCode = 0x36
	CALLDATACOPY   OpCode = 0x37
	CODESIZE       OpCode = 0x38
	CODECOPY       OpCode = 0x39
	GASPRICE       OpCode = 0x3A
	EXTCODESIZE    OpCode = 0x3B
	EXTCODECOPY    OpCode = 0x3C
	RETURNDATASIZE OpCode = 0x3D
	RETURNDATACOPY OpCode = 0x3E
	EXTCODEHASH    OpCode = 0x3F
	BLOCKHASH      OpCode = 0x40
	COINBASE       OpCode = 0x41
	TIMESTAMP      OpCode = 0x42
	NUMBER         OpCode = 0x43
	PREVRANDAO     OpCode = 0x44
	GASLIMIT       OpCode = 0x45
	CHAINID        OpCode = 0x46
	SELFBALANCE    OpCode = 0x47
	BASEFEE        OpCode = 0x48
	BLOBHASH       OpCode = 0x49
	BLOBBASEFEE    OpCode = 0x4A
	POP            OpCode = 0x50
	MLOAD          OpCode = 0x51
	MSTORE         OpCode = 0x52
	MSTORE8        OpCode = 0x53
	SLOAD          OpCode = 0x54
	SSTORE         OpCode = 0x55
	JUMP           OpCode = 0x56
	JUMPI          OpCode = 0x57
	PC             OpCode = 0x58
	MSIZE          OpCode = 0x59
	GAS            OpCode = 0x5A
	JUMPDEST       OpCode = 0x5B
	TLOAD          OpCode = 0x5C
	TSTORE         OpCode = 0x5D
	PUSH0          OpCode = 0x5F
	MCOPY          OpCode = 0x5E
	PUSH1          OpCode = 0x60
	PUSH2          OpCode = 0x61
	PUSH3          OpCode = 0x62
	PUSH4          OpCode = 0x63
	PUSH5          OpCode = 0x64
	PUSH6          OpCode = 0x65
	PUSH7          OpCode = 0x66
	PUSH8          OpCode = 0x67
	PUSH9          OpCode = 0x68
	PUSH10         OpCode = 0x69
	PUSH11         OpCode = 0x6A
	PUSH12         OpCode = 0x6B
	PUSH13         OpCode = 0x6C
	PUSH14         OpCode = 0x6D
	PUSH15         OpCode = 0x6E
	PUSH16         OpCode = 0x6F
	PUSH17         OpCode = 0x70
	PUSH18         OpCode = 0x71
	PUSH19         OpCode = 0x72
	PUSH20         OpCode = 0x73
	PUSH21         OpCode = 0x74
	PUSH22         OpCode = 0x75
	PUSH23         OpCode = 0x76
	PUSH24         OpCode = 0x77
	PUSH25         OpCode = 0x78
	PUSH26         OpCode = 0x79
	PUSH27         OpCode = 0x7A
	PUSH28         OpCode = 0x7B
	PUSH29         OpCode = 0x7C
	PUSH30         OpCode = 0x7D
	PUSH31         OpCode = 0x7E
	PUSH32         OpCode = 0x7F
	DUP1           OpCode = 0x80
	DUP2           OpCode = 0x81
	DUP3           OpCode = 0x82
	DUP4           OpCode = 0x83
	DUP5           OpCode = 0x84
	DUP6           OpCode = 0x85
	DUP7           OpCode = 0x86
	DUP8           OpCode = 0x87
	DUP9           OpCode = 0x88
	DUP10          OpCode = 0x89
	DUP11          OpCode = 0x8A
	DUP12          OpCode = 0x8B
	DUP13          OpCode = 0x8C
	DUP14          OpCode = 0x8D
	DUP15          OpCode = 0x8E
	DUP16          OpCode = 0x8F
	SWAP1          OpCode = 0x90
	SWAP2          OpCode = 0x91
	SWAP3          OpCode = 0x92
	SWAP4          OpCode = 0x93
	SWAP5          OpCode = 0x94
	SWAP6          OpCode = 0x95
	SWAP7          OpCode = 0x96
	SWAP8          OpCode = 0x97
	SWAP9          OpCode = 0x98
	SWAP10         OpCode = 0x99
	SWAP11         OpCode = 0x9A
	SWAP12         OpCode = 0x9B
	SWAP13         OpCode = 0x9C
	SWAP14         OpCode = 0x9D
	SWAP15         OpCode = 0x9E
	SWAP16         OpCode = 0x9F
	LOG0           OpCode = 0xA0
	LOG1           OpCode = 0xA1
	LOG2           OpCode = 0xA2
	LOG3           OpCode = 0xA3
	LOG4           OpCode = 0xA4
	CREATE         OpCode = 0xF0
	CALL           OpCode = 0xF1
	CALLCODE       OpCode = 0xF2
	RETURN         OpCode = 0xF3
	DELEGATECALL   OpCode = 0xF4
	CREATE2        OpCode = 0xF5
	STATICCALL     OpCode = 0xFA
	REVERT         OpCode = 0xFD
	INVALID        OpCode = 0xFE
	SELFDESTRUCT   OpCode = 0xFF
)

func (op OpCode) Width() int {
	if PUSH1 <= op && op <= PUSH32 {
		return int(op-PUSH1) + 2
	} else {
		return 1
	}
}

// IsValid determines whether the given OpCode is a valid operation
// for any revision.
func IsValid(op OpCode) bool {
	if op == INVALID {
		return false
	}
	// We use the fact that all valid instructions have a non-generic
	// print output.
	return !strings.HasPrefix(op.String(), "OpCode(")
}

// OpCodesNoPush returns a slice of valid op codes, but no PUSH instruction.
func ValidOpCodesNoPush() []OpCode {
	res := make([]OpCode, 0, 256)
	for i := 0; i < 256; i++ {
		op := OpCode(i)
		if PUSH0 <= op && op <= PUSH32 {
			continue
		}
		if IsValid(op) {
			res = append(res, op)
		}
	}
	return res
}
