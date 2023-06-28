package lfvm

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/holiman/uint256"
)

// ----------------------------- Super Instructions -----------------------------

// checkResultSuper checks the result with an expectation
// status = expected status; res = expected result; res_pc = expected code pointer
func checkResultSuper(t *testing.T, ctxt *context, status Status, res *[]uint256.Int, res_pc int32) {
	if ctxt.stack.len() != len(*res) {
		t.Errorf("expected stack size of %d, got %d", len(*res), ctxt.stack.len())
		return
	}

	if status != ctxt.status {
		t.Errorf("expected status %s, got %s", status.String(), ctxt.status.String())
	}

	for i := len(*res) - 1; i >= 0; i-- {
		r := (*res)[i]
		got := ctxt.stack.pop()

		if !r.Eq(got) {
			t.Errorf("expected[%d] %s, got %s", i, r.Hex(), got.Hex())
		}
	}

	if ctxt.pc != res_pc {
		t.Errorf("expected pc %d, got %d", res_pc, ctxt.pc)
	}
}

// getTestEnvDataSuper creates context and prepares data into stack
func getTestEnvDataSuper(aData []uint256.Int, aCode []Instruction) *context {
	ctxt := context{
		code:  aCode,
		stack: NewStack(),
	}
	for _, d := range aData {
		ctxt.stack.push(&d)
	}
	return &ctxt
}

// runTestSuperInstr executes the individual tests defined in testData
func runTestSuperInstr(t *testing.T, testData []tTestDataSuperOp) {
	for _, data := range testData {
		t.Run(data.name, func(t *testing.T) {
			ctxt := getTestEnvDataSuper(data.data, data.code)

			// Create a dummy contract
			addr := vm.AccountRef{}
			ctxt.contract = vm.NewContract(addr, addr, big.NewInt(0), data.gas)

			data.op(ctxt)

			// control of the consumed gas
			expectedGas := data.gas
			consumedGas := data.gas - ctxt.contract.Gas
			if consumedGas != expectedGas {
				t.Errorf("expected consumed gas %d, got %d", expectedGas, consumedGas)
			}

			checkResultSuper(t, ctxt, data.status, &data.res, data.res_pc)
		})
	}
}

// TestSuperInstr tests super instructions.
func TestSuperInstr(t *testing.T) {
	// preparation of data
	testData := testDataSuperOp

	// execution of tests
	runTestSuperInstr(t, testData)
}
