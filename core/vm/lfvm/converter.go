package lfvm

import (
	"fmt"
	"strings"
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"
)

type cache_key struct {
	addr            common.Address
	contract_length int
}

type cache_val struct {
	oldCode []byte
	code    Code
}

var changedAddress01 = common.HexToAddress("0xA7CC236F81b04c1058e9bfb70E0Ee9940e271676")
var changedAddress02 = common.HexToAddress("0xAD0FB83a110c3694faDa81e8B396716a610c4030")
var changedAddress03 = common.HexToAddress("0xA8B3C9f298877dD93F30E8Ed359956faE10E8797")
var changedAddress04 = common.HexToAddress("0x6DBd5d37397afF80BE434F74cc89b9d933635784")

var mu = sync.Mutex{}
var cache = map[cache_key]cache_val{}

func clearConversionCache() {
	mu.Lock()
	defer mu.Unlock()
	cache = map[cache_key]cache_val{}
}

func Convert(addr common.Address, code []byte, with_super_instructions bool, blk uint64, create bool) (Code, error) {
	key := cache_key{addr, len(code)}
	mu.Lock()
	res, exists := cache[key]
	if exists && !create {
		isEqual := true
		if addr == changedAddress01 || addr == changedAddress02 || addr == changedAddress03 || addr == changedAddress04 {
			// fmt.Println("Address: ", addr.String(), " blk: ", blk)

			for i, v := range res.oldCode {
				if v != code[i] {
					fmt.Println("Different code for address: ", addr.String(), " blk: ", blk)
					isEqual = false
					break
				}
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
		cache[key] = cache_val{oldCode: code, code: resCode}
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
	for _, op := range b.code[b.nextPos:pos] {
		op.opcode = NOOP
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
		if code[i] == byte(vm.JUMPDEST) {
			if res.length() > i {
				return nil, fmt.Errorf("unable to convert code, encountered targe block larger than input")
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

func appendInstructions(res *codeBuilder, pos int, code []byte, with_super_instructions bool) int {
	// Convert super instructions.
	if with_super_instructions {
		if len(code) > pos+7 {
			op0 := vm.OpCode(code[pos])
			op1 := vm.OpCode(code[pos+1])
			op2 := vm.OpCode(code[pos+2])
			op3 := vm.OpCode(code[pos+3])
			op4 := vm.OpCode(code[pos+4])
			op5 := vm.OpCode(code[pos+5])
			op6 := vm.OpCode(code[pos+6])
			op7 := vm.OpCode(code[pos+7])
			if op0 == vm.PUSH1 && op2 == vm.PUSH4 && op7 == vm.DUP3 {
				res.appendOp(PUSH1_PUSH4_DUP3, uint16(op1)<<8)
				res.appendData(uint16(op3)<<8 | uint16(op4))
				res.appendData(uint16(op5)<<8 | uint16(op6))
				return 7
			}
			if op0 == vm.PUSH1 && op2 == vm.PUSH1 && op4 == vm.PUSH1 && op6 == vm.SHL && op7 == vm.SUB {
				res.appendOp(PUSH1_PUSH1_PUSH1_SHL_SUB, uint16(op1)<<8|uint16(op3))
				res.appendData(uint16(op5))
				return 7
			}
		}
		if len(code) > pos+4 {
			op0 := vm.OpCode(code[pos])
			op1 := vm.OpCode(code[pos+1])
			op2 := vm.OpCode(code[pos+2])
			op3 := vm.OpCode(code[pos+3])
			op4 := vm.OpCode(code[pos+4])
			if op0 == vm.AND && op1 == vm.SWAP1 && op2 == vm.POP && op3 == vm.SWAP2 && op4 == vm.SWAP1 {
				res.appendCode(AND_SWAP1_POP_SWAP2_SWAP1)
				return 4
			}
			if op0 == vm.ISZERO && op1 == vm.PUSH2 && op4 == vm.JUMPI {
				res.appendOp(ISZERO_PUSH2_JUMPI, uint16(op2)<<8|uint16(op3))
				return 4
			}
		}
		if len(code) > pos+3 {
			op0 := vm.OpCode(code[pos])
			op1 := vm.OpCode(code[pos+1])
			op2 := vm.OpCode(code[pos+2])
			op3 := vm.OpCode(code[pos+3])
			if op0 == vm.SWAP2 && op1 == vm.SWAP1 && op2 == vm.POP && op3 == vm.JUMP {
				res.appendCode(SWAP2_SWAP1_POP_JUMP)
				return 3
			}
			if op0 == vm.SWAP1 && op1 == vm.POP && op2 == vm.SWAP2 && op3 == vm.SWAP1 {
				res.appendCode(SWAP1_POP_SWAP2_SWAP1)
				return 3
			}
			if op0 == vm.POP && op1 == vm.SWAP2 && op2 == vm.SWAP1 && op3 == vm.POP {
				res.appendCode(POP_SWAP2_SWAP1_POP)
				return 3
			}
			if op0 == vm.PUSH2 && op3 == vm.JUMP {
				res.appendOp(PUSH2_JUMP, uint16(op1)<<8|uint16(op2))
				return 3
			}
			if op0 == vm.PUSH2 && op3 == vm.JUMPI {
				res.appendOp(PUSH2_JUMPI, uint16(op1)<<8|uint16(op2))
				return 3
			}
			if op0 == vm.PUSH1 && op2 == vm.PUSH1 {
				res.appendOp(PUSH1_PUSH1, uint16(op1)<<8|uint16(op3))
				return 3
			}
		}
		if len(code) > pos+2 {
			op0 := vm.OpCode(code[pos])
			op1 := vm.OpCode(code[pos+1])
			op2 := vm.OpCode(code[pos+2])
			if op0 == vm.PUSH1 && op2 == vm.ADD {
				res.appendOp(PUSH1_ADD, uint16(op1))
				return 2
			}
			if op0 == vm.PUSH1 && op2 == vm.SHL {
				res.appendOp(PUSH1_SHL, uint16(op1))
				return 2
			}
			if op0 == vm.PUSH1 && op2 == vm.DUP1 {
				res.appendOp(PUSH1_DUP1, uint16(op1))
				return 2
			}
		}
		if len(code) > pos+1 {
			op0 := vm.OpCode(code[pos])
			op1 := vm.OpCode(code[pos+1])
			if op0 == vm.SWAP1 && op1 == vm.POP {
				res.appendCode(SWAP1_POP)
				return 1
			}
			if op0 == vm.POP && op1 == vm.JUMP {
				res.appendCode(POP_JUMP)
				return 1
			}
			if op0 == vm.POP && op1 == vm.POP {
				res.appendCode(POP_POP)
				return 1
			}
			if op0 == vm.SWAP2 && op1 == vm.SWAP1 {
				res.appendCode(SWAP2_SWAP1)
				return 1
			}
			if op0 == vm.SWAP2 && op1 == vm.POP {
				res.appendCode(SWAP2_POP)
				return 1
			}
			if op0 == vm.DUP2 && op1 == vm.MSTORE {
				res.appendCode(DUP2_MSTORE)
				return 1
			}
			if op0 == vm.DUP2 && op1 == vm.LT {
				res.appendCode(DUP2_LT)
				return 1
			}
		}
	}

	// Convert individual instructions.
	opcode := vm.OpCode(code[pos])

	if opcode == vm.PC {
		if pos > 1<<16 {
			res.appendCode(INVALID)
			return 1
		}
		res.appendOp(PC, uint16(pos))
		return 0
	}

	if vm.PUSH1 <= opcode && opcode <= vm.PUSH32 {
		// Determine the number of bytes to be pushed.
		n := int(opcode) - int(vm.PUSH1) + 1

		var data []byte
		// If there are not enough bytes left in the code, rest is filled with 0
		// zeros are padded right
		if len(code) < pos+n+2 {
			ext := (pos + n + 2 - len(code)) / 2
			if (pos+n+2-len(code))%2 > 0 {
				ext++
			}
			if ext > 0 {
				ins := make([]Instruction, len(res.code)+ext)
				copy(ins, res.code[:])
				res.code = ins
			}
			data = make([]byte, n+1)
			copy(data, code[pos+1:])
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
	res[vm.POP] = POP

	res[vm.DUP1] = DUP1
	res[vm.DUP2] = DUP2
	res[vm.DUP3] = DUP3
	res[vm.DUP4] = DUP4
	res[vm.DUP5] = DUP5
	res[vm.DUP6] = DUP6
	res[vm.DUP7] = DUP7
	res[vm.DUP8] = DUP8
	res[vm.DUP9] = DUP9
	res[vm.DUP10] = DUP10
	res[vm.DUP11] = DUP11
	res[vm.DUP12] = DUP12
	res[vm.DUP13] = DUP13
	res[vm.DUP14] = DUP14
	res[vm.DUP15] = DUP15
	res[vm.DUP16] = DUP16

	res[vm.SWAP1] = SWAP1
	res[vm.SWAP2] = SWAP2
	res[vm.SWAP3] = SWAP3
	res[vm.SWAP4] = SWAP4
	res[vm.SWAP5] = SWAP5
	res[vm.SWAP6] = SWAP6
	res[vm.SWAP7] = SWAP7
	res[vm.SWAP8] = SWAP8
	res[vm.SWAP9] = SWAP9
	res[vm.SWAP10] = SWAP10
	res[vm.SWAP11] = SWAP11
	res[vm.SWAP12] = SWAP12
	res[vm.SWAP13] = SWAP13
	res[vm.SWAP14] = SWAP14
	res[vm.SWAP15] = SWAP15
	res[vm.SWAP16] = SWAP16

	// Memory operations
	res[vm.MLOAD] = MLOAD
	res[vm.MSTORE] = MSTORE
	res[vm.MSTORE8] = MSTORE8
	res[vm.MSIZE] = MSIZE

	// Storage operations
	res[vm.SLOAD] = SLOAD
	res[vm.SSTORE] = SSTORE

	// Control flow
	res[vm.JUMP] = JUMP
	res[vm.JUMPI] = JUMPI
	res[vm.JUMPDEST] = JUMPDEST
	res[vm.STOP] = STOP
	res[vm.RETURN] = RETURN
	res[vm.REVERT] = REVERT
	res[vm.INVALID] = INVALID
	res[vm.PC] = PC

	// Arithmethic operations
	res[vm.ADD] = ADD
	res[vm.MUL] = MUL
	res[vm.SUB] = SUB
	res[vm.DIV] = DIV
	res[vm.SDIV] = SDIV
	res[vm.MOD] = MOD
	res[vm.SMOD] = SMOD
	res[vm.ADDMOD] = ADDMOD
	res[vm.MULMOD] = MULMOD
	res[vm.EXP] = EXP
	res[vm.SIGNEXTEND] = SIGNEXTEND

	// Complex function
	res[vm.SHA3] = SHA3

	// Comparison operations
	res[vm.LT] = LT
	res[vm.GT] = GT
	res[vm.SLT] = SLT
	res[vm.SGT] = SGT
	res[vm.EQ] = EQ
	res[vm.ISZERO] = ISZERO

	// Bit-pattern operations
	res[vm.AND] = AND
	res[vm.OR] = OR
	res[vm.XOR] = XOR
	res[vm.NOT] = NOT
	res[vm.BYTE] = BYTE
	res[vm.SHL] = SHL
	res[vm.SHR] = SHR
	res[vm.SAR] = SAR

	// System instructions
	res[vm.ADDRESS] = ADDRESS
	res[vm.BALANCE] = BALANCE
	res[vm.ORIGIN] = ORIGIN
	res[vm.CALLER] = CALLER
	res[vm.CALLVALUE] = CALLVALUE
	res[vm.CALLDATALOAD] = CALLDATALOAD
	res[vm.CALLDATASIZE] = CALLDATASIZE
	res[vm.CALLDATACOPY] = CALLDATACOPY
	res[vm.CODESIZE] = CODESIZE
	res[vm.CODECOPY] = CODECOPY
	res[vm.GAS] = GAS
	res[vm.GASPRICE] = GASPRICE
	res[vm.EXTCODESIZE] = EXTCODESIZE
	res[vm.EXTCODECOPY] = EXTCODECOPY
	res[vm.RETURNDATASIZE] = RETURNDATASIZE
	res[vm.RETURNDATACOPY] = RETURNDATACOPY
	res[vm.EXTCODEHASH] = EXTCODEHASH
	res[vm.CREATE] = CREATE
	res[vm.CALL] = CALL
	res[vm.CALLCODE] = CALLCODE
	res[vm.DELEGATECALL] = DELEGATECALL
	res[vm.CREATE2] = CREATE2
	res[vm.STATICCALL] = STATICCALL
	res[vm.SELFDESTRUCT] = SELFDESTRUCT

	// Block chain instructions
	res[vm.BLOCKHASH] = BLOCKHASH
	res[vm.COINBASE] = COINBASE
	res[vm.TIMESTAMP] = TIMESTAMP
	res[vm.NUMBER] = NUMBER
	res[vm.DIFFICULTY] = DIFFICULTY
	res[vm.GASLIMIT] = GASLIMIT
	res[vm.CHAINID] = CHAINID
	res[vm.SELFBALANCE] = SELFBALANCE
	res[vm.BASEFEE] = BASEFEE

	// Log instructions
	res[vm.LOG0] = LOG0
	res[vm.LOG1] = LOG1
	res[vm.LOG2] = LOG2
	res[vm.LOG3] = LOG3
	res[vm.LOG4] = LOG4

	// Test that all EVM instructions are covered.
	for i := 0; i < 256; i++ {
		code := vm.OpCode(i)

		// Known OpCodes that are indeed invalid.
		if code == vm.INVALID || code == vm.PUSH || code == vm.SWAP || code == vm.DUP {
			continue
		}

		// Push operations are not required to be mapped, they are handled explicitly.
		if vm.PUSH1 <= code && code <= vm.PUSH32 {
			continue
		}

		opIsValid := !strings.Contains(fmt.Sprintf("%v", code), "not defined")
		if res[code] == INVALID && opIsValid {
			panic(fmt.Sprintf("Missing instruction coverage for: %v", code))
		}
	}

	return res
}
