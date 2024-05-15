//
// Copyright (c) 2024 Fantom Foundation
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at fantom.foundation/bsl11.
//
// Change Date: 2028-4-16
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by the GNU Lesser General Public Licence v3
//

package lfvm

import (
	"bytes"
	"errors"
	"fmt"
	"log"
	"slices"
	"strings"
	"sync"

	"github.com/Fantom-foundation/Tosca/go/ct/common"
	"github.com/Fantom-foundation/Tosca/go/vm"

	// This is only imported to get the EVM opcode definitions.
	// TODO: write up our own op-code definition and remove this dependency.

	evm "github.com/ethereum/go-ethereum/core/vm"
)

type cache_val struct {
	oldCode []byte
	code    Code
}

var mu = sync.Mutex{}
var cache = map[vm.Hash]cache_val{}

func clearConversionCache() {
	mu.Lock()
	defer mu.Unlock()
	cache = map[vm.Hash]cache_val{}
}

func Convert(code []byte, with_super_instructions bool, create bool, noCodeCache bool, codeHash vm.Hash) (Code, error) {
	// TODO: clean up this code; it does some checks that seems to be once used
	// for debugging an issue that do not make sense any more.

	// Do not cache use-once code in create calls.
	// In those cases the codeHash is also invalid.
	if create {
		return convert(code, with_super_instructions)
	}

	mu.Lock()
	res, exists := cache[codeHash]
	if exists && !create {
		isEqual := true
		if noCodeCache {

			if !bytes.Equal(res.oldCode, code) {
				log.Printf("Different code for hash %v", codeHash)
				isEqual = false
			}
		}

		if isEqual {
			mu.Unlock()
			return res.code, nil
		}
	}
	mu.Unlock()
	resCode, error := convert(code, with_super_instructions)
	if error != nil {
		return nil, error
	}
	if !create {
		mu.Lock()
		cache[codeHash] = cache_val{oldCode: code, code: resCode}
		mu.Unlock()
	}
	return resCode, nil
}

type codeBuilder struct {
	code    []Instruction
	nextPos int
}

func newCodeBuilder(codelength int) codeBuilder {
	return codeBuilder{make([]Instruction, codelength), 0}
}

func (b *codeBuilder) length() int {
	return b.nextPos
}

func (b *codeBuilder) appendOp(opcode OpCode, arg uint16) *codeBuilder {
	b.code[b.nextPos].opcode = opcode
	b.code[b.nextPos].arg = arg
	b.nextPos++
	return b
}

func (b *codeBuilder) appendCode(opcode OpCode) *codeBuilder {
	b.code[b.nextPos].opcode = opcode
	b.nextPos++
	return b
}

func (b *codeBuilder) appendData(data uint16) *codeBuilder {
	return b.appendOp(DATA, data)
}

func (b *codeBuilder) padNoOpsUntil(pos int) {
	for i := b.nextPos; i < pos; i++ {
		b.code[i].opcode = NOOP
	}
	b.nextPos = pos
}

func (b *codeBuilder) toCode() Code {
	return b.code[0:b.nextPos]
}

func convert(code []byte, with_super_instructions bool) (Code, error) {
	res := newCodeBuilder(len(code))

	// Convert each individual instruction.
	for i := 0; i < len(code); {
		// Handle jump destinations
		if code[i] == byte(evm.JUMPDEST) {
			if res.length() > i {
				return nil, errors.New("unable to convert code, encountered targe block larger than input")
			}
			// Jump to the next jump destination and fill space with noops
			if res.length() < i {
				res.appendOp(JUMP_TO, uint16(i))
			}
			res.padNoOpsUntil(i)
			res.appendCode(JUMPDEST)
			i++
			continue
		}

		// Convert instructions
		inc := appendInstructions(&res, i, code, with_super_instructions)
		i += inc + 1
	}
	return res.toCode(), nil
}

// PcMap is a bidirectional map to map program counters between evm <-> lfvm.
type PcMap struct {
	evmToLfvm map[uint16]uint16
	lfvmToEvm map[uint16]uint16
}

// GenPcMap creates a bidirectional program counter map for a given code,
// allowing mapping from a program counter in evm code to lfvm and vice versa.
func GenPcMap(code []byte, with_super_instructions bool) (*PcMap, error) {
	if with_super_instructions {
		return nil, errors.New("super instructions are not yet supported for program counter mapping")
	}

	pcMap := PcMap{make(map[uint16]uint16, len(code)), make(map[uint16]uint16, len(code))}
	// Entry point always maps from 0 <-> 0, even when there is no code.
	pcMap.evmToLfvm[0] = 0
	pcMap.lfvmToEvm[0] = 0
	res := newCodeBuilder(len(code))

	// Convert each individual instruction.
	for i := 0; i < len(code); {
		// Handle jump destinations.
		if code[i] == byte(evm.JUMPDEST) {
			if res.length() > i {
				return nil, errors.New("unable to convert code, encountered target block larger than input")
			}

			// All lfvm opcodes from jmpto until jmpdest, including the potential nops in between map to evm jmpdest.
			for j := res.nextPos; j <= i; j++ {
				pcMap.lfvmToEvm[uint16(j)] = uint16(i)
			}

			// Jump to the next jump destination and fill space with noops.
			if res.length() < i {
				res.appendOp(JUMP_TO, uint16(i))
			}
			res.padNoOpsUntil(i)
			res.appendCode(JUMPDEST)

			// Jumpdest in lfvm and evm share the same PC.
			pcMap.evmToLfvm[uint16(i)] = uint16(i)
			i++
			continue
		}

		// Convert instructions.
		pcMap.evmToLfvm[uint16(i)] = uint16(res.nextPos)
		pcMap.lfvmToEvm[uint16(res.nextPos)] = uint16(i)
		inc := appendInstructions(&res, i, code, with_super_instructions)
		i += inc + 1
	}

	// One past the end is a valid state for the PC after the execution has stopped.
	pcMap.evmToLfvm[uint16(len(code))] = uint16(res.length())
	pcMap.lfvmToEvm[uint16(res.length())] = uint16(len(code))

	return &pcMap, nil
}

// GenPcMapWithSuperInstructions creates a bidirectional program counter map for a given code,
// allowing mapping from a program counter in evm code to lfvm code utilizing super instructions and vice versa.
func GenPcMapWithSuperInstructions(code []byte) (*PcMap, error) {
	return GenPcMap(code, true)
}

// GenPcMapWithoutSuperInstructions creates a bidirectional program counter map for a given code,
// allowing mapping from a program counter in evm code to lfvm code not making use of super instructions and vice versa.
func GenPcMapWithoutSuperInstructions(code []byte) (*PcMap, error) {
	return GenPcMap(code, false)
}

func appendInstructions(res *codeBuilder, pos int, code []byte, with_super_instructions bool) int {
	// Convert super instructions.
	if with_super_instructions {
		if len(code) > pos+7 {
			op0 := evm.OpCode(code[pos])
			op1 := evm.OpCode(code[pos+1])
			op2 := evm.OpCode(code[pos+2])
			op3 := evm.OpCode(code[pos+3])
			op4 := evm.OpCode(code[pos+4])
			op5 := evm.OpCode(code[pos+5])
			op6 := evm.OpCode(code[pos+6])
			op7 := evm.OpCode(code[pos+7])
			if op0 == evm.PUSH1 && op2 == evm.PUSH4 && op7 == evm.DUP3 {
				res.appendOp(PUSH1_PUSH4_DUP3, uint16(op1)<<8)
				res.appendData(uint16(op3)<<8 | uint16(op4))
				res.appendData(uint16(op5)<<8 | uint16(op6))
				return 7
			}
			if op0 == evm.PUSH1 && op2 == evm.PUSH1 && op4 == evm.PUSH1 && op6 == evm.SHL && op7 == evm.SUB {
				res.appendOp(PUSH1_PUSH1_PUSH1_SHL_SUB, uint16(op1)<<8|uint16(op3))
				res.appendData(uint16(op5))
				return 7
			}
		}
		if len(code) > pos+4 {
			op0 := evm.OpCode(code[pos])
			op1 := evm.OpCode(code[pos+1])
			op2 := evm.OpCode(code[pos+2])
			op3 := evm.OpCode(code[pos+3])
			op4 := evm.OpCode(code[pos+4])
			if op0 == evm.AND && op1 == evm.SWAP1 && op2 == evm.POP && op3 == evm.SWAP2 && op4 == evm.SWAP1 {
				res.appendCode(AND_SWAP1_POP_SWAP2_SWAP1)
				return 4
			}
			if op0 == evm.ISZERO && op1 == evm.PUSH2 && op4 == evm.JUMPI {
				res.appendOp(ISZERO_PUSH2_JUMPI, uint16(op2)<<8|uint16(op3))
				return 4
			}
		}
		if len(code) > pos+3 {
			op0 := evm.OpCode(code[pos])
			op1 := evm.OpCode(code[pos+1])
			op2 := evm.OpCode(code[pos+2])
			op3 := evm.OpCode(code[pos+3])
			if op0 == evm.SWAP2 && op1 == evm.SWAP1 && op2 == evm.POP && op3 == evm.JUMP {
				res.appendCode(SWAP2_SWAP1_POP_JUMP)
				return 3
			}
			if op0 == evm.SWAP1 && op1 == evm.POP && op2 == evm.SWAP2 && op3 == evm.SWAP1 {
				res.appendCode(SWAP1_POP_SWAP2_SWAP1)
				return 3
			}
			if op0 == evm.POP && op1 == evm.SWAP2 && op2 == evm.SWAP1 && op3 == evm.POP {
				res.appendCode(POP_SWAP2_SWAP1_POP)
				return 3
			}
			if op0 == evm.PUSH2 && op3 == evm.JUMP {
				res.appendOp(PUSH2_JUMP, uint16(op1)<<8|uint16(op2))
				return 3
			}
			if op0 == evm.PUSH2 && op3 == evm.JUMPI {
				res.appendOp(PUSH2_JUMPI, uint16(op1)<<8|uint16(op2))
				return 3
			}
			if op0 == evm.PUSH1 && op2 == evm.PUSH1 {
				res.appendOp(PUSH1_PUSH1, uint16(op1)<<8|uint16(op3))
				return 3
			}
		}
		if len(code) > pos+2 {
			op0 := evm.OpCode(code[pos])
			op1 := evm.OpCode(code[pos+1])
			op2 := evm.OpCode(code[pos+2])
			if op0 == evm.PUSH1 && op2 == evm.ADD {
				res.appendOp(PUSH1_ADD, uint16(op1))
				return 2
			}
			if op0 == evm.PUSH1 && op2 == evm.SHL {
				res.appendOp(PUSH1_SHL, uint16(op1))
				return 2
			}
			if op0 == evm.PUSH1 && op2 == evm.DUP1 {
				res.appendOp(PUSH1_DUP1, uint16(op1))
				return 2
			}
		}
		if len(code) > pos+1 {
			op0 := evm.OpCode(code[pos])
			op1 := evm.OpCode(code[pos+1])
			if op0 == evm.SWAP1 && op1 == evm.POP {
				res.appendCode(SWAP1_POP)
				return 1
			}
			if op0 == evm.POP && op1 == evm.JUMP {
				res.appendCode(POP_JUMP)
				return 1
			}
			if op0 == evm.POP && op1 == evm.POP {
				res.appendCode(POP_POP)
				return 1
			}
			if op0 == evm.SWAP2 && op1 == evm.SWAP1 {
				res.appendCode(SWAP2_SWAP1)
				return 1
			}
			if op0 == evm.SWAP2 && op1 == evm.POP {
				res.appendCode(SWAP2_POP)
				return 1
			}
			if op0 == evm.DUP2 && op1 == evm.MSTORE {
				res.appendCode(DUP2_MSTORE)
				return 1
			}
			if op0 == evm.DUP2 && op1 == evm.LT {
				res.appendCode(DUP2_LT)
				return 1
			}
		}
	}

	// Convert individual instructions.
	opcode := evm.OpCode(code[pos])

	if opcode == evm.PC {
		if pos > 1<<16 {
			res.appendCode(INVALID)
			return 1
		}
		res.appendOp(PC, uint16(pos))
		return 0
	}

	if evm.PUSH1 <= opcode && opcode <= evm.PUSH32 {
		// Determine the number of bytes to be pushed.
		n := int(opcode) - int(evm.PUSH1) + 1

		var data []byte
		// If there are not enough bytes left in the code, rest is filled with 0
		// zeros are padded right
		if len(code) < pos+n+2 {
			ext := (pos + n + 2 - len(code)) / 2
			if (pos+n+2-len(code))%2 > 0 {
				ext++
			}
			if ext > 0 {
				ins := common.RightPadSlice(res.code[:], len(res.code)+ext)
				res.code = ins
			}
			data = common.RightPadSlice(code[pos+1:], n+1)
		} else {
			data = code[pos+1 : pos+1+n]
		}

		// Fix the op-codes of the resulting instructions
		if n == 1 {
			res.appendOp(PUSH1, uint16(data[0])<<8)
		} else {
			res.appendOp(PUSH1+OpCode(n-1), uint16(data[0])<<8|uint16(data[1]))
		}

		// Fix the arguments by packing them in pairs into the instructions.
		for i := 2; i < n-1; i += 2 {
			res.appendData(uint16(data[i])<<8 | uint16(data[i+1]))
		}
		if n > 1 && n%2 == 1 {
			res.appendData(uint16(data[n-1]) << 8)
		}

		return n
	}

	// All the rest converts to a single instruction.
	res.appendCode(op_2_op[opcode])
	return 0
}

var op_2_op = createOpToOpMap()

func createOpToOpMap() []OpCode {
	res := make([]OpCode, 256)
	for i := range res {
		res[i] = INVALID
	}

	// Stack operations
	res[evm.POP] = POP

	res[evm.DUP1] = DUP1
	res[evm.DUP2] = DUP2
	res[evm.DUP3] = DUP3
	res[evm.DUP4] = DUP4
	res[evm.DUP5] = DUP5
	res[evm.DUP6] = DUP6
	res[evm.DUP7] = DUP7
	res[evm.DUP8] = DUP8
	res[evm.DUP9] = DUP9
	res[evm.DUP10] = DUP10
	res[evm.DUP11] = DUP11
	res[evm.DUP12] = DUP12
	res[evm.DUP13] = DUP13
	res[evm.DUP14] = DUP14
	res[evm.DUP15] = DUP15
	res[evm.DUP16] = DUP16

	res[evm.SWAP1] = SWAP1
	res[evm.SWAP2] = SWAP2
	res[evm.SWAP3] = SWAP3
	res[evm.SWAP4] = SWAP4
	res[evm.SWAP5] = SWAP5
	res[evm.SWAP6] = SWAP6
	res[evm.SWAP7] = SWAP7
	res[evm.SWAP8] = SWAP8
	res[evm.SWAP9] = SWAP9
	res[evm.SWAP10] = SWAP10
	res[evm.SWAP11] = SWAP11
	res[evm.SWAP12] = SWAP12
	res[evm.SWAP13] = SWAP13
	res[evm.SWAP14] = SWAP14
	res[evm.SWAP15] = SWAP15
	res[evm.SWAP16] = SWAP16

	// Memory operations
	res[evm.MLOAD] = MLOAD
	res[evm.MSTORE] = MSTORE
	res[evm.MSTORE8] = MSTORE8
	res[evm.MSIZE] = MSIZE

	// Storage operations
	res[evm.SLOAD] = SLOAD
	res[evm.SSTORE] = SSTORE

	// Control flow
	res[evm.JUMP] = JUMP
	res[evm.JUMPI] = JUMPI
	res[evm.JUMPDEST] = JUMPDEST
	res[evm.STOP] = STOP
	res[evm.RETURN] = RETURN
	res[evm.REVERT] = REVERT
	res[evm.INVALID] = INVALID
	res[evm.PC] = PC

	// Arithmethic operations
	res[evm.ADD] = ADD
	res[evm.MUL] = MUL
	res[evm.SUB] = SUB
	res[evm.DIV] = DIV
	res[evm.SDIV] = SDIV
	res[evm.MOD] = MOD
	res[evm.SMOD] = SMOD
	res[evm.ADDMOD] = ADDMOD
	res[evm.MULMOD] = MULMOD
	res[evm.EXP] = EXP
	res[evm.SIGNEXTEND] = SIGNEXTEND

	// Complex function
	res[evm.KECCAK256] = SHA3

	// Comparison operations
	res[evm.LT] = LT
	res[evm.GT] = GT
	res[evm.SLT] = SLT
	res[evm.SGT] = SGT
	res[evm.EQ] = EQ
	res[evm.ISZERO] = ISZERO

	// Bit-pattern operations
	res[evm.AND] = AND
	res[evm.OR] = OR
	res[evm.XOR] = XOR
	res[evm.NOT] = NOT
	res[evm.BYTE] = BYTE
	res[evm.SHL] = SHL
	res[evm.SHR] = SHR
	res[evm.SAR] = SAR

	// System instructions
	res[evm.ADDRESS] = ADDRESS
	res[evm.BALANCE] = BALANCE
	res[evm.ORIGIN] = ORIGIN
	res[evm.CALLER] = CALLER
	res[evm.CALLVALUE] = CALLVALUE
	res[evm.CALLDATALOAD] = CALLDATALOAD
	res[evm.CALLDATASIZE] = CALLDATASIZE
	res[evm.CALLDATACOPY] = CALLDATACOPY
	res[evm.CODESIZE] = CODESIZE
	res[evm.CODECOPY] = CODECOPY
	res[evm.GAS] = GAS
	res[evm.GASPRICE] = GASPRICE
	res[evm.EXTCODESIZE] = EXTCODESIZE
	res[evm.EXTCODECOPY] = EXTCODECOPY
	res[evm.RETURNDATASIZE] = RETURNDATASIZE
	res[evm.RETURNDATACOPY] = RETURNDATACOPY
	res[evm.EXTCODEHASH] = EXTCODEHASH
	res[evm.CREATE] = CREATE
	res[evm.CALL] = CALL
	res[evm.CALLCODE] = CALLCODE
	res[evm.DELEGATECALL] = DELEGATECALL
	res[evm.CREATE2] = CREATE2
	res[evm.STATICCALL] = STATICCALL
	res[evm.SELFDESTRUCT] = SELFDESTRUCT

	// Block chain instructions
	res[evm.BLOCKHASH] = BLOCKHASH
	res[evm.COINBASE] = COINBASE
	res[evm.TIMESTAMP] = TIMESTAMP
	res[evm.NUMBER] = NUMBER
	res[evm.DIFFICULTY] = DIFFICULTY
	res[evm.GASLIMIT] = GASLIMIT
	res[evm.CHAINID] = CHAINID
	res[evm.SELFBALANCE] = SELFBALANCE
	res[evm.BASEFEE] = BASEFEE

	// Log instructions
	res[evm.LOG0] = LOG0
	res[evm.LOG1] = LOG1
	res[evm.LOG2] = LOG2
	res[evm.LOG3] = LOG3
	res[evm.LOG4] = LOG4

	// Test that all EVM instructions are covered.
	for i := 0; i < 256; i++ {
		code := evm.OpCode(i)

		// Known OpCodes that are indeed invalid.
		if code == evm.INVALID {
			continue
		}

		// Push operations are not required to be mapped, they are handled explicitly.
		if evm.PUSH1 <= code && code <= evm.PUSH32 {
			continue
		}

		toImplement := []evm.OpCode{evm.PREVRANDAO, evm.PUSH0, evm.TLOAD, evm.TSTORE, evm.MCOPY, evm.BLOBHASH, evm.BLOBBASEFEE} // TODO implement for new revision support
		opIsValid := !strings.Contains(fmt.Sprintf("%v", code), "not defined")
		if res[code] == INVALID && opIsValid && !slices.Contains(toImplement, code) {
			panic(fmt.Sprintf("Missing instruction coverage for: %v", code))
		}
	}

	return res
}
