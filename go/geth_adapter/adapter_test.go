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
	"fmt"
	"math/big"
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

func TestRunContextAdapter_GethInterpretersIsAvailable(t *testing.T) {
	if res, err := tosca.NewInterpreter("geth", nil); res == nil || err != nil {
		t.Fatal("Geth interpreter not available in Tosca")
	}
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
	adapter.GetNonce(address)
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
	adapter.GetCode(address)
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
	adapter.SetStorage(address, key, future)

	stateDb.EXPECT().GetState(gc.Address(address), gc.Hash(key)).Return(gc.Hash(current))
	adapter.GetStorage(address, key)
}

func TestRunContextAdapter_GetAndSetTransientStorage(t *testing.T) {
	ctrl := gomock.NewController(t)
	stateDb := NewMockStateDb(ctrl)
	adapter := &runContextAdapter{evm: &geth.EVM{StateDB: stateDb}}

	address := tosca.Address{0x42}
	key := tosca.Key{10}
	value := tosca.Word{100}

	stateDb.EXPECT().GetTransientState(gc.Address(address), gc.Hash(key)).Return(gc.Hash(value))
	adapter.GetTransientStorage(address, key)

	stateDb.EXPECT().SetTransientState(gc.Address(address), gc.Hash(key), gc.Hash(value))
	adapter.SetTransientStorage(address, key, value)
}

func TestRunContextAdapter_SelfDestruct(t *testing.T) {
	ctrl := gomock.NewController(t)
	stateDb := NewMockStateDb(ctrl)

	address := tosca.Address{0x42}
	beneficiary := tosca.Address{0x43}
	contractRef := testContractRef{address: gc.Address(address)}

	blockContext := geth.BlockContext{
		BlockNumber: big.NewInt(0),
		Time:        uint64(0),
	}
	chainConfig := &params.ChainConfig{
		ChainID: big.NewInt(0),
	}
	evm := geth.NewEVM(blockContext,
		geth.TxContext{},
		stateDb,
		chainConfig,
		geth.Config{},
	)
	adapter := &runContextAdapter{evm: evm, contract: geth.NewContract(contractRef, contractRef, nil, 0)}

	stateDb.EXPECT().HasSelfDestructed(gc.Address(address)).Return(false)
	stateDb.EXPECT().GetBalance(gc.Address(address)).Return(uint256.NewInt(0))
	stateDb.EXPECT().AddBalance(gc.Address(beneficiary), uint256.NewInt(0), tracing.BalanceDecreaseSelfdestruct)
	stateDb.EXPECT().SelfDestruct(gc.Address(address))

	adapter.SelfDestruct(address, beneficiary)
}

func TestRunContextAdapter_SnapshotHandling(t *testing.T) {
	ctrl := gomock.NewController(t)
	stateDb := NewMockStateDb(ctrl)
	adapter := &runContextAdapter{evm: &geth.EVM{StateDB: stateDb}}

	snapshot := tosca.Snapshot(1)

	stateDb.EXPECT().Snapshot().Return(int(snapshot))
	adapter.CreateSnapshot()

	stateDb.EXPECT().RevertToSnapshot(int(snapshot))
	adapter.RestoreSnapshot(snapshot)
}

func TestRunContextAdapter_AccountOperations(t *testing.T) {
	ctrl := gomock.NewController(t)
	stateDb := NewMockStateDb(ctrl)
	adapter := &runContextAdapter{evm: &geth.EVM{StateDB: stateDb}}

	address := tosca.Address{0x42}
	code := tosca.Code{1, 2, 3}

	stateDb.EXPECT().AddressInAccessList(gc.Address(address))
	stateDb.EXPECT().AddAddressToAccessList(gc.Address(address))
	adapter.AccessAccount(address)

	stateDb.EXPECT().Exist(gc.Address(address)).Return(true)
	adapter.AccountExists(address)

	stateDb.EXPECT().Exist(gc.Address(address)).Return(false)
	stateDb.EXPECT().CreateAccount(gc.Address(address))
	stateDb.EXPECT().SetCode(gc.Address(address), code)
	adapter.CreateAccount(address, code)

	stateDb.EXPECT().AddressInAccessList(gc.Address(address))
	adapter.IsAddressInAccessList(address)
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

			refundShift := uint64(1 << 60)
			stateDb.EXPECT().AddRefund(refundShift)

			interpreter.EXPECT().Run(gomock.Any()).Return(tosca.Result{Success: success}, nil)

			if success {
				stateDb.EXPECT().AddRefund(uint64(0))
				stateDb.EXPECT().GetRefund().Return(refundShift)
				stateDb.EXPECT().SubRefund(refundShift)
			}

			blockParameters := geth.BlockContext{BlockNumber: big.NewInt(0)}
			chainConfig := &params.ChainConfig{ChainID: big.NewInt(0)}
			evm := geth.NewEVM(blockParameters, geth.TxContext{}, stateDb, chainConfig, geth.Config{})

			adapter := &gethInterpreterAdapter{
				evm:         evm,
				interpreter: interpreter,
			}

			contractRef := testContractRef{address: gc.Address{0x42}}
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
