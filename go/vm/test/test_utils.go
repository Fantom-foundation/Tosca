package vm_test

import (
	"math"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
)

var (
	Variants = []string{
		"geth",
		"lfvm",
		"lfvm-si",
		"lfvm-no-sha-cache",
		"lfvm-no-code-cache",
		"evmone",
		"evmone-basic",
		"evmone-advanced",
		// "evmzero", TODO: evmzero is not yet in a state where these tests succeed!
	}
)

type TestEVM struct {
	evm *vm.EVM
}

func GetCleanEVM(revision Revision, interpreter string, stateDB vm.StateDB) TestEVM {
	// Set hard forks for chainconfig
	chainConfig := params.ChainConfig{
		IstanbulBlock: big.NewInt(Istanbul.GetForkBlock()),
		BerlinBlock:   big.NewInt(Berlin.GetForkBlock()),
		LondonBlock:   big.NewInt(London.GetForkBlock()),
	}
	// Create empty block context based on block number
	blockCtx := vm.BlockContext{
		BlockNumber: big.NewInt(revision.GetForkBlock() + 2),
		Time:        big.NewInt(1),
		Difficulty:  big.NewInt(1),
		GasLimit:    1 << 63,
	}
	// Create empty tx context
	txCtx := vm.TxContext{
		GasPrice: big.NewInt(1),
	}
	// Set interpreter variant for this VM
	config := vm.Config{
		InterpreterImpl: interpreter,
	}
	return TestEVM{vm.NewEVM(blockCtx, txCtx, stateDB, &chainConfig, config)}
}

type RunResult struct {
	Output  []byte
	GasUsed uint64
}

func (e *TestEVM) Run(code []byte, input []byte) (RunResult, error) {
	const initialGas = math.MaxInt64

	addr := vm.AccountRef{}
	contract := vm.NewContract(addr, addr, big.NewInt(0), initialGas)
	contract.CodeAddr = &common.Address{}
	contract.Code = code
	contract.CodeHash = crypto.Keccak256Hash(code)

	output, err := e.GetInterpreter().Run(contract, input, false)
	return RunResult{
		Output:  output,
		GasUsed: math.MaxInt64 - contract.Gas,
	}, err
}

func (e *TestEVM) GetInterpreter() vm.EVMInterpreter {
	return e.evm.Interpreter()
}
