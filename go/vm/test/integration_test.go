package vm_test

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math/big"
	"math/rand"
	"testing"

	vm_mock "github.com/Fantom-foundation/Tosca/go/vm/test/mocks"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/golang/mock/gomock"
	"github.com/holiman/uint256"
)

const HUGE_GAS_SENT_WITH_CALL int64 = 1000000000000

func TestMaxCallDepth(t *testing.T) {
	var mockCtrl *gomock.Controller
	var mockStateDB *vm_mock.MockStateDB

	// For every variant of interpreter
	for _, variant := range Variants {

		for _, revision := range revisions {

			t.Run(fmt.Sprintf("%s/%s", variant, revision), func(t *testing.T) {

				mockCtrl = gomock.NewController(t)
				mockStateDB = vm_mock.NewMockStateDB(mockCtrl)

				zeroVal := big.NewInt(0)
				account := common.Address{byte(0)}

				// return and input data size is 32bytes, memory offset is 0 for all
				callStackValues := []*big.Int{big.NewInt(32), zeroVal, big.NewInt(32), zeroVal, zeroVal, account.Hash().Big(), big.NewInt(HUGE_GAS_SENT_WITH_CALL)}
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
				if err != nil {
					t.Errorf("execution failed and should not fail, error is: %v", err)
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
	for _, variant := range Variants {

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
					_, err := evm.Run(code, []byte{})

					// Check the result.
					if err == nil {
						t.Errorf("execution should fail with error : %v", vm.ErrInvalidJump)
					} else {
						if err != vm.ErrInvalidJump {
							t.Errorf("execution should fail with error : %v but got: %v", vm.ErrInvalidJump, err)
						}
					}
				})
			}
		}
	}
}

func TestReturnDataCopy(t *testing.T) {
	var mockCtrl *gomock.Controller
	var mockStateDB *vm_mock.MockStateDB

	type test struct {
		name           string
		dataSize       uint
		dataOffset     *big.Int
		returnDataSize uint
		err            error
	}

	zero := big.NewInt(0)
	one := big.NewInt(1)
	overflowValue, _ := big.NewInt(0).SetString("0x1000000000000000d", 0)

	tests := []test{
		{name: "no data", dataSize: 0, dataOffset: zero, returnDataSize: 0, err: nil},
		{name: "offset > return data", dataSize: 0, dataOffset: one, returnDataSize: 0, err: vm.ErrReturnDataOutOfBounds},
		{name: "same data", dataSize: 0, dataOffset: one, returnDataSize: 1, err: nil},
		{name: "size > return data", dataSize: 1, dataOffset: zero, returnDataSize: 0, err: vm.ErrReturnDataOutOfBounds},
		{name: "size + offset > return data", dataSize: 1, dataOffset: one, returnDataSize: 0, err: vm.ErrReturnDataOutOfBounds},
		{name: "same data", dataSize: 1, dataOffset: zero, returnDataSize: 2, err: nil},
		{name: "offset overflow", dataSize: 0, dataOffset: overflowValue, returnDataSize: 0, err: vm.ErrReturnDataOutOfBounds},
	}
	// For every variant of interpreter
	for _, variant := range Variants {

		for _, revision := range revisions {

			for _, tst := range tests {

				t.Run(fmt.Sprintf("%s/%s/%s", variant, revision, tst.name), func(t *testing.T) {

					mockCtrl = gomock.NewController(t)
					mockStateDB = vm_mock.NewMockStateDB(mockCtrl)

					zeroVal := big.NewInt(0)
					account := common.Address{byte(0)}
					returnDataSize := big.NewInt(10)

					// stack values for inner contract call
					callStackValues := []*big.Int{returnDataSize, zeroVal, zeroVal, zeroVal, zeroVal, account.Hash().Big(), big.NewInt(HUGE_GAS_SENT_WITH_CALL)}
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
					_, err := evm.Run(code, []byte{})

					// Check the result.
					if err != tst.err {
						if tst.err == nil {
							t.Errorf("execution should not fail, but got error: %v", err)
						} else {
							t.Errorf("execution should fail with error: %v, but got:%v", tst.err, err)
						}
					}
				})
			}
		}
	}
}

func TestReadOnlyStaticCall(t *testing.T) {
	var mockCtrl *gomock.Controller
	var mockStateDB *vm_mock.MockStateDB

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
	for _, variant := range Variants {

		for _, revision := range revisions {

			for _, call := range calls {

				for _, instruction := range readOnlyInstructions {

					t.Run(fmt.Sprintf("%s/%s/%s/%s", variant, revision, call.instruction, instruction), func(t *testing.T) {

						mockCtrl = gomock.NewController(t)
						mockStateDB = vm_mock.NewMockStateDB(mockCtrl)

						zeroVal := big.NewInt(0)
						account := common.Address{byte(0)}

						// stack values for inner contract call
						callStackValues := []*big.Int{zeroVal, zeroVal, zeroVal, zeroVal, zeroVal, account.Hash().Big(), big.NewInt(HUGE_GAS_SENT_WITH_CALL)}
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

						res := big.NewInt(0).SetBytes(result.Output[0:32])
						success := res.Cmp(zeroVal) != 0
						if success == call.shouldFail {
							t.Errorf("execution should fail because of read only call, but did not fail")
						}

						if err != nil {
							t.Errorf("execution should not return an error, but got:%v", err)
						}
					})
				}
			}
		}
	}
}

func TestInstructionDataInitialization(t *testing.T) {

	var mockCtrl *gomock.Controller
	var mockStateDB *vm_mock.MockStateDB

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
		{vm.REVERT, []error{vm.ErrExecutionReverted}},
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
			{"huge size normal offset", instructionCase.instruction, sizeHuge, sizeNormal, []error{vm.ErrOutOfGas}},
			{"over size normal offset", instructionCase.instruction, sizeOverUint64, sizeNormal, []error{vm.ErrGasUintOverflow, vm.ErrOutOfGas}},
			{"normal size huge offset", instructionCase.instruction, sizeNormal, sizeHuge, []error{vm.ErrOutOfGas}},
			{"normal size over offset", instructionCase.instruction, sizeNormal, sizeOverUint64, []error{vm.ErrGasUintOverflow, vm.ErrOutOfGas}},
			{"huge size huge offset", instructionCase.instruction, sizeHuge, sizeHuge, []error{vm.ErrGasUintOverflow, vm.ErrOutOfGas}},
			{"zero size over offset", instructionCase.instruction, big.NewInt(0), sizeOverUint64, instructionCase.okError},
		}
		tests = append(tests, testForInstruction...)
	}

	// For every variant of interpreter
	for _, variant := range Variants {

		for _, revision := range revisions {

			for _, test := range tests {

				t.Run(fmt.Sprintf("%s/%s/%s/%s", variant, revision, test.instruction, test.name), func(t *testing.T) {

					mockCtrl = gomock.NewController(t)
					mockStateDB = vm_mock.NewMockStateDB(mockCtrl)
					// set mock for inner call
					setDefaultCallStateDBMock(mockStateDB, common.Address{byte(0)}, make([]byte, 0))

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
					_, err := evm.Run(code, []byte{})

					// Check the result.
					if err == nil && test.err != nil {
						t.Errorf("execution should fail with error: %v", test.err)
					}

					if test.err == nil && err != nil {
						t.Errorf("execution should not fail but got: %v", err)
					} else {
						if err != nil && !contains(test.err, err) {
							t.Errorf("execution should fail with error: %v, but got: %v", test.err, err)
						}
					}
				})
			}
		}
	}
}

func TestCreateDataInitialization(t *testing.T) {

	var mockCtrl *gomock.Controller
	var mockStateDB *vm_mock.MockStateDB

	account := common.Address{byte(0)}
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
		{"over size zero offset", numOverUint64, big.NewInt(0), []error{vm.ErrGasUintOverflow, vm.ErrOutOfGas}},
		{"over size over offset", numOverUint64, numOverUint64, []error{vm.ErrGasUintOverflow, vm.ErrOutOfGas}},
		{"add size and offset overflows", numHuge, numHuge, []error{vm.ErrGasUintOverflow, vm.ErrOutOfGas}},
	}

	// For every variant of interpreter
	for _, variant := range Variants {

		for _, revision := range revisions {

			for _, instruction := range instructions {

				for _, test := range tests {

					t.Run(fmt.Sprintf("%s/%s/%s/%s", variant, revision, instruction, test.name), func(t *testing.T) {

						mockCtrl = gomock.NewController(t)
						mockStateDB = vm_mock.NewMockStateDB(mockCtrl)

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
						_, err := evm.Run(code, []byte{})

						// Check the result.
						if err == nil && test.err != nil {
							t.Errorf("execution should fail with error: %v", test.err)
						}

						if test.err == nil && err != nil {
							t.Errorf("execution should not fail but got: %v", err)
						} else {
							if err != nil && !contains(test.err, err) {
								t.Errorf("execution should fail with error: %v, but got: %v", test.err, err)
							}
						}
					})
				}
			}
		}
	}
}

func TestMemoryNotWrittenWithZeroReturnData(t *testing.T) {

	var mockCtrl *gomock.Controller
	var mockStateDB *vm_mock.MockStateDB

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
	for _, variant := range Variants {

		for _, revision := range revisions {

			for _, call := range calls {

				t.Run(fmt.Sprintf("%s/%s/%s/%s", variant, revision, call.instruction, call.callOutputMemSize), func(t *testing.T) {

					mockCtrl = gomock.NewController(t)
					mockStateDB = vm_mock.NewMockStateDB(mockCtrl)

					account := common.Address{byte(0)}

					// Store data into memory to test, if it would be overwritten
					wantMemWord := getRandomBigIntArray(1)
					wantMemWord = append(wantMemWord, zeroVal)
					code, _ := addValuesToStack(wantMemWord, 0)
					code = append(code,
						byte(vm.MSTORE))

					// stack values for inner contract call
					var callStackValues []*big.Int
					if call.instruction == vm.CALL || call.instruction == vm.CALLCODE {
						callStackValues = []*big.Int{call.callOutputMemSize, zeroVal, zeroVal, zeroVal, zeroVal, account.Hash().Big(), big.NewInt(HUGE_GAS_SENT_WITH_CALL)}
					} else {
						callStackValues = []*big.Int{call.callOutputMemSize, zeroVal, zeroVal, zeroVal, account.Hash().Big(), big.NewInt(HUGE_GAS_SENT_WITH_CALL)}
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
							t.Errorf("memmory should change when return data size > 0, but it didn't")
						} else {
							t.Errorf("memmory should not change when return data size is 0, but it did")
						}
					}

					if gotMemSize.Uint64() != call.afterCallMemSize.Uint64() {
						t.Errorf("memmory size after call is not as expected, want %v, got %v", call.afterCallMemSize.Uint64(), gotMemSize.Uint64())
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

	var mockCtrl *gomock.Controller
	var mockStateDB *vm_mock.MockStateDB

	account := common.Address{byte(0)}
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
	for _, variant := range Variants {

		if skipTestForVariant(t.Name(), variant) {
			continue
		}

		for _, revision := range revisions {

			for _, test := range tests {

				t.Run(fmt.Sprintf("%s/%s/%s/%s", variant, revision, test.name, test.createInstruction), func(t *testing.T) {

					mockCtrl = gomock.NewController(t)
					mockStateDB = vm_mock.NewMockStateDB(mockCtrl)

					createStackValues := []*big.Int{big.NewInt(32), big.NewInt(0), big.NewInt(0)}
					if test.createInstruction == vm.CREATE2 {
						createStackValues = append([]*big.Int{big.NewInt(0)}, createStackValues...)
					}
					createInputCode := []byte{
						byte(vm.PUSH1), byte(32),
						byte(vm.PUSH1), byte(0),
						byte(test.returnInstruction)}

					createInputBytes := common.RightPadBytes(createInputCode, 32)
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

					contractAddr := crypto.CreateAddress2(account, [32]byte{}, crypto.Keccak256Hash(createInputBytes).Bytes())

					// set mock for inner call
					setDefaultCallStateDBMock(mockStateDB, account, make([]byte, 0))
					mockStateDB.EXPECT().GetCodeHash(contractAddr).AnyTimes().Return(common.Hash{byte(0)})
					mockStateDB.EXPECT().CreateAccount(contractAddr).AnyTimes()
					mockStateDB.EXPECT().SetCode(contractAddr, gomock.Any()).AnyTimes()

					evm := GetCleanEVM(revision, variant, mockStateDB)

					// Run an interpreter
					result, err := evm.Run(code, []byte{})
					returnDataSize := big.NewInt(0).SetBytes(result.Output[0:32])

					// Check the result.
					if returnDataSize.Uint64() != test.returnDataSize {
						t.Errorf("expected return data size: %v, but got: %v", test.returnDataSize, returnDataSize)
					}

					if err != nil {
						t.Errorf("execution should not fail but got: %v", err)
					}
				})
			}
		}
	}
}

func TestExtCodeHashOnEmptyAccount(t *testing.T) {
	var mockCtrl *gomock.Controller
	var mockStateDB *vm_mock.MockStateDB

	type extCodeHashTest struct {
		name   string
		exist  bool
		empty  bool
		result common.Hash
		hash   common.Hash
	}

	codeHash := common.Hash{byte(2)}

	tests := []extCodeHashTest{
		{"account for slot exist and is empty", true, true, common.Hash{byte(0)}, codeHash},
		{"account for slot doesn't exist is empty", false, true, common.Hash{byte(0)}, codeHash},
		{"account for slot exist and is not empty", true, false, codeHash, codeHash},
		{"account for slot doesn't exist and is not empty", false, false, common.Hash{byte(0)}, codeHash},
	}

	// For every variant of interpreter
	for _, variant := range Variants {

		if skipTestForVariant(t.Name(), variant) {
			continue
		}

		for _, revision := range revisions {
			for _, test := range tests {

				t.Run(fmt.Sprintf("%s/%s/%s", variant, revision, test.name), func(t *testing.T) {

					mockCtrl = gomock.NewController(t)
					mockStateDB = vm_mock.NewMockStateDB(mockCtrl)

					account := common.Address{byte(1)}

					// stack values for inner contract call
					code, _ := addValuesToStack([]*big.Int{account.Hash().Big()}, 0)

					// code for inner contract call
					code = append(code, []byte{
						byte(vm.EXTCODEHASH),
						byte(vm.PUSH1), byte(0),
						byte(vm.MSTORE),
						byte(vm.PUSH1), byte(32),
						byte(vm.PUSH1), byte(0),
						byte(vm.RETURN)}...)

					// set mock for inner call
					mockStateDB.EXPECT().Empty(account).AnyTimes().Return(test.empty)
					mockStateDB.EXPECT().Exist(account).AnyTimes().Return(test.exist)
					mockStateDB.EXPECT().AddressInAccessList(account).AnyTimes().Return(false)
					mockStateDB.EXPECT().AddAddressToAccessList(account).AnyTimes()
					// when account doesn't exists stateDB should take care about it
					if test.exist {
						mockStateDB.EXPECT().GetCodeHash(account).AnyTimes().Return(test.hash)
					} else {
						mockStateDB.EXPECT().GetCodeHash(account).AnyTimes().Return(common.Hash{byte(0)})
					}

					evm := GetCleanEVM(revision, variant, mockStateDB)

					// Run an interpreter
					result, err := evm.Run(code, []byte{})

					// Check the result.
					if !bytes.Equal(result.Output, test.result.Bytes()) {
						t.Errorf("execution should return zero value on stack, got, %v", result.Output)
					}

					if err != nil {
						t.Errorf("execution should not fail, but got error: %v", err)
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

	tests := []shiftTestCase{
		{"all zero", big.NewInt(0), big.NewInt(0), big.NewInt(0)},
		{"0>>1", big.NewInt(0), big.NewInt(1), big.NewInt(0)},
		{"over64>>1", sizeOverUint64, big.NewInt(1), sizeOverUint64ByOne},
		{"over64>>over64", sizeOverUint64, sizeOverUint64, big.NewInt(0)},
		{"maxPositiveInt256>>254", sizeMaxIntPositive, big.NewInt(254), big.NewInt(1)},

		{"negative>>0", negativeInt, big.NewInt(0), negativeInt},
		{"negative>>2", negativeInt, big.NewInt(2), getNegativeBigIntSignInBits(big.NewInt(-8))},
		{"negativeOver64>>1", negativeOverUint64, big.NewInt(1), negativeOverUint64ByOne},
		{"negativeOver64>>257", negativeOverUint64, big.NewInt(257), mostNegativeShiftRight},
		{"negativeOver64>>over64", negativeOverUint64, sizeOverUint64, mostNegativeShiftRight},
		{"maxNegativeInt256>>1", sizeMaxIntNegative, big.NewInt(1), sizeMaxIntNegativeByOne},
		{"maxNegativeInt256>>254", sizeMaxIntNegative, big.NewInt(254), getNegativeBigIntSignInBits(big.NewInt(-2))},
		{"maxNegativeInt256>>over64", sizeMaxIntNegative, sizeOverUint64, mostNegativeShiftRight},
	}
	runShiftTests(t, vm.SAR, tests)
}

func TestSHRInstruction(t *testing.T) {

	sizeOverUint64, _ := big.NewInt(0).SetString("0x100000000000000001", 0)
	sizeOverUint64ByOne, _ := big.NewInt(0).SetString("0x80000000000000000", 0)
	sizeMaxUint256, _ := big.NewInt(0).SetString("0xFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF", 0)

	tests := []shiftTestCase{
		{"all zero", big.NewInt(0), big.NewInt(0), big.NewInt(0)},
		{"0>>1", big.NewInt(0), big.NewInt(1), big.NewInt(0)},
		{"over64>>1", sizeOverUint64, big.NewInt(1), sizeOverUint64ByOne},
		{"over64>>over64", sizeOverUint64, sizeOverUint64, big.NewInt(0)},
		{"sizeMaxUint256>>255", sizeMaxUint256, big.NewInt(255), big.NewInt(1)},
	}
	runShiftTests(t, vm.SHR, tests)
}

func TestSHLInstruction(t *testing.T) {

	sizeOverUint64, _ := big.NewInt(0).SetString("0x100000000000000001", 0)
	sizeOverUint64ByOne, _ := big.NewInt(0).SetString("0x200000000000000002", 0)
	sizeUint256, _ := big.NewInt(0).SetString("0x4000000000000000000000000000000000000000000000000000000000000000", 0)
	sizeMaxUint256, _ := big.NewInt(0).SetString("0xFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF", 0)
	sizeUint256result, _ := big.NewInt(0).SetString("0x8000000000000000000000000000000000000000000000000000000000000000", 0)

	tests := []shiftTestCase{
		{"all zero", big.NewInt(0), big.NewInt(0), big.NewInt(0)},
		{"0<<1", big.NewInt(0), big.NewInt(1), big.NewInt(0)},
		{"over64<<1", sizeOverUint64, big.NewInt(1), sizeOverUint64ByOne},
		{"over64<<over64", sizeOverUint64, sizeOverUint64, big.NewInt(0)},
		{"sizeUint256<<1", sizeUint256, big.NewInt(1), sizeUint256result},
		{"sizeMaxUint256<<255", sizeMaxUint256, big.NewInt(255), sizeUint256result},
	}
	runShiftTests(t, vm.SHL, tests)
}

type shiftTestCase struct {
	name   string
	value  *big.Int
	shift  *big.Int
	result *big.Int
}

func runShiftTests(t *testing.T, instruction vm.OpCode, tests []shiftTestCase) {
	// For every variant of interpreter
	for _, variant := range Variants {

		for _, revision := range revisions {

			for _, test := range tests {

				t.Run(fmt.Sprintf("%s/%s/%s/%s", variant, revision, "SAR", test.name), func(t *testing.T) {

					evm := GetCleanEVM(revision, variant, nil)

					// Generate code
					code, _ := addValuesToStack([]*big.Int{test.value, test.shift}, 0)
					code = append(code, byte(instruction))
					returnCode, _ := getReturnStackCode(1, 0, 0)
					code = append(code, returnCode...)

					// Run an interpreter
					res, err := evm.Run(code, []byte{})

					// Check the result.
					if res.Output != nil && len(res.Output) >= 32 {
						result := big.NewInt(0).SetBytes(res.Output[0:32])
						if result.Cmp(test.result) != 0 {
							t.Errorf("execution result is different want: %v, got: %v", test.result, result)
						}
					} else {
						t.Errorf("execution should return a result with stack values")
					}

					if err != nil {
						t.Errorf("execution should not fail with error, but got: %v", err)
					}
				})
			}
		}
	}
}

// Creates EVM code for returning specified number of 32byte values from stack
func getReturnStackCode(valuesCount uint32, initialOffset uint64, pushGas uint64) ([]byte, uint64) {
	var (
		retCode []byte = make([]byte, 0)
		usedGas uint64
	)
	for i := 0; i < int(valuesCount); i++ {
		bytes := getBytes(uint64(i*32) + initialOffset)
		retCode, usedGas = addBytesTostack(bytes, retCode, usedGas, pushGas)
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
func getReturnMemoryCode(size uint64, offset uint64, pushGas uint64) ([]byte, uint64) {
	var (
		retCode []byte = make([]byte, 0)
		gasUsed uint64
	)

	// memory size to return
	bytes := getBytes(size)
	retCode, gasUsed = addBytesTostack(bytes, retCode, gasUsed, pushGas)

	bytes = getBytes(offset)
	retCode, gasUsed = addBytesTostack(bytes, retCode, gasUsed, pushGas)

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

func contains(s []error, elem error) bool {
	for _, a := range s {
		if a == elem {
			return true
		}
	}
	return false
}

func setDefaultCallStateDBMock(mockStateDB *vm_mock.MockStateDB, account common.Address, code []byte) {

	var emptyCodeHash = crypto.Keccak256Hash(nil)
	contractAddrCreate := crypto.CreateAddress(account, 0)
	contractAddrCreate2 := crypto.CreateAddress2(account, [32]byte{}, emptyCodeHash.Bytes())
	balance, _ := big.NewInt(0).SetString("0x2000000000000000", 0)

	// mock state calls for call instruction
	mockStateDB.EXPECT().GetRefund().AnyTimes().Return(uint64(0))
	mockStateDB.EXPECT().SubRefund(gomock.Any()).AnyTimes()
	mockStateDB.EXPECT().AddRefund(gomock.Any()).AnyTimes()
	mockStateDB.EXPECT().AddBalance(gomock.Any(), gomock.Any()).AnyTimes()
	mockStateDB.EXPECT().GetBalance(gomock.Any()).AnyTimes().Return(balance)
	mockStateDB.EXPECT().GetCodeHash(account).AnyTimes().Return(common.Hash{byte(0)})
	mockStateDB.EXPECT().GetCodeHash(contractAddrCreate).AnyTimes().Return(common.Hash{byte(0)})
	mockStateDB.EXPECT().CreateAccount(contractAddrCreate).AnyTimes()
	mockStateDB.EXPECT().GetCodeHash(contractAddrCreate2).AnyTimes().Return(common.Hash{byte(0)})
	mockStateDB.EXPECT().CreateAccount(contractAddrCreate2).AnyTimes()
	mockStateDB.EXPECT().Snapshot().AnyTimes().Return(0)
	mockStateDB.EXPECT().Exist(account).AnyTimes().Return(true)
	mockStateDB.EXPECT().GetCode(account).AnyTimes().Return(code)
	mockStateDB.EXPECT().SetCode(contractAddrCreate, gomock.Any()).AnyTimes()
	mockStateDB.EXPECT().SetCode(contractAddrCreate2, gomock.Any()).AnyTimes()
	mockStateDB.EXPECT().AddressInAccessList(account).AnyTimes().Return(true)
	mockStateDB.EXPECT().RevertToSnapshot(gomock.Any()).AnyTimes()
	mockStateDB.EXPECT().AddLog(gomock.Any()).AnyTimes()
	mockStateDB.EXPECT().GetState(gomock.Any(), gomock.Any()).AnyTimes().Return(common.Hash{byte(0)})
	mockStateDB.EXPECT().SetState(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
	mockStateDB.EXPECT().GetNonce(gomock.Any()).AnyTimes().Return(uint64(0))
	mockStateDB.EXPECT().SetNonce(gomock.Any(), gomock.Any()).AnyTimes()
	mockStateDB.EXPECT().Empty(gomock.Any()).AnyTimes().Return(true)
	mockStateDB.EXPECT().HasSuicided(gomock.Any()).AnyTimes().Return(true)
	mockStateDB.EXPECT().Suicide(gomock.Any()).AnyTimes().Return(true)
	mockStateDB.EXPECT().SlotInAccessList(gomock.Any(), gomock.Any()).AnyTimes().Return(true, true)
	mockStateDB.EXPECT().AddAddressToAccessList(gomock.Any()).AnyTimes()
}
