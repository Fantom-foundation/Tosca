package vm_test

import (
	"fmt"
	"math/big"
)

type DynGasTest struct {
	testName    string
	stackValues []*big.Int
	expectedGas uint64
}

// EXP instruction
// gas_cost = 10 + 50 * byte_len_exponent
// byte_len_exponent: the number of bytes in the exponent (exponent is b in the stack representation)
func gasEXP() []*DynGasTest {

	testCases := []*DynGasTest{}

	for i := 0; i < 32; i++ {
		exp := big.NewInt(1)
		num := big.NewInt(5)
		testName := fmt.Sprint(num) + "**1<<" + fmt.Sprint(i*8)
		stackValues := []*big.Int{exp.Lsh(exp, uint(i)*8), num}
		expectedGas := uint64(10 + (i+1)*50)
		// Append test
		testCases = append(testCases, &DynGasTest{testName, stackValues, expectedGas})
	}
	return testCases
}

// SHA3 instruction
// 30 is a static gas
// gas_cost = 30 + 6 * data_size_words + mem_expansion_cost
// data_size: size of the message to hash in bytes (len in the stack representation)
// data_size_words = (data_size + 31) // 32: number of (32-byte) words in the message to hash
// mem_expansion_cost: the cost of any memory expansion required (see A0-1)
func gasDynamicSHA3() []*DynGasTest {

	testCases := []*DynGasTest{}

	for i := 0; i < 10; i++ {
		// Steps of 256 bytes memory addition to check non linear gas cost for expansion
		var dataSize uint64 = 256 * uint64(i)
		offset := big.NewInt(0)
		dataSizeWords := (dataSize + 31) / 32
		testName := "size " + fmt.Sprint(dataSize)
		stackValues := []*big.Int{big.NewInt(int64(dataSize)), offset}
		expectedGas := 6*dataSizeWords + memoryExpansionGasCost(dataSize)
		// Append test
		testCases = append(testCases, &DynGasTest{testName, stackValues, expectedGas})
	}
	return testCases
}

// The following applies for the operations CALLDATACOPY and CODECOPY (not EXTCODECOPY)
// RETURNDATACOPY needs an external call to have return data to be copied

// data_size: size of the data to copy in bytes (len in the stack representation)
// data_size_words = (data_size + 31) // 32: number of (32-byte) words in the data to copy
// mem_expansion_cost: the cost of any memory expansion required (see A0-1)

// gas_cost = 3 + 3 * data_size_words + mem_expansion_cost

func gasDynamicCopy() []*DynGasTest {

	testCases := []*DynGasTest{}

	for i := 0; i < 10; i++ {
		// Steps of 256 bytes memory addition to check non linear gas cost for expansion
		var dataSize uint64 = 256 * uint64(i)
		offset := big.NewInt(0)
		dataSizeWords := (dataSize + 31) / 32
		testName := "size " + fmt.Sprint(dataSize)
		stackValues := []*big.Int{big.NewInt(int64(dataSize)), offset, offset}
		expectedGas := 3*dataSizeWords + memoryExpansionGasCost(dataSize)
		// Append test
		testCases = append(testCases, &DynGasTest{testName, stackValues, expectedGas})
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
