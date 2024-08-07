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
	"math"

	"github.com/Fantom-foundation/Tosca/go/tosca"
	"github.com/ethereum/go-ethereum/params"
	"github.com/holiman/uint256"
	"golang.org/x/crypto/sha3"
)

func opStop(c *context) {
	c.status = STOPPED
}

func opRevert(c *context) {
	c.result_offset = *c.stack.pop()
	c.result_size = *c.stack.pop()
	c.status = REVERTED
}

func opReturn(c *context) {
	c.result_offset = *c.stack.pop()
	c.result_size = *c.stack.pop()
	c.status = RETURNED
}

func opPc(c *context) {
	c.stack.pushEmpty().SetUint64(uint64(c.code[c.pc].arg))
}

func checkJumpDest(c *context) {
	if int(c.pc+1) >= len(c.code) || c.code[c.pc+1].opcode != JUMPDEST {
		c.SignalError(errInvalidJump)
	}
}

func opJump(c *context) {
	destination := c.stack.pop()
	// overflow check
	if !destination.IsUint64() || destination.Uint64() > math.MaxInt32 {
		c.SignalError(errInvalidJump)
		return
	}
	// Update the PC to the jump destination -1 since interpreter will increase PC by 1 afterward.
	c.pc = int32(destination.Uint64()) - 1
	checkJumpDest(c)
}

func opJumpi(c *context) {
	destination := c.stack.pop()
	condition := c.stack.pop()
	if !condition.IsZero() {
		// overflow check
		if !destination.IsUint64() || destination.Uint64() > math.MaxInt32 {
			c.SignalError(errInvalidJump)
			return
		}
		// Update the PC to the jump destination -1 since interpreter will increase PC by 1 afterward.
		c.pc = int32(destination.Uint64()) - 1
		checkJumpDest(c)
	}
}

func opJumpTo(c *context) {
	// Update the PC to the jump destination -1 since interpreter will increase PC by 1 afterward.
	c.pc = int32(c.code[c.pc].arg) - 1
}

func opPop(c *context) {
	c.stack.pop()
}

func opPush(c *context, n int) {
	z := c.stack.pushEmpty()
	num_instructions := int32(n/2 + n%2)
	data := c.code[c.pc : c.pc+num_instructions]

	_ = data[num_instructions-1]
	var value [32]byte
	for i := 0; i < n; i++ {
		if i%2 == 0 {
			value[i] = byte(data[i/2].arg >> 8)
		} else {
			value[i] = byte(data[i/2].arg)
		}
	}
	z.SetBytes(value[0:n])
	c.pc += num_instructions - 1
}

func opPush0(c *context) {
	if c.isShanghai() {
		z := c.stack.pushEmpty()
		z[3], z[2], z[1], z[0] = 0, 0, 0, 0
	} else {
		c.status = INVALID_INSTRUCTION
	}
}

func opPush1(c *context) {
	z := c.stack.pushEmpty()
	z[3], z[2], z[1] = 0, 0, 0
	z[0] = uint64(c.code[c.pc].arg >> 8)
}

func opPush2(c *context) {
	z := c.stack.pushEmpty()
	z[3], z[2], z[1] = 0, 0, 0
	z[0] = uint64(c.code[c.pc].arg)
}

func opPush3(c *context) {
	z := c.stack.pushEmpty()
	z[3], z[2], z[1] = 0, 0, 0
	data := c.code[c.pc : c.pc+2]
	_ = data[1]
	z[0] = uint64(data[0].arg)<<8 | uint64(data[1].arg>>8)
	c.pc += 1
}

func opPush4(c *context) {
	z := c.stack.pushEmpty()
	z[3], z[2], z[1] = 0, 0, 0

	data := c.code[c.pc : c.pc+2]
	_ = data[1] // causes bound check to be performed only once (may become unneded in the future)
	z[0] = (uint64(data[0].arg) << 16) | uint64(data[1].arg)
	c.pc += 1
}

func opPush32(c *context) {
	z := c.stack.pushEmpty()

	data := c.code[c.pc : c.pc+16]
	_ = data[15] // causes bound check to be performed only once (may become unneded in the future)
	z[3] = (uint64(data[0].arg) << 48) | (uint64(data[1].arg) << 32) | (uint64(data[2].arg) << 16) | uint64(data[3].arg)
	z[2] = (uint64(data[4].arg) << 48) | (uint64(data[5].arg) << 32) | (uint64(data[6].arg) << 16) | uint64(data[7].arg)
	z[1] = (uint64(data[8].arg) << 48) | (uint64(data[9].arg) << 32) | (uint64(data[10].arg) << 16) | uint64(data[11].arg)
	z[0] = (uint64(data[12].arg) << 48) | (uint64(data[13].arg) << 32) | (uint64(data[14].arg) << 16) | uint64(data[15].arg)
	c.pc += 15
}

func opDup(c *context, pos int) {
	c.stack.dup(pos)
}

func opSwap(c *context, pos int) {
	c.stack.swap(pos + 1)
}

func opMstore(c *context) {
	var addr = c.stack.pop()
	var value = c.stack.pop()

	offset, overflow := addr.Uint64WithOverflow()
	if overflow {
		c.status = ERROR
		return
	}
	if err := c.memory.SetWord(offset, value, c); err != nil {
		c.SignalError(err)
	}
}

func opMstore8(c *context) {
	var addr = c.stack.pop()
	var value = c.stack.pop()

	offset, overflow := addr.Uint64WithOverflow()
	if overflow {
		c.status = ERROR
		return
	}
	if err := c.memory.SetByte(offset, byte(value.Uint64()), c); err != nil {
		c.SignalError(err)
	}
}

func opMcopy(c *context) {

	if !c.isCancun() {
		c.status = INVALID_INSTRUCTION
		c.gas = 0
		return
	}

	var destAddr = c.stack.pop()
	var srcAddr = c.stack.pop()
	var sizeU256 = c.stack.pop()

	if sizeU256.IsZero() {
		// zero size skips expansions although offset may be off-bounds
		return
	}

	destOffset, destOverflow := destAddr.Uint64WithOverflow()
	srcOffset, srcOverflow := srcAddr.Uint64WithOverflow()
	if destOverflow || srcOverflow || !sizeU256.IsUint64() {
		c.status = ERROR
		return
	}

	size := sizeU256.Uint64()
	price := tosca.Gas(3 * tosca.SizeInWords(size))
	if !c.UseGas(price) {
		return
	}

	data, err := c.memory.GetSliceWithCapacityAndGas(srcOffset, size, c)
	if err != nil {
		return
	}
	if err := c.memory.SetWithCapacityAndGasCheck(destOffset, size, data, c); err != nil {
		return
	}
}

func opMload(c *context) {
	var trg = c.stack.peek()
	var addr = *trg

	if !addr.IsUint64() {
		c.SignalError(errGasUintOverflow)
		return
	}
	offset := addr.Uint64()
	if err := c.memory.CopyWord(offset, trg, c); err != nil {
		c.SignalError(err)
	}
}

func opMsize(c *context) {
	c.stack.pushEmpty().SetUint64(uint64(c.memory.Len()))
}

func opSstore(c *context) {
	gasfunc := gasSStore
	if c.isBerlin() {
		gasfunc = gasSStoreEIP2929
	}

	// Charge the gas price for this operation
	price, err := gasfunc(c)
	if err != nil || !c.UseGas(price) {
		return
	}

	var key = tosca.Key(c.stack.pop().Bytes32())
	var value = tosca.Word(c.stack.pop().Bytes32())

	// Perform storage update
	c.context.SetStorage(c.params.Recipient, key, value)
}

func opSload(c *context) {
	var top = c.stack.peek()

	slot := tosca.Key(top.Bytes32())
	if c.isBerlin() {
		// Check slot presence in the access list
		//lint:ignore SA1019 deprecated functions to be migrated in #616
		if _, slotPresent := c.context.IsSlotInAccessList(c.params.Recipient, slot); !slotPresent {
			// If the caller cannot afford the cost, this change will be rolled back
			// If he does afford it, we can skip checking the same thing later on, during execution
			c.context.AccessStorage(c.params.Recipient, slot)
			if !c.UseGas(tosca.Gas(params.ColdSloadCostEIP2929)) {
				return
			}
		} else {
			if !c.UseGas(tosca.Gas(params.WarmStorageReadCostEIP2929)) {
				return
			}
		}
	}
	value := c.context.GetStorage(c.params.Recipient, slot)
	top.SetBytes32(value[:])
}

func opTstore(c *context) {
	if !c.isCancun() {
		c.status = INVALID_INSTRUCTION
		return
	}

	key := tosca.Key(c.stack.pop().Bytes32())
	value := tosca.Word(c.stack.pop().Bytes32())
	c.context.SetTransientStorage(c.params.Recipient, key, value)
}

func opTload(c *context) {
	if !c.isCancun() {
		c.status = INVALID_INSTRUCTION
		return
	}

	top := c.stack.peek()
	key := tosca.Key(top.Bytes32())
	value := c.context.GetTransientStorage(c.params.Recipient, key)
	top.SetBytes32(value[:])
}

func opCaller(c *context) {
	c.stack.pushEmpty().SetBytes20(c.params.Sender[:])
}

func opCallvalue(c *context) {
	c.stack.pushEmpty().SetBytes32(c.params.Value[:])
}

func opCallDatasize(c *context) {
	size := len(c.params.Input)
	c.stack.pushEmpty().SetUint64(uint64(size))
}

func opCallDataload(c *context) {
	top := c.stack.peek()
	if !top.IsUint64() {
		top.Clear()
		return
	}

	offset := top.Uint64()
	input := c.params.Input
	var value [32]byte
	for i := 0; i < 32; i++ {
		pos := i + int(offset)
		if pos < 0 {
			top.Clear()
			return
		}
		if pos < len(input) {
			value[i] = input[pos]
		} else {
			value[i] = 0
		}
	}
	top.SetBytes(value[:])
}

func opCallDataCopy(c *context) {
	var (
		memOffset  = c.stack.pop()
		dataOffset = c.stack.pop()
		length     = c.stack.pop()
	)
	dataOffset64, overflow := dataOffset.Uint64WithOverflow()
	if overflow {
		dataOffset64 = 0xffffffffffffffff
	}

	memOffset64, overflow := memOffset.Uint64WithOverflow()
	if overflow {
		memOffset64 = 0xffffffffffffffff
	}

	length64, overflow := length.Uint64WithOverflow()
	if overflow || length64+31 < length64 {
		c.status = OUT_OF_GAS
		return
	}

	// Charge for the copy costs
	words := tosca.SizeInWords(length64)
	price := tosca.Gas(3 * words)
	if !c.UseGas(price) {
		return
	}

	if c.memory.EnsureCapacity(memOffset64, length64, c) != nil {
		return
	}

	if err := c.memory.Set(memOffset64, length64, getData(c.params.Input, dataOffset64, length64)); err != nil {
		c.SignalError(err)
	}
}

func opAnd(c *context) {
	a := c.stack.pop()
	b := c.stack.peek()
	b.And(a, b)
}

func opOr(c *context) {
	a := c.stack.pop()
	b := c.stack.peek()
	b.Or(a, b)
}

func opNot(c *context) {
	a := c.stack.peek()
	a.Not(a)
}

func opXor(c *context) {
	a := c.stack.pop()
	b := c.stack.peek()
	b.Xor(a, b)
}

func opIszero(c *context) {
	top := c.stack.peek()
	if top.IsZero() {
		top.SetOne()
	} else {
		top.Clear()
	}
}

func opEq(c *context) {
	a := c.stack.pop()
	b := c.stack.peek()
	res := a.Cmp(b)
	for i := range b {
		b[i] = 0
	}
	if res == 0 {
		b[0] = 1
	} else {
		b[0] = 0
	}
}

func opLt(c *context) {
	a := c.stack.pop()
	b := c.stack.peek()
	if a.Lt(b) {
		b.SetOne()
	} else {
		b.Clear()
	}
}

func opGt(c *context) {
	a := c.stack.pop()
	b := c.stack.peek()
	if a.Gt(b) {
		b.SetOne()
	} else {
		b.Clear()
	}
}

func opSlt(c *context) {
	a := c.stack.pop()
	b := c.stack.peek()
	if a.Slt(b) {
		b.SetOne()
	} else {
		b.Clear()
	}
}
func opSgt(c *context) {
	a := c.stack.pop()
	b := c.stack.peek()
	if a.Sgt(b) {
		b.SetOne()
	} else {
		b.Clear()
	}
}

func opShr(c *context) {
	a := c.stack.pop()
	b := c.stack.peek()
	if a.LtUint64(256) {
		b.Rsh(b, uint(a.Uint64()))
	} else {
		b.Clear()
	}
}

func opShl(c *context) {
	a := c.stack.pop()
	b := c.stack.peek()
	if a.LtUint64(256) {
		b.Lsh(b, uint(a.Uint64()))
	} else {
		b.Clear()
	}
}

func opSar(c *context) {
	a := c.stack.pop()
	b := c.stack.peek()
	if a.GtUint64(256) {
		if b.Sign() >= 0 {
			b.Clear()
		} else {
			b.SetAllOne()
		}
		return
	}
	b.SRsh(b, uint(a.Uint64()))
}

func opSignExtend(c *context) {
	back, num := c.stack.pop(), c.stack.peek()
	num.ExtendSign(num, back)
}

func opByte(c *context) {
	th, val := c.stack.pop(), c.stack.peek()
	val.Byte(th)
}

func opAdd(c *context) {
	a := c.stack.pop()
	b := c.stack.peek()
	b.Add(a, b)
}

func opSub(c *context) {
	a := c.stack.pop()
	b := c.stack.peek()
	b.Sub(a, b)
}

func opMul(c *context) {
	a := c.stack.pop()
	b := c.stack.peek()
	b.Mul(a, b)
}

func opMulMod(c *context) {
	a := c.stack.pop()
	b := c.stack.pop()
	n := c.stack.peek()
	n.MulMod(a, b, n)
}

func opDiv(c *context) {
	a := c.stack.pop()
	b := c.stack.peek()
	b.Div(a, b)
}

func opSDiv(c *context) {
	a := c.stack.pop()
	b := c.stack.peek()
	b.SDiv(a, b)
}

func opMod(c *context) {
	a := c.stack.pop()
	b := c.stack.peek()
	b.Mod(a, b)
}

func opAddMod(c *context) {
	a := c.stack.pop()
	b := c.stack.pop()
	n := c.stack.peek()
	n.AddMod(a, b, n)
}

func opSMod(c *context) {
	a := c.stack.pop()
	b := c.stack.peek()
	b.SMod(a, b)
}

func opExp(c *context) {
	base, exponent := c.stack.pop(), c.stack.peek()
	if !c.UseGas(tosca.Gas(50 * exponent.ByteLen())) {
		return
	}
	exponent.Exp(base, exponent)
}

// Evaluations show a 96% hit rate of this configuration.
var hashCache = newHashCache(1<<16, 1<<18)

func opSha3(c *context) {
	offset, size := c.stack.pop(), c.stack.peek()

	if err := checkSizeOffsetUint64Overflow(offset, size); err != nil {
		c.SignalError(err)
		return
	}

	data, err := c.memory.GetSliceWithCapacityAndGas(offset.Uint64(), size.Uint64(), c)
	if err != nil {
		return
	}

	// charge dynamic gas price
	words := tosca.SizeInWords(size.Uint64())
	price := tosca.Gas(6 * words)
	if !c.UseGas(price) {
		return
	}
	if c.shaCache {
		// Cache hashes since identical values are frequently re-hashed.
		c.hasherBuf = hashCache.hash(c, data)
	} else {
		if c.hasher == nil {
			c.hasher = sha3.NewLegacyKeccak256().(keccakState)
		} else {
			c.hasher.Reset()
		}
		_, _ = c.hasher.Write(data)          // hash.Hash.Write() never returns an error
		_, _ = c.hasher.Read(c.hasherBuf[:]) // sha3.state.Read() never returns an error
	}

	size.SetBytes32(c.hasherBuf[:])
}

func opGas(c *context) {
	c.stack.pushEmpty().SetUint64(uint64(c.gas))
}

// opPrevRandao / opDifficulty
func opPrevRandao(c *context) {
	prevRandao := c.params.PrevRandao
	c.stack.pushEmpty().SetBytes32(prevRandao[:])
}

func opTimestamp(c *context) {
	time := c.params.Timestamp
	c.stack.pushEmpty().SetUint64(uint64(time))
}

func opNumber(c *context) {
	number := c.params.BlockNumber
	c.stack.pushEmpty().SetUint64(uint64(number))
}

func opCoinbase(c *context) {
	coinbase := c.params.Coinbase
	c.stack.pushEmpty().SetBytes20(coinbase[:])
}

func opGasLimit(c *context) {
	limit := c.params.GasLimit
	c.stack.pushEmpty().SetUint64(uint64(limit))
}

func opGasPrice(c *context) {
	price := c.params.GasPrice
	c.stack.pushEmpty().SetBytes32(price[:])
}

func opBalance(c *context) {
	slot := c.stack.peek()
	address := tosca.Address(slot.Bytes20())
	err := gasEip2929AccountCheck(c, address)
	if err != nil {
		return
	}
	balance := c.context.GetBalance(address)
	slot.SetBytes32(balance[:])
}

func opSelfbalance(c *context) {
	balance := c.context.GetBalance(c.params.Recipient)
	c.stack.pushEmpty().SetBytes32(balance[:])
}

func opBaseFee(c *context) {
	if c.isLondon() {
		fee := c.params.BaseFee
		c.stack.pushEmpty().SetBytes32(fee[:])
	} else {
		c.status = INVALID_INSTRUCTION
		return
	}
}

func opBlobHash(c *context) {
	if !c.isCancun() {
		c.status = INVALID_INSTRUCTION
		return
	}

	index := c.stack.pop()
	blobHashesLength := uint64(len(c.params.BlobHashes))
	if index.IsUint64() && index.Uint64() < blobHashesLength {
		c.stack.pushEmpty().SetBytes32(c.params.BlobHashes[index.Uint64()][:])
	} else {
		c.stack.push(uint256.NewInt(0))
	}
}

func opBlobBaseFee(c *context) {
	if c.isCancun() {
		fee := c.params.BlobBaseFee
		c.stack.pushEmpty().SetBytes32(fee[:])
	} else {
		c.status = INVALID_INSTRUCTION
		return
	}
}

func opSelfdestruct(c *context) {
	gasfunc := gasSelfdestruct
	if c.isBerlin() {
		gasfunc = gasSelfdestructEIP2929
	}
	// even death is not for free
	if !c.UseGas(gasfunc(c)) {
		return
	}
	beneficiary := tosca.Address(c.stack.pop().Bytes20())
	c.context.SelfDestruct(c.params.Recipient, beneficiary)
	c.status = SUICIDED
}

func opChainId(c *context) {
	id := c.params.ChainID
	c.stack.pushEmpty().SetBytes32(id[:])
}

func opBlockhash(c *context) {
	num := c.stack.peek()
	num64, overflow := num.Uint64WithOverflow()

	if overflow {
		num.Clear()
		return
	}
	var upper, lower uint64
	upper = uint64(c.params.BlockNumber)
	if upper < 257 {
		lower = 0
	} else {
		lower = upper - 256
	}
	if num64 >= lower && num64 < upper {
		hash := c.context.GetBlockHash(int64(num64))
		num.SetBytes(hash[:])
	} else {
		num.Clear()
	}
}

func opAddress(c *context) {
	c.stack.pushEmpty().SetBytes20(c.params.Recipient[:])
}

func opOrigin(c *context) {
	origin := c.params.Origin
	c.stack.pushEmpty().SetBytes20(origin[:])
}

func opCodeSize(c *context) {
	size := len(c.params.Code)
	c.stack.pushEmpty().SetUint64(uint64(size))
}

func opCodeCopy(c *context) {
	var (
		memOffset  = c.stack.pop()
		codeOffset = c.stack.pop()
		length     = c.stack.pop()
	)

	if checkSizeOffsetUint64Overflow(memOffset, length) != nil {
		c.SignalError(errGasUintOverflow)
		return
	}

	uint64CodeOffset, overflow := codeOffset.Uint64WithOverflow()
	if overflow {
		uint64CodeOffset = 0xffffffffffffffff
	}

	// Charge for length of copied code
	words := tosca.SizeInWords(length.Uint64())
	if !c.UseGas(tosca.Gas(3 * words)) {
		return
	}

	if c.memory.EnsureCapacity(memOffset.Uint64(), length.Uint64(), c) != nil {
		return
	}
	codeCopy := getData(c.params.Code, uint64CodeOffset, length.Uint64())
	if err := c.memory.Set(memOffset.Uint64(), length.Uint64(), codeCopy); err != nil {
		c.SignalError(err)
	}
}

func opExtcodesize(c *context) {
	top := c.stack.peek()
	addr := tosca.Address(top.Bytes20())
	err := gasEip2929AccountCheck(c, addr)
	if err != nil {
		return
	}
	top.SetUint64(uint64(c.context.GetCodeSize(addr)))
}

func opExtcodehash(c *context) {
	slot := c.stack.peek()
	address := tosca.Address(slot.Bytes20())
	err := gasEip2929AccountCheck(c, address)
	if err != nil {
		return
	}
	if !c.context.AccountExists(address) {
		slot.Clear()
	} else {
		hash := c.context.GetCodeHash(address)
		slot.SetBytes32(hash[:])
	}
}

func checkInitCodeSize(c *context, size *uint256.Int) bool {
	const (
		MaxCodeSize     = 24576           // Maximum bytecode to permit for a contract
		MaxInitCodeSize = 2 * MaxCodeSize // Maximum initcode to permit in a creation transaction and create instructions
		InitCodeWordGas = 2               // Once per word of the init code when creating a contract.
	)

	if !c.isShanghai() {
		return true
	}
	if !size.IsUint64() || size.Uint64() > MaxInitCodeSize {
		c.UseGas(c.gas)
		c.status = MAX_INIT_CODE_SIZE_EXCEEDED
		return false
	}
	if !c.UseGas(tosca.Gas(InitCodeWordGas * tosca.SizeInWords(size.Uint64()))) {
		c.status = OUT_OF_GAS
		return false
	}

	return true
}

func opCreate(c *context) {
	var (
		value  = c.stack.pop()
		offset = c.stack.pop()
		size   = c.stack.pop()
	)
	if err := checkSizeOffsetUint64Overflow(offset, size); err != nil {
		c.SignalError(err)
		return
	}

	if c.memory.EnsureCapacity(offset.Uint64(), size.Uint64(), c) != nil {
		return
	}

	if !checkInitCodeSize(c, size) {
		return
	}

	if !value.IsZero() {
		balance := c.context.GetBalance(c.params.Recipient)
		balanceU256 := new(uint256.Int).SetBytes(balance[:])

		if value.Gt(balanceU256) {
			c.stack.pushEmpty().Clear()
			c.return_data = nil
			return
		}
	}

	input := c.memory.GetSlice(offset.Uint64(), size.Uint64())

	gas := c.gas
	if true /*c.evm.chainRules.IsEIP150*/ {
		gas -= gas / 64
	}

	c.UseGas(gas)

	res, err := c.context.Call(tosca.Create, tosca.CallParameters{
		Sender: c.params.Recipient,
		Value:  tosca.Value(value.Bytes32()),
		Input:  input,
		Gas:    gas,
	})

	c.gas += res.GasLeft
	c.refund += res.GasRefund

	success := c.stack.pushEmpty()
	if !res.Success || err != nil {
		success.Clear()
	} else {
		success.SetBytes20(res.CreatedAddress[:])
	}

	if !res.Success && err == nil {
		c.return_data = res.Output
	} else {
		c.return_data = nil
	}
}

func opCreate2(c *context) {
	var (
		value  = c.stack.pop()
		offset = c.stack.pop()
		size   = c.stack.pop()
		salt   = c.stack.pop()
	)
	if err := checkSizeOffsetUint64Overflow(offset, size); err != nil {
		c.SignalError(err)
		return
	}

	if c.memory.EnsureCapacity(offset.Uint64(), size.Uint64(), c) != nil {
		return
	}

	if !checkInitCodeSize(c, size) {
		return
	}

	// Charge for the code size
	words := tosca.SizeInWords(size.Uint64())
	if !c.UseGas(tosca.Gas(6 * words)) {
		return
	}

	if !value.IsZero() {
		balance := c.context.GetBalance(c.params.Recipient)
		balanceU256 := new(uint256.Int).SetBytes(balance[:])

		if value.Gt(balanceU256) {
			c.stack.pushEmpty().Clear()
			c.return_data = nil
			return
		}
	}

	input := c.memory.GetSlice(offset.Uint64(), size.Uint64())

	// Apply EIP150
	gas := c.gas
	gas -= gas / 64
	if !c.UseGas(gas) {
		return
	}

	res, err := c.context.Call(tosca.Create2, tosca.CallParameters{
		Sender: c.params.Recipient,
		Value:  tosca.Value(value.Bytes32()),
		Input:  input,
		Gas:    gas,
		Salt:   tosca.Hash(salt.Bytes32()),
	})

	// Push item on the stack based on the returned error.
	success := c.stack.pushEmpty()
	if !res.Success || err != nil {
		success.Clear()
	} else {
		success.SetBytes20(res.CreatedAddress[:])
	}

	if !res.Success && err == nil {
		c.return_data = res.Output
	} else {
		c.return_data = nil
	}
	c.gas += res.GasLeft
	c.refund += res.GasRefund
}

func getData(data []byte, start uint64, size uint64) []byte {
	length := uint64(len(data))
	if start > length {
		start = length
	}
	end := start + size
	if end > length {
		end = length
	}
	// Apply some right-padding to the result.
	res := make([]byte, int(size))
	copy(res, data[start:end])
	return res
}

func opExtCodeCopy(c *context) {
	var (
		stack      = c.stack
		a          = stack.pop()
		memOffset  = stack.pop()
		codeOffset = stack.pop()
		length     = stack.pop()
	)
	if checkSizeOffsetUint64Overflow(memOffset, length) != nil {
		c.SignalError(errGasUintOverflow)
		return
	}

	// Charge for length of copied code
	words := tosca.SizeInWords(length.Uint64())
	if !c.UseGas(tosca.Gas(3 * words)) {
		return
	}

	addr := tosca.Address(a.Bytes20())
	err := gasEip2929AccountCheck(c, addr)
	if err != nil {
		return
	}
	var uint64CodeOffset uint64
	if codeOffset.IsUint64() {
		uint64CodeOffset = codeOffset.Uint64()
	} else {
		uint64CodeOffset = math.MaxUint64
	}

	if c.memory.EnsureCapacity(memOffset.Uint64(), length.Uint64(), c) != nil {
		return
	}
	codeCopy := getData(c.context.GetCode(addr), uint64CodeOffset, length.Uint64())
	if err = c.memory.Set(memOffset.Uint64(), length.Uint64(), codeCopy); err != nil {
		c.SignalError(err)
	}
}

func checkSizeOffsetUint64Overflow(offset, size *uint256.Int) error {
	if size.IsZero() {
		return nil
	}
	if !offset.IsUint64() || !size.IsUint64() || offset.Uint64()+size.Uint64() < offset.Uint64() {
		return errGasUintOverflow
	}
	return nil
}

func neededMemorySize(c *context, offset, size *uint256.Int) (uint64, error) {
	if err := checkSizeOffsetUint64Overflow(offset, size); err != nil {
		c.SignalError(err)
		return 0, err
	}
	if size.IsZero() {
		return 0, nil
	}
	return offset.Uint64() + size.Uint64(), nil
}

func genericCall(c *context, kind tosca.CallKind) {
	warmAccess, coldCost, err := addressInAccessList(c)
	if err != nil {
		return
	}
	stack := c.stack
	value := uint256.NewInt(0)

	// Pop call parameters.
	provided_gas, addr := stack.pop(), stack.pop()
	if kind == tosca.Call || kind == tosca.CallCode {
		value = stack.pop()
	}
	inOffset, inSize, retOffset, retSize := stack.pop(), stack.pop(), stack.pop(), stack.pop()

	// We need to check the existence of the target account before removing
	// the gas price for the other cost factors to make sure that the read
	// in the state DB is always happening. This is the current EVM behavior,
	// and not doing it would be identified by the replay tool as an error.
	toAddr := tosca.Address(addr.Bytes20())

	// Compute and charge gas price for call
	arg_memory_size, err := neededMemorySize(c, inOffset, inSize)
	if err != nil {
		return
	}
	ret_memory_size, err := neededMemorySize(c, retOffset, retSize)
	if err != nil {
		return
	}

	needed_memory_size := arg_memory_size
	if ret_memory_size > arg_memory_size {
		needed_memory_size = ret_memory_size
	}

	baseGas := c.memory.ExpansionCosts(needed_memory_size)
	checkGas := func(cost tosca.Gas) bool {
		return 0 <= cost && cost <= c.gas
	}
	if !checkGas(baseGas) {
		c.status = OUT_OF_GAS
		return
	}

	// for static and delegate calls, the following value checks will always be zero.
	// Charge for transferring value to a new address
	if !value.IsZero() {
		baseGas += tosca.Gas(params.CallValueTransferGas)
	}
	if !checkGas(baseGas) {
		c.status = OUT_OF_GAS
		return
	}

	// EIP158 states that non-zero value calls that create a new account should
	// be charged an additional gas fee.
	if kind == tosca.Call && !value.IsZero() && !c.context.AccountExists(toAddr) {
		baseGas += tosca.Gas(params.CallNewAccountGas)
	}
	if !checkGas(baseGas) {
		c.status = OUT_OF_GAS
		return
	}

	cost := callGas(c.gas, baseGas, provided_gas)
	if !warmAccess {
		// In case of a cold access, we temporarily add the cold charge back, and also
		// add it to the returned gas. By adding it to the return, it will be charged
		// outside of this function, as part of the dynamic gas, and that will make it
		// also become correctly reported to tracers.
		c.gas += coldCost
		baseGas += coldCost
	}
	if !c.UseGas(baseGas + cost) {
		return
	}

	// first use static and dynamic gas cost and then resize the memory
	// when out of gas is happening, then mem should not be resized
	c.memory.EnsureCapacityWithoutGas(needed_memory_size)
	if !value.IsZero() {
		cost += tosca.Gas(params.CallStipend)
	}

	// Check that the caller has enough balance to transfer the requested value.
	if (kind == tosca.Call || kind == tosca.CallCode) && !value.IsZero() {
		balance := c.context.GetBalance(c.params.Recipient)
		balanceU256 := new(uint256.Int).SetBytes32(balance[:])
		if balanceU256.Lt(value) {
			c.stack.pushEmpty().Clear()
			c.return_data = nil
			c.gas += cost // the gas send to the nested contract is returned
			return
		}
	}

	// If we are in static mode, recursive calls are to be treated like
	// static calls. This is a consequence of the unification of the
	// interpreter interfaces of EVMC and Geth.
	// This problem was encountered in block 58413779, transaction 7.
	if c.params.Static && kind == tosca.Call {
		kind = tosca.StaticCall
	}

	// Get arguments from the memory.
	args := c.memory.GetSlice(inOffset.Uint64(), inSize.Uint64())

	// Prepare arguments, depending on call kind
	callParams := tosca.CallParameters{
		Input: args,
		Gas:   cost,
		Value: tosca.Value(value.Bytes32()),
	}

	switch kind {
	case tosca.Call, tosca.StaticCall:
		callParams.Sender = c.params.Recipient
		callParams.Recipient = toAddr

	case tosca.CallCode:
		callParams.Sender = c.params.Recipient
		callParams.Recipient = c.params.Recipient
		callParams.CodeAddress = toAddr

	case tosca.DelegateCall:
		callParams.Sender = c.params.Sender
		callParams.Recipient = c.params.Recipient
		callParams.CodeAddress = toAddr
		callParams.Value = c.params.Value
	}

	// Perform the call.
	ret, err := c.context.Call(kind, callParams)

	if err == nil {
		if memSetErr := c.memory.Set(retOffset.Uint64(), retSize.Uint64(), ret.Output); memSetErr != nil {
			c.SignalError(memSetErr)
		}
	}

	success := stack.pushEmpty()
	if err != nil || !ret.Success {
		success.Clear()
	} else {
		success.SetOne()
	}
	c.gas += ret.GasLeft
	c.refund += ret.GasRefund
	c.return_data = ret.Output
}

func opCall(c *context) {
	value := c.stack.data[c.stack.stack_ptr-3]
	// In a static call, no value must be transferred.
	if c.params.Static && !value.IsZero() {
		c.SignalError(errWriteProtection)
		return
	}
	genericCall(c, tosca.Call)
}

func opCallCode(c *context) {
	genericCall(c, tosca.CallCode)
}

func opStaticCall(c *context) {
	genericCall(c, tosca.StaticCall)
}

func opDelegateCall(c *context) {
	genericCall(c, tosca.DelegateCall)
}

func opReturnDataSize(c *context) {
	c.stack.pushEmpty().SetUint64(uint64(len(c.return_data)))
}

func opReturnDataCopy(c *context) {
	var (
		memOffset  = c.stack.pop()
		dataOffset = c.stack.pop()
		length     = c.stack.pop()
	)

	offset64, overflow := dataOffset.Uint64WithOverflow()
	if overflow {
		c.SignalError(errReturnDataOutOfBounds)
		return
	}
	// we can reuse dataOffset now (aliasing it for clarity)
	var end = dataOffset
	end.Add(dataOffset, length)
	end64, overflow := end.Uint64WithOverflow()
	if overflow || uint64(len(c.return_data)) < end64 {
		c.SignalError(errReturnDataOutOfBounds)
		return
	}

	if err := checkSizeOffsetUint64Overflow(memOffset, length); err != nil {
		c.SignalError(err)
		return
	}

	words := tosca.SizeInWords(length.Uint64())
	if !c.UseGas(tosca.Gas(3 * words)) {
		return
	}

	if err := c.memory.SetWithCapacityAndGasCheck(memOffset.Uint64(), length.Uint64(), c.return_data[offset64:end64], c); err != nil {
		c.SignalError(err)
	}
}

func opLog(c *context, size int) {
	topics := make([]tosca.Hash, size)
	stack := c.stack
	mStart, mSize := stack.pop(), stack.pop()

	if err := checkSizeOffsetUint64Overflow(mStart, mSize); err != nil {
		c.SignalError(err)
		return
	}

	for i := 0; i < size; i++ {
		addr := stack.pop()
		topics[i] = addr.Bytes32()
	}

	// Expand memory if needed
	start := mStart.Uint64()
	log_size := mSize.Uint64()

	// charge for log size
	if !c.UseGas(tosca.Gas(8 * log_size)) {
		return
	}

	d, err := c.memory.GetSliceWithCapacityAndGas(start, log_size, c)
	if err != nil {
		return
	}

	// make a copy of the data to disconnect from memory
	log_data := bytes.Clone(d)
	c.context.EmitLog(tosca.Log{
		Address: c.params.Recipient,
		Topics:  topics,
		Data:    log_data,
	})
}
