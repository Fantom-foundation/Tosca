// Copyright (c) 2024 Fantom Foundation
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at fantom.foundation/bsl11.
//
// Change Date: 2028-4-16
//
// On the date above, in accordance with the Business Source License, use of
// this software will be governed by the GNU Lesser General Public License v3.

package ct_test

import (
	"fmt"
	"testing"

	"pgregory.net/rand"

	"github.com/Fantom-foundation/Tosca/go/ct"
	. "github.com/Fantom-foundation/Tosca/go/ct/common"
	"github.com/Fantom-foundation/Tosca/go/ct/st"
	"github.com/Fantom-foundation/Tosca/go/interpreter/geth"
	"github.com/Fantom-foundation/Tosca/go/interpreter/lfvm"
	"github.com/Fantom-foundation/Tosca/go/tosca"
)

// FuzzGeth is a fuzzing test for the geth EVM implementation
// TODO: it would be interesting to have a method to extend the corpus of the fuzzer
// with the failures found in other tests.
// So far failures will be stored in the folder: testdata/fuzz/FuzzerFunctionName/
func FuzzGeth(f *testing.F) {
	fuzzVm(geth.NewConformanceTestingTarget(), f)
}

// FuzzLfvm is a fuzzing test for lfvm
func FuzzLfvm(f *testing.F) {
	fuzzVm(lfvm.NewConformanceTestingTarget(), f)
}

// FuzzLfvm is a fuzzing test for evmzero, (issue #549 )
// func FuzzEvmzero(f *testing.F) {
// 	fuzzVm(evmzero.NewConformanceTestingTarget(), f)
// }

// FuzzDifferentialLfvmVsGeth compares state output between lfvm and geth
func FuzzDifferentialLfvmVsGeth(f *testing.F) {
	differentialFuzz(f,
		lfvm.NewConformanceTestingTarget(),
		geth.NewConformanceTestingTarget(),
	)
}

// TODO: This test makes sense but cannot be enabled yet:
// - The evmzero fails the differential test against geth. (issue #549)
// - Any other invocation of fuzzing tests in this file seem to
// invoke the all other tests in the file, and they will fail.
// func FuzzDifferentialEvmzeroVsGeth(f *testing.F) {
// 	differentialFuzz(f,
// 		evmzero.NewConformanceTestingTarget(),
// 		geth.NewConformanceTestingTarget(),
// 	)
// }

//////////////////////////////////////////////////////////////////////////////
// Fuzzing helpers

func differentialFuzz(f *testing.F, testeeVm, referenceVm ct.Evm) {

	prepareFuzzingSeeds(f)

	// Note: changing signature requires changing the prepareFuzzingSeeds function
	f.Fuzz(func(t *testing.T, opCodes []byte, gas int64, revision byte, stackBytes []byte) {

		state, err := corpusEntryToCtState(opCodes, gas, revision, stackBytes)
		if err != nil {
			t.Skip(err)
		}
		defer state.Release()

		testeeResultState, err := testeeVm.StepN(state.Clone(), 1)
		if err != nil {
			t.Fatalf("failed to run test case: %v", err)
		}
		defer testeeResultState.Release()

		referenceResultState, err := referenceVm.StepN(state.Clone(), 1)
		defer referenceResultState.Release()
		if err != nil {
			t.Fatalf("failed to run test case in reference VM: %v", err)
		}

		if testeeResultState.Status != referenceResultState.Status {
			t.Fatal("invalid result, status does not match reference status:", errorReportString(state, testeeResultState, referenceResultState))
		}

		// if result is other than running, further checks may be misleading
		// TODO: let Stopped, Returned, and Reverted be diffed as well as their results are more stable than Failed results
		// This can be done once issue #547 is solved
		if testeeResultState.Status != st.Running {
			return
		}

		if testeeResultState.Gas != referenceResultState.Gas {
			t.Fatal("invalid result, gas does not match reference gas:", errorReportString(state, testeeResultState, referenceResultState))
		}

		// Hack: lfvm does a pc transformation, but for code smaller than the required jump the pc will point to a different location
		// - Geth will point to pc+offset, whenever it is an overflow
		// - Lfvm will point min(pc+offset, len(code))
		if testeeResultState.Pc == uint16(len(opCodes)) &&
			testeeResultState.Pc != referenceResultState.Pc {
			testeeResultState.Pc = referenceResultState.Pc
		}

		if !testeeResultState.Eq(referenceResultState) {
			t.Fatal("invalid result, resulting state does not match reference state:", testeeResultState.Diff(referenceResultState), errorReportString(state, testeeResultState, referenceResultState))
		}
	})
}

func fuzzVm(testee ct.Evm, f *testing.F) {

	prepareFuzzingSeeds(f)

	// Note: changing signature requires changing the prepareFuzzingSeeds function
	f.Fuzz(func(t *testing.T, opCodes []byte, gas int64, revision byte, stackBytes []byte) {
		state, err := corpusEntryToCtState(opCodes, gas, revision, stackBytes)
		if err != nil {
			t.Skip(err)
		}

		result, err := testee.StepN(state, 1)
		if err != nil {
			t.Fatalf("failed to run test case: %v", err)
		}
		result.Release()
	})
}

const (
	fuzzIdealStackSize     = 7             // < max pops in a single instruction
	fuzzMaximumCodeSegment = 33            // < 1 instruction with 32 data bytes
	fuzzMaxGas             = 5_000_000_000 // < gas limits memory usage
)

// prepareFuzzingSeeds is a helper function to be used by similar fuzzing tests
// the arguments passed to the f.Add function needs to match the arguments
// passed to the f.Fuzz function in type, position and number
// Such types can only be of the allowed by the fuzzing engine
func prepareFuzzingSeeds(f *testing.F) {

	rnd := rand.New(0)

	// every possible revision
	for revision := MinRevision; revision <= NewestSupportedRevision; revision++ {
		// every possible opCode, even if invalid
		for op := 0x00; op <= 0xFF; op++ {
			// Some gas values: this is a hand made sampling of interesting values,
			// the fuzzer will generate more interesting values around these, the initial
			// list just sketches a region of interest around which the fuzzer will generate
			// more values. I found no measurable difference about being more accurate.
			for _, gas := range []int64{0, 1, 6, 10, 1000, fuzzMaxGas} {

				// generate a code segment with the operation followed by 6 random values
				ops := make([]byte, fuzzMaximumCodeSegment)
				_, _ = rand.Read(ops[:]) // rnd.Read never returns an error
				ops[0] = byte(op)

				// generate a stack: the stack contains a mixture of values
				// as seed for mutations
				stack := make([]byte, fuzzIdealStackSize*32)
				// fill a quarter with random values
				var i int
				for i = 0; i < 7*8; i++ {
					stack[i] = byte(rnd.Int31n(256))
				}
				// leave a quarter with zeros
				for ; i < 7*16; i++ {
					stack[i] = 0
				}
				// fill a quarter with max values
				for ; i < 7*24; i++ {
					stack[i] = 255
				}
				// fill a quarter with avg values
				for ; i < 7*32; i++ {
					stack[i] = 127
				}

				f.Add(
					ops,            // opCodes
					int64(gas),     // gas
					byte(revision), // revision
					stack,          // stack
				)
			}
		}
	}

	// add one more with a full stack
	ops := make([]byte, fuzzMaximumCodeSegment)
	_, _ = rnd.Read(ops[:]) // rnd.Read never returns an error
	ops[0] = byte(0x00)
	fullStack := make([]byte, 1024*32)
	_, _ = rnd.Read(fullStack[:]) // rnd.Read never returns an error
	f.Add(
		ops,                      // opCodes
		int64(0),                 // gas
		byte(tosca.R07_Istanbul), // revision
		fullStack,                // stack
	)
}

func corpusEntryToCtState(opCodes []byte, gas int64, revision byte, stackBytes []byte) (*st.State, error) {
	if gas < 0 {
		return nil, fmt.Errorf("negative gas %v", gas)
	}

	if gas > fuzzMaxGas {
		return nil, fmt.Errorf("gas too large %v", gas)
	}

	if tosca.Revision(revision) < MinRevision || tosca.Revision(revision) > NewestSupportedRevision {
		return nil, fmt.Errorf("unsupported revision %v", revision)
	}

	if len(opCodes) == 0 {
		return nil, fmt.Errorf("empty opCodes")
	}

	if len(opCodes) > fuzzMaximumCodeSegment {
		return nil, fmt.Errorf("too many opCodes, not interesting")
	}

	// Ignore stack sizes larger than 7 words, as they are not interesting
	// Do not ignore stack sizes close to the overflow, as they are interesting
	if len(stackBytes) > fuzzIdealStackSize*32 && len(stackBytes) < (1024-fuzzIdealStackSize)*32 {
		return nil, fmt.Errorf("Uninteresting stack size %d", len(stackBytes))
	}

	if len(stackBytes) > 1024*32 {
		return nil, fmt.Errorf("Stack too large %d", len(stackBytes))
	}

	stack := st.NewStack()
	for i := 0; i < len(stackBytes); i += 32 {
		var stackValue [32]byte
		if i+32 <= len(stackBytes) {
			copy(stackValue[:], stackBytes[i:i+32])
		} else {
			copy(stackValue[:], stackBytes[i:])
		}
		stack.Push(NewU256FromBytes(stackValue[:]...))
	}

	code := st.NewCode(opCodes)
	state := st.NewState(code)
	state.Gas = tosca.Gas(gas)
	state.Revision = tosca.Revision(revision)
	state.Stack = stack
	state.BlockContext.TimeStamp = GetForkTime(state.Revision)
	return state, nil
}

// errorReportString is a helper function to print a summary of the test states.
// Fuzzing generates a single error at the time, but when tunning the initial test
// data, such states are evaluated in parallel and errors are produced in large numbers.
// This function aims to present a compact overview of the most significant differences to
// keep process output small and readable.
func errorReportString(original *st.State, resultState *st.State, referenceState *st.State) string {
	return fmt.Sprintf(`
	--- revision: %v
	--- code(pc:%d) %v
	--- status: testee %v, reference %v
	--- gas: start %v, testee %v, reference %v
	--- stack size: start %v, testee %v, reference %v
	--- mem size: start %v, testee %v, reference %v`,
		original.Revision,
		original.Pc, original.Code.ToHumanReadableString(0, 7),
		resultState.Status, referenceState.Status,
		original.Gas, resultState.Gas, referenceState.Gas,
		original.Stack.Size(), resultState.Stack.Size(), referenceState.Stack.Size(),
		original.Memory.Size(), resultState.Memory.Size(), referenceState.Memory.Size(),
	)
}

func TestCorpusEntryToCtState(t *testing.T) {

	tests := map[string]struct {
		opCodes     []byte
		gas         int64
		revision    byte
		stackBytes  []byte
		expectedErr error
	}{
		"Valid input": {
			opCodes:     []byte{0x60, 0x80},
			gas:         1000,
			revision:    byte(tosca.R12_Shanghai),
			stackBytes:  []byte{0x01, 0x02, 0x03},
			expectedErr: nil,
		},
		"empty stack": {
			opCodes:     []byte{0x60, 0x80},
			gas:         1000,
			revision:    byte(tosca.R12_Shanghai),
			stackBytes:  []byte{},
			expectedErr: nil,
		},
		"zero gas": {
			opCodes:     []byte{0x60, 0x80},
			gas:         0,
			revision:    byte(tosca.R12_Shanghai),
			stackBytes:  []byte{0x01, 0x02, 0x03},
			expectedErr: nil,
		},
		"Negative gas": {
			opCodes:     []byte{0x60, 0x80},
			gas:         -1000,
			revision:    byte(tosca.R12_Shanghai),
			stackBytes:  []byte{0x01, 0x02, 0x03},
			expectedErr: fmt.Errorf("negative gas -1000"),
		},
		"Unsupported revision": {
			opCodes:     []byte{0x60, 0x80},
			gas:         1000,
			revision:    0x10,
			stackBytes:  []byte{0x01, 0x02, 0x03},
			expectedErr: fmt.Errorf("unsupported revision 16"),
		},
		"Empty opCodes": {
			opCodes:     []byte{},
			gas:         1000,
			revision:    byte(tosca.R12_Shanghai),
			stackBytes:  []byte{0x01, 0x02, 0x03},
			expectedErr: fmt.Errorf("empty opCodes"),
		},
		"Too many opCodes": {
			opCodes:     make([]byte, 34),
			gas:         1000,
			revision:    byte(tosca.R12_Shanghai),
			stackBytes:  []byte{0x01, 0x02, 0x03},
			expectedErr: fmt.Errorf("too many opCodes, not interesting"),
		},
		"Uninteresting stack size": {
			opCodes:     []byte{0x60, 0x80},
			gas:         1000,
			revision:    byte(tosca.R12_Shanghai),
			stackBytes:  make([]byte, 8*32),
			expectedErr: fmt.Errorf("Uninteresting stack size 256"),
		},
		"Stack too large": {
			opCodes:     []byte{0x60, 0x80},
			gas:         1000,
			revision:    byte(tosca.R12_Shanghai),
			stackBytes:  make([]byte, 1025*32),
			expectedErr: fmt.Errorf("Stack too large 32800"),
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			state, err := corpusEntryToCtState(tt.opCodes, tt.gas, tt.revision, tt.stackBytes)
			if err != nil {
				if err.Error() != tt.expectedErr.Error() {
					t.Errorf("Unexpected error. Got: %v, Want: %v", err, tt.expectedErr)
				}
				return
			}

			if state.Code == nil {
				t.Errorf("Unexpected nil code")
			}

			if state.Code.Length() != len(tt.opCodes) {
				t.Errorf("Unexpected code size. Got: %v, Want: %v", state.Code.Length(), len(tt.opCodes))
			}

			if state.Revision != tosca.Revision(tt.revision) {
				t.Errorf("Unexpected revision. Got: %v, Want: %v", state.Revision, tosca.Revision(tt.revision))
			}

			if state.Gas != tosca.Gas(tt.gas) {
				t.Errorf("Unexpected gas. Got: %v, Want: %v", state.Gas, tosca.Gas(tt.gas))
			}

			if state.Stack == nil {
				t.Errorf("Unexpected nil stack")
			}

			stateStackSize := (len(tt.stackBytes) + 31) / 32
			if state.Stack.Size() != stateStackSize {
				t.Errorf("Unexpected stack size. Got: %v, Want: %v", state.Stack.Size(), stateStackSize)
			}
		})
	}
}
