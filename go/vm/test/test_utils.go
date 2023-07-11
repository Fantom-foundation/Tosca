package vm_test

import (
	"math"
	"math/big"

	_ "github.com/Fantom-foundation/Tosca/go/vm/evmone"
	_ "github.com/Fantom-foundation/Tosca/go/vm/evmzero"
	_ "github.com/Fantom-foundation/Tosca/go/vm/lfvm"
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
		"lfvm-logging",
		"evmone",
		"evmone-basic",
		"evmone-advanced",
		"evmzero",
		"evmzero-logging",
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
		EIP150Block:   big.NewInt(0),
		EIP155Block:   big.NewInt(0),
		EIP158Block:   big.NewInt(0),
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
		GasLimit:    1 << 62,
		GetHash:     getHash,
		BaseFee:     big.NewInt(100),
		Transfer:    transferFunc,
		CanTransfer: canTransferFunc,
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

// transferFunc subtracts amount from sender and adds amount to recipient using the given Db
// Now is doing nothing as this is not changing gas computation
func transferFunc(stateDB vm.StateDB, callerAddress common.Address, to common.Address, value *big.Int) {
	// Can be something like this:
	// stateDB.SubBalance(callerAddress, value)
	// stateDB.AddBalance(to, value)
}

// canTransferFunc is the signature of a transfer function
func canTransferFunc(stateDB vm.StateDB, callerAddress common.Address, value *big.Int) bool {
	return stateDB.GetBalance(callerAddress).Cmp(value) >= 0
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
