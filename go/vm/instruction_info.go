package vm_test

import (
	"fmt"

	"github.com/ethereum/go-ethereum/core/vm"
)

// Revision references a EVM specification version.
type Revision int

const (
	Istanbul Revision = 1
	Berlin   Revision = 2
	London   Revision = 3

	LatestRevision = London

	// Chain config for hardforks
	ISTANBUL_FORK = 00
	BERLIN_FORK   = 10
	LONDON_FORK   = 20
)

func (r Revision) String() string {
	switch r {
	case Istanbul:
		return "Istanbul"
	case Berlin:
		return "Berlin"
	case London:
		return "London"
	}
	return "Unknown"
}

func (r Revision) GetForkBlock() int64 {
	switch r {
	case Istanbul:
		return ISTANBUL_FORK
	case Berlin:
		return BERLIN_FORK
	case London:
		return LONDON_FORK
	}
	panic(fmt.Sprintf("unknown revision: %v", r))
}

// revisions lists all revisions covered by the tests in this package.
var revisions = []Revision{Istanbul, Berlin, London}

// InstructionInfo contains meta-information about instructions used for
// generating test cases.
type InstructionInfo struct {
	// add information as needed
}

// getInstructions returns a map of OpCodes for the respective revision.
func getInstructions(revision Revision) map[vm.OpCode]InstructionInfo {
	switch revision {
	case Istanbul:
		return getInstanbulInstructions()
	case Berlin:
		return getBerlinInstructions()
	case London:
		return getLondonInstructions()
	}
	panic(fmt.Sprintf("unknown revision: %v", revision))
}

func getInstanbulInstructions() map[vm.OpCode]InstructionInfo {
	res := map[vm.OpCode]InstructionInfo{
		vm.STOP:           {},
		vm.ADD:            {},
		vm.MUL:            {},
		vm.SUB:            {},
		vm.DIV:            {},
		vm.SDIV:           {},
		vm.MOD:            {},
		vm.SMOD:           {},
		vm.ADDMOD:         {},
		vm.MULMOD:         {},
		vm.EXP:            {},
		vm.SIGNEXTEND:     {},
		vm.LT:             {},
		vm.GT:             {},
		vm.SLT:            {},
		vm.SGT:            {},
		vm.EQ:             {},
		vm.ISZERO:         {},
		vm.AND:            {},
		vm.XOR:            {},
		vm.OR:             {},
		vm.NOT:            {},
		vm.SHL:            {},
		vm.SHR:            {},
		vm.SAR:            {},
		vm.BYTE:           {},
		vm.SHA3:           {},
		vm.ADDRESS:        {},
		vm.BALANCE:        {},
		vm.ORIGIN:         {},
		vm.CALLER:         {},
		vm.CALLVALUE:      {},
		vm.CALLDATALOAD:   {},
		vm.CALLDATASIZE:   {},
		vm.CALLDATACOPY:   {},
		vm.CODESIZE:       {},
		vm.CODECOPY:       {},
		vm.GASPRICE:       {},
		vm.EXTCODESIZE:    {},
		vm.EXTCODECOPY:    {},
		vm.RETURNDATASIZE: {},
		vm.RETURNDATACOPY: {},
		vm.EXTCODEHASH:    {},
		vm.BLOCKHASH:      {},
		vm.COINBASE:       {},
		vm.TIMESTAMP:      {},
		vm.NUMBER:         {},
		vm.DIFFICULTY:     {},
		vm.GASLIMIT:       {},
		vm.CHAINID:        {},
		vm.SELFBALANCE:    {},
		vm.POP:            {},
		vm.MLOAD:          {},
		vm.MSTORE:         {},
		vm.MSTORE8:        {},
		vm.SLOAD:          {},
		vm.SSTORE:         {},
		vm.JUMP:           {},
		vm.JUMPI:          {},
		vm.PC:             {},
		vm.MSIZE:          {},
		vm.GAS:            {},
		vm.JUMPDEST:       {},
		vm.PUSH1:          {},
		vm.PUSH2:          {},
		vm.PUSH3:          {},
		vm.PUSH4:          {},
		vm.PUSH5:          {},
		vm.PUSH6:          {},
		vm.PUSH7:          {},
		vm.PUSH8:          {},
		vm.PUSH9:          {},
		vm.PUSH10:         {},
		vm.PUSH11:         {},
		vm.PUSH12:         {},
		vm.PUSH13:         {},
		vm.PUSH14:         {},
		vm.PUSH15:         {},
		vm.PUSH16:         {},
		vm.PUSH17:         {},
		vm.PUSH18:         {},
		vm.PUSH19:         {},
		vm.PUSH20:         {},
		vm.PUSH21:         {},
		vm.PUSH22:         {},
		vm.PUSH23:         {},
		vm.PUSH24:         {},
		vm.PUSH25:         {},
		vm.PUSH26:         {},
		vm.PUSH27:         {},
		vm.PUSH28:         {},
		vm.PUSH29:         {},
		vm.PUSH30:         {},
		vm.PUSH31:         {},
		vm.PUSH32:         {},
		vm.DUP1:           {},
		vm.DUP2:           {},
		vm.DUP3:           {},
		vm.DUP4:           {},
		vm.DUP5:           {},
		vm.DUP6:           {},
		vm.DUP7:           {},
		vm.DUP8:           {},
		vm.DUP9:           {},
		vm.DUP10:          {},
		vm.DUP11:          {},
		vm.DUP12:          {},
		vm.DUP13:          {},
		vm.DUP14:          {},
		vm.DUP15:          {},
		vm.DUP16:          {},
		vm.SWAP1:          {},
		vm.SWAP2:          {},
		vm.SWAP3:          {},
		vm.SWAP4:          {},
		vm.SWAP5:          {},
		vm.SWAP6:          {},
		vm.SWAP7:          {},
		vm.SWAP8:          {},
		vm.SWAP9:          {},
		vm.SWAP10:         {},
		vm.SWAP11:         {},
		vm.SWAP12:         {},
		vm.SWAP13:         {},
		vm.SWAP14:         {},
		vm.SWAP15:         {},
		vm.SWAP16:         {},
		vm.LOG0:           {},
		vm.LOG1:           {},
		vm.LOG2:           {},
		vm.LOG3:           {},
		vm.LOG4:           {},
		vm.CREATE:         {},
		vm.CALL:           {},
		vm.CALLCODE:       {},
		vm.RETURN:         {},
		vm.DELEGATECALL:   {},
		vm.CREATE2:        {},
		vm.STATICCALL:     {},
		vm.REVERT:         {},
		vm.SELFDESTRUCT:   {},
	}
	return res
}

func getBerlinInstructions() map[vm.OpCode]InstructionInfo {
	// Berlin only modifies gas computations.
	// https://eips.ethereum.org/EIPS/eip-2929
	return getInstanbulInstructions()
}

func getLondonInstructions() map[vm.OpCode]InstructionInfo {
	res := getBerlinInstructions()
	// One additional instruction: BASEFEE
	// https://eips.ethereum.org/EIPS/eip-3198
	res[vm.BASEFEE] = InstructionInfo{}
	return res
}
