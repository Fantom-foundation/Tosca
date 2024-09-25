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
	"math"
	"unsafe"

	"github.com/Fantom-foundation/Tosca/go/ct/common"
	"github.com/Fantom-foundation/Tosca/go/tosca"

	"github.com/Fantom-foundation/Tosca/go/tosca/vm"
	lru "github.com/hashicorp/golang-lru/v2"
)

// ConversionConfig contains a set of configuration options for the code conversion.
type ConversionConfig struct {
	// CacheSize is the maximum size of the maintained code cache in bytes.
	// If set to 0, a default size is used. If negative, no cache is used.
	// Cache sizes are grown in increments of maxCachedCodeLength.
	// Positive values larger than 0 but less than maxCachedCodeLength are
	// reported as invalid cache sizes during initialization.
	CacheSize int
	// WithSuperInstructions enables the use of super instructions.
	WithSuperInstructions bool
}

// Converter converts EVM code to LFVM code.
type Converter struct {
	config ConversionConfig
	cache  *lru.Cache[tosca.Hash, Code]
}

// NewConverter creates a new code converter with the provided configuration.
func NewConverter(config ConversionConfig) (*Converter, error) {
	if config.CacheSize == 0 {
		config.CacheSize = (1 << 30) // = 1GiB
	}

	var cache *lru.Cache[tosca.Hash, Code]
	if config.CacheSize > 0 {
		var err error
		const instructionSize = int(unsafe.Sizeof(Instruction{}))
		capacity := config.CacheSize / maxCachedCodeLength / instructionSize
		cache, err = lru.New[tosca.Hash, Code](capacity)
		if err != nil {
			return nil, err
		}
	}
	return &Converter{
		config: config,
		cache:  cache,
	}, nil
}

// Convert converts EVM code to LFVM code. If the provided code hash is not nil,
// it is assumed to be a valid hash of the code and is used to cache the
// conversion result. If the hash is nil, the conversion result is not cached.
func (c *Converter) Convert(code []byte, codeHash *tosca.Hash) Code {
	if c.cache == nil || codeHash == nil {
		return convert(code, c.config)
	}

	res, exists := c.cache.Get(*codeHash)
	if exists {
		return res
	}

	res = convert(code, c.config)
	if len(res) > maxCachedCodeLength {
		return res
	}

	c.cache.Add(*codeHash, res)
	return res
}

// maxCachedCodeLength is the maximum length of a code in bytes that are
// retained in the cache. To avoid excessive memory usage, longer codes are not
// cached. The defined limit is the current limit for codes stored on the chain.
// Only initialization codes can be longer. Since the Shanghai hard fork, the
// maximum size of initialization codes is 2 * 24_576 = 49_152 bytes (see
// https://eips.ethereum.org/EIPS/eip-3860). Such init codes are deliberately
// not cached due to the expected limited re-use and the missing code hash.
const maxCachedCodeLength = 1<<14 + 1<<13 // = 24_576 bytes

// --- code builder ---

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

func convert(code []byte, options ConversionConfig) Code {
	return convertWithObserver(code, options, func(int, int) {})
}

// convertWithObserver converts EVM code to LFVM code and calls the observer
// with the code position of every pair of instructions converted.
func convertWithObserver(
	code []byte,
	options ConversionConfig,
	observer func(evmPc int, lfvmPc int),
) Code {
	res := newCodeBuilder(len(code))

	// Convert each individual instruction.
	for i := 0; i < len(code); {
		// Handle jump destinations
		if code[i] == byte(vm.JUMPDEST) {
			// Jump to the next jump destination and fill space with noops
			if res.length() < i {
				res.appendOp(JUMP_TO, uint16(i))
			}
			res.padNoOpsUntil(i)
			res.appendCode(JUMPDEST)
			observer(i, i)
			i++
			continue
		}

		// Convert instructions
		observer(i, res.nextPos)
		inc := appendInstructions(&res, i, code, options.WithSuperInstructions)
		i += inc + 1
	}
	return res.toCode()
}

func appendInstructions(res *codeBuilder, pos int, code []byte, withSuperInstructions bool) int {
	// Convert super instructions.
	if withSuperInstructions {
		if n := appendSuperInstructions(res, pos, code); n > 0 {
			return n
		}
	}

	// Convert individual instructions.
	toscaOpCode := vm.OpCode(code[pos])

	if toscaOpCode == vm.PC {
		if pos > math.MaxUint16 {
			res.appendCode(INVALID)
			return 1
		}
		res.appendOp(PC, uint16(pos))
		return 0
	}

	if vm.PUSH1 <= toscaOpCode && toscaOpCode <= vm.PUSH32 {
		// Determine the number of bytes to be pushed.
		numBytes := int(toscaOpCode) - int(vm.PUSH1) + 1

		var data []byte
		// If there are not enough bytes left in the code, rest is filled with 0
		// zeros are padded right
		if len(code) < pos+numBytes+2 {
			extension := (pos + numBytes + 2 - len(code)) / 2
			if (pos+numBytes+2-len(code))%2 > 0 {
				extension++
			}
			if extension > 0 {
				instruction := common.RightPadSlice(res.code[:], len(res.code)+extension)
				res.code = instruction
			}
			data = common.RightPadSlice(code[pos+1:], numBytes+1)
		} else {
			data = code[pos+1 : pos+1+numBytes]
		}

		// Fix the op-codes of the resulting instructions
		if numBytes == 1 {
			res.appendOp(PUSH1, uint16(data[0])<<8)
		} else {
			res.appendOp(PUSH1+OpCode(numBytes-1), uint16(data[0])<<8|uint16(data[1]))
		}

		// Fix the arguments by packing them in pairs into the instructions.
		for i := 2; i < numBytes-1; i += 2 {
			res.appendData(uint16(data[i])<<8 | uint16(data[i+1]))
		}
		if numBytes > 1 && numBytes%2 == 1 {
			res.appendData(uint16(data[numBytes-1]) << 8)
		}

		return numBytes
	}

	// All the rest converts to a single instruction.
	res.appendCode(OpCode(toscaOpCode))
	return 0
}

func appendSuperInstructions(res *codeBuilder, pos int, code []byte) int {
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
	return 0
}
