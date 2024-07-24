// Copyright (c) 2024 Fantom Foundation
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at fantom.foundation/bsl11.
//
// Change Date: 2028-4-16
//
// On the date above, in accordance with the Business Source License, use of
// this software will be governed by the GNU Lesser General Public License v3.

package examples

import (
	"fmt"
	"math"

	"github.com/Fantom-foundation/Tosca/go/tosca"
	"golang.org/x/crypto/sha3"
)

// Example is an executable description of a contract and an entry point with a (int)->int signature.
type Example struct {
	exampleSpec
	codeHash tosca.Hash // the hash of the code
}

// exampleSpec specifies a contract and an entry point with a (int)->int signature.
type exampleSpec struct {
	Name      string
	Code      []byte        // some contract code
	function  uint32        // identifier of the function in the contract to be called
	reference func(int) int // a reference function computing the same function
}

func (s exampleSpec) build() Example {
	hasher := sha3.NewLegacyKeccak256()
	hasher.Write(s.Code)
	var hash tosca.Hash
	hasher.Sum(hash[0:0])
	return Example{
		exampleSpec: s,
		codeHash:    hash,
	}
}

type Result struct {
	Result  int
	UsedGas int64
}

// RunOn runs this example on the given interpreter, using the given argument.
func (e *Example) RunOn(interpreter tosca.Interpreter, argument int) (Result, error) {

	const initialGas = math.MaxInt64
	params := tosca.Parameters{
		Code:     e.Code,
		CodeHash: (*tosca.Hash)(&e.codeHash),
		Input:    encodeArgument(e.function, argument),
		Gas:      initialGas,
	}

	res, err := interpreter.Run(params)
	if err != nil {
		return Result{}, err
	}

	result, err := decodeOutput(res.Output)
	if err != nil {
		return Result{}, err
	}
	return Result{
		Result:  result,
		UsedGas: initialGas - int64(res.GasLeft),
	}, nil
}

// RunOnProcessor runs this example on the given processor, using the given argument.
func (e *Example) RunOnProcessor(processor tosca.Processor, argument int,
	transaction tosca.Transaction, transactionContext tosca.TransactionContext) (Result, error) {

	blockParameters := tosca.BlockParameters{}
	transaction.Input = encodeArgument(e.function, argument)

	receipt, err := processor.Run(blockParameters, transaction, transactionContext)
	if err != nil {
		return Result{}, err
	}

	if !receipt.Success {
		return Result{}, fmt.Errorf("contract execution failed")
	}

	result, err := decodeOutput(receipt.Output)
	if err != nil {
		return Result{}, err
	}
	return Result{
		Result:  result,
		UsedGas: int64(receipt.GasUsed),
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
