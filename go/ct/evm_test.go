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
	"math"
	"testing"

	"pgregory.net/rand"

	"github.com/Fantom-foundation/Tosca/go/ct"
	. "github.com/Fantom-foundation/Tosca/go/ct/common"
	"github.com/Fantom-foundation/Tosca/go/ct/gen"
	. "github.com/Fantom-foundation/Tosca/go/ct/rlz"
	"github.com/Fantom-foundation/Tosca/go/ct/spc"
	"github.com/Fantom-foundation/Tosca/go/ct/st"
	"github.com/Fantom-foundation/Tosca/go/vm"
	"github.com/Fantom-foundation/Tosca/go/vm/evmzero"
	"github.com/Fantom-foundation/Tosca/go/vm/geth"
	"github.com/Fantom-foundation/Tosca/go/vm/lfvm"
)

var evms = map[string]ct.Evm{
	// "geth":    vm.NewConformanceTestingTarget(), // < TODO: fix and reenable
	"lfvm":    lfvm.NewConformanceTestingTarget(),
	"evmzero": evmzero.NewConformanceTestingTarget(),
}

func TestCt_ExplicitCases(t *testing.T) {

	revisions := []Revision{}

	for i := R07_Istanbul; i <= NewestSupportedRevision; i++ {
		revisions = append(revisions, i)
	}

	tests := map[string]Condition{}
	for _, revision := range revisions {
		tests["jump_to_2^32_"+revision.String()] =
			And(
				IsRevision(revision),
				Eq(Status(), st.Running),
				Eq(Op(Pc()), JUMP),
				Eq(Op(Constant(NewU256(0))), JUMPDEST),
				Eq(Param(0), NewU256(1<<32)),
				Ge(Gas(), vm.Gas(8)),
			)
		tests["jumpi_to_2^32"+revision.String()] =
			And(
				IsRevision(revision),
				Eq(Status(), st.Running),
				Eq(Op(Pc()), JUMPI),
				Eq(Op(Constant(NewU256(0))), JUMPDEST),
				Eq(Param(0), NewU256(1<<32)),
				Ne(Param(1), NewU256(0)),
				Ge(Gas(), vm.Gas(10)),
			)
	}

	random := rand.New(0)
	for name, condition := range tests {
		t.Run(name, func(t *testing.T) {
			generator := gen.NewStateGenerator()
			condition.Restrict(generator)

			input, err := generator.Generate(random)
			if err != nil {
				t.Fatalf("failed to generate satisfying state for %v: %v", condition, err)
			}
			if ok, err := condition.Check(input); !ok || err != nil {
				t.Fatalf("failed to generate satisfying state for %v, got %v, satisfying: %v, error:%v", condition, input, ok, err)
			}

			rules := spc.Spec.GetRulesFor(input)
			if len(rules) == 0 {
				t.Fatalf("no rule for test state: %v", input)
			}

			output := input.Clone()
			rules[0].Effect.Apply(output)

			for name, evm := range evms {
				t.Run(name, func(t *testing.T) {
					res, err := evm.StepN(input.Clone(), 1)
					if err != nil {
						t.Fatalf("failed to run test case: %v", err)
					}
					if !res.Eq(output) {
						t.Errorf("Invalid result, wanted %v, got %v", output, res)
						for _, diff := range output.Diff(res) {
							t.Errorf(diff)
						}
					}
				})
			}
		})
	}
}

const (
	fuzzIdealStackSize     int = 7  // < max pops in a single instruction
	fuzzMaximumCodeSegment int = 33 // < 1 instruction with 32 data bytes
)

// prepareInitialTestData is a helper function to be used by similar fuzzing tests
// the arguments passed to the f.Add function needs to match the arguments
// passed to the f.Fuzz function in type, position and number
func prepareInitialTestData(f *testing.F, rnd *rand.Rand) {

	for revision := R07_Istanbul; revision <= NewestSupportedRevision; revision++ {
		for op := 0x00; op < 0xFF; op++ {
			for _, gas := range []int64{0, 1, 6, 10, 1000, math.MaxInt64} {

				// generate a code segment with the operation followed by 6 random values
				ops := [fuzzMaximumCodeSegment]byte{}
				rand.Read(ops[:])
				ops[0] = byte(op)

				// generate a stack
				stack := [fuzzIdealStackSize * 32]byte{}
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
					ops[:],         // opCodes
					int64(gas),     // gas
					byte(revision), // revision
					stack[:],       // stack
				)
			}
		}
	}

	// add one more with a full stack
	ops := [fuzzMaximumCodeSegment]byte{}
	rnd.Read(ops[:])
	ops[0] = byte(0x00)
	fullStack := [1024]byte{}
	rnd.Read(fullStack[:])
	f.Add(
		ops[:],             // opCodes
		int64(0),           // gas
		byte(R07_Istanbul), // revision
		fullStack[:],       // stack
	)
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
		original.Pc, original.Code.HumanReadableString(0, 7),
		resultState.Status, referenceState.Status,
		original.Gas, resultState.Gas, referenceState.Gas,
		original.Stack.Size(), resultState.Stack.Size(), referenceState.Stack.Size(),
		original.Memory.Size(), resultState.Memory.Size(), referenceState.Memory.Size(),
	)
}

func FuzzDifferentialLfvmVsGeth(f *testing.F) {

	gethVm := geth.NewConformanceTestingTarget()
	lfvmVm := lfvm.NewConformanceTestingTarget()

	rnd := rand.New(0)

	prepareInitialTestData(f, rnd)

	f.Fuzz(func(t *testing.T, opCodes []byte, gas int64, revision byte, stackBytes []byte) {
		if gas < 0 {
			t.Skip("negative gas", gas)
		}

		if Revision(revision) < R07_Istanbul || Revision(revision) > NewestSupportedRevision {
			t.Skip("unsupported revision", revision)
		}

		if len(opCodes) == 0 {
			t.Skip("empty opCodes")
		}

		if len(opCodes) > fuzzMaximumCodeSegment {
			t.Skip("too many opCodes,  not interesting")
		}

		// Ignore stack sizes larger than 7 words, as they are not interesting
		// Do not ignore stack sizes close to the overflow, as they are interesting
		if len(stackBytes) > fuzzIdealStackSize*32 && len(stackBytes) < (1024-fuzzIdealStackSize) {
			t.Skip("Uninteresting stack size", len(stackBytes))
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
		state.Gas = vm.Gas(gas)
		state.Revision = Revision(revision)
		state.Stack = stack
		state.BlockContext.TimeStamp = GetForkTime(state.Revision)

		lfvmResultState, err := lfvmVm.StepN(state.Clone(), 1)
		defer lfvmResultState.Release()

		if err != nil {
			t.Fatalf("failed to run test case: %v", err)
		}

		gethResultState, err := gethVm.StepN(state.Clone(), 1)
		defer gethResultState.Release()

		if err != nil {
			t.Fatalf("failed to run test case in  VM: %v", err)
		}

		if lfvmResultState.Status != gethResultState.Status {
			t.Fatal("invalid result, status does not match reference status:", errorReportString(state, lfvmResultState, gethResultState))
		}
		if lfvmResultState.Status != st.Running {
			return
		}

		if lfvmResultState.Gas != gethResultState.Gas {
			t.Fatal("invalid result,  gas does not match reference gas:", errorReportString(state, lfvmResultState, gethResultState))
		}

		// Hack: lfvm does a pc transformation, but for code smaller than the required jump the pc will point to a different location
		// - Geth will point to pc+offset, whenever it is an overflow
		// - Lfvm will point min(pc+offset, len(code))
		if lfvmResultState.Pc == uint16(len(opCodes)) &&
			lfvmResultState.Pc != gethResultState.Pc {
			lfvmResultState.Pc = gethResultState.Pc
		}

		if !lfvmResultState.Eq(gethResultState) {
			t.Fatal("invalid result,  result state does not match reference state:", lfvmResultState.Diff(gethResultState), errorReportString(state, lfvmResultState, gethResultState))
		}
	})
}

func fuzzVm(testee ct.Evm, f *testing.F) {

	rnd := rand.New(0)

	prepareInitialTestData(f, rnd)

	f.Fuzz(func(t *testing.T, opCodes []byte, gas int64, revision byte, stackBytes []byte) {

		if gas < 0 {
			t.Skip("negative gas", gas)
		}

		if Revision(revision) < R07_Istanbul || Revision(revision) > NewestSupportedRevision {
			t.Skip("unsupported revision", revision)
		}

		if len(opCodes) == 0 {
			t.Skip("empty opCodes")
		}

		if len(opCodes) > fuzzMaximumCodeSegment {
			t.Skip("too many opCodes,  not interesting")
		}

		// Ignore stack sizes larger than 7 words, as they are not interesting
		// Do not ignore stack sizes close to the overflow, as they are interesting
		if len(stackBytes) > fuzzMaximumCodeSegment*32 && len(stackBytes) < (1024-fuzzMaximumCodeSegment) {
			t.Skip("Uninteresting stack size", len(stackBytes))
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
		state.Gas = vm.Gas(gas)
		state.Revision = Revision(revision)
		state.Stack = stack
		state.BlockContext.TimeStamp = GetForkTime(state.Revision)

		result, _ := testee.StepN(state.Clone(), 1)
		result.Release()

	})
}

// FuzzGeth is a fuzzing test for the geth EVM implementation
// TODO: it would be interesting to have a method to extend the corpus of the fuzzer
// with the failures found in other tests.
// So far failures will be stored in the folder: testdata/fuzz/FuzzerFunctionName/
func FuzzGeth(f *testing.F) {
	fuzzVm(geth.NewConformanceTestingTarget(), f)
}

// FuzzLfvm is a fuzzing test for the lfvm EVM implementation
func FuzzLfvm(f *testing.F) {
	fuzzVm(lfvm.NewConformanceTestingTarget(), f)
}
