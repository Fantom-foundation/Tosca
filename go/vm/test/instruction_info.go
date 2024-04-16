//
// Copyright (c) 2024 Fantom Foundation
//
// Use of this software is governed by the Business Source License included
// in the LICENSE.TXT file and at fantom.foundation/bsl11.
//
// Change Date: 2028-4-16
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by the GNU Lesser General Public Licence v3
//

package vm_test

import (
	"fmt"

	"github.com/Fantom-foundation/Tosca/go/vm"

	// This is only imported to get the EVM opcode definitions.
	// TODO: write up our own op-code definition and remove this dependency.
	evm "github.com/ethereum/go-ethereum/core/vm"
)

// Revision references a EVM specification version.
// TODO: remove this and replace it with vm.Revision
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
	gas   GasUsage
	// add information as needed
}

type StackUsage struct {
	popped int // < the number of elements popped from the stack
	pushed int // < the number of elements pushed on the stack
}

type GasUsage struct {
	static  vm.Gas
	dynamic func(revision Revision) []*DynGasTest
}

// getInstructions returns a map of OpCodes for the respective revision.
func getInstructions(revision Revision) map[evm.OpCode]*InstructionInfo {
	switch revision {
	case Istanbul:
		return getIstanbulInstructions()
	case Berlin:
		return getBerlinInstructions()
	case London:
		return getLondonInstructions()
	}
	panic(fmt.Sprintf("unknown revision: %v", revision))
}

func getIstanbulInstructions() map[evm.OpCode]*InstructionInfo {
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

	const gasJumpDest vm.Gas = 1
	const gasQuickStep vm.Gas = 2
	const gasFastestStep vm.Gas = 3
	const gasFastStep vm.Gas = 5
	const gasMidStep vm.Gas = 8
	const gasSlowStep vm.Gas = 10
	const gasBalance vm.Gas = 700
	const gasExtStep vm.Gas = 20
	const gasExtCode vm.Gas = 700
	const gasSha3 vm.Gas = 30
	const gasSloadEIP2200 vm.Gas = 800
	const gasExtCodeHash vm.Gas = 700
	const gasCallEIP150 vm.Gas = 700
	const gasCreate vm.Gas = 32000

	noGas := GasUsage{0, nil}

	gas := func(static vm.Gas, dynamic func(revision Revision) []*DynGasTest) GasUsage {
		return GasUsage{static, dynamic}
	}

	gasD := func(dynamic func(revision Revision) []*DynGasTest) GasUsage {
		return GasUsage{0, dynamic}
	}

	gasS := func(static vm.Gas) GasUsage {
		return GasUsage{static, nil}
	}

	res := map[evm.OpCode]*InstructionInfo{
		evm.STOP:           {stack: none, gas: noGas},
		evm.ADD:            {stack: op(2), gas: gasS(gasFastestStep)},
		evm.MUL:            {stack: op(2), gas: gasS(gasFastStep)},
		evm.SUB:            {stack: op(2), gas: gasS(gasFastestStep)},
		evm.DIV:            {stack: op(2), gas: gasS(gasFastStep)},
		evm.SDIV:           {stack: op(2), gas: gasS(gasFastStep)},
		evm.MOD:            {stack: op(2), gas: gasS(gasFastStep)},
		evm.SMOD:           {stack: op(2), gas: gasS(gasFastStep)},
		evm.ADDMOD:         {stack: op(3), gas: gasS(gasMidStep)},
		evm.MULMOD:         {stack: op(3), gas: gasS(gasMidStep)},
		evm.EXP:            {stack: op(2), gas: gasD(gasEXP)},
		evm.SIGNEXTEND:     {stack: op(2), gas: gasS(gasFastStep)},
		evm.LT:             {stack: op(2), gas: gasS(gasFastestStep)},
		evm.GT:             {stack: op(2), gas: gasS(gasFastestStep)},
		evm.SLT:            {stack: op(2), gas: gasS(gasFastestStep)},
		evm.SGT:            {stack: op(2), gas: gasS(gasFastestStep)},
		evm.EQ:             {stack: op(2), gas: gasS(gasFastestStep)},
		evm.ISZERO:         {stack: op(1), gas: gasS(gasFastestStep)},
		evm.AND:            {stack: op(2), gas: gasS(gasFastestStep)},
		evm.XOR:            {stack: op(2), gas: gasS(gasFastestStep)},
		evm.OR:             {stack: op(2), gas: gasS(gasFastestStep)},
		evm.NOT:            {stack: op(1), gas: gasS(gasFastestStep)},
		evm.BYTE:           {stack: op(2), gas: gasS(gasFastestStep)},
		evm.SHL:            {stack: op(2), gas: gasS(gasFastestStep)},
		evm.SHR:            {stack: op(2), gas: gasS(gasFastestStep)},
		evm.SAR:            {stack: op(2), gas: gasS(gasFastestStep)},
		evm.SHA3:           {stack: op(2), gas: gas(gasSha3, gasDynamicSHA3)},
		evm.ADDRESS:        {stack: op(0), gas: gasS(gasQuickStep)},
		evm.BALANCE:        {stack: op(1), gas: gasS(gasBalance)},
		evm.ORIGIN:         {stack: op(0), gas: gasS(gasQuickStep)},
		evm.CALLER:         {stack: op(0), gas: gasS(gasQuickStep)},
		evm.CALLVALUE:      {stack: op(0), gas: gasS(gasQuickStep)},
		evm.CALLDATALOAD:   {stack: op(1), gas: gasS(gasFastestStep)},
		evm.CALLDATASIZE:   {stack: op(0), gas: gasS(gasQuickStep)},
		evm.CALLDATACOPY:   {stack: consume(3), gas: gas(gasFastestStep, gasDynamicCopy)},
		evm.CODESIZE:       {stack: op(0), gas: gasS(gasQuickStep)},
		evm.CODECOPY:       {stack: consume(3), gas: gas(gasFastestStep, gasDynamicCopy)},
		evm.GASPRICE:       {stack: op(0), gas: gasS(gasQuickStep)},
		evm.EXTCODESIZE:    {stack: op(1), gas: gasS(gasExtCode)},
		evm.EXTCODECOPY:    {stack: consume(4), gas: gas(gasExtCode, gasDynamicExtCodeCopy)},
		evm.RETURNDATASIZE: {stack: op(0), gas: gasS(gasQuickStep)},
		evm.RETURNDATACOPY: {stack: consume(3), gas: gas(gasFastestStep, gasDynamicCopy)},
		evm.EXTCODEHASH:    {stack: op(1), gas: gasS(gasExtCodeHash)},
		evm.BLOCKHASH:      {stack: op(1), gas: gasS(gasExtStep)},
		evm.COINBASE:       {stack: op(0), gas: gasS(gasQuickStep)},
		evm.TIMESTAMP:      {stack: op(0), gas: gasS(gasQuickStep)},
		evm.NUMBER:         {stack: op(0), gas: gasS(gasQuickStep)},
		evm.DIFFICULTY:     {stack: op(0), gas: gasS(gasQuickStep)},
		evm.GASLIMIT:       {stack: op(0), gas: gasS(gasQuickStep)},
		evm.CHAINID:        {stack: op(0), gas: gasS(gasQuickStep)},
		evm.SELFBALANCE:    {stack: op(0), gas: gasS(gasFastStep)},
		evm.POP:            {stack: consume(1), gas: gasS(gasQuickStep)},
		evm.MLOAD:          {stack: op(1), gas: gas(gasFastestStep, gasDynamicMemory)},
		evm.MSTORE:         {stack: consume(2), gas: gas(gasFastestStep, gasDynamicMemory)},
		evm.MSTORE8:        {stack: consume(2), gas: gas(gasFastestStep, gasDynamicMemory)},
		evm.SLOAD:          {stack: op(1), gas: gasS(gasSloadEIP2200)},
		evm.SSTORE:         {stack: consume(2), gas: gas(0, gasDynamicSStore)},
		evm.JUMP:           {stack: consume(1), gas: gasS(gasMidStep)},
		evm.JUMPI:          {stack: consume(2), gas: gasS(gasSlowStep)},
		evm.PC:             {stack: op(0), gas: gasS(gasQuickStep)},
		evm.MSIZE:          {stack: op(0), gas: gasS(gasQuickStep)},
		evm.GAS:            {stack: op(0), gas: gasS(gasQuickStep)},
		evm.JUMPDEST:       {stack: none, gas: gasS(gasJumpDest)},
		evm.PUSH1:          {stack: op(0), gas: gasS(gasFastestStep)},
		evm.PUSH2:          {stack: op(0), gas: gasS(gasFastestStep)},
		evm.PUSH3:          {stack: op(0), gas: gasS(gasFastestStep)},
		evm.PUSH4:          {stack: op(0), gas: gasS(gasFastestStep)},
		evm.PUSH5:          {stack: op(0), gas: gasS(gasFastestStep)},
		evm.PUSH6:          {stack: op(0), gas: gasS(gasFastestStep)},
		evm.PUSH7:          {stack: op(0), gas: gasS(gasFastestStep)},
		evm.PUSH8:          {stack: op(0), gas: gasS(gasFastestStep)},
		evm.PUSH9:          {stack: op(0), gas: gasS(gasFastestStep)},
		evm.PUSH10:         {stack: op(0), gas: gasS(gasFastestStep)},
		evm.PUSH11:         {stack: op(0), gas: gasS(gasFastestStep)},
		evm.PUSH12:         {stack: op(0), gas: gasS(gasFastestStep)},
		evm.PUSH13:         {stack: op(0), gas: gasS(gasFastestStep)},
		evm.PUSH14:         {stack: op(0), gas: gasS(gasFastestStep)},
		evm.PUSH15:         {stack: op(0), gas: gasS(gasFastestStep)},
		evm.PUSH16:         {stack: op(0), gas: gasS(gasFastestStep)},
		evm.PUSH17:         {stack: op(0), gas: gasS(gasFastestStep)},
		evm.PUSH18:         {stack: op(0), gas: gasS(gasFastestStep)},
		evm.PUSH19:         {stack: op(0), gas: gasS(gasFastestStep)},
		evm.PUSH20:         {stack: op(0), gas: gasS(gasFastestStep)},
		evm.PUSH21:         {stack: op(0), gas: gasS(gasFastestStep)},
		evm.PUSH22:         {stack: op(0), gas: gasS(gasFastestStep)},
		evm.PUSH23:         {stack: op(0), gas: gasS(gasFastestStep)},
		evm.PUSH24:         {stack: op(0), gas: gasS(gasFastestStep)},
		evm.PUSH25:         {stack: op(0), gas: gasS(gasFastestStep)},
		evm.PUSH26:         {stack: op(0), gas: gasS(gasFastestStep)},
		evm.PUSH27:         {stack: op(0), gas: gasS(gasFastestStep)},
		evm.PUSH28:         {stack: op(0), gas: gasS(gasFastestStep)},
		evm.PUSH29:         {stack: op(0), gas: gasS(gasFastestStep)},
		evm.PUSH30:         {stack: op(0), gas: gasS(gasFastestStep)},
		evm.PUSH31:         {stack: op(0), gas: gasS(gasFastestStep)},
		evm.PUSH32:         {stack: op(0), gas: gasS(gasFastestStep)},
		evm.DUP1:           {stack: dup(1), gas: gasS(gasFastestStep)},
		evm.DUP2:           {stack: dup(2), gas: gasS(gasFastestStep)},
		evm.DUP3:           {stack: dup(3), gas: gasS(gasFastestStep)},
		evm.DUP4:           {stack: dup(4), gas: gasS(gasFastestStep)},
		evm.DUP5:           {stack: dup(5), gas: gasS(gasFastestStep)},
		evm.DUP6:           {stack: dup(6), gas: gasS(gasFastestStep)},
		evm.DUP7:           {stack: dup(7), gas: gasS(gasFastestStep)},
		evm.DUP8:           {stack: dup(8), gas: gasS(gasFastestStep)},
		evm.DUP9:           {stack: dup(9), gas: gasS(gasFastestStep)},
		evm.DUP10:          {stack: dup(10), gas: gasS(gasFastestStep)},
		evm.DUP11:          {stack: dup(11), gas: gasS(gasFastestStep)},
		evm.DUP12:          {stack: dup(12), gas: gasS(gasFastestStep)},
		evm.DUP13:          {stack: dup(13), gas: gasS(gasFastestStep)},
		evm.DUP14:          {stack: dup(14), gas: gasS(gasFastestStep)},
		evm.DUP15:          {stack: dup(15), gas: gasS(gasFastestStep)},
		evm.DUP16:          {stack: dup(16), gas: gasS(gasFastestStep)},
		evm.SWAP1:          {stack: swap(1), gas: gasS(gasFastestStep)},
		evm.SWAP2:          {stack: swap(2), gas: gasS(gasFastestStep)},
		evm.SWAP3:          {stack: swap(3), gas: gasS(gasFastestStep)},
		evm.SWAP4:          {stack: swap(4), gas: gasS(gasFastestStep)},
		evm.SWAP5:          {stack: swap(5), gas: gasS(gasFastestStep)},
		evm.SWAP6:          {stack: swap(6), gas: gasS(gasFastestStep)},
		evm.SWAP7:          {stack: swap(7), gas: gasS(gasFastestStep)},
		evm.SWAP8:          {stack: swap(8), gas: gasS(gasFastestStep)},
		evm.SWAP9:          {stack: swap(9), gas: gasS(gasFastestStep)},
		evm.SWAP10:         {stack: swap(10), gas: gasS(gasFastestStep)},
		evm.SWAP11:         {stack: swap(11), gas: gasS(gasFastestStep)},
		evm.SWAP12:         {stack: swap(12), gas: gasS(gasFastestStep)},
		evm.SWAP13:         {stack: swap(13), gas: gasS(gasFastestStep)},
		evm.SWAP14:         {stack: swap(14), gas: gasS(gasFastestStep)},
		evm.SWAP15:         {stack: swap(15), gas: gasS(gasFastestStep)},
		evm.SWAP16:         {stack: swap(16), gas: gasS(gasFastestStep)},
		evm.LOG0:           {stack: consume(2), gas: gasD(gasDynamicLog0)},
		evm.LOG1:           {stack: consume(3), gas: gasD(gasDynamicLog1)},
		evm.LOG2:           {stack: consume(4), gas: gasD(gasDynamicLog2)},
		evm.LOG3:           {stack: consume(5), gas: gasD(gasDynamicLog3)},
		evm.LOG4:           {stack: consume(6), gas: gasD(gasDynamicLog4)},
		evm.CREATE:         {stack: op(3), gas: gas(gasCreate, gasDynamicCreate)},
		evm.CALL:           {stack: op(7), gas: gas(gasCallEIP150, gasDynamicCall)},
		evm.CALLCODE:       {stack: op(7), gas: gas(gasCallEIP150, gasDynamicCallCodeCall)},
		evm.RETURN:         {stack: consume(2), gas: gasD(gasDynamicMemory)},
		evm.DELEGATECALL:   {stack: op(6), gas: gas(gasCallEIP150, gasDynamicStaticDelegateCall)},
		evm.CREATE2:        {stack: op(4), gas: gas(gasCreate, gasDynamicCreate2)},
		evm.STATICCALL:     {stack: op(6), gas: gas(gasCallEIP150, gasDynamicStaticDelegateCall)},
		evm.REVERT:         {stack: consume(2), gas: gasD(gasDynamicMemory)},
		evm.SELFDESTRUCT:   {stack: consume(1), gas: gasD(gasDynamicSelfDestruct)},
	}
	return res
}

func getBerlinInstructions() map[evm.OpCode]*InstructionInfo {
	// Berlin only modifies gas computations.
	// https://eips.ethereum.org/EIPS/eip-2929
	const gasWarmStorageReadCostEIP2929 vm.Gas = 100
	const gasSelfDestruct vm.Gas = 5000

	res := getIstanbulInstructions()

	// Static and dynamic gas calculation is changing for these instructions
	res[evm.SSTORE].gas = GasUsage{0, gasDynamicSStore}
	res[evm.SLOAD].gas = GasUsage{gasWarmStorageReadCostEIP2929, gasDynamicSLoad}
	res[evm.EXTCODECOPY].gas = GasUsage{gasWarmStorageReadCostEIP2929, gasDynamicExtCodeCopy}
	res[evm.EXTCODESIZE].gas = GasUsage{gasWarmStorageReadCostEIP2929, gasDynamicAccountAccess}
	res[evm.EXTCODEHASH].gas = GasUsage{gasWarmStorageReadCostEIP2929, gasDynamicAccountAccess}
	res[evm.BALANCE].gas = GasUsage{gasWarmStorageReadCostEIP2929, gasDynamicAccountAccess}
	res[evm.CALL].gas = GasUsage{gasWarmStorageReadCostEIP2929, gasDynamicCall}
	res[evm.CALLCODE].gas = GasUsage{gasWarmStorageReadCostEIP2929, gasDynamicCallCodeCall}
	res[evm.STATICCALL].gas = GasUsage{gasWarmStorageReadCostEIP2929, gasDynamicStaticDelegateCall}
	res[evm.DELEGATECALL].gas = GasUsage{gasWarmStorageReadCostEIP2929, gasDynamicStaticDelegateCall}
	// Selfdestruct dynamic gas calculation has changed in Berlin
	// Test is universal for all revisions, keeping here to know, there is change in calculation
	// res[evm.SELFDESTRUCT].gas = GasUsage{gasSelfDestruct, gasDynamicSelfDestruct}

	return res
}

func getLondonInstructions() map[evm.OpCode]*InstructionInfo {
	const gasQuickStep vm.Gas = 2
	res := getBerlinInstructions()
	// One additional instruction: BASEFEE
	// https://eips.ethereum.org/EIPS/eip-3198
	res[evm.BASEFEE] = &InstructionInfo{
		stack: StackUsage{pushed: 1},
		gas:   GasUsage{gasQuickStep, nil},
	}

	// Selfdestruct dynamic gas calculation has changed in London
	// Test is universal for all revisions, keeping here to know, there is change in calculation
	// res[evm.SELFDESTRUCT].gas.dynamic = gasDynamicSelfDestruct
	return res
}
