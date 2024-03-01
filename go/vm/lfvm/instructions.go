package lfvm

import (
	"errors"
	"math/big"
	"math/bits"

	"github.com/ethereum/go-ethereum/params"
	"github.com/holiman/uint256"
	"golang.org/x/crypto/sha3"

	"github.com/Fantom-foundation/Tosca/go/vm"
)

var (
	big0 = big.NewInt(0)
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
func opInvalid(c *context) {
	c.status = INVALID_INSTRUCTION
}

func opPc(c *context) {
	c.stack.pushEmpty().SetUint64(uint64(c.code[c.pc].arg))
}

func checkJumpDest(c *context) {
	if int(c.pc+1) >= len(c.code) || c.code[c.pc+1].opcode != JUMPDEST {
		c.SignalError(ErrInvalidJump)
	}
}

func opJump(c *context) {
	destination := c.stack.pop()
	// overflow check
	if !destination.IsUint64() || destination.Uint64()>>33 > 0 {
		c.SignalError(ErrInvalidJump)
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
		if !destination.IsUint64() || destination.Uint64()>>33 > 0 {
			c.SignalError(ErrInvalidJump)
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

func opNoop(c *context) {
	c.SignalError(errors.New("instruction NOOP should not be reachable by interpreter"))
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
	if c.memory.EnsureCapacity(offset, 32, c) != nil {
		return
	}
	if err := c.memory.SetWord(offset, value); err != nil {
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
	if c.memory.EnsureCapacity(offset, 1, c) != nil {
		return
	}
	if err := c.memory.SetByte(offset, byte(value.Uint64())); err != nil {
		c.SignalError(err)
	}
}

func opMload(c *context) {
	var trg = c.stack.peek()
	var addr = *trg

	if !addr.IsUint64() {
		c.SignalError(ErrGasUintOverflow)
		return
	}
	offset := addr.Uint64()
	if c.memory.EnsureCapacity(offset, 32, c) != nil {
		return
	}
	if err := c.memory.CopyWord(offset, trg); err != nil {
		c.SignalError(err)
	}
}

func opMsize(c *context) {
	c.stack.pushEmpty().SetUint64(uint64(c.memory.Len()))
}

func opSstore(c *context) {
	gasfunc := gasSStore
	if c.isBerlin {
		gasfunc = gasSStoreEIP2929
	}

	// Charge the gas price for this operation
	price, err := gasfunc(c)
	if err != nil || !c.UseGas(price) {
		return
	}

	var key = vm.Key(c.stack.pop().Bytes32())
	var value = vm.Word(c.stack.pop().Bytes32())

	// Perform storage update
	c.context.SetStorage(c.params.Recipient, key, value)
}

func opSload(c *context) {
	var top = c.stack.peek()

	slot := vm.Key(top.Bytes32())
	if c.isBerlin {
		// Check slot presence in the access list
		if _, slotPresent := c.context.IsSlotInAccessList(c.params.Recipient, slot); !slotPresent {
			// If the caller cannot afford the cost, this change will be rolled back
			// If he does afford it, we can skip checking the same thing later on, during execution
			if !c.IsShadowed() {
				c.context.AccessStorage(c.params.Recipient, slot)
			}
			if !c.UseGas(params.ColdSloadCostEIP2929) {
				return
			}
		} else {
			if !c.UseGas(params.WarmStorageReadCostEIP2929) {
				return
			}
		}
	}
	value := c.context.GetStorage(c.params.Recipient, slot)
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
	words := (length64 + 31) / 32
	price := 3 * words
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
	if !c.UseGas(vm.Gas(50 * exponent.ByteLen())) {
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

	if c.memory.EnsureCapacity(offset.Uint64(), size.Uint64(), c) != nil {
		return
	}
	data := c.memory.GetSlice(offset.Uint64(), size.Uint64())

	// charge dynamic gas price
	minimum_word_size := vm.Gas((size.Uint64() + 31) / 32)
	price := 6 * minimum_word_size
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
		c.hasher.Write(data)
		c.hasher.Read(c.hasherBuf[:])
	}

	size.SetBytes32(c.hasherBuf[:])
}

func opGas(c *context) {
	c.stack.pushEmpty().SetUint64(uint64(c.gas))
}

func opDifficulty(c *context) {
	difficulty := c.context.GetTransactionContext().PrevRandao
	c.stack.pushEmpty().SetBytes32(difficulty[:])
}

func opTimestamp(c *context) {
	time := c.context.GetTransactionContext().Timestamp
	c.stack.pushEmpty().SetUint64(uint64(time))
}

func opNumber(c *context) {
	number := c.context.GetTransactionContext().BlockNumber
	c.stack.pushEmpty().SetUint64(uint64(number))
}

func opCoinbase(c *context) {
	coinbase := c.context.GetTransactionContext().Coinbase
	c.stack.pushEmpty().SetBytes32(coinbase[:])
}

func opGasLimit(c *context) {
	limit := c.context.GetTransactionContext().GasLimit
	c.stack.pushEmpty().SetUint64(uint64(limit))
}

func opGasPrice(c *context) {
	price := c.context.GetTransactionContext().GasPrice
	c.stack.pushEmpty().SetBytes32(price[:])
}

func opBalance(c *context) {
	slot := c.stack.peek()
	address := vm.Address(slot.Bytes20())
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
	if c.isLondon {
		fee := c.context.GetTransactionContext().BaseFee
		c.stack.pushEmpty().SetBytes32(fee[:])
	} else {
		c.status = INVALID_INSTRUCTION
		return
	}
}

func opSelfdestruct(c *context) {
	gasfunc := gasSelfdestruct
	if c.isBerlin {
		gasfunc = gasSelfdestructEIP2929
	}
	// even death is not for free
	if !c.UseGas(vm.Gas(gasfunc(c))) {
		return
	}
	beneficiary := vm.Address(c.stack.pop().Bytes20())
	c.context.SelfDestruct(c.params.Recipient, beneficiary)
	c.status = SUICIDED
}

func opChainId(c *context) {
	id := c.context.GetTransactionContext().ChainID
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
	upper = uint64(c.context.GetTransactionContext().BlockNumber)
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
	c.stack.pushEmpty().SetBytes20(c.params.Sender[:])
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
		c.SignalError(ErrGasUintOverflow)
		return
	}

	uint64CodeOffset, overflow := codeOffset.Uint64WithOverflow()
	if overflow {
		uint64CodeOffset = 0xffffffffffffffff
	}

	// Charge for length of copied code
	words := (length.Uint64() + 31) / 32
	if !c.UseGas(vm.Gas(3 * words)) {
		return
	}

	codeCopy := getData(c.params.Code, uint64CodeOffset, length.Uint64())
	if c.memory.EnsureCapacity(memOffset.Uint64(), length.Uint64(), c) != nil {
		return
	}
	if err := c.memory.Set(memOffset.Uint64(), length.Uint64(), codeCopy); err != nil {
		c.SignalError(err)
	}
}

func opExtcodesize(c *context) {
	top := c.stack.peek()
	addr := vm.Address(top.Bytes20())
	err := gasEip2929AccountCheck(c, addr)
	if err != nil {
		return
	}
	top.SetUint64(uint64(c.context.GetCodeSize(addr)))
}

func opExtcodehash(c *context) {
	slot := c.stack.peek()
	address := vm.Address(slot.Bytes20())
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

func opCreate(c *context) {
	var (
		value        = vm.Value(c.stack.pop().Bytes32())
		offset, size = c.stack.pop(), c.stack.pop()
	)
	if err := checkSizeOffsetUint64Overflow(offset, size); err != nil {
		c.SignalError(err)
		return
	}

	if c.memory.EnsureCapacity(offset.Uint64(), size.Uint64(), c) != nil {
		return
	}

	input := c.memory.GetSlice(offset.Uint64(), size.Uint64())

	gas := c.gas
	if true /*c.evm.chainRules.IsEIP150*/ {
		gas -= gas / 64
	}

	c.UseGas(gas)

	res, gasLeft, gasRefund, createdAddr, reverted, subErr := c.context.Call(
		vm.Create,
		vm.Address{}, // < recipient
		c.params.Recipient, // < sender
		value,
		input,
		gas,
		c.params.Depth + 1,
		false,
		vm.Hash{}, // salt
		vm.Address{}, // codeAddress
	)

	c.gas = gasLeft
	c.refund += gasRefund

	success := c.stack.pushEmpty()
	if reverted || subErr != nil {
		success.Clear()
	} else {
		success.SetBytes20(createdAddr[:])
	}
	
	if reverted {
		c.return_data = res
	} else {
		c.return_data = nil
	}
}

func opCreate2(c *context) {
	var (
		endowment    = c.stack.pop()
		offset, size = c.stack.pop(), c.stack.pop()
		salt         = c.stack.pop()
	)
	if err := checkSizeOffsetUint64Overflow(offset, size); err != nil {
		c.SignalError(err)
		return
	}

	if c.memory.EnsureCapacity(offset.Uint64(), size.Uint64(), c) != nil {
		return
	}
	input := c.memory.GetSlice(offset.Uint64(), size.Uint64())

	// Charge for the code size
	words := (size.Uint64() + 31) / 32
	if !c.UseGas(6 * words) {
		return
	}

	// Apply EIP150
	gas := c.contract.Gas
	gas -= gas / 64
	if !c.UseGas(gas) {
		return
	}

	//TODO: use uint256.Int instead of converting with toBig()
	bigEndowment := big0
	if !endowment.IsZero() {
		bigEndowment = endowment.ToBig()
	}

	res, addr, returnGas, suberr := c.evm.Create2(c.contract, input, gas, bigEndowment, salt)

	// Push item on the stack based on the returned error.
	success := c.stack.pushEmpty()
	if suberr != nil {
		success.Clear()
	} else {
		success.SetBytes(addr.Bytes())
	}
	c.contract.Gas += returnGas

	if suberr == vm.ErrExecutionReverted {
		c.return_data = res
	} else {
		c.return_data = nil
	}
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
	return common.RightPadBytes(data[start:end], int(size))
}

func opExtCodeCopy(c *context) {
	var (
		stack      = c.stack
		a          = stack.pop()
		memOffset  = stack.pop()
		codeOffset = stack.pop()
		length     = stack.pop()
	)
	if checkSizeOffsetUint64Overflow(memOffset, length) != nil || checkSizeOffsetUint64Overflow(codeOffset, length) != nil {
		c.SignalError(vm.ErrGasUintOverflow)
		return
	}
	uint64CodeOffset := codeOffset.Uint64()

	// Charge for length of copied code
	words := (length.Uint64() + 31) / 32
	if !c.UseGas(3 * words) {
		return
	}

	addr := common.Address(a.Bytes20())
	err := gasEip2929AccountCheck(c, addr)
	if err != nil {
		return
	}
	codeCopy := getData(c.evm.StateDB.GetCode(addr), uint64CodeOffset, length.Uint64())
	if c.memory.EnsureCapacity(memOffset.Uint64(), length.Uint64(), c) != nil {
		return
	}
	if err = c.memory.Set(memOffset.Uint64(), length.Uint64(), codeCopy); err != nil {
		c.SignalError(err)
	}
}

func checkSizeOffsetUint64Overflow(offset, size *uint256.Int) error {
	if size.IsZero() {
		return nil
	}
	if !offset.IsUint64() || !size.IsUint64() || offset.Uint64()+size.Uint64() < offset.Uint64() {
		return vm.ErrGasUintOverflow
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

func opCall(c *context) {
	warmAccess, coldCost, err := addressInAccessList(c)
	if err != nil {
		return
	}
	stack := c.stack
	// Pop call parameters.
	provided_gas, addr, value, inOffset, inSize, retOffset, retSize := stack.pop(), stack.pop(), stack.pop(), stack.pop(), stack.pop(), stack.pop(), stack.pop()

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
	base_gas := c.memory.ExpansionCosts(needed_memory_size)

	// We need to check the existence of the target account before removing
	// the gas price for the other cost factors to make sure that the read
	// in the state DB is always happening. This is the current EVM behaviour,
	// and not doing it would be identified by the replay tool as an error.
	toAddr := common.Address(addr.Bytes20())

	// Charge for transfering value to a new address
	if !value.IsZero() {
		base_gas += params.CallValueTransferGas
	}

	// if evm.chainRules.IsEIP158 according to GETH it is EIP158 since 2016
	// !!!! but need to touch stateDB for the address to have it in the substate record key/value
	if !value.IsZero() && !c.stateDB.Exist(toAddr) {
		base_gas += params.CallNewAccountGas
	}

	cost := callGas(c.contract.Gas, base_gas, provided_gas)

	if warmAccess {
		if !c.UseGas(base_gas + cost) {
			return
		}
	} else {
		// In case of a cold access, we temporarily add the cold charge back, and also
		// add it to the returned gas. By adding it to the return, it will be charged
		// outside of this function, as part of the dynamic gas, and that will make it
		// also become correctly reported to tracers.
		c.contract.Gas += coldCost
		if !c.UseGas(base_gas + cost + coldCost) {
			return
		}
	}

	// first use static and dynamic gas cost and then resize the memory
	// when out of gas is happening, then mem should not be resized
	c.memory.EnsureCapacityWithoutGas(needed_memory_size, c)

	var bigVal = big0
	//TODO: use uint256.Int instead of converting with toBig()
	// By using big0 here, we save an alloc for the most common case (non-ether-transferring contract calls),
	// but it would make more sense to extend the usage of uint256.Int
	if !value.IsZero() {
		cost += params.CallStipend
		bigVal = value.ToBig()
	}

	// Get the arguments from the memory.
	args := c.memory.GetSlice(inOffset.Uint64(), inSize.Uint64())
	ret, returnGas, err := c.evm.Call(c.contract, toAddr, args, cost, bigVal)

	if err == nil || err == vm.ErrExecutionReverted {
		if memSetErr := c.memory.Set(retOffset.Uint64(), retSize.Uint64(), ret); memSetErr != nil {
			c.SignalError(memSetErr)
		}
	}

	success := stack.pushEmpty()
	if err != nil {
		success.Clear()
	} else {
		success.SetOne()
	}
	c.contract.Gas += returnGas
	c.return_data = ret
}

func opCallCode(c *context) {
	warmAccess, coldCost, err := addressInAccessList(c)
	if err != nil {
		return
	}
	stack := c.stack
	// Pop call parameters.
	provided_gas, addr, value, inOffset, inSize, retOffset, retSize := stack.pop(), stack.pop(), stack.pop(), stack.pop(), stack.pop(), stack.pop(), stack.pop()

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
	base_gas := c.memory.ExpansionCosts(needed_memory_size)

	// We need to check the existence of the target account before removing
	// the gas price for the other cost factors to make sure that the read
	// in the state DB is always happening. This is the current EVM behaviour,
	// and not doing it would be identified by the replay tool as an error.
	toAddr := common.Address(addr.Bytes20())

	// Charge for transfering value to a new address
	if !value.IsZero() {
		base_gas += params.CallValueTransferGas
	}

	cost := callGas(c.contract.Gas, base_gas, provided_gas)

	if warmAccess {
		if !c.UseGas(base_gas + cost) {
			return
		}
	} else {
		// In case of a cold access, we temporarily add the cold charge back, and also
		// add it to the returned gas. By adding it to the return, it will be charged
		// outside of this function, as part of the dynamic gas, and that will make it
		// also become correctly reported to tracers.
		c.contract.Gas += coldCost
		if !c.UseGas(base_gas + cost + coldCost) {
			return
		}
	}

	// first use static and dynamic gas cost and then resize the memory
	// when out of gas is happening, then mem should not be resized
	c.memory.EnsureCapacityWithoutGas(needed_memory_size, c)

	var bigVal = big0
	//TODO: use uint256.Int instead of converting with toBig()
	// By using big0 here, we save an alloc for the most common case (non-ether-transferring contract calls),
	// but it would make more sense to extend the usage of uint256.Int
	if !value.IsZero() {
		cost += params.CallStipend
		bigVal = value.ToBig()
	}

	// Get the arguments from the memory.
	args := c.memory.GetSlice(inOffset.Uint64(), inSize.Uint64())
	ret, returnGas, err := c.evm.CallCode(c.contract, toAddr, args, cost, bigVal)

	if err == nil || err == vm.ErrExecutionReverted {
		if memSetErr := c.memory.Set(retOffset.Uint64(), retSize.Uint64(), ret); memSetErr != nil {
			c.SignalError(memSetErr)
		}
	}

	success := stack.pushEmpty()
	if err != nil {
		success.Clear()
	} else {
		success.SetOne()
	}
	c.contract.Gas += returnGas
	c.return_data = ret
}

func opStaticCall(c *context) {
	stack := c.stack

	warmAccess, coldCost, err := addressInAccessList(c)
	if err != nil {
		return
	}

	// Pop call parameters.
	provided_gas, addr, inOffset, inSize, retOffset, retSize := stack.pop(), stack.pop(), stack.pop(), stack.pop(), stack.pop(), stack.pop()

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
	base_gas := c.memory.ExpansionCosts(needed_memory_size)
	gas := callGas(c.contract.Gas, base_gas, provided_gas)

	if warmAccess {
		if !c.UseGas(base_gas + gas) {
			return
		}
	} else {
		// In case of a cold access, we temporarily add the cold charge back, and also
		// add it to the returned gas. By adding it to the return, it will be charged
		// outside of this function, as part of the dynamic gas, and that will make it
		// also become correctly reported to tracers.
		c.contract.Gas += coldCost
		if !c.UseGas(base_gas + gas + coldCost) {
			return
		}
	}

	// first use static and dynamic gas cost and then resize the memory
	// when out of gas is happening, then mem should not be resized
	c.memory.EnsureCapacityWithoutGas(needed_memory_size, c)

	toAddr := common.Address(addr.Bytes20())
	// Get arguments from the memory.
	args := c.memory.GetSlice(inOffset.Uint64(), inSize.Uint64())

	ret, returnGas, err := c.evm.StaticCall(c.contract, toAddr, args, gas)

	if err == nil || err == vm.ErrExecutionReverted {
		if memSetErr := c.memory.Set(retOffset.Uint64(), retSize.Uint64(), ret); memSetErr != nil {
			c.SignalError(memSetErr)
		}
	}

	success := stack.pushEmpty()
	if err != nil {
		success.Clear()
	} else {
		success.SetOne()
	}
	c.contract.Gas += returnGas
	c.return_data = ret
}

func opDelegateCall(c *context) {
	warmAccess, coldCost, err := addressInAccessList(c)
	if err != nil {
		return
	}
	stack := c.stack
	// Pop call parameters.
	provided_gas, addr, inOffset, inSize, retOffset, retSize := stack.pop(), stack.pop(), stack.pop(), stack.pop(), stack.pop(), stack.pop()

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
	base_gas := c.memory.ExpansionCosts(needed_memory_size)
	gas := callGas(c.contract.Gas, base_gas, provided_gas)

	if warmAccess {
		if !c.UseGas(base_gas + gas) {
			return
		}
	} else {
		// In case of a cold access, we temporarily add the cold charge back, and also
		// add it to the returned gas. By adding it to the return, it will be charged
		// outside of this function, as part of the dynamic gas, and that will make it
		// also become correctly reported to tracers.
		c.contract.Gas += coldCost
		if !c.UseGas(base_gas + gas + coldCost) {
			return
		}
	}

	// first use static and dynamic gas cost and then resize the memory
	// when out of gas is happening, then mem should not be resized
	c.memory.EnsureCapacityWithoutGas(needed_memory_size, c)

	toAddr := common.Address(addr.Bytes20())
	// Get arguments from the memory.
	args := c.memory.GetSlice(inOffset.Uint64(), inSize.Uint64())

	ret, returnGas, err := c.evm.DelegateCall(c.contract, toAddr, args, gas)

	if err == nil || err == vm.ErrExecutionReverted {
		if memSetErr := c.memory.Set(retOffset.Uint64(), retSize.Uint64(), ret); memSetErr != nil {
			c.SignalError(memSetErr)
		}
	}

	success := stack.pushEmpty()
	if err != nil {
		success.Clear()
	} else {
		success.SetOne()
	}
	c.contract.Gas += returnGas
	c.return_data = ret
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
		c.SignalError(vm.ErrReturnDataOutOfBounds)
		return
	}
	// we can reuse dataOffset now (aliasing it for clarity)
	var end = dataOffset
	end.Add(dataOffset, length)
	end64, overflow := end.Uint64WithOverflow()
	if overflow || uint64(len(c.return_data)) < end64 {
		c.SignalError(vm.ErrReturnDataOutOfBounds)
		return
	}

	if err := checkSizeOffsetUint64Overflow(memOffset, length); err != nil {
		c.SignalError(err)
		return
	}

	if c.memory.EnsureCapacity(memOffset.Uint64(), length.Uint64(), c) != nil {
		return
	}

	words := (length.Uint64() + 31) / 32
	if !c.UseGas(3 * words) {
		return
	}

	if err := c.memory.Set(memOffset.Uint64(), length.Uint64(), c.return_data[offset64:end64]); err != nil {
		c.SignalError(err)
	}
}

func opLog(c *context, size int) {
	topics := make([]common.Hash, size)
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

	// charge for log size
	if !c.UseGas(8 * mSize.Uint64()) {
		return
	}

	// Expand memory if needed
	start := mStart.Uint64()
	log_size := mSize.Uint64()
	if c.memory.EnsureCapacity(start, log_size, c) != nil {
		return
	}
	d := c.memory.GetSlice(start, log_size)

	// make a copy of the data to disconnect from memory
	log_data := common.CopyBytes(d)

	c.evm.StateDB.AddLog(&types.Log{
		Address: c.contract.Address(),
		Topics:  topics,
		Data:    log_data,
		// This is a non-consensus field, but assigned here because
		// core/state doesn't know the current block number.
		BlockNumber: c.evm.Context.BlockNumber.Uint64(),
	})
}

// ----------------------------- Super Instructions -----------------------------

func opSwap1_Pop(c *context) {
	a1 := c.stack.pop()
	a2 := c.stack.peek()
	*a2 = *a1
}

func opSwap2_Pop(c *context) {
	a1 := c.stack.pop()
	*c.stack.Back(1) = *a1
}

func opPush1_Push1(c *context) {
	arg := c.code[c.pc].arg
	c.stack.stack_ptr += 2
	c.stack.Back(0).SetUint64(uint64(arg & 0xFF))
	c.stack.Back(1).SetUint64(uint64(arg >> 8))
}

func opPush1_Add(c *context) {
	arg := c.code[c.pc].arg
	trg := c.stack.peek()
	var carry uint64
	trg[0], carry = bits.Add64(trg[0], uint64(arg), 0)
	trg[1], carry = bits.Add64(trg[1], 0, carry)
	trg[2], carry = bits.Add64(trg[2], 0, carry)
	trg[3], _ = bits.Add64(trg[3], 0, carry)
}

func opPush1_Shl(c *context) {
	arg := c.code[c.pc].arg
	trg := c.stack.peek()
	trg.Lsh(trg, uint(arg))
}

func opPush1_Dup1(c *context) {
	arg := c.code[c.pc].arg
	c.stack.stack_ptr += 2
	c.stack.Back(0).SetUint64(uint64(arg))
	c.stack.Back(1).SetUint64(uint64(arg))
}

func opPush2_Jump(c *context) {
	// Directly take pushed value and jump to destination.
	c.pc = int32(c.code[c.pc].arg) - 1
	checkJumpDest(c)
}

func opPush2_Jumpi(c *context) {
	// Directly take pushed value and jump to destination.
	condition := c.stack.pop()
	if !condition.IsZero() {
		c.pc = int32(c.code[c.pc].arg) - 1
		checkJumpDest(c)
	}
}

func opSwap2_Swap1(c *context) {
	a1 := c.stack.Back(0)
	a2 := c.stack.Back(1)
	a3 := c.stack.Back(2)
	*a1, *a2, *a3 = *a2, *a3, *a1
}

func opDup2_Mstore(c *context) {
	var value = c.stack.pop()
	var addr = c.stack.peek()

	offset := addr.Uint64()
	if c.memory.EnsureCapacity(offset, 32, c) != nil {
		return
	}
	if err := c.memory.SetWord(offset, value); err != nil {
		c.SignalError(err)
	}
}

func opDup2_Lt(c *context) {
	b := c.stack.Back(0)
	a := c.stack.Back(1)
	if a.Lt(b) {
		b.SetOne()
	} else {
		b.Clear()
	}
}

func opPopPop(c *context) {
	c.stack.stack_ptr -= 2
}

func opPop_Jump(c *context) {
	opPop(c)
	opJump(c)
}

func opIsZero_Push2_Jumpi(c *context) {
	condition := c.stack.pop()
	if condition.IsZero() {
		c.pc = int32(c.code[c.pc].arg) - 1
		checkJumpDest(c)
	}
}

func opSwap2_Swap1_Pop_Jump(c *context) {
	top := c.stack.pop()
	c.stack.pop()
	trg := c.stack.peek()
	c.pc = int32(trg.Uint64()) - 1
	*trg = *top
}

func opSwap1_Pop_Swap2_Swap1(c *context) {
	a1 := c.stack.pop()
	a2 := c.stack.Back(0)
	a3 := c.stack.Back(1)
	a4 := c.stack.Back(2)
	*a2, *a3, *a4 = *a3, *a4, *a1
}

func opPop_Swap2_Swap1_Pop(c *context) {
	c.stack.pop()
	a2 := c.stack.pop()
	a3 := c.stack.Back(0)
	a4 := c.stack.Back(1)
	*a3, *a4 = *a4, *a2
}

func opPush1_Push4_Dup3(c *context) {
	opPush1(c)
	c.pc++
	opPush4(c)
	opDup(c, 3)
}

func opAnd_Swap1_Pop_Swap2_Swap1(c *context) {
	opAnd(c)
	opSwap1_Pop_Swap2_Swap1(c)
}

func opPush1_Push1_Push1_Shl_Sub(c *context) {
	arg1 := c.code[c.pc].arg
	arg2 := c.code[c.pc+1].arg
	shift := uint8(arg2)
	value := uint8(arg1 & 0xFF)
	delta := uint8(arg1 >> 8)
	trg := c.stack.pushEmpty()
	trg.SetUint64(uint64(value))
	trg.Lsh(trg, uint(shift))
	trg.Sub(trg, uint256.NewInt(uint64(delta)))
	c.pc++
}
