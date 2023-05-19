package vm_test

import (
	"errors"
	"math/big"
	"reflect"
	"strings"
	"testing"

	bridge "github.com/Fantom-foundation/Tosca/go/common"
	"github.com/Fantom-foundation/Tosca/go/vm/lfvm"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"
)

const (
	MAX_STACK_SIZE int    = 1024
	GAS_START      uint64 = 1 << 32
)

var (

	// TODO: Duplicated for now with variants from  vm_test.go not to have too much errors for c++ version
	interpreterVariants = []string{
		"geth",
		"lfvm",
		"lfvm-si",
		"lfvm-no-code-cache",
		//"evmone",
		//"evmone-basic",
		//"evmone-advanced",
	}

	HASH_0    = common.Hash{0}
	HASH_1    = common.BigToHash(big.NewInt(1))
	HASH_2    = common.BigToHash(big.NewInt(2))
	ADDRESS_0 = common.Address{0}

	ErrStackUnderflow = errors.New("stack underflow")
	ErrStackOverflow  = errors.New("stack overflow converted")
	ErrInvalidOpCode  = errors.New("invalid instruction")
)

type OpcodeTest struct {
	name        string
	code        []vm.OpCode
	stackPtrPos int
	err         error
	blockNumber uint64
	mockCalls   func(mockStateDB *lfvm.MockStateDB)
	gasStart    uint64
	gasConsumed uint64
}

func TestStackMaxBoundry(t *testing.T) {

	// For every variant of interpreter
	for _, variant := range interpreterVariants {

		// Add tests for execution
		for _, test := range addFullStackFailOpCodes(nil) {
			t.Run(variant+"/"+test.name, func(t *testing.T) {
				var stateDB vm.StateDB
				evm := GetCleanEVM(LatestRevision, variant, stateDB)

				// Fill stack with PUSH1 instruction
				code := make([]byte, test.stackPtrPos*2+1)
				for i := 0; i < test.stackPtrPos*2-1; i += 2 {
					code[i] = byte(vm.PUSH1)
					code[i+1] = byte(1)
				}

				// Set a tested instruction as last one
				code[test.stackPtrPos*2] = byte(test.code[0])

				// Run an interpreter
				result, err := evm.Run(code, []byte{})
				err = convertError(evm.GetInterpreter(), err)

				// Check the result.
				if err != ErrStackOverflow {
					t.Errorf("execution failed %v should end with stack overflow: status is %v", test.name, err)
				}
				if result.GasUsed != test.gasConsumed {
					t.Errorf("execution failed %v wrong gas: used %v, wanted %v", test.name, result.GasUsed, test.gasConsumed)
				}
			})
		}
	}
}

func TestStackMinBoundry(t *testing.T) {

	// For every variant of interpreter
	for _, variant := range interpreterVariants {

		// Add tests for execution
		for _, test := range addEmptyStackFailOpCodes(nil) {
			t.Run(variant+"/"+test.name, func(t *testing.T) {
				var stateDB *lfvm.MockStateDB

				evm := GetCleanEVM(LatestRevision, variant, stateDB)

				// Execute only solo instruction with empty stack
				code := []byte{byte(test.code[0])}

				// Run an interpreter
				result, err := evm.Run(code, []byte{})
				err = convertError(evm.GetInterpreter(), err)

				// Check the result.
				if err != ErrStackUnderflow {
					t.Errorf("execution failed %v should end with stack overflow: status is %v", test.name, err)
				}
				if result.GasUsed != test.gasConsumed {
					t.Errorf("execution failed %v wrong gas: used %v, wanted %v", test.name, result.GasUsed, test.gasConsumed)
				}
			})
		}
	}
}

var fullStackFailOpCodes = []vm.OpCode{
	vm.MSIZE, vm.ADDRESS, vm.ORIGIN, vm.CALLER, vm.CALLVALUE, vm.CALLDATASIZE,
	vm.CODESIZE, vm.GASPRICE, vm.COINBASE, vm.TIMESTAMP, vm.NUMBER,
	vm.DIFFICULTY, vm.GASLIMIT, vm.PC, vm.GAS, vm.RETURNDATASIZE,
	vm.SELFBALANCE, vm.CHAINID, vm.BASEFEE,
	// TODO: Superinstructions to be covered
	//PUSH1_PUSH1_PUSH1_SHL_SUB,
	//PUSH1_DUP1, PUSH1_PUSH1, PUSH1_PUSH4_DUP3,
}

var emptyStackFailOpCodes = []vm.OpCode{
	vm.POP, vm.ADD, vm.SUB, vm.MUL, vm.DIV, vm.SDIV, vm.MOD, vm.SMOD, vm.EXP, vm.SIGNEXTEND,
	vm.SHA3, vm.LT, vm.GT, vm.SLT, vm.SGT, vm.EQ, vm.AND, vm.XOR, vm.OR, vm.BYTE,
	vm.SHL, vm.SHR, vm.SAR, vm.ADDMOD, vm.MULMOD, vm.ISZERO, vm.NOT, vm.BALANCE, vm.CALLDATALOAD, vm.EXTCODESIZE,
	vm.BLOCKHASH, vm.MLOAD, vm.SLOAD, vm.EXTCODEHASH, vm.JUMP, vm.SELFDESTRUCT,
	vm.MSTORE, vm.MSTORE8, vm.SSTORE, vm.JUMPI, vm.RETURN, vm.REVERT,
	vm.CALLDATACOPY, vm.CODECOPY, vm.RETURNDATACOPY,
	vm.EXTCODECOPY, vm.CREATE, vm.CREATE2, vm.CALL, vm.CALLCODE,
	vm.STATICCALL, vm.DELEGATECALL,
	// TODO: Superinstructions to be covered
	//POP_POP, POP_JUMP, SWAP2_POP, PUSH1_ADD, PUSH1_SHL,
	//SWAP2_SWAP1_POP_JUMP, PUSH2_JUMPI, ISZERO_PUSH2_JUMPI, SWAP2_SWAP1,
	//DUP2_LT, SWAP1_POP_SWAP2_SWAP1, POP_SWAP2_SWAP1_POP,
	//AND_SWAP1_POP_SWAP2_SWAP1, SWAP1_POP, DUP2_MSTORE,
}

func addEmptyStackFailOpCodes(tests []OpcodeTest) []OpcodeTest {
	var addedTests []OpcodeTest
	addedTests = append(addedTests, tests...)
	var opCodes []vm.OpCode
	opCodes = append(opCodes, emptyStackFailOpCodes...)
	opCodes = append(opCodes, getOpcodes(vm.DUP1, vm.DUP16)...)
	opCodes = append(opCodes, getOpcodes(vm.SWAP1, vm.SWAP16)...)
	opCodes = append(opCodes, getOpcodes(vm.LOG0, vm.LOG4)...)
	for _, opCode := range opCodes {
		addedTests = append(addedTests, OpcodeTest{opCode.String(), []vm.OpCode{opCode}, 0, ErrStackUnderflow, uint64(London.GetForkBlock()) + 2, nil, GAS_START, 0})
	}
	return addedTests
}

func addFullStackFailOpCodes(tests []OpcodeTest) []OpcodeTest {
	var addedTests []OpcodeTest
	addedTests = append(addedTests, tests...)
	var opCodes []vm.OpCode
	opCodes = append(opCodes, fullStackFailOpCodes...)
	opCodes = append(opCodes, getOpcodes(vm.PUSH1, vm.PUSH32)...)
	opCodes = append(opCodes, getOpcodes(vm.DUP1, vm.DUP16)...)
	for _, opCode := range opCodes {
		// Consumed gas here is 3*1024=3072 as there will be 1024 x PUSH1 instruction to fill stack,
		// where static gas for one PUSH1 instruction is 3 and stack length is 1024
		addedTests = append(addedTests, OpcodeTest{opCode.String(), []vm.OpCode{opCode}, MAX_STACK_SIZE, ErrStackOverflow, uint64(London.GetForkBlock()) + 2, nil, GAS_START, 3072})
	}
	return addedTests
}

func getOpcodes(start vm.OpCode, end vm.OpCode) (opCodes []vm.OpCode) {
	for i := start; i <= end; i++ {
		opCodes = append(opCodes, vm.OpCode(i))
	}
	return
}

func convertError(interpreter vm.EVMInterpreter, err error) error {
	switch interpreter.(type) {
	case *vm.GethEVMInterpreter, *lfvm.EVMInterpreter:
		switch err.(type) {
		case *vm.ErrStackUnderflow:
			return ErrStackUnderflow
		case *vm.ErrStackOverflow:
			return ErrStackOverflow
		case *vm.ErrInvalidOpCode:
			return ErrInvalidOpCode
		}
	case *bridge.EvmcInterpreter:
		if strings.Contains(reflect.TypeOf(err).String(), "vm.") {
			return err
		}
		return errors.New("not implemented error translation for: " + err.Error())
	}
	return err
}
