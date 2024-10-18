// Copyright (c) 2024 Fantom Foundation
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at fantom.foundation/bsl11.
//
// Change Date: 2028-4-16
//
// On the date above, in accordance with the Business Source License, use of
// this software will be governed by the GNU Lesser General Public License v3.

package lfvm

import (
	"bytes"
	"errors"
	"reflect"
	"testing"

	"github.com/Fantom-foundation/Tosca/go/ct"
	cc "github.com/Fantom-foundation/Tosca/go/ct/common"
	"github.com/Fantom-foundation/Tosca/go/ct/st"
	"github.com/Fantom-foundation/Tosca/go/tosca"
	"github.com/Fantom-foundation/Tosca/go/tosca/vm"
)

func TestCtAdapter_Add(t *testing.T) {
	s := st.NewState(st.NewCode([]byte{
		byte(vm.PUSH1), 3,
		byte(vm.PUSH1), 4,
		byte(vm.ADD),
	}))
	s.Status = st.Running
	s.Revision = tosca.R07_Istanbul
	s.Pc = 0
	s.Gas = 100
	s.Stack = st.NewStack(cc.NewU256(1), cc.NewU256(2))
	defer s.Stack.Release()
	s.Memory = st.NewMemory(1, 2, 3)

	c := NewConformanceTestingTarget()

	s, err := c.StepN(s, 4)

	if err != nil {
		t.Fatalf("unexpected conversion error: %v", err)
	}

	if want, got := st.Stopped, s.Status; want != got {
		t.Fatalf("unexpected status: wanted %v, got %v", want, got)
	}

	if want, got := cc.NewU256(3+4), s.Stack.Get(0); !want.Eq(got) {
		t.Errorf("unexpected result: wanted %s, got %s", want, got)
	}
}

func TestCtAdapter_Interface(t *testing.T) {
	// Compile time check that ctAdapter implements the st.Evm interface.
	var _ ct.Evm = &ctAdapter{}
}

func TestCTAdapter_DoesNotAddDuplicatedCodeToPCMap(t *testing.T) {
	s := st.NewState(st.NewCode([]byte{
		byte(vm.STOP),
	}))
	c := NewConformanceTestingTarget()

	for i := 0; i < 3; i++ {
		s.Status = st.Running
		_, err := c.StepN(s, 1)
		if err != nil {
			t.Fatalf("unexpected conversion error: %v", err)
		}
		if want, got := 1, c.(*ctAdapter).pcMapCache.Len(); want != got {
			t.Fatalf("unexpected pc map size, wanted %d, got %d", want, got)
		}
	}
}

func TestCtAdapter_ReturnsErrorForUnsupportedRevisions(t *testing.T) {
	unsupportedRevision := newestSupportedRevision + 1
	want := &tosca.ErrUnsupportedRevision{Revision: unsupportedRevision}
	s := st.NewState(st.NewCode([]byte{
		byte(vm.STOP),
	}))
	s.Revision = unsupportedRevision

	c := NewConformanceTestingTarget()
	_, err := c.StepN(s, 1)

	var e *tosca.ErrUnsupportedRevision
	if !errors.As(err, &e) {
		t.Errorf("unexpected error, wanted %v, got %v", want, err)
	}
}

func TestCtAdapter_DoesNotAffectNonRunningStates(t *testing.T) {
	s := st.NewState(st.NewCode([]byte{
		byte(vm.STOP),
	}))
	s.Status = st.Stopped

	c := NewConformanceTestingTarget()
	s2, err := c.StepN(s.Clone(), 1)
	if err != nil {
		t.Fatalf("unexpected conversion error: %v", err)
	}
	if !s.Eq(s2) {
		t.Errorf("unexpected state, wanted %v, got %v", s, s2)
	}
}

func TestCtAdapter_SetsPcOnResultingState(t *testing.T) {
	s := st.NewState(st.NewCode([]byte{
		byte(vm.PUSH1),
		0x01,
		byte(vm.PUSH0),
	}))
	s.Gas = 100
	s.Stack = st.NewStack()
	defer s.Stack.Release()
	c := NewConformanceTestingTarget()
	s2, err := c.StepN(s, 1)
	if err != nil {
		t.Fatalf("unexpected conversion error: %v", err)
	}
	if want, got := uint16(2), s2.Pc; want != got {
		t.Errorf("unexpected pc, wanted %d, got %d", want, got)
	}
}

func TestCtAdapter_FillsReturnDataOnResultingState(t *testing.T) {
	s := st.NewState(st.NewCode([]byte{
		byte(vm.PUSH1), byte(1),
		byte(vm.PUSH1), byte(0),
		byte(vm.RETURN),
	}))
	s.Gas = 100
	memory := []byte{0xFA}
	s.Memory.Append(memory)
	c := NewConformanceTestingTarget()
	s2, err := c.StepN(s, 3)
	if err != nil {
		t.Fatalf("unexpected conversion error: %v", err)
	}
	if want, got := memory, s2.ReturnData.ToBytes(); !bytes.Equal(want, got) {
		t.Errorf("unexpected return data, wanted %v, got %v", want, got)
	}
}

////////////////////////////////////////////////////////////
// ct -> lfvm

func TestConvertToLfvm_StatusCode(t *testing.T) {

	tests := map[status]st.StatusCode{
		statusRunning:        st.Running,
		statusReverted:       st.Reverted,
		statusReturned:       st.Stopped,
		statusStopped:        st.Stopped,
		statusSelfDestructed: st.Stopped,
		statusFailed:         st.Failed,
	}

	for status, test := range tests {
		got := convertLfvmStatusToCtStatus(status)
		if want, got := test, got; want != got {
			t.Errorf("unexpected conversion, wanted %v, got %v", want, got)
		}
	}
}

func TestConvertToLfvm_StatusCodeFailsOnUnknownStatus(t *testing.T) {
	status := convertLfvmStatusToCtStatus(statusFailed + 1)
	if status != st.Failed {
		t.Errorf("unexpected conversion, wanted %v, got %v", st.Failed, status)
	}
}

func TestConvertToLfvm_Pc(t *testing.T) {
	tests := map[string][]struct {
		evmCode []byte
		evmPc   uint16
		lfvmPc  uint16
	}{
		"empty":        {{}},
		"pos-0":        {{[]byte{byte(vm.STOP)}, 0, 0}},
		"pos-1":        {{[]byte{byte(vm.STOP), byte(vm.STOP), byte(vm.STOP)}, 1, 1}},
		"one-past-end": {{[]byte{byte(vm.STOP)}, 1, 1}},
		"shifted": {{[]byte{
			byte(vm.PUSH1), 0x01,
			byte(vm.PUSH1), 0x02,
			byte(vm.ADD)}, 2, 1}},
		"jumpdest": {{[]byte{
			byte(vm.PUSH3), 0x00, 0x00, 0x06,
			byte(vm.JUMP),
			byte(vm.INVALID),
			byte(vm.JUMPDEST)},
			6, 6}},
		"extra padding for truncated push": {{[]byte{
			byte(vm.PUSH14), 0x2e, 0x5a, 0x30, 0x10, 0x64,
		}, 6, 7}},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			for _, cur := range test {
				pcMap := genPcMap(cur.evmCode)
				lfvmPc := pcMap.evmToLfvm[cur.evmPc]
				if want, got := cur.lfvmPc, lfvmPc; want != got {
					t.Errorf("invalid conversion, wanted %d, got %d", want, got)
				}
			}
		})
	}
}

func TestConvertToLfvm_Code(t *testing.T) {
	tests := map[string][]struct {
		evmCode  []byte
		lfvmCode Code
	}{
		"empty": {{}},
		"stop":  {{[]byte{byte(vm.STOP)}, Code{Instruction{STOP, 0x0000}}}},
		"add": {{[]byte{
			byte(vm.PUSH1), 0x01,
			byte(vm.PUSH1), 0x02,
			byte(vm.ADD)},
			Code{Instruction{PUSH1, 0x0100},
				Instruction{PUSH1, 0x0200},
				Instruction{ADD, 0x0000}}}},
		"jump": {{[]byte{
			byte(vm.PUSH1), 0x04,
			byte(vm.JUMP),
			byte(vm.INVALID),
			byte(vm.JUMPDEST)},
			Code{Instruction{PUSH1, 0x0400},
				Instruction{JUMP, 0x0000},
				Instruction{INVALID, 0x0000},
				Instruction{JUMP_TO, 0x0004},
				Instruction{JUMPDEST, 0x0000}}}},
		"jumpdest": {{[]byte{
			byte(vm.PUSH3), 0x00, 0x00, 0x06,
			byte(vm.JUMP),
			byte(vm.INVALID),
			byte(vm.JUMPDEST)},
			Code{Instruction{PUSH3, 0x0000},
				Instruction{DATA, 0x0600},
				Instruction{JUMP, 0x0000},
				Instruction{INVALID, 0x0000},
				Instruction{JUMP_TO, 0x0006},
				Instruction{NOOP, 0x0000},
				Instruction{JUMPDEST, 0x0000}}}},
		"push2": {{[]byte{byte(vm.PUSH2), 0xBA, 0xAD}, Code{Instruction{PUSH2, 0xBAAD}}}},
		"push3": {{[]byte{byte(vm.PUSH3), 0xBA, 0xAD, 0xC0}, Code{Instruction{PUSH3, 0xBAAD}, Instruction{DATA, 0xC000}}}},
		"push31": {{[]byte{
			byte(vm.PUSH31),
			0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C, 0x0D, 0x0E, 0x0F, 0x10,
			0x11, 0x12, 0x13, 0x14, 0x15, 0x16, 0x17, 0x18, 0x19, 0x1A, 0x1B, 0x1C, 0x1D, 0x1E, 0x1F},
			Code{Instruction{PUSH31, 0x0102},
				Instruction{DATA, 0x0304},
				Instruction{DATA, 0x0506},
				Instruction{DATA, 0x0708},
				Instruction{DATA, 0x090A},
				Instruction{DATA, 0x0B0C},
				Instruction{DATA, 0x0D0E},
				Instruction{DATA, 0x0F10},
				Instruction{DATA, 0x1112},
				Instruction{DATA, 0x1314},
				Instruction{DATA, 0x1516},
				Instruction{DATA, 0x1718},
				Instruction{DATA, 0x191A},
				Instruction{DATA, 0x1B1C},
				Instruction{DATA, 0x1D1E},
				Instruction{DATA, 0x1F00}}}},
		"push32": {{[]byte{
			byte(vm.PUSH32),
			0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C, 0x0D, 0x0E, 0x0F, 0x10,
			0x11, 0x12, 0x13, 0x14, 0x15, 0x16, 0x17, 0x18, 0x19, 0x1A, 0x1B, 0x1C, 0x1D, 0x1E, 0x1F, 0xFF},
			Code{Instruction{PUSH32, 0x0102},
				Instruction{DATA, 0x0304},
				Instruction{DATA, 0x0506},
				Instruction{DATA, 0x0708},
				Instruction{DATA, 0x090A},
				Instruction{DATA, 0x0B0C},
				Instruction{DATA, 0x0D0E},
				Instruction{DATA, 0x0F10},
				Instruction{DATA, 0x1112},
				Instruction{DATA, 0x1314},
				Instruction{DATA, 0x1516},
				Instruction{DATA, 0x1718},
				Instruction{DATA, 0x191A},
				Instruction{DATA, 0x1B1C},
				Instruction{DATA, 0x1D1E},
				Instruction{DATA, 0x1FFF}}}},
		"invalid": {{[]byte{byte(vm.INVALID)}, Code{Instruction{INVALID, 0x0000}}}},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			for _, cur := range test {
				got := convert(cur.evmCode, ConversionConfig{})

				want := cur.lfvmCode

				if wantSize, gotSize := len(want), len(got); wantSize != gotSize {
					t.Fatalf("unexpected code size, wanted %d, got %d", wantSize, gotSize)
				}

				for i := 0; i < len(got); i++ {
					if wantInst, gotInst := want[i], got[i]; wantInst != gotInst {
						t.Errorf("unexpected instruction, wanted %v, got %v", wantInst, gotInst)
					}
				}
			}
		})
	}
}

func TestConvertToLfvm_CodeWithSuperInstructions(t *testing.T) {
	tests := map[string]struct {
		evmCode []byte
		want    Code
	}{
		"PUSH1PUSH4DUP3": {
			[]byte{byte(vm.PUSH1), 0x01,
				byte(vm.PUSH4), 0x01, 0x02, 0x03, 0x04,
				byte(vm.DUP3)},
			Code{Instruction{PUSH1_PUSH4_DUP3, 0x0100},
				Instruction{DATA, 0x0102},
				Instruction{DATA, 0x0304},
			}},
		"PUSH1_PUSH1_PUSH1_SHL_SUB": {
			[]byte{byte(vm.PUSH1), 0x01,
				byte(vm.PUSH1), 0x01,
				byte(vm.PUSH1), 0x01,
				byte(vm.SHL),
				byte(vm.SUB)},
			Code{Instruction{PUSH1_PUSH1_PUSH1_SHL_SUB, 0x0101},
				Instruction{DATA, 0x0001},
			}},
		"AND_SWAP1_POP_SWAP2_SWAP1": {
			[]byte{byte(vm.AND), byte(vm.SWAP1), byte(vm.POP),
				byte(vm.SWAP2), byte(vm.SWAP1)},
			Code{Instruction{AND_SWAP1_POP_SWAP2_SWAP1, 0x0000}}},
		"ISZERO_PUSH2_JUMPI": {
			[]byte{byte(vm.ISZERO),
				byte(vm.PUSH2), 0x01, 0x02,
				byte(vm.JUMPI)},
			Code{Instruction{ISZERO_PUSH2_JUMPI, 0x0102}}},
		"SWAP2_SWAP1_POP_JUMP": {
			[]byte{byte(vm.SWAP2), byte(vm.SWAP1), byte(vm.POP),
				byte(vm.JUMP)},
			Code{Instruction{SWAP2_SWAP1_POP_JUMP, 0x0000}}},
		"SWAP1_POP_SWAP2_SWAP1": {
			[]byte{byte(vm.SWAP1), byte(vm.POP), byte(vm.SWAP2),
				byte(vm.SWAP1)},
			Code{Instruction{SWAP1_POP_SWAP2_SWAP1, 0x0000}}},
		"POP_SWAP2_SWAP1_POP": {
			[]byte{byte(vm.POP), byte(vm.SWAP2), byte(vm.SWAP1),
				byte(vm.POP)},
			Code{Instruction{POP_SWAP2_SWAP1_POP, 0x0000}}},
		"PUSH2_JUMP": {
			[]byte{byte(vm.PUSH2), 0x01, 0x02,
				byte(vm.JUMP)},
			Code{Instruction{PUSH2_JUMP, 0x0102}}},
		"PUSH2_JUMPI": {
			[]byte{byte(vm.PUSH2), 0x01, 0x02,
				byte(vm.JUMPI)},
			Code{Instruction{PUSH2_JUMPI, 0x0102}}},
		"PUSH1_PUSH1": {
			[]byte{byte(vm.PUSH1), 0x01,
				byte(vm.PUSH1), 0x01},
			Code{Instruction{PUSH1_PUSH1, 0x0101}}},
		"PUSH1_ADD": {
			[]byte{byte(vm.PUSH1), 0x01,
				byte(vm.ADD)},
			Code{Instruction{PUSH1_ADD, 0x0001}}},
		"PUSH1_SHL": {
			[]byte{byte(vm.PUSH1), 0x01,
				byte(vm.SHL)},
			Code{Instruction{PUSH1_SHL, 0x0001}}},
		"PUSH1_DUP1": {
			[]byte{byte(vm.PUSH1), 0x01,
				byte(vm.DUP1)},
			Code{Instruction{PUSH1_DUP1, 0x0001}}},
		"SWAP1_POP": {
			[]byte{byte(vm.SWAP1), byte(vm.POP)},
			Code{Instruction{SWAP1_POP, 0x0000}}},
		"POP_JUMP": {
			[]byte{byte(vm.POP), byte(vm.JUMP)},
			Code{Instruction{POP_JUMP, 0x0000}}},
		"POP_POP": {
			[]byte{byte(vm.POP), byte(vm.POP)},
			Code{Instruction{POP_POP, 0x0000}}},
		"SWAP2_SWAP1": {
			[]byte{byte(vm.SWAP2), byte(vm.SWAP1)},
			Code{Instruction{SWAP2_SWAP1, 0x0000}}},
		"SWAP2_POP": {
			[]byte{byte(vm.SWAP2), byte(vm.POP)},
			Code{Instruction{SWAP2_POP, 0x0000}}},
		"DUP2_MSTORE": {
			[]byte{byte(vm.DUP2), byte(vm.MSTORE)},
			Code{Instruction{DUP2_MSTORE, 0x0000}}},
		"DUP2_LT": {
			[]byte{byte(vm.DUP2), byte(vm.LT)},
			Code{Instruction{DUP2_LT, 0x0000}}},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			options := ConversionConfig{WithSuperInstructions: true}
			got := convert(test.evmCode, options)
			if !reflect.DeepEqual(test.want, got) {
				t.Fatalf("unexpected code, wanted %v, got %v", test.want, got)
			}
		})
	}
}

func TestConvertToLfvm_Stack(t *testing.T) {
	newLfvmStack := func(values ...cc.U256) *stack {
		stack := NewStack()
		for i := 0; i < len(values); i++ {
			value := values[i].Uint256()
			stack.push(&value)
		}
		return stack
	}

	tests := map[string]struct {
		ctStack   *st.Stack
		lfvmStack *stack
	}{
		"empty": {
			st.NewStack(),
			newLfvmStack()},
		"one-element": {
			st.NewStack(cc.NewU256(7)),
			newLfvmStack(cc.NewU256(7))},
		"two-elements": {
			st.NewStack(cc.NewU256(1), cc.NewU256(2)),
			newLfvmStack(cc.NewU256(1), cc.NewU256(2))},
		"three-elements": {
			st.NewStack(cc.NewU256(1), cc.NewU256(2), cc.NewU256(3)),
			newLfvmStack(cc.NewU256(1), cc.NewU256(2), cc.NewU256(3))},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			stack := convertCtStackToLfvmStack(test.ctStack)
			if want, got := test.lfvmStack.len(), stack.len(); want != got {
				t.Fatalf("unexpected stack size, wanted %v, got %v", want, got)
			}
			for i := 0; i < stack.len(); i++ {
				want := *test.lfvmStack.get(i)
				got := *stack.get(i)
				if want != got {
					t.Errorf("unexpected stack value, wanted %v, got %v", want, got)
				}
			}
			ReturnStack(test.lfvmStack)
			ReturnStack(stack)
			test.ctStack.Release()
		})
	}
}

////////////////////////////////////////////////////////////
// lfvm -> ct

func TestConvertToCt_Pc(t *testing.T) {
	tests := map[string][]struct {
		evmCode []byte
		lfvmPc  uint16
		evmPc   uint16
	}{
		"empty":        {{}},
		"pos-0":        {{[]byte{byte(vm.STOP)}, 0, 0}},
		"pos-1":        {{[]byte{byte(vm.STOP), byte(vm.STOP), byte(vm.STOP)}, 1, 1}},
		"one-past-end": {{[]byte{byte(vm.STOP)}, 1, 1}},
		"shifted": {{[]byte{
			byte(vm.PUSH1), 0x01,
			byte(vm.PUSH1), 0x02,
			byte(vm.ADD)}, 1, 2}},
		"jumpdest": {{[]byte{
			byte(vm.PUSH3), 0x00, 0x00, 0x06,
			byte(vm.JUMP),
			byte(vm.INVALID),
			byte(vm.JUMPDEST)},
			6, 6}},
		"extra padding for truncated push": {{[]byte{
			byte(vm.PUSH14), 0x2e, 0x5a, 0x30, 0x10, 0x64,
		}, 7, 6}},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			for _, cur := range test {
				pcMap := genPcMap(cur.evmCode)
				evmPc := pcMap.lfvmToEvm[cur.lfvmPc]
				if want, got := cur.evmPc, evmPc; want != got {
					t.Errorf("invalid conversion, wanted %d, got %d", want, got)
				}
			}
		})
	}
}

func TestConvertToCt_Stack(t *testing.T) {
	newLfvmStack := func(values ...cc.U256) *stack {
		stack := NewStack()
		for i := 0; i < len(values); i++ {
			value := values[i].Uint256()
			stack.push(&value)
		}
		return stack
	}

	tests := map[string]struct {
		lfvmStack *stack
		ctStack   *st.Stack
	}{
		"empty": {
			newLfvmStack(),
			st.NewStack()},
		"one-element": {
			newLfvmStack(cc.NewU256(7)),
			st.NewStack(cc.NewU256(7))},
		"two-elements": {
			newLfvmStack(cc.NewU256(1), cc.NewU256(2)),
			st.NewStack(cc.NewU256(1), cc.NewU256(2))},
		"three-elements": {
			newLfvmStack(cc.NewU256(1), cc.NewU256(2), cc.NewU256(3)),
			st.NewStack(cc.NewU256(1), cc.NewU256(2), cc.NewU256(3))},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			want := test.ctStack
			ctStack := st.NewStack()
			got := convertLfvmStackToCtStack(test.lfvmStack, ctStack)

			diffs := got.Diff(want)
			for _, diff := range diffs {
				t.Errorf("%s", diff)
			}
			ReturnStack(test.lfvmStack)
			test.ctStack.Release()
			ctStack.Release()
		})
	}
}

func BenchmarkLfvmStackToCtStack(b *testing.B) {
	stack := NewStack()
	for i := 0; i < MAX_STACK_SIZE/2; i++ {
		stack.pushUndefined().SetUint64(uint64(i))
	}
	ctStack := st.NewStack()
	for i := 0; i < b.N; i++ {
		convertLfvmStackToCtStack(stack, ctStack)
	}
}
