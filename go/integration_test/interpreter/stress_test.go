// Copyright (c) 2024 Fantom Foundation
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at fantom.foundation/bsl11.
//
// Change Date: 2028-4-16
//
// On the date above, in accordance with the Business Source License, use of
// this software will be governed by the GNU Lesser General Public License v3.

package interpreter_test

import (
	"bytes"
	"context"
	"fmt"
	"testing"

	"github.com/Fantom-foundation/Tosca/go/interpreter/lfvm"
	"github.com/Fantom-foundation/Tosca/go/tosca"
	"github.com/Fantom-foundation/Tosca/go/tosca/vm"
	"golang.org/x/sync/errgroup"
)

func BenchmarkParallel_StressTests(b *testing.B) {
	codes := map[string][]byte{
		"smallTransaction": getStressBenchmark_SmallTransaction(),
		"largeCodes":       getStressBenchmark_LargeCodes(),
		"stressHashCache":  getStressBenchmark_StressHashCache(),
		"noCodeReUsage":    getStressBenchmark_NoCodeReUsage(),
	}

	for _, interpreterName := range getAllInterpreterVariantsForTests() {
		interpreter, err := tosca.NewInterpreter(interpreterName)
		if err != nil {
			b.Fatalf("failed to load %s with error: %v", interpreterName, err)
		}
		for codeName, code := range codes {
			for _, numCalls := range []int{1, 10, 100, 1000, 10000} {
				b.Run(fmt.Sprintf("%s-%s-%d", interpreterName, codeName, numCalls), func(b *testing.B) {
					hash := lfvm.Keccak256(code)
					errs, _ := errgroup.WithContext(context.Background())
					errs.SetLimit(-1) // set limit of max goroutines to unlimited

					b.ResetTimer()
					for range b.N {
						for i := 0; i < numCalls; i++ {
							errs.Go(func() error {

								localCode := bytes.Clone(code)
								localHash := hash
								if codeName == "noCodeReUsage" {
									// code has to be different for each call
									localCode = append(localCode, uint32ToBytes(uint32(i))...)
									localCode = append(localCode, byte(vm.STOP)) // stop
									localHash = lfvm.Keccak256(localCode)
								}

								params := tosca.Parameters{
									Gas:      100000,
									Input:    uint32ToBytes(uint32(i)),
									CodeHash: &localHash,
									Code:     localCode,
								}

								result, err := interpreter.Run(params)
								if err != nil || !result.Success {
									return fmt.Errorf("interpreter run failed or was not successful, err: %v", err)
								}
								return nil
							})
						}
						err = errs.Wait()
						if err != nil {
							b.Fatalf("failed to run interpreter: %v", err)
						}
					}
				})
			}
		}
	}
}

func getStressBenchmark_NoCodeReUsage() []byte {
	code := make([]byte, 6*3*1000)
	for i := 0; i < 6*3*1000; i += 6 {
		code[i] = byte(vm.PUSH1)
		code[i+1] = byte(i)
		code[i+2] = byte(vm.PUSH1)
		code[i+3] = byte(i)
		code[i+4] = byte(vm.ADD)
		code[i+5] = byte(vm.POP)
	}
	code = append(code, []byte{byte(vm.PUSH1), byte(0), byte(vm.PUSH4)}...)
	// code is extended inside call loop
	return code
}

func getStressBenchmark_SmallTransaction() []byte {
	code := []byte{
		byte(vm.PUSH1), byte(21), // push 21
		byte(vm.PUSH1), byte(21), // push 21
		byte(vm.ADD),  // add
		byte(vm.STOP), // stop
	}
	return code
}

func getStressBenchmark_LargeCodes() []byte {
	code := []byte{
		byte(vm.PUSH1), byte(0), // counter = 0
		byte(vm.JUMPDEST),       // loop start
		byte(vm.PUSH1), byte(1), // push 1
		byte(vm.ADD),              // increment counter
		byte(vm.DUP1),             // duplicate counter
		byte(vm.PUSH1), byte(255), // push 255
		byte(vm.GT),             // check if counter > 255
		byte(vm.PUSH1), byte(2), // loop start
		byte(vm.JUMPI), // loop until counter = 255
		byte(vm.STOP),  // stop
	}
	return code
}

func getStressBenchmark_StressHashCache() []byte {
	code := []byte{
		byte(vm.PUSH1), byte(0), // call data offset
		byte(vm.CALLDATALOAD),     // load index from calldata
		byte(vm.PUSH1), byte(100), // push 100
		byte(vm.MUL),              // multiply index by 100
		byte(vm.DUP1),             // duplicate index (start)
		byte(vm.PUSH1), byte(100), // push 100
		byte(vm.ADD),   // add 100 to index (upper bound)
		byte(vm.SWAP1), // swap start and end

		byte(vm.JUMPDEST),       // loop start
		byte(vm.PUSH1), byte(1), // push 1
		byte(vm.ADD), // increment counter

		byte(vm.DUP1),           // duplicate counter
		byte(vm.PUSH1), byte(0), // offset in memory
		byte(vm.MSTORE),          // store counter in memory
		byte(vm.PUSH1), byte(32), // length of hash
		byte(vm.PUSH1), byte(0), // offset in memory
		byte(vm.SHA3), // hash counter
		byte(vm.POP),  // pop hash

		byte(vm.DUP1),            // duplicate counter
		byte(vm.DUP3),            // duplicate upper bound
		byte(vm.GT),              // check if counter > upper bound
		byte(vm.PUSH1), byte(11), // loop start
		byte(vm.JUMPI), // loop until counter = index*100 + 100
		byte(vm.STOP),  // stop
	}
	return code
}

func uint32ToBytes(i uint32) []byte {
	bytes := make([]byte, 4)
	bytes[0] = byte(i)
	bytes[1] = byte(i >> 8)
	bytes[2] = byte(i >> 16)
	bytes[3] = byte(i >> 24)
	return bytes
}
