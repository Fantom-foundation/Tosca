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
	"github.com/holiman/uint256"
)

func opStop(c *context) {
	c.status = statusStopped
}

func opRevert(c *context) {
	c.resultOffset = *c.stack.pop()
	c.resultSize = *c.stack.pop()
	c.status = statusReverted
}

func opReturn(c *context) {
	c.resultOffset = *c.stack.pop()
	c.resultSize = *c.stack.pop()
	c.status = statusReturned
}

func opPc(c *context) {
	c.stack.pushUndefined().SetUint64(uint64(c.code[c.pc].arg))
}

func checkJumpDest(c *context) {
	if int(c.pc+1) >= len(c.code) || c.code[c.pc+1].opcode != JUMPDEST {
		c.signalError()
	}
}

func opJump(c *context) {
	destination := c.stack.pop()
	// overflow check
	if !destination.IsUint64() || destination.Uint64() > math.MaxInt32 {
		c.signalError()
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
			c.signalError()
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
	z := c.stack.pushUndefined()
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
	if c.isAtLeast(tosca.R12_Shanghai) {
		z := c.stack.pushUndefined()
		z[3], z[2], z[1], z[0] = 0, 0, 0, 0
	} else {
		c.status = statusInvalidInstruction
	}
}

func opPush1(c *context) {
	z := c.stack.pushUndefined()
	z[3], z[2], z[1] = 0, 0, 0
	z[0] = uint64(c.code[c.pc].arg >> 8)
}

func opPush2(c *context) {
	z := c.stack.pushUndefined()
	z[3], z[2], z[1] = 0, 0, 0
	z[0] = uint64(c.code[c.pc].arg)
}

func opPush3(c *context) {
	z := c.stack.pushUndefined()
	z[3], z[2], z[1] = 0, 0, 0
	data := c.code[c.pc : c.pc+2]
	_ = data[1]
	z[0] = uint64(data[0].arg)<<8 | uint64(data[1].arg>>8)
	c.pc += 1
}

func opPush4(c *context) {
	z := c.stack.pushUndefined()
	z[3], z[2], z[1] = 0, 0, 0

	data := c.code[c.pc : c.pc+2]
	_ = data[1] // causes bound check to be performed only once (may become unneeded in the future)
	z[0] = (uint64(data[0].arg) << 16) | uint64(data[1].arg)
	c.pc += 1
}

func opPush32(c *context) {
	z := c.stack.pushUndefined()

	data := c.code[c.pc : c.pc+16]
	_ = data[15] // causes bound check to be performed only once (may become unneded in the future)
	z[3] = (uint64(data[0].arg) << 48) | (uint64(data[1].arg) << 32) | (uint64(data[2].arg) << 16) | uint64(data[3].arg)
	z[2] = (uint64(data[4].arg) << 48) | (uint64(data[5].arg) << 32) | (uint64(data[6].arg) << 16) | uint64(data[7].arg)
	z[1] = (uint64(data[8].arg) << 48) | (uint64(data[9].arg) << 32) | (uint64(data[10].arg) << 16) | uint64(data[11].arg)
	z[0] = (uint64(data[12].arg) << 48) | (uint64(data[13].arg) << 32) | (uint64(data[14].arg) << 16) | uint64(data[15].arg)
	c.pc += 15
}

func opDup(c *context, pos int) {
	c.stack.dup(pos - 1)
}

func opSwap(c *context, pos int) {
	c.stack.swap(pos)
}

func opMstore(c *context) {
	var addr = c.stack.pop()
	var value = c.stack.pop()

	offset, overflow := addr.Uint64WithOverflow()
	if overflow {
		c.status = statusError
		return
	}
	if c.memory.setWord(offset, value, c) != nil {
		c.signalError()
	}
}

func opMstore8(c *context) {
	var addr = c.stack.pop()
	var value = c.stack.pop()

	offset, overflow := addr.Uint64WithOverflow()
	if overflow {
		c.status = statusError
		return
	}
	if c.memory.setByte(offset, byte(value.Uint64()), c) != nil {
		c.signalError()
	}
}

func opMcopy(c *context) {

	if !c.isAtLeast(tosca.R13_Cancun) {
		c.status = statusInvalidInstruction
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
		c.status = statusError
		return
	}

	size := sizeU256.Uint64()
	price := tosca.Gas(3 * tosca.SizeInWords(size))
	if !c.useGas(price) {
		return
	}

	data, err := c.memory.GetSliceWithCapacityAndGas(srcOffset, size, c)
	if err != nil {
		return
	}
	if c.memory.set(destOffset, size, data, c) != nil {
		return
	}
}

func opMload(c *context) {
	var trg = c.stack.peek()
	var addr = *trg

	if !addr.IsUint64() {
		c.signalError()
		return
	}
	offset := addr.Uint64()
	if err := c.memory.CopyWord(offset, trg, c); err != nil {
		c.signalError()
	}
}

func opMsize(c *context) {
	c.stack.pushUndefined().SetUint64(uint64(c.memory.length()))
}

func opSstore(c *context) {

	// SStore is a write instruction, it shall not be executed in static mode.
	if c.params.Static {
		c.signalError()
		return
	}

	// EIP-2200 demands that at least 2300 gas is available for SSTORE
	if c.gas <= 2300 {
		c.signalError()
		return
	}

	var key = tosca.Key(c.stack.pop().Bytes32())
	var value = tosca.Word(c.stack.pop().Bytes32())

	cost := tosca.Gas(0)
	if c.isAtLeast(tosca.R09_Berlin) &&
		c.context.AccessStorage(c.params.Recipient, key) == tosca.ColdAccess {
		cost += 2100
	}

	storageStatus := c.context.SetStorage(c.params.Recipient, key, value)

	cost += getDynamicCostsForSstore(c.params.Revision, storageStatus)
	if !c.useGas(cost) {
		return
	}

	c.refund += getRefundForSstore(c.params.Revision, storageStatus)
}

func opSload(c *context) {
	var top = c.stack.peek()

	addr := c.params.Recipient
	slot := tosca.Key(top.Bytes32())
	if c.isAtLeast(tosca.R09_Berlin) {
		// charge costs for warm/cold slot access
		costs := tosca.Gas(100)
		if c.context.AccessStorage(addr, slot) == tosca.ColdAccess {
			costs = 2100
		}
		if !c.useGas(costs) {
			return
		}
	}
	value := c.context.GetStorage(addr, slot)
	top.SetBytes32(value[:])
}

func opTstore(c *context) {

	// Although not mentioned in the yellow paper, nor in CALL description at
	// website (https://www.evm.codes/#FA) Geth treats this Op as a write instruction.
	// therefore it shall not be executed in static mode.
	if c.params.Static {
		c.signalError()
		return
	}

	if !c.isAtLeast(tosca.R13_Cancun) {
		c.status = statusInvalidInstruction
		return
	}

	key := tosca.Key(c.stack.pop().Bytes32())
	value := tosca.Word(c.stack.pop().Bytes32())
	c.context.SetTransientStorage(c.params.Recipient, key, value)
}

func opTload(c *context) {
	if !c.isAtLeast(tosca.R13_Cancun) {
		c.status = statusInvalidInstruction
		return
	}

	top := c.stack.peek()
	key := tosca.Key(top.Bytes32())
	value := c.context.GetTransientStorage(c.params.Recipient, key)
	top.SetBytes32(value[:])
}

func opCaller(c *context) {
	c.stack.pushUndefined().SetBytes20(c.params.Sender[:])
}

func opCallvalue(c *context) {
	c.stack.pushUndefined().SetBytes32(c.params.Value[:])
}

func opCallDatasize(c *context) {
	size := len(c.params.Input)
	c.stack.pushUndefined().SetUint64(uint64(size))
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
		c.status = statusOutOfGas
		return
	}

	// Charge for the copy costs
	words := tosca.SizeInWords(length64)
	price := tosca.Gas(3 * words)
	if !c.useGas(price) {
		return
	}

	if c.memory.expandMemory(memOffset64, length64, c) != nil {
		return
	}

	if c.memory.trySet(memOffset64, length64, getData(c.params.Input, dataOffset64, length64)) != nil {
		c.signalError()
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
	if !c.useGas(tosca.Gas(50 * exponent.ByteLen())) {
		return
	}
	exponent.Exp(base, exponent)
}

// Evaluations show a 96% hit rate of this configuration.
var sha3Cache = newSha3HashCache(1<<16, 1<<18)

func opSha3(c *context) {
	offset, size := c.stack.pop(), c.stack.peek()

	if err := checkSizeOffsetUint64Overflow(offset, size); err != nil {
		c.signalError()
		return
	}

	data, err := c.memory.GetSliceWithCapacityAndGas(offset.Uint64(), size.Uint64(), c)
	if err != nil {
		return
	}

	// charge dynamic gas price
	words := tosca.SizeInWords(size.Uint64())
	price := tosca.Gas(6 * words)
	if !c.useGas(price) {
		return
	}
	var hash tosca.Hash
	if c.withShaCache {
		// Cache hashes since identical values are frequently re-hashed.
		hash = sha3Cache.hash(data)
	} else {
		hash = Keccak256(data)
	}

	size.SetBytes32(hash[:])
}

func opGas(c *context) {
	c.stack.pushUndefined().SetUint64(uint64(c.gas))
}

// opPrevRandao / opDifficulty
func opPrevRandao(c *context) {
	prevRandao := c.params.PrevRandao
	c.stack.pushUndefined().SetBytes32(prevRandao[:])
}

func opTimestamp(c *context) {
	time := c.params.Timestamp
	c.stack.pushUndefined().SetUint64(uint64(time))
}

func opNumber(c *context) {
	number := c.params.BlockNumber
	c.stack.pushUndefined().SetUint64(uint64(number))
}

func opCoinbase(c *context) {
	coinbase := c.params.Coinbase
	c.stack.pushUndefined().SetBytes20(coinbase[:])
}

func opGasLimit(c *context) {
	limit := c.params.GasLimit
	c.stack.pushUndefined().SetUint64(uint64(limit))
}

func opGasPrice(c *context) {
	price := c.params.GasPrice
	c.stack.pushUndefined().SetBytes32(price[:])
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
	c.stack.pushUndefined().SetBytes32(balance[:])
}

func opBaseFee(c *context) {
	if c.isAtLeast(tosca.R10_London) {
		fee := c.params.BaseFee
		c.stack.pushUndefined().SetBytes32(fee[:])
	} else {
		c.status = statusInvalidInstruction
		return
	}
}

func opBlobHash(c *context) {
	if !c.isAtLeast(tosca.R13_Cancun) {
		c.status = statusInvalidInstruction
		return
	}

	index := c.stack.pop()
	blobHashesLength := uint64(len(c.params.BlobHashes))
	if index.IsUint64() && index.Uint64() < blobHashesLength {
		c.stack.pushUndefined().SetBytes32(c.params.BlobHashes[index.Uint64()][:])
	} else {
		c.stack.push(uint256.NewInt(0))
	}
}

func opBlobBaseFee(c *context) {
	if c.isAtLeast(tosca.R13_Cancun) {
		fee := c.params.BlobBaseFee
		c.stack.pushUndefined().SetBytes32(fee[:])
	} else {
		c.status = statusInvalidInstruction
		return
	}
}

func opSelfdestruct(c *context) {

	// SelfDestruct is a write instruction, it shall not be executed in static mode.
	if c.params.Static {
		c.signalError()
		return
	}

	beneficiary := tosca.Address(c.stack.pop().Bytes20())
	// Selfdestruct gas cost defined in EIP-105 (see https://eips.ethereum.org/EIPS/eip-150)
	cost := tosca.Gas(0)
	if c.isAtLeast(tosca.R09_Berlin) {
		// as https://eips.ethereum.org/EIPS/eip-2929#selfdestruct-changes says,
		// selfdestruct does not charge for warm access
		if accessStatus := c.context.AccessAccount(beneficiary); accessStatus != tosca.WarmAccess {
			cost += getAccessCost(accessStatus)
		}
	}
	cost += selfDestructNewAccountCost(c.context.AccountExists(beneficiary),
		c.context.GetBalance(c.params.Recipient))
	// even death is not for free
	if !c.useGas(cost) {
		return
	}

	destructed := c.context.SelfDestruct(c.params.Recipient, beneficiary)
	c.refund += selfDestructRefund(destructed, c.params.Revision)
	c.status = statusSelfDestructed
}

func selfDestructNewAccountCost(accountExists bool, balance tosca.Value) tosca.Gas {
	if !accountExists && balance != (tosca.Value{}) {
		// cost of creating an account defined in eip-150 (see https://eips.ethereum.org/EIPS/eip-150)
		// CreateBySelfdestructGas is used when the refunded account is one that does
		// not exist. This logic is similar to call.
		return 25_000
	}
	return 0
}

func selfDestructRefund(destructed bool, revision tosca.Revision) tosca.Gas {
	// Since London and after there is no more refund (see https://eips.ethereum.org/EIPS/eip-3529)
	if destructed && revision < tosca.R10_London {
		return 24_000
	}
	return 0
}

func opChainId(c *context) {
	id := c.params.ChainID
	c.stack.pushUndefined().SetBytes32(id[:])
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
	c.stack.pushUndefined().SetBytes20(c.params.Recipient[:])
}

func opOrigin(c *context) {
	origin := c.params.Origin
	c.stack.pushUndefined().SetBytes20(origin[:])
}

func opCodeSize(c *context) {
	size := len(c.params.Code)
	c.stack.pushUndefined().SetUint64(uint64(size))
}

func opCodeCopy(c *context) {
	var (
		memOffset  = c.stack.pop()
		codeOffset = c.stack.pop()
		length     = c.stack.pop()
	)

	if checkSizeOffsetUint64Overflow(memOffset, length) != nil {
		c.signalError()
		return
	}

	uint64CodeOffset, overflow := codeOffset.Uint64WithOverflow()
	if overflow {
		uint64CodeOffset = 0xffffffffffffffff
	}

	// Charge for length of copied code
	words := tosca.SizeInWords(length.Uint64())
	if !c.useGas(tosca.Gas(3 * words)) {
		return
	}

	if c.memory.expandMemory(memOffset.Uint64(), length.Uint64(), c) != nil {
		return
	}
	codeCopy := getData(c.params.Code, uint64CodeOffset, length.Uint64())
	if c.memory.trySet(memOffset.Uint64(), length.Uint64(), codeCopy) != nil {
		c.signalError()
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

	if !c.isAtLeast(tosca.R12_Shanghai) {
		return true
	}
	if !size.IsUint64() || size.Uint64() > MaxInitCodeSize {
		c.useGas(c.gas)
		c.signalError()
		return false
	}
	if !c.useGas(tosca.Gas(InitCodeWordGas * tosca.SizeInWords(size.Uint64()))) {
		c.status = statusOutOfGas
		return false
	}

	return true
}

func opCreate(c *context) {

	// Create is a write instruction, it shall not be executed in static mode.
	if c.params.Static {
		c.signalError()
		return
	}

	var (
		value  = c.stack.pop()
		offset = c.stack.pop()
		size   = c.stack.pop()
	)
	if err := checkSizeOffsetUint64Overflow(offset, size); err != nil {
		c.signalError()
		return
	}

	if c.memory.expandMemory(offset.Uint64(), size.Uint64(), c) != nil {
		return
	}

	if !checkInitCodeSize(c, size) {
		return
	}

	if !value.IsZero() {
		balance := c.context.GetBalance(c.params.Recipient)
		balanceU256 := new(uint256.Int).SetBytes(balance[:])

		if value.Gt(balanceU256) {
			c.stack.pushUndefined().Clear()
			c.returnData = nil
			return
		}
	}

	input := c.memory.GetSlice(offset.Uint64(), size.Uint64())

	gas := c.gas
	if true /*c.evm.chainRules.IsEIP150*/ {
		gas -= gas / 64
	}

	c.useGas(gas)

	res, err := c.context.Call(tosca.Create, tosca.CallParameters{
		Sender: c.params.Recipient,
		Value:  tosca.Value(value.Bytes32()),
		Input:  input,
		Gas:    gas,
	})

	c.gas += res.GasLeft
	c.refund += res.GasRefund

	success := c.stack.pushUndefined()
	if !res.Success || err != nil {
		success.Clear()
	} else {
		success.SetBytes20(res.CreatedAddress[:])
	}

	if !res.Success && err == nil {
		c.returnData = res.Output
	} else {
		c.returnData = nil
	}
}

func opCreate2(c *context) {

	// Create2 is a write instruction, it shall not be executed in static mode.
	if c.params.Static {
		c.signalError()
		return
	}

	var (
		value  = c.stack.pop()
		offset = c.stack.pop()
		size   = c.stack.pop()
		salt   = c.stack.pop()
	)
	if err := checkSizeOffsetUint64Overflow(offset, size); err != nil {
		c.signalError()
		return
	}

	if c.memory.expandMemory(offset.Uint64(), size.Uint64(), c) != nil {
		return
	}

	if !checkInitCodeSize(c, size) {
		return
	}

	// Charge for the code size
	words := tosca.SizeInWords(size.Uint64())
	if !c.useGas(tosca.Gas(6 * words)) {
		return
	}

	if !value.IsZero() {
		balance := c.context.GetBalance(c.params.Recipient)
		balanceU256 := new(uint256.Int).SetBytes(balance[:])

		if value.Gt(balanceU256) {
			c.stack.pushUndefined().Clear()
			c.returnData = nil
			return
		}
	}

	input := c.memory.GetSlice(offset.Uint64(), size.Uint64())

	// Apply EIP150
	gas := c.gas
	gas -= gas / 64
	if !c.useGas(gas) {
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
	success := c.stack.pushUndefined()
	if !res.Success || err != nil {
		success.Clear()
	} else {
		success.SetBytes20(res.CreatedAddress[:])
	}

	if !res.Success && err == nil {
		c.returnData = res.Output
	} else {
		c.returnData = nil
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
		c.signalError()
		return
	}

	// Charge for length of copied code
	words := tosca.SizeInWords(length.Uint64())
	if !c.useGas(tosca.Gas(3 * words)) {
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

	if c.memory.expandMemory(memOffset.Uint64(), length.Uint64(), c) != nil {
		return
	}
	codeCopy := getData(c.context.GetCode(addr), uint64CodeOffset, length.Uint64())
	if c.memory.trySet(memOffset.Uint64(), length.Uint64(), codeCopy) != nil {
		c.signalError()
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
		c.signalError()
		return 0, err
	}
	if size.IsZero() {
		return 0, nil
	}
	return offset.Uint64() + size.Uint64(), nil
}

func getAccessCost(accessStatus tosca.AccessStatus) tosca.Gas {
	// EIP-2929 says that cold access cost is 2600 and warm is 100.
	// (https://eips.ethereum.org/EIPS/eip-2929)
	if accessStatus == tosca.ColdAccess {
		return tosca.Gas(2600)
	}
	return tosca.Gas(100)
}

func genericCall(c *context, kind tosca.CallKind) {
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

	baseGas := c.memory.getExpansionCosts(needed_memory_size)
	// from berlin onwards access cost changes depending on warm/cold access.
	if c.isAtLeast(tosca.R09_Berlin) {
		baseGas += getAccessCost(c.context.AccessAccount(toAddr))
	}
	checkGas := func(cost tosca.Gas) bool {
		return 0 <= cost && cost <= c.gas
	}
	if !checkGas(baseGas) {
		c.status = statusOutOfGas
		return
	}

	// for static and delegate calls, the following value checks will always be zero.
	// Charge for transferring value to a new address
	if !value.IsZero() {
		baseGas += CallValueTransferGas
	}
	if !checkGas(baseGas) {
		c.status = statusOutOfGas
		return
	}

	// EIP158 states that non-zero value calls that create a new account should
	// be charged an additional gas fee.
	if kind == tosca.Call && !value.IsZero() && !c.context.AccountExists(toAddr) {
		baseGas += CallNewAccountGas
	}
	if !checkGas(baseGas) {
		c.status = statusOutOfGas
		return
	}

	cost := callGas(c.gas, baseGas, provided_gas)
	if !c.useGas(baseGas + cost) {
		return
	}

	// first use static and dynamic gas cost and then resize the memory
	// when out of gas is happening, then mem should not be resized
	c.memory.expandMemoryWithoutCharging(needed_memory_size)
	if !value.IsZero() {
		cost += CallStipend
	}

	// Check that the caller has enough balance to transfer the requested value.
	if (kind == tosca.Call || kind == tosca.CallCode) && !value.IsZero() {
		balance := c.context.GetBalance(c.params.Recipient)
		balanceU256 := new(uint256.Int).SetBytes32(balance[:])
		if balanceU256.Lt(value) {
			c.stack.pushUndefined().Clear()
			c.returnData = nil
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
		if c.memory.trySet(retOffset.Uint64(), retSize.Uint64(), ret.Output) != nil {
			c.signalError()
		}
	}

	success := stack.pushUndefined()
	if err != nil || !ret.Success {
		success.Clear()
	} else {
		success.SetOne()
	}
	c.gas += ret.GasLeft
	c.refund += ret.GasRefund
	c.returnData = ret.Output
}

func opCall(c *context) {
	value := c.stack.peekN(2)
	// In a static call, no value must be transferred.
	if c.params.Static && !value.IsZero() {
		c.signalError()
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
	c.stack.pushUndefined().SetUint64(uint64(len(c.returnData)))
}

func opReturnDataCopy(c *context) {
	var (
		memOffset  = c.stack.pop()
		dataOffset = c.stack.pop()
		length     = c.stack.pop()
	)

	offset64, overflow := dataOffset.Uint64WithOverflow()
	if overflow {
		c.signalError()
		return
	}
	// we can reuse dataOffset now (aliasing it for clarity)
	var end = dataOffset
	end.Add(dataOffset, length)
	end64, overflow := end.Uint64WithOverflow()
	if overflow || uint64(len(c.returnData)) < end64 {
		c.signalError()
		return
	}

	if err := checkSizeOffsetUint64Overflow(memOffset, length); err != nil {
		c.signalError()
		return
	}

	words := tosca.SizeInWords(length.Uint64())
	if !c.useGas(tosca.Gas(3 * words)) {
		return
	}

	if c.memory.set(memOffset.Uint64(), length.Uint64(), c.returnData[offset64:end64], c) != nil {
		c.signalError()
	}
}

func opLog(c *context, size int) {

	// LogN op codes are write instructions, they shall not be executed in static mode.
	if c.params.Static {
		c.signalError()
		return
	}

	topics := make([]tosca.Hash, size)
	stack := c.stack
	mStart, mSize := stack.pop(), stack.pop()

	if err := checkSizeOffsetUint64Overflow(mStart, mSize); err != nil {
		c.signalError()
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
	if !c.useGas(tosca.Gas(8 * log_size)) {
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
