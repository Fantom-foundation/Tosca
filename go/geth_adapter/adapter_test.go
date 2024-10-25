// Copyright (c) 2024 Fantom Foundation
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at fantom.foundation/bsl11.
//
// Change Date: 2028-4-16
//
// On the date above, in accordance with the Business Source License, use of
// this software will be governed by the GNU Lesser General Public License v3.

package geth_adapter

import (
	"bytes"
	"errors"
	"fmt"
	"math/big"
	"reflect"
	"slices"
	"testing"

	"github.com/Fantom-foundation/Tosca/go/tosca"
	gc "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/tracing"
	geth "github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/params"
	"github.com/holiman/uint256"
	"go.uber.org/mock/gomock"

	_ "github.com/Fantom-foundation/Tosca/go/interpreter/geth"
)

//go:generate mockgen -source adapter_test.go -destination adapter_test_mocks.go -package geth_adapter

type StateDb interface {
	geth.StateDB
}

func TestGethAdapter_RunContextAdapterImplementsRunContextInterface(t *testing.T) {
	var _ tosca.RunContext = &runContextAdapter{}
}

func TestRunContextAdapter_SetBalanceHasCorrectEffect(t *testing.T) {
	tests := []struct {
		before tosca.Value
		after  tosca.Value
		add    tosca.Value
		sub    tosca.Value
	}{
		{},
		{
			before: tosca.NewValue(10),
			after:  tosca.NewValue(10),
		},
		{
			before: tosca.NewValue(0),
			after:  tosca.NewValue(1),
			add:    tosca.NewValue(1),
		},
		{
			before: tosca.NewValue(1),
			after:  tosca.NewValue(0),
			sub:    tosca.NewValue(1),
		},
		{
			before: tosca.NewValue(123),
			after:  tosca.NewValue(321),
			add:    tosca.NewValue(321 - 123),
		},
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("%v_to_%v", test.before, test.after), func(t *testing.T) {
			ctrl := gomock.NewController(t)
			stateDb := NewMockStateDb(ctrl)
			stateDb.EXPECT().GetBalance(gomock.Any()).Return(test.before.ToUint256())
			if test.add != (tosca.Value{}) {
				diff := test.add.ToUint256()
				stateDb.EXPECT().AddBalance(gomock.Any(), diff, gomock.Any())
			}
			if test.sub != (tosca.Value{}) {
				diff := test.sub.ToUint256()
				stateDb.EXPECT().SubBalance(gomock.Any(), diff, gomock.Any())
			}

			adapter := &runContextAdapter{evm: &geth.EVM{StateDB: stateDb}}
			adapter.SetBalance(tosca.Address{}, test.after)
		})
	}
}

func TestRunContextAdapter_SetAndGetNonce(t *testing.T) {
	ctrl := gomock.NewController(t)
	stateDb := NewMockStateDb(ctrl)
	adapter := &runContextAdapter{evm: &geth.EVM{StateDB: stateDb}}

	address := tosca.Address{0x42}
	nonce := uint64(123)

	stateDb.EXPECT().SetNonce(gc.Address(address), nonce)
	adapter.SetNonce(address, nonce)

	stateDb.EXPECT().GetNonce(gc.Address(address)).Return(nonce)
	got := adapter.GetNonce(address)
	if got != nonce {
		t.Errorf("Got wrong nonce %v, expected %v", got, nonce)
	}
}

func TestRunContextAdapter_SetAndGetCode(t *testing.T) {
	ctrl := gomock.NewController(t)
	stateDb := NewMockStateDb(ctrl)
	adapter := &runContextAdapter{evm: &geth.EVM{StateDB: stateDb}}

	address := tosca.Address{0x42}
	code := []byte{1, 2, 3}

	stateDb.EXPECT().SetCode(gc.Address(address), code)
	adapter.SetCode(address, code)

	stateDb.EXPECT().GetCode(gc.Address(address)).Return(code)
	got := adapter.GetCode(address)
	if !bytes.Equal(got, code) {
		t.Errorf("Got wrong code %v, expected %v", got, code)
	}
}

func TestRunContextAdapter_SetAndGetStorage(t *testing.T) {
	ctrl := gomock.NewController(t)
	stateDb := NewMockStateDb(ctrl)
	adapter := &runContextAdapter{evm: &geth.EVM{StateDB: stateDb}}

	address := tosca.Address{0x42}
	key := tosca.Key{10}
	original := tosca.Word{0}
	current := tosca.Word{1}
	future := tosca.Word{2}

	stateDb.EXPECT().GetState(gc.Address(address), gc.Hash(key)).Return(gc.Hash(current))
	stateDb.EXPECT().GetCommittedState(gc.Address(address), gc.Hash(key)).Return(gc.Hash(original))
	stateDb.EXPECT().SetState(gc.Address(address), gc.Hash(key), gc.Hash(future))
	status := adapter.SetStorage(address, key, future)
	if status != tosca.StorageAssigned {
		t.Errorf("Storage status did not match expected, want %v, got %v", tosca.StorageAssigned, status)
	}

	stateDb.EXPECT().GetState(gc.Address(address), gc.Hash(key)).Return(gc.Hash(current))
	got := adapter.GetStorage(address, key)
	if got != current {
		t.Errorf("Got wrong storage value %v, expected %v", got, current)
	}
}

func TestRunContextAdapter_GetAndSetTransientStorage(t *testing.T) {
	ctrl := gomock.NewController(t)
	stateDb := NewMockStateDb(ctrl)
	adapter := &runContextAdapter{evm: &geth.EVM{StateDB: stateDb}}

	address := tosca.Address{0x42}
	key := tosca.Key{10}
	value := tosca.Word{100}

	stateDb.EXPECT().SetTransientState(gc.Address(address), gc.Hash(key), gc.Hash(value))
	adapter.SetTransientStorage(address, key, value)

	stateDb.EXPECT().GetTransientState(gc.Address(address), gc.Hash(key)).Return(gc.Hash(value))
	got := adapter.GetTransientStorage(address, key)
	if got != value {
		t.Errorf("Got wrong transient storage value %v, expected %v", got, value)
	}
}

func TestRunContextAdapter_SelfDestruct(t *testing.T) {
	cancunTime := uint64(42)
	londonBlock := big.NewInt(42)
	tests := map[string]struct {
		selfdestructed bool
		blockTime      uint64
	}{
		"selfdestructedPreCancun": {
			true,
			cancunTime - 1,
		},
		"notSelfdestructedPreCancun": {
			false,
			cancunTime - 1,
		},
		"selddestructedCancun": {
			true,
			cancunTime,
		},
		"notSelfdestructedCancun": {
			false,
			cancunTime,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			stateDb := NewMockStateDb(ctrl)

			address := tosca.Address{0x42}
			beneficiary := tosca.Address{0x43}
			contractRef := testContractRef{address: gc.Address(address)}

			blockContext := geth.BlockContext{
				BlockNumber: londonBlock.Add(londonBlock, big.NewInt(1)),
				Time:        test.blockTime,
			}
			chainConfig := &params.ChainConfig{
				CancunTime:  &cancunTime,
				LondonBlock: londonBlock,
				ChainID:     big.NewInt(42),
			}
			evm := geth.NewEVM(blockContext,
				geth.TxContext{},
				stateDb,
				chainConfig,
				geth.Config{},
			)
			adapter := &runContextAdapter{evm: evm, contract: geth.NewContract(contractRef, contractRef, nil, 0)}

			stateDb.EXPECT().HasSelfDestructed(gc.Address(address)).Return(test.selfdestructed)
			stateDb.EXPECT().GetBalance(gc.Address(address)).Return(uint256.NewInt(42))
			stateDb.EXPECT().AddBalance(gc.Address(beneficiary), uint256.NewInt(42), tracing.BalanceDecreaseSelfdestruct)

			if test.blockTime < cancunTime {
				stateDb.EXPECT().SelfDestruct(gc.Address(address))
			} else {
				stateDb.EXPECT().SubBalance(gc.Address(address), uint256.NewInt(42), tracing.BalanceDecreaseSelfdestruct)
				stateDb.EXPECT().Selfdestruct6780(gc.Address(address))
			}

			got := adapter.SelfDestruct(address, beneficiary)
			if got == test.selfdestructed {
				t.Errorf("Selfdestruct should only return true if it has not been called before")
			}
		})
	}
}

func TestRunContextAdapter_SnapshotHandling(t *testing.T) {
	ctrl := gomock.NewController(t)
	stateDb := NewMockStateDb(ctrl)
	adapter := &runContextAdapter{evm: &geth.EVM{StateDB: stateDb}}

	snapshot := tosca.Snapshot(1)

	stateDb.EXPECT().RevertToSnapshot(int(snapshot))
	adapter.RestoreSnapshot(snapshot)

	stateDb.EXPECT().Snapshot().Return(int(snapshot))
	got := adapter.CreateSnapshot()
	if got != snapshot {
		t.Errorf("Got wrong snapshot %v, expected %v", got, snapshot)
	}
}

func TestRunContextAdapter_AccountOperations(t *testing.T) {
	ctrl := gomock.NewController(t)
	stateDb := NewMockStateDb(ctrl)
	adapter := &runContextAdapter{evm: &geth.EVM{StateDB: stateDb}}

	address := tosca.Address{0x42}
	code := tosca.Code{1, 2, 3}

	stateDb.EXPECT().AddressInAccessList(gc.Address(address)).Return(false)
	stateDb.EXPECT().AddAddressToAccessList(gc.Address(address))
	accessStatus := adapter.AccessAccount(address)
	if accessStatus != tosca.ColdAccess {
		t.Errorf("Got wrong access type %v, expected %v", accessStatus, tosca.ColdAccess)
	}

	stateDb.EXPECT().Exist(gc.Address(address)).Return(true)
	exits := adapter.AccountExists(address)
	if !exits {
		t.Errorf("Account should exist")
	}

	stateDb.EXPECT().Exist(gc.Address(address)).Return(false)
	stateDb.EXPECT().CreateAccount(gc.Address(address))
	stateDb.EXPECT().SetCode(gc.Address(address), code)
	created := adapter.CreateAccount(address, code)
	if !created {
		t.Errorf("Account should have been created")
	}

	stateDb.EXPECT().AddressInAccessList(gc.Address(address)).Return(true)
	inAccessList := adapter.IsAddressInAccessList(address)
	if !inAccessList {
		t.Errorf("Address should be in access list")
	}
}

type testContractRef struct {
	address gc.Address
}

func (c testContractRef) Address() gc.Address {
	return c.address
}

func TestRunContextAdapter_Call(t *testing.T) {
	ctrl := gomock.NewController(t)
	stateDb := NewMockStateDb(ctrl)

	address := gc.Address{0x42}

	stateDb.EXPECT().Snapshot().Return(1)
	stateDb.EXPECT().Exist(address).Return(true)
	stateDb.EXPECT().GetCode(address).Return([]byte{})
	stateDb.EXPECT().Witness()

	contractRef := testContractRef{address: address}

	canTransfer := func(geth.StateDB, gc.Address, *uint256.Int) bool { return true }
	transfer := func(geth.StateDB, gc.Address, gc.Address, *uint256.Int) {}

	runContextAdapter := &runContextAdapter{
		evm: &geth.EVM{
			StateDB: stateDb,
			Context: geth.BlockContext{
				CanTransfer: canTransfer,
				Transfer:    transfer,
			},
		},
		contract: geth.NewContract(contractRef, contractRef, nil, 0),
	}

	gas := tosca.Gas(42)

	parameters := tosca.CallParameters{
		Recipient: tosca.Address(address),
		Gas:       gas,
	}

	result, err := runContextAdapter.Call(tosca.Call, parameters)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if !result.Success {
		t.Errorf("Call was not successful")
	}
	if result.GasLeft != gas {
		t.Errorf("Call has the wrong amount of gas left: %v, expected: %v", result.GasLeft, gas)
	}
}

func TestRunContextAdapter_getPrevRandaoReturnsHashBasedOnRevision(t *testing.T) {
	tests := map[string]struct {
		revision tosca.Revision
		want     tosca.Hash
	}{
		"london": {
			revision: tosca.R10_London,
			want:     tosca.Hash(tosca.NewValue(42)),
		},
		"paris": {
			revision: tosca.R11_Paris,
			want:     tosca.Hash{0x24},
		},
		"shanghai": {
			revision: tosca.R12_Shanghai,
			want:     tosca.Hash{0x24},
		},
	}

	random := gc.Hash{0x24}
	context := geth.BlockContext{
		Difficulty: big.NewInt(42),
		Random:     &random,
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {

			got, err := getPrevRandao(&context, test.revision)
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
			if got != test.want {
				t.Errorf("Got wrong prevRandao %v, expected %v", got, test.want)
			}
		})
	}
}

func TestRunContextAdapter_getPrevRandaoErrorIfDifficultyCanNotBeConverted(t *testing.T) {
	context := geth.BlockContext{
		Difficulty: big.NewInt(-42),
	}

	_, err := getPrevRandao(&context, tosca.R10_London)
	if err == nil {
		t.Errorf("Expected error, got nil")
	}
}

func TestRunContextAdapter_Run(t *testing.T) {
	tests := map[string]bool{
		"success": true,
		"failure": false,
	}

	for name, success := range tests {
		t.Run(name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			stateDb := NewMockStateDb(ctrl)
			interpreter := tosca.NewMockInterpreter(ctrl)

			chainId := int64(42)
			blockNumber := int64(24)
			address := tosca.Address{0x42}

			blockParameters := geth.BlockContext{BlockNumber: big.NewInt(blockNumber)}
			chainConfig := &params.ChainConfig{ChainID: big.NewInt(chainId), IstanbulBlock: big.NewInt(23)}
			evm := geth.NewEVM(blockParameters, geth.TxContext{}, stateDb, chainConfig, geth.Config{})
			adapter := &gethInterpreterAdapter{
				evm:         evm,
				interpreter: interpreter,
			}

			blockParams := tosca.BlockParameters{
				ChainID:     tosca.Word(tosca.NewValue(uint64(chainId))),
				BlockNumber: blockNumber,
			}
			expectedParams := tosca.Parameters{
				BlockParameters: blockParams,
				Kind:            tosca.Call,
				Static:          false,
				Recipient:       address,
				Sender:          address,
			}

			interpreter.EXPECT().Run(gomock.Any()).DoAndReturn(func(params tosca.Parameters) (tosca.Result, error) {
				// The parameters save the context as a pointer, its value can
				// not be predicted during the setup phase of the mock.
				if expectedParams.BlockParameters.ChainID != params.BlockParameters.ChainID ||
					expectedParams.BlockParameters.BlockNumber != params.BlockParameters.BlockNumber ||
					expectedParams.BlockParameters.Timestamp != params.BlockParameters.Timestamp ||
					expectedParams.BlockParameters.Coinbase != params.BlockParameters.Coinbase ||
					expectedParams.BlockParameters.GasLimit != params.BlockParameters.GasLimit ||
					expectedParams.BlockParameters.PrevRandao != params.BlockParameters.PrevRandao ||
					expectedParams.BlockParameters.BaseFee != params.BlockParameters.BaseFee ||
					expectedParams.BlockParameters.BlobBaseFee != params.BlockParameters.BlobBaseFee ||
					expectedParams.BlockParameters.Revision != params.BlockParameters.Revision ||
					expectedParams.TransactionParameters.Origin != params.TransactionParameters.Origin ||
					expectedParams.TransactionParameters.GasPrice != params.TransactionParameters.GasPrice ||
					!slices.Equal(expectedParams.TransactionParameters.BlobHashes, params.TransactionParameters.BlobHashes) ||
					expectedParams.Kind != params.Kind ||
					expectedParams.Static != params.Static ||
					expectedParams.Depth != params.Depth ||
					expectedParams.Gas != params.Gas ||
					expectedParams.Recipient != params.Recipient ||
					expectedParams.Sender != params.Sender ||
					!slices.Equal(expectedParams.Input, params.Input) ||
					expectedParams.Value != params.Value ||
					expectedParams.CodeHash != params.CodeHash ||
					!bytes.Equal(expectedParams.Code, params.Code) {
					t.Errorf("Parameters did not match, expected %v, got %v", params, expectedParams)
				}

				return tosca.Result{Success: success}, nil
			})

			refundShift := uint64(1 << 60)
			stateDb.EXPECT().AddRefund(refundShift)
			if success {
				stateDb.EXPECT().AddRefund(uint64(0))
				stateDb.EXPECT().GetRefund().Return(refundShift)
				stateDb.EXPECT().SubRefund(refundShift)
			}

			contractRef := testContractRef{address: gc.Address(address)}
			contract := geth.NewContract(contractRef, contractRef, nil, 0)

			_, err := adapter.Run(contract, []byte{}, false)
			if success && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
			if !success && err == nil {
				t.Errorf("Expected error, got nil")
			}
		})
	}
}

func TestRunContextAdapter_bigIntToValue(t *testing.T) {
	tests := map[string]struct {
		input         *big.Int
		want          tosca.Value
		expectedError bool
	}{
		"nil": {
			input:         nil,
			want:          tosca.Value{},
			expectedError: false,
		},
		"zero": {
			input:         big.NewInt(0),
			want:          tosca.NewValue(0),
			expectedError: false,
		},
		"positive": {
			input:         big.NewInt(42),
			want:          tosca.NewValue(42),
			expectedError: false,
		},
		"negative": {
			input:         big.NewInt(-42),
			want:          tosca.Value{},
			expectedError: true,
		},
		"overflow": {
			input:         big.NewInt(1).Lsh(big.NewInt(1), 256),
			want:          tosca.Value{},
			expectedError: true,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			got, err := bigIntToValue(test.input)
			if test.expectedError && err == nil {
				t.Errorf("Expected error, got nil")
			}
			if !test.expectedError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
			if got != test.want {
				t.Errorf("Conversion returned wrong value, expected %v, got %v", test.want, got)
			}
		})
	}
}

func TestRunContextAdapter_bigIntToHash(t *testing.T) {
	input := big.NewInt(42)
	want := tosca.Hash(tosca.NewValue(42))
	got, err := bigIntToHash(input)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if got != want {
		t.Errorf("Conversion returned wrong value, expected %v, got %v", want, got)
	}
}

func TestRunContextAdapter_bigIntToWord(t *testing.T) {
	input := big.NewInt(42)
	want := tosca.Word(tosca.NewValue(42))
	got, err := bigIntToWord(input)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if got != want {
		t.Errorf("Conversion returned wrong value, expected %v, got %v", want, got)
	}
}

func TestRunContextAdapter_ConvertRevision(t *testing.T) {
	cancunTime := uint64(1000)
	shanghaiTime := uint64(900)
	parisBlock := big.NewInt(100)
	londonBlock := big.NewInt(90)
	berlinBlock := big.NewInt(80)
	istanbulBlock := big.NewInt(70)

	tests := map[string]struct {
		random *gc.Hash
		block  *big.Int
		time   uint64
		want   tosca.Revision
	}{
		"Istanbul": {
			block: istanbulBlock,
			time:  uint64(0),
			want:  tosca.R07_Istanbul,
		},
		"Berlin": {
			block: berlinBlock,
			time:  uint64(0),
			want:  tosca.R09_Berlin,
		},
		"London": {
			block: londonBlock,
			time:  uint64(0),
			want:  tosca.R10_London,
		},
		"Paris": {
			random: &gc.Hash{0x42},
			block:  parisBlock,
			time:   uint64(0),
			want:   tosca.R11_Paris,
		},
		"Shanghai": {
			random: &gc.Hash{0x42},
			block:  parisBlock,
			time:   shanghaiTime,
			want:   tosca.R12_Shanghai,
		},
		"Cancun": {
			random: &gc.Hash{0x42},
			block:  parisBlock,
			time:   cancunTime,
			want:   tosca.R13_Cancun,
		},
	}

	chainConfig := &params.ChainConfig{
		ChainID:            big.NewInt(42),
		IstanbulBlock:      istanbulBlock,
		LondonBlock:        londonBlock,
		BerlinBlock:        berlinBlock,
		MergeNetsplitBlock: parisBlock,
		ShanghaiTime:       &shanghaiTime,
		CancunTime:         &cancunTime,
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			evm := geth.NewEVM(geth.BlockContext{Random: test.random}, geth.TxContext{}, nil, chainConfig, geth.Config{})
			rules := evm.ChainConfig().Rules(test.block, evm.Context.Random != nil, test.time)
			got, err := convertRevision(rules)
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
			if got != test.want {
				t.Errorf("Conversion returned wrong value, expected %v, got %v", test.want, got)
			}
		})
	}
}

func TestRunContextAdapter_ConvertRevisionReturnsUnsupportedRevisionError(t *testing.T) {
	rules := params.Rules{
		IsHomestead: true,
	}
	_, err := convertRevision(rules)
	targetError := &tosca.ErrUnsupportedRevision{}
	if !errors.As(err, &targetError) {
		t.Errorf("Expected unsupported revision error, got %v", err)
	}
}

func TestRunContextAdapter_gethToVMErrors(t *testing.T) {
	gas := tosca.Gas(42)
	otherError := fmt.Errorf("other error")
	tests := map[string]struct {
		input      error
		wantResult tosca.CallResult
		wantError  error
	}{
		"nil": {
			input: nil,
		},
		"insufficientBalance": {
			input:      geth.ErrInsufficientBalance,
			wantResult: tosca.CallResult{GasLeft: gas},
			wantError:  nil,
		},
		"maxCallDepth": {
			input:      geth.ErrDepth,
			wantResult: tosca.CallResult{GasLeft: gas},
			wantError:  nil,
		},
		"nonceOverflow": {
			input:      geth.ErrNonceUintOverflow,
			wantResult: tosca.CallResult{GasLeft: gas},
			wantError:  nil,
		},
		"OutOfGas": {
			input:      geth.ErrOutOfGas,
			wantResult: tosca.CallResult{},
			wantError:  nil,
		},
		"stackUnderflow": {
			input:      &geth.ErrStackUnderflow{},
			wantResult: tosca.CallResult{},
			wantError:  nil,
		},
		"stackOverflow": {
			input:      &geth.ErrStackOverflow{},
			wantResult: tosca.CallResult{},
			wantError:  nil,
		},
		"invalidOpCode": {
			input:      &geth.ErrInvalidOpCode{},
			wantResult: tosca.CallResult{},
			wantError:  nil,
		},
		"other": {
			input:      otherError,
			wantResult: tosca.CallResult{},
			wantError:  otherError,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			gotResult, gotErr := gethToVMErrors(test.input, gas)
			if !errors.Is(gotErr, test.wantError) {
				t.Errorf("Unexpected error: expected %v, got %v", test.wantError, gotErr)
			}
			reflect.DeepEqual(gotResult, test.wantResult)
		})
	}
}

func TestRunContextAdapter_AllGethErrorsAreHandled(t *testing.T) {
	// all errors defined in geth/core/vm/gethErrors.go
	gethErrors := []error{
		geth.ErrOutOfGas,
		geth.ErrCodeStoreOutOfGas,
		geth.ErrDepth,
		geth.ErrInsufficientBalance,
		geth.ErrContractAddressCollision,
		geth.ErrExecutionReverted,
		geth.ErrMaxCodeSizeExceeded,
		geth.ErrMaxInitCodeSizeExceeded,
		geth.ErrInvalidJump,
		geth.ErrWriteProtection,
		geth.ErrReturnDataOutOfBounds,
		geth.ErrGasUintOverflow,
		geth.ErrInvalidCode,
		geth.ErrNonceUintOverflow,

		&geth.ErrStackUnderflow{},
		&geth.ErrStackOverflow{},
		&geth.ErrInvalidOpCode{},
	}

	for _, inErr := range gethErrors {
		_, outErr := gethToVMErrors(inErr, tosca.Gas(42))
		if outErr != nil {
			t.Errorf("Unexpected return error %v", outErr)
		}
	}
}

func TestAdapter_ReadOnlyIsSetAndResetCorrectly(t *testing.T) {
	tests := map[string]bool{
		"readOnly":    true,
		"notReadOnly": false,
	}
	recipient := tosca.Address{0x42}
	depth := 42
	gas := uint64(42)
	for name, readOnly := range tests {
		t.Run(name, func(t *testing.T) {
			setGas := encodeReadOnlyInGas(gas, recipient, readOnly)
			gotReadOnly, unsetGas := decodeReadOnlyFromGas(depth, readOnly, setGas)

			if unsetGas != gas {
				t.Errorf("Gas was not set or unset correctly, expected %v, got %v", gas, unsetGas)
			}
			if gotReadOnly != readOnly {
				t.Errorf("ReadOnly was not set or unset correctly, expected %v, got %v", readOnly, gotReadOnly)
			}
		})
	}
}

func TestGethInterpreterAdapter_RefundShiftIsReverted(t *testing.T) {
	tests := map[string]struct {
		err    error
		refund uint64
	}{
		"noErrorHighRefund": {
			err:    nil,
			refund: 100,
		},
		"noErrorLowRefund": {
			err:    nil,
			refund: 10,
		},
		"errorHighRefund": {
			err:    fmt.Errorf("error"),
			refund: 100,
		},
		"errorLowRefund": {
			err:    fmt.Errorf("error"),
			refund: 10,
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			stateDb := NewMockStateDb(ctrl)

			shift := uint64(42)
			expectedSub := shift
			if test.refund < shift {
				expectedSub = test.refund
			}

			if test.err == nil {
				stateDb.EXPECT().GetRefund().Return(test.refund)
				stateDb.EXPECT().SubRefund(expectedSub)
			}

			undoRefundShift(stateDb, test.err, shift)
		})
	}
}
