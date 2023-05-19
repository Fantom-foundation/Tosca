package vm

import (
	"crypto/sha256"
	"math"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/params"
)

var (
	variants = []string{
		"geth",
		"lfvm",
		"lfvm-si",
		"lfvm-no-sha-cache",
		"evmone",
		"evmone-basic",
		"evmone-advanced",
	}
)

func newTestEVM(r Revision) *vm.EVM {
	// Configure the block numbers for revision changes.
	chainConfig := params.AllEthashProtocolChanges
	chainConfig.BerlinBlock = big.NewInt(10)
	chainConfig.LondonBlock = big.NewInt(20)

	// Choose the block height to run.
	block := 5
	if r == Berlin {
		block = 15
	} else if r == London {
		block = 25
	}

	blockCtxt := vm.BlockContext{
		BlockNumber: big.NewInt(int64(block)),
		Time:        big.NewInt(1000),
		Difficulty:  big.NewInt(1),
		GasLimit:    1 << 63,
	}
	txCtxt := vm.TxContext{
		GasPrice: big.NewInt(1),
	}
	config := vm.Config{}
	return vm.NewEVM(blockCtxt, txCtxt, nil, chainConfig, config)
}

func runCode(interpreter vm.EVMInterpreter, code []byte, input []byte) error {
	const initialGas = math.MaxInt64

	// Create a dummy contract using the code hash as a address. This is required
	// to avoid all tests using the same cached LFVM byte code.
	addr := vm.AccountRef{}
	contract := vm.NewContract(addr, addr, big.NewInt(0), initialGas)
	contract.Code = code
	contract.CodeHash = getSha256Hash(code)

	// TODO: remove this once code caching is code-hash based.
	var codeAddr common.Address
	copy(codeAddr[:], contract.CodeHash[:])
	contract.CodeAddr = &codeAddr

	_, err := interpreter.Run(contract, input, false)
	return err
}

func getSha256Hash(code []byte) common.Hash {
	hasher := sha256.New()
	hasher.Write(code)
	var hash common.Hash
	hasher.Sum(hash[0:0])
	return hash
}
