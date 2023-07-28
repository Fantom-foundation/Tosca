package examples

import (
	"fmt"
	"math"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
)

// Example is an executable description of a contract and an entry point with a (int)->int signature.
type Example struct {
	exampleSpec
	codeHash common.Hash // the hash of the code
}

// exampleSpec specifies a contract and an entry point with a (int)->int signature.
type exampleSpec struct {
	Name      string
	code      []byte        // some contract code
	function  uint32        // identifier of the function in the contract to be called
	reference func(int) int // a reference function computing the same function
}

func (s exampleSpec) build() Example {
	return Example{
		exampleSpec: s,
		codeHash:    crypto.Keccak256Hash(s.code),
	}
}

type Result struct {
	Result  int
	UsedGas int64
}

// RunOn runs this example on the given interpreter, using the given argument.
func (e *Example) RunOn(interpreter vm.EVMInterpreter, argument int) (Result, error) {
	const initialGas = math.MaxInt64
	input := encodeArgument(e.function, argument)

	// create a dummy contract
	addr := vm.AccountRef{}
	contract := vm.NewContract(addr, addr, big.NewInt(0), initialGas)
	contract.Code = e.code
	contract.CodeHash = e.codeHash
	contract.CodeAddr = &common.Address{}

	output, err := interpreter.Run(contract, input, false)
	if err != nil {
		return Result{}, err
	}

	result, err := decodeOutput(output)
	if err != nil {
		return Result{}, err
	}
	return Result{
		Result:  result,
		UsedGas: initialGas - int64(contract.Gas),
	}, nil
}

// RunRef runs the reference function of this example to produce the expected result.
func (e *Example) RunReference(argument int) int {
	return e.reference(argument)
}

func encodeArgument(function uint32, arg int) []byte {
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

func decodeOutput(output []byte) (int, error) {
	if len(output) != 32 {
		return 0, fmt.Errorf("unexpected length of output; wanted 32, got %d", len(output))
	}
	return (int(output[28]) << 24) | (int(output[29]) << 16) | (int(output[30]) << 8) | (int(output[31]) << 0), nil
}
