package lfvm

import (
	"bytes"
	"fmt"
	"hash"
	"sort"
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/params"
	"github.com/holiman/uint256"
)

type Status byte

const (
	RUNNING Status = iota
	STOPPED
	REVERTED
	RETURNED
	SUICIDED
	INVALID_INSTRUCTION
	OUT_OF_GAS
	SEGMENTATION_FAULT
	ERROR
)

// keccakState wraps sha3.state. In addition to the usual hash methods, it also supports
// Read to get a variable amount of data from the hash state. Read is faster than Sum
// because it doesn't copy the internal state, but also modifies the internal state.
type keccakState interface {
	hash.Hash
	Read([]byte) (int, error)
}

type context struct {
	// Context instances
	evm *vm.EVM

	// Execution state
	pc      int32
	stack   *Stack
	memory  *Memory
	stateDB vm.StateDB
	status  Status
	err     error

	// Inputs
	contract *vm.Contract
	code     Code
	data     []byte
	callsize uint256.Int
	readOnly bool
	isBerlin bool
	isLondon bool
	shaCache bool

	// Intermediate data
	return_data []byte
	hasher      keccakState // Keccak256 hasher instance shared across opcodes
	hasherBuf   common.Hash

	// Outputs
	result_offset uint256.Int
	result_size   uint256.Int

	// Debugging
	interpreter *vm.InterpreterState
}

func (c *context) UseGas(amount uint64) bool {
	if c.contract.UseGas(amount) {
		return true
	}
	c.status = OUT_OF_GAS
	return false
}

func (c *context) SignalError(err error) {
	c.status = ERROR
	c.err = err
}

func (c *context) IsShadowed() bool {
	return c.interpreter != nil
}

func Run(evm *vm.EVM, cfg vm.Config, contract *vm.Contract, code Code, data []byte, readOnly bool, state vm.StateDB, with_shadow_vm, with_statistics bool, no_shaCache bool) ([]byte, error) {
	if evm.Depth == 0 {
		ClearShadowValues()
	}

	// Don't bother with the execution if there's no code.
	if len(contract.Code) == 0 {
		return nil, nil
	}

	// Increment the call depth which is restricted to 1024
	evm.Depth++
	defer func() { evm.Depth-- }()

	main_evm := *evm

	var shadow_interpreter *vm.InterpreterState
	if with_shadow_vm {
		// Set up a shadow context for the EVM implementation
		shadow_contract := *contract
		shadow_evm := *evm
		shadow_call_context := ShadowCallContext{}
		shadow_evm.CallContext = &shadow_call_context
		shadow_evm.StateDB = ShadowStateDB{state: state}

		// Introduce an interceptor for recursive EVM calls
		main_evm.CallContext = &CaptureCallContext{evm, &shadow_call_context}

		// Start shadow interceptor
		shadow_interpreter = vm.NewEVMInterpreter(&shadow_evm, cfg).Start(&shadow_contract, data, readOnly)

		defer func() {
			shadow_interpreter.Stop()
		}()
	}

	// Set up execution context.
	var ctxt = context{
		evm:         &main_evm,
		contract:    contract,
		code:        code,
		data:        data,
		stack:       NewStack(),
		memory:      NewMemory(),
		stateDB:     state,
		callsize:    *uint256.NewInt(uint64(len(data))),
		interpreter: shadow_interpreter,
		readOnly:    readOnly,
		isBerlin:    evm.ChainConfig().IsBerlin(evm.Context.BlockNumber),
		isLondon:    evm.ChainConfig().IsLondon(evm.Context.BlockNumber),
		shaCache:    !no_shaCache,
	}
	defer func() {
		ReturnStack(ctxt.stack)
	}()

	// Run interpreter.
	if ctxt.IsShadowed() {
		runWithShadowInterpreter(&ctxt)
	} else if with_statistics {
		runWithStatistics(&ctxt)
	} else {
		run(&ctxt)
	}

	var res []byte
	if ctxt.status == RETURNED || ctxt.status == REVERTED {
		// Extract the result from the memory.
		offset := ctxt.result_offset.Uint64()
		size := ctxt.result_size.Uint64()
		res = make([]byte, size)
		if err := ctxt.memory.EnsureCapacity(offset, size, &ctxt); err != nil {
			return nil, err
		}
		ctxt.memory.CopyData(offset, res[:])
	}

	// Handle return status
	switch ctxt.status {
	case STOPPED:
		return nil, nil
	case RETURNED:
		return res, nil
	case REVERTED:
		return res, vm.ErrExecutionReverted
	case SUICIDED:
		return res, nil
	case OUT_OF_GAS:
		return nil, vm.ErrOutOfGas
	case INVALID_INSTRUCTION:
		return nil, vm.ErrInvalidCode
	case SEGMENTATION_FAULT:
		return nil, vm.ErrInvalidCode
	case ERROR:
		if ctxt.err != nil {
			return nil, ctxt.err
		}
		return nil, fmt.Errorf("unspecified error in interpreter")
	}

	if ctxt.err != nil {
		ctxt.status = ERROR
		return nil, fmt.Errorf("unknown interpreter status %d with error %v", ctxt.status, ctxt.err)
	} else {
		return nil, fmt.Errorf("unknown interpreter status %d", ctxt.status)
	}
}

func run(c *context) {
	stepToEnd(c)
}

func runWithShadowInterpreter(c *context) {
	count := 0
	for c.status == RUNNING {

		if int(c.pc) >= len(c.code) {
			opStop(c)
			return
		}

		for c.code[c.pc].opcode == JUMP_TO {
			step(c)
			c.interpreter.Step()
		}
		count++

		// Make a step in this interpreter.
		fmt.Printf("%5d - %v\n", count, c.code[c.pc].opcode)
		step(c)

		// Make a step in the shadow interpreter
		fmt.Printf("%5d - % 92v\n", count, c.interpreter.GetCurrentOpCode())
		c.interpreter.Step()

		// Compare states and look for missalignments.
		if (c.status != RUNNING) != (c.interpreter.IsDone()) {
			fmt.Printf("Left done:  %t\n", c.status != RUNNING)
			fmt.Printf("Right done: %t\n", c.interpreter.IsDone())
			panic("One side terminated while other hasn't")
		}
		if c.status != RUNNING {
			continue
		}
		if c.contract.Gas != c.interpreter.Contract.Gas {
			fmt.Printf("Left:  %v\n", c.contract.Gas)
			fmt.Printf("Right: %v\n", c.interpreter.Contract.Gas)
			fmt.Printf("Diff:  %v\n", int64(c.contract.Gas)-int64(c.interpreter.Contract.Gas))
			panic("Gas value diverged!")
		}
		if c.stack.len() != c.interpreter.Stack.Len() {
			fmt.Printf("Left:  %d\n", c.stack.len())
			fmt.Printf("Right: %d\n", c.interpreter.Stack.Len())
			panic("Stack length diverged!")
		}
		if c.stack.len() > 0 && *c.stack.peek() != *c.interpreter.Stack.Back(0) {
			fmt.Printf("Left:  %v\n", *c.stack.peek())
			fmt.Printf("Right: %v\n", *c.interpreter.Stack.Back(0))
			panic("Stack top value divereged!")
		}
		if c.memory.Len() != uint64(c.interpreter.Memory.Len()) {
			fmt.Printf("Left:  %v\n", c.memory.Len())
			fmt.Printf("Right: %v\n", c.interpreter.Memory.Len())
			panic("Memory size divereged!")
		}
		if !bytes.Equal(c.memory.Data(), c.interpreter.Memory.Data()) {
			fmt.Printf("Left:\n")
			c.memory.Print()
			fmt.Printf("Right:\n")
			c.interpreter.Memory.Print()
			panic("Memory content divereged!")
		}
	}
}

type entry struct {
	value uint64
	count uint64
}

func getTopN(data map[uint64]uint64, n int) []entry {
	list := make([]entry, 0, len(data))
	for k, c := range data {
		list = append(list, entry{k, c})
	}
	sort.Slice(list, func(i, j int) bool { return list[i].count > list[j].count })
	if len(list) < n {
		return list
	}
	return list[0:n]
}

type statistics struct {
	count        uint64
	single_count map[uint64]uint64
	pair_count   map[uint64]uint64
	triple_count map[uint64]uint64
	quad_count   map[uint64]uint64
}

func newStatistics() statistics {
	return statistics{
		single_count: map[uint64]uint64{},
		pair_count:   map[uint64]uint64{},
		triple_count: map[uint64]uint64{},
		quad_count:   map[uint64]uint64{},
	}
}

func (s *statistics) Insert(src *statistics) {
	s.count += src.count
	for k, v := range src.single_count {
		s.single_count[k] += v
	}
	for k, v := range src.pair_count {
		s.pair_count[k] += v
	}
	for k, v := range src.triple_count {
		s.triple_count[k] += v
	}
	for k, v := range src.quad_count {
		s.quad_count[k] += v
	}
}

func (s *statistics) Print() {
	fmt.Printf("\n----- Statistiscs ------\n")
	fmt.Printf("\nSteps: %d\n", s.count)
	fmt.Printf("\nSingels:\n")
	for _, e := range getTopN(s.single_count, 5) {
		fmt.Printf("\t%-30v: %d (%.2f%%)\n", OpCode(e.value), e.count, float32(e.count*100)/float32(s.count))
	}
	fmt.Printf("\nPairs:\n")
	for _, e := range getTopN(s.pair_count, 5) {
		fmt.Printf("\t%-30v%-30v: %d (%.2f%%)\n", OpCode(e.value>>16), OpCode(e.value), e.count, float32(e.count*100)/float32(s.count))
	}
	fmt.Printf("\nTriples:\n")
	for _, e := range getTopN(s.triple_count, 5) {
		fmt.Printf("\t%-30v%-30v%-30v: %d (%.2f%%)\n", OpCode(e.value>>32), OpCode(e.value>>16), OpCode(e.value), e.count, float32(e.count*100)/float32(s.count))
	}

	fmt.Printf("\nQuads:\n")
	for _, e := range getTopN(s.quad_count, 5) {
		fmt.Printf("\t%-30v%-30v%-30v%-30v: %d (%.2f%%)\n", OpCode(e.value>>48), OpCode(e.value>>32), OpCode(e.value>>16), OpCode(e.value), e.count, float32(e.count*100)/float32(s.count))
	}
	fmt.Printf("\n")
}

type stats_collector struct {
	stats statistics

	last        uint64
	second_last uint64
	third_last  uint64
}

func (s *stats_collector) NextOp(op OpCode) {
	if op > 255 {
		panic("Instruction sequence statistics does not support opcodes > 255")
	}
	cur := uint64(op)
	s.stats.count++
	s.stats.single_count[cur]++
	if s.stats.count == 1 {
		s.last, s.second_last, s.third_last = cur, s.last, s.second_last
		return
	}
	s.stats.pair_count[s.last<<16|cur]++
	if s.stats.count == 2 {
		s.last, s.second_last, s.third_last = cur, s.last, s.second_last
		return
	}
	s.stats.triple_count[s.second_last<<32|s.last<<16|cur]++
	if s.stats.count == 3 {
		s.last, s.second_last, s.third_last = cur, s.last, s.second_last
		return
	}
	s.stats.quad_count[s.third_last<<48|s.second_last<<32|s.last<<16|cur]++
	s.last, s.second_last, s.third_last = cur, s.last, s.second_last
}

var global_stats_mu = sync.Mutex{}
var global_statistics = newStatistics()

func PrintCollectedInstructionStatistics() {
	global_stats_mu.Lock()
	defer global_stats_mu.Unlock()
	global_statistics.Print()
}

func runWithStatistics(c *context) {
	stats := stats_collector{stats: newStatistics()}
	for c.status == RUNNING {
		stats.NextOp(c.code[c.pc].opcode)
		step(c)
	}
	global_stats_mu.Lock()
	defer global_stats_mu.Unlock()
	global_statistics.Insert(&stats.stats)
}

func step(c *context) {
	steps(c, true)
}

func stepToEnd(c *context) {
	steps(c, false)
}

func checkStackBoundry(c *context, op OpCode) error {
	stackLen := c.stack.len()
	if stackLen < staticStackBoundry[op].stackMin {
		c.err = &vm.ErrStackUnderflow{}
		c.status = ERROR
		return c.err
	}
	if stackLen > int(params.StackLimit)-1 && stackLen > staticStackBoundry[op].stackMax {
		c.err = &vm.ErrStackOverflow{}
		c.status = ERROR
		return c.err
	}
	return nil
}

func steps(c *context, one_step_only bool) {
	// Idea: handle static gas price in static dispatch below (saves an array lookup)
	static_gas_prices := getStaticGasPrices(c.isBerlin)
	for c.status == RUNNING {
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
			c.SignalError(vm.ErrInvalidCode)
			return
		}

		// Need to check Call stack boundry before using static gas
		if op == CALL && checkStackBoundry(c, op) != nil {
			return
		}

		// If the interpreter is operating in readonly mode, make sure no
		// state-modifying operation is performed. The 3rd stack item
		// for a call operation is the value. Transferring value from one
		// account to the others means the state is modified and should also
		// return with an error.
		if c.readOnly && (isWriteInstruction(op) || (op == CALL && c.stack.Back(2).Sign() != 0)) {
			c.err = vm.ErrWriteProtection
			c.status = ERROR
			return
		}

		// Consume static gas price for instruction before execution
		if !c.UseGas(static_gas_prices[op]) {
			return
		}

		// Check stack boundry for every instruction
		if checkStackBoundry(c, op) != nil {
			return
		}

		// Execute instruction
		switch op {
		case POP:
			opPop(c)
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
		case NOOP:
			opNoop(c)
		case DATA:
			c.status = SEGMENTATION_FAULT
			return
		case INVALID:
			opInvalid(c)
		case SLOAD:
			opSload(c)
		case SSTORE:
			opSstore(c)
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
		case SELFDESTRUCT:
			opSelfdestruct(c)
		case CHAINID:
			opChainId(c)
		case GAS:
			opGas(c)
		case DIFFICULTY:
			opDifficulty(c)
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
			c.status = INVALID_INSTRUCTION
			return
		}
		c.pc++

		if one_step_only {
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
		1<<(SELFDESTRUCT-SSTORE)

	return SSTORE <= opCode && mask&(1<<(opCode-SSTORE)) != 0
}
