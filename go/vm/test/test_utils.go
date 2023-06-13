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
	}
)

const InitialTestGas = math.MaxInt64

type TestEVM struct {
	evm *vm.EVM
}

func GetCleanEVM(revision Revision, interpreter string, stateDB vm.StateDB) TestEVM {
	// Set hard forks for chainconfig
	chainConfig := params.ChainConfig{
		ChainID:       big.NewInt(0),
		IstanbulBlock: big.NewInt(Istanbul.GetForkBlock()),
		BerlinBlock:   big.NewInt(Berlin.GetForkBlock()),
		LondonBlock:   big.NewInt(London.GetForkBlock()),
	}

	// Hashing function used in the context for BLOCKHASH instruction
	getHash := func(num uint64) common.Hash {
		return common.Hash{}
	}

	// Create empty block context based on block number
	blockCtx := vm.BlockContext{
		BlockNumber: big.NewInt(revision.GetForkBlock() + 2),
		Time:        big.NewInt(1),
		Difficulty:  big.NewInt(1),
		GasLimit:    1 << 63,
		GetHash:     getHash,
		BaseFee:     big.NewInt(100),
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

	addr := vm.AccountRef{}
	contract := vm.NewContract(addr, addr, big.NewInt(0), InitialTestGas)
	contract.CodeAddr = &common.Address{}
	contract.Code = code
	contract.CodeHash = crypto.Keccak256Hash(code)
	contract.CallerAddress = common.Address{}

	output, err := e.GetInterpreter().Run(contract, input, false)
	return RunResult{
		Output:  output,
		GasUsed: InitialTestGas - contract.Gas,
	}, err
}

func (e *TestEVM) GetInterpreter() vm.EVMInterpreter {
	return e.evm.Interpreter()
}
