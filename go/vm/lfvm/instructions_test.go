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
	"math"
	"testing"

	"github.com/Fantom-foundation/Tosca/go/vm"
	"github.com/holiman/uint256"
	"go.uber.org/mock/gomock"
)

func TestPushN(t *testing.T) {
	data := make([]byte, 32)
	for i := range data {
		data[i] = byte(i + 1)
	}

	code := make([]Instruction, 16)
	for i := 0; i < 32; i++ {
		code[i/2].arg = code[i/2].arg<<8 | uint16(data[i])
	}

	for n := 1; n <= 32; n++ {
		ctxt := context{
			code:  code,
			stack: NewStack(),
		}

		opPush(&ctxt, n)
		ctxt.pc++

		if ctxt.stack.len() != 1 {
			t.Errorf("expected stack size of 1, got %d", ctxt.stack.len())
			return
		}

		if int(ctxt.pc) != n/2+n%2 {
			t.Errorf("for PUSH%d program counter did not progress to %d, got %d", n, n/2+n%2, ctxt.pc)
		}

		got := ctxt.stack.peek().Bytes()
		if len(got) != n {
			t.Errorf("expected %d bytes on the stack, got %d with values %v", n, len(got), got)
		}

		for i := range got {
			if data[i] != got[i] {
				t.Errorf("for PUSH%d expected value %d to be %d, got %d", n, i, data[i], got[i])
			}
		}
	}
}

func TestPush1(t *testing.T) {
	code := []Instruction{
		{opcode: PUSH1, arg: 0x1234},
	}

	ctxt := context{
		code:  code,
		stack: NewStack(),
	}

	opPush1(&ctxt)
	ctxt.pc++

	if ctxt.stack.len() != 1 {
		t.Errorf("expected stack size of 1, got %d", ctxt.stack.len())
		return
	}

	if int(ctxt.pc) != 1 {
		t.Errorf("program counter did not progress to %d, got %d", 1, ctxt.pc)
	}

	got := ctxt.stack.peek().Bytes()
	if len(got) != 1 {
		t.Errorf("expected 1 byte on the stack, got %d with values %v", len(got), got)
	}
	if got[0] != 0x12 {
		t.Errorf("expected %d for first byte, got %d", 0x12, got[0])
	}
}

func TestPush2(t *testing.T) {
	code := []Instruction{
		{opcode: PUSH2, arg: 0x1234},
	}

	ctxt := context{
		code:  code,
		stack: NewStack(),
	}

	opPush2(&ctxt)
	ctxt.pc++

	if ctxt.stack.len() != 1 {
		t.Errorf("expected stack size of 1, got %d", ctxt.stack.len())
		return
	}

	if int(ctxt.pc) != 1 {
		t.Errorf("program counter did not progress to %d, got %d", 1, ctxt.pc)
	}

	got := ctxt.stack.peek().Bytes()
	if len(got) != 2 {
		t.Errorf("expected 2 byte on the stack, got %d with values %v", len(got), got)
	}
	if got[0] != 0x12 {
		t.Errorf("expected %d for first byte, got %d", 0x12, got[0])
	}
	if got[1] != 0x34 {
		t.Errorf("expected %d for second byte, got %d", 0x34, got[1])
	}
}

func TestPush3(t *testing.T) {
	code := []Instruction{
		{opcode: PUSH2, arg: 0x1234},
		{opcode: DATA, arg: 0x5678},
	}

	ctxt := context{
		code:  code,
		stack: NewStack(),
	}

	opPush3(&ctxt)
	ctxt.pc++

	if ctxt.stack.len() != 1 {
		t.Errorf("expected stack size of 1, got %d", ctxt.stack.len())
		return
	}

	if int(ctxt.pc) != 2 {
		t.Errorf("program counter did not progress to %d, got %d", 2, ctxt.pc)
	}

	got := ctxt.stack.peek().Bytes()
	if len(got) != 3 {
		t.Errorf("expected 3 byte on the stack, got %d with values %v", len(got), got)
	}
	if got[0] != 0x12 {
		t.Errorf("expected %d for first byte, got %d", 0x12, got[0])
	}
	if got[1] != 0x34 {
		t.Errorf("expected %d for second byte, got %d", 0x34, got[1])
	}
	if got[2] != 0x56 {
		t.Errorf("expected %d for third byte, got %d", 0x56, got[2])
	}
}

func TestPush4(t *testing.T) {
	code := []Instruction{
		{opcode: PUSH2, arg: 0x1234},
		{opcode: DATA, arg: 0x5678},
	}

	ctxt := context{
		code:  code,
		stack: NewStack(),
	}

	opPush4(&ctxt)
	ctxt.pc++

	if ctxt.stack.len() != 1 {
		t.Errorf("expected stack size of 1, got %d", ctxt.stack.len())
		return
	}

	if int(ctxt.pc) != 2 {
		t.Errorf("program counter did not progress to %d, got %d", 2, ctxt.pc)
	}

	got := ctxt.stack.peek().Bytes()
	if len(got) != 4 {
		t.Errorf("expected 3 byte on the stack, got %d with values %v", len(got), got)
	}
	if got[0] != 0x12 {
		t.Errorf("expected %d for first byte, got %d", 0x12, got[0])
	}
	if got[1] != 0x34 {
		t.Errorf("expected %d for second byte, got %d", 0x34, got[1])
	}
	if got[2] != 0x56 {
		t.Errorf("expected %d for third byte, got %d", 0x56, got[2])
	}
	if got[3] != 0x78 {
		t.Errorf("expected %d for 4th byte, got %d", 0x78, got[3])
	}
}

func TestCallChecksBalances(t *testing.T) {
	ctrl := gomock.NewController(t)
	runContext := vm.NewMockRunContext(ctrl)

	source := vm.Address{1}
	target := vm.Address{2}
	ctxt := context{
		status: RUNNING,
		params: vm.Parameters{
			Recipient: source,
		},
		context: runContext,
		stack:   NewStack(),
		memory:  NewMemory(),
		gas:     1 << 20,
	}

	// Prepare stack arguments.
	ctxt.stack.stack_ptr = 7
	ctxt.stack.data[4].Set(uint256.NewInt(1)) // < the value to be transferred
	ctxt.stack.data[5].SetBytes(target[:])    // < the target address for the call

	// The target account should exist and the source account without funds.
	runContext.EXPECT().AccountExists(target).Return(true)
	runContext.EXPECT().GetBalance(source).Return(vm.Value{})

	opCall(&ctxt)

	if want, got := RUNNING, ctxt.status; want != got {
		t.Errorf("unexpected status after call, wanted %v, got %v", want, got)
	}

	if want, got := 1, ctxt.stack.len(); want != got {
		t.Fatalf("unexpected stack size, wanted %d, got %d", want, got)
	}

	if want, got := *uint256.NewInt(0), ctxt.stack.data[0]; want != got {
		t.Fatalf("unexpected value on top of stack, wanted %v, got %v", want, got)
	}
}

func TestCreateChecksBalance(t *testing.T) {
	ctrl := gomock.NewController(t)
	runContext := vm.NewMockRunContext(ctrl)

	source := vm.Address{1}
	ctxt := context{
		status: RUNNING,
		params: vm.Parameters{
			Recipient: source,
		},
		context: runContext,
		stack:   NewStack(),
		memory:  NewMemory(),
		gas:     1 << 20,
	}

	// Prepare stack arguments.
	ctxt.stack.stack_ptr = 3
	ctxt.stack.data[2].Set(uint256.NewInt(1)) // < the value to be transferred

	// The source account should have enough funds.
	runContext.EXPECT().GetBalance(source).Return(vm.Value{})

	opCreate(&ctxt)

	if want, got := RUNNING, ctxt.status; want != got {
		t.Errorf("unexpected status after call, wanted %v, got %v", want, got)
	}

	if want, got := 1, ctxt.stack.len(); want != got {
		t.Fatalf("unexpected stack size, wanted %d, got %d", want, got)
	}

	if want, got := *uint256.NewInt(0), ctxt.stack.data[0]; want != got {
		t.Fatalf("unexpected value on top of stack, wanted %v, got %v", want, got)
	}
}

func TestLogOpSizeOverflow(t *testing.T) {

	originalBugValue := uint256.MustFromHex("0x3030303030303030")
	maxUint64 := uint256.NewInt(math.MaxUint64)
	zero := uint256.NewInt(0)

	tests := map[string]struct {
		logn     int
		size     *uint256.Int
		logCalls int
		want     Status
	}{
		"log0_zero":        {logn: 0, size: zero, logCalls: 1, want: RUNNING},
		"log1_zero":        {logn: 1, size: zero, logCalls: 1, want: RUNNING},
		"log2_zero":        {logn: 2, size: zero, logCalls: 1, want: RUNNING},
		"log3_zero":        {logn: 3, size: zero, logCalls: 1, want: RUNNING},
		"log4_zero":        {logn: 4, size: zero, logCalls: 1, want: RUNNING},
		"log0_max":         {logn: 0, size: maxUint64, logCalls: 0, want: ERROR},
		"log1_max":         {logn: 1, size: maxUint64, logCalls: 0, want: ERROR},
		"log2_max":         {logn: 2, size: maxUint64, logCalls: 0, want: ERROR},
		"log3_max":         {logn: 3, size: maxUint64, logCalls: 0, want: ERROR},
		"log4_max":         {logn: 4, size: maxUint64, logCalls: 0, want: ERROR},
		"log0_much_larger": {logn: 0, size: originalBugValue, logCalls: 0, want: ERROR},
		"log1_much_larger": {logn: 1, size: originalBugValue, logCalls: 0, want: ERROR},
		"log2_much_larger": {logn: 2, size: originalBugValue, logCalls: 0, want: ERROR},
		"log3_much_larger": {logn: 3, size: originalBugValue, logCalls: 0, want: ERROR},
		"log4_much_larger": {logn: 4, size: originalBugValue, logCalls: 0, want: ERROR},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {

			ctrl := gomock.NewController(t)
			runContext := vm.NewMockRunContext(ctrl)
			runContext.EXPECT().EmitLog(gomock.Any()).Times(test.logCalls)

			stack := NewStack()
			for i := 0; i < test.logn; i++ {
				stack.push(uint256.NewInt(0))
			}
			stack.push(test.size)
			stack.push(uint256.NewInt(0))

			ctxt := context{
				status:  RUNNING,
				gas:     392,
				context: runContext,
				stack:   stack,
				memory:  NewMemory(),
			}

			opLog(&ctxt, test.logn)

			if ctxt.status != test.want {
				t.Fatalf("unexpected status, wanted %v, got %v", test.want, ctxt.status)
			}
		})
	}
}

func TestBlobHash(t *testing.T) {

	hash := vm.Hash{1}

	tests := map[string]struct {
		setup    func(*vm.Parameters, *Stack)
		gas      vm.Gas
		revision vm.Revision
		status   Status
		want     vm.Hash
	}{
		"regular": {
			setup: func(params *vm.Parameters, stack *Stack) {
				stack.push(uint256.NewInt(0))
				params.BlobHashes = []vm.Hash{hash}
			},
			gas:      2,
			revision: vm.R13_Cancun,
			status:   RUNNING,
			want:     hash,
		},
		"old-revision": {
			setup:    func(params *vm.Parameters, stack *Stack) {},
			gas:      2,
			revision: vm.R12_Shanghai,
			status:   INVALID_INSTRUCTION,
			want:     vm.Hash{},
		},
		"no-hashes": {
			setup: func(params *vm.Parameters, stack *Stack) {
				stack.push(uint256.NewInt(0))
			},
			gas:      2,
			revision: vm.R13_Cancun,
			status:   RUNNING,
			want:     vm.Hash{},
		},
		"target-non-existent": {
			setup: func(params *vm.Parameters, stack *Stack) {
				stack.push(uint256.NewInt(1))
			},
			gas:      2,
			revision: vm.R13_Cancun,
			status:   RUNNING,
			want:     vm.Hash{},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {

			ctxt := context{
				status: RUNNING,
				params: vm.Parameters{
					Recipient: vm.Address{1},
				},
				stack:  NewStack(),
				memory: NewMemory(),
			}
			ctxt.gas = test.gas
			ctxt.revision = test.revision

			test.setup(&ctxt.params, ctxt.stack)

			opBlobHash(&ctxt)

			if ctxt.status != test.status {
				t.Fatalf("unexpected status, wanted %v, got %v", test.status, ctxt.status)
			}
			if want, got := test.want, ctxt.stack.data[0]; vm.Hash(got.Bytes32()) != want && ctxt.status == RUNNING {
				t.Fatalf("unexpected value on top of stack, wanted %v, got %v", want, got)
			}
		})
	}
}

func TestBlobBaseFee(t *testing.T) {

	blobBaseFeeValue := vm.Value{1}

	tests := map[string]struct {
		setup    func(*vm.Parameters)
		gas      vm.Gas
		revision vm.Revision
		status   Status
		want     vm.Value
	}{
		"regular": {
			setup: func(params *vm.Parameters) {
				params.BlobBaseFee = blobBaseFeeValue
			},
			gas:      2,
			revision: vm.R13_Cancun,
			status:   RUNNING,
			want:     blobBaseFeeValue,
		},
		"old-revision": {
			setup:    func(*vm.Parameters) {},
			gas:      2,
			revision: vm.R12_Shanghai,
			status:   INVALID_INSTRUCTION,
			want:     vm.Value{},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {

			ctxt := context{
				status: RUNNING,
				params: vm.Parameters{
					Recipient: vm.Address{1},
				},
				stack:  NewStack(),
				memory: NewMemory(),
			}
			ctxt.gas = test.gas
			ctxt.revision = test.revision

			test.setup(&ctxt.params)

			opBlobBaseFee(&ctxt)

			if ctxt.status != test.status {
				t.Fatalf("unexpected status, wanted %v, got %v", test.status, ctxt.status)
			}
			if want, got := test.want, ctxt.stack.data[0]; got.Cmp(new(uint256.Int).SetBytes(want[:])) != 0 && ctxt.status == RUNNING {
				t.Fatalf("unexpected value on top of stack, wanted %v, got %v", want, got)
			}
		})
	}
}
func TestMCopy(t *testing.T) {

	tests := map[string]struct {
		gasBefore      vm.Gas
		gasAfter       vm.Gas
		revision       vm.Revision
		dest           uint64
		src            uint64
		size           uint64
		memSize        uint64
		expectedStatus Status
		memoryBefore   []byte
		memoryAfter    []byte
	}{
		"empty": {
			gasBefore:      0,
			gasAfter:       0,
			revision:       vm.R13_Cancun,
			dest:           0,
			src:            0,
			size:           0,
			expectedStatus: RUNNING,
			memoryBefore: []byte{
				1, 2, 3, 4, 5, 6, 7, 8, // 0-7
				0, 0, 0, 0, 0, 0, 0, 0, // 8-15
				0, 0, 0, 0, 0, 0, 0, 0, // 16-23
				0, 0, 0, 0, 0, 0, 0, 0, // 24-31
			},
			memoryAfter: []byte{
				1, 2, 3, 4, 5, 6, 7, 8, // 0-7
				0, 0, 0, 0, 0, 0, 0, 0, // 8-15
				0, 0, 0, 0, 0, 0, 0, 0, // 16-23
				0, 0, 0, 0, 0, 0, 0, 0, // 24-31
			},
		},
		"old-revision": {
			revision:       vm.R12_Shanghai,
			expectedStatus: INVALID_INSTRUCTION,
		},
		"copy": {
			revision:       vm.R13_Cancun,
			gasBefore:      1000,
			gasAfter:       1000 - 3,
			dest:           1,
			src:            0,
			size:           8,
			expectedStatus: RUNNING,
			memoryBefore: []byte{
				1, 2, 3, 4, 5, 6, 7, 8, // 0-7
				0, 0, 0, 0, 0, 0, 0, 0, // 8-15
				0, 0, 0, 0, 0, 0, 0, 0, // 16-23
				0, 0, 0, 0, 0, 0, 0, 0, // 24-31
			},
			memoryAfter: []byte{
				1, 1, 2, 3, 4, 5, 6, 7, // 0-7
				8, 0, 0, 0, 0, 0, 0, 0, // 8-15
				0, 0, 0, 0, 0, 0, 0, 0, // 16-23
				0, 0, 0, 0, 0, 0, 0, 0, // 24-31
			},
		},
		"memory-expansion": {
			revision:       vm.R13_Cancun,
			gasBefore:      1000,
			gasAfter:       1000 - 9,
			dest:           32,
			src:            0,
			size:           4,
			expectedStatus: RUNNING,
			memoryBefore: []byte{
				1, 2, 3, 4, 0, 0, 0, 0, // 0-7
				0, 0, 0, 0, 0, 0, 0, 0, // 8-15
				0, 0, 0, 0, 0, 0, 0, 0, // 16-23
				0, 0, 0, 0, 0, 0, 0, 0, // 24-31
			},
			memoryAfter: []byte{
				1, 2, 3, 4, 0, 0, 0, 0, // 0-7
				0, 0, 0, 0, 0, 0, 0, 0, // 8-15
				0, 0, 0, 0, 0, 0, 0, 0, // 16-23
				0, 0, 0, 0, 0, 0, 0, 0, // 24-31
				1, 2, 3, 4, 0, 0, 0, 0, // 32-39
				0, 0, 0, 0, 0, 0, 0, 0, // 40-47
				0, 0, 0, 0, 0, 0, 0, 0, // 48-55
				0, 0, 0, 0, 0, 0, 0, 0, // 56-63
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {

			ctxt := context{
				status: RUNNING,
				stack:  NewStack(),
				memory: NewMemory(),
			}
			ctxt.revision = test.revision
			ctxt.gas = test.gasBefore
			ctxt.stack.push(uint256.NewInt(test.size))
			ctxt.stack.push(uint256.NewInt(test.src))
			ctxt.stack.push(uint256.NewInt(test.dest))
			ctxt.memory.store = append(ctxt.memory.store, test.memoryBefore...)

			opMcopy(&ctxt)

			if ctxt.status != test.expectedStatus {
				t.Errorf("expected status %v, got %v", test.expectedStatus, ctxt.status)
				return
			}
			if ctxt.memory.Len() != uint64(len(test.memoryAfter)) {
				t.Errorf("expected memory size %d, got %d", uint64(len(test.memoryAfter)), ctxt.memory.Len())
			}
			if !bytes.Equal(ctxt.memory.Data(), test.memoryAfter) {
				t.Errorf("expected memory %v, got %v", test.memoryAfter, ctxt.memory.Data())
			}
			if ctxt.gas != test.gasAfter {
				t.Errorf("expected gas %d, got %d", test.gasAfter, ctxt.gas)
			}
		})
	}
}

func TestCreateShanghaiInitCodeSize(t *testing.T) {
	maxInitCodeSize := uint64(49152)
	tests := []struct {
		revision       vm.Revision
		init_code_size uint64
		expected       Status
	}{
		{vm.R11_Paris, 0, RUNNING},
		{vm.R11_Paris, 1, RUNNING},
		{vm.R11_Paris, 2000, RUNNING},
		{vm.R11_Paris, maxInitCodeSize - 1, RUNNING},
		{vm.R11_Paris, maxInitCodeSize, RUNNING},
		{vm.R11_Paris, maxInitCodeSize + 1, RUNNING},
		{vm.R11_Paris, 100000, RUNNING},
		{vm.R11_Paris, math.MaxUint64, ERROR},

		{vm.R12_Shanghai, 0, RUNNING},
		{vm.R12_Shanghai, 1, RUNNING},
		{vm.R12_Shanghai, 2000, RUNNING},
		{vm.R12_Shanghai, maxInitCodeSize - 1, RUNNING},
		{vm.R12_Shanghai, maxInitCodeSize, RUNNING},
		{vm.R12_Shanghai, maxInitCodeSize + 1, MAX_INIT_CODE_SIZE_EXCEEDED},
		{vm.R12_Shanghai, 100000, MAX_INIT_CODE_SIZE_EXCEEDED},
		{vm.R12_Shanghai, math.MaxUint64, ERROR},
	}

	for _, test := range tests {
		ctrl := gomock.NewController(t)
		runContext := vm.NewMockRunContext(ctrl)

		source := vm.Address{1}
		ctxt := context{
			status: RUNNING,
			params: vm.Parameters{
				Recipient: source,
			},
			context:  runContext,
			stack:    NewStack(),
			memory:   NewMemory(),
			gas:      50000,
			revision: test.revision,
		}

		// Prepare stack arguments.
		ctxt.stack.stack_ptr = 3
		ctxt.stack.data[0].Set(uint256.NewInt(test.init_code_size))

		if test.expected == RUNNING {
			runContext.EXPECT().Call(vm.Create, gomock.Any()).Return(vm.CallResult{}, nil)
		}

		opCreate(&ctxt)

		if want, got := test.expected, ctxt.status; want != got {
			t.Errorf("unexpected status after call, wanted %v, got %v", want, got)
		}
	}
}

func TestCreateShanghaiDeploymentCost(t *testing.T) {
	tests := []struct {
		revision     vm.Revision
		initCodeSize uint64
	}{
		// gas cost from evm.codes
		{vm.R11_Paris, 0},
		{vm.R11_Paris, 1},
		{vm.R11_Paris, 2000},
		{vm.R11_Paris, 49152},

		{vm.R12_Shanghai, 0},
		{vm.R12_Shanghai, 1},
		{vm.R12_Shanghai, 2000},
		{vm.R12_Shanghai, 49152},
	}

	dynamicCost := func(revision vm.Revision, size uint64) uint64 {
		words := sizeInWords(size)
		// TODO: check for overflow, fix in #524
		memoryExpansionCost := (words*words)/512 + 3*words
		if revision >= vm.R12_Shanghai {
			return 2*words + memoryExpansionCost
		}
		return memoryExpansionCost
	}

	for _, test := range tests {
		ctrl := gomock.NewController(t)
		runContext := vm.NewMockRunContext(ctrl)

		cost := dynamicCost(test.revision, test.initCodeSize)

		source := vm.Address{1}
		ctxt := context{
			status: RUNNING,
			params: vm.Parameters{
				Recipient: source,
			},
			context:  runContext,
			stack:    NewStack(),
			memory:   NewMemory(),
			gas:      vm.Gas(cost),
			revision: test.revision,
		}

		// Prepare stack arguments.
		ctxt.stack.stack_ptr = 3
		ctxt.stack.data[0].Set(uint256.NewInt(test.initCodeSize))

		runContext.EXPECT().Call(vm.Create, gomock.Any()).Return(vm.CallResult{}, nil)

		opCreate(&ctxt)

		if ctxt.status != RUNNING {
			t.Errorf("unexpected status after call, wanted RUNNING, got %v", ctxt.status)
		}

		if ctxt.gas != 0 {
			t.Errorf("unexpected gas cost, wanted %d, got %d", cost, cost-uint64(ctxt.gas))
		}
	}
}

func TestTransientStorageOperations(t *testing.T) {
	tests := map[string]struct {
		op       func(*context)
		setup    func(*vm.MockRunContext)
		stackPtr int
		revision vm.Revision
		status   Status
	}{
		"tload-regular": {
			op: opTload,
			setup: func(runContext *vm.MockRunContext) {
				runContext.EXPECT().GetTransientStorage(gomock.Any(), gomock.Any()).Return(vm.Word{})
			},
			stackPtr: 1,
			revision: vm.R13_Cancun,
			status:   RUNNING,
		},
		"tload-old-revision": {
			op:       opTload,
			setup:    func(runContext *vm.MockRunContext) {},
			stackPtr: 1,
			revision: vm.R11_Paris,
			status:   INVALID_INSTRUCTION,
		},
		"tstore-regular": {
			op: opTstore,
			setup: func(runContext *vm.MockRunContext) {
				runContext.EXPECT().SetTransientStorage(gomock.Any(), gomock.Any(), gomock.Any())
			},
			stackPtr: 2,
			revision: vm.R13_Cancun,
			status:   RUNNING,
		},
		"tstore-old-revision": {
			op:       opTstore,
			setup:    func(runContext *vm.MockRunContext) {},
			stackPtr: 2,
			revision: vm.R11_Paris,
			status:   INVALID_INSTRUCTION,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			ctxt := context{
				status: RUNNING,
				params: vm.Parameters{
					Recipient: vm.Address{1},
				},
				stack:    NewStack(),
				revision: test.revision,
			}
			runContext := vm.NewMockRunContext(ctrl)
			test.setup(runContext)
			ctxt.context = runContext
			ctxt.stack.stack_ptr = test.stackPtr

			test.op(&ctxt)

			if ctxt.status != test.status {
				t.Errorf("unexpected status, wanted %v, got %v", test.status, ctxt.status)
			}
		})
	}
}
