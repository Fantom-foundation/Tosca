// Copyright (c) 2024 Fantom Foundation
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at fantom.foundation/bsl11.
//
// Change Date: 2028-4-16
//
// On the date above, in accordance with the Business Source License, use of
// this software will be governed by the GNU Lesser General Public License v3.

package floria

import (
	"errors"
	"testing"

	"github.com/Fantom-foundation/Tosca/go/tosca"
	"go.uber.org/mock/gomock"
)

var (
	address0x00 = tosca.Address{0x00}
	address0x01 = tosca.Address{0x01}

	remainingGas = tosca.Gas(10)
)

func TestStateContract_executeStateSetBalance(t *testing.T) {
	tests := map[string]struct {
		input         []byte
		gas           tosca.Gas
		sender        tosca.Address
		expectedError error
	}{
		"outOfGas": {
			make([]byte, 64),
			1,
			tosca.Address{},
			ErrOutOfGas,
		},
		"invalidInput": {
			make([]byte, 63),
			callValueTransferGas + remainingGas,
			tosca.Address{},
			ErrExecutionReverted,
		},
		"sameAddress": {
			make([]byte, 64),
			callValueTransferGas + remainingGas,
			tosca.Address{},
			ErrExecutionReverted,
		},
		"successful": {
			make([]byte, 64),
			callValueTransferGas + remainingGas,
			address0x01,
			nil,
		},
	}

	for testName, test := range tests {
		t.Run(testName, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			state := tosca.NewMockWorldState(ctrl)

			wantGas := tosca.Gas(0)
			if test.expectedError == nil {
				wantGas = remainingGas
				state.EXPECT().SetBalance(tosca.Address{}, tosca.Value{})
			}

			gas, err := executeStateSetBalance(state, test.sender, test.input, test.gas)
			if !errors.Is(err, test.expectedError) {
				t.Errorf("unexpected error, want %v, got %v", test.expectedError, err)
			}
			if gas != wantGas {
				t.Errorf("unexpected gas, want %v, got %v", wantGas, gas)
			}
		})
	}
}

func TestStateContract_executeStateCopyCode(t *testing.T) {
	codeSize := 42
	tests := map[string]struct {
		input         []byte
		gas           tosca.Gas
		expectedError error
	}{
		"outOfGas": {
			make([]byte, 64),
			1,
			ErrOutOfGas,
		},
		"invalidInput": {
			make([]byte, 63),
			createGas + remainingGas,
			ErrExecutionReverted,
		},
		"insufficientMemoryGas": {
			make([]byte, 64),
			createGas + remainingGas,
			ErrOutOfGas,
		},
		"successful": {
			append(append(make([]byte, 12), address0x01[:]...), make([]byte, 32)...),
			createGas + tosca.Gas(codeSize)*(createDataGas+memoryGas) + remainingGas,
			nil,
		},
	}

	for testName, test := range tests {
		t.Run(testName, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			state := tosca.NewMockWorldState(ctrl)
			state.EXPECT().GetCode(tosca.Address{}).Return(make([]byte, codeSize)).AnyTimes()

			wantGas := tosca.Gas(0)
			if test.expectedError == nil {
				wantGas = remainingGas
				state.EXPECT().SetCode(address0x01, tosca.Code(make([]byte, codeSize)))
			}

			gas, err := executeStateContractCopyCode(state, test.input, test.gas)
			if !errors.Is(err, test.expectedError) {
				t.Errorf("unexpected error, want %v, got %v", test.expectedError, err)
			}
			if gas != wantGas {
				t.Errorf("unexpected gas, want %v, got %v", wantGas, gas)
			}
		})
	}
}

func TestStateContract_executeStateSwapCode(t *testing.T) {
	codeSize := 42
	tests := map[string]struct {
		input         []byte
		gas           tosca.Gas
		expectedError error
	}{
		"outOfGas": {
			make([]byte, 64),
			1,
			ErrOutOfGas,
		},
		"invalidInput": {
			make([]byte, 63),
			2*createGas + remainingGas,
			ErrExecutionReverted,
		},
		"insufficientMemoryGas": {
			make([]byte, 64),
			2*createGas + remainingGas,
			ErrOutOfGas,
		},
		"successful": {
			append(append(make([]byte, 12), address0x01[:]...), make([]byte, 32)...),
			2*createGas + tosca.Gas(codeSize)*(createDataGas+memoryGas) + remainingGas,
			nil,
		},
	}

	for testName, test := range tests {
		t.Run(testName, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			state := tosca.NewMockWorldState(ctrl)
			state.EXPECT().GetCode(address0x00).Return(make([]byte, codeSize)).AnyTimes()
			state.EXPECT().GetCode(address0x01).Return(make([]byte, codeSize)).AnyTimes()

			wantGas := tosca.Gas(0)
			if test.expectedError == nil {
				wantGas = remainingGas

				state.EXPECT().SetCode(address0x00, tosca.Code(make([]byte, codeSize)))
				state.EXPECT().SetCode(address0x01, tosca.Code(make([]byte, codeSize)))
			}

			gas, err := executeStateContractSwapCode(state, test.input, test.gas)
			if !errors.Is(err, test.expectedError) {
				t.Errorf("unexpected error, want %v, got %v", test.expectedError, err)
			}
			if gas != wantGas {
				t.Errorf("unexpected gas, want %v, got %v", wantGas, gas)
			}
		})
	}
}

func TestStateContract_executeStateSetStorage(t *testing.T) {
	tests := map[string]struct {
		input         []byte
		gas           tosca.Gas
		expectedError error
	}{
		"outOfGas": {
			make([]byte, 96),
			1,
			ErrOutOfGas,
		},
		"invalidInput": {
			make([]byte, 63),
			sStoreSetGasEIP2200 + remainingGas,
			ErrExecutionReverted,
		},
		"successful": {
			make([]byte, 96),
			sStoreSetGasEIP2200 + remainingGas,
			nil,
		},
	}

	for testName, test := range tests {
		t.Run(testName, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			state := tosca.NewMockWorldState(ctrl)

			wantGas := tosca.Gas(0)
			if test.expectedError == nil {
				wantGas = remainingGas
				state.EXPECT().SetStorage(tosca.Address{}, tosca.Key{}, tosca.Word{})
			}

			gas, err := executeStateContractSetStorage(state, test.input, test.gas)
			if !errors.Is(err, test.expectedError) {
				t.Errorf("unexpected error, want %v, got %v", test.expectedError, err)
			}
			if gas != wantGas {
				t.Errorf("unexpected gas, want %v, got %v", wantGas, gas)
			}
		})
	}
}

func TestStateContract_executeStateIncNonce(t *testing.T) {
	increment := byte(5)
	tests := map[string]struct {
		input         []byte
		gas           tosca.Gas
		sender        tosca.Address
		expectedError error
	}{
		"outOfGas": {
			make([]byte, 64),
			1,
			tosca.Address{},
			ErrOutOfGas,
		},
		"invalidInput": {
			make([]byte, 63),
			callValueTransferGas + remainingGas,
			tosca.Address{},
			ErrExecutionReverted,
		},
		"sameAddress": {
			make([]byte, 64),
			callValueTransferGas + remainingGas,
			tosca.Address{},
			ErrExecutionReverted,
		},
		"invalidValue": {
			make([]byte, 64),
			callValueTransferGas + remainingGas,
			address0x01,
			ErrExecutionReverted,
		},
		"successful": {
			append(make([]byte, 63), increment),
			callValueTransferGas + remainingGas,
			address0x01,
			nil,
		},
	}

	for testName, test := range tests {
		t.Run(testName, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			state := tosca.NewMockWorldState(ctrl)

			wantGas := tosca.Gas(0)
			if test.expectedError == nil {
				wantGas = remainingGas

				state.EXPECT().GetNonce(tosca.Address{}).Return(uint64(5))
				state.EXPECT().SetNonce(tosca.Address{}, uint64(5+increment))
			}

			gas, err := executeStateContractIncNonce(state, test.sender, test.input, test.gas)
			if !errors.Is(err, test.expectedError) {
				t.Errorf("unexpected error, want %v, got %v", test.expectedError, err)
			}
			if gas != wantGas {
				t.Errorf("unexpected gas, want %v, got %v", wantGas, gas)
			}
		})
	}

}

func TestStateContract_HandleStatePrecompiled(t *testing.T) {
	codeSize := 42
	increment := byte(5)
	tests := map[string]struct {
		recipient          tosca.Address
		inputPrefix        []byte
		input              []byte
		isStatePrecompiled bool
		mockSetUp          func(*tosca.MockWorldState)
	}{
		"nonPrecompiled": {
			tosca.Address{0x20},
			[]byte{0x01, 0x02, 0x03, 0x04},
			make([]byte, 64),
			false,
			func(mock *tosca.MockWorldState) {},
		},
		"setBalance": {
			StateContractAddress(),
			[]byte{0xe3, 0x4, 0x43, 0xbc},
			make([]byte, 64),
			true,
			func(mock *tosca.MockWorldState) {
				mock.EXPECT().SetBalance(tosca.Address{}, tosca.Value{})
			},
		},
		"copyCode": {
			StateContractAddress(),
			[]byte{0xd6, 0xa0, 0xc7, 0xaf},
			append(append(make([]byte, 12), address0x01[:]...), make([]byte, 32)...),
			true,
			func(mock *tosca.MockWorldState) {
				mock.EXPECT().GetCode(tosca.Address{}).Return(make([]byte, codeSize))
				mock.EXPECT().SetCode(address0x01, tosca.Code(make([]byte, codeSize)))
			},
		},
		"swapCode": {
			StateContractAddress(),
			[]byte{0x7, 0x69, 0xb, 0x2a},
			append(append(make([]byte, 12), address0x01[:]...), make([]byte, 32)...),
			true,
			func(mock *tosca.MockWorldState) {
				mock.EXPECT().GetCode(address0x00).Return(make([]byte, codeSize))
				mock.EXPECT().GetCode(address0x01).Return(make([]byte, codeSize))
				mock.EXPECT().SetCode(address0x00, tosca.Code(make([]byte, codeSize)))
				mock.EXPECT().SetCode(address0x01, tosca.Code(make([]byte, codeSize)))
			},
		},
		"setStorage": {
			StateContractAddress(),
			[]byte{0x39, 0xe5, 0x3, 0xab},
			make([]byte, 96),
			true,
			func(mock *tosca.MockWorldState) {
				mock.EXPECT().SetStorage(tosca.Address{}, tosca.Key{}, tosca.Word{})
			},
		},
		"incNonce": {
			StateContractAddress(),
			[]byte{0x79, 0xbe, 0xad, 0x38},
			append(make([]byte, 63), increment),
			true,
			func(mock *tosca.MockWorldState) {
				mock.EXPECT().GetNonce(tosca.Address{}).Return(uint64(5))
				mock.EXPECT().SetNonce(tosca.Address{}, uint64(5+increment))
			},
		},
	}
	for testName, test := range tests {
		t.Run(testName, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			state := tosca.NewMockWorldState(ctrl)
			test.mockSetUp(state)

			sender := DriverAddress()
			gas := tosca.Gas(1000000)
			input := append(test.inputPrefix, test.input...)

			result, isStatePrecompiled := handleStateContract(state, sender, test.recipient, input, gas)
			if isStatePrecompiled != test.isStatePrecompiled {
				t.Errorf("wrong state precompiled address, want %v, got %v", test.isStatePrecompiled, isStatePrecompiled)
			}
			if isStatePrecompiled && result.Success != true {
				t.Errorf("execution was not successful")
			}
		})
	}
}

func TestStateContract_CodeOperationsWorkWithNil(t *testing.T) {
	tests := map[string]struct {
		operation func(tosca.WorldState, []byte, tosca.Gas) (tosca.Gas, error)
		mockSetUp func(*tosca.MockWorldState)
	}{
		"copyCode": {
			executeStateContractCopyCode,
			func(state *tosca.MockWorldState) {},
		},
		"swapCode": {
			executeStateContractSwapCode,
			func(state *tosca.MockWorldState) {
				state.EXPECT().GetCode(address0x01).Return(nil)
				state.EXPECT().SetCode(address0x00, tosca.Code(nil))
			},
		},
	}
	input := append(append(make([]byte, 12), address0x01[:]...), make([]byte, 32)...)
	gas := tosca.Gas(1000000)

	for testName, test := range tests {
		t.Run(testName, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			state := tosca.NewMockWorldState(ctrl)
			state.EXPECT().GetCode(address0x00).Return(nil)
			state.EXPECT().SetCode(address0x01, tosca.Code(nil))
			test.mockSetUp(state)

			_, err := test.operation(state, input, gas)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestStateContract_InvalidCallReportsFailure(t *testing.T) {
	tests := map[string]struct {
		sender tosca.Address
		input  []byte
	}{
		"invalidSender": {
			address0x01,
			make([]byte, 64),
		},
		"invalidPrefix": {
			DriverAddress(),
			make([]byte, 64),
		},
		"tooShortInput": {
			DriverAddress(),
			make([]byte, 3),
		},
	}
	for testName, test := range tests {
		t.Run(testName, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			state := tosca.NewMockWorldState(ctrl)

			result, isStatePrecompiled := handleStateContract(state, test.sender, StateContractAddress(), test.input, 1000000)
			if isStatePrecompiled != true {
				t.Errorf("state contract address was not handled correctly")
			}
			if result.Success != false {
				t.Errorf("invalid execution was successful")
			}
			if result.GasLeft != tosca.Gas(0) {
				t.Errorf("unexpected gas, want %v, got %v", 0, result.GasLeft)
			}
		})
	}

}
