// Copyright (c) 2024 Fantom Foundation
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at fantom.foundation/bsl11.
//
// Change Date: 2028-4-16
//
// On the date above, in accordance with the Business Source License, use of
// this software will be governed by the GNU Lesser General Public License v3.

package interpreter_test

import (
	"fmt"

	"github.com/Fantom-foundation/Tosca/go/tosca"
	"github.com/Fantom-foundation/Tosca/go/tosca/vm"
)

// Revision references a vm specification version.
// TODO: remove this and replace it with tosca.Revision
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
// TODO: add newer revisions
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
	static  tosca.Gas
	dynamic func(revision Revision) []*DynGasTest
}

// getInstructions returns a map of OpCodes for the respective revision.
func getInstructions(revision Revision) map[vm.OpCode]*InstructionInfo {
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

func getIstanbulInstructions() map[vm.OpCode]*InstructionInfo {
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

	const gasJumpDest tosca.Gas = 1
	const gasQuickStep tosca.Gas = 2
	const gasFastestStep tosca.Gas = 3
	const gasFastStep tosca.Gas = 5
	const gasMidStep tosca.Gas = 8
	const gasSlowStep tosca.Gas = 10
	const gasBalance tosca.Gas = 700
	const gasExtStep tosca.Gas = 20
	const gasExtCode tosca.Gas = 700
	const gasSha3 tosca.Gas = 30
	const gasSloadEIP2200 tosca.Gas = 800
	const gasExtCodeHash tosca.Gas = 700
	const gasCallEIP150 tosca.Gas = 700
	const gasCreate tosca.Gas = 32000

	noGas := GasUsage{0, nil}

	gas := func(static tosca.Gas, dynamic func(revision Revision) []*DynGasTest) GasUsage {
		return GasUsage{static, dynamic}
	}

	gasD := func(dynamic func(revision Revision) []*DynGasTest) GasUsage {
		return GasUsage{0, dynamic}
	}

	gasS := func(static tosca.Gas) GasUsage {
		return GasUsage{static, nil}
	}

	res := map[vm.OpCode]*InstructionInfo{
		vm.STOP:           {stack: none, gas: noGas},
		vm.ADD:            {stack: op(2), gas: gasS(gasFastestStep)},
		vm.MUL:            {stack: op(2), gas: gasS(gasFastStep)},
		vm.SUB:            {stack: op(2), gas: gasS(gasFastestStep)},
		vm.DIV:            {stack: op(2), gas: gasS(gasFastStep)},
		vm.SDIV:           {stack: op(2), gas: gasS(gasFastStep)},
		vm.MOD:            {stack: op(2), gas: gasS(gasFastStep)},
		vm.SMOD:           {stack: op(2), gas: gasS(gasFastStep)},
		vm.ADDMOD:         {stack: op(3), gas: gasS(gasMidStep)},
		vm.MULMOD:         {stack: op(3), gas: gasS(gasMidStep)},
		vm.EXP:            {stack: op(2), gas: gasD(gasEXP)},
		vm.SIGNEXTEND:     {stack: op(2), gas: gasS(gasFastStep)},
		vm.LT:             {stack: op(2), gas: gasS(gasFastestStep)},
		vm.GT:             {stack: op(2), gas: gasS(gasFastestStep)},
		vm.SLT:            {stack: op(2), gas: gasS(gasFastestStep)},
		vm.SGT:            {stack: op(2), gas: gasS(gasFastestStep)},
		vm.EQ:             {stack: op(2), gas: gasS(gasFastestStep)},
		vm.ISZERO:         {stack: op(1), gas: gasS(gasFastestStep)},
		vm.AND:            {stack: op(2), gas: gasS(gasFastestStep)},
		vm.XOR:            {stack: op(2), gas: gasS(gasFastestStep)},
		vm.OR:             {stack: op(2), gas: gasS(gasFastestStep)},
		vm.NOT:            {stack: op(1), gas: gasS(gasFastestStep)},
		vm.BYTE:           {stack: op(2), gas: gasS(gasFastestStep)},
		vm.SHL:            {stack: op(2), gas: gasS(gasFastestStep)},
		vm.SHR:            {stack: op(2), gas: gasS(gasFastestStep)},
		vm.SAR:            {stack: op(2), gas: gasS(gasFastestStep)},
		vm.SHA3:           {stack: op(2), gas: gas(gasSha3, gasDynamicSHA3)},
		vm.ADDRESS:        {stack: op(0), gas: gasS(gasQuickStep)},
		vm.BALANCE:        {stack: op(1), gas: gasS(gasBalance)},
		vm.ORIGIN:         {stack: op(0), gas: gasS(gasQuickStep)},
		vm.CALLER:         {stack: op(0), gas: gasS(gasQuickStep)},
		vm.CALLVALUE:      {stack: op(0), gas: gasS(gasQuickStep)},
		vm.CALLDATALOAD:   {stack: op(1), gas: gasS(gasFastestStep)},
		vm.CALLDATASIZE:   {stack: op(0), gas: gasS(gasQuickStep)},
		vm.CALLDATACOPY:   {stack: consume(3), gas: gas(gasFastestStep, gasDynamicCopy)},
		vm.CODESIZE:       {stack: op(0), gas: gasS(gasQuickStep)},
		vm.CODECOPY:       {stack: consume(3), gas: gas(gasFastestStep, gasDynamicCopy)},
		vm.GASPRICE:       {stack: op(0), gas: gasS(gasQuickStep)},
		vm.EXTCODESIZE:    {stack: op(1), gas: gasS(gasExtCode)},
		vm.EXTCODECOPY:    {stack: consume(4), gas: gas(gasExtCode, gasDynamicExtCodeCopy)},
		vm.RETURNDATASIZE: {stack: op(0), gas: gasS(gasQuickStep)},
		vm.RETURNDATACOPY: {stack: consume(3), gas: gas(gasFastestStep, gasDynamicCopy)},
		vm.EXTCODEHASH:    {stack: op(1), gas: gasS(gasExtCodeHash)},
		vm.BLOCKHASH:      {stack: op(1), gas: gasS(gasExtStep)},
		vm.COINBASE:       {stack: op(0), gas: gasS(gasQuickStep)},
		vm.TIMESTAMP:      {stack: op(0), gas: gasS(gasQuickStep)},
		vm.NUMBER:         {stack: op(0), gas: gasS(gasQuickStep)},
		vm.PREVRANDAO:     {stack: op(0), gas: gasS(gasQuickStep)},
		vm.GASLIMIT:       {stack: op(0), gas: gasS(gasQuickStep)},
		vm.CHAINID:        {stack: op(0), gas: gasS(gasQuickStep)},
		vm.SELFBALANCE:    {stack: op(0), gas: gasS(gasFastStep)},
		vm.POP:            {stack: consume(1), gas: gasS(gasQuickStep)},
		vm.MLOAD:          {stack: op(1), gas: gas(gasFastestStep, gasDynamicMemory)},
		vm.MSTORE:         {stack: consume(2), gas: gas(gasFastestStep, gasDynamicMemory)},
		vm.MSTORE8:        {stack: consume(2), gas: gas(gasFastestStep, gasDynamicMemory)},
		vm.SLOAD:          {stack: op(1), gas: gasS(gasSloadEIP2200)},
		vm.SSTORE:         {stack: consume(2), gas: gas(0, gasDynamicSStore)},
		vm.JUMP:           {stack: consume(1), gas: gasS(gasMidStep)},
		vm.JUMPI:          {stack: consume(2), gas: gasS(gasSlowStep)},
		vm.PC:             {stack: op(0), gas: gasS(gasQuickStep)},
		vm.MSIZE:          {stack: op(0), gas: gasS(gasQuickStep)},
		vm.GAS:            {stack: op(0), gas: gasS(gasQuickStep)},
		vm.JUMPDEST:       {stack: none, gas: gasS(gasJumpDest)},
		vm.PUSH1:          {stack: op(0), gas: gasS(gasFastestStep)},
		vm.PUSH2:          {stack: op(0), gas: gasS(gasFastestStep)},
		vm.PUSH3:          {stack: op(0), gas: gasS(gasFastestStep)},
		vm.PUSH4:          {stack: op(0), gas: gasS(gasFastestStep)},
		vm.PUSH5:          {stack: op(0), gas: gasS(gasFastestStep)},
		vm.PUSH6:          {stack: op(0), gas: gasS(gasFastestStep)},
		vm.PUSH7:          {stack: op(0), gas: gasS(gasFastestStep)},
		vm.PUSH8:          {stack: op(0), gas: gasS(gasFastestStep)},
		vm.PUSH9:          {stack: op(0), gas: gasS(gasFastestStep)},
		vm.PUSH10:         {stack: op(0), gas: gasS(gasFastestStep)},
		vm.PUSH11:         {stack: op(0), gas: gasS(gasFastestStep)},
		vm.PUSH12:         {stack: op(0), gas: gasS(gasFastestStep)},
		vm.PUSH13:         {stack: op(0), gas: gasS(gasFastestStep)},
		vm.PUSH14:         {stack: op(0), gas: gasS(gasFastestStep)},
		vm.PUSH15:         {stack: op(0), gas: gasS(gasFastestStep)},
		vm.PUSH16:         {stack: op(0), gas: gasS(gasFastestStep)},
		vm.PUSH17:         {stack: op(0), gas: gasS(gasFastestStep)},
		vm.PUSH18:         {stack: op(0), gas: gasS(gasFastestStep)},
		vm.PUSH19:         {stack: op(0), gas: gasS(gasFastestStep)},
		vm.PUSH20:         {stack: op(0), gas: gasS(gasFastestStep)},
		vm.PUSH21:         {stack: op(0), gas: gasS(gasFastestStep)},
		vm.PUSH22:         {stack: op(0), gas: gasS(gasFastestStep)},
		vm.PUSH23:         {stack: op(0), gas: gasS(gasFastestStep)},
		vm.PUSH24:         {stack: op(0), gas: gasS(gasFastestStep)},
		vm.PUSH25:         {stack: op(0), gas: gasS(gasFastestStep)},
		vm.PUSH26:         {stack: op(0), gas: gasS(gasFastestStep)},
		vm.PUSH27:         {stack: op(0), gas: gasS(gasFastestStep)},
		vm.PUSH28:         {stack: op(0), gas: gasS(gasFastestStep)},
		vm.PUSH29:         {stack: op(0), gas: gasS(gasFastestStep)},
		vm.PUSH30:         {stack: op(0), gas: gasS(gasFastestStep)},
		vm.PUSH31:         {stack: op(0), gas: gasS(gasFastestStep)},
		vm.PUSH32:         {stack: op(0), gas: gasS(gasFastestStep)},
		vm.DUP1:           {stack: dup(1), gas: gasS(gasFastestStep)},
		vm.DUP2:           {stack: dup(2), gas: gasS(gasFastestStep)},
		vm.DUP3:           {stack: dup(3), gas: gasS(gasFastestStep)},
		vm.DUP4:           {stack: dup(4), gas: gasS(gasFastestStep)},
		vm.DUP5:           {stack: dup(5), gas: gasS(gasFastestStep)},
		vm.DUP6:           {stack: dup(6), gas: gasS(gasFastestStep)},
		vm.DUP7:           {stack: dup(7), gas: gasS(gasFastestStep)},
		vm.DUP8:           {stack: dup(8), gas: gasS(gasFastestStep)},
		vm.DUP9:           {stack: dup(9), gas: gasS(gasFastestStep)},
		vm.DUP10:          {stack: dup(10), gas: gasS(gasFastestStep)},
		vm.DUP11:          {stack: dup(11), gas: gasS(gasFastestStep)},
		vm.DUP12:          {stack: dup(12), gas: gasS(gasFastestStep)},
		vm.DUP13:          {stack: dup(13), gas: gasS(gasFastestStep)},
		vm.DUP14:          {stack: dup(14), gas: gasS(gasFastestStep)},
		vm.DUP15:          {stack: dup(15), gas: gasS(gasFastestStep)},
		vm.DUP16:          {stack: dup(16), gas: gasS(gasFastestStep)},
		vm.SWAP1:          {stack: swap(1), gas: gasS(gasFastestStep)},
		vm.SWAP2:          {stack: swap(2), gas: gasS(gasFastestStep)},
		vm.SWAP3:          {stack: swap(3), gas: gasS(gasFastestStep)},
		vm.SWAP4:          {stack: swap(4), gas: gasS(gasFastestStep)},
		vm.SWAP5:          {stack: swap(5), gas: gasS(gasFastestStep)},
		vm.SWAP6:          {stack: swap(6), gas: gasS(gasFastestStep)},
		vm.SWAP7:          {stack: swap(7), gas: gasS(gasFastestStep)},
		vm.SWAP8:          {stack: swap(8), gas: gasS(gasFastestStep)},
		vm.SWAP9:          {stack: swap(9), gas: gasS(gasFastestStep)},
		vm.SWAP10:         {stack: swap(10), gas: gasS(gasFastestStep)},
		vm.SWAP11:         {stack: swap(11), gas: gasS(gasFastestStep)},
		vm.SWAP12:         {stack: swap(12), gas: gasS(gasFastestStep)},
		vm.SWAP13:         {stack: swap(13), gas: gasS(gasFastestStep)},
		vm.SWAP14:         {stack: swap(14), gas: gasS(gasFastestStep)},
		vm.SWAP15:         {stack: swap(15), gas: gasS(gasFastestStep)},
		vm.SWAP16:         {stack: swap(16), gas: gasS(gasFastestStep)},
		vm.LOG0:           {stack: consume(2), gas: gasD(gasDynamicLog0)},
		vm.LOG1:           {stack: consume(3), gas: gasD(gasDynamicLog1)},
		vm.LOG2:           {stack: consume(4), gas: gasD(gasDynamicLog2)},
		vm.LOG3:           {stack: consume(5), gas: gasD(gasDynamicLog3)},
		vm.LOG4:           {stack: consume(6), gas: gasD(gasDynamicLog4)},
		vm.CREATE:         {stack: op(3), gas: gas(gasCreate, gasDynamicCreate)},
		vm.CALL:           {stack: op(7), gas: gas(gasCallEIP150, gasDynamicCall)},
		vm.CALLCODE:       {stack: op(7), gas: gas(gasCallEIP150, gasDynamicCallCodeCall)},
		vm.RETURN:         {stack: consume(2), gas: gasD(gasDynamicMemory)},
		vm.DELEGATECALL:   {stack: op(6), gas: gas(gasCallEIP150, gasDynamicStaticDelegateCall)},
		vm.CREATE2:        {stack: op(4), gas: gas(gasCreate, gasDynamicCreate2)},
		vm.STATICCALL:     {stack: op(6), gas: gas(gasCallEIP150, gasDynamicStaticDelegateCall)},
		vm.REVERT:         {stack: consume(2), gas: gasD(gasDynamicMemory)},
		vm.SELFDESTRUCT:   {stack: consume(1), gas: gasD(gasDynamicSelfDestruct)},
	}
	return res
}

func getBerlinInstructions() map[vm.OpCode]*InstructionInfo {
	// Berlin only modifies gas computations.
	// https://eips.ethereum.org/EIPS/eip-2929
	const gasWarmStorageReadCostEIP2929 tosca.Gas = 100

	res := getIstanbulInstructions()

	// Static and dynamic gas calculation is changing for these instructions
	res[vm.SSTORE].gas = GasUsage{0, gasDynamicSStore}
	res[vm.SLOAD].gas = GasUsage{gasWarmStorageReadCostEIP2929, gasDynamicSLoad}
	res[vm.EXTCODECOPY].gas = GasUsage{gasWarmStorageReadCostEIP2929, gasDynamicExtCodeCopy}
	res[vm.EXTCODESIZE].gas = GasUsage{gasWarmStorageReadCostEIP2929, gasDynamicAccountAccess}
	res[vm.EXTCODEHASH].gas = GasUsage{gasWarmStorageReadCostEIP2929, gasDynamicAccountAccess}
	res[vm.BALANCE].gas = GasUsage{gasWarmStorageReadCostEIP2929, gasDynamicAccountAccess}
	res[vm.CALL].gas = GasUsage{gasWarmStorageReadCostEIP2929, gasDynamicCall}
	res[vm.CALLCODE].gas = GasUsage{gasWarmStorageReadCostEIP2929, gasDynamicCallCodeCall}
	res[vm.STATICCALL].gas = GasUsage{gasWarmStorageReadCostEIP2929, gasDynamicStaticDelegateCall}
	res[vm.DELEGATECALL].gas = GasUsage{gasWarmStorageReadCostEIP2929, gasDynamicStaticDelegateCall}
	// Selfdestruct dynamic gas calculation has changed in Berlin
	// Test is universal for all revisions, keeping here to know, there is change in calculation
	// const gasSelfDestruct tosca.Gas = 5000
	// res[vm.SELFDESTRUCT].gas = GasUsage{gasSelfDestruct, gasDynamicSelfDestruct}

	return res
}

func getLondonInstructions() map[vm.OpCode]*InstructionInfo {
	const gasQuickStep tosca.Gas = 2
	res := getBerlinInstructions()
	// One additional instruction: BASEFEE
	// https://eips.ethereum.org/EIPS/eip-3198
	res[vm.BASEFEE] = &InstructionInfo{
		stack: StackUsage{pushed: 1},
		gas:   GasUsage{gasQuickStep, nil},
	}

	// Selfdestruct dynamic gas calculation has changed in London
	// Test is universal for all revisions, keeping here to know, there is change in calculation
	// res[vm.SELFDESTRUCT].gas.dynamic = gasDynamicSelfDestruct
	return res
}
