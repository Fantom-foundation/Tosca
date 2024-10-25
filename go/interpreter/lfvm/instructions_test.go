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
	"crypto/rand"
	"fmt"
	"math"
	"slices"
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
	runContext.EXPECT().GetNonce(target).Return(uint64(1))
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

	err := genericCreate(&ctxt, tosca.Create)
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

func TestBlobHash_PushesCorrectValueOnStack(t *testing.T) {
	hash := tosca.Hash{1}

	tests := map[string]struct {
		setup func(*tosca.Parameters, *stack)
		want  tosca.Hash
	}{
		"regular": {
			setup: func(params *tosca.Parameters, stack *stack) {
				stack.push(uint256.NewInt(0))
				params.BlobHashes = []tosca.Hash{hash}
			},
			want: hash,
		},
		"no-hashes": {
			setup: func(params *tosca.Parameters, stack *stack) {
				stack.push(uint256.NewInt(0))
			},
			want: tosca.Hash{},
		},
		"target-non-existent": {
			setup: func(params *tosca.Parameters, stack *stack) {
				stack.push(uint256.NewInt(1))
			},
			want: tosca.Hash{},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {

			ctxt := context{
				stack: NewStack(),
			}
			ctxt.params.Revision = tosca.R13_Cancun
			test.setup(&ctxt.params, ctxt.stack)

			err := opBlobHash(&ctxt)
			if err != nil {
				t.Fatalf("unexpected return: %v", err)
			}
			if want, got := test.want, ctxt.stack.data[0]; tosca.Hash(got.Bytes32()) != want {
				t.Fatalf("unexpected value on top of stack, wanted %v, got %v", want, got)
			}
		})
	}
}

func TestBlobBaseFee_ReturnsErrorWhenCalledWithUnsupportedRevision(t *testing.T) {
	ctxt := context{
		stack: NewStack(),
	}
	ctxt.params.Revision = tosca.R12_Shanghai

	err := opBlobBaseFee(&ctxt)
	if want, got := errInvalidRevision, err; want != got {
		t.Fatalf("unexpected return, wanted %v, got %v", want, got)
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

			err := genericCreate(&ctxt, tosca.Create)
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

		err := genericCreate(&ctxt, tosca.Create)
		if err != nil {
			t.Errorf("opCreate failed: %v", err)
		}
		if ctxt.gas != 0 {
			t.Errorf("unexpected gas cost, wanted %d, got %d", cost, cost-uint64(ctxt.gas))
		}
	}
}

func TestTransientStorageOperations(t *testing.T) {
	address := tosca.Address{}
	_, _ = rand.Read(address[:])
	key := tosca.Key{}
	_, _ = rand.Read(key[:])
	value := tosca.Word{}
	_, _ = rand.Read(value[:])

	tests := map[string]struct {
		op       func(*context) error
		setup    func(*tosca.MockRunContext)
		stack    []uint256.Int
		revision tosca.Revision
		err      error
	}{
		"tload-regular": {
			op: opTload,
			setup: func(runContext *tosca.MockRunContext) {
				runContext.EXPECT().GetTransientStorage(address, key).Return(tosca.Word{})
			},
			stack: []uint256.Int{
				*new(uint256.Int).SetBytes(key[:]),
			},
			revision: tosca.R13_Cancun,
		},
		"tload-old-revision": {
			op:       opTload,
			revision: tosca.R11_Paris,
			err:      errInvalidRevision,
		},
		"tstore-regular": {
			op: opTstore,
			setup: func(runContext *tosca.MockRunContext) {
				runContext.EXPECT().SetTransientStorage(address, key, value)
			},
			stack: []uint256.Int{
				*new(uint256.Int).SetBytes(key[:]),
				*new(uint256.Int).SetBytes(value[:]),
			},
			revision: tosca.R13_Cancun,
		},
		"tstore-old-revision": {
			op:       opTstore,
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
			if test.setup != nil {
				runContext := tosca.NewMockRunContext(ctrl)
				test.setup(runContext)
				ctxt.context = runContext
			}
			ctxt.stack = fillStack(test.stack...)
			ctxt.params.Recipient = address

			err := test.op(&ctxt)
			if want, got := test.err, err; want != got {
				t.Fatalf("unexpected return, wanted %v, got %v", want, got)
			}
		})
	}
}

func TestGenericDataCopy_CopiesDataIntoMemoryAndPadsExcessWithZeroes(t *testing.T) {

	testBuffer := []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}

	ctxt := getEmptyContext()
	ctxt.stack = fillStack(
		*uint256.NewInt(0),  // memory offset
		*uint256.NewInt(0),  // data offset
		*uint256.NewInt(15), // size
	)

	err := genericDataCopy(&ctxt, testBuffer)
	if err != nil {
		t.Fatalf("genericDataCopy failed: %v", err)
	}

	// 15 bytes read, expanded to 32 bytes: 0-14 are copied, 15-31 are zeroed.
	expected := []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 31: 0}
	if want, got := expected[:], ctxt.memory.store; !bytes.Equal(want, got) {
		t.Errorf("unexpected memory, wanted %v, got %v", want, got)
	}
}

func TestGenericDataCopy_ReturnsErrorOn(t *testing.T) {
	testBuffer := make([]byte, 1024)
	size := *uint256.NewInt(10)

	tests := map[string]struct {
		memOffset     uint256.Int
		gas           tosca.Gas
		expectedError error
	}{
		"not enough gas": {
			memOffset:     *uint256.NewInt(10),
			gas:           1,
			expectedError: errOutOfGas,
		},

		"expansion failure": {
			memOffset:     *uint256.NewInt(math.MaxUint64),
			gas:           1 << 32,
			expectedError: errOverflow,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {

			ctxt := getEmptyContext()
			ctxt.gas = test.gas
			ctxt.stack = fillStack(
				test.memOffset,
				*uint256.NewInt(0), // data offset
				size,
			)

			err := genericDataCopy(&ctxt, testBuffer)
			if want, got := test.expectedError, err; want != got {
				t.Fatalf("unexpected return, wanted %v, got %v", want, got)
			}
		})
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
	zero := *uint256.NewInt(0)
	one := *uint256.NewInt(1)
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

	ctxt.stack = fillStack(zero, one, zero, zero, zero, zero, zero, zero)

	err := genericCall(&ctxt, tosca.Call)
	if err != nil {
		t.Errorf("genericCall failed: %v", err)
	}
	if ctxt.gas != 0 {
		t.Errorf("unexpected gas cost, wanted 0, got %v", ctxt.gas)
	}
}

func TestCall_ChargesForAccessAfterBerlin(t *testing.T) {
	one := *uint256.NewInt(1)
	zero := *uint256.NewInt(0)
	for _, accessStatus := range []tosca.AccessStatus{tosca.WarmAccess, tosca.ColdAccess} {

		ctrl := gomock.NewController(t)
		runContext := tosca.NewMockRunContext(ctrl)
		runContext.EXPECT().AccessAccount(one.Bytes20()).Return(accessStatus)
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
		ctxt.stack = fillStack(zero, one, zero, zero, zero, zero, zero, zero)

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
		beneficiaryEmpty bool
		balance          tosca.Value
		cost             tosca.Gas
	}{
		"beneficiary empty no balance": {
			beneficiaryEmpty: true,
			balance:          tosca.Value{},
			cost:             0,
		},
		"beneficiary empty with balance": {
			beneficiaryEmpty: true,
			balance:          tosca.Value{1},
			cost:             25_000,
		},
		"beneficiary not empty without balance": {
			beneficiaryEmpty: false,
			balance:          tosca.Value{},
			cost:             0,
		},
		"beneficiary not empty with balance": {
			beneficiaryEmpty: false,
			balance:          tosca.Value{1},
			cost:             0,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			cost := selfDestructNewAccountCost(test.beneficiaryEmpty, test.balance)
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
	runContext.EXPECT().GetBalance(beneficiaryAddress)
	runContext.EXPECT().GetNonce(beneficiaryAddress)
	runContext.EXPECT().GetCodeSize(beneficiaryAddress)
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

func TestSelfDestruct_ProperlyReportsNotEnoughGas(t *testing.T) {
	for _, beneficiaryAccess := range []tosca.AccessStatus{tosca.WarmAccess, tosca.ColdAccess} {
		for _, accountEmpty := range []bool{true, false} {
			t.Run(fmt.Sprintf("beneficiaryAccess:%v_accountEmpty:%v", beneficiaryAccess, accountEmpty), func(t *testing.T) {
				beneficiaryAddress := tosca.Address{1}
				selfAddress := tosca.Address{2}

				ctrl := gomock.NewController(t)
				runContext := tosca.NewMockRunContext(ctrl)
				runContext.EXPECT().AccessAccount(beneficiaryAddress).Return(beneficiaryAccess)

				if accountEmpty {
					runContext.EXPECT().GetCodeSize(beneficiaryAddress).Return(1)
				} else {
					runContext.EXPECT().GetCodeSize(beneficiaryAddress).Return(0)
				}
				runContext.EXPECT().GetBalance(beneficiaryAddress).AnyTimes()
				runContext.EXPECT().GetNonce(beneficiaryAddress).AnyTimes()

				runContext.EXPECT().GetBalance(selfAddress).Return(tosca.Value{1})

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
				}
				if beneficiaryAccess == tosca.ColdAccess {
					ctxt.gas += 2600
				}
				if !accountEmpty {
					ctxt.gas += 25000
				}
				ctxt.gas -= 1

				ctxt.stack.push(new(uint256.Int).SetBytes(beneficiaryAddress[:]))

				_, err := opSelfdestruct(&ctxt)
				if err != errOutOfGas {
					t.Fatalf("expected out of gas but got %v", err)
				}

			})
		}
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

func TestGenericCreate_ReportsErrors(t *testing.T) {
	one := uint256.NewInt(1)
	tests := map[string]struct {
		offset, size  uint256.Int
		kind          tosca.CallKind
		revision      tosca.Revision
		expectedError error
	}{
		"not enough gas for code size": {
			offset:        *one,
			size:          *uint256.NewInt(31),
			revision:      tosca.R12_Shanghai,
			kind:          tosca.Create,
			expectedError: errOutOfGas,
		},
		"gas not checked for max code size before shanghai": {
			offset:        *one,
			size:          *uint256.NewInt(31),
			revision:      tosca.R11_Paris,
			kind:          tosca.Create,
			expectedError: nil,
		},
		"not enough gas for create2 init code hashing": {
			offset:        *one,
			size:          *one,
			kind:          tosca.Create2,
			expectedError: errOutOfGas,
		},
		"does not charge init code hashing in create": {
			offset:        *one,
			size:          *one,
			kind:          tosca.Create,
			expectedError: nil,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			mockRunContext := tosca.NewMockRunContext(gomock.NewController(t))
			mockRunContext.EXPECT().Call(gomock.Any(), gomock.Any()).Return(tosca.CallResult{}, nil).AnyTimes()
			ctxt := getEmptyContext()
			ctxt.context = mockRunContext
			ctxt.params.Revision = test.revision
			ctxt.gas = 3

			ctxt.stack.push(uint256.NewInt(0)) // salt
			ctxt.stack.push(&test.size)
			ctxt.stack.push(&test.offset)
			ctxt.stack.push(uint256.NewInt(0)) // value

			err := genericCreate(&ctxt, test.kind)
			if err != test.expectedError {
				t.Errorf("unexpected err. wanted %v, got %v", test.expectedError, err)
			}
		})
	}
}

func TestGenericCreate_ResultIsWrittenToStack(t *testing.T) {
	CreatedAddress := tosca.Address([20]byte{19: 0x1})
	for _, success := range []bool{true, false} {
		runContext := tosca.NewMockRunContext(gomock.NewController(t))
		runContext.EXPECT().Call(gomock.Any(), gomock.Any()).Return(tosca.CallResult{Success: success, CreatedAddress: CreatedAddress}, nil)
		ctxt := getEmptyContext()
		ctxt.context = runContext
		ctxt.stack.push(uint256.NewInt(0))
		ctxt.stack.push(uint256.NewInt(0))
		ctxt.stack.push(uint256.NewInt(0))
		err := genericCreate(&ctxt, tosca.Create)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		want := uint256.NewInt(0)
		if success {
			want = new(uint256.Int).SetBytes(CreatedAddress[:])
		}
		if got := ctxt.stack.peek(); !want.Eq(got) {
			t.Errorf("unexpected return value, wanted %v, got %v", want, got)
		}
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

func TestInstructions_StorageOps_CallStorageContext(t *testing.T) {
	address := tosca.Address{}
	_, _ = rand.Read(address[:])
	key := tosca.Key{}
	_, _ = rand.Read(key[:])
	value := tosca.Word{}
	_, _ = rand.Read(value[:])

	tests := map[OpCode]struct {
		implementation func(*context) error
		stack          []uint256.Int
	}{
		SLOAD: {
			implementation: opSload,
			stack: []uint256.Int{
				*new(uint256.Int).SetBytes(key[:]),
			},
		},
		SSTORE: {
			implementation: opSstore,
			stack: []uint256.Int{
				*new(uint256.Int).SetBytes(key[:]),
				*new(uint256.Int).SetBytes(value[:]),
			},
		},
	}

	for op, test := range tests {
		t.Run(op.String(), func(t *testing.T) {
			forEachRevision(t, op, func(t *testing.T, revision tosca.Revision) {

				ctxt := getEmptyContext()
				ctxt.params.Recipient = address
				ctxt.params.Revision = revision
				ctxt.stack = fillStack(test.stack...)
				runContext := tosca.NewMockRunContext(gomock.NewController(t))
				if revision >= tosca.R09_Berlin {
					runContext.EXPECT().AccessStorage(address, key).Return(tosca.WarmAccess)
				}
				if op == SLOAD {
					runContext.EXPECT().GetStorage(address, key).Return(value)
				}
				if op == SSTORE {
					runContext.EXPECT().SetStorage(address, key, value)
				}
				ctxt.context = runContext

				err := test.implementation(&ctxt)
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}

				if op == SLOAD {
					if got := ctxt.stack.peek(); got.Cmp(new(uint256.Int).SetBytes(value[:])) != 0 {
						t.Errorf("unexpected return value, wanted %v, got %v", value, got)
					}
				}
			})
		})
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
	zero := *uint256.NewInt(0)
	one := *uint256.NewInt(1)
	maxUint256 := *uint256.NewInt(0).Sub(uint256.NewInt(0), uint256.NewInt(1))

	tests := map[OpCode]struct {
		implementation func(*context) error
		stack          []uint256.Int
	}{
		JUMPI: {
			implementation: opJumpi,
			// ignores destination, even if it would overflow
			stack: []uint256.Int{maxUint256, zero},
		},
		PUSH2_JUMPI: {
			implementation: opPush2_Jumpi,
			stack:          []uint256.Int{zero},
		},
		ISZERO_PUSH2_JUMPI: {
			implementation: opIsZero_Push2_Jumpi,
			stack:          []uint256.Int{one},
		},
	}

	for op, test := range tests {
		t.Run(op.String(), func(t *testing.T) {
			ctxt := getEmptyContext()
			ctxt.code = Code{{op, 0}}
			ctxt.stack = fillStack(test.stack...)

			err := test.implementation(&ctxt)
			if want, got := error(nil), err; want != got {
				t.Fatalf("unexpected error, wanted %v, got %v", want, got)
			}
		})
	}
}

func TestInstructions_JumpOpsReturnErrorWithJumpDestinationOutOfBounds(t *testing.T) {
	tests := map[OpCode]struct {
		implementation func(*context) error
		stack          []uint256.Int
	}{
		JUMP: {
			implementation: opJump,
			stack: []uint256.Int{
				*uint256.NewInt(math.MaxInt32 + 1),
			},
		},
		JUMPI: {
			implementation: opJumpi,
			stack: []uint256.Int{
				*uint256.NewInt(math.MaxInt32 + 1),
				*uint256.NewInt(1),
			},
		},
	}

	for op, test := range tests {
		t.Run(op.String(), func(t *testing.T) {
			ctxt := getEmptyContext()
			ctxt.code = Code{{op, 0}}
			ctxt.stack = fillStack(test.stack...)

			err := test.implementation(&ctxt)
			if want, got := errInvalidJump, err; want != got {
				t.Fatalf("unexpected error, wanted %v, got %v", want, got)
			}
		})
	}

}

func TestGetData(t *testing.T) {

	tests := map[string]struct {
		data           []byte
		offset         *uint256.Int
		size           uint64
		expectedResult []byte
	}{
		"returns slice in bounds": {
			data:           []byte{0x00, 0x1, 0x2, 0x3, 0xFF},
			offset:         uint256.NewInt(1),
			size:           3,
			expectedResult: []byte{0x1, 0x2, 0x3},
		},
		"returns empty slice when size is 0": {
			data:           []byte{},
			offset:         uint256.NewInt(0),
			size:           0,
			expectedResult: nil,
		},
		"adds zeroes right padding": {
			data:           []byte{0xFF},
			offset:         uint256.NewInt(0),
			size:           2,
			expectedResult: []byte{0xFF, 0x0},
		},
		"reads beyond limit yield zeroes": {
			data:           []byte{0xFF, 0x1},
			offset:         uint256.NewInt(12),
			size:           2,
			expectedResult: []byte{0x0, 0x0},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			res := getData(test.data, test.offset, test.size)
			if want, got := test.expectedResult, res; !bytes.Equal(want, got) {
				t.Errorf("unexpected data, wanted %v, got %v", want, got)
			}
		})
	}
}

func TestGenericCall_ProperlyReportsErrors(t *testing.T) {

	one := uint256.NewInt(1)
	u64overflow := new(uint256.Int).Add(uint256.NewInt(2), uint256.NewInt(math.MaxUint64))
	address := tosca.Address{1}

	tests := map[string]struct {
		// stack order
		retSize, retOffset, inSize, inOffset, value, provided_gas *uint256.Int
		gas                                                       tosca.Gas
		expectedError                                             error
	}{
		"input offset overflow": {
			// size needs to be one, otherwise offset is ignored.
			inSize:        one,
			inOffset:      u64overflow,
			expectedError: errOverflow,
		},
		"return Size overflow": {
			retSize:       u64overflow,
			expectedError: errOverflow,
		},
		"input memory too big": {
			inSize:        uint256.NewInt(maxMemoryExpansionSize + 1),
			expectedError: errMaxMemoryExpansionSize,
		},
		"not enough gas for output memory expansion": {
			retSize:       one,
			gas:           1,
			expectedError: errOutOfGas,
		},
		"not enough gas for access cost": {
			gas:           99,
			expectedError: errOutOfGas,
		},
		"not enough gas for value transfer": {
			value:         one,
			gas:           9099, // 9000 for value transfer, 100 for warm access cost
			expectedError: errOutOfGas,
		},
		"not enough gas for new account": {
			value:         one,
			gas:           33099, // 25000 for new account, 9000 for value transfer, 100 for warm access cost
			expectedError: errOutOfGas,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			runContext := tosca.NewMockRunContext(gomock.NewController(t))
			runContext.EXPECT().AccessAccount(address).Return(tosca.WarmAccess).AnyTimes()
			runContext.EXPECT().GetNonce(address).AnyTimes()
			runContext.EXPECT().GetBalance(address).AnyTimes()
			runContext.EXPECT().GetCodeSize(address).AnyTimes()

			ctxt := getEmptyContext()
			ctxt.context = runContext
			ctxt.params.Revision = tosca.R13_Cancun
			ctxt.gas = test.gas

			getValueOrZeroOf := func(i *uint256.Int) *uint256.Int {
				if i == nil {
					return uint256.NewInt(0)
				}
				return i
			}

			ctxt.stack.push(getValueOrZeroOf(test.retSize))
			ctxt.stack.push(getValueOrZeroOf(test.retOffset))
			ctxt.stack.push(getValueOrZeroOf(test.inSize))
			ctxt.stack.push(getValueOrZeroOf(test.inOffset))
			ctxt.stack.push(getValueOrZeroOf(test.value))
			ctxt.stack.push(uint256.NewInt(0).SetBytes20(address[:]))
			ctxt.stack.push(getValueOrZeroOf(test.provided_gas))

			err := genericCall(&ctxt, tosca.Call)

			if err != test.expectedError {
				t.Errorf("unexpected status after call, wanted %v, got %v", test.expectedError, err)
			}
		})
	}
}

func TestGenericCall_CallKindPropagatesStaticMode(t *testing.T) {
	zero := *uint256.NewInt(0)
	one := *uint256.NewInt(1)
	runContext := tosca.NewMockRunContext(gomock.NewController(t))
	runContext.EXPECT().Call(tosca.StaticCall, gomock.Any()).Return(tosca.CallResult{}, nil)
	ctxt := getEmptyContext()
	ctxt.context = runContext
	ctxt.params.Static = true
	ctxt.stack = fillStack(zero, one, zero, zero, zero, zero, zero)

	err := genericCall(&ctxt, tosca.Call)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestGenericCall_ResultIsWrittenToStack(t *testing.T) {
	zero := *uint256.NewInt(0)
	one := *uint256.NewInt(1)
	for _, success := range []bool{true, false} {
		runContext := tosca.NewMockRunContext(gomock.NewController(t))
		runContext.EXPECT().Call(tosca.Call, gomock.Any()).Return(tosca.CallResult{Success: success}, nil)
		ctxt := getEmptyContext()
		ctxt.context = runContext
		ctxt.stack = fillStack(zero, one, zero, zero, zero, zero, zero)
		_ = genericCall(&ctxt, tosca.Call)
		want := uint256.NewInt(0)
		if success {
			want = uint256.NewInt(1)
		}
		if got := ctxt.stack.data[0]; !want.Eq(&got) {
			t.Errorf("unexpected return value, wanted %v, got %v", want, got)
		}
	}
}

func TestGenericCall_HandlesBigProvidedGasValues(t *testing.T) {
	zero := *uint256.NewInt(0)
	gas := tosca.Gas(50_000) // value big enough to cover all gas costs
	tests := map[string]uint256.Int{
		"maxInt64-1": *uint256.NewInt(math.MaxInt64 - 1),
		"maxInt64":   *uint256.NewInt(math.MaxInt64),
		"maxInt64+1": *uint256.NewInt(math.MaxInt64 + 1),
		"maxUint64":  *uint256.NewInt(math.MaxUint64),
	}

	for name, providedGas := range tests {
		t.Run(name, func(t *testing.T) {
			nestedGas := tosca.Gas(gas - gas/64)
			runContext := tosca.NewMockRunContext(gomock.NewController(t))
			runContext.EXPECT().Call(tosca.Call, tosca.CallParameters{Gas: nestedGas}).Return(tosca.CallResult{}, nil)
			ctxt := context{gas: gas}
			ctxt.context = runContext
			ctxt.stack = fillStack(providedGas, zero, zero, zero, zero, zero, zero)

			err := genericCall(&ctxt, tosca.Call)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestGenericCall_ForwardsCallParamsDependingOnCallKind(t *testing.T) {

	zero := *uint256.NewInt(0)
	sender := tosca.Address{1}
	recipient := tosca.Address{2}
	value := tosca.Value{3}
	targetAddress := tosca.Address{4}
	targetAddressU256 := *new(uint256.Int).SetBytes20(targetAddress[:])

	tests := map[tosca.CallKind]struct {
		sender, recipient tosca.Address
		value             tosca.Value
	}{
		tosca.Call: {
			sender:    recipient,
			recipient: targetAddress,
		},
		tosca.StaticCall: {
			sender:    recipient,
			recipient: targetAddress,
		},
		tosca.DelegateCall: {
			sender:    sender,
			recipient: recipient,
			value:     value,
		},
		tosca.CallCode: {
			sender:    recipient,
			recipient: recipient,
		},
	}

	for kind, test := range tests {
		t.Run(kind.String(), func(t *testing.T) {

			runContext := tosca.NewMockRunContext(gomock.NewController(t))
			wantParams := tosca.CallParameters{
				Sender:      test.sender,
				Recipient:   test.recipient,
				Value:       test.value,
				CodeAddress: targetAddress,
			}
			runContext.EXPECT().Call(kind, wantParams).Return(tosca.CallResult{}, nil)
			ctxt := getEmptyContext()
			ctxt.context = runContext
			ctxt.params.Sender = sender
			ctxt.params.Recipient = recipient
			ctxt.params.Value = value

			ctxt.stack = fillStack(zero, targetAddressU256, zero, zero, zero, zero, zero)

			err := genericCall(&ctxt, kind)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestInstructions_ComparisonAndShiftOperations(t *testing.T) {

	zero := *uint256.NewInt(0)
	one := *uint256.NewInt(1)
	two := *uint256.NewInt(2)
	signedMinusOne := *uint256.NewInt(0).Sub(&zero, &one)
	signedMinusTwo := *uint256.NewInt(0).Sub(&zero, &two)
	u256 := *uint256.NewInt(256)
	u257 := *uint256.NewInt(257)

	tests := map[string]struct {
		opImplementation func(*context)
		stackInputs      *stack
		expectedOutput   uint256.Int
	}{
		"isZero/true": {
			opImplementation: opIszero,
			stackInputs:      fillStack(zero),
			expectedOutput:   one,
		},
		"isZero/false": {
			opImplementation: opIszero,
			stackInputs:      fillStack(one),
			expectedOutput:   zero,
		},
		"eq/true": {
			opImplementation: opEq,
			stackInputs:      fillStack(one, one),
			expectedOutput:   one,
		},
		"eq/false": {
			opImplementation: opEq,
			stackInputs:      fillStack(one, two),
			expectedOutput:   zero,
		},
		"lt/true": {
			opImplementation: opLt,
			stackInputs:      fillStack(one, two),
			expectedOutput:   one,
		},
		"lt/false": {
			opImplementation: opLt,
			stackInputs:      fillStack(one, one),
			expectedOutput:   zero,
		},
		"gt/true": {
			opImplementation: opGt,
			stackInputs:      fillStack(two, one),
			expectedOutput:   one,
		},
		"gt/false": {
			opImplementation: opGt,
			stackInputs:      fillStack(one, one),
			expectedOutput:   zero,
		},
		"slt/true": {
			opImplementation: opSlt,
			stackInputs:      fillStack(signedMinusOne, one),
			expectedOutput:   one,
		},
		"slt/false": {
			opImplementation: opSlt,
			stackInputs:      fillStack(one, one),
			expectedOutput:   zero,
		},
		"sgt/true": {
			opImplementation: opSgt,
			stackInputs:      fillStack(one, signedMinusOne),
			expectedOutput:   one,
		},
		"sgt/false": {
			opImplementation: opSgt,
			stackInputs:      fillStack(signedMinusOne, one),
			expectedOutput:   zero,
		},
		"shr/under256": {
			opImplementation: opShr,
			stackInputs:      fillStack(one, two),
			expectedOutput:   one,
		},
		"shr/over256": {
			opImplementation: opShr,
			stackInputs:      fillStack(u256, one),
			expectedOutput:   zero,
		},
		"shl/under256": {
			opImplementation: opShl,
			stackInputs:      fillStack(one, one),
			expectedOutput:   two,
		},
		"shl/over256": {
			opImplementation: opShl,
			stackInputs:      fillStack(u256, one),
			expectedOutput:   zero,
		},
		"sar/under256": {
			opImplementation: opSar,
			stackInputs:      fillStack(one, signedMinusTwo),
			expectedOutput:   signedMinusOne,
		},
		"sar/over256/signed": {
			opImplementation: opSar,
			stackInputs:      fillStack(u257, signedMinusOne),
			expectedOutput:   signedMinusOne,
		},
		"sar/over256/unsigned": {
			opImplementation: opSar,
			stackInputs:      fillStack(u257, one),
			expectedOutput:   zero,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			ctxt := context{
				stack: test.stackInputs,
			}

			test.opImplementation(&ctxt)
			result := ctxt.stack.pop()
			if result.Cmp(&test.expectedOutput) != 0 {
				t.Errorf("unexpected result, wanted %d, got %d", test.expectedOutput, result)
			}
		})
	}
}

func TestInstructions_OpExtCodeCopy_CallsContextAndCopiesCodeSlice(t *testing.T) {

	code := []byte{0x1, 0x2, 0x3, 0x4}
	address := tosca.Address{}
	_, _ = rand.Read(address[:])
	var offset uint64 = 1
	var size uint64 = 2

	runContext := tosca.NewMockRunContext(gomock.NewController(t))
	runContext.EXPECT().GetCode(address).Return(code)

	ctxt := getEmptyContext()
	ctxt.context = runContext

	ctxt.stack = fillStack(
		*new(uint256.Int).SetBytes(address[:]),
		*uint256.NewInt(0),      // memOffset
		*uint256.NewInt(offset), // codeOffset
		*uint256.NewInt(size),   // length
	)

	err := opExtCodeCopy(&ctxt)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if want, got := code[offset:offset+size], ctxt.memory.store[0:size]; !bytes.Equal(want, got) {
		t.Errorf("unexpected memory, wanted %v, got %v", want, got)
	}
}

func TestInstructions_opExtcodesize_CallsContextAndWritesResultInStack(t *testing.T) {

	address := tosca.Address{}
	_, _ = rand.Read(address[:])
	runContext := tosca.NewMockRunContext(gomock.NewController(t))
	runContext.EXPECT().GetCodeSize(address).Return(1234)

	ctxt := getEmptyContext()
	ctxt.context = runContext

	ctxt.stack.push(new(uint256.Int).SetBytes(address[:]))

	err := opExtcodesize(&ctxt)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if want, got := uint256.NewInt(1234), ctxt.stack.pop(); want.Cmp(got) != 0 {
		t.Errorf("unexpected result, wanted %v, got %v", want, got)
	}
}

func TestOpBlockhash(t *testing.T) {

	hash := tosca.Hash{}
	_, _ = rand.Read(hash[:])
	zeroHash := tosca.Hash{}
	u64overflow := new(uint256.Int).Add(uint256.NewInt(2), uint256.NewInt(math.MaxUint64))

	type testInput struct {
		currentBlockNumber   int64
		requestedBlockNumber *uint256.Int
	}

	tests := map[string]struct {
		expectedValue tosca.Hash
		inputs        map[string]testInput
	}{
		"produces zero if requested block number is older than available history": {
			expectedValue: zeroHash,
			inputs: map[string]testInput{
				"default": {
					currentBlockNumber: 1024, requestedBlockNumber: uint256.NewInt(36),
				},
			},
		},
		"produces zero if requested block number is newer than history": {
			expectedValue: zeroHash,
			inputs: map[string]testInput{
				"current is not included in history": {
					currentBlockNumber: 500, requestedBlockNumber: uint256.NewInt(500),
				},
				"and history has less than 256 elements": {
					currentBlockNumber: 35, requestedBlockNumber: uint256.NewInt(36),
				},
				"and request overflows uint64": {
					currentBlockNumber: 5000, requestedBlockNumber: u64overflow,
				},
			},
		},
		"produces existing hash if requested is in history range": {
			expectedValue: hash,
			inputs: map[string]testInput{
				"default": {currentBlockNumber: 5000, requestedBlockNumber: uint256.NewInt(4990)},
				"and history has less than 256 elements": {
					currentBlockNumber: 128, requestedBlockNumber: uint256.NewInt(16),
				},
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {

			for name, input := range test.inputs {
				t.Run(name, func(t *testing.T) {

					ctxt := getEmptyContext()
					ctxt.stack = fillStack(*input.requestedBlockNumber)
					ctxt.params.BlockNumber = input.currentBlockNumber

					runContext := tosca.NewMockRunContext(gomock.NewController(t))
					if hash == test.expectedValue {
						runContext.EXPECT().GetBlockHash(gomock.Any()).Return(hash).AnyTimes()
					}
					ctxt.context = runContext

					opBlockhash(&ctxt)

					if want, got := new(uint256.Int).SetBytes(test.expectedValue[:]), ctxt.stack.pop(); want.Cmp(got) != 0 {
						t.Errorf("unexpected result, wanted %v, got %v", want, got)
					}
				})
			}
		})
	}
}
func TestInstructions_ReturnDataCopy_ReturnsErrorOn(t *testing.T) {

	zero := *uint256.NewInt(0)
	one := *uint256.NewInt(1)
	maxUint64 := *uint256.NewInt(math.MaxUint64)
	uint64Overflow := *new(uint256.Int).Add(&maxUint64, uint256.NewInt(1))
	returnDataSize := uint64(10)

	tests := map[string]struct {
		stack []uint256.Int // memoryOffset, dataOffset, length
	}{
		"length overflow": {
			stack: []uint256.Int{zero, one, uint64Overflow},
		},
		"dataOffset overflow": {
			stack: []uint256.Int{zero, uint64Overflow, one},
		},
		"offset + length overflow": {
			stack: []uint256.Int{zero, maxUint64, one},
		},
		"offset + length greater than returnData": {
			stack: []uint256.Int{zero, zero, *uint256.NewInt(returnDataSize + 1)},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			ctxt := getEmptyContext()
			ctxt.stack = fillStack(test.stack...)
			ctxt.returnData = make([]byte, returnDataSize)

			err := opReturnDataCopy(&ctxt)
			if err != errOverflow {
				t.Fatalf("expected overflow error, got %v", err)
			}
		})
	}
}

func TestOpExp_ProducesCorrectResults(t *testing.T) {
	ctxt := context{gas: tosca.Gas(uint256.NewInt(8).ByteLen() * 50)}
	ctxt.stack = NewStack()
	ctxt.stack.push(uint256.NewInt(8)) // exponent
	ctxt.stack.push(uint256.NewInt(2)) // base
	err := opExp(&ctxt)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := uint256.NewInt(256)
	if got := ctxt.stack.pop(); got.Cmp(expected) != 0 {
		t.Errorf("unexpected result, wanted %v, got %v", expected, got)
	}
}

func TestOpExp_ReportsOutOfGas(t *testing.T) {
	ctxt := context{gas: 3}
	ctxt.stack = NewStack()
	ctxt.stack.push(uint256.NewInt(256)) // exponent
	ctxt.stack.push(uint256.NewInt(2))   // base
	err := opExp(&ctxt)
	if err != errOutOfGas {
		t.Errorf("expected out of gas error, got %v", err)
	}
}

func TestInstructions_Sha3_ReportsOutOfGas(t *testing.T) {
	tests := map[string]struct {
		size          uint64
		expectedError error
	}{
		"memory expansion": {
			size:          64,
			expectedError: errOutOfGas,
		},
		"dynamic gas price": {
			size:          1,
			expectedError: errOutOfGas,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			ctxt := context{gas: 3}
			ctxt.memory = NewMemory()
			ctxt.stack = NewStack()
			ctxt.stack.push(uint256.NewInt(test.size))
			ctxt.stack.push(uint256.NewInt(0))
			err := opSha3(&ctxt)
			if err != test.expectedError {
				t.Fatalf("unexpected error, wanted %v, got %v", test.expectedError, err)
			}
		})
	}
}

func TestInstructions_Sha3_WritesCorrectHashInStack(t *testing.T) {

	want := Keccak256([]byte{0})

	for _, withShaCache := range []bool{true, false} {
		t.Run(fmt.Sprintf("withShaCache:%v", withShaCache), func(t *testing.T) {
			ctxt := getEmptyContext()
			ctxt.withShaCache = withShaCache
			ctxt.stack.push(uint256.NewInt(1))
			ctxt.stack.push(uint256.NewInt(0))

			err := opSha3(&ctxt)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			got := ctxt.stack.pop()
			if !bytes.Equal(got.Bytes(), want[:]) {
				t.Errorf("unexpected hash wanted %x, got %x", want, got.Bytes())
			}
		})
	}
}

func TestOpExtCodeHash_WritesHashOnStackIfAccountExists(t *testing.T) {

	tests := map[string]struct {
		accountEmpty bool
	}{
		"account empty":     {accountEmpty: true},
		"account not empty": {accountEmpty: false},
	}

	hash := tosca.Hash{0x1, 0x2, 0x3}
	address := tosca.Address{0x1}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			ctxt := getEmptyContext()
			ctxt.stack = fillStack(*new(uint256.Int).SetBytes20(address[:]))

			runContext := tosca.NewMockRunContext(gomock.NewController(t))

			runContext.EXPECT().GetBalance(address).AnyTimes()
			runContext.EXPECT().GetNonce(address).AnyTimes()
			if test.accountEmpty {
				runContext.EXPECT().GetCodeSize(address).Return(0)
			} else {
				runContext.EXPECT().GetCodeSize(address).Return(1)
			}

			if !test.accountEmpty {
				runContext.EXPECT().GetCodeHash(address).Return(hash)
			}
			ctxt.context = runContext

			err := opExtcodehash(&ctxt)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			want := hash[:]
			if test.accountEmpty {
				want = []byte{}
			}
			if got := ctxt.stack.pop().Bytes(); !bytes.Equal(want, got) {
				t.Errorf("unexpected result, wanted %v, got %v", want, got)
			}
		})
	}
}

func TestInstructions_opLog(t *testing.T) {

	tests := map[string]struct {
		offset, size  *uint256.Int
		expectedError error
		reduceGas     uint64
	}{
		"returns error if memory expansion fails": {
			offset: uint256.NewInt(math.MaxUint64), size: uint256.NewInt(1),
			expectedError: errOverflow,
		},
		"returns error if word gas cost is more than available gas": {
			offset: uint256.NewInt(10), size: uint256.NewInt(128),
			reduceGas:     1,
			expectedError: errOutOfGas,
		},
		"calls emitLog with recipient, defined order of topics and memory copy": {
			offset: uint256.NewInt(1), size: uint256.NewInt(2),
		},
	}
	for name, test := range tests {
		for n := 0; n < 4; n++ {
			t.Run(fmt.Sprintf("%v/LOG%d", name, n), func(t *testing.T) {

				ctxt := getEmptyContext()
				for i := n - 1; i >= 0; i-- {
					ctxt.stack.push(uint256.NewInt(uint64(i)))
				}
				ctxt.stack.push(test.size)
				ctxt.stack.push(test.offset)
				ctxt.gas = tosca.Gas(test.size.Uint64()*8 - test.reduceGas)
				ctxt.params.Recipient = tosca.Address{1}
				// ignore the expansion error to focus on log operation
				// this expansion is done to remove expansion costs (if any) and
				// test word count cost only.
				memoryContents := []byte{0, 1, 2, 3}
				_ = ctxt.memory.set(uint256.NewInt(0), memoryContents, &context{gas: math.MaxInt64})

				if test.expectedError == nil {
					runContext := tosca.NewMockRunContext(gomock.NewController(t))
					runContext.EXPECT().EmitLog(gomock.Any()).Do(func(log tosca.Log) {
						if want, got := ctxt.params.Recipient, log.Address; want != got {
							t.Errorf("unexpected log address, wanted %v, got %v", want, got)
						}
						if want, got := n, len(log.Topics); want != got {
							t.Errorf("unexpected number of topics, wanted %v, got %v", want, got)
						}

						for i := n; i > n; i++ {
							if want, got := tosca.Hash(uint256.NewInt(uint64(i)).Bytes32()), log.Topics[i]; want != got {
								t.Errorf("unexpected topic #%d, wanted %v, got %v", i, want, got)
							}
						}
						from := test.offset.Uint64()
						to := from + test.size.Uint64()
						if want, got := memoryContents[from:to], log.Data; !slices.Equal(want, got) {
							t.Errorf("unexpected log data, wanted %v,got %v", want, got)
						}
					})
					ctxt.context = runContext
				}

				err := opLog(&ctxt, n)
				if want, got := test.expectedError, err; want != got {
					t.Fatalf("unexpected error, wanted %v, got %v", want, got)
				}
			})
		}
	}
}

func TestInstructions_MCopy_DoesNothingWithSizeZero(t *testing.T) {

	data := [1024]byte{}
	_, _ = rand.Read(data[:])

	ctxt := getEmptyContext()
	ctxt.params.Revision = tosca.R13_Cancun
	ctxt.stack = fillStack(
		*uint256.NewInt(2500), // destOffset
		*uint256.NewInt(137),  // offset
		*uint256.NewInt(0))    // size
	ctxt.gas = 0

	err := ctxt.memory.set(
		uint256.NewInt(0),
		data[:],
		&context{gas: 1 << 32},
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	err = opMcopy(&ctxt)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if want, got := data[:], ctxt.memory.store; !bytes.Equal(want, got) {
		t.Errorf("unexpected memory, wanted %v, got %v", want, got)
	}
}

func TestInstructions_MCopy_ReturnsErrorOnFailure(t *testing.T) {

	tests := map[string]struct {
		destOffset, offset, size uint64
		expectedError            error
		gasRemoved               uint64
	}{
		"returns error when failed read memory expansion": {
			destOffset: 0, offset: math.MaxUint64, size: 1,
			expectedError: errOverflow,
		},
		"returns error when failed write memory expansion": {
			destOffset: math.MaxUint64, offset: 0, size: 1,
			expectedError: errOverflow,
		},
		"returns error if gas is not enough for size": {
			destOffset: 0, offset: 0, size: 128,
			expectedError: errOutOfGas,
			gasRemoved:    1,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			ctxt := getEmptyContext()
			ctxt.params.Revision = tosca.R13_Cancun
			ctxt.stack = fillStack(
				*uint256.NewInt(test.destOffset),
				*uint256.NewInt(test.offset),
				*uint256.NewInt(test.size),
			)

			// ignore memory setup errors, to focus on the mcopy operation
			// expansion is done to accumulate memory cost and focus on the
			// word count gas cost.
			_ = ctxt.memory.expandMemory(test.destOffset, test.size, &context{gas: 1 << 32})
			_ = ctxt.memory.expandMemory(test.offset, test.size, &context{gas: 1 << 32})
			ctxt.gas = tosca.Gas(3*tosca.SizeInWords(test.size) - test.gasRemoved)

			err := opMcopy(&ctxt)
			if err != test.expectedError {
				t.Fatalf("unexpected error, wanted %v, got %v", test.expectedError, err)
			}
		})
	}
}

func TestInstructions_MCopy_CopiesOverlappingRanges(t *testing.T) {

	ctxt := getEmptyContext()
	ctxt.params.Revision = tosca.R13_Cancun
	ctxt.stack = fillStack(
		*uint256.NewInt(5),
		*uint256.NewInt(1),
		*uint256.NewInt(10))

	err := ctxt.memory.set(
		uint256.NewInt(0),
		[]byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9},
		&context{gas: 1 << 32},
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	ctxt.gas = tosca.Gas(3 * tosca.SizeInWords(10))

	err = opMcopy(&ctxt)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := [32]byte{0, 1, 2, 3, 4, 1, 2, 3, 4, 5, 6, 7, 8, 9}
	if want, got := expected[:], ctxt.memory.store; !bytes.Equal(want, got) {
		t.Errorf("unexpected memory, wanted %v, got %v", want, got)
	}
}

func TestInstructions_ReturnDataCopy_ReturnsOutOfGas(t *testing.T) {
	zero := *uint256.NewInt(0)
	ctxt := context{gas: 3, stack: fillStack(zero, zero, *uint256.NewInt(65))}
	err := opReturnDataCopy(&ctxt)
	if err != errOutOfGas {
		t.Fatalf("expected overflow error, got %v", err)
	}
}

////////////////////////////////////////////////////////////////////////////////
// Helper functions

// fillStack creates a new stack and pushes the given values onto it.
// function arguments interpret top of the stack as the rightmost argument.
func fillStack(values ...uint256.Int) *stack {
	s := NewStack()
	for i := len(values) - 1; i >= 0; i-- {
		s.push(&values[i])
	}
	return s
}
