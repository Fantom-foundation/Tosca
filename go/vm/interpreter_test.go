package vm

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
	"github.com/ethereum/go-ethereum/params"
)

const (
	MAX_STACK_SIZE int    = 1024
	GAS_START      uint64 = 1 << 32

	// Chain config for hardforks
	ISTANBUL_FORK = 1
	BERLIN_FORK   = 10
	LONDON_FORK   = 20

	// Block numbers for different fork version
	ISTANBUL_BLOCK = 2
	BERLIN_BLOCK   = 12
	LONDON_BLOCK   = 22
)

var (

	// TODO: Duplicated for now with variants from  vm_test.go not to have too much errors for c++ version
	interpreterVariants = []string{
		"geth",
		"lfvm",
		"lfvm-si",
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

func getCleanEVM(blockNR uint64, interpreter string, stateDB vm.StateDB) *vm.EVM {
	// Create empty block context based on block number
	blockCtx := vm.BlockContext{
		BlockNumber: big.NewInt(int64(blockNR)),
		Time:        big.NewInt(1),
		Difficulty:  big.NewInt(1),
		GasLimit:    1 << 63,
	}
	// Create empty tx context
	txCtx := vm.TxContext{
		GasPrice: big.NewInt(1),
	}
	// Set hard forks for chainconfig
	chainConfig := params.ChainConfig{
		IstanbulBlock: big.NewInt(ISTANBUL_FORK),
		BerlinBlock:   big.NewInt(BERLIN_FORK),
		LondonBlock:   big.NewInt(LONDON_FORK),
	}
	// Set interpreter variant for this VM
	config := vm.Config{
		InterpreterImpl: interpreter,
	}

	return vm.NewEVM(blockCtx, txCtx, stateDB, &chainConfig, config)
}

func TestStackMaxBoundry(t *testing.T) {

	// For every variant of interpreter
	for _, variant := range interpreterVariants {

		// Add tests for execution
		for _, test := range addFullStackFailOpCodes(nil) {
			t.Run(variant+"/"+test.name, func(t *testing.T) {
				var stateDB *lfvm.MockStateDB

				evm := getCleanEVM(test.blockNumber, variant, stateDB)
				addr := vm.AccountRef{}
				contract := vm.NewContract(addr, addr, big.NewInt(0), test.gasStart)
				contract.CodeAddr = &common.Address{}

				// Fill stack with PUSH1 instruction
				code := make([]byte, test.stackPtrPos*2+1)
				for i := 0; i < test.stackPtrPos*2-1; i += 2 {
					code[i] = byte(vm.PUSH1)
					code[i+1] = byte(1)
				}

				// Set a tested instruction as last one
				code[test.stackPtrPos*2] = byte(test.code[0])
				contract.Code = code
				gas := contract.Gas

				// Run an interpreter
				_, err := evm.Interpreter().Run(contract, []byte{}, false)
				gas -= contract.Gas
				err = convertError(evm.Interpreter(), err)

				// Check the result.
				if err != ErrStackOverflow {
					t.Errorf("execution failed %v should end with stack overflow: status is %v", test.name, err)
				}
				if gas != test.gasConsumed {
					t.Errorf("execution failed %v wrong gas: status is %v, wanted %v", test.name, gas, test.gasConsumed)
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

				evm := getCleanEVM(test.blockNumber, variant, stateDB)
				addr := vm.AccountRef{}
				contract := vm.NewContract(addr, addr, big.NewInt(0), test.gasStart)
				contract.CodeAddr = &common.Address{}

				code := make([]byte, test.stackPtrPos*2+1)

				// Set a tested instruction as last one
				code[test.stackPtrPos*2] = byte(test.code[0])
				contract.Code = code
				gas := contract.Gas

				// Run an interpreter
				_, err := evm.Interpreter().Run(contract, []byte{}, false)
				gas -= contract.Gas
				err = convertError(evm.Interpreter(), err)

				// Check the result.
				if err != ErrStackUnderflow {
					t.Errorf("execution failed %v should end with stack overflow: status is %v", test.name, err)
				}
				if gas != test.gasConsumed {
					t.Errorf("execution failed %v wrong gas: status is %v, wanted %v", test.name, gas, test.gasConsumed)
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
		addedTests = append(addedTests, OpcodeTest{opCode.String(), []vm.OpCode{opCode}, 0, ErrStackUnderflow, LONDON_BLOCK, nil, GAS_START, 0})
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
		addedTests = append(addedTests, OpcodeTest{opCode.String(), []vm.OpCode{opCode}, MAX_STACK_SIZE, ErrStackOverflow, LONDON_BLOCK, nil, GAS_START, 3072})
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
