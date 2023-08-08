package vm_test

import (
	"fmt"
	"math/big"

	vm_mock "github.com/Fantom-foundation/Tosca/go/vm/test/mocks"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/golang/mock/gomock"
	"github.com/holiman/uint256"
)

// Structure for dynamic gas instruction test
type DynGasTest struct {
	testName    string                                 // test name
	stackValues []*big.Int                             // values to be put on stack
	expectedGas uint64                                 // gas amout after test evaluation
	mockCalls   func(mockStateDB *vm_mock.MockStateDB) // defines expected stateDB calls during test execution
	memValues   []*big.Int
}

// Structure for dynamic gas instruction test
type FailGasTest struct {
	testName      string                                 // test name
	instruction   vm.OpCode                              // tested instruction opcode
	expectedError error                                  // error at the end of execution
	stackValues   []*big.Int                             // values to be put on stack
	initialGas    uint64                                 // gas amout for the test
	mockCalls     func(mockStateDB *vm_mock.MockStateDB) // defines expected stateDB calls during test execution
}

// EXP instruction
// gas_cost = 10 + 50 * byte_len_exponent
// byte_len_exponent: the number of bytes in the exponent (exponent is b in the stack representation)
func gasEXP(revision Revision) []*DynGasTest {

	testCases := []*DynGasTest{}

	for i := 0; i < 32; i++ {
		exp := big.NewInt(1)
		num := big.NewInt(5)
		testName := fmt.Sprint(num) + "**1<<" + fmt.Sprint(i*8)
		stackValues := []*big.Int{exp.Lsh(exp, uint(i)*8), num}
		expectedGas := uint64(10 + (i+1)*50)
		// Append test
		testCases = append(testCases, &DynGasTest{testName, stackValues, expectedGas, nil, nil})
	}
	return testCases
}

// SHA3 instruction
// 30 is a static gas
// gas_cost = 30 + 6 * data_size_words + mem_expansion_cost
// data_size: size of the message to hash in bytes (len in the stack representation)
// data_size_words = (data_size + 31) // 32: number of (32-byte) words in the message to hash
// mem_expansion_cost: the cost of any memory expansion required (see A0-1)
func gasDynamicSHA3(revision Revision) []*DynGasTest {
	return getDynamicMemGas(6, 2)
}

// The following applies for the operations CALLDATACOPY and CODECOPY (not EXTCODECOPY)
// RETURNDATACOPY needs an external call to have return data to be copied

// data_size: size of the data to copy in bytes (len in the stack representation)
// data_size_words = (data_size + 31) // 32: number of (32-byte) words in the data to copy
// mem_expansion_cost: the cost of any memory expansion required (see A0-1)
// gas_cost = 3 + 3 * data_size_words + mem_expansion_cost
func gasDynamicCopy(revision Revision) []*DynGasTest {
	return getDynamicMemGas(3, 3)
}

// Common function for SHA3, CALLDATACOPY, CODECOPY and RETURNDATACOPY
func getDynamicMemGas(gasCoeficient uint64, numStackValues int) []*DynGasTest {
	testCases := []*DynGasTest{}

	for i := 0; i < 10; i++ {
		// Steps of 256 bytes memory addition to check non linear gas cost for expansion
		var dataSize uint64 = 256 * uint64(i)
		offset := big.NewInt(0)
		testName := "size " + fmt.Sprint(dataSize)
		stackValues := []*big.Int{big.NewInt(int64(dataSize)), offset}
		if numStackValues > len(stackValues) {
			for i := len(stackValues); i < numStackValues; i++ {
				stackValues = append(stackValues, offset)
			}
		}
		expectedGas := gasCoeficient*getDataSizeWords(dataSize) + memoryExpansionGasCost(dataSize)
		// Append test
		testCases = append(testCases, &DynGasTest{testName, stackValues, expectedGas, nil, nil})
	}
	return testCases
}

// EXTCODECOPY instruction
// target_addr: the address to copy code from (addr in the stack representation)
// access_cost: The cost of accessing a warm vs. cold account (see A0-2)
// access_cost = 100 if target_addr in touched_addresses (warm access)
// access_cost = 2600 if target_addr not in touched_addresses (cold access)
// data_size: size of the data to copy in bytes (len in the stack representation)
// data_size_words = (data_size + 31) // 32: number of (32-byte) words in the data to copy
// mem_expansion_cost: the cost of any memory expansion required (see A0-1)
//
// Gas Calculation:
// gas_cost = access_cost + 3 * data_size_words + mem_expansion_cost
func gasDynamicExtCodeCopy(revision Revision) []*DynGasTest {

	testCases := []*DynGasTest{}
	copyCode := make([]byte, 0, 1000)
	name := []string{"Address in access list", "Address not in access list"}

	for i := 0; i < 10; i++ {
		address := common.Address{byte(i + 1)}
		hash := common.Hash{byte(i + 1)}

		inAccessList := i%2 == 0
		accessCost := getAccessCost(revision, inAccessList, false)

		// Steps of 256 bytes memory addition to check non linear gas cost for expansion
		var dataSize uint64 = 256 * uint64(i)
		offset := big.NewInt(0)
		testName := name[i%2] + " size " + fmt.Sprint(dataSize)
		stackValues := []*big.Int{big.NewInt(int64(dataSize)), offset, offset, address.Hash().Big()}

		// Expected gas calculation
		expectedGas := accessCost + 3*getDataSizeWords(dataSize) + memoryExpansionGasCost(dataSize)

		mockCalls := func(mockStateDB *vm_mock.MockStateDB) {
			mockStateDB.EXPECT().GetCodeHash(address).AnyTimes().Return(hash)
			mockStateDB.EXPECT().GetCode(address).AnyTimes().Return(copyCode)
			mockStateDB.EXPECT().AddressInAccessList(address).AnyTimes().Return(inAccessList)
			mockStateDB.EXPECT().AddAddressToAccessList(address).AnyTimes()
		}
		// Append test
		testCases = append(testCases, &DynGasTest{testName, stackValues, expectedGas, mockCalls, nil})
	}
	return testCases
}

// The opcodes BALANCE, EXTCODESIZE, EXTCODEHASH have the same pricing function
// based on making a single account access. See A0-2 for details on EIP-2929 and touched_addresses.

// target_addr: the address of interest (addr in the opcode stack representations)
// Gas Calculation:
// For Istanbul revision these use only static gas
// gas_cost = 100 if target_addr in touched_addresses (warm access)
// gas_cost = 2600 if target_addr not in touched_addresses (cold access)
func gasDynamicAccountAccess(revision Revision) []*DynGasTest {

	type accessTest struct {
		testName         string
		addrInAccessList bool
	}

	tests := []accessTest{
		{"Address in access list", true},
		{"Address not in access list", false},
	}

	testCases := []*DynGasTest{}

	for i, test := range tests {
		address := common.Address{byte(i + 1)}
		hash := common.Hash{byte(i + 1)}
		inAccessList := test.addrInAccessList
		// Expected gas calculation
		expectedGas := getAccessCost(revision, test.addrInAccessList, false)
		stackValues := []*big.Int{address.Hash().Big()}
		mockCalls := func(mockStateDB *vm_mock.MockStateDB) {
			mockStateDB.EXPECT().AddressInAccessList(address).AnyTimes().Return(inAccessList)
			mockStateDB.EXPECT().AddAddressToAccessList(address).AnyTimes()
			mockStateDB.EXPECT().GetCodeSize(address).AnyTimes().Return(0)
			mockStateDB.EXPECT().GetCodeHash(address).AnyTimes().Return(hash)
			mockStateDB.EXPECT().Empty(address).AnyTimes().Return(false)
			mockStateDB.EXPECT().GetBalance(address).AnyTimes().Return(big.NewInt(0))
		}
		// Append test
		testCases = append(testCases, &DynGasTest{test.testName, stackValues, expectedGas, mockCalls, nil})
	}
	return testCases
}

// SLOAD instruction
// context_addr: the address of the current execution context (i.e. what ADDRESS would put on the stack)
// target_storage_key: The 32-byte storage index to load from (key in the stack representation)
// Gas Calculation:

// gas_cost = 100 if (context_addr, target_storage_key) in touched_storage_slots (warm access)
// gas_cost = 2100 if (context_addr, target_storage_key) not in touched_storage_slots (cold access)
func gasDynamicSLoad(revision Revision) []*DynGasTest {

	type sloadTest struct {
		testName         string
		slotInAccessList bool
	}

	tests := []sloadTest{
		{"Address in ACL, slot in ACL", true},
		{"Address in ACL, slot not in access list", false},
	}

	testCases := []*DynGasTest{}

	for i, test := range tests {
		address := common.Address{byte(0)}
		slot := common.Hash{byte(i + 1)}
		slotInACL := test.slotInAccessList
		stackValues := []*big.Int{slot.Big()}
		mockCalls := func(mockStateDB *vm_mock.MockStateDB) {
			mockStateDB.EXPECT().SlotInAccessList(address, slot).AnyTimes().Return(true, slotInACL)
			mockStateDB.EXPECT().AddSlotToAccessList(address, slot).AnyTimes()
			mockStateDB.EXPECT().GetState(address, slot).AnyTimes().Return(common.Hash{})
		}
		// Expected gas calculation
		expectedGas := getAccessCost(revision, test.slotInAccessList, true)

		// Append test
		testCases = append(testCases, &DynGasTest{test.testName, stackValues, expectedGas, mockCalls, nil})
	}
	return testCases
}

// SSTORE instruction
// context_addr: the address of the current execution context (i.e. what ADDRESS would put on the stack)
// target_storage_key: The 32-byte storage index to store to (key in the stack representation)
// orig_val: the value of the storage slot if the current transaction is reverted
// current_val: the value of the storage slot immediately before the sstore op in question
// new_val: the value of the storage slot immediately after the sstore op in question
func gasDynamicSStore(revision Revision) []*DynGasTest {

	type sloadTest struct {
		testName            string
		addressInAccessList bool
		slotInAccessList    bool
		origValue           common.Hash
		currentValue        common.Hash
		newValue            common.Hash
	}

	val0 := common.Hash{0}
	val1 := common.Hash{1}
	val2 := common.Hash{2}

	tests := []sloadTest{
		{"Address in ACL, slot in ACL", true, true, val1, val2, val2},
		{"Address in ACL, slot not in ACL", true, false, val1, val2, val2},
		{"Address not in ACL, slot in ACL", false, true, val1, val2, val2},

		{"Slot 0, current 0, new 1", true, true, val0, val0, val1},
		{"Slot 1, current 1, new 2", true, true, val1, val1, val2},
		{"Slot 1, current 1, new 0", true, true, val1, val1, val0},
		{"Slot 1, current 0, new 2", true, true, val1, val0, val2},
		{"Slot 1, current 2, new 0", true, true, val1, val2, val0},
		{"Slot 1, current 2, new 1", true, true, val1, val2, val1},
		{"Slot 0, current 1, new 0", true, true, val0, val1, val0},
	}

	testCases := []*DynGasTest{}

	for i, test := range tests {
		address := common.Address{byte(0)}
		key := common.Hash{byte(i + 1)}

		origValue := test.origValue
		currentValue := test.currentValue
		newValue := test.newValue

		addressInACL := test.addressInAccessList
		slotInACL := test.slotInAccessList
		stackValues := []*big.Int{newValue.Big(), key.Big()}

		// TODO: if remaining gas < 2300 there has to be OUT_OF_GAS error

		// Expected gas calculation
		var expectedGas uint64
		var gasRefund int

		// Access list access for slots in SSTORE
		warmAccessCost, coldAccessCost := getSStoreAccessCost(revision, true)

		if !test.slotInAccessList {
			expectedGas += coldAccessCost
		}

		refundGas, resetGas := getSStoreGasAmounts(revision)

		expectedGas, gasRefund = calculateSStoreGas(origValue, currentValue, newValue, expectedGas, warmAccessCost, coldAccessCost, refundGas, resetGas)

		mockCalls := func(mockStateDB *vm_mock.MockStateDB) {
			mockStateDB.EXPECT().SlotInAccessList(address, key).AnyTimes().Return(addressInACL, slotInACL)
			mockStateDB.EXPECT().AddSlotToAccessList(address, key).AnyTimes()
			mockStateDB.EXPECT().GetCommittedState(address, key).AnyTimes().Return(origValue)
			mockStateDB.EXPECT().GetState(address, key).AnyTimes().Return(currentValue)
			mockStateDB.EXPECT().SetState(address, key, newValue).AnyTimes()
			if gasRefund != 0 {
				if gasRefund > 0 {
					mockStateDB.EXPECT().AddRefund(uint64(gasRefund))
				} else {
					mockStateDB.EXPECT().SubRefund(uint64(gasRefund * -1))
				}
			}
		}

		// Append test
		testCases = append(testCases, &DynGasTest{test.testName, stackValues, expectedGas, mockCalls, nil})
	}
	return testCases
}

type instructionGasTest struct {
	instruction vm.OpCode
	initialGas  uint64
}

func testsContainOpCode(op vm.OpCode, tests []instructionGasTest) bool {
	for _, test := range tests {
		if test.instruction == op {
			return true
		}
	}
	return false
}

// Tests for which dynamic gas should run out and OutOfGas error should happen
func getOutOfDynamicGasTests(revision Revision) []*FailGasTest {
	var (
		// SSTORE has to fail if gas < 2300
		sstoreLowGas uint64 = 2299
		// This gas is not sufficient for cold gas access
		accessLowGas uint64 = 300
		// Memory expansion for a 1 word and offset by 1 gas is 6
		memoryLowGasTwoWords uint64 = 5
		// Same as above with added 3 static gas for instruction
		memoryLowGasTwoWordsWithStatic = memoryLowGasTwoWords + 3
		// Memory expanded by 1 word needs 3 gas
		memoryLowGasOneWord uint64 = 2
		// Copy of one word is 3 + 3 for static gas
		copyLowGas uint64 = 5
		// Log of size 1 needs 8 gas
		logLowGas uint64 = 7
		// Log static gas multiplier
		logStaticGas uint64 = 375
		// Exp is 50 * exponent byte len which is 1 for test
		expLowGas uint64 = 49
		// SHA3 static gas is 30 + needed memory expansion, then 6 * word size
		sha3LowGas uint64 = 30 + 3 + 5
	)
	testCases := []*FailGasTest{}

	tests := []instructionGasTest{
		{vm.SSTORE, sstoreLowGas},
		{vm.SLOAD, accessLowGas},
		{vm.BALANCE, accessLowGas},
		{vm.EXTCODESIZE, accessLowGas},
		{vm.EXTCODEHASH, accessLowGas},
		{vm.EXTCODECOPY, accessLowGas},
		{vm.CALL, accessLowGas},
		{vm.STATICCALL, accessLowGas},
		{vm.DELEGATECALL, accessLowGas},
		{vm.CALLCODE, accessLowGas},
		{vm.SELFDESTRUCT, accessLowGas},
		{vm.CREATE, accessLowGas},
		{vm.CREATE2, accessLowGas},
		{vm.EXP, expLowGas},
		{vm.CODECOPY, copyLowGas},
		{vm.CALLDATACOPY, copyLowGas},
		{vm.MLOAD, memoryLowGasTwoWordsWithStatic},
		{vm.MSTORE, memoryLowGasTwoWordsWithStatic},
		{vm.MSTORE8, memoryLowGasOneWord},
		{vm.LOG0, 1*logStaticGas + logLowGas},
		{vm.LOG1, 2*logStaticGas + logLowGas},
		{vm.LOG2, 3*logStaticGas + logLowGas},
		{vm.LOG3, 4*logStaticGas + logLowGas},
		{vm.LOG4, 5*logStaticGas + logLowGas},
		{vm.SHA3, sha3LowGas},
		{vm.RETURN, memoryLowGasOneWord},
		{vm.REVERT, memoryLowGasOneWord},
	}

	// Check if all opcodes with dynamic gas calculation are present in the tests
	for op, info := range getInstructions(revision) {
		if op == vm.RETURNDATACOPY || // can't be tested in this way because of inner call needed
			info.gas.dynamic == nil {
			continue
		} else {
			if !testsContainOpCode(op, tests) {
				panic(fmt.Sprintf("dynamic out of gas tests don't contain instruction %v for revision %v", op.String(), revision))
			}
		}
	}

	mockCalls := func(mockStateDB *vm_mock.MockStateDB) {
		mockStateDB.EXPECT().SlotInAccessList(gomock.Any(), gomock.Any()).AnyTimes().Return(false, false)
		mockStateDB.EXPECT().AddressInAccessList(gomock.Any()).AnyTimes().Return(false)
		mockStateDB.EXPECT().AddAddressToAccessList(gomock.Any()).AnyTimes()
		mockStateDB.EXPECT().Exist(gomock.Any()).AnyTimes().Return(false)
		mockStateDB.EXPECT().AddSlotToAccessList(gomock.Any(), gomock.Any()).AnyTimes()
		mockStateDB.EXPECT().Empty(gomock.Any()).AnyTimes().Return(true)
		mockStateDB.EXPECT().GetBalance(gomock.Any()).AnyTimes().Return(big.NewInt(0))
		mockStateDB.EXPECT().HasSuicided(gomock.Any()).AnyTimes().Return(true)
		mockStateDB.EXPECT().GetState(gomock.Any(), gomock.Any()).AnyTimes().Return(common.Hash{1})
	}

	// Generate test cases
	for _, test := range tests {
		stackValCount := getInstructions(revision)[test.instruction].stack.popped
		stackValues := make([]*big.Int, 0)
		for i := 0; i < stackValCount; i++ {
			stackValues = append(stackValues, big.NewInt(1))
		}
		testName := fmt.Sprintf("%v using %v gas", test.instruction.String(), test.initialGas)

		testCases = append(testCases, &FailGasTest{testName, test.instruction, vm.ErrOutOfGas, stackValues, test.initialGas, mockCalls})
	}

	return testCases
}

// Returns refund and reset gas values according to revision
func getSStoreGasAmounts(revision Revision) (int, uint64) {
	switch revision {
	case Istanbul:
		return 15000, 5000
	case Berlin:
		return 15000, 2900
	case London:
		return 4800, 2900
	default:
		return 0, 0
	}
}

// Returns expected gas and gas to refund for a SSTORE instruction
func calculateSStoreGas(origValue common.Hash, currentValue common.Hash, newValue common.Hash, expectedGas uint64, warmAccessCost uint64, coldAccessCost uint64, refundAmount int, resetGasAmount uint64) (uint64, int) {
	zeroVal := common.Hash{0}
	var gasRefund int

	if newValue == currentValue {
		expectedGas += warmAccessCost
	} else {
		if currentValue == origValue {
			if origValue == zeroVal {
				expectedGas += 20000
			} else {
				expectedGas += resetGasAmount
				if newValue == zeroVal {
					gasRefund += refundAmount
				}
			}
		} else {
			expectedGas += warmAccessCost
			if origValue != zeroVal {
				if currentValue == zeroVal {
					gasRefund -= refundAmount
				} else if newValue == zeroVal {
					gasRefund += refundAmount
				}
			}
			if newValue == origValue {
				if origValue == zeroVal {
					gasRefund += 20000 - int(warmAccessCost)
				} else {
					gasRefund += 5000 - int(coldAccessCost) - int(warmAccessCost)
				}
			}
		}
	}
	return expectedGas, gasRefund
}

// LOG instruction
// num_topics: the * of the LOG* op. e.g. LOG0 has num_topics = 0, LOG4 has num_topics = 4
// data_size: size of the data to log in bytes (len in the stack representation).
// mem_expansion_cost: the cost of any memory expansion required (see A0-1)
// Gas Calculation:
// gas_cost = 375 + 375 * num_topics + 8 * data_size + mem_expansion_cost
func gasDynamicLog0(revision Revision) []*DynGasTest {
	return gasDynamicLog(revision, 0)
}
func gasDynamicLog1(revision Revision) []*DynGasTest {
	return gasDynamicLog(revision, 1)
}
func gasDynamicLog2(revision Revision) []*DynGasTest {
	return gasDynamicLog(revision, 2)
}
func gasDynamicLog3(revision Revision) []*DynGasTest {
	return gasDynamicLog(revision, 3)
}
func gasDynamicLog4(revision Revision) []*DynGasTest {
	return gasDynamicLog(revision, 4)
}

func gasDynamicLog(revision Revision, size int) []*DynGasTest {

	testCases := []*DynGasTest{}
	for i := 0; i < 100; i++ {

		// Steps of 256 bytes memory addition to check non linear gas cost for expansion
		var dataSize uint64 = 256 * uint64(i)
		offset := big.NewInt(0)
		testName := "size " + fmt.Sprint(dataSize)

		stackValues := []*big.Int{}
		for j := 0; j < size; j++ {
			stackValues = append(stackValues, common.Hash{byte(j)}.Big())
		}
		stackValues = append(stackValues, big.NewInt(int64(dataSize)), offset)

		// Expected gas calculation
		expectedGas := uint64(375+375*size) + 8*dataSize + memoryExpansionGasCost(dataSize)

		mockCalls := func(mockStateDB *vm_mock.MockStateDB) {
			mockStateDB.EXPECT().AddLog(gomock.Any()).AnyTimes()
		}
		// Append test
		testCases = append(testCases, &DynGasTest{testName, stackValues, expectedGas, mockCalls, nil})
	}
	return testCases
}

// CALL instruction
// base_gas = access_cost + mem_expansion_cost
// If call_value > 0 (sending value with call):
// base_gas += 9000
// If is_empty(target_addr) (forcing a new account to be created in the state trie):
// base_gas += 25000
// Calculate the gas_sent_with_call below.

// And the final cost of the operation:
// gas_cost = base_gas + gas_sent_with_call

// call_value: the value sent with the call (val in the stack representation)
// target_addr: the recipient of the call (addr in the stack representation)
// access_cost: The cost of accessing a warm vs. cold account (see A0-2)
// access_cost = 100 if target_addr in touched_addresses (warm access)
// access_cost = 2600 if target_addr not in touched_addresses (cold access)
// mem_expansion_cost: the cost of any memory expansion required (see A0-1)
// gas_sent_with_call: the gas ultimately sent with the call
func gasDynamicCall(revision Revision) []*DynGasTest {
	return gasDynamicCallCommon(revision, true, true)
}

// STATICCALL & DELEGATECALL instruction
// Same as CALL instruction, but not using call value
func gasDynamicStaticDelegateCall(revision Revision) []*DynGasTest {
	return gasDynamicCallCommon(revision, false, false)
}

// CALLCODE instruction
// Same as CALL instruction, different address creation gas computation
func gasDynamicCallCodeCall(revision Revision) []*DynGasTest {
	return gasDynamicCallCommon(revision, true, false)
}

func gasDynamicCallCommon(revision Revision, useCallValue bool, addressCreationGas bool) []*DynGasTest {

	calledCode := []byte{byte(vm.STOP)}

	type callTest struct {
		testName         string
		addrExist        bool
		addrInAccessList bool
		callValue        *big.Int
	}

	tests := []callTest{
		{"Address exist and in access list", true, true, big.NewInt(0)},
		{"Address exist and not in access list", true, false, big.NewInt(0)},
	}

	if useCallValue {
		tests = append(tests, []callTest{
			{"Call value is > 0", true, true, big.NewInt(1)},
			{"Call value is > 0 & address not exist", false, false, big.NewInt(1)},
		}...)
	}

	testCases := []*DynGasTest{}

	for i, test := range tests {
		address := common.Address{byte(i + 1)}
		hash := common.Hash{byte(i + 1)}
		exist := test.addrExist
		inAccessList := test.addrInAccessList
		mockCalls := func(mockStateDB *vm_mock.MockStateDB) {
			mockStateDB.EXPECT().Snapshot().AnyTimes().Return(0)
			mockStateDB.EXPECT().GetCodeHash(address).AnyTimes().Return(hash)
			mockStateDB.EXPECT().GetCode(address).AnyTimes().Return(calledCode)
			mockStateDB.EXPECT().Exist(address).AnyTimes().Return(exist)
			mockStateDB.EXPECT().AddressInAccessList(address).AnyTimes().Return(inAccessList)
			mockStateDB.EXPECT().AddAddressToAccessList(address).AnyTimes()
			mockStateDB.EXPECT().CreateAccount(address).AnyTimes()
			mockStateDB.EXPECT().AddBalance(address, big.NewInt(0)).AnyTimes()
			mockStateDB.EXPECT().Empty(address).AnyTimes().Return(!exist)
		}

		// The WarmStorageReadCostEIP2929 (100) is already deducted in the form of a constant cost, so
		// the cost to charge for cold access, if any, is Cold - Warm
		expectedGas := getAccessCost(revision, test.addrInAccessList, false)

		// Include also memory
		step := 256 * uint64(i)
		dataSize := big.NewInt(int64(step))
		memExpansionCost := memoryExpansionGasCost(step)

		expectedGas += memExpansionCost

		if useCallValue && test.callValue.Cmp(big.NewInt(0)) > 0 {
			expectedGas += 9000 - 2300
			if addressCreationGas && !test.addrExist {
				expectedGas += 25000
			}
		}

		// gas_sent_with_call
		requestedGas := uint64(InitialTestGas)
		remainingGas := InitialTestGas - expectedGas
		allButOne64th := remainingGas - (remainingGas / uint64(64))
		var gasSentWithCall uint64
		if requestedGas < allButOne64th {
			gasSentWithCall = requestedGas
		} else {
			gasSentWithCall = allButOne64th
		}

		zeroVal := big.NewInt(0)

		var stackValues []*big.Int
		if useCallValue {
			// retSize, retOffset, inSize, inOffset, value, addr, provided_gas
			stackValues = []*big.Int{zeroVal, zeroVal, dataSize, zeroVal, test.callValue, address.Hash().Big(), big.NewInt(int64(gasSentWithCall))}
		} else {
			// retSize, retOffset, inSize, inOffset, addr, provided_gas
			stackValues = []*big.Int{zeroVal, zeroVal, dataSize, zeroVal, address.Hash().Big(), big.NewInt(int64(gasSentWithCall))}
		}

		// Append test
		testCases = append(testCases, &DynGasTest{test.testName, stackValues, expectedGas, mockCalls, nil})
	}
	return testCases
}

// RETURN, MLOAD, MSTORE, MSTORE8 instructions
// Use memory expansion cost if it expands memory, otherwise only static gas is used
func gasDynamicMemory(revision Revision) []*DynGasTest {

	testCases := []*DynGasTest{}
	var data uint64 = 32
	offset := big.NewInt(0)
	testName := "size " + fmt.Sprint(data)

	stackValues := []*big.Int{big.NewInt(int64(data)), offset}

	// Expected gas calculation
	expectedGas := memoryExpansionGasCost(data)

	// Append test
	testCases = append(testCases, &DynGasTest{testName, stackValues, expectedGas, nil, nil})

	return testCases
}

// CREATE instruction
// mem_expansion_cost: the cost of any memory expansion required (see A0-1)
// code_deposit_cost: the per-byte cost incurred for storing the deployed code (see A9-F).
// Gas Calculation:
// code_deposit_cost = 200 * returned_code_size
// gas_cost = 32000 + mem_expansion_cost + code_deposit_cost
func gasDynamicCreate(revision Revision) []*DynGasTest {
	return gasDynCreate(revision, false)
}

// CREATE2 instruction
// mem_expansion_cost: the cost of any memory expansion required (see A0-1)
// code_deposit_cost: the per-byte cost incurred for storing the deployed code (see A9-F).
// returned_code_size: the length of the returned runtime code
// Gas Calculation:
// code_deposit_cost = 200 * returned_code_size
// gas_cost = 32000 + 6 * data_size_words + mem_expansion_cost + code_deposit_cost
func gasDynamicCreate2(revision Revision) []*DynGasTest {
	return gasDynCreate(revision, true)
}

func gasDynCreate(revision Revision, isCreate2 bool) []*DynGasTest {

	var emptyCodeHash = crypto.Keccak256Hash(nil)

	testCases := []*DynGasTest{}
	for i := 0; i < 3; i++ {

		offset := big.NewInt(0)
		value := big.NewInt(0)
		returnSize := 4 * i // different sizes of returned data
		code, retrunGas, codeLength := getCreateContractCode(returnSize)
		codeVal := big.NewInt(0).SetBytes(code)

		// Values to put into evm memory
		memValues := []*big.Int{codeVal, offset}
		dataSize := len(code)

		callerAddress := common.Address{0}
		hash := crypto.Keccak256Hash(code)
		testName := "return size " + fmt.Sprint(returnSize)

		// Expected gas calculation
		expectedGas := uint64(retrunGas) // For gas used in new contract execution

		var contractAddr common.Address
		var stackValues []*big.Int
		if isCreate2 {
			salt := big.NewInt(1)
			saltBytes := uint256.NewInt(1)
			stackValues = []*big.Int{salt, big.NewInt(int64(dataSize)), offset, value}
			contractAddr = crypto.CreateAddress2(callerAddress, saltBytes.Bytes32(), hash.Bytes())
			expectedGas += getDataSizeWords(codeLength) * 6
		} else {
			contractAddr = crypto.CreateAddress(callerAddress, 0)
			stackValues = []*big.Int{big.NewInt(int64(dataSize)), offset, value}
		}
		// memory expansion cost
		expectedGas += memoryExpansionGasCost(uint64(dataSize))
		// code_deposit_cost
		expectedGas += uint64(200 * returnSize)

		mockCalls := func(mockStateDB *vm_mock.MockStateDB) {
			mockStateDB.EXPECT().GetNonce(callerAddress).AnyTimes().Return(uint64(0))
			mockStateDB.EXPECT().SetNonce(callerAddress, uint64(1))
			mockStateDB.EXPECT().GetCodeHash(contractAddr).Return(emptyCodeHash)
			mockStateDB.EXPECT().GetNonce(contractAddr).Return(uint64(0))
			mockStateDB.EXPECT().Snapshot().Return(1)
			mockStateDB.EXPECT().CreateAccount(contractAddr)
			mockStateDB.EXPECT().SetCode(contractAddr, gomock.Any())
			mockStateDB.EXPECT().AddAddressToAccessList(contractAddr).AnyTimes()
			mockStateDB.EXPECT().SetNonce(contractAddr, uint64(1))
		}
		// Append test
		testCases = append(testCases, &DynGasTest{testName, stackValues, expectedGas, mockCalls, memValues})
	}
	return testCases
}

// Returns contract code with its gas cost for CREATE instruction
func getCreateContractCode(returnSize int) ([]byte, uint64, uint64) {
	code := [32]byte{}
	code[0] = byte(vm.PUSH1)
	code[1] = byte(0)
	code[2] = byte(vm.PUSH1)
	code[3] = byte(returnSize)
	code[4] = byte(vm.MSTORE)
	code[5] = byte(vm.PUSH1)
	code[6] = byte(returnSize)
	code[7] = byte(vm.PUSH1)
	code[8] = byte(0)
	code[9] = byte(vm.RETURN)

	codeLength := uint64(10)

	expansionCost := memoryExpansionGasCost(uint64(returnSize))
	// 18 = 4x3 for PUSH1 + 3 for MSTORE + 3 for RETURN
	execGas := uint64(18) + expansionCost

	return code[:], execGas, codeLength
}

// SELFDESTRUCT instruction
// target_addr: the recipient of the self-destructing contract's funds (addr in the stack representation)
// context_addr: the address of the current execution context (i.e. what ADDRESS would put on the stack)

// Gas Calculation
// gas_cost = 5000: base cost
// If balance(context_addr) > 0 && is_empty(target_addr) (sending funds to a previously empty address):
// gas_cost += 25000
// If target_addr not in touched_addresses (cold access):
// gas_cost += 2600
func gasDynamicSelfDestruct(revision Revision) []*DynGasTest {

	testCases := []*DynGasTest{}

	type selfdestructTest struct {
		testName        string
		balance         int
		targetAddrEmpty bool
		targetAddrInACL bool
		hasSuicided     bool
	}

	tests := []selfdestructTest{
		{"Target address empty, in ACL, no balance, not suicided", 0, true, true, false},
		{"Target address empty, in ACL, with balance, not suicided", 1, true, true, false},
		{"Target address empty, not in ACL no balance, not suicided", 0, true, false, false},
		{"Target address empty, not in ACL with balance, not suicided", 1, true, false, false},
		{"Target address empty, in ACL, no balance, suicided", 0, true, true, true},
		{"Target address empty, in ACL, with balance suicided", 1, true, true, true},
		{"Target address empty, not in ACL, no balance suicided", 0, true, false, true},
		{"Target address empty, not in ACL, with balance suicided", 1, true, false, true},
		{"Target address not empty, in ACL, no balance, not suicided", 0, false, true, false},
		{"Target address not empty, in ACL, with balance, not suicided", 1, false, true, false},
		{"Target address not empty, not in ACL no balance, not suicided", 0, false, false, false},
		{"Target address not empty, not in ACL with balance, not suicided", 1, false, false, false},
		{"Target address not empty, in ACL, no balance, suicided", 0, false, true, true},
		{"Target address not empty, in ACL, with balance suicided", 1, false, true, true},
		{"Target address not empty, not in ACL, no balance suicided", 0, false, false, true},
		{"Target address not empty, not in ACL, with balance suicided", 1, false, false, true},
	}

	for i, test := range tests {
		// Offset target address from contract address
		targetAddress := common.Address{byte(i + 1)}
		contractAddress := common.Address{0}
		empty := test.targetAddrEmpty
		balance := test.balance
		inAcl := test.targetAddrInACL
		suicided := test.hasSuicided

		stackValues := []*big.Int{targetAddress.Hash().Big()}

		// Expected gas calculation
		expectedGas := uint64(5000)

		// Sending balance to an empty address
		if empty && balance > 0 {
			expectedGas += 25000
		}

		// Cold access for a target address
		if !inAcl && revision >= Berlin {
			expectedGas += 2600
		}

		mockCalls := func(mockStateDB *vm_mock.MockStateDB) {
			mockStateDB.EXPECT().HasSuicided(contractAddress).AnyTimes().Return(suicided)
			mockStateDB.EXPECT().Empty(targetAddress).AnyTimes().Return(empty)

			mockStateDB.EXPECT().Suicide(contractAddress).AnyTimes().Return(true)
			mockStateDB.EXPECT().GetBalance(contractAddress).AnyTimes().Return(big.NewInt(int64(balance)))
			mockStateDB.EXPECT().AddBalance(targetAddress, big.NewInt(int64(balance))).AnyTimes()
			mockStateDB.EXPECT().Exist(targetAddress).AnyTimes().Return(!empty)
			mockStateDB.EXPECT().AddressInAccessList(targetAddress).AnyTimes().Return(inAcl)
			mockStateDB.EXPECT().AddAddressToAccessList(targetAddress).AnyTimes()
			if revision < London {
				if !suicided {
					mockStateDB.EXPECT().AddRefund(uint64(24000))
				}
			}
		}
		// Append test
		testCases = append(testCases, &DynGasTest{test.testName, stackValues, expectedGas, mockCalls, nil})
	}
	return testCases
}

// A0-1: Memory Expansion
// new_mem_size: the highest referenced memory address after the operation in question (in bytes)
// new_mem_size_words = (new_mem_size + 31) // 32
// gas_cost = (new_mem_size_words ^ 2 // 512) + (3 * new_mem_size_words)
// The memory cost function is linear up to 724 bytes of memory used, at which point additional memory costs substantially more.
func memoryExpansionGasCost(newMemSize uint64) uint64 {
	newMemSizeWords := (newMemSize + 31) / 32
	gasCost := ((newMemSizeWords * newMemSizeWords) / 512) + (3 * newMemSizeWords)
	return gasCost
}

// Address access
// access_cost: The cost of accessing a warm vs. cold account (see A0-2)
// access_cost = 100 if target_addr in touched_addresses (warm access)
// access_cost = 2600 if target_addr not in touched_addresses (cold access)
//
// Slot access
// gas_cost = 100 if (context_addr, target_storage_key) in touched_storage_slots (warm access)
// gas_cost = 2100 if (context_addr, target_storage_key) not in touched_storage_slots (cold access)
//
// Static access cost is included in the instruction info and added during
// dynamic gas computation
func getAccessCost(revision Revision, warmAccess bool, isSlot bool) uint64 {

	if warmAccess {
		// Warm access is already included as a static gas in instruction info
		return 0
	} else {

		switch revision {
		case Istanbul:
			return 0
		default:
			if !isSlot {
				// 2600 - 100 static gas at instruction info
				return 2500
			} else {
				// 2100 - 100 static gas at instruction info
				return 2000
			}
		}
	}
}

// Returns ACL access cost for SSTORE
// No static gas for instruction is involved
func getSStoreAccessCost(revision Revision, warmAccess bool) (uint64, uint64) {
	switch revision {
	case Istanbul:
		return 800, 0
	default:
		return 100, 2100
	}
}

// getDataSizeWords computesword size of data
func getDataSizeWords(dataSize uint64) uint64 {
	dataSizeWords := (dataSize + 31) / 32
	return dataSizeWords
}
