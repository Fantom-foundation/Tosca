package lfvm

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/holiman/uint256"
)

const defaultStackVal uint16 = 1

func getTestEnv(stackLen int, code Code) *context {
	ctxt := context{
		code:  code,
		stack: NewStack(),
	}
	for i := 0; i < stackLen; i++ {
		ctxt.stack.push(uint256.NewInt(uint64(defaultStackVal)))
	}
	return &ctxt
}

func TestStopInstruction(t *testing.T) {
	ctxt := getTestEnv(0, nil)
	opStop(ctxt)
	if ctxt.status != STOPPED {
		t.Errorf("expected status stopped = 1, got %v", ctxt.status)
		return
	}
}

func TestRevertInstruction(t *testing.T) {
	ctxt := getTestEnv(2, nil)
	opRevert(ctxt)
	if ctxt.stack.len() != 0 {
		t.Errorf("expected empty stack, got %d", ctxt.stack.len())
		return
	}
	if ctxt.status != REVERTED {
		t.Errorf("expected status reverted = 2, got %v", ctxt.status)
		return
	}
}

func TestReturnInstruction(t *testing.T) {
	ctxt := getTestEnv(2, nil)
	opReturn(ctxt)
	if ctxt.stack.len() != 0 {
		t.Errorf("expected empty stack, got %d", ctxt.stack.len())
		return
	}
	if ctxt.status != RETURNED {
		t.Errorf("expected status returned = 3, got %v", ctxt.status)
		return
	}
}

func TestInvalidInstruction(t *testing.T) {
	ctxt := getTestEnv(0, nil)
	opInvalid(ctxt)
	if ctxt.status != INVALID_INSTRUCTION {
		t.Errorf("expected status invalid_instruction = 5, got %v", ctxt.status)
		return
	}
}

func TestPcInstruction(t *testing.T) {
	code := Code{Instruction{opcode: PC, arg: defaultStackVal}}

	ctxt := getTestEnv(0, code)
	opPc(ctxt)
	if ctxt.stack.len() != 1 {
		t.Errorf("expected stack size of 1, got %d", ctxt.stack.len())
		return
	}
	if uint16(ctxt.stack.peek().Uint64()) != defaultStackVal {
		t.Errorf("expected stack value of %d, got %d", defaultStackVal, ctxt.stack.len())
		return
	}
}

func TestJumpInstruction(t *testing.T) {
	code := Code{
		Instruction{opcode: PC, arg: defaultStackVal},
		Instruction{opcode: JUMPDEST, arg: defaultStackVal},
	}

	ctxt := getTestEnv(1, code)
	opJump(ctxt)
	if ctxt.stack.len() != 0 {
		t.Errorf("expected empty stack, got %d", ctxt.stack.len())
		return
	}
	if ctxt.status != RUNNING {
		t.Errorf("expected status running = 0, got %v", ctxt.status)
		return
	}

	// test overflow of destination
	bignum := uint256.NewInt(uint64(defaultStackVal))
	bignum[0] = 1 << 40
	ctxt.stack.push(bignum)
	opJump(ctxt)
	if ctxt.status != ERROR {
		t.Errorf("expected status error = 7, got %v", ctxt.status)
		return
	}

	// test invalid destination
	invalidDest := uint256.NewInt(uint64(20))
	ctxt.stack.push(invalidDest)
	opJump(ctxt)
	if ctxt.status != ERROR {
		t.Errorf("expected status error = 7, got %v", ctxt.status)
		return
	}

}

func TestJumpiInstruction(t *testing.T) {
	code := Code{
		Instruction{opcode: PC, arg: defaultStackVal},
		Instruction{opcode: JUMPDEST, arg: defaultStackVal},
	}

	// ok path
	ctxt := getTestEnv(2, code)
	opJumpi(ctxt)
	if ctxt.stack.len() != 0 {
		t.Errorf("expected empty stack, got %d", ctxt.stack.len())
		return
	}
	if ctxt.status != RUNNING {
		t.Errorf("expected status running = 0, got %v", ctxt.status)
		return
	}

	// test overflow of destination
	ctxt.stack.push(uint256.NewInt(uint64(defaultStackVal)))
	bignum := uint256.NewInt(uint64(defaultStackVal))
	bignum[0] = 1 << 40
	ctxt.stack.push(bignum)
	opJumpi(ctxt)
	if ctxt.status != ERROR {
		t.Errorf("expected status error = 7, got %v", ctxt.status)
		return
	}

	// test invalid destination
	invalidDest := uint256.NewInt(uint64(20))
	ctxt.stack.push(uint256.NewInt(uint64(defaultStackVal)))
	ctxt.stack.push(invalidDest)
	opJumpi(ctxt)
	if ctxt.status != ERROR {
		t.Errorf("expected status error = 7, got %v", ctxt.status)
		return
	}

}

func TestPushN(t *testing.T) {
	data := make([]byte, 32)
	for i := range data {
		data[i] = byte(i + 1)
	}

	code := make([]Instruction, 16)
	for i := 0; i < 32; i++ {
		code[i/2].arg = code[i/2].arg<<8 | uint16(data[i])
	}

	for n := 1; n <= 32; n++ {
		ctxt := context{
			code:  code,
			stack: NewStack(),
		}

		opPush(&ctxt, n)
		ctxt.pc++

		if ctxt.stack.len() != 1 {
			t.Errorf("expected stack size of 1, got %d", ctxt.stack.len())
			return
		}

		if int(ctxt.pc) != n/2+n%2 {
			t.Errorf("for PUSH%d program counter did not progress to %d, got %d", n, n/2+n%2, ctxt.pc)
		}

		got := ctxt.stack.peek().Bytes()
		if len(got) != n {
			t.Errorf("expected %d bytes on the stack, got %d with values %v", n, len(got), got)
		}

		for i := range got {
			if data[i] != got[i] {
				t.Errorf("for PUSH%d expected value %d to be %d, got %d", n, i, data[i], got[i])
			}
		}
	}
}

func TestPush1(t *testing.T) {
	code := []Instruction{
		{opcode: PUSH1, arg: 0x1234},
	}

	ctxt := context{
		code:  code,
		stack: NewStack(),
	}

	opPush1(&ctxt)
	ctxt.pc++

	if ctxt.stack.len() != 1 {
		t.Errorf("expected stack size of 1, got %d", ctxt.stack.len())
		return
	}

	if int(ctxt.pc) != 1 {
		t.Errorf("program counter did not progress to %d, got %d", 1, ctxt.pc)
	}

	got := ctxt.stack.peek().Bytes()
	if len(got) != 1 {
		t.Errorf("expected 1 byte on the stack, got %d with values %v", len(got), got)
	}
	if got[0] != 0x12 {
		t.Errorf("expected %d for first byte, got %d", 0x12, got[0])
	}
}

func TestPush2(t *testing.T) {
	code := []Instruction{
		{opcode: PUSH2, arg: 0x1234},
	}

	ctxt := context{
		code:  code,
		stack: NewStack(),
	}

	opPush2(&ctxt)
	ctxt.pc++

	if ctxt.stack.len() != 1 {
		t.Errorf("expected stack size of 1, got %d", ctxt.stack.len())
		return
	}

	if int(ctxt.pc) != 1 {
		t.Errorf("program counter did not progress to %d, got %d", 1, ctxt.pc)
	}

	got := ctxt.stack.peek().Bytes()
	if len(got) != 2 {
		t.Errorf("expected 2 byte on the stack, got %d with values %v", len(got), got)
	}
	if got[0] != 0x12 {
		t.Errorf("expected %d for first byte, got %d", 0x12, got[0])
	}
	if got[1] != 0x34 {
		t.Errorf("expected %d for second byte, got %d", 0x34, got[1])
	}
}

func TestPush3(t *testing.T) {
	code := []Instruction{
		{opcode: PUSH2, arg: 0x1234},
		{opcode: DATA, arg: 0x5678},
	}

	ctxt := context{
		code:  code,
		stack: NewStack(),
	}

	opPush3(&ctxt)
	ctxt.pc++

	if ctxt.stack.len() != 1 {
		t.Errorf("expected stack size of 1, got %d", ctxt.stack.len())
		return
	}

	if int(ctxt.pc) != 2 {
		t.Errorf("program counter did not progress to %d, got %d", 2, ctxt.pc)
	}

	got := ctxt.stack.peek().Bytes()
	if len(got) != 3 {
		t.Errorf("expected 3 byte on the stack, got %d with values %v", len(got), got)
	}
	if got[0] != 0x12 {
		t.Errorf("expected %d for first byte, got %d", 0x12, got[0])
	}
	if got[1] != 0x34 {
		t.Errorf("expected %d for second byte, got %d", 0x34, got[1])
	}
	if got[2] != 0x56 {
		t.Errorf("expected %d for third byte, got %d", 0x56, got[2])
	}
}

func TestPush4(t *testing.T) {
	code := []Instruction{
		{opcode: PUSH2, arg: 0x1234},
		{opcode: DATA, arg: 0x5678},
	}

	ctxt := context{
		code:  code,
		stack: NewStack(),
	}

	opPush4(&ctxt)
	ctxt.pc++

	if ctxt.stack.len() != 1 {
		t.Errorf("expected stack size of 1, got %d", ctxt.stack.len())
		return
	}

	if int(ctxt.pc) != 2 {
		t.Errorf("program counter did not progress to %d, got %d", 2, ctxt.pc)
	}

	got := ctxt.stack.peek().Bytes()
	if len(got) != 4 {
		t.Errorf("expected 3 byte on the stack, got %d with values %v", len(got), got)
	}
	if got[0] != 0x12 {
		t.Errorf("expected %d for first byte, got %d", 0x12, got[0])
	}
	if got[1] != 0x34 {
		t.Errorf("expected %d for second byte, got %d", 0x34, got[1])
	}
	if got[2] != 0x56 {
		t.Errorf("expected %d for third byte, got %d", 0x56, got[2])
	}
	if got[3] != 0x78 {
		t.Errorf("expected %d for 4th byte, got %d", 0x78, got[3])
	}
}

// Subfunctions for bitwise logic, arithmetic and comparison operations.

// getTestEnvData creates context and prepares data into stack
func getTestEnvData(data []uint256.Int) *context {
	ctxt := context{
		code:  nil,
		stack: NewStack(),
	}
	for _, d := range data {
		ctxt.stack.push(&d)
	}
	return &ctxt
}

// text representation of the status
func (state Status) String() string {
	statusStr := []string{
		"RUNNING",
		"STOPPED",
		"REVERTED",
		"RETURNED",
		"SUICIDED",
		"INVALID_INSTRUCTION",
		"OUT_OF_GAS",
		"SEGMENTATION_FAULT",
		"ERROR"}
	if state < RUNNING || state > ERROR {
		return "Unknown state"
	}
	return statusStr[state]
}

// checkResult checks the result with an expectation
// status = expected status; res = expected result
func checkResult(t *testing.T, ctxt *context, status Status, res *uint256.Int) {
	if ctxt.stack.len() != 1 {
		t.Errorf("expected stack size of 1, got %d", ctxt.stack.len())
		return
	}

	if status != ctxt.status {
		t.Errorf("expected status %s, got %s", status.String(), ctxt.status.String())
	}

	got := ctxt.stack.peek()

	if !res.Eq(got) {
		t.Errorf("expected %s, got %s", res.Hex(), got.Hex())
	}
}

// runTestInstr executes the individual tests defined in testData
func runTestInstr(t *testing.T, testData []tTestDataOp) {
	for _, data := range testData {
		t.Run(data.name, func(t *testing.T) {
			ctxt := getTestEnvData(data.data)

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

			checkResult(t, ctxt, data.status, &data.res)
		})
	}
}

// TestInstr tests bitwise logical, arithmetic and comparison operations.
func TestInstr(t *testing.T) {
	// preparation of data
	// bitwise logic operations (And, Or, Not, Xor, Byte, Shl, Shr, Sar)
	testData := testDataBitwiseLogicOp
	// arithmetic operations (Add, Sub, Mul, MulMod, Div, SDiv, Mod, AddMod, SMod, Exp, SignExtend)
	testData = append(testData, testDataArithmeticOp...)
	// comparison operations (IsZero, Eq, Lt, Gt, Slt, Sgt)
	for _, dc := range testDataComparsionOp {
		d := tTestDataOp{dc.name, dc.op, dc.data, uint256.Int{0, 0, 0, 0}, dc.status, 0}
		d.res.Clear()
		if dc.res {
			d.res[0] = 1
		}
		testData = append(testData, d)
	}

	// execution of tests
	runTestInstr(t, testData)
}

// checkResultStack checks the result with an expectation
// status = expected status; res = expected result
func checkResultStack(t *testing.T, ctxt *context, status Status, res []uint256.Int) {
	if ctxt.stack.len() != len(res) {
		t.Errorf("expected stack size of %d, got %d", len(res), ctxt.stack.len())
		return
	}

	if status != ctxt.status {
		t.Errorf("expected status %s, got %s", status.String(), ctxt.status.String())
	}

	for i := len(res) - 1; i >= 0; i-- {
		r := res[i]
		got := ctxt.stack.pop()

		if !r.Eq(got) {
			t.Errorf("expected[%d] %s, got %s", i, r.Hex(), got.Hex())
		}
	}
}

func runTestStackInstr(t *testing.T, testData []tTestDataStackOp) {
	for _, data := range testData {
		t.Run(data.name, func(t *testing.T) {
			ctxt := getTestEnvData(data.data)

			data.op(ctxt, data.pos)

			checkResultStack(t, ctxt, data.status, data.res)
		})
	}
}

// exchange operations (Swap) and duplication operations (Dup)
func TestStackInstr(t *testing.T) {
	runTestStackInstr(t, testDataStackOp)
}

// Stack, Memory, Storage and Flow Operations (Pop, Mload, Mstore, Mstore8,
// Sload, Sstore, Msize, Gas)

// checkResultMem checks the result with an expectation
// status = expected status; res = expected result
func checkResultMem(t *testing.T, ctxt *context, status Status, res []uint256.Int) {
	if ctxt.stack.len() != 0 {
		t.Errorf("expected stack size of 0, got %d", ctxt.stack.len())
		return
	}

	if status != ctxt.status {
		t.Errorf("expected status %s, got %s", status.String(), ctxt.status.String())
	}

	if ctxt.status != RUNNING {
		return
	}

	for i := 0; i < len(res)/2; i++ {
		ctxt.stack.push(&res[2*i])
		opMload(ctxt)
		if status != ctxt.status {
			t.Errorf("expected status %s, got %s", status.String(), ctxt.status.String())
		} else {

			got := ctxt.stack.peek()

			if !res[2*i+1].Eq(got) {
				t.Errorf("expected %s, got %s", res[2*i+1].Hex(), got.Hex())
			}
		}
	}
}

func runTestMemInstr(t *testing.T, testData []tTestDataMemOp) {
	for _, data := range testData {
		t.Run(data.name, func(t *testing.T) {
			ctxt := getTestEnvData(data.data)
			ctxt.memory = NewMemory()

			// Create a dummy contract
			addr := vm.AccountRef{}
			ctxt.contract = vm.NewContract(addr, addr, big.NewInt(0), data.gasStore+data.gasLoad)

			for i := 0; i < len(data.data)/2; i++ {
				data.op(ctxt)
			}

			// control of the consumed gas
			expectedGas := data.gasStore
			consumedGas := data.gasStore + data.gasLoad - ctxt.contract.Gas
			if consumedGas != expectedGas {
				t.Errorf("expected consumed gas %d, got %d", expectedGas, consumedGas)
			}

			checkResultMem(t, ctxt, data.status, data.res)
		})
	}
}

// operation Mstore, Mload, Mstore8
func TestMemInstr(t *testing.T) {
	runTestMemInstr(t, testDataMemOp)
}

// operation Msize
func runTestMsizeInstr(t *testing.T, testData []tTestDataOp) {
	for _, data := range testData {
		t.Run(data.name, func(t *testing.T) {
			ctxt := getTestEnvData(data.data)
			ctxt.memory = NewMemory()

			// Create a dummy contract
			addr := vm.AccountRef{}
			ctxt.contract = vm.NewContract(addr, addr, big.NewInt(0), data.gas)

			opMstore(ctxt)
			data.op(ctxt)

			checkResult(t, ctxt, data.status, &data.res)
		})
	}
}

func TestMSizeInstruction(t *testing.T) {
	runTestMsizeInstr(t, testDataMsizeOp)
}

// push operations (Push32)

// operation Push32
func testPush32(t *testing.T, testData []tTestDataPush32) {
	for _, data := range testData {
		t.Run(data.name, func(t *testing.T) {
			ctxt := context{
				code:  data.data,
				stack: NewStack(),
			}

			data.op(&ctxt)
			ctxt.pc++

			if ctxt.stack.len() != 1 {
				t.Errorf("expected stack size of 1, got %d", ctxt.stack.len())
				return
			}

			if int(ctxt.pc) != 16 {
				t.Errorf("program counter did not progress to %d, got %d", 2, ctxt.pc)
			}

			got256 := ctxt.stack.peek()
			if got256.ByteLen() != data.res.ByteLen() {
				t.Errorf("expected %d byte on the stack, got %d with values %v", got256.ByteLen(), data.res.ByteLen(), got256)
			}

			checkResult(t, &ctxt, data.status, &data.res)
		})
	}
}

func TestPush32Instruction(t *testing.T) {
	testPush32(t, testDataPush32)
}

// Function with examples that cause a runtime error/fatal error.
// Individual examples are commented, see test data.
func TestInstrRTError(t *testing.T) {
	runTestStackInstr(t, testDataStackOpError)
	runTestMemInstr(t, testDataMemOpError)
}
