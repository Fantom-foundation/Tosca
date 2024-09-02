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
	"testing"

	"github.com/Fantom-foundation/Tosca/go/ct/gen"
	"github.com/Fantom-foundation/Tosca/go/tosca/vm"

	"pgregory.net/rand"
)

func FuzzLfvmConverter(f *testing.F) {

	// Add empty code
	f.Add([]byte{})

	// Use CT code generator to generate one contract starting with each
	// opcode
	rnd := rand.New(1) // deterministic to preserve initial corpus coherence
	generator := gen.NewCodeGenerator()
	empty := generator.Clone()
	for i := 0; i <= 0xFF; i++ {
		op := vm.OpCode(i)
		if !vm.IsValid(op) {
			continue
		}
		generator.Restore(empty)
		generator.SetOperation(0, op)
		code, err := generator.Generate(gen.Assignment{}, rnd)
		if err != nil {
			f.Errorf("Error generating code for opCode %v", op)
		}
		f.Add(code.Copy())
	}

	f.Fuzz(func(t *testing.T, toscaCode []byte) {

		// EIP-170 stablish maximum code size
		// (see https://eips.ethereum.org/EIPS/eip-170)
		maxCodeSize := 24_576
		if len(toscaCode) > maxCodeSize {
			t.Skip()
		}

		type pair struct {
			originalPos, lfvmPos int
		}
		var pairs []pair
		lfvmCode := convertWithObserver(toscaCode, ConversionConfig{}, func(evm, lfvm int) {
			pairs = append(pairs, pair{evm, lfvm})
		})

		// Check that all operations are mapped to matching operations.
		for _, p := range pairs {

			toscaOpCode := vm.OpCode(toscaCode[p.originalPos])
			lfvmOpCode := lfvmCode[p.lfvmPos].opcode

			if !vm.IsValid(toscaOpCode) && lfvmOpCode != INVALID {
				t.Errorf("Expected INVALID, got %v", lfvmOpCode.String())
			}

			if vm.IsValid(toscaOpCode) {
				if got, want := toscaToLfvmOpcode(toscaOpCode), lfvmOpCode; got != want {
					t.Errorf("Expected %v, got %v", want, got)
				}
			}
		}

		// Check that the position of JUMPDEST ops are preserved.
		for _, p := range pairs {
			if vm.OpCode(toscaCode[p.originalPos]) == vm.JUMPDEST {
				if p.originalPos != p.lfvmPos {
					t.Errorf("Expected JUMPDEST at %d, got %d", p.originalPos, p.lfvmPos)
				}
			}
		}
	})
}

func TestFuzzerTooling_OpCodeConversionIsComplete(t *testing.T) {
	// Check that all Tosca opcodes are mapped to LFVM opcodes.
	for i := 0; i <= 0xFF; i++ {
		op := vm.OpCode(i)
		if !vm.IsValid(op) {
			continue
		}
		if toscaToLfvmOpcode(op) == INVALID {
			t.Errorf("Opcode %v is not mapped to LFVM", op)
		}
	}
}

func toscaToLfvmOpcode(op vm.OpCode) OpCode {

	switch op {
	case vm.POP:
		return POP
	case vm.PUSH2:
		return PUSH2
	case vm.JUMP:
		return JUMP
	case vm.SWAP1:
		return SWAP1
	case vm.SWAP2:
		return SWAP2
	case vm.DUP3:
		return DUP3
	case vm.PUSH1:
		return PUSH1
	case vm.PUSH4:
		return PUSH4
	case vm.AND:
		return AND
	case vm.SWAP3:
		return SWAP3
	case vm.JUMPI:
		return JUMPI
	case vm.JUMPDEST:
		return JUMPDEST
	case vm.GT:
		return GT
	case vm.DUP4:
		return DUP4
	case vm.DUP2:
		return DUP2
	case vm.ISZERO:
		return ISZERO
	case vm.SUB:
		return SUB
	case vm.ADD:
		return ADD
	case vm.DUP5:
		return DUP5
	case vm.DUP1:
		return DUP1
	case vm.EQ:
		return EQ
	case vm.LT:
		return LT
	case vm.SLT:
		return SLT
	case vm.SHR:
		return SHR
	case vm.DUP6:
		return DUP6
	case vm.RETURN:
		return RETURN
	case vm.REVERT:
		return REVERT
	case vm.PUSH32:
		return PUSH32

	case vm.PUSH0:
		return PUSH0
	case vm.PUSH3:
		return PUSH3
	case vm.PUSH5:
		return PUSH5
	case vm.PUSH6:
		return PUSH6
	case vm.PUSH7:
		return PUSH7
	case vm.PUSH8:
		return PUSH8
	case vm.PUSH9:
		return PUSH9
	case vm.PUSH10:
		return PUSH10
	case vm.PUSH11:
		return PUSH11
	case vm.PUSH12:
		return PUSH12
	case vm.PUSH13:
		return PUSH13
	case vm.PUSH14:
		return PUSH14
	case vm.PUSH15:
		return PUSH15
	case vm.PUSH16:
		return PUSH16
	case vm.PUSH17:
		return PUSH17
	case vm.PUSH18:
		return PUSH18
	case vm.PUSH19:
		return PUSH19
	case vm.PUSH20:
		return PUSH20
	case vm.PUSH21:
		return PUSH21
	case vm.PUSH22:
		return PUSH22
	case vm.PUSH23:
		return PUSH23
	case vm.PUSH24:
		return PUSH24
	case vm.PUSH25:
		return PUSH25
	case vm.PUSH26:
		return PUSH26
	case vm.PUSH27:
		return PUSH27
	case vm.PUSH28:
		return PUSH28
	case vm.PUSH29:
		return PUSH29
	case vm.PUSH30:
		return PUSH30
	case vm.PUSH31:
		return PUSH31
	case vm.DUP7:
		return DUP7
	case vm.DUP8:
		return DUP8
	case vm.DUP9:
		return DUP9
	case vm.DUP10:
		return DUP10
	case vm.DUP11:
		return DUP11
	case vm.DUP12:
		return DUP12
	case vm.DUP13:
		return DUP13
	case vm.DUP14:
		return DUP14
	case vm.DUP15:
		return DUP15
	case vm.DUP16:
		return DUP16
	case vm.SWAP4:
		return SWAP4
	case vm.SWAP5:
		return SWAP5
	case vm.SWAP6:
		return SWAP6
	case vm.SWAP7:
		return SWAP7
	case vm.SWAP8:
		return SWAP8
	case vm.SWAP9:
		return SWAP9
	case vm.SWAP10:
		return SWAP10
	case vm.SWAP11:
		return SWAP11
	case vm.SWAP12:
		return SWAP12
	case vm.SWAP13:
		return SWAP13
	case vm.SWAP14:
		return SWAP14
	case vm.SWAP15:
		return SWAP15
	case vm.SWAP16:
		return SWAP16

	case vm.STOP:
		return STOP
	case vm.PC:
		return PC

	case vm.MUL:
		return MUL
	case vm.DIV:
		return DIV
	case vm.SDIV:
		return SDIV
	case vm.MOD:
		return MOD
	case vm.SMOD:
		return SMOD
	case vm.ADDMOD:
		return ADDMOD
	case vm.MULMOD:
		return MULMOD
	case vm.EXP:
		return EXP
	case vm.SIGNEXTEND:
		return SIGNEXTEND

	case vm.SHA3:
		return SHA3

	case vm.SGT:
		return SGT

	case vm.OR:
		return OR
	case vm.XOR:
		return XOR
	case vm.NOT:
		return NOT
	case vm.BYTE:
		return BYTE
	case vm.SHL:
		return SHL
	case vm.SAR:
		return SAR

	case vm.MSTORE:
		return MSTORE
	case vm.MSTORE8:
		return MSTORE8
	case vm.MLOAD:
		return MLOAD
	case vm.MSIZE:
		return MSIZE
	case vm.MCOPY:
		return MCOPY

	case vm.SLOAD:
		return SLOAD
	case vm.SSTORE:
		return SSTORE
	case vm.TLOAD:
		return TLOAD
	case vm.TSTORE:
		return TSTORE

	case vm.LOG0:
		return LOG0
	case vm.LOG1:
		return LOG1
	case vm.LOG2:
		return LOG2
	case vm.LOG3:
		return LOG3
	case vm.LOG4:
		return LOG4

	case vm.ADDRESS:
		return ADDRESS
	case vm.BALANCE:
		return BALANCE
	case vm.ORIGIN:
		return ORIGIN
	case vm.CALLER:
		return CALLER
	case vm.CALLVALUE:
		return CALLVALUE
	case vm.CALLDATALOAD:
		return CALLDATALOAD
	case vm.CALLDATASIZE:
		return CALLDATASIZE
	case vm.CALLDATACOPY:
		return CALLDATACOPY
	case vm.CODESIZE:
		return CODESIZE
	case vm.CODECOPY:
		return CODECOPY
	case vm.GASPRICE:
		return GASPRICE
	case vm.EXTCODESIZE:
		return EXTCODESIZE
	case vm.EXTCODECOPY:
		return EXTCODECOPY
	case vm.RETURNDATASIZE:
		return RETURNDATASIZE
	case vm.RETURNDATACOPY:
		return RETURNDATACOPY
	case vm.EXTCODEHASH:
		return EXTCODEHASH
	case vm.CREATE:
		return CREATE
	case vm.CALL:
		return CALL
	case vm.CALLCODE:
		return CALLCODE
	case vm.DELEGATECALL:
		return DELEGATECALL
	case vm.CREATE2:
		return CREATE2
	case vm.STATICCALL:
		return STATICCALL
	case vm.SELFDESTRUCT:
		return SELFDESTRUCT

	case vm.BLOCKHASH:
		return BLOCKHASH
	case vm.COINBASE:
		return COINBASE
	case vm.TIMESTAMP:
		return TIMESTAMP
	case vm.NUMBER:
		return NUMBER
	case vm.PREVRANDAO:
		return PREVRANDAO
	case vm.GAS:
		return GAS
	case vm.GASLIMIT:
		return GASLIMIT
	case vm.CHAINID:
		return CHAINID
	case vm.SELFBALANCE:
		return SELFBALANCE
	case vm.BASEFEE:
		return BASEFEE
	case vm.BLOBHASH:
		return BLOBHASH
	case vm.BLOBBASEFEE:
		return BLOBBASEFEE
	}
	return INVALID
}
