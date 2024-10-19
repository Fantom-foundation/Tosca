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
	"bytes"
	"encoding/binary"
	"fmt"
	"math"
	"math/big"
	"math/rand"
	"testing"

	"github.com/Fantom-foundation/Tosca/go/ct/common"
	"github.com/Fantom-foundation/Tosca/go/tosca"
	"github.com/Fantom-foundation/Tosca/go/tosca/vm"
	"github.com/holiman/uint256"
	"go.uber.org/mock/gomock"

	// This is only imported to get the EVM error definitions.
	// TODO: write up our own error definition and remove this dependency.
	geth "github.com/ethereum/go-ethereum/core/vm"
)

const HUGE_GAS_SENT_WITH_CALL int64 = 1000000000000

func TestMaxCallDepth(t *testing.T) {
	// For every variant of interpreter
	for _, variant := range getAllInterpreterVariantsForTests() {
		for _, revision := range revisions {
			t.Run(fmt.Sprintf("%s/%s", variant, revision), func(t *testing.T) {
				ctrl := gomock.NewController(t)
				mockStateDB := newMockStateDBForIntegrationTests(ctrl)

				zeroVal := big.NewInt(0)
				account := tosca.Address{}

				// return and input data size is 32bytes, memory offset is 0 for all
				callStackValues := []*big.Int{big.NewInt(32), zeroVal, big.NewInt(32), zeroVal, zeroVal, addressToBigInt(account), big.NewInt(HUGE_GAS_SENT_WITH_CALL)}
				pushCode, _ := addValuesToStack(callStackValues, 0)

				// put 32byte input value with 0 offset from memory to stack, add 1 to it and put it back to memory with 0 offset
				code := []byte{
					byte(vm.PUSH1), byte(0),
					byte(vm.CALLDATALOAD),
					byte(vm.PUSH1), byte(1),
					byte(vm.ADD),
					byte(vm.PUSH1), byte(0),
					byte(vm.MSTORE)}

				// add stack values for call instruction
				code = append(code, pushCode...)

				// make inner call and return 32byte value with 0 offset from memory
				codeReturn := []byte{
					byte(vm.CALL),
					byte(vm.PUSH1), byte(32),
					byte(vm.PUSH1), byte(0),
					byte(vm.RETURN)}
				code = append(code, codeReturn...)

				setDefaultCallStateDBMock(mockStateDB, account, code)

				evm := GetCleanEVM(revision, variant, mockStateDB)

				// Run an interpreter
				result, err := evm.Run(code, []byte{})

				// Check the result.
				if err != nil || !result.Success {
					t.Errorf("execution failed and should not fail, error is: %v, result %v", err, result)
				} else {
					expectedDepth := 1025
					depth := big.NewInt(0).SetBytes(result.Output).Uint64()
					if depth != uint64(expectedDepth) {
						t.Errorf("expected call depth is %v, got %v", expectedDepth, depth)
					}
				}
			})
		}
	}
}

func TestInvalidJumpOverflow(t *testing.T) {

	// For every variant of interpreter
	for _, variant := range getAllInterpreterVariantsForTests() {

		for _, revision := range revisions {

			testInstructions := []vm.OpCode{vm.JUMP, vm.JUMPI}

			for _, instruction := range testInstructions {

				t.Run(fmt.Sprintf("%s/%s/%s", variant, revision, instruction.String()), func(t *testing.T) {

					evm := GetCleanEVM(revision, variant, nil)

					// condition for JUMPI instruction
					condition := big.NewInt(1)
					// destination number bigger then uint64 with uint64 part containing relevant jump destination
					dst, _ := big.NewInt(0).SetString("0x1000000000000000d", 0)
					code, _ := addValuesToStack([]*big.Int{condition, dst}, 0)
					codeJump := []byte{
						byte(instruction),
						byte(vm.JUMPDEST),
						byte(vm.PUSH1), byte(0),
						byte(vm.STOP)}
					code = append(code, codeJump...)

					// Run an interpreter
					result, err := evm.Run(code, []byte{})

					// Check the result.
					if err != nil {
						t.Fatalf("unexpected internal interpreter error: %v", err)
					}
					if result.Success {
						t.Errorf("execution should fail, but got: %v", result)
					}
				})
			}
		}
	}
}

func TestCodeCopy(t *testing.T) {
	type test struct {
		offset *big.Int
		size   *big.Int
	}

	const codeSize = 40
	zero := big.NewInt(0)
	overflowValue, _ := big.NewInt(0).SetString("0x1000000000000000d", 0)
	if overflowValue.IsUint64() {
		t.Fatalf("overflow value does not overflow!")
	}

	tests := map[string]test{
		"read_zero_bytes_from_start": {
			offset: big.NewInt(0),
			size:   big.NewInt(0),
		},
		"read_one_byte_from_start": {
			offset: big.NewInt(0),
			size:   big.NewInt(1),
		},
		"read_two_bytes_from_start": {
			offset: big.NewInt(0),
			size:   big.NewInt(2),
		},
		"read_two_bytes_from_end": {
			offset: big.NewInt(codeSize - 2),
			size:   big.NewInt(2),
		},
		"read_two_bytes_from_near_end": {
			offset: big.NewInt(codeSize - 3),
			size:   big.NewInt(2),
		},
		"read_ten_bytes_crossing_end": {
			offset: big.NewInt(codeSize - 5),
			size:   big.NewInt(10),
		},
		"read_ten_bytes_beyond_the_end": {
			offset: big.NewInt(codeSize + 5),
			size:   big.NewInt(10),
		},
		"read_ten_bytes_beyond_the_64_bit_range": {
			offset: overflowValue,
			size:   big.NewInt(10),
		},
	}

	// For every variant of interpreter
	for _, variant := range getAllInterpreterVariantsForTests() {
		for _, revision := range revisions {
			for name, test := range tests {
				t.Run(fmt.Sprintf("%s/%s/%s", variant, revision, name), func(t *testing.T) {
					mockCtrl := gomock.NewController(t)
					mockStateDB := NewMockStateDB(mockCtrl)

					// Create a program that
					// - runs CODECOPY for the range specified by the test case spec
					// - returns the data extracted from CODECOPY as the result of the program

					// create code preparing the parameters for the code copy call
					codeCopyParameters := []*big.Int{
						test.size,
						test.offset,
						zero,
					}
					code, _ := addValuesToStack(codeCopyParameters, 0)
					code = append(code, byte(vm.CODECOPY))

					returnParameter := []*big.Int{
						test.size,
						zero,
					}
					returnParameterSetupCode, _ := addValuesToStack(returnParameter, 0)
					code = append(code, returnParameterSetupCode...)
					code = append(code, byte(vm.RETURN))

					if got, limit := len(code), codeSize; got > limit {
						t.Fatalf("unexpected code size, limit %d, got %d", limit, got)
					}

					// pad the code with extra data to match the code size
					for len(code) < codeSize {
						code = append(code, byte(len(code)))
					}

					// Run the code through an interpreter.
					evm := GetCleanEVM(revision, variant, mockStateDB)
					res, err := evm.Run(code, []byte{})

					if err != nil {
						t.Fatalf("unexpected execution error: %v", err)
					}

					// Check the output.
					want := make([]byte, test.size.Int64())
					if test.offset.Cmp(big.NewInt(int64(len(code)))) < 0 {
						start := int(test.offset.Uint64())
						end := start + int(test.size.Uint64())
						if end > len(code) {
							end = len(code)
						}
						copy(want, code[start:end])
					}
					if got := res.Output; !bytes.Equal(got, want) {
						t.Errorf("unexpected result, wanted %x, got %x", want, got)
					}
				})
			}
		}
	}
}

func TestReturnDataCopy(t *testing.T) {
	type test struct {
		name           string
		dataSize       uint
		dataOffset     *big.Int
		returnDataSize uint
		expectFailure  bool
	}

	zero := big.NewInt(0)
	one := big.NewInt(1)
	overflowValue, _ := big.NewInt(0).SetString("0x1000000000000000d", 0)

	tests := []test{
		{name: "no data", dataSize: 0, dataOffset: zero, returnDataSize: 0, expectFailure: false},
		{name: "offset > return data", dataSize: 0, dataOffset: one, returnDataSize: 0, expectFailure: true},
		{name: "same data", dataSize: 0, dataOffset: one, returnDataSize: 1, expectFailure: false},
		{name: "size > return data", dataSize: 1, dataOffset: zero, returnDataSize: 0, expectFailure: true},
		{name: "size + offset > return data", dataSize: 1, dataOffset: one, returnDataSize: 0, expectFailure: true},
		{name: "same data", dataSize: 1, dataOffset: zero, returnDataSize: 2, expectFailure: false},
		{name: "offset overflow", dataSize: 0, dataOffset: overflowValue, returnDataSize: 0, expectFailure: true},
	}
	// For every variant of interpreter
	for _, variant := range getAllInterpreterVariantsForTests() {

		for _, revision := range revisions {

			for _, tst := range tests {

				t.Run(fmt.Sprintf("%s/%s/%s", variant, revision, tst.name), func(t *testing.T) {

					mockCtrl := gomock.NewController(t)
					mockStateDB := newMockStateDBForIntegrationTests(mockCtrl)

					zeroVal := big.NewInt(0)
					account := tosca.Address{}
					returnDataSize := big.NewInt(10)

					// stack values for inner contract call
					callStackValues := []*big.Int{returnDataSize, zeroVal, zeroVal, zeroVal, zeroVal, addressToBigInt(account), big.NewInt(HUGE_GAS_SENT_WITH_CALL)}
					code, _ := addValuesToStack(callStackValues, 0)
					code = append(code, byte(vm.CALL))

					callStackValues = []*big.Int{big.NewInt(int64(tst.dataSize)), tst.dataOffset, zeroVal}
					codeValues, _ := addValuesToStack(callStackValues, 0)
					code = append(code, codeValues...)
					code = append(code,
						byte(vm.RETURNDATACOPY),
						byte(vm.STOP))

					// code for inner contract call
					codeReturn := []byte{
						byte(vm.PUSH1), byte(tst.returnDataSize),
						byte(vm.PUSH1), byte(0),
						byte(vm.RETURN)}

					// set mock for inner call
					setDefaultCallStateDBMock(mockStateDB, account, codeReturn)

					evm := GetCleanEVM(revision, variant, mockStateDB)

					// Run an interpreter
					result, err := evm.Run(code, []byte{})

					// Check the result.
					if err != nil {
						t.Fatalf("unexpected internal interpreter error: %v", err)
					}

					if want, got := !tst.expectFailure, result.Success; want != got {
						t.Errorf("unexpected result, wanted success to be %t got %t", want, got)
					}
				})
			}
		}
	}
}

func TestReadOnlyStaticCall(t *testing.T) {
	type callsType struct {
		instruction vm.OpCode
		shouldFail  bool
	}

	// all types of inner call to be tested
	calls := []callsType{
		{vm.CALL, false},
		{vm.STATICCALL, true},
		{vm.DELEGATECALL, false},
		{vm.CALLCODE, false},
		{vm.CREATE, false},
		{vm.CREATE2, false},
	}

	readOnlyInstructions := []vm.OpCode{
		vm.LOG0, vm.LOG1, vm.LOG2, vm.LOG3, vm.LOG4,
		vm.SSTORE, vm.CREATE, vm.CREATE2, vm.SELFDESTRUCT,
	}

	// For every variant of interpreter
	for _, variant := range getAllInterpreterVariantsForTests() {

		for _, revision := range revisions {

			for _, call := range calls {

				for _, instruction := range readOnlyInstructions {

					t.Run(fmt.Sprintf("%s/%s/%s/%s", variant, revision, call.instruction, instruction), func(t *testing.T) {

						mockCtrl := gomock.NewController(t)
						mockStateDB := newMockStateDBForIntegrationTests(mockCtrl)

						zeroVal := big.NewInt(0)
						account := tosca.Address{byte(0)}

						// stack values for inner contract call
						callStackValues := []*big.Int{zeroVal, zeroVal, zeroVal, zeroVal, zeroVal, addressToBigInt(account), big.NewInt(HUGE_GAS_SENT_WITH_CALL)}
						code, _ := addValuesToStack(callStackValues, 0)
						code = append(code,
							byte(call.instruction),
							byte(vm.PUSH1), byte(0),
							byte(vm.MSTORE),
							byte(vm.PUSH1), byte(32),
							byte(vm.PUSH1), byte(0),
							byte(vm.RETURN))

						// code for inner contract call
						innerCallCode := []byte{
							// push zero values to stack to have data for write instructions
							byte(vm.PUSH1), byte(0),
							byte(vm.DUP1), byte(vm.DUP1), byte(vm.DUP1), byte(vm.DUP1),
							byte(vm.DUP1), byte(vm.DUP1), byte(vm.DUP1), byte(vm.DUP1),
							byte(instruction),
							byte(vm.PUSH1), byte(0),
							byte(vm.DUP1),
							byte(vm.RETURN)}

						// set mock for inner call
						setDefaultCallStateDBMock(mockStateDB, account, innerCallCode)

						evm := GetCleanEVM(revision, variant, mockStateDB)

						// Run an interpreter
						result, err := evm.Run(code, []byte{})

						// Check for internal errors.
						if err != nil {
							t.Fatalf("unexpected internal error: %v", err)
						}

						res := big.NewInt(0).SetBytes(result.Output[0:32])
						success := res.Cmp(zeroVal) != 0
						want := !call.shouldFail
						if success != want {
							t.Errorf("unexpected result of execution, wanted success=%t, got %t", want, success)
						}
					})
				}
			}
		}
	}
}

func TestInstructionDataInitialization(t *testing.T) {

	type test struct {
		name        string
		instruction vm.OpCode
		size        *big.Int
		offset      *big.Int
		err         []error
	}

	type instructionTest struct {
		instruction vm.OpCode
		okError     []error
	}

	instructions := []instructionTest{
		{vm.RETURN, nil},
		{vm.REVERT, []error{geth.ErrExecutionReverted}},
		{vm.SHA3, nil},
		{vm.LOG0, nil},
		{vm.CODECOPY, nil},
		{vm.EXTCODECOPY, nil},
	}

	sizeNormal, _ := big.NewInt(0).SetString("0x10000", 0)
	// Adding two huge together overflows uint64
	sizeHuge, _ := big.NewInt(0).SetString("0x8000000000000000", 0)
	sizeOverUint64, _ := big.NewInt(0).SetString("0x100000000000000000", 0)

	tests := make([]test, 0)
	for _, instructionCase := range instructions {
		testForInstruction := []test{
			{"zero size and offset", instructionCase.instruction, big.NewInt(0), big.NewInt(0), instructionCase.okError},
			{"normal size and offset", instructionCase.instruction, sizeNormal, sizeNormal, instructionCase.okError},
			{"huge size normal offset", instructionCase.instruction, sizeHuge, sizeNormal, []error{geth.ErrOutOfGas}},
			{"over size normal offset", instructionCase.instruction, sizeOverUint64, sizeNormal, []error{geth.ErrGasUintOverflow, geth.ErrOutOfGas}},
			{"normal size huge offset", instructionCase.instruction, sizeNormal, sizeHuge, []error{geth.ErrOutOfGas}},
			{"normal size over offset", instructionCase.instruction, sizeNormal, sizeOverUint64, []error{geth.ErrGasUintOverflow, geth.ErrOutOfGas}},
			{"huge size huge offset", instructionCase.instruction, sizeHuge, sizeHuge, []error{geth.ErrGasUintOverflow, geth.ErrOutOfGas}},
			{"zero size over offset", instructionCase.instruction, big.NewInt(0), sizeOverUint64, instructionCase.okError},
		}
		tests = append(tests, testForInstruction...)
	}

	// For every variant of interpreter
	for _, variant := range getAllInterpreterVariantsForTests() {
		for _, revision := range revisions {
			for _, test := range tests {
				t.Run(fmt.Sprintf("%s/%s/%s/%s", variant, revision, test.instruction, test.name), func(t *testing.T) {

					mockCtrl := gomock.NewController(t)
					mockStateDB := NewMockStateDB(mockCtrl)
					// set mock for inner call
					setDefaultCallStateDBMock(mockStateDB, tosca.Address{byte(0)}, make([]byte, 0))

					callStackValues := []*big.Int{test.size, test.offset}
					if test.instruction == vm.CODECOPY || test.instruction == vm.EXTCODECOPY {
						callStackValues = append(callStackValues, test.offset)
					}
					if test.instruction == vm.EXTCODECOPY {
						callStackValues = append(callStackValues, big.NewInt(0))
					}
					code, _ := addValuesToStack(callStackValues, 0)
					code = append(code, byte(test.instruction))

					evm := GetCleanEVM(revision, variant, mockStateDB)

					// Run an interpreter
					result, err := evm.Run(code, []byte{})

					// Check the result.
					if err != nil {
						t.Fatalf("unexpected internal failure in EVM: %v", err)
					}
					want := len(test.err) == 0
					got := result.Success
					if want != got {
						t.Errorf("unexpected result, wanted success=%t, got %t", want, got)
					}
				})
			}
		}
	}
}

func TestCreateDataInitialization(t *testing.T) {

	account := tosca.Address{byte(0)}
	type test struct {
		name   string
		size   *big.Int
		offset *big.Int
		err    []error
	}

	// Adding two huge together overflows uint64
	numHuge, _ := big.NewInt(0).SetString("0x8000000000000000", 0)
	numOverUint64, _ := big.NewInt(0).SetString("0x100000000000000000", 0)

	instructions := []vm.OpCode{vm.CREATE, vm.CREATE2}
	tests := []test{
		{"zero size over offset", big.NewInt(0), numOverUint64, nil},
		{"over size zero offset", numOverUint64, big.NewInt(0), []error{geth.ErrGasUintOverflow, geth.ErrOutOfGas}},
		{"over size over offset", numOverUint64, numOverUint64, []error{geth.ErrGasUintOverflow, geth.ErrOutOfGas}},
		{"add size and offset overflows", numHuge, numHuge, []error{geth.ErrGasUintOverflow, geth.ErrOutOfGas}},
	}

	// For every variant of interpreter
	for _, variant := range getAllInterpreterVariantsForTests() {

		for _, revision := range revisions {

			for _, instruction := range instructions {

				for _, test := range tests {

					t.Run(fmt.Sprintf("%s/%s/%s/%s", variant, revision, instruction, test.name), func(t *testing.T) {

						mockCtrl := gomock.NewController(t)
						mockStateDB := NewMockStateDB(mockCtrl)

						callStackValues := []*big.Int{test.size, test.offset, big.NewInt(0)}
						if instruction == vm.CREATE2 {
							callStackValues = append([]*big.Int{big.NewInt(0)}, callStackValues...)
						}

						code, _ := addValuesToStack(callStackValues, 0)
						code = append(code, byte(instruction))

						// set mock for inner call
						setDefaultCallStateDBMock(mockStateDB, account, make([]byte, 0))

						evm := GetCleanEVM(revision, variant, mockStateDB)
						// Run an interpreter
						result, err := evm.Run(code, []byte{})

						// Check the result.
						if err != nil {
							t.Fatalf("unexpected internal failure in EVM: %v", err)
						}
						want := len(test.err) == 0
						got := result.Success
						if want != got {
							t.Errorf("expected success to be %t, got %t", want, got)
						}
					})
				}
			}
		}
	}
}

func TestMemoryNotWrittenWithZeroReturnData(t *testing.T) {
	zeroVal := big.NewInt(0)
	size32 := big.NewInt(32)
	size64 := big.NewInt(64)

	type callsType struct {
		instruction       vm.OpCode
		callOutputMemSize *big.Int
		afterCallMemSize  *big.Int
		memShouldChange   bool
	}

	// all types of inner call to be tested
	calls := []callsType{
		{vm.CALL, zeroVal, size32, false},
		{vm.STATICCALL, zeroVal, size32, false},
		{vm.DELEGATECALL, zeroVal, size32, false},
		{vm.CALLCODE, zeroVal, size32, false},
		{vm.CALL, size64, size64, true},
		{vm.STATICCALL, size64, size64, true},
		{vm.DELEGATECALL, size64, size64, true},
		{vm.CALLCODE, size64, size64, true},
	}

	// For every variant of interpreter
	for _, variant := range getAllInterpreterVariantsForTests() {

		for _, revision := range revisions {

			for _, call := range calls {

				t.Run(fmt.Sprintf("%s/%s/%s/%s", variant, revision, call.instruction, call.callOutputMemSize), func(t *testing.T) {

					mockCtrl := gomock.NewController(t)
					mockStateDB := newMockStateDBForIntegrationTests(mockCtrl)

					account := tosca.Address{}

					// Store data into memory to test, if it would be overwritten
					wantMemWord := getRandomBigIntArray(1)
					wantMemWord = append(wantMemWord, zeroVal)
					code, _ := addValuesToStack(wantMemWord, 0)
					code = append(code,
						byte(vm.MSTORE))

					// stack values for inner contract call
					var callStackValues []*big.Int
					if call.instruction == vm.CALL || call.instruction == vm.CALLCODE {
						callStackValues = []*big.Int{call.callOutputMemSize, zeroVal, zeroVal, zeroVal, zeroVal, addressToBigInt(account), big.NewInt(HUGE_GAS_SENT_WITH_CALL)}
					} else {
						callStackValues = []*big.Int{call.callOutputMemSize, zeroVal, zeroVal, zeroVal, addressToBigInt(account), big.NewInt(HUGE_GAS_SENT_WITH_CALL)}
					}
					pushCode, _ := addValuesToStack(callStackValues, 0)
					code = append(code, pushCode...)
					code = append(code,
						byte(call.instruction),
						byte(vm.MSIZE),
						byte(vm.PUSH1), byte(32),
						byte(vm.MSTORE),
						byte(vm.PUSH1), byte(64),
						byte(vm.PUSH1), byte(0),
						byte(vm.RETURN))

					// code for inner call
					// return 32 bytes of memory
					innerCallCode := []byte{
						byte(vm.PUSH1), byte(32),
						byte(vm.DUP1),
						byte(vm.RETURN)}

					// set mock for inner call
					setDefaultCallStateDBMock(mockStateDB, account, innerCallCode)

					evm := GetCleanEVM(revision, variant, mockStateDB)

					// Run an interpreter
					result, err := evm.Run(code, []byte{})

					gotMemWord := big.NewInt(0).SetBytes(result.Output[0:32])
					gotMemSize := big.NewInt(0).SetBytes(result.Output[0:64])

					// compare results
					memWordIsSame := gotMemWord.Cmp(wantMemWord[0]) == 0
					if memWordIsSame == call.memShouldChange {
						if call.memShouldChange {
							t.Errorf("memory should change when return data size > 0, but it didn't")
						} else {
							t.Errorf("memory should not change when return data size is 0, but it did")
						}
					}

					if gotMemSize.Uint64() != call.afterCallMemSize.Uint64() {
						t.Errorf("memory size after call is not as expected, want %v, got %v", call.afterCallMemSize.Uint64(), gotMemSize.Uint64())
					}

					if err != nil {
						t.Errorf("execution should not return an error, but got:%v", err)
					}
				})
			}
		}
	}
}

func TestNoReturnDataForCreate(t *testing.T) {
	account := tosca.Address{byte(0)}
	type test struct {
		name              string
		createInstruction vm.OpCode
		returnInstruction vm.OpCode
		returnDataSize    uint64
	}

	tests := []test{
		{"no return data", vm.CREATE, vm.RETURN, 0},
		{"no return data", vm.CREATE2, vm.RETURN, 0},
		{"has revert data", vm.CREATE, vm.REVERT, 32},
		{"has revert data", vm.CREATE2, vm.REVERT, 32},
	}

	// For every variant of interpreter
	for _, variant := range getAllInterpreterVariantsForTests() {

		if skipTestForVariant(t.Name(), variant) {
			continue
		}

		for _, revision := range revisions {

			for _, test := range tests {

				t.Run(fmt.Sprintf("%s/%s/%s/%s", variant, revision, test.name, test.createInstruction), func(t *testing.T) {

					mockCtrl := gomock.NewController(t)
					mockStateDB := newMockStateDBForIntegrationTests(mockCtrl)

					createStackValues := []*big.Int{big.NewInt(32), big.NewInt(0), big.NewInt(0)}
					if test.createInstruction == vm.CREATE2 {
						createStackValues = append([]*big.Int{big.NewInt(0)}, createStackValues...)
					}
					createInputCode := []byte{
						byte(vm.PUSH1), byte(32),
						byte(vm.PUSH1), byte(0),
						byte(test.returnInstruction)}

					createInputBytes := common.RightPadSlice(createInputCode, 32)
					createInputValue := []*big.Int{big.NewInt(0).SetBytes(createInputBytes)}

					code, _ := addValuesToStack(createInputValue, 0)
					code = append(code,
						byte(vm.PUSH1), byte(0),
						byte(vm.MSTORE))

					pushCode, _ := addValuesToStack(createStackValues, 0)
					code = append(code, pushCode...)
					code = append(code, byte(test.createInstruction))
					code = append(code,
						byte(vm.RETURNDATASIZE),
						byte(vm.PUSH1), byte(0),
						byte(vm.MSTORE),
						byte(vm.PUSH1), byte(32),
						byte(vm.PUSH1), byte(0),
						byte(vm.RETURN))

					contractAddr := TestEvmCreatedAccountAddress

					// set mock for inner call
					setDefaultCallStateDBMock(mockStateDB, account, make([]byte, 0))
					mockStateDB.EXPECT().GetCodeHash(contractAddr).AnyTimes().Return(tosca.Hash{})

					evm := GetCleanEVM(revision, variant, mockStateDB)

					// Run an interpreter
					result, err := evm.Run(code, []byte{})
					if err != nil {
						t.Errorf("unexpected internal interpreter failure: %v", err)
					}

					returnDataSize := big.NewInt(0).SetBytes(result.Output[0:32])

					// Check the result.
					if returnDataSize.Uint64() != test.returnDataSize {
						t.Errorf("expected return data size: %v, but got: %v", test.returnDataSize, returnDataSize)
					}
				})
			}
		}
	}
}

func TestExtCodeHashOnEmptyAccount(t *testing.T) {
	codeHash := tosca.Hash{byte(2)}
	tests := map[string]struct {
		empty  bool
		result tosca.Hash
	}{
		"empty_account":     {true, tosca.Hash{}},
		"non_empty_account": {false, codeHash},
	}

	// For every variant of interpreter
	for _, variant := range getAllInterpreterVariantsForTests() {

		if skipTestForVariant(t.Name(), variant) {
			continue
		}

		for _, revision := range revisions {
			for name, test := range tests {

				t.Run(fmt.Sprintf("%s/%s/%s", variant, revision, name), func(t *testing.T) {

					mockCtrl := gomock.NewController(t)
					mockStateDB := NewMockStateDB(mockCtrl)

					account := tosca.Address{byte(1)}

					// stack values for inner contract call
					code, _ := addValuesToStack([]*big.Int{addressToBigInt(account)}, 0)

					// code for inner contract call
					code = append(code, []byte{
						byte(vm.EXTCODEHASH),
						byte(vm.PUSH1), byte(0),
						byte(vm.MSTORE),
						byte(vm.PUSH1), byte(32),
						byte(vm.PUSH1), byte(0),
						byte(vm.RETURN)}...)

					// set mock for inner call
					if test.empty {
						mockStateDB.EXPECT().GetBalance(account).AnyTimes().Return(tosca.Value{0})
					} else {
						mockStateDB.EXPECT().GetBalance(account).AnyTimes().Return(tosca.Value{1})
					}
					mockStateDB.EXPECT().GetNonce(account).AnyTimes().Return(uint64(0))
					mockStateDB.EXPECT().GetCodeSize(account).AnyTimes().Return(0)
					mockStateDB.EXPECT().IsAddressInAccessList(account).AnyTimes().Return(false)
					mockStateDB.EXPECT().AccessAccount(account).AnyTimes().Return(tosca.ColdAccess)

					mockStateDB.EXPECT().GetCodeHash(account).AnyTimes().Return(codeHash)

					evm := GetCleanEVM(revision, variant, mockStateDB)

					// Run an interpreter
					result, err := evm.Run(code, []byte{})
					if err != nil {
						t.Fatalf("unexpected internal failure in EVM: %v", err)
					}

					// Check the result.
					if want, got := test.result[:], result.Output; !bytes.Equal(want, got) {
						t.Errorf("expected hash %v, got %v", want, got)
					}
				})
			}
		}
	}
}

// Returns *big.Int with bits set as signed integer
func getNegativeBigIntSignInBits(value *big.Int) *big.Int {
	neg, _ := uint256.FromBig(value)
	return neg.ToBig()
}

func TestSARInstruction(t *testing.T) {

	sizeOverUint64, _ := big.NewInt(0).SetString("0x100000000000000001", 0)
	sizeOverUint64ByOne, _ := big.NewInt(0).SetString("0x80000000000000000", 0)
	sizeMaxIntPositive, _ := big.NewInt(0).SetString("0x7FFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF", 0)

	sizeMaxIntNegative, _ := big.NewInt(0).SetString("0x8000000000000000000000000000000000000000000000000000000000000001", 0)
	sizeMaxIntNegativeByOne, _ := big.NewInt(0).SetString("0xc000000000000000000000000000000000000000000000000000000000000000", 0)

	negativeInt := getNegativeBigIntSignInBits(big.NewInt(-32))
	negativeOverUint64 := getNegativeBigIntSignInBits(big.NewInt(0).Neg(sizeOverUint64))
	negOverUint64ByOne, _ := big.NewInt(0).SetString("0x80000000000000001", 0)
	negativeOverUint64ByOne := getNegativeBigIntSignInBits(big.NewInt(0).Neg(negOverUint64ByOne))

	// It is -1 if all bits are set to 1
	mostNegativeShiftRight := uint256.NewInt(0).SetAllOne().ToBig()

	tests := []overflowTestCase{
		{"all zero", []*big.Int{big.NewInt(0), big.NewInt(0)}, zeroInput, big.NewInt(0), nil},
		{"0>>1", []*big.Int{big.NewInt(0), big.NewInt(1)}, zeroInput, big.NewInt(0), nil},
		{"over64>>1", []*big.Int{sizeOverUint64, big.NewInt(1)}, zeroInput, sizeOverUint64ByOne, nil},
		{"over64>>over64", []*big.Int{sizeOverUint64, sizeOverUint64}, zeroInput, big.NewInt(0), nil},
		{"maxPositiveInt256>>254", []*big.Int{sizeMaxIntPositive, big.NewInt(254)}, zeroInput, big.NewInt(1), nil},

		{"negative>>0", []*big.Int{negativeInt, big.NewInt(0)}, zeroInput, negativeInt, nil},
		{"negative>>2", []*big.Int{negativeInt, big.NewInt(2)}, zeroInput, getNegativeBigIntSignInBits(big.NewInt(-8)), nil},
		{"negativeOver64>>1", []*big.Int{negativeOverUint64, big.NewInt(1)}, zeroInput, negativeOverUint64ByOne, nil},
		{"negativeOver64>>257", []*big.Int{negativeOverUint64, big.NewInt(257)}, zeroInput, mostNegativeShiftRight, nil},
		{"negativeOver64>>over64", []*big.Int{negativeOverUint64, sizeOverUint64}, zeroInput, mostNegativeShiftRight, nil},
		{"maxNegativeInt256>>1", []*big.Int{sizeMaxIntNegative, big.NewInt(1)}, zeroInput, sizeMaxIntNegativeByOne, nil},
		{"maxNegativeInt256>>254", []*big.Int{sizeMaxIntNegative, big.NewInt(254)}, zeroInput, getNegativeBigIntSignInBits(big.NewInt(-2)), nil},
		{"maxNegativeInt256>>over64", []*big.Int{sizeMaxIntNegative, sizeOverUint64}, zeroInput, mostNegativeShiftRight, nil},
	}
	runOverflowTests(t, vm.SAR, tests)
}

func TestSHRInstruction(t *testing.T) {

	sizeOverUint64, _ := big.NewInt(0).SetString("0x100000000000000001", 0)
	sizeOverUint64ByOne, _ := big.NewInt(0).SetString("0x80000000000000000", 0)
	sizeMaxUint256, _ := big.NewInt(0).SetString("0xFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF", 0)

	tests := []overflowTestCase{
		{"all zero", []*big.Int{big.NewInt(0), big.NewInt(0)}, zeroInput, big.NewInt(0), nil},
		{"0>>1", []*big.Int{big.NewInt(0), big.NewInt(1)}, zeroInput, big.NewInt(0), nil},
		{"over64>>1", []*big.Int{sizeOverUint64, big.NewInt(1)}, zeroInput, sizeOverUint64ByOne, nil},
		{"over64>>over64", []*big.Int{sizeOverUint64, sizeOverUint64}, zeroInput, big.NewInt(0), nil},
		{"sizeMaxUint256>>255", []*big.Int{sizeMaxUint256, big.NewInt(255)}, zeroInput, big.NewInt(1), nil},
	}
	runOverflowTests(t, vm.SHR, tests)
}

func TestSHLInstruction(t *testing.T) {

	sizeOverUint64, _ := big.NewInt(0).SetString("0x100000000000000001", 0)
	sizeOverUint64ByOne, _ := big.NewInt(0).SetString("0x200000000000000002", 0)
	sizeUint256, _ := big.NewInt(0).SetString("0x4000000000000000000000000000000000000000000000000000000000000000", 0)
	sizeMaxUint256, _ := big.NewInt(0).SetString("0xFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF", 0)
	sizeUint256result, _ := big.NewInt(0).SetString("0x8000000000000000000000000000000000000000000000000000000000000000", 0)

	tests := []overflowTestCase{
		{"all zero", []*big.Int{big.NewInt(0), big.NewInt(0)}, zeroInput, big.NewInt(0), nil},
		{"0<<1", []*big.Int{big.NewInt(0), big.NewInt(1)}, zeroInput, big.NewInt(0), nil},
		{"over64<<1", []*big.Int{sizeOverUint64, big.NewInt(1)}, zeroInput, sizeOverUint64ByOne, nil},
		{"over64<<over64", []*big.Int{sizeOverUint64, sizeOverUint64}, zeroInput, big.NewInt(0), nil},
		{"sizeUint256<<1", []*big.Int{sizeUint256, big.NewInt(1)}, zeroInput, sizeUint256result, nil},
		{"sizeMaxUint256<<255", []*big.Int{sizeMaxUint256, big.NewInt(255)}, zeroInput, sizeUint256result, nil},
	}
	runOverflowTests(t, vm.SHL, tests)
}

var zeroInput = []byte{}

type overflowTestCase struct {
	name      string
	arguments []*big.Int
	input     []byte
	result    *big.Int
	err       []error
}

func runOverflowTests(t *testing.T, instruction vm.OpCode, tests []overflowTestCase) {
	// For every variant of interpreter
	for _, variant := range getAllInterpreterVariantsForTests() {

		for _, revision := range revisions {

			for _, test := range tests {

				t.Run(fmt.Sprintf("%s/%s/%s/%s", variant, revision, instruction, test.name), func(t *testing.T) {

					evm := GetCleanEVM(revision, variant, nil)

					// Generate code
					code, _ := addValuesToStack(test.arguments, 0)
					code = append(code, byte(instruction))
					returnCode, _ := getReturnStackCode(1, 0, 0)
					code = append(code, returnCode...)

					// Run an interpreter
					res, err := evm.Run(code, test.input)
					if err != nil {
						t.Fatalf("unexpected internal interpreter error: %v", err)
					}

					// Check the result.
					if test.result != nil {

						if len(res.Output) >= 32 {
							result := big.NewInt(0).SetBytes(res.Output[0:32])
							if result.Cmp(test.result) != 0 {
								t.Errorf("execution result is different want: %v, got: %v", test.result, result)
							}
						} else {
							t.Errorf("execution has no return value, should have %v", test.result)
						}

					}

					want := len(test.err) == 0
					got := res.Success
					if want != got {
						t.Errorf("wanted success = %t, got %t", want, got)
					}
				})
			}
		}
	}
}

func TestCallDataLoadInstructionInputOverflow(t *testing.T) {

	sizeUint64, sizeOverUint64, sizeUint256 := getCornerSizeValues()

	b := []byte{1}
	input := common.LeftPadSlice(b[:], 32)

	tests := []overflowTestCase{
		{"all zero", []*big.Int{big.NewInt(0), big.NewInt(0)}, zeroInput, big.NewInt(0), nil},
		{"data input one", []*big.Int{big.NewInt(0), big.NewInt(0)}, input, big.NewInt(1), nil},
		{"data input one, offset one", []*big.Int{big.NewInt(0), big.NewInt(1)}, input, big.NewInt(256), nil},
		{"data input one, offset 31", []*big.Int{big.NewInt(0), big.NewInt(31)}, input, big.NewInt(0).SetBytes(common.RightPadSlice(b[:], 32)), nil},
		{"data input one, offset 300", []*big.Int{big.NewInt(0), big.NewInt(300)}, input, big.NewInt(0), nil},
		{"data input one, offset uint64", []*big.Int{big.NewInt(0), sizeUint64}, input, big.NewInt(0), nil},
		{"data input one, offset over64", []*big.Int{big.NewInt(0), sizeOverUint64}, input, big.NewInt(0), nil},
		{"data input one, offset uint256", []*big.Int{big.NewInt(0), sizeUint256}, input, big.NewInt(0), nil},
	}
	runOverflowTests(t, vm.CALLDATALOAD, tests)
}

func TestCallDataCopyInstructionInputOverflow(t *testing.T) {

	_, sizeOverUint64, sizeUint256 := getCornerSizeValues()

	b := []byte{1}
	input := common.LeftPadSlice(b[:], 32)

	tests := []overflowTestCase{
		{"all zero", []*big.Int{big.NewInt(0), big.NewInt(0), big.NewInt(0), big.NewInt(0)}, zeroInput, big.NewInt(0), nil},
		{"length 100", []*big.Int{big.NewInt(1), big.NewInt(100), big.NewInt(0), big.NewInt(0)}, input, big.NewInt(1), nil},
		{"length maxUint64", []*big.Int{big.NewInt(1), big.NewInt(0).SetUint64(math.MaxUint64), big.NewInt(0), big.NewInt(0)}, input, nil, []error{geth.ErrGasUintOverflow, geth.ErrOutOfGas}},
		{"length over64", []*big.Int{big.NewInt(1), sizeOverUint64, big.NewInt(0), big.NewInt(0)}, input, nil, []error{geth.ErrGasUintOverflow, geth.ErrOutOfGas}},
		{"length uint256", []*big.Int{big.NewInt(1), sizeUint256, big.NewInt(0), big.NewInt(0)}, input, nil, []error{geth.ErrGasUintOverflow, geth.ErrOutOfGas}},
		{"memory offset 100", []*big.Int{big.NewInt(1), big.NewInt(1), big.NewInt(0), big.NewInt(100)}, input, big.NewInt(1), nil},
		{"memory offset over64", []*big.Int{big.NewInt(1), big.NewInt(1), big.NewInt(0), sizeOverUint64}, input, nil, []error{geth.ErrGasUintOverflow, geth.ErrOutOfGas}},
		{"memory offset uint256", []*big.Int{big.NewInt(1), big.NewInt(1), big.NewInt(0), sizeUint256}, input, nil, []error{geth.ErrGasUintOverflow, geth.ErrOutOfGas}},
		{"data offset 100", []*big.Int{big.NewInt(1), big.NewInt(1), big.NewInt(1), big.NewInt(100), big.NewInt(0)}, input, big.NewInt(1), nil},
		{"data offset over64", []*big.Int{big.NewInt(1), big.NewInt(1), sizeOverUint64, big.NewInt(0)}, input, big.NewInt(1), nil},
		{"data offset uint256", []*big.Int{big.NewInt(1), big.NewInt(1), sizeUint256, big.NewInt(0)}, input, big.NewInt(1), nil},
	}
	runOverflowTests(t, vm.CALLDATACOPY, tests)
}

// Returns predefined memory offset or size corner values
func getCornerSizeValues() (sizeUint64, sizeOverUint64, sizeUint256 *big.Int) {
	sizeUint64, _ = big.NewInt(0).SetString("0x8000000000000001", 0)
	sizeOverUint64, _ = big.NewInt(0).SetString("0x100000000000000001", 0)
	sizeUint256, _ = big.NewInt(0).SetString("0x4000000000000000000000000000000000000000000000000000000000000001", 0)
	return
}

// Creates EVM code for returning specified number of 32byte values from stack
func getReturnStackCode(valuesCount uint32, initialOffset uint64, pushGas tosca.Gas) ([]byte, tosca.Gas) {
	var (
		retCode []byte = make([]byte, 0)
		usedGas tosca.Gas
	)
	for i := 0; i < int(valuesCount); i++ {
		bytes := getBytes(uint64(i*32) + initialOffset)
		retCode, usedGas = addBytesToStack(bytes, retCode, usedGas, pushGas)
		retCode = append(retCode, byte(vm.MSTORE))
		// Add 3 gas for MSTORE instruction static gas
		usedGas += 3
	}

	size := uint64(valuesCount * 32)
	usedGas += memoryExpansionGasCost(size)
	returtnCode, returnGas := getReturnMemoryCode(size, initialOffset, pushGas)

	return append(retCode, returtnCode...), usedGas + returnGas
}

// Creates EVM code for returning specified size of memory
func getReturnMemoryCode(size uint64, offset uint64, pushGas tosca.Gas) ([]byte, tosca.Gas) {
	var (
		retCode []byte = make([]byte, 0)
		gasUsed tosca.Gas
	)

	// memory size to return
	bytes := getBytes(size)
	retCode, gasUsed = addBytesToStack(bytes, retCode, gasUsed, pushGas)

	bytes = getBytes(offset)
	retCode, gasUsed = addBytesToStack(bytes, retCode, gasUsed, pushGas)

	retCode = append(retCode, byte(vm.RETURN))
	return retCode, gasUsed
}

// Returns trimmed byte array for uint64 number
func getBytes(num uint64) []byte {
	ret := make([]byte, 8)
	binary.BigEndian.PutUint64(ret, num)
	ret = bytes.TrimLeft(ret, string(byte(0)))
	return ret
}

// Get array of random big Integers
func getRandomBigIntArray(count int) []*big.Int {
	ret := make([]*big.Int, count)
	for i := 0; i < count; i++ {
		ret[i] = big.NewInt(0).SetBytes(getRadomByte32())
	}
	return ret
}

// Get 32 byte array of random bytes
func getRadomByte32() []byte {
	array := make([]byte, 32)
	for i := 0; i < 32; i++ {
		array[i] = byte(rand.Intn(256))
	}
	return array
}

func setDefaultCallStateDBMock(mockStateDB *MockStateDB, account tosca.Address, code []byte) {
	// mock state calls for call instruction
	mockStateDB.EXPECT().GetBalance(gomock.Any()).AnyTimes().Return(tosca.Value{1})
	mockStateDB.EXPECT().GetNonce(gomock.Any()).AnyTimes().Return(uint64(123))
	mockStateDB.EXPECT().SetNonce(gomock.Any(), gomock.Any()).AnyTimes()
	mockStateDB.EXPECT().GetCodeHash(gomock.Any()).AnyTimes().Return(tosca.Hash{})
	mockStateDB.EXPECT().GetCode(account).AnyTimes().Return(code)
	mockStateDB.EXPECT().IsAddressInAccessList(account).AnyTimes().Return(true)
	mockStateDB.EXPECT().EmitLog(gomock.Any()).AnyTimes()
	mockStateDB.EXPECT().GetStorage(gomock.Any(), gomock.Any()).AnyTimes().Return(tosca.Word{})
	mockStateDB.EXPECT().SetStorage(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
	mockStateDB.EXPECT().AccountExists(gomock.Any()).AnyTimes().Return(true)
	mockStateDB.EXPECT().HasSelfDestructed(gomock.Any()).AnyTimes().Return(true)
	mockStateDB.EXPECT().IsSlotInAccessList(gomock.Any(), gomock.Any()).AnyTimes().Return(true, true)
	mockStateDB.EXPECT().AccessAccount(gomock.Any()).AnyTimes().Return(tosca.WarmAccess)
	mockStateDB.EXPECT().AccessStorage(gomock.Any(), gomock.Any()).AnyTimes().Return(tosca.WarmAccess)
}

func newMockStateDBForIntegrationTests(ctrl *gomock.Controller) *MockStateDB {
	mockStateDB := NewMockStateDB(ctrl)

	// World state interactions always triggered by the EVM but not relevant for individual tests.
	mockStateDB.EXPECT().GetBalance(gomock.Any()).AnyTimes().Return(tosca.Value{1})
	mockStateDB.EXPECT().SetBalance(gomock.Any(), gomock.Any()).AnyTimes()
	mockStateDB.EXPECT().GetNonce(gomock.Any()).AnyTimes()
	mockStateDB.EXPECT().SetNonce(gomock.Any(), gomock.Any()).AnyTimes()
	mockStateDB.EXPECT().SetCode(gomock.Any(), gomock.Any()).AnyTimes()

	return mockStateDB
}

func TestEVM_CanSuccessfullyProcessPcBiggerThanCodeLength(t *testing.T) {
	for _, variant := range getAllInterpreterVariantsForTests() {
		revision := Istanbul
		t.Run(variant, func(t *testing.T) {
			// all implementations should be able to handle PC that goes beyond the code length.
			// this is reproducible by not calling STOP, RETURN, REVERT or SELFDESTRUCT at the end of the code.
			code := []byte{byte(vm.PUSH1), byte(32)}
			evm := GetCleanEVM(revision, variant, nil)

			// Run an interpreter
			result, err := evm.Run(code, []byte{})

			// Check the result.
			if err != nil || !result.Success {
				t.Errorf("execution should not fail and err should be nil, error is: %v, success %v", err, result.Success)
			}
		})

	}
}
