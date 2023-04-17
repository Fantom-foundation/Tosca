package evmone

import (
	"encoding/hex"
	"log"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/core/vm"
)

func TestLoadingEvmone(t *testing.T) {
	NewInterpreter(&vm.EVM{}, vm.Config{})
}

type example struct {
	code     []byte // Some contract code
	function uint32 // The identifier of the function in the contract to be called
}

func getFibExample() example {
	// An implementation of the fib function in EVM byte code.
	code, err := hex.DecodeString("608060405234801561001057600080fd5b506004361061002b5760003560e01c8063f9b7c7e514610030575b600080fd5b61004a600480360381019061004591906100f6565b610060565b6040516100579190610132565b60405180910390f35b600060018263ffffffff161161007957600190506100b0565b61008e600283610089919061017c565b610060565b6100a360018461009e919061017c565b610060565b6100ad91906101b4565b90505b919050565b600080fd5b600063ffffffff82169050919050565b6100d3816100ba565b81146100de57600080fd5b50565b6000813590506100f0816100ca565b92915050565b60006020828403121561010c5761010b6100b5565b5b600061011a848285016100e1565b91505092915050565b61012c816100ba565b82525050565b60006020820190506101476000830184610123565b92915050565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052601160045260246000fd5b6000610187826100ba565b9150610192836100ba565b9250828203905063ffffffff8111156101ae576101ad61014d565b5b92915050565b60006101bf826100ba565b91506101ca836100ba565b9250828201905063ffffffff8111156101e6576101e561014d565b5b9291505056fea26469706673582212207fd33e47e97ce5871bb05401e6710238af535ae8aeaab013ca9a9c29152b8a1b64736f6c637827302e382e31372d646576656c6f702e323032322e382e392b636f6d6d69742e62623161386466390058")
	if err != nil {
		log.Fatalf("Unable to decode fib-code: %v", err)
	}

	return example{
		code:     code,
		function: 0xF9B7C7E5, // The function selector for the fib function.
	}
}

func fib(x int) int {
	if x <= 1 {
		return 1
	}
	return fib(x-1) + fib(x-2)
}

func benchmarkFib(b *testing.B, arg int) {
	example := getFibExample()

	// Create input data.

	// See details of argument encoding: t.ly/kBl6
	data := make([]byte, 4+32) // < the parameter is padded up to 32 bytes

	// Encode function selector in big-endian format.
	data[0] = byte(example.function >> 24)
	data[1] = byte(example.function >> 16)
	data[2] = byte(example.function >> 8)
	data[3] = byte(example.function)

	// Encode argument as a big-endian value.
	data[4+28] = byte(arg >> 24)
	data[5+28] = byte(arg >> 16)
	data[6+28] = byte(arg >> 8)
	data[7+28] = byte(arg)

	// Create a dummy contract
	addr := vm.AccountRef{}
	contract := vm.NewContract(addr, addr, big.NewInt(0), 1<<62) // Gas is int64, not uint64
	contract.Code = example.code

	// Create execution context.
	// ctxt := context{
	// 	code:     converted,
	// 	data:     data,
	// 	callsize: *uint256.NewInt(uint64(len(data))),
	// 	stack:    NewStack(),
	// 	memory:   NewMemory(),
	// 	readOnly: true,
	// 	contract: contract,
	// }

	// Compute expected value.
	wanted := fib(arg)

	// blockCtx := vm.BlockContext{
	// 	GasLimit: math.MaxUint64,
	// }
	// txCtx := vm.TxContext{}
	// chainConfig := params.ChainConfig{}
	// config := vm.Config{}

	// vm.NewEVM(blockCtx, txCtx, statedb, chainConfig, config)

	evm := vm.EVM{
		// blockCtx: vm.BlockContext{ GasLimit: math.MaxUint64 },
	}

	cfg := vm.Config{}

	interpreter := NewInterpreter(&evm, cfg)

	for i := 0; i < b.N; i++ {

		output, err := interpreter.Run(contract, data, false)
		if err != nil {
			b.Fatalf("interpreter failed %s", err)
		}

		if len(output) != 32 {
			b.Fatalf("unexpected length of end; wanted 32, got %d", len(output))
		}

		// res := make([]byte, len(output))

		got := (int(output[28]) << 24) | (int(output[29]) << 16) | (int(output[30]) << 8) | (int(output[31]) << 0)
		if wanted != got {
			b.Fatalf("unexpected result, wanted %d, got %d", wanted, got)
		}

		// Reset the context.
		// ctxt.pc = 0
		// ctxt.status = RUNNING
		// ctxt.contract.Gas = 1 << 31
		// ctxt.stack.stack_ptr = 0

		// // Run the code (actual benchmark).
		// run(&ctxt)

		// // Check the result.
		// if ctxt.status != RETURNED {
		// 	b.Fatalf("execution failed: status is %v, error %v", ctxt.status, ctxt.err)
		// }

		// size := ctxt.result_size
		// if size.Uint64() != 32 {
		// 	b.Fatalf("unexpected length of end; wanted 32, got %d", size.Uint64())
		// }
		// res := make([]byte, size.Uint64())
		// offset := ctxt.result_offset
		// ctxt.memory.CopyData(offset.Uint64(), res)

		// got := (int(res[28]) << 24) | (int(res[29]) << 16) | (int(res[30]) << 8) | (int(res[31]) << 0)
		// if wanted != got {
		// 	b.Fatalf("unexpected result, wanted %d, got %d", wanted, got)
		// }
	}
}

func BenchmarkFib10(b *testing.B) {
	benchmarkFib(b, 10)
}
