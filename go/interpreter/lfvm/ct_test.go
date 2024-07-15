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
	var _ ct.Evm = ctAdapter{}
}

////////////////////////////////////////////////////////////
// ct -> lfvm

func TestConvertToLfvm_StatusCode(t *testing.T) {

	expected := map[Status]st.StatusCode{
		RUNNING:  st.Running,
		REVERTED: st.Reverted,
		RETURNED: st.Stopped,
		STOPPED:  st.Stopped,
		SUICIDED: st.Stopped,
	}

	for i := 0; i < 100; i++ {
		status := Status(i)
		want, found := expected[status]
		if !found {
			want = st.Failed
		}
		got, err := convertLfvmStatusToCtStatus(status)
		if err != nil {
			if found {
				t.Errorf("failed conversion of %v, wanted %v, got error %v", status, want, err)
			}
		} else {
			if want != got {
				t.Errorf("invalid conversion of %v, expected %v, got %v", status, want, got)
			}
		}
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
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			for _, cur := range test {
				pcMap, err := GenPcMapWithoutSuperInstructions(cur.evmCode)
				if err != nil {
					t.Fatalf("failed to generate pc map: %v", err)
				}
				lfvmPc, found := pcMap.evmToLfvm[cur.evmPc]
				if !found {
					t.Errorf("failed to resolve evm PC of %d in converted code", cur.evmPc)
				} else if want, got := cur.lfvmPc, lfvmPc; want != got {
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
				got, err := convert(cur.evmCode, false)
				if err != nil {
					t.Fatalf("failed to convert VM code to lfvm context: %v", err)
				}

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

func TestConvertToLfvm_Stack(t *testing.T) {
	newLfvmStack := func(values ...cc.U256) *Stack {
		stack := NewStack()
		for i := 0; i < len(values); i++ {
			value := values[i].Uint256()
			stack.push(&value)
		}
		return stack
	}

	tests := map[string][]struct {
		ctStack   *st.Stack
		lfvmStack *Stack
	}{
		"empty": {{
			st.NewStack(),
			newLfvmStack()}},
		"one-element": {{
			st.NewStack(cc.NewU256(7)),
			newLfvmStack(cc.NewU256(7))}},
		"two-elements": {{
			st.NewStack(cc.NewU256(1), cc.NewU256(2)),
			newLfvmStack(cc.NewU256(1), cc.NewU256(2))}},
		"three-elements": {{
			st.NewStack(cc.NewU256(1), cc.NewU256(2), cc.NewU256(3)),
			newLfvmStack(cc.NewU256(1), cc.NewU256(2), cc.NewU256(3))}},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			for _, cur := range test {
				stack := convertCtStackToLfvmStack(cur.ctStack)

				if want, got := cur.lfvmStack.len(), stack.len(); want != got {
					t.Fatalf("unexpected stack size, wanted %v, got %v", want, got)
				}

				for i := 0; i < stack.len(); i++ {
					want := cur.lfvmStack.Data()[i]
					got := stack.Data()[i]
					if want != got {
						t.Errorf("unexpected stack value, wanted %v, got %v", want, got)
					}
				}
			}
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
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			for _, cur := range test {
				pcMap, err := GenPcMapWithoutSuperInstructions(cur.evmCode)
				if err != nil {
					t.Fatalf("failed to generate pc map: %v", err)
				}
				evmPc, found := pcMap.lfvmToEvm[cur.lfvmPc]
				if !found {
					t.Errorf("failed to resolve lfvm PC of %d in converted code", cur.evmPc)
				} else if want, got := cur.evmPc, evmPc; want != got {
					t.Errorf("invalid conversion, wanted %d, got %d", want, got)
				}
			}
		})
	}
}

func TestConvertToCt_Stack(t *testing.T) {
	newLfvmStack := func(values ...cc.U256) *Stack {
		stack := NewStack()
		for i := 0; i < len(values); i++ {
			value := values[i].Uint256()
			stack.push(&value)
		}
		return stack
	}

	tests := map[string][]struct {
		lfvmStack *Stack
		ctStack   *st.Stack
	}{
		"empty": {{
			newLfvmStack(),
			st.NewStack()}},
		"one-element": {{
			newLfvmStack(cc.NewU256(7)),
			st.NewStack(cc.NewU256(7))}},
		"two-elements": {{
			newLfvmStack(cc.NewU256(1), cc.NewU256(2)),
			st.NewStack(cc.NewU256(1), cc.NewU256(2))}},
		"three-elements": {{
			newLfvmStack(cc.NewU256(1), cc.NewU256(2), cc.NewU256(3)),
			st.NewStack(cc.NewU256(1), cc.NewU256(2), cc.NewU256(3))}},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			for _, cur := range test {
				want := cur.ctStack
				ctStack := st.NewStack()
				got := convertLfvmStackToCtStack(cur.lfvmStack, ctStack)

				diffs := got.Diff(want)

				for _, diff := range diffs {
					t.Errorf("%s", diff)
				}
			}
		})
	}
}

func BenchmarkLfvmStackToCtStack(b *testing.B) {
	stack := NewStack()
	for i := 0; i < MAX_STACK_SIZE/2; i++ {
		stack.pushEmpty().SetUint64(uint64(i))
	}
	ctStack := st.NewStack()
	for i := 0; i < b.N; i++ {
		convertLfvmStackToCtStack(stack, ctStack)
	}
}
