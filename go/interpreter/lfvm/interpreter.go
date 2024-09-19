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

	"github.com/Fantom-foundation/Tosca/go/tosca"
	"github.com/holiman/uint256"
)

// status is enumeration of the execution state of an interpreter run.
type status byte

const (
	statusRunning        status = iota // < all fine, ops are processed
	statusStopped                      // < execution stopped with a STOP
	statusReverted                     // < execution stopped with a REVERT
	statusReturned                     // < execution stopped with a RETURN
	statusSelfDestructed               // < execution stopped with a SELF-DESTRUCT
	statusError                        // < execution stopped with an error (e.g. stack underflow, out-of-gas, invalid-opcode)
)

// context is the execution environment of an interpreter run. It contains all
// the necessary state to execute a contract, including input parameters, the
// contract code, and internal execution state such as the program counter,
// stack, and memory. For each contract execution, a new context is created.
type context struct {
	// Inputs
	params  tosca.Parameters
	context tosca.RunContext
	code    Code // the contract code in LFVM format

	// Execution state
	status status
	pc     int32
	gas    tosca.Gas
	refund tosca.Gas
	stack  *stack
	memory *Memory

	// Intermediate data
	returnData []byte // < the result of the last nested contract call

	// Outputs
	resultOffset uint256.Int
	resultSize   uint256.Int

	// Configuration flags
	withShaCache bool
}

// useGas reduces the gas level by the given amount. If the gas level drops
// below zero, the caller should stop the execution with an error status. The function
// returns true if sufficient gas was available and execution can continue,
// false otherwise.
func (c *context) useGas(amount tosca.Gas) error {
	if c.gas < 0 || amount < 0 || c.gas < amount {
		c.gas = 0
		return errOutOfGas
	}
	c.gas -= amount
	return nil
}

// signalError informs the context that an error was encountered that should
// result in the termination of the execution covered by this context.
func (c *context) signalError() {
	c.status = statusError
}

// isAtLeast returns true if the interpreter is is running at least at the given
// revision or newer, false otherwise.
func (c *context) isAtLeast(revision tosca.Revision) bool {
	return c.params.Revision >= revision
}

// --- Interpreter ---

type runner interface {
	run(*context)
}

type interpreterConfig struct {
	withShaCache bool
	runner       runner
}

func run(
	config interpreterConfig,
	params tosca.Parameters,
	code Code,
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
		withShaCache: config.withShaCache,
	}

	defer ReturnStack(ctxt.stack)

	// Run interpreter.
	if config.runner == nil {
		config.runner = vanillaRunner{}
	}
	config.runner.run(&ctxt)

	return generateResult(&ctxt)
}

func generateResult(ctxt *context) (tosca.Result, error) {

	res, err := getOutput(ctxt)
	if err != nil {
		return tosca.Result{Success: false}, nil
	}

	// Handle return status
	switch ctxt.status {
	case statusStopped, statusSelfDestructed:
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
	case statusError:
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
			if err := ctxt.memory.expandMemory(offset, size, ctxt); err != nil {
				return nil, err
			}
			res = make([]byte, size)
			ctxt.memory.CopyData(offset, res)
		}
	}
	return res, nil
}

// --- Runners ---

// vanillaRunner is the default runner that executes the contract code without
// any additional features.
type vanillaRunner struct{}

func (r vanillaRunner) run(c *context) {
	steps(c, false)
}

// loggingRunner is a runner that logs the execution of the contract code to
// stdout. It is used for debugging purposes.
type loggingRunner struct{}

func (r loggingRunner) run(c *context) {
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

// --- Execution ---

func step(c *context) {
	steps(c, true)
}

func steps(c *context, oneStepOnly bool) {
	// Idea: handle static gas price in static dispatch below (saves an array lookup)
	staticGasPrices := getStaticGasPrices(c.params.Revision)
	for c.status == statusRunning {
		if int(c.pc) >= len(c.code) {
			opStop(c)
			return
		}

		op := c.code[c.pc].opcode

		// Check stack boundary for every instruction
		if !satisfiesStackRequirements(c.stack.len(), op) {
			c.signalError()
			return
		}

		// Consume static gas price for instruction before execution
		if err := c.useGas(staticGasPrices.get(op)); err != nil {
			c.signalError()
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
			c.signalError()
			return
		}
		c.pc++

		if oneStepOnly {
			return
		}
	}
}

// satisfiesStackRequirements checks that the opCode will not make an out of
// bounds access with the current stack size.
// Caller should handle false return as an error.
func satisfiesStackRequirements(stackLen int, op OpCode) bool {
	limits := _precomputedStackLimits.get(op)
	if stackLen < limits.min || stackLen > limits.max {
		return false
	}
	return true
}

// stackLimits defines the stack usage of a single OpCode.
type stackLimits struct {
	min int // The minimum stack size required by an OpCode.
	max int // The maximum stack size allowed before running an OpCode.
}

var _precomputedStackLimits = newOpCodePropertyMap(func(op OpCode) stackLimits {
	usage := computeStackUsage(op)
	return stackLimits{
		min: -usage.from,
		max: maxStackSize - usage.to,
	}
})
