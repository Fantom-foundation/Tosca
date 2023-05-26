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
	stack StackUsage
	// add information as needed
}

type StackUsage struct {
	popped int // < the number of elements popped from the stack
	pushed int // < the number of elements pushed on the stack
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
	none := StackUsage{}

	op := func(x int) StackUsage {
		return StackUsage{popped: x, pushed: 1}
	}

	consume := func(x int) StackUsage {
		return StackUsage{popped: x}
	}

	dup := func(x int) StackUsage {
		return StackUsage{popped: x, pushed: x + 1}
	}

	swap := func(x int) StackUsage {
		return StackUsage{popped: x + 1, pushed: x + 1}
	}

	res := map[vm.OpCode]InstructionInfo{
		vm.STOP:           {stack: none},
		vm.ADD:            {stack: op(2)},
		vm.MUL:            {stack: op(2)},
		vm.SUB:            {stack: op(2)},
		vm.DIV:            {stack: op(2)},
		vm.SDIV:           {stack: op(2)},
		vm.MOD:            {stack: op(2)},
		vm.SMOD:           {stack: op(2)},
		vm.ADDMOD:         {stack: op(3)},
		vm.MULMOD:         {stack: op(3)},
		vm.EXP:            {stack: op(2)},
		vm.SIGNEXTEND:     {stack: op(2)},
		vm.LT:             {stack: op(2)},
		vm.GT:             {stack: op(2)},
		vm.SLT:            {stack: op(2)},
		vm.SGT:            {stack: op(2)},
		vm.EQ:             {stack: op(2)},
		vm.ISZERO:         {stack: op(1)},
		vm.AND:            {stack: op(2)},
		vm.XOR:            {stack: op(2)},
		vm.OR:             {stack: op(2)},
		vm.NOT:            {stack: op(1)},
		vm.SHL:            {stack: op(2)},
		vm.SHR:            {stack: op(2)},
		vm.SAR:            {stack: op(2)},
		vm.BYTE:           {stack: op(1)},
		vm.SHA3:           {stack: op(2)},
		vm.ADDRESS:        {stack: op(0)},
		vm.BALANCE:        {stack: op(1)},
		vm.ORIGIN:         {stack: op(0)},
		vm.CALLER:         {stack: op(0)},
		vm.CALLVALUE:      {stack: op(0)},
		vm.CALLDATALOAD:   {stack: op(1)},
		vm.CALLDATASIZE:   {stack: op(0)},
		vm.CALLDATACOPY:   {stack: consume(3)},
		vm.CODESIZE:       {stack: op(0)},
		vm.CODECOPY:       {stack: consume(3)},
		vm.GASPRICE:       {stack: op(0)},
		vm.EXTCODESIZE:    {stack: op(1)},
		vm.EXTCODECOPY:    {stack: consume(4)},
		vm.RETURNDATASIZE: {stack: op(0)},
		vm.RETURNDATACOPY: {stack: consume(3)},
		vm.EXTCODEHASH:    {stack: op(1)},
		vm.BLOCKHASH:      {stack: op(1)},
		vm.COINBASE:       {stack: op(0)},
		vm.TIMESTAMP:      {stack: op(0)},
		vm.NUMBER:         {stack: op(0)},
		vm.DIFFICULTY:     {stack: op(0)},
		vm.GASLIMIT:       {stack: op(0)},
		vm.CHAINID:        {stack: op(0)},
		vm.SELFBALANCE:    {stack: op(0)},
		vm.POP:            {stack: consume(1)},
		vm.MLOAD:          {stack: op(1)},
		vm.MSTORE:         {stack: consume(2)},
		vm.MSTORE8:        {stack: consume(2)},
		vm.SLOAD:          {stack: op(1)},
		vm.SSTORE:         {stack: consume(2)},
		vm.JUMP:           {stack: consume(1)},
		vm.JUMPI:          {stack: consume(2)},
		vm.PC:             {stack: op(0)},
		vm.MSIZE:          {stack: op(0)},
		vm.GAS:            {stack: op(0)},
		vm.JUMPDEST:       {stack: none},
		vm.PUSH1:          {stack: op(0)},
		vm.PUSH2:          {stack: op(0)},
		vm.PUSH3:          {stack: op(0)},
		vm.PUSH4:          {stack: op(0)},
		vm.PUSH5:          {stack: op(0)},
		vm.PUSH6:          {stack: op(0)},
		vm.PUSH7:          {stack: op(0)},
		vm.PUSH8:          {stack: op(0)},
		vm.PUSH9:          {stack: op(0)},
		vm.PUSH10:         {stack: op(0)},
		vm.PUSH11:         {stack: op(0)},
		vm.PUSH12:         {stack: op(0)},
		vm.PUSH13:         {stack: op(0)},
		vm.PUSH14:         {stack: op(0)},
		vm.PUSH15:         {stack: op(0)},
		vm.PUSH16:         {stack: op(0)},
		vm.PUSH17:         {stack: op(0)},
		vm.PUSH18:         {stack: op(0)},
		vm.PUSH19:         {stack: op(0)},
		vm.PUSH20:         {stack: op(0)},
		vm.PUSH21:         {stack: op(0)},
		vm.PUSH22:         {stack: op(0)},
		vm.PUSH23:         {stack: op(0)},
		vm.PUSH24:         {stack: op(0)},
		vm.PUSH25:         {stack: op(0)},
		vm.PUSH26:         {stack: op(0)},
		vm.PUSH27:         {stack: op(0)},
		vm.PUSH28:         {stack: op(0)},
		vm.PUSH29:         {stack: op(0)},
		vm.PUSH30:         {stack: op(0)},
		vm.PUSH31:         {stack: op(0)},
		vm.PUSH32:         {stack: op(0)},
		vm.DUP1:           {stack: dup(1)},
		vm.DUP2:           {stack: dup(2)},
		vm.DUP3:           {stack: dup(3)},
		vm.DUP4:           {stack: dup(4)},
		vm.DUP5:           {stack: dup(5)},
		vm.DUP6:           {stack: dup(6)},
		vm.DUP7:           {stack: dup(7)},
		vm.DUP8:           {stack: dup(8)},
		vm.DUP9:           {stack: dup(9)},
		vm.DUP10:          {stack: dup(10)},
		vm.DUP11:          {stack: dup(11)},
		vm.DUP12:          {stack: dup(12)},
		vm.DUP13:          {stack: dup(13)},
		vm.DUP14:          {stack: dup(14)},
		vm.DUP15:          {stack: dup(15)},
		vm.DUP16:          {stack: dup(16)},
		vm.SWAP1:          {stack: swap(1)},
		vm.SWAP2:          {stack: swap(2)},
		vm.SWAP3:          {stack: swap(3)},
		vm.SWAP4:          {stack: swap(4)},
		vm.SWAP5:          {stack: swap(5)},
		vm.SWAP6:          {stack: swap(6)},
		vm.SWAP7:          {stack: swap(7)},
		vm.SWAP8:          {stack: swap(8)},
		vm.SWAP9:          {stack: swap(9)},
		vm.SWAP10:         {stack: swap(10)},
		vm.SWAP11:         {stack: swap(11)},
		vm.SWAP12:         {stack: swap(12)},
		vm.SWAP13:         {stack: swap(13)},
		vm.SWAP14:         {stack: swap(14)},
		vm.SWAP15:         {stack: swap(15)},
		vm.SWAP16:         {stack: swap(16)},
		vm.LOG0:           {stack: consume(2)},
		vm.LOG1:           {stack: consume(3)},
		vm.LOG2:           {stack: consume(4)},
		vm.LOG3:           {stack: consume(5)},
		vm.LOG4:           {stack: consume(6)},
		vm.CREATE:         {stack: op(3)},
		vm.CALL:           {stack: op(7)},
		vm.CALLCODE:       {stack: op(7)},
		vm.RETURN:         {stack: consume(2)},
		vm.DELEGATECALL:   {stack: op(6)},
		vm.CREATE2:        {stack: op(4)},
		vm.STATICCALL:     {stack: op(6)},
		vm.REVERT:         {stack: consume(2)},
		vm.SELFDESTRUCT:   {stack: consume(1)},
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
	res[vm.BASEFEE] = InstructionInfo{
		stack: StackUsage{pushed: 1},
	}
	return res
}
