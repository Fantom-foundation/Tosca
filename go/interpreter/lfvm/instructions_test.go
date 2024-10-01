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
	"fmt"
	"math"
	"testing"

	"github.com/Fantom-foundation/Tosca/go/tosca"
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
	runContext := tosca.NewMockRunContext(ctrl)

	source := tosca.Address{1}
	target := tosca.Address{2}
	ctxt := context{
		params: tosca.Parameters{
			Recipient: source,
		},
		context: runContext,
		stack:   NewStack(),
		memory:  NewMemory(),
		gas:     1 << 20,
	}

	// Prepare stack arguments.
	ctxt.stack.stackPointer = 7
	ctxt.stack.data[4].Set(uint256.NewInt(1)) // < the value to be transferred
	ctxt.stack.data[5].SetBytes(target[:])    // < the target address for the call

	// The target account should exist and the source account without funds.
	runContext.EXPECT().AccountExists(target).Return(true)
	runContext.EXPECT().GetBalance(source).Return(tosca.Value{})

	err := opCall(&ctxt)
	if err != nil {
		t.Errorf("opCall failed: %v", err)
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
	runContext := tosca.NewMockRunContext(ctrl)

	source := tosca.Address{1}
	ctxt := context{
		params: tosca.Parameters{
			Recipient: source,
		},
		context: runContext,
		stack:   NewStack(),
		memory:  NewMemory(),
		gas:     1 << 20,
	}

	// Prepare stack arguments.
	ctxt.stack.stackPointer = 3
	ctxt.stack.data[2].Set(uint256.NewInt(1)) // < the value to be transferred

	// The source account should have enough funds.
	runContext.EXPECT().GetBalance(source).Return(tosca.Value{})

	err := opCreate(&ctxt)
	if err != nil {
		t.Errorf("opCreate failed: %v", err)
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
		logn          int
		size          *uint256.Int
		logCalls      int
		expectedError error
	}{
		"log0_zero":        {logn: 0, size: zero, logCalls: 1, expectedError: nil},
		"log1_zero":        {logn: 1, size: zero, logCalls: 1, expectedError: nil},
		"log2_zero":        {logn: 2, size: zero, logCalls: 1, expectedError: nil},
		"log3_zero":        {logn: 3, size: zero, logCalls: 1, expectedError: nil},
		"log4_zero":        {logn: 4, size: zero, logCalls: 1, expectedError: nil},
		"log0_max":         {logn: 0, size: maxUint64, logCalls: 0, expectedError: errOutOfGas},
		"log1_max":         {logn: 1, size: maxUint64, logCalls: 0, expectedError: errOutOfGas},
		"log2_max":         {logn: 2, size: maxUint64, logCalls: 0, expectedError: errOutOfGas},
		"log3_max":         {logn: 3, size: maxUint64, logCalls: 0, expectedError: errOutOfGas},
		"log4_max":         {logn: 4, size: maxUint64, logCalls: 0, expectedError: errOutOfGas},
		"log0_much_larger": {logn: 0, size: originalBugValue, logCalls: 0, expectedError: errOutOfGas},
		"log1_much_larger": {logn: 1, size: originalBugValue, logCalls: 0, expectedError: errOutOfGas},
		"log2_much_larger": {logn: 2, size: originalBugValue, logCalls: 0, expectedError: errOutOfGas},
		"log3_much_larger": {logn: 3, size: originalBugValue, logCalls: 0, expectedError: errOutOfGas},
		"log4_much_larger": {logn: 4, size: originalBugValue, logCalls: 0, expectedError: errOutOfGas},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {

			ctrl := gomock.NewController(t)
			runContext := tosca.NewMockRunContext(ctrl)
			runContext.EXPECT().EmitLog(gomock.Any()).Times(test.logCalls)

			stack := NewStack()
			for i := 0; i < test.logn; i++ {
				stack.push(uint256.NewInt(0))
			}
			stack.push(test.size)
			stack.push(uint256.NewInt(0))

			ctxt := context{
				gas:     392,
				context: runContext,
				stack:   stack,
				memory:  NewMemory(),
			}

			err := opLog(&ctxt, test.logn)
			if got, want := err, test.expectedError; got != want {
				t.Fatalf("unexpected result, wanted %v, got %v", want, got)
			}
		})
	}
}

func TestBlobHash(t *testing.T) {

	hash := tosca.Hash{1}

	tests := map[string]struct {
		setup    func(*tosca.Parameters, *stack)
		gas      tosca.Gas
		revision tosca.Revision
		err      error
		want     tosca.Hash
	}{
		"regular": {
			setup: func(params *tosca.Parameters, stack *stack) {
				stack.push(uint256.NewInt(0))
				params.BlobHashes = []tosca.Hash{hash}
			},
			gas:      2,
			revision: tosca.R13_Cancun,
			want:     hash,
		},
		"old-revision": {
			setup:    func(params *tosca.Parameters, stack *stack) {},
			gas:      2,
			revision: tosca.R12_Shanghai,
			err:      errInvalidRevision,
			want:     tosca.Hash{},
		},
		"no-hashes": {
			setup: func(params *tosca.Parameters, stack *stack) {
				stack.push(uint256.NewInt(0))
			},
			gas:      2,
			revision: tosca.R13_Cancun,
			want:     tosca.Hash{},
		},
		"target-non-existent": {
			setup: func(params *tosca.Parameters, stack *stack) {
				stack.push(uint256.NewInt(1))
			},
			gas:      2,
			revision: tosca.R13_Cancun,
			want:     tosca.Hash{},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {

			ctxt := context{
				params: tosca.Parameters{
					Recipient: tosca.Address{1},
				},
				stack:  NewStack(),
				memory: NewMemory(),
			}
			ctxt.gas = test.gas
			ctxt.params.Revision = test.revision

			test.setup(&ctxt.params, ctxt.stack)

			err := opBlobHash(&ctxt)
			if want, got := test.err, err; want != got {
				t.Fatalf("unexpected return, wanted %v, got %v", want, got)
			}

			if want, got := test.want, ctxt.stack.data[0]; tosca.Hash(got.Bytes32()) != want {
				t.Fatalf("unexpected value on top of stack, wanted %v, got %v", want, got)
			}
		})
	}
}

func TestBlobBaseFee(t *testing.T) {

	blobBaseFeeValue := tosca.Value{1}

	tests := map[string]struct {
		setup    func(*tosca.Parameters)
		gas      tosca.Gas
		revision tosca.Revision
		err      error
		want     tosca.Value
	}{
		"regular": {
			setup: func(params *tosca.Parameters) {
				params.BlobBaseFee = blobBaseFeeValue
			},
			gas:      2,
			revision: tosca.R13_Cancun,
			want:     blobBaseFeeValue,
		},
		"old-revision": {
			setup:    func(*tosca.Parameters) {},
			gas:      2,
			revision: tosca.R12_Shanghai,
			err:      errInvalidRevision,
			want:     tosca.Value{},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {

			ctxt := context{
				params: tosca.Parameters{
					Recipient: tosca.Address{1},
				},
				stack:  NewStack(),
				memory: NewMemory(),
			}
			ctxt.gas = test.gas
			ctxt.params.Revision = test.revision

			test.setup(&ctxt.params)

			err := opBlobBaseFee(&ctxt)
			if want, got := test.err, err; want != got {
				t.Fatalf("unexpected return, wanted %v, got %v", want, got)
			}

			if want, got := test.want, ctxt.stack.data[0]; got.Cmp(new(uint256.Int).SetBytes(want[:])) != 0 {
				t.Fatalf("unexpected value on top of stack, wanted %v, got %v", want, got)
			}
		})
	}
}

func TestMCopy(t *testing.T) {

	tests := map[string]struct {
		gasBefore    tosca.Gas
		gasAfter     tosca.Gas
		revision     tosca.Revision
		dest         uint64
		src          uint64
		size         uint64
		memSize      uint64
		memoryBefore []byte
		memoryAfter  []byte
		err          error
	}{
		"empty": {
			gasBefore: 0,
			gasAfter:  0,
			revision:  tosca.R13_Cancun,
			dest:      0,
			src:       0,
			size:      0,
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
			revision: tosca.R12_Shanghai,
			err:      errInvalidRevision,
		},
		"copy": {
			revision:  tosca.R13_Cancun,
			gasBefore: 1000,
			gasAfter:  1000 - 3,
			dest:      1,
			src:       0,
			size:      8,
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
			revision:  tosca.R13_Cancun,
			gasBefore: 1000,
			gasAfter:  1000 - 9,
			dest:      32,
			src:       0,
			size:      4,
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
				stack:  NewStack(),
				memory: NewMemory(),
			}
			ctxt.params.Revision = test.revision
			ctxt.gas = test.gasBefore
			ctxt.stack.push(uint256.NewInt(test.size))
			ctxt.stack.push(uint256.NewInt(test.src))
			ctxt.stack.push(uint256.NewInt(test.dest))
			ctxt.memory.store = append(ctxt.memory.store, test.memoryBefore...)

			err := opMcopy(&ctxt)
			if want, got := test.err, err; want != got {
				t.Fatalf("unexpected return, wanted %v, got %v", want, got)
			}

			if ctxt.memory.length() != uint64(len(test.memoryAfter)) {
				t.Errorf("expected memory size %d, got %d", uint64(len(test.memoryAfter)), ctxt.memory.length())
			}
			data, err := ctxt.memory.getSlice(0, ctxt.memory.length(), &ctxt)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if !bytes.Equal(data, test.memoryAfter) {
				t.Errorf("expected memory %v, got %v", test.memoryAfter, data)
			}
			if ctxt.gas != test.gasAfter {
				t.Errorf("expected gas %d, got %d", test.gasAfter, ctxt.gas)
			}
		})
	}
}

func TestCreateShanghaiInitCodeSize(t *testing.T) {
	maxInitCodeSize := uint64(49152)
	tests := map[string]struct {
		revision       tosca.Revision
		init_code_size uint64
		expecedErr     error
	}{
		"paris-0-running": {
			revision:       tosca.R11_Paris,
			init_code_size: 0,
		},
		"paris-1-running": {
			revision:       tosca.R11_Paris,
			init_code_size: 1,
		},
		"paris-2k-running": {
			revision:       tosca.R11_Paris,
			init_code_size: 2000,
		},
		"paris-max-1-running": {
			revision:       tosca.R11_Paris,
			init_code_size: maxInitCodeSize - 1,
		},
		"paris-max-running": {
			revision:       tosca.R11_Paris,
			init_code_size: maxInitCodeSize,
		},
		"paris-max+1-running": {
			revision:       tosca.R11_Paris,
			init_code_size: maxInitCodeSize + 1,
		},
		"paris-100k-running": {
			revision:       tosca.R11_Paris,
			init_code_size: 100000,
		},
		"paris-maxuint64-running": {
			revision:       tosca.R11_Paris,
			init_code_size: math.MaxUint64,
			expecedErr:     errOverflow,
		},

		"shanghai-0-running": {
			revision:       tosca.R12_Shanghai,
			init_code_size: 0,
		},
		"shanghai-1-running": {
			revision:       tosca.R12_Shanghai,
			init_code_size: 1,
		},
		"shanghai-2k-running": {
			revision:       tosca.R12_Shanghai,
			init_code_size: 2000,
		},
		"shanghai-max-1-running": {
			revision:       tosca.R12_Shanghai,
			init_code_size: maxInitCodeSize - 1,
		},
		"shanghai-max-running": {
			revision:       tosca.R12_Shanghai,
			init_code_size: maxInitCodeSize,
		},
		"shanghai-max+1-running": {
			revision:       tosca.R12_Shanghai,
			init_code_size: maxInitCodeSize + 1,
			expecedErr:     errInitCodeTooLarge,
		},
		"shanghai-100k-running": {
			revision:       tosca.R12_Shanghai,
			init_code_size: 100000,
			expecedErr:     errInitCodeTooLarge,
		},
		"shanghai-maxuint64-running": {
			revision:       tosca.R12_Shanghai,
			init_code_size: math.MaxUint64,
			expecedErr:     errOverflow,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			runContext := tosca.NewMockRunContext(ctrl)

			source := tosca.Address{1}
			ctxt := context{
				params: tosca.Parameters{
					BlockParameters: tosca.BlockParameters{
						Revision: test.revision,
					},
					Recipient: source,
				},
				context: runContext,
				stack:   NewStack(),
				memory:  NewMemory(),
				gas:     50000,
			}

			// Prepare stack arguments.
			ctxt.stack.push(uint256.NewInt(test.init_code_size))
			ctxt.stack.push(uint256.NewInt(0))
			ctxt.stack.push(uint256.NewInt(0))

			if test.expecedErr == nil {
				runContext.EXPECT().Call(tosca.Create, gomock.Any()).Return(tosca.CallResult{}, nil)
			}

			err := opCreate(&ctxt)
			if want, got := test.expecedErr, err; want != got {
				t.Fatalf("unexpected return, wanted %v, got %v", want, got)
			}
		})
	}
}

func TestCreateShanghaiDeploymentCost(t *testing.T) {
	tests := []struct {
		revision     tosca.Revision
		initCodeSize uint64
	}{
		// gas cost from evm.codes
		{tosca.R11_Paris, 0},
		{tosca.R11_Paris, 1},
		{tosca.R11_Paris, 2000},
		{tosca.R11_Paris, 49152},

		{tosca.R12_Shanghai, 0},
		{tosca.R12_Shanghai, 1},
		{tosca.R12_Shanghai, 2000},
		{tosca.R12_Shanghai, 49152},
	}

	dynamicCost := func(revision tosca.Revision, size uint64) uint64 {
		words := tosca.SizeInWords(size)
		// prevent overflow just like geth does
		if size > maxMemoryExpansionSize {
			return math.MaxInt64
		}
		memoryExpansionCost := (words*words)/512 + 3*words
		if revision >= tosca.R12_Shanghai {
			return 2*words + memoryExpansionCost
		}
		return memoryExpansionCost
	}

	for _, test := range tests {
		ctrl := gomock.NewController(t)
		runContext := tosca.NewMockRunContext(ctrl)

		cost := dynamicCost(test.revision, test.initCodeSize)

		source := tosca.Address{1}
		ctxt := context{
			params: tosca.Parameters{
				BlockParameters: tosca.BlockParameters{
					Revision: test.revision,
				},
				Recipient: source,
			},
			context: runContext,
			stack:   NewStack(),
			memory:  NewMemory(),
			gas:     tosca.Gas(cost),
		}

		// Prepare stack arguments.
		ctxt.stack.push(uint256.NewInt(test.initCodeSize))
		ctxt.stack.push(uint256.NewInt(0))
		ctxt.stack.push(uint256.NewInt(0))

		runContext.EXPECT().Call(tosca.Create, gomock.Any()).Return(tosca.CallResult{}, nil)

		err := opCreate(&ctxt)
		if err != nil {
			t.Errorf("opCreate failed: %v", err)
		}
		if ctxt.gas != 0 {
			t.Errorf("unexpected gas cost, wanted %d, got %d", cost, cost-uint64(ctxt.gas))
		}
	}
}

func TestTransientStorageOperations(t *testing.T) {
	tests := map[string]struct {
		op       func(*context) error
		setup    func(*tosca.MockRunContext)
		stackPtr int
		revision tosca.Revision
		err      error
	}{
		"tload-regular": {
			op: opTload,
			setup: func(runContext *tosca.MockRunContext) {
				runContext.EXPECT().GetTransientStorage(gomock.Any(), gomock.Any()).Return(tosca.Word{})
			},
			stackPtr: 1,
			revision: tosca.R13_Cancun,
		},
		"tload-old-revision": {
			op:       opTload,
			setup:    func(runContext *tosca.MockRunContext) {},
			stackPtr: 1,
			revision: tosca.R11_Paris,
			err:      errInvalidRevision,
		},
		"tstore-regular": {
			op: opTstore,
			setup: func(runContext *tosca.MockRunContext) {
				runContext.EXPECT().SetTransientStorage(gomock.Any(), gomock.Any(), gomock.Any())
			},
			stackPtr: 2,
			revision: tosca.R13_Cancun,
		},
		"tstore-old-revision": {
			op:       opTstore,
			setup:    func(runContext *tosca.MockRunContext) {},
			stackPtr: 2,
			revision: tosca.R11_Paris,
			err:      errInvalidRevision,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			ctxt := context{
				params: tosca.Parameters{
					BlockParameters: tosca.BlockParameters{
						Revision: test.revision,
					},
					Recipient: tosca.Address{1},
				},
				stack: NewStack(),
			}
			runContext := tosca.NewMockRunContext(ctrl)
			test.setup(runContext)
			ctxt.context = runContext
			ctxt.stack.stackPointer = test.stackPtr

			err := test.op(&ctxt)
			if want, got := test.err, err; want != got {
				t.Fatalf("unexpected return, wanted %v, got %v", want, got)
			}
		})
	}
}

func TestExpansionCostOverflow(t *testing.T) {
	memTestValues := []uint64{
		maxMemoryExpansionSize,
		maxMemoryExpansionSize + 1,
		math.MaxUint64,
	}

	tests := map[string]struct {
		op         func(*context) error
		stackSize  int
		memIndexes []int
		setup      func(*tosca.MockRunContext)
	}{
		"mcopy": {
			op:         opMcopy,
			stackSize:  3,
			memIndexes: []int{0, 1, 2},
			setup:      func(runContext *tosca.MockRunContext) {},
		},
		"calldatacopy": {
			op:         opCallDataCopy,
			stackSize:  3,
			memIndexes: []int{0, 2},
			setup:      func(runContext *tosca.MockRunContext) {},
		},
		"codecopy": {
			op:         opCodeCopy,
			stackSize:  3,
			memIndexes: []int{0, 2},
			setup:      func(runContext *tosca.MockRunContext) {},
		},
		"extcodecopy": {
			op:         opExtCodeCopy,
			stackSize:  4,
			memIndexes: []int{0, 2},
			setup: func(runContext *tosca.MockRunContext) {
				runContext.EXPECT().AccessAccount(gomock.Any()).AnyTimes().Return(tosca.WarmAccess)
				runContext.EXPECT().GetCode(gomock.Any()).AnyTimes().Return([]byte{0x01, 0x02, 0x03, 0x04})
			},
		},
	}

	for name, test := range tests {
		for _, memIndex := range test.memIndexes {
			for _, memValue := range memTestValues {
				t.Run(fmt.Sprintf("%v_i:%v_v:%v", name, memIndex, memValue), func(t *testing.T) {
					ctrl := gomock.NewController(t)
					runContext := tosca.NewMockRunContext(ctrl)
					test.setup(runContext)

					ctxt := context{
						params: tosca.Parameters{
							BlockParameters: tosca.BlockParameters{
								Revision: tosca.R13_Cancun,
							},
						},
						stack:   NewStack(),
						memory:  NewMemory(),
						context: runContext,
						gas:     12884901899,
					}
					ctxt.stack.stackPointer = test.stackSize
					ctxt.stack.data[memIndex].Set(uint256.NewInt(memValue))
					for i := range test.memIndexes {
						if i != memIndex {
							ctxt.stack.data[i].Set(uint256.NewInt(1))
						}
					}

					err := test.op(&ctxt)
					// FIXME: can we narrow the error? test says overflow, it manifests an out of gas error
					if err == nil {
						t.Fatalf("expected error, got nil")
					}
				})
			}
		}
	}
}

func TestGetAccessCost_RespondsWithProperGasPrice(t *testing.T) {
	if want, got := tosca.Gas(100), getAccessCost(tosca.WarmAccess); want != got {
		t.Errorf("unexpected gas cost, wanted %d, got %d", want, got)
	}
	if want, got := tosca.Gas(2600), getAccessCost(tosca.ColdAccess); want != got {
		t.Errorf("unexpected gas cost, wanted %d, got %d", want, got)
	}
}

func TestCall_ChargesNothingForColdAccessBeforeBerlin(t *testing.T) {

	ctrl := gomock.NewController(t)
	runContext := tosca.NewMockRunContext(ctrl)
	runContext.EXPECT().Call(tosca.Call, gomock.Any()).Return(tosca.CallResult{}, nil)
	ctxt := context{
		params: tosca.Parameters{
			BlockParameters: tosca.BlockParameters{
				Revision: tosca.R07_Istanbul,
			},
		},
		stack:   NewStack(),
		memory:  NewMemory(),
		context: runContext,
		gas:     0,
	}

	ctxt.stack.stackPointer = 8

	err := genericCall(&ctxt, tosca.Call)
	if err != nil {
		t.Errorf("genericCall failed: %v", err)
	}
	if ctxt.gas != 0 {
		t.Errorf("unexpected gas cost, wanted 0, got %v", ctxt.gas)
	}
}

func TestCall_ChargesForAccessAfterBerlin(t *testing.T) {

	for _, accessStatus := range []tosca.AccessStatus{tosca.WarmAccess, tosca.ColdAccess} {

		ctrl := gomock.NewController(t)
		runContext := tosca.NewMockRunContext(ctrl)
		runContext.EXPECT().AccessAccount(gomock.Any()).Return(accessStatus)
		runContext.EXPECT().Call(tosca.Call, gomock.Any()).Return(tosca.CallResult{}, nil)
		delta := tosca.Gas(1)
		ctxt := context{
			params: tosca.Parameters{
				BlockParameters: tosca.BlockParameters{
					Revision: tosca.R09_Berlin,
				},
			},
			stack:   NewStack(),
			memory:  NewMemory(),
			context: runContext,
			gas:     2600 + delta,
		}
		ctxt.stack.stackPointer = 8

		err := genericCall(&ctxt, tosca.Call)
		if err != nil {
			t.Errorf("genericCall failed: %v", err)
		}

		want := tosca.Gas(delta)
		if accessStatus == tosca.WarmAccess {
			want = 2500 + delta
		}
		if ctxt.gas != want {
			t.Errorf("unexpected gas cost, wanted %v, got %v", want, ctxt.gas)
		}
	}
}

func TestSelfDestruct_Refund(t *testing.T) {
	tests := map[string]struct {
		destructed bool
		revision   tosca.Revision
		refund     tosca.Gas
	}{
		"istanbul": {
			revision: tosca.R07_Istanbul,
		},
		"berlin-first-destructed": {
			destructed: true,
			revision:   tosca.R09_Berlin,
			refund:     24_000,
		},
		"berlin-not-first-destructed": {
			revision: tosca.R09_Berlin,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			refund := selfDestructRefund(test.destructed, test.revision)
			if refund != test.refund {
				t.Errorf("unexpected refund, wanted %d, got %d", test.refund, refund)
			}
		})
	}
}

func TestSelfDestruct_NewAccountCost(t *testing.T) {

	tests := map[string]struct {
		beneficiaryExists bool
		balance           tosca.Value
		cost              tosca.Gas
	}{
		"account exists no balance": {
			beneficiaryExists: true,
			balance:           tosca.Value{},
			cost:              0,
		},
		"account exists with balance": {
			beneficiaryExists: true,
			balance:           tosca.Value{1},
			cost:              0,
		},
		"new account without balance": {
			beneficiaryExists: false,
			balance:           tosca.Value{},
			cost:              0,
		},
		"new account with balance": {
			beneficiaryExists: false,
			balance:           tosca.Value{1},
			cost:              25_000,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			cost := selfDestructNewAccountCost(test.beneficiaryExists, test.balance)
			if cost != test.cost {
				t.Errorf("unexpected gas, wanted %d, got %d", test.cost, cost)
			}
		})
	}
}

func TestSelfDestruct_ExistingAccountToNewBeneficiary(t *testing.T) {
	// This tests produces the combination of context calls/results for the maximum dynamic gas cost possible.

	beneficiaryAddress := tosca.Address{1}
	selfAddress := tosca.Address{2}
	// added to gas to ensure operation is not simply setting gas to zero.
	gasDelta := tosca.Gas(1)

	ctrl := gomock.NewController(t)
	runContext := tosca.NewMockRunContext(ctrl)
	runContext.EXPECT().AccessAccount(beneficiaryAddress).Return(tosca.ColdAccess)
	runContext.EXPECT().AccountExists(beneficiaryAddress).Return(false)
	runContext.EXPECT().GetBalance(selfAddress).Return(tosca.Value{1})
	runContext.EXPECT().SelfDestruct(selfAddress, beneficiaryAddress).Return(true)

	ctxt := context{
		params: tosca.Parameters{
			BlockParameters: tosca.BlockParameters{
				Revision: tosca.R13_Cancun,
			},
			Recipient: selfAddress,
		},
		stack:   NewStack(),
		memory:  NewMemory(),
		context: runContext,
		// 25_000 for new account, 2_600 for beneficiary access
		gas: 27_600 + gasDelta,
	}
	ctxt.stack.push(new(uint256.Int).SetBytes(beneficiaryAddress[:]))

	status, err := opSelfdestruct(&ctxt)
	if err != nil {
		t.Fatalf("unexpected error, got %v", err)
	}
	if want, got := statusSelfDestructed, status; want != got {
		t.Fatalf("unexpected status, wanted %v, got %v", want, got)
	}
	if ctxt.gas != gasDelta {
		t.Errorf("unexpected remaining gas, wanted %v, got %d", gasDelta, ctxt.gas)
	}
}

func TestComputeCodeSizeCost(t *testing.T) {
	if cost, err := computeCodeSizeCost(24576*2 + 1); err == nil || cost != 0 {
		t.Errorf("check should have failed with size 49153 but did not. err %v, cost %v", err, cost)
	}
	if cost, err := computeCodeSizeCost(24576 * 2); err != nil || cost != 3072 {
		t.Errorf("should not have failed with size 49152, err %v, cost %v", err, cost)
	}
}

func TestGenericCreate_MaxInitCodeSizeIsNotCheckedBeforeShanghai(t *testing.T) {

	tests := []struct {
		revision      tosca.Revision
		expectedError error
	}{
		{tosca.R11_Paris, nil},
		{tosca.R12_Shanghai, errInitCodeTooLarge},
	}

	for _, test := range tests {
		t.Run(test.revision.String(), func(t *testing.T) {

			runContext := tosca.NewMockRunContext(gomock.NewController(t))
			runContext.EXPECT().Call(tosca.Create, gomock.Any()).Return(tosca.CallResult{}, nil).AnyTimes()

			ctxt := context{
				params: tosca.Parameters{
					BlockParameters: tosca.BlockParameters{
						Revision: test.revision,
					},
					Recipient: tosca.Address{1},
				},
				context: runContext,
				stack:   NewStack(),
				memory:  NewMemory(),
				gas:     50000,
			}

			ctxt.stack.push(uint256.NewInt(49153)) // size
			ctxt.stack.push(uint256.NewInt(0))     // offset
			ctxt.stack.push(uint256.NewInt(0))     // value

			err := genericCreate(&ctxt, tosca.Create)

			if err != test.expectedError {
				t.Errorf("unexpected status after call, wanted %v, got %v", test.expectedError, err)
			}

		})
	}
}

func TestOpEndWithResult_ReturnsExpectedState(t *testing.T) {
	c := getEmptyContext()
	c.stack.push(uint256.NewInt(1))
	c.stack.push(uint256.NewInt(1))
	c.memory.store = []byte{0x1, 0xff, 0x2}

	err := opEndWithResult(&c)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !bytes.Equal(c.returnData, []byte{0xff}) {
		t.Errorf("unexpected return data, wanted %v, got %v", []byte{0x1}, c.returnData)
	}
}

func TestOpEndWithResult_ReportOverflow(t *testing.T) {
	overflow64 := new(uint256.Int).Add(uint256.NewInt(math.MaxUint64), uint256.NewInt(math.MaxUint64))
	c := getEmptyContext()
	c.stack.push(overflow64)
	c.stack.push(overflow64)
	c.memory.store = []byte{0x1, 0xff, 0x2}
	err := opEndWithResult(&c)
	if err != errOverflow {
		t.Fatalf("should have produced overflow error, instead got: %v", err)
	}
}

func TestInstructions_EIP2929_staticGasCostIsZero(t *testing.T) {
	ops := []OpCode{BALANCE, EXTCODECOPY, EXTCODEHASH, EXTCODESIZE, CALL, CALLCODE, DELEGATECALL, STATICCALL}
	for _, op := range ops {
		if getBerlinGasPriceInternal(op) != 0 {
			t.Errorf("expected zero gas cost for %v", op)
		}
	}
}

func TestInstructions_EIP2929_dynamicGasCostReportsOutOfGas(t *testing.T) {
	type accessCost struct {
		warm tosca.Gas
		cold tosca.Gas
	}

	var eip2929AccessCost = newOpCodePropertyMap(func(op OpCode) accessCost {
		switch op {
		case SLOAD:
			return accessCost{warm: 100, cold: 2100}
		case SSTORE:
			return accessCost{warm: 100, cold: 2100 + 100}
		}
		return accessCost{warm: 100, cold: 2600}
	})

	tests := map[OpCode]func(*context) error{
		BALANCE:      opBalance,
		EXTCODECOPY:  opExtCodeCopy,
		EXTCODEHASH:  opExtcodehash,
		EXTCODESIZE:  opExtcodesize,
		CALL:         opCall,
		CALLCODE:     opCallCode,
		DELEGATECALL: opDelegateCall,
		STATICCALL:   opStaticCall,
		SLOAD:        opSload,
	}

	for op, implementation := range tests {
		for revision := tosca.R09_Berlin; revision <= newestSupportedRevision; revision++ {
			for _, access := range []tosca.AccessStatus{tosca.WarmAccess, tosca.ColdAccess} {
				t.Run(fmt.Sprintf("%v/%v/%v", op, revision, access), func(t *testing.T) {
					ctxt := context{
						params: tosca.Parameters{
							BlockParameters: tosca.BlockParameters{
								Revision: revision,
							},
						},
						stack:  NewStack(),
						memory: NewMemory(),
					}

					accessCosts := eip2929AccessCost.get(op)

					ctxt.gas = accessCosts.warm - 1
					if access == tosca.ColdAccess {
						ctxt.gas = accessCosts.cold - 1
					}
					mockRunContext := tosca.NewMockRunContext(gomock.NewController(t))
					mockRunContext.EXPECT().AccessStorage(gomock.Any(), gomock.Any()).Return(access).AnyTimes()
					mockRunContext.EXPECT().AccessAccount(gomock.Any()).Return(access).AnyTimes()
					ctxt.context = mockRunContext
					ctxt.stack.stackPointer = 7

					err := implementation(&ctxt)
					if err != errOutOfGas {
						t.Errorf("unexpected error: %v", err)
					}
				})
			}
		}
	}
}

func TestInstructions_EIP2929_SSTOREReportsOutOfGas(t *testing.T) {
	// SSTORE needs to be tested on its own because it demands that at least 2300 gas are available.
	// Hence we cannot take the same testing approach as for the other operations in EIP-2929.

	testGasValues := []tosca.Gas{
		2300, //< SSTORE demands at least 2300 gas to be available
		2301, //< not enough to afford StorageAdded, StorageModified, or StorageDeleted.
	}

	// dynamic gas check can only fail for the following storage status values
	failsForDynamicGas := []tosca.StorageStatus{tosca.StorageAdded, tosca.StorageModified, tosca.StorageDeleted}

	for _, availableGas := range testGasValues {
		for _, storageStatus := range failsForDynamicGas {
			for revision := tosca.R09_Berlin; revision <= newestSupportedRevision; revision++ {
				for _, access := range []tosca.AccessStatus{tosca.WarmAccess, tosca.ColdAccess} {
					t.Run(fmt.Sprintf("%v/%v/%v/%v", SSTORE, revision, access, storageStatus), func(t *testing.T) {

						ctxt := context{
							params: tosca.Parameters{
								BlockParameters: tosca.BlockParameters{
									Revision: revision,
								},
							},
							stack: NewStack(),
						}
						ctxt.gas = availableGas
						mockRunContext := tosca.NewMockRunContext(gomock.NewController(t))
						mockRunContext.EXPECT().AccessStorage(gomock.Any(), gomock.Any()).Return(access).AnyTimes()
						mockRunContext.EXPECT().SetStorage(gomock.Any(), gomock.Any(), gomock.Any()).Return(storageStatus).AnyTimes()
						ctxt.context = mockRunContext
						ctxt.stack.push(uint256.NewInt(1))
						ctxt.stack.push(uint256.NewInt(1))

						err := opSstore(&ctxt)
						if err != errOutOfGas {
							t.Errorf("unexpected error: %v", err)
						}
					})
				}
			}
		}
	}
}

func TestInstructions_JumpOpsCheckJUMPDEST(t *testing.T) {
	tests := map[OpCode]struct {
		implementation func(*context) error
		stack          []uint64
	}{
		JUMP: {
			implementation: opJump,
			stack:          []uint64{1},
		},
		JUMPI: {
			implementation: opJumpi,
			stack:          []uint64{1, 1},
		},
		SWAP2_SWAP1_POP_JUMP: {
			implementation: opSwap2_Swap1_Pop_Jump,
			stack:          []uint64{1, 1, 1},
		},
		POP_JUMP: {
			implementation: opPop_Jump,
			stack:          []uint64{1, 1},
		},
		PUSH2_JUMP: {
			implementation: opPush2_Jump,
			stack:          []uint64{1},
		},
		PUSH2_JUMPI: {
			implementation: opPush2_Jumpi,
			stack:          []uint64{1},
		},
		ISZERO_PUSH2_JUMPI: {
			implementation: opIsZero_Push2_Jumpi,
			stack:          []uint64{0},
		},
	}

	// test that all jump instructions are tested
	for _, op := range allOpCodesWhere(isJump) {
		if _, ok := tests[op]; !ok {
			t.Fatalf("missing test for jump instruction %v", op)
		}
	}

	for op, test := range tests {
		t.Run(op.String(), func(t *testing.T) {
			ctxt := getEmptyContext()
			ctxt.code = Code{{op, 0}}
			for _, v := range test.stack {
				ctxt.stack.push(uint256.NewInt(v))
			}

			err := test.implementation(&ctxt)
			if want, got := errInvalidJump, err; want != got {
				t.Fatalf("unexpected error, wanted %v, got %v", want, got)
			}
		})
	}
}

func TestInstructions_ConditionalJumpOpsIgnoreDestinationWhenJumpNotTaken(t *testing.T) {
	tests := map[OpCode]struct {
		implementation func(*context) error
		stack          []uint64
	}{
		JUMPI: {
			implementation: opJumpi,
			stack:          []uint64{0, 1},
		},
		PUSH2_JUMPI: {
			implementation: opPush2_Jumpi,
			stack:          []uint64{0},
		},
		ISZERO_PUSH2_JUMPI: {
			implementation: opIsZero_Push2_Jumpi,
			stack:          []uint64{1},
		},
	}

	for op, test := range tests {
		t.Run(op.String(), func(t *testing.T) {
			ctxt := getEmptyContext()
			ctxt.code = Code{{op, 0}}
			for _, v := range test.stack {
				ctxt.stack.push(uint256.NewInt(v))
			}

			err := test.implementation(&ctxt)
			if want, got := error(nil), err; want != got {
				t.Fatalf("unexpected error, wanted %v, got %v", want, got)
			}
		})
	}
}

func TestGetData(t *testing.T) {

	tests := map[string]struct {
		data     []byte
		start    uint64
		size     uint64
		expected []byte
	}{
		"AddsRightPadding": {
			data:     []byte{0xFF},
			start:    0,
			size:     2,
			expected: []byte{0xff, 0x0},
		},
		"ReadsBeyondLimitYieldZeroes": {
			data:     []byte{0xFF, 0x1},
			start:    12,
			size:     2,
			expected: []byte{0x0, 0x0},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			res := getData(test.data, test.start, test.size)
			if want, got := test.size, uint64(len(res)); want != got {
				t.Errorf("unexpected data length, wanted %d, got %d", want, got)
			}
			if want, got := test.expected, res; !bytes.Equal(want, got) {
				t.Errorf("unexpected data, wanted %v, got %v", want, got)
			}
		})
	}
}
