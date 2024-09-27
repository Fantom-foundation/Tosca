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

func opStop() status {
	return statusStopped
}

func opEndWithResult(c *context) error {
	offset := *c.stack.pop()
	size := *c.stack.pop()
	if err := checkSizeOffsetUint64Overflow(&offset, &size); err != nil {
		return err
	}
	var err error
	c.returnData, err = c.memory.getSlice(offset.Uint64(), size.Uint64(), c)
	return err
}

func opPc(c *context) {
	c.stack.pushUndefined().SetUint64(uint64(c.code[c.pc].arg))
}

func checkJumpDest(c *context) error {
	if int(c.pc+1) >= len(c.code) || c.code[c.pc+1].opcode != JUMPDEST {
		return errInvalidJump
	}
	return nil
}

func opJump(c *context) error {
	destination := c.stack.pop()
	// overflow check
	if !destination.IsUint64() || destination.Uint64() > math.MaxInt32 {
		return errOverflow
	}
	// Update the PC to the jump destination -1 since interpreter will increase PC by 1 afterward.
	c.pc = int32(destination.Uint64()) - 1
	return checkJumpDest(c)
}

func opJumpi(c *context) error {
	destination := c.stack.pop()
	condition := c.stack.pop()
	if !condition.IsZero() {
		// overflow check
		if !destination.IsUint64() || destination.Uint64() > math.MaxInt32 {
			return errOverflow
		}
		// Update the PC to the jump destination -1 since interpreter will increase PC by 1 afterward.
		c.pc = int32(destination.Uint64()) - 1
		return checkJumpDest(c)
	}
	return nil
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

func opPush0(c *context) error {
	if !c.isAtLeast(tosca.R12_Shanghai) {
		return errInvalidRevision
	}
	z := c.stack.pushUndefined()
	z[3], z[2], z[1], z[0] = 0, 0, 0, 0
	return nil
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
	_ = data[15] // causes bound check to be performed only once (may become unneeded in the future)
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

func opMstore(c *context) error {
	var addr = c.stack.pop()
	var value = c.stack.pop()

	offset, overflow := addr.Uint64WithOverflow()
	if overflow {
		return errOverflow
	}
	data := value.Bytes32()
	return c.memory.set(offset, data[:], c)
}

func opMstore8(c *context) error {
	var addr = c.stack.pop()
	var value = c.stack.pop()

	offset, overflow := addr.Uint64WithOverflow()
	if overflow {
		return errOverflow
	}
	return c.memory.set(offset, []byte{byte(value.Uint64())}, c)
}

func opMcopy(c *context) error {

	if !c.isAtLeast(tosca.R13_Cancun) {
		return errInvalidRevision
	}

	var destAddr = c.stack.pop()
	var srcAddr = c.stack.pop()
	var sizeU256 = c.stack.pop()

	if sizeU256.IsZero() {
		// zero size skips expansions although offset may be off-bounds
		return nil
	}

	destOffset, destOverflow := destAddr.Uint64WithOverflow()
	srcOffset, srcOverflow := srcAddr.Uint64WithOverflow()
	if destOverflow || srcOverflow || !sizeU256.IsUint64() {
		return errOverflow
	}

	size := sizeU256.Uint64()
	price := tosca.Gas(3 * tosca.SizeInWords(size))
	if err := c.useGas(price); err != nil {
		return err
	}

	data, err := c.memory.getSlice(srcOffset, size, c)
	if err != nil {
		return err
	}
	if err := c.memory.set(destOffset, data, c); err != nil {
		return err
	}
	return nil
}

func opMload(c *context) error {
	var trg = c.stack.peek()
	var addr = *trg

	if !addr.IsUint64() {
		return errOverflow
	}
	offset := addr.Uint64()
	return c.memory.readWord(offset, trg, c)
}

func opMsize(c *context) {
	c.stack.pushUndefined().SetUint64(uint64(c.memory.length()))
}

func opSstore(c *context) error {

	// SStore is a write instruction, it shall not be executed in static mode.
	if c.params.Static {
		return errStaticContextViolation
	}

	// EIP-2200 demands that at least 2300 gas is available for SSTORE
	if c.gas <= 2300 {
		return errOutOfGas
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
	if err := c.useGas(cost); err != nil {
		return err
	}

	c.refund += getRefundForSstore(c.params.Revision, storageStatus)
	return nil
}

func opSload(c *context) error {
	var top = c.stack.peek()

	addr := c.params.Recipient
	slot := tosca.Key(top.Bytes32())
	if c.isAtLeast(tosca.R09_Berlin) {
		// charge costs for warm/cold slot access
		costs := tosca.Gas(100)
		if c.context.AccessStorage(addr, slot) == tosca.ColdAccess {
			costs = 2100
		}
		if err := c.useGas(costs); err != nil {
			return err
		}
	}
	value := c.context.GetStorage(addr, slot)
	top.SetBytes32(value[:])
	return nil
}

func opTstore(c *context) error {

	if !c.isAtLeast(tosca.R13_Cancun) {
		return errInvalidRevision
	}

	// Although not mentioned in the yellow paper, nor in CALL description at
	// website (https://www.evm.codes/#FA) Geth treats this Op as a write instruction.
	// therefore it shall not be executed in static mode.
	if c.params.Static {
		return errStaticContextViolation
	}

	key := tosca.Key(c.stack.pop().Bytes32())
	value := tosca.Word(c.stack.pop().Bytes32())
	c.context.SetTransientStorage(c.params.Recipient, key, value)
	return nil
}

func opTload(c *context) error {
	if !c.isAtLeast(tosca.R13_Cancun) {
		return errInvalidRevision
	}

	top := c.stack.peek()
	key := tosca.Key(top.Bytes32())
	value := c.context.GetTransientStorage(c.params.Recipient, key)
	top.SetBytes32(value[:])
	return nil
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

func opCallDataCopy(c *context) error {
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
		return errOverflow
	}

	// Charge for the copy costs
	words := tosca.SizeInWords(length64)
	price := tosca.Gas(3 * words)
	if err := c.useGas(price); err != nil {
		return err
	}

	data, err := c.memory.getSlice(memOffset64, length64, c)
	if err != nil {
		return err
	}
	codeCopy := getData(c.params.Input, dataOffset64, length64)
	copy(data, codeCopy)
	return nil
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

func opExp(c *context) error {
	base, exponent := c.stack.pop(), c.stack.peek()
	if err := c.useGas(tosca.Gas(50 * exponent.ByteLen())); err != nil {
		return err
	}
	exponent.Exp(base, exponent)
	return nil
}

// Evaluations show a 96% hit rate of this configuration.
var sha3Cache = newSha3HashCache(1<<16, 1<<18)

func opSha3(c *context) error {
	offset, size := c.stack.pop(), c.stack.peek()

	if checkSizeOffsetUint64Overflow(offset, size) != nil {
		return errOverflow
	}

	data, err := c.memory.getSlice(offset.Uint64(), size.Uint64(), c)
	if err != nil {
		return err
	}

	// charge dynamic gas price
	words := tosca.SizeInWords(size.Uint64())
	price := tosca.Gas(6 * words)
	if err := c.useGas(price); err != nil {
		return err
	}
	var hash tosca.Hash
	if c.withShaCache {
		// Cache hashes since identical values are frequently re-hashed.
		hash = sha3Cache.hash(data)
	} else {
		hash = Keccak256(data)
	}

	size.SetBytes32(hash[:])
	return nil
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

func opBalance(c *context) error {
	slot := c.stack.peek()
	address := tosca.Address(slot.Bytes20())
	if c.isAtLeast(tosca.R09_Berlin) {
		if err := c.useGas(getAccessCost(c.context.AccessAccount(address))); err != nil {
			return err
		}
	}
	balance := c.context.GetBalance(address)
	slot.SetBytes32(balance[:])
	return nil
}

func opSelfbalance(c *context) {
	balance := c.context.GetBalance(c.params.Recipient)
	c.stack.pushUndefined().SetBytes32(balance[:])
}

func opBaseFee(c *context) error {
	if !c.isAtLeast(tosca.R10_London) {
		return errInvalidRevision
	}
	fee := c.params.BaseFee
	c.stack.pushUndefined().SetBytes32(fee[:])
	return nil
}

func opBlobHash(c *context) error {
	if !c.isAtLeast(tosca.R13_Cancun) {
		return errInvalidRevision
	}

	index := c.stack.pop()
	blobHashesLength := uint64(len(c.params.BlobHashes))
	if index.IsUint64() && index.Uint64() < blobHashesLength {
		c.stack.pushUndefined().SetBytes32(c.params.BlobHashes[index.Uint64()][:])
	} else {
		c.stack.push(uint256.NewInt(0))
	}
	return nil
}

func opBlobBaseFee(c *context) error {
	if !c.isAtLeast(tosca.R13_Cancun) {
		return errInvalidRevision
	}
	fee := c.params.BlobBaseFee
	c.stack.pushUndefined().SetBytes32(fee[:])
	return nil
}

func opSelfdestruct(c *context) (status, error) {

	// SelfDestruct is a write instruction, it shall not be executed in static mode.
	if c.params.Static {
		return statusStopped, errStaticContextViolation
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
	if err := c.useGas(cost); err != nil {
		return statusStopped, err
	}

	destructed := c.context.SelfDestruct(c.params.Recipient, beneficiary)
	c.refund += selfDestructRefund(destructed, c.params.Revision)
	return statusSelfDestructed, nil
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

func opCodeCopy(c *context) error {
	var (
		memOffset  = c.stack.pop()
		codeOffset = c.stack.pop()
		length     = c.stack.pop()
	)

	if err := checkSizeOffsetUint64Overflow(memOffset, length); err != nil {
		return err
	}

	uint64CodeOffset, overflow := codeOffset.Uint64WithOverflow()
	if overflow {
		uint64CodeOffset = 0xffffffffffffffff
	}

	// Charge for length of copied code
	words := tosca.SizeInWords(length.Uint64())
	if err := c.useGas(tosca.Gas(3 * words)); err != nil {
		return err
	}

	data, err := c.memory.getSlice(memOffset.Uint64(), length.Uint64(), c)
	if err != nil {
		return err
	}
	codeCopy := getData(c.params.Code, uint64CodeOffset, length.Uint64())
	copy(data, codeCopy)
	return nil
}

func opExtcodesize(c *context) error {
	top := c.stack.peek()
	address := tosca.Address(top.Bytes20())
	if c.isAtLeast(tosca.R09_Berlin) {
		if err := c.useGas(getAccessCost(c.context.AccessAccount(address))); err != nil {
			return err
		}
	}
	top.SetUint64(uint64(c.context.GetCodeSize(address)))
	return nil
}

func opExtcodehash(c *context) error {
	slot := c.stack.peek()
	address := tosca.Address(slot.Bytes20())
	if c.isAtLeast(tosca.R09_Berlin) {
		if err := c.useGas(getAccessCost(c.context.AccessAccount(address))); err != nil {
			return err
		}
	}
	if !c.context.AccountExists(address) {
		slot.Clear()
	} else {
		hash := c.context.GetCodeHash(address)
		slot.SetBytes32(hash[:])
	}
	return nil
}

func opCreate(c *context) error {
	return genericCreate(c, tosca.Create)
}

func opCreate2(c *context) error {
	return genericCreate(c, tosca.Create2)
}

func genericCreate(c *context, kind tosca.CallKind) error {

	// Create is a write instruction, it shall not be executed in static mode.
	if c.params.Static {
		return errStaticContextViolation
	}

	var (
		value  = c.stack.pop()
		offset = c.stack.pop()
		size   = c.stack.pop()
		salt   = tosca.Hash{}
	)
	if kind == tosca.Create2 {
		salt = c.stack.pop().Bytes32() // pop salt value for Create2
	}

	if checkSizeOffsetUint64Overflow(offset, size) != nil {
		return errOverflow
	}

	sizeU64 := size.Uint64()
	input, err := c.memory.getSlice(offset.Uint64(), sizeU64, c)
	if err != nil {
		return err
	}

	if c.isAtLeast(tosca.R12_Shanghai) {
		initCodeCost, err := computeCodeSizeCost(sizeU64)
		if err != nil {
			return err
		}
		if err = c.useGas(tosca.Gas(initCodeCost)); err != nil {
			return err
		}
	}

	if kind == tosca.Create2 {
		// Charge for hashing the init code to compute the target address.
		words := tosca.SizeInWords(sizeU64)
		if err := c.useGas(tosca.Gas(6 * words)); err != nil {
			return err
		}
	}

	if !value.IsZero() {
		balance := c.context.GetBalance(c.params.Recipient)
		balanceU256 := new(uint256.Int).SetBytes(balance[:])

		if value.Gt(balanceU256) {
			c.stack.pushUndefined().Clear()
			c.returnData = nil
			return nil
		}
	}

	// Apply EIP150
	gas := c.gas
	gas -= gas / 64
	if err := c.useGas(gas); err != nil {
		return err
	}

	res, err := c.context.Call(kind, tosca.CallParameters{
		Sender: c.params.Recipient,
		Value:  tosca.Value(value.Bytes32()),
		Input:  input,
		Gas:    gas,
		Salt:   salt,
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
	return nil
}

// computeCodeSizeCost checks the size of the init code.
// Returns the gas cost for the size of the init code and nil, or
// zero and an error if size is greater than MaxInitCodeSize.
func computeCodeSizeCost(size uint64) (tosca.Gas, error) {
	const (
		maxCodeSize     = 24576           // Maximum bytecode to permit for a contract
		maxInitCodeSize = 2 * maxCodeSize // Maximum initcode to permit in a creation transaction and create instructions
	)
	if size > maxInitCodeSize {
		return 0, errInitCodeTooLarge
	}
	// Once per word of the init code when creating a contract.
	const initCodeWordGas = 2
	return tosca.Gas(initCodeWordGas * tosca.SizeInWords(size)), nil
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

func opExtCodeCopy(c *context) error {
	var (
		stack      = c.stack
		a          = stack.pop()
		memOffset  = stack.pop()
		codeOffset = stack.pop()
		length     = stack.pop()
	)
	if err := checkSizeOffsetUint64Overflow(memOffset, length); err != nil {
		return err
	}

	// Charge for length of copied code
	words := tosca.SizeInWords(length.Uint64())
	if err := c.useGas(tosca.Gas(3 * words)); err != nil {
		return err
	}

	address := tosca.Address(a.Bytes20())
	if c.isAtLeast(tosca.R09_Berlin) {
		if err := c.useGas(getAccessCost(c.context.AccessAccount(address))); err != nil {
			return err
		}
	}
	var uint64CodeOffset uint64
	if codeOffset.IsUint64() {
		uint64CodeOffset = codeOffset.Uint64()
	} else {
		uint64CodeOffset = math.MaxUint64
	}

	data, err := c.memory.getSlice(memOffset.Uint64(), length.Uint64(), c)
	if err != nil {
		return err
	}
	codeCopy := getData(c.context.GetCode(address), uint64CodeOffset, length.Uint64())
	copy(data, codeCopy)
	return nil
}

func checkSizeOffsetUint64Overflow(offset, size *uint256.Int) error {
	if size.IsZero() {
		return nil
	}
	if !offset.IsUint64() || !size.IsUint64() || offset.Uint64()+size.Uint64() < offset.Uint64() {
		return errOverflow
	}
	return nil
}

func getAccessCost(accessStatus tosca.AccessStatus) tosca.Gas {
	// EIP-2929 says that cold access cost is 2600 and warm is 100.
	// (https://eips.ethereum.org/EIPS/eip-2929)
	if accessStatus == tosca.ColdAccess {
		return tosca.Gas(2600)
	}
	return tosca.Gas(100)
}

func genericCall(c *context, kind tosca.CallKind) error {
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

	if checkSizeOffsetUint64Overflow(inOffset, inSize) != nil {
		return errOverflow
	}

	if checkSizeOffsetUint64Overflow(retOffset, retSize) != nil {
		return errOverflow
	}

	// Get arguments from the memory.
	args, err := c.memory.getSlice(inOffset.Uint64(), inSize.Uint64(), c)
	if err != nil {
		return err
	}
	output, err := c.memory.getSlice(retOffset.Uint64(), retSize.Uint64(), c)
	if err != nil {
		return err
	}

	// from berlin onwards access cost changes depending on warm/cold access.
	if c.isAtLeast(tosca.R09_Berlin) {
		if err := c.useGas(getAccessCost(c.context.AccessAccount(toAddr))); err != nil {
			return err
		}
	}

	// for static and delegate calls, the following value checks will always be zero.
	// Charge for transferring value to a new address
	if !value.IsZero() {
		if err := c.useGas(CallValueTransferGas); err != nil {
			return err
		}
	}

	// EIP158 states that non-zero value calls that create a new account should
	// be charged an additional gas fee.
	if kind == tosca.Call && !value.IsZero() && !c.context.AccountExists(toAddr) {
		if err := c.useGas(CallNewAccountGas); err != nil {
			return err
		}
	}

	// The Homestead hard-fork introduced a limit on the amount of gas that can be
	// forwarded to recursive calls. EIP-150 (https://eips.ethereum.org/EIPS/eip-150)
	// defines that at all but one 64th of the available gas in one scope may be passed
	// to a nested call.
	nestedCallGas := tosca.Gas(c.gas - c.gas/64)
	if provided_gas.IsUint64() && (nestedCallGas >= tosca.Gas(provided_gas.Uint64())) {
		nestedCallGas = tosca.Gas(provided_gas.Uint64())
	}
	if err := c.useGas(nestedCallGas); err != nil {
		// this usage can never fail because the endowment is at most
		// 63/64 of the current gas level.
		return err
	}

	// first use static and dynamic gas cost and then resize the memory
	// when out of gas is happening, then mem should not be resized
	if !value.IsZero() {
		nestedCallGas += CallStipend
	}

	// Check that the caller has enough balance to transfer the requested value.
	if (kind == tosca.Call || kind == tosca.CallCode) && !value.IsZero() {
		balance := c.context.GetBalance(c.params.Recipient)
		balanceU256 := new(uint256.Int).SetBytes32(balance[:])
		if balanceU256.Lt(value) {
			c.stack.pushUndefined().Clear()
			c.returnData = nil
			c.gas += nestedCallGas // the gas send to the nested contract is returned
			return nil
		}
	}

	// If we are in static mode, recursive calls are to be treated like
	// static calls. This is a consequence of the unification of the
	// interpreter interfaces of EVMC and Geth.
	// This problem was encountered in block 58413779, transaction 7.
	if c.params.Static && kind == tosca.Call {
		kind = tosca.StaticCall
	}

	// Prepare arguments, depending on call kind
	callParams := tosca.CallParameters{
		Input: args,
		Gas:   nestedCallGas,
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
		copy(output, ret.Output)
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
	return nil
}

func opCall(c *context) error {
	value := c.stack.peekN(2)
	// In a static call, no value must be transferred.
	if c.params.Static && !value.IsZero() {
		return errStaticContextViolation
	}
	return genericCall(c, tosca.Call)
}

func opCallCode(c *context) error {
	return genericCall(c, tosca.CallCode)
}

func opStaticCall(c *context) error {
	return genericCall(c, tosca.StaticCall)
}

func opDelegateCall(c *context) error {
	return genericCall(c, tosca.DelegateCall)
}

func opReturnDataSize(c *context) {
	c.stack.pushUndefined().SetUint64(uint64(len(c.returnData)))
}

func opReturnDataCopy(c *context) error {
	var (
		memOffset  = c.stack.pop()
		dataOffset = c.stack.pop()
		length     = c.stack.pop()
	)

	offset64, overflow := dataOffset.Uint64WithOverflow()
	if overflow {
		return errOverflow
	}
	// we can reuse dataOffset now (aliasing it for clarity)
	var end = dataOffset
	end.Add(dataOffset, length)
	end64, overflow := end.Uint64WithOverflow()
	if overflow {
		return errOverflow
	}

	if uint64(len(c.returnData)) < end64 {
		return errOverflow
	}

	if err := checkSizeOffsetUint64Overflow(memOffset, length); err != nil {
		return err
	}

	words := tosca.SizeInWords(length.Uint64())
	if err := c.useGas(tosca.Gas(3 * words)); err != nil {
		return errOutOfGas
	}

	return c.memory.set(memOffset.Uint64(), c.returnData[offset64:end64], c)
}

func opLog(c *context, size int) error {

	// LogN op codes are write instructions, they shall not be executed in static mode.
	if c.params.Static {
		return errStaticContextViolation
	}

	topics := make([]tosca.Hash, size)
	stack := c.stack
	mStart, mSize := stack.pop(), stack.pop()

	if err := checkSizeOffsetUint64Overflow(mStart, mSize); err != nil {
		return err
	}

	for i := 0; i < size; i++ {
		addr := stack.pop()
		topics[i] = addr.Bytes32()
	}

	// Expand memory if needed
	start := mStart.Uint64()
	log_size := mSize.Uint64()

	// charge for log size
	if err := c.useGas(tosca.Gas(8 * log_size)); err != nil {
		return err
	}

	data, err := c.memory.getSlice(start, log_size, c)
	if err != nil {
		return err
	}

	// make a copy of the data to disconnect from memory
	log_data := bytes.Clone(data)
	c.context.EmitLog(tosca.Log{
		Address: c.params.Recipient,
		Topics:  topics,
		Data:    log_data,
	})
	return nil
}
