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
	"fmt"
	"log"
	"sort"
	"sync"

	"github.com/Fantom-foundation/Tosca/go/tosca"
	"github.com/holiman/uint256"
)

type status byte

const (
	statusRunning status = iota
	statusStopped
	statusReverted
	statusReturned
	statusSuicided
	statusInvalidInstruction
	statusOutOfGas
	statusMaximumInitCodeSizeExceeded
	statusError
)

type context struct {
	// Context instances
	params  tosca.Parameters
	context tosca.RunContext

	// Execution state
	pc     int32
	gas    tosca.Gas
	refund tosca.Gas
	stack  *Stack
	memory *Memory
	status status

	// Inputs
	code     Code
	revision tosca.Revision

	// Intermediate data
	returnData []byte

	// Outputs
	resultOffset uint256.Int
	resultSize   uint256.Int

	// Configuration flags
	withShaCache bool
}

func (c *context) UseGas(amount tosca.Gas) bool {
	if c.gas < 0 || amount < 0 || c.gas < amount {
		c.status = statusOutOfGas
		return false
	}
	c.gas -= amount
	return true
}

func (c *context) SignalError(error) { // < TODO: remove error parameter
	c.status = statusError
}

func (c *context) isBerlin() bool {
	return c.revision >= tosca.R09_Berlin
}

func (c *context) isLondon() bool {
	return c.revision >= tosca.R10_London
}

func (c *context) isShanghai() bool {
	return c.revision >= tosca.R12_Shanghai
}

func (c *context) isCancun() bool {
	return c.revision >= tosca.R13_Cancun
}

func Run(
	params tosca.Parameters,
	code Code,
	withStatistics bool,
	withShaCache bool,
	withLogging bool,
) (tosca.Result, error) {
	// Don't bother with the execution if there's no code.
	if len(params.Code) == 0 {
		return tosca.Result{
			Output:  nil,
			GasLeft: params.Gas,
			Success: true,
		}, nil
	}

	// Set up execution context.
	var ctxt = context{
		params:       params,
		context:      params.Context,
		gas:          params.Gas,
		stack:        NewStack(),
		memory:       NewMemory(),
		status:       statusRunning,
		code:         code,
		revision:     params.Revision,
		withShaCache: withShaCache,
	}

	defer func() {
		ReturnStack(ctxt.stack)
	}()

	// Run interpreter.
	if withStatistics {
		runWithStatistics(&ctxt)
	} else if withLogging {
		runWithLogging(&ctxt)
	} else {
		run(&ctxt)
	}

	return generateResult(&ctxt)
}

func generateResult(ctxt *context) (tosca.Result, error) {

	res, err := getOutput(ctxt)
	if err != nil {
		return tosca.Result{Success: false}, nil
	}

	// Handle return status
	switch ctxt.status {
	case statusStopped, statusSuicided:
		return tosca.Result{
			Success:   true,
			GasLeft:   ctxt.gas,
			GasRefund: ctxt.refund,
		}, nil
	case statusReturned:
		return tosca.Result{
			Success:   true,
			Output:    res,
			GasLeft:   ctxt.gas,
			GasRefund: ctxt.refund,
		}, nil
	case statusReverted:
		return tosca.Result{
			Success: false,
			Output:  res,
			GasLeft: ctxt.gas,
		}, nil
	case statusInvalidInstruction, statusOutOfGas, statusMaximumInitCodeSizeExceeded, statusError: // < TODO: if all these are handled the same, no need to have anything but statusError
		return tosca.Result{
			Success: false,
		}, nil
	default:
		return tosca.Result{}, fmt.Errorf("unexpected error in interpreter, unknown status: %v", ctxt.status)
	}
}

func getOutput(ctxt *context) ([]byte, error) {
	var res []byte
	if ctxt.status == statusReturned || ctxt.status == statusReverted {
		size, overflow := ctxt.resultSize.Uint64WithOverflow()
		if overflow {
			return nil, errGasUintOverflow
		}

		if size != 0 {
			offset, overflow := ctxt.resultOffset.Uint64WithOverflow()
			if overflow {
				return nil, errGasUintOverflow
			}

			// Extract the result from the memory
			if err := ctxt.memory.EnsureCapacity(offset, size, ctxt); err != nil {
				return nil, err
			}
			res = make([]byte, size)
			ctxt.memory.CopyData(offset, res)
		}
	}
	return res, nil
}

func run(c *context) {
	stepToEnd(c)
}

type statistics struct {
	count       uint64
	singleCount map[uint64]uint64
	pairCount   map[uint64]uint64
	tripleCount map[uint64]uint64
	quadCount   map[uint64]uint64
}

func newStatistics() statistics {
	return statistics{
		singleCount: map[uint64]uint64{},
		pairCount:   map[uint64]uint64{},
		tripleCount: map[uint64]uint64{},
		quadCount:   map[uint64]uint64{},
	}
}

func (s *statistics) Insert(src *statistics) {
	s.count += src.count
	for k, v := range src.singleCount {
		s.singleCount[k] += v
	}
	for k, v := range src.pairCount {
		s.pairCount[k] += v
	}
	for k, v := range src.tripleCount {
		s.tripleCount[k] += v
	}
	for k, v := range src.quadCount {
		s.quadCount[k] += v
	}
}

func (s *statistics) Print() {

	type entry struct {
		value uint64
		count uint64
	}

	getTopN := func(data map[uint64]uint64, n int) []entry {
		list := make([]entry, 0, len(data))
		for k, c := range data {
			list = append(list, entry{k, c})
		}
		sort.Slice(list, func(i, j int) bool {
			return list[i].count > list[j].count
		})
		if len(list) < n {
			return list
		}
		return list[0:n]
	}

	log.Printf("\n----- Statistics ------\n")
	log.Printf("\nSteps: %d\n", s.count)
	log.Printf("\nSingles:\n")
	for _, e := range getTopN(s.singleCount, 5) {
		log.Printf("\t%-30v: %d (%.2f%%)\n", OpCode(e.value), e.count, float32(e.count*100)/float32(s.count))
	}
	log.Printf("\nPairs:\n")
	for _, e := range getTopN(s.pairCount, 5) {
		log.Printf("\t%-30v%-30v: %d (%.2f%%)\n", OpCode(e.value>>16), OpCode(e.value), e.count, float32(e.count*100)/float32(s.count))
	}
	log.Printf("\nTriples:\n")
	for _, e := range getTopN(s.tripleCount, 5) {
		log.Printf("\t%-30v%-30v%-30v: %d (%.2f%%)\n", OpCode(e.value>>32), OpCode(e.value>>16), OpCode(e.value), e.count, float32(e.count*100)/float32(s.count))
	}

	log.Printf("\nQuads:\n")
	for _, e := range getTopN(s.quadCount, 5) {
		log.Printf("\t%-30v%-30v%-30v%-30v: %d (%.2f%%)\n", OpCode(e.value>>48), OpCode(e.value>>32), OpCode(e.value>>16), OpCode(e.value), e.count, float32(e.count*100)/float32(s.count))
	}
	log.Printf("\n")
}

type statsCollector struct {
	stats statistics

	last       uint64
	secondLast uint64
	thirdLast  uint64
}

func (s *statsCollector) NextOp(op OpCode) {
	if op > 255 {
		panic("Instruction sequence statistics does not support opcodes > 255")
	}
	cur := uint64(op)
	s.stats.count++
	s.stats.singleCount[cur]++
	if s.stats.count == 1 {
		s.last, s.secondLast, s.thirdLast = cur, s.last, s.secondLast
		return
	}
	s.stats.pairCount[s.last<<16|cur]++
	if s.stats.count == 2 {
		s.last, s.secondLast, s.thirdLast = cur, s.last, s.secondLast
		return
	}
	s.stats.tripleCount[s.secondLast<<32|s.last<<16|cur]++
	if s.stats.count == 3 {
		s.last, s.secondLast, s.thirdLast = cur, s.last, s.secondLast
		return
	}
	s.stats.quadCount[s.thirdLast<<48|s.secondLast<<32|s.last<<16|cur]++
	s.last, s.secondLast, s.thirdLast = cur, s.last, s.secondLast
}

// TODO: get rid fo this global values
var globalStatisticsMutex = sync.Mutex{}
var globalStatistics = newStatistics()

func printCollectedInstructionStatistics() {
	globalStatisticsMutex.Lock()
	defer globalStatisticsMutex.Unlock()
	globalStatistics.Print()
}

func resetCollectedInstructionStatistics() {
	globalStatisticsMutex.Lock()
	defer globalStatisticsMutex.Unlock()
	globalStatistics = newStatistics()
}

func runWithStatistics(c *context) {
	stats := statsCollector{stats: newStatistics()}
	for c.status == statusRunning {
		stats.NextOp(c.code[c.pc].opcode)
		step(c)
	}
	globalStatisticsMutex.Lock()
	defer globalStatisticsMutex.Unlock()
	globalStatistics.Insert(&stats.stats)
}

func runWithLogging(c *context) {
	for c.status == statusRunning {
		// log format: <op>, <gas>, <top-of-stack>\n
		if int(c.pc) < len(c.code) {
			top := "-empty-"
			if c.stack.len() > 0 {
				top = c.stack.peek().ToBig().String()
			}
			fmt.Printf("%v, %d, %v\n", c.code[c.pc].opcode, c.gas, top)
		}
		step(c)
	}
}

func step(c *context) {
	steps(c, true)
}

func stepToEnd(c *context) {
	steps(c, false)
}

func checkStackBoundary(c *context, op OpCode) error {
	stackLen := c.stack.len()
	if stackLen < staticStackBoundary[op].stackMin {
		c.status = statusError
		return errStackUnderflow
	}
	if stackLen > staticStackBoundary[op].stackMax {
		c.status = statusError
		return errStackOverflow
	}
	return nil
}

func steps(c *context, oneStepOnly bool) {
	// Idea: handle static gas price in static dispatch below (saves an array lookup)
	staticGasPrices := getStaticGasPrices(c.isBerlin())
	for c.status == statusRunning {
		if int(c.pc) >= len(c.code) {
			opStop(c)
			return
		}

		op := c.code[c.pc].opcode

		// JUMP_TO is an LFVM specific operation that has no gas costs nor stack usage.
		if c.code[c.pc].opcode == JUMP_TO {
			c.pc = int32(c.code[c.pc].arg)
			op = c.code[c.pc].opcode
		}

		// Catch invalid op-codes here, to avoid the need to check them at other places multiple times.
		if op >= NUM_EXECUTABLE_OPCODES {
			c.SignalError(errInvalidCode)
			return
		}

		// Need to check Call stack boundary before using static gas
		if op == CALL && checkStackBoundary(c, op) != nil {
			return
		}

		// If the interpreter is operating in readonly mode, make sure no
		// state-modifying operation is performed. The 3rd stack item
		// for a call operation is the value. Transferring value from one
		// account to the others means the state is modified and should also
		// return with an error.
		if c.params.Static && (isWriteInstruction(op) || (op == CALL && c.stack.Back(2).Sign() != 0)) {
			c.status = statusError
			return
		}

		// Check stack boundary for every instruction
		if checkStackBoundary(c, op) != nil {
			return
		}

		// Consume static gas price for instruction before execution
		if !c.UseGas(staticGasPrices[op]) {
			return
		}

		// Execute instruction
		switch op {
		case POP:
			opPop(c)
		case PUSH0:
			opPush0(c)
		case PUSH1:
			opPush1(c)
		case PUSH2:
			opPush2(c)
		case PUSH3:
			opPush3(c)
		case PUSH4:
			opPush4(c)
		case PUSH5:
			opPush(c, 5)
		case PUSH31:
			opPush(c, 31)
		case PUSH32:
			opPush32(c)
		case JUMP:
			opJump(c)
		case JUMPDEST:
			// nothing
		case SWAP1:
			opSwap(c, 1)
		case SWAP2:
			opSwap(c, 2)
		case DUP3:
			opDup(c, 3)
		case AND:
			opAnd(c)
		case SWAP3:
			opSwap(c, 3)
		case JUMPI:
			opJumpi(c)
		case GT:
			opGt(c)
		case DUP4:
			opDup(c, 4)
		case DUP2:
			opDup(c, 2)
		case ISZERO:
			opIszero(c)
		case ADD:
			opAdd(c)
		case OR:
			opOr(c)
		case XOR:
			opXor(c)
		case NOT:
			opNot(c)
		case SUB:
			opSub(c)
		case MUL:
			opMul(c)
		case MULMOD:
			opMulMod(c)
		case DIV:
			opDiv(c)
		case SDIV:
			opSDiv(c)
		case MOD:
			opMod(c)
		case SMOD:
			opSMod(c)
		case ADDMOD:
			opAddMod(c)
		case EXP:
			opExp(c)
		case DUP5:
			opDup(c, 5)
		case DUP1:
			opDup(c, 1)
		case EQ:
			opEq(c)
		case PC:
			opPc(c)
		case CALLER:
			opCaller(c)
		case CALLDATALOAD:
			opCallDataload(c)
		case CALLDATASIZE:
			opCallDatasize(c)
		case CALLDATACOPY:
			opCallDataCopy(c)
		case MLOAD:
			opMload(c)
		case MSTORE:
			opMstore(c)
		case MSTORE8:
			opMstore8(c)
		case MSIZE:
			opMsize(c)
		case MCOPY:
			opMcopy(c)
		case LT:
			opLt(c)
		case SLT:
			opSlt(c)
		case SGT:
			opSgt(c)
		case SHR:
			opShr(c)
		case SHL:
			opShl(c)
		case SAR:
			opSar(c)
		case SIGNEXTEND:
			opSignExtend(c)
		case BYTE:
			opByte(c)
		case SHA3:
			opSha3(c)
		case CALLVALUE:
			opCallvalue(c)
		case PUSH6:
			opPush(c, 6)
		case PUSH7:
			opPush(c, 7)
		case PUSH8:
			opPush(c, 8)
		case PUSH9:
			opPush(c, 9)
		case PUSH10:
			opPush(c, 10)
		case PUSH11:
			opPush(c, 11)
		case PUSH12:
			opPush(c, 12)
		case PUSH13:
			opPush(c, 13)
		case PUSH14:
			opPush(c, 14)
		case PUSH15:
			opPush(c, 15)
		case PUSH16:
			opPush(c, 16)
		case PUSH17:
			opPush(c, 17)
		case PUSH18:
			opPush(c, 18)
		case PUSH19:
			opPush(c, 19)
		case PUSH20:
			opPush(c, 20)
		case PUSH21:
			opPush(c, 21)
		case PUSH22:
			opPush(c, 22)
		case PUSH23:
			opPush(c, 23)
		case PUSH24:
			opPush(c, 24)
		case PUSH25:
			opPush(c, 25)
		case PUSH26:
			opPush(c, 26)
		case PUSH27:
			opPush(c, 27)
		case PUSH28:
			opPush(c, 28)
		case PUSH29:
			opPush(c, 29)
		case PUSH30:
			opPush(c, 30)
		case SWAP4:
			opSwap(c, 4)
		case SWAP5:
			opSwap(c, 5)
		case SWAP6:
			opSwap(c, 6)
		case SWAP7:
			opSwap(c, 7)
		case SWAP8:
			opSwap(c, 8)
		case SWAP9:
			opSwap(c, 9)
		case SWAP10:
			opSwap(c, 10)
		case SWAP11:
			opSwap(c, 11)
		case SWAP12:
			opSwap(c, 12)
		case SWAP13:
			opSwap(c, 13)
		case SWAP14:
			opSwap(c, 14)
		case SWAP15:
			opSwap(c, 15)
		case SWAP16:
			opSwap(c, 16)
		case DUP6:
			opDup(c, 6)
		case DUP7:
			opDup(c, 7)
		case DUP8:
			opDup(c, 8)
		case DUP9:
			opDup(c, 9)
		case DUP10:
			opDup(c, 10)
		case DUP11:
			opDup(c, 11)
		case DUP12:
			opDup(c, 12)
		case DUP13:
			opDup(c, 13)
		case DUP14:
			opDup(c, 14)
		case DUP15:
			opDup(c, 15)
		case DUP16:
			opDup(c, 16)
		case RETURN:
			opReturn(c)
		case REVERT:
			opRevert(c)
		case JUMP_TO:
			opJumpTo(c)
		case SLOAD:
			opSload(c)
		case SSTORE:
			opSstore(c)
		case TLOAD:
			opTload(c)
		case TSTORE:
			opTstore(c)
		case CODESIZE:
			opCodeSize(c)
		case CODECOPY:
			opCodeCopy(c)
		case EXTCODESIZE:
			opExtcodesize(c)
		case EXTCODEHASH:
			opExtcodehash(c)
		case EXTCODECOPY:
			opExtCodeCopy(c)
		case BALANCE:
			opBalance(c)
		case SELFBALANCE:
			opSelfbalance(c)
		case BASEFEE:
			opBaseFee(c)
		case BLOBHASH:
			opBlobHash(c)
		case BLOBBASEFEE:
			opBlobBaseFee(c)
		case SELFDESTRUCT:
			opSelfdestruct(c)
		case CHAINID:
			opChainId(c)
		case GAS:
			opGas(c)
		case PREVRANDAO:
			opPrevRandao(c)
		case TIMESTAMP:
			opTimestamp(c)
		case NUMBER:
			opNumber(c)
		case GASLIMIT:
			opGasLimit(c)
		case GASPRICE:
			opGasPrice(c)
		case CALL:
			opCall(c)
		case CALLCODE:
			opCallCode(c)
		case STATICCALL:
			opStaticCall(c)
		case DELEGATECALL:
			opDelegateCall(c)
		case RETURNDATASIZE:
			opReturnDataSize(c)
		case RETURNDATACOPY:
			opReturnDataCopy(c)
		case BLOCKHASH:
			opBlockhash(c)
		case COINBASE:
			opCoinbase(c)
		case ORIGIN:
			opOrigin(c)
		case ADDRESS:
			opAddress(c)
		case STOP:
			opStop(c)
		case CREATE:
			opCreate(c)
		case CREATE2:
			opCreate2(c)
		case LOG0:
			opLog(c, 0)
		case LOG1:
			opLog(c, 1)
		case LOG2:
			opLog(c, 2)
		case LOG3:
			opLog(c, 3)
		case LOG4:
			opLog(c, 4)
		// --- Super Instructions ---
		case SWAP2_SWAP1_POP_JUMP:
			opSwap2_Swap1_Pop_Jump(c)
		case SWAP1_POP_SWAP2_SWAP1:
			opSwap1_Pop_Swap2_Swap1(c)
		case POP_SWAP2_SWAP1_POP:
			opPop_Swap2_Swap1_Pop(c)
		case POP_POP:
			opPopPop(c)
		case PUSH1_SHL:
			opPush1_Shl(c)
		case PUSH1_ADD:
			opPush1_Add(c)
		case PUSH1_DUP1:
			opPush1_Dup1(c)
		case PUSH2_JUMP:
			opPush2_Jump(c)
		case PUSH2_JUMPI:
			opPush2_Jumpi(c)
		case PUSH1_PUSH1:
			opPush1_Push1(c)
		case SWAP1_POP:
			opSwap1_Pop(c)
		case POP_JUMP:
			opPop_Jump(c)
		case SWAP2_SWAP1:
			opSwap2_Swap1(c)
		case SWAP2_POP:
			opSwap2_Pop(c)
		case DUP2_MSTORE:
			opDup2_Mstore(c)
		case DUP2_LT:
			opDup2_Lt(c)
		case ISZERO_PUSH2_JUMPI:
			opIsZero_Push2_Jumpi(c)
		case PUSH1_PUSH4_DUP3:
			opPush1_Push4_Dup3(c)
		case AND_SWAP1_POP_SWAP2_SWAP1:
			opAnd_Swap1_Pop_Swap2_Swap1(c)
		case PUSH1_PUSH1_PUSH1_SHL_SUB:
			opPush1_Push1_Push1_Shl_Sub(c)
		default:
			c.status = statusInvalidInstruction
			return
		}
		c.pc++

		if oneStepOnly {
			return
		}
	}
}

func isWriteInstruction(opCode OpCode) bool {
	const mask uint32 = 1 | // = 1 << (SSTORE - SSTORE) |
		1<<(LOG0-SSTORE) |
		1<<(LOG1-SSTORE) |
		1<<(LOG2-SSTORE) |
		1<<(LOG3-SSTORE) |
		1<<(LOG4-SSTORE) |
		1<<(CREATE-SSTORE) |
		1<<(CREATE2-SSTORE) |
		1<<(SELFDESTRUCT-SSTORE) |
		1<<(TSTORE-SSTORE)

	return SSTORE <= opCode && mask&(1<<(opCode-SSTORE)) != 0
}
