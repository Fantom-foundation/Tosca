// Copyright (c) 2024 Fantom Foundation
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at fantom.foundation/bsl11.
//
// Change Date: 2028-4-16
//
// On the date above, in accordance with the Business Source License, use of
// this software will be governed by the GNU Lesser General Public License v3.

package evmrs

import (
	"bytes"
	"testing"

	"github.com/Fantom-foundation/Tosca/go/ct/st"
	"github.com/Fantom-foundation/Tosca/go/examples"
	"github.com/Fantom-foundation/Tosca/go/tosca"
	"github.com/Fantom-foundation/Tosca/go/tosca/vm"
)

func TestFib10(t *testing.T) {
	const arg = 10

	example := examples.GetFibExample()

	// compute expected value
	wanted := example.RunReference(arg)

	interpreter := tosca.GetInterpreter("evmrs")
	got, err := example.RunOn(interpreter, arg)
	if err != nil {
		t.Fatalf("running the fib example failed: %v", err)
	}

	if got.Result != wanted {
		t.Fatalf("unexpected result, wanted %v, got %v", wanted, got.Result)
	}
}

func TestEvmrs_DumpProfile(t *testing.T) {
	example := examples.GetFibExample()
	interpreter, ok := tosca.GetInterpreter("evmrs-profiling").(tosca.ProfilingInterpreter)
	if !ok || interpreter == nil {
		t.Fatalf("profiling evmrs configuration does not support profiling")
	}
	for i := 0; i < 10; i++ {
		_, err := example.RunOn(interpreter, 10)
		if err != nil {
			t.Fatalf("running the fib example failed: %v", err)
		}
		interpreter.DumpProfile()
		if i == 5 {
			interpreter.ResetProfile()
		}
	}
}

func BenchmarkNewEvmcInterpreter(b *testing.B) {
	b.Run("evmrs", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			tosca.GetInterpreter("evmrs")
		}
	})
}

func BenchmarkFib10(b *testing.B) {
	benchmarkFib(b, 10)
}

func benchmarkFib(b *testing.B, arg int) {
	example := examples.GetFibExample()

	// compute expected value
	wanted := example.RunReference(arg)

	b.Run("evmrs", func(b *testing.B) {
		interpreter := tosca.GetInterpreter("evmrs")
		for i := 0; i < b.N; i++ {
			got, err := example.RunOn(interpreter, arg)
			if err != nil {
				b.Fatalf("running the fib example failed: %v", err)
			}

			if wanted != got.Result {
				b.Fatalf("unexpected result, wanted %d, got %d", wanted, got.Result)
			}
		}
	})
}

func TestEvmcInterpreter_BlobHashCanBeRead(t *testing.T) {

	// create a test state with a code push at index 0
	code := []byte{
		byte(vm.PUSH1), 0, // add to stack index to read from blobhash
		byte(vm.BLOBHASH), // read from blobhash index 0 and push it into stack
		byte(vm.PUSH1), 0, // push to stack offset to write in memory
		byte(vm.MSTORE),    // write in memory offset 0 value returned from blobhash
		byte(vm.PUSH1), 32, // push size of hash to read
		byte(vm.PUSH1), 0, // push to stack offset to read from memory
		byte(vm.RETURN),
	}

	params := tosca.Parameters{
		Gas: 20000,
		BlockParameters: tosca.BlockParameters{
			Revision: tosca.R13_Cancun,
		},
		TransactionParameters: tosca.TransactionParameters{
			BlobHashes: []tosca.Hash{{2}},
		},
		Code: code,
	}

	evm := tosca.GetInterpreter("evmrs")
	if evm == nil {
		t.Fatalf("failed to locate evmrs")
	}

	result, err := evm.Run(params)
	if err != nil {
		t.Fatalf("failed to run evmrs interpreter: %v", err)
	}
	if result.Output == nil {
		t.Fatalf("expected output, got nothing")
	}
	if !bytes.Equal(result.Output, params.BlobHashes[0][:]) {
		t.Errorf("unexpected output, wanted %v, got %v", params.BlobHashes[0], result.Output)
	}
}

func TestEvmcSteppableInterpreter_BlobHashCanBeRead(t *testing.T) {

	code := []byte{
		byte(vm.PUSH1), 0, // add to stack index to read from blobhash
		byte(vm.BLOBHASH), // read from blobhash index 0 and push it into stack
		byte(vm.PUSH1), 0, // push to stack offset to write in memory
		byte(vm.MSTORE),    // write in memory offset 0 value returned from blobhash
		byte(vm.PUSH1), 32, // push size of hash to read
		byte(vm.PUSH1), 0, // push to stack offset to read from memory
		byte(vm.RETURN),
	}

	blobhashes := []tosca.Hash{{2}}

	inputState := st.NewState(st.NewCode(code))
	inputState.Gas = 20000
	inputState.Revision = tosca.R13_Cancun
	inputState.TransactionContext = st.NewTransactionContext()
	inputState.TransactionContext.BlobHashes = blobhashes

	evm := NewConformanceTestingTarget()
	if evm == nil {
		t.Fatalf("failed to locate evmrs")
	}

	resultState, err := evm.StepN(inputState, 8)
	if err != nil {
		t.Fatalf("failed to run evmrs interpreter: %v", err)
	}
	if resultState.TransactionContext.BlobHashes == nil {
		t.Fatalf("expected output, got nothing")
	}
	if !bytes.Equal(resultState.ReturnData.ToBytes(), blobhashes[0][:]) {
		t.Errorf("unexpected output, wanted %v, got %v", blobhashes[0], resultState.ReturnData)
	}
}
