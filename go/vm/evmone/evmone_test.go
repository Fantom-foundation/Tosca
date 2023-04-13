package evmone

import (
	"encoding/hex"
	"fmt"
	"log"
	"math"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/core/vm"
)

func TestFib10(t *testing.T) {
	const arg = 10

	example := getFibExample()
	input := getFibInput(example.function, arg)

	// create a dummy contract
	addr := vm.AccountRef{}
	contract := vm.NewContract(addr, addr, big.NewInt(0), math.MaxInt64) // gas is int64, not uint64
	contract.Code = example.code

	// compute expected value
	wanted := fib(arg)

	interpreter := NewInterpreter(&vm.EVM{}, vm.Config{})

	output, err := interpreter.Run(contract, input, false)
	if err != nil {
		t.Fatalf("interpreter error: %v", err)
	}

	got, err := convertFibOutput(output)
	if err != nil {
		t.Fatal("could not convert fib output")
	}

	if got != wanted {
		t.Fatalf("unexpected result, wanted %v, got %v", wanted, got)
	}
}

func BenchmarkFib10(b *testing.B) {
	benchmarkFib(b, 10)
}

type example struct {
	code     []byte // some contract code
	function uint32 // identifier of the function in the contract to be called
}

func getFibExample() example {
	// An implementation of the fib function in EVM byte code.
	code, err := hex.DecodeString("608060405234801561001057600080fd5b506004361061002b5760003560e01c8063f9b7c7e514610030575b600080fd5b61004a600480360381019061004591906100f6565b610060565b6040516100579190610132565b60405180910390f35b600060018263ffffffff161161007957600190506100b0565b61008e600283610089919061017c565b610060565b6100a360018461009e919061017c565b610060565b6100ad91906101b4565b90505b919050565b600080fd5b600063ffffffff82169050919050565b6100d3816100ba565b81146100de57600080fd5b50565b6000813590506100f0816100ca565b92915050565b60006020828403121561010c5761010b6100b5565b5b600061011a848285016100e1565b91505092915050565b61012c816100ba565b82525050565b60006020820190506101476000830184610123565b92915050565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052601160045260246000fd5b6000610187826100ba565b9150610192836100ba565b9250828203905063ffffffff8111156101ae576101ad61014d565b5b92915050565b60006101bf826100ba565b91506101ca836100ba565b9250828201905063ffffffff8111156101e6576101e561014d565b5b9291505056fea26469706673582212207fd33e47e97ce5871bb05401e6710238af535ae8aeaab013ca9a9c29152b8a1b64736f6c637827302e382e31372d646576656c6f702e323032322e382e392b636f6d6d69742e62623161386466390058")
	if err != nil {
		log.Fatalf("Unable to decode fib-code: %v", err)
	}

	return example{
		code:     code,
		function: 0xF9B7C7E5, // function selector for the fib function
	}
}

func getFibInput(function uint32, arg int) []byte {
	// see details of argument encoding: t.ly/kBl6
	data := make([]byte, 4+32) // parameter is padded up to 32 bytes

	// encode function selector in big-endian format
	data[0] = byte(function >> 24)
	data[1] = byte(function >> 16)
	data[2] = byte(function >> 8)
	data[3] = byte(function)

	// encode argument as a big-endian value
	data[4+28] = byte(arg >> 24)
	data[5+28] = byte(arg >> 16)
	data[6+28] = byte(arg >> 8)
	data[7+28] = byte(arg)

	return data
}

func convertFibOutput(output []byte) (int, error) {
	if len(output) != 32 {
		return 0, fmt.Errorf("unexpected length of end; wanted 32, got %d", len(output))
	}
	return (int(output[28]) << 24) | (int(output[29]) << 16) | (int(output[30]) << 8) | (int(output[31]) << 0), nil
}

func fib(x int) int {
	if x <= 1 {
		return 1
	}
	return fib(x-1) + fib(x-2)
}

func benchmarkFib(b *testing.B, arg int) {
	example := getFibExample()
	input := getFibInput(example.function, arg)

	// create a dummy contract
	addr := vm.AccountRef{}
	contract := vm.NewContract(addr, addr, big.NewInt(0), math.MaxInt64) // gas is int64, not uint64
	contract.Code = example.code

	// compute expected value
	wanted := fib(arg)

	interpreter := NewInterpreter(&vm.EVM{}, vm.Config{})

	for i := 0; i < b.N; i++ {
		output, err := interpreter.Run(contract, input, false)
		if err != nil {
			b.Fatalf("interpreter failed %v", err)
		}

		got, err := convertFibOutput(output)
		if err != nil {
			b.Fatal("could not convert fib output")
		}

		if wanted != got {
			b.Fatalf("unexpected result, wanted %d, got %d", wanted, got)
		}
	}
}
