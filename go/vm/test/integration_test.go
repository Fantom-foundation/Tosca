package vm_test

import (
	"fmt"
	"math/big"
	"testing"

	vm_mock "github.com/Fantom-foundation/Tosca/go/vm/test/mocks"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/golang/mock/gomock"
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
			{"normal size over offset", instructionCase.instruction, sizeHuge, sizeHuge, []error{vm.ErrGasUintOverflow, vm.ErrOutOfGas}},
			{"zero size over offset", instructionCase.instruction, big.NewInt(0), sizeOverUint64, instructionCase.okError},
		}
		tests = append(tests, testForInstruction...)
	}

	// For every variant of interpreter
	for _, variant := range Variants {

		for _, revision := range revisions {

			for _, test := range tests {

				t.Run(fmt.Sprintf("%s/%s/%s/%s", variant, revision, test.instruction, test.name), func(t *testing.T) {

					callStackValues := []*big.Int{test.size, test.offset}
					code, _ := addValuesToStack(callStackValues, 0)
					code = append(code, byte(test.instruction))

					evm := GetCleanEVM(revision, variant, nil)

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
