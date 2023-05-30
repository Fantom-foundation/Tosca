package vm_test

import (
	"fmt"
	"math/big"
)

type DynGasTest struct {
	testName    string
	stackValues []*big.Int
	expectedGas int
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
		expectedGas := 10 + (i+1)*50
		// Append test
		testCases = append(testCases, &DynGasTest{testName, stackValues, expectedGas})
	}
	return testCases
}
