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
	"strings"
	"testing"

	test_utils "github.com/Fantom-foundation/Tosca/go/processor"
	"github.com/Fantom-foundation/Tosca/go/tosca"
	"go.uber.org/mock/gomock"
)

func TestPrecompiled_RightNumberOfContractsDependingOnRevision(t *testing.T) {
	tests := []struct {
		revision          tosca.Revision
		numberOfContracts int
	}{
		{tosca.R07_Istanbul, 9},
		{tosca.R09_Berlin, 9},
		{tosca.R10_London, 9},
		{tosca.R11_Paris, 9},
		{tosca.R12_Shanghai, 9},
		{tosca.R13_Cancun, 10},
	}

	for _, test := range tests {
		count := 0
		for i := byte(0x01); i < byte(0x42); i++ {
			address := test_utils.NewAddress(i)
			_, isPrecompiled := getPrecompiledContract(address, test.revision)
			if isPrecompiled {
				count++
			}
		}
		if count != test.numberOfContracts {
			t.Errorf("unexpected number of precompiled contracts for revision %v, want %v, got %v", test.revision, test.numberOfContracts, count)
		}
	}
}

func TestPrecompiled_AddressesAreHandledCorrectly(t *testing.T) {
	tests := map[string]struct {
		revision      tosca.Revision
		address       tosca.Address
		gas           tosca.Gas
		isPrecompiled bool
		success       bool
	}{
		"nonPrecompiled":            {tosca.R09_Berlin, test_utils.NewAddress(0x20), 3000, false, false},
		"ecrecover-success":         {tosca.R10_London, test_utils.NewAddress(0x01), 3000, true, true},
		"ecrecover-outOfGas":        {tosca.R10_London, test_utils.NewAddress(0x01), 1, true, false},
		"pointEvaluation-success":   {tosca.R13_Cancun, test_utils.NewAddress(0x0a), 55000, true, true},
		"pointEvaluation-outOfGas":  {tosca.R13_Cancun, test_utils.NewAddress(0x0a), 1, true, false},
		"pointEvaluation-preCancun": {tosca.R10_London, test_utils.NewAddress(0x0a), 3000, false, false},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {

			input := tosca.Data{}
			if strings.Contains(name, "pointEvaluation") {
				input = test_utils.ValidPointEvaluationInput
			}

			result, isPrecompiled := handlePrecompiledContract(test.revision, input, test.address, test.gas)
			if isPrecompiled != test.isPrecompiled {
				t.Errorf("unexpected precompiled, want %v, got %v", test.isPrecompiled, isPrecompiled)
			}
			if result.Success != test.success {
				t.Errorf("unexpected success, want %v, got %v", test.success, result.Success)
			}
		})
	}
}

func TestPrecompiled_StateSetBalance(t *testing.T) {
	tests := map[string]struct {
		input  []byte
		gas    tosca.Gas
		sender tosca.Address
		err    error
	}{
		"outOfGas": {
			make([]byte, 64),
			1,
			tosca.Address{},
			ErrOutOfGas,
		},
		"invalidInput": {
			make([]byte, 63),
			9000,
			tosca.Address{},
			ErrExecutionReverted,
		},
		"sameAddress": {
			make([]byte, 64),
			9000,
			tosca.Address{},
			ErrExecutionReverted,
		},
		"successful": {
			make([]byte, 64),
			9000,
			tosca.Address{0x01},
			nil,
		},
	}

	for testName, test := range tests {
		t.Run(testName, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			state := tosca.NewMockWorldState(ctrl)
			if test.err == nil {
				state.EXPECT().SetBalance(tosca.Address{}, tosca.Value{})
			}

			gas, err := stateSetBalance(state, test.sender, test.input, test.gas)
			if err != test.err {
				t.Errorf("unexpected error, want %v, got %v", test.err, err)
			}
			if gas != 0 {
				t.Errorf("unexpected gas, want %v, got %v", 0, gas)
			}
		})
	}
}

func TestPrecompiled_StateCopyCode(t *testing.T) {
	codeSize := tosca.Gas(42)
	tests := map[string]struct {
		input []byte
		gas   tosca.Gas
		err   error
	}{
		"outOfGas": {
			make([]byte, 64),
			1,
			ErrOutOfGas,
		},
		"invalidInput": {
			make([]byte, 63),
			32000,
			ErrExecutionReverted,
		},
		"insufficientMemoryGas": {
			make([]byte, 64),
			32000,
			ErrOutOfGas,
		},
		"successful": {
			append(append(make([]byte, 12), 0x01), make([]byte, 19+32)...),
			32000 + codeSize*203,
			nil,
		},
	}

	for testName, test := range tests {
		t.Run(testName, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			state := tosca.NewMockWorldState(ctrl)
			state.EXPECT().GetCode(tosca.Address{}).Return(make([]byte, codeSize)).AnyTimes()
			if test.err == nil {
				state.EXPECT().SetCode(tosca.Address{0x01}, tosca.Code(make([]byte, codeSize)))
			}

			gas, err := stateCopyCode(state, test.input, test.gas)
			if err != test.err {
				t.Errorf("unexpected error, want %v, got %v", test.err, err)
			}
			if gas != 0 {
				t.Errorf("unexpected gas, want %v, got %v", 0, gas)
			}
		})
	}
}

func TestPrecompiled_StateSwapCode(t *testing.T) {
	codeSize := tosca.Gas(42)
	tests := map[string]struct {
		input []byte
		gas   tosca.Gas
		err   error
	}{
		"outOfGas": {
			make([]byte, 64),
			1,
			ErrOutOfGas,
		},
		"invalidInput": {
			make([]byte, 63),
			2 * 32000,
			ErrExecutionReverted,
		},
		"insufficientMemoryGas": {
			make([]byte, 64),
			2 * 32000,
			ErrOutOfGas,
		},
		"successful": {
			append(append(make([]byte, 12), 0x01), make([]byte, 19+32)...),
			2*32000 + codeSize*203,
			nil,
		},
	}

	for testName, test := range tests {
		t.Run(testName, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			state := tosca.NewMockWorldState(ctrl)
			state.EXPECT().GetCode(tosca.Address{0x00}).Return(make([]byte, codeSize)).AnyTimes()
			state.EXPECT().GetCode(tosca.Address{0x01}).Return(make([]byte, codeSize)).AnyTimes()
			if test.err == nil {
				state.EXPECT().SetCode(tosca.Address{0x00}, tosca.Code(make([]byte, codeSize)))
				state.EXPECT().SetCode(tosca.Address{0x01}, tosca.Code(make([]byte, codeSize)))
			}

			gas, err := stateSwapCode(state, test.input, test.gas)
			if err != test.err {
				t.Errorf("unexpected error, want %v, got %v", test.err, err)
			}
			if gas != 0 {
				t.Errorf("unexpected gas, want %v, got %v", 0, gas)
			}
		})
	}
}

func TestPrecompiled_StateSetStorage(t *testing.T) {
	tests := map[string]struct {
		input []byte
		gas   tosca.Gas
		err   error
	}{
		"outOfGas": {
			make([]byte, 96),
			1,
			ErrOutOfGas,
		},
		"invalidInput": {
			make([]byte, 63),
			20000,
			ErrExecutionReverted,
		},
		"successful": {
			make([]byte, 96),
			20000,
			nil,
		},
	}

	for testName, test := range tests {
		t.Run(testName, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			state := tosca.NewMockWorldState(ctrl)
			if test.err == nil {
				state.EXPECT().SetStorage(tosca.Address{}, tosca.Key{}, tosca.Word{})
			}

			gas, err := stateSetStorage(state, test.input, test.gas)
			if err != test.err {
				t.Errorf("unexpected error, want %v, got %v", test.err, err)
			}
			if gas != 0 {
				t.Errorf("unexpected gas, want %v, got %v", 0, gas)
			}
		})
	}
}

func TestPrecompiled_StateIncNonce(t *testing.T) {
	increment := byte(5)
	tests := map[string]struct {
		input  []byte
		gas    tosca.Gas
		sender tosca.Address
		err    error
	}{
		"outOfGas": {
			make([]byte, 64),
			1,
			tosca.Address{},
			ErrOutOfGas,
		},
		"invalidInput": {
			make([]byte, 63),
			9000,
			tosca.Address{},
			ErrExecutionReverted,
		},
		"sameAddress": {
			make([]byte, 64),
			9000,
			tosca.Address{},
			ErrExecutionReverted,
		},
		"invalidValue": {
			make([]byte, 64),
			9000,
			tosca.Address{0x01},
			ErrExecutionReverted,
		},
		"successful": {
			append(append(make([]byte, 32), increment), make([]byte, 31)...),
			9000,
			tosca.Address{0x01},
			nil,
		},
	}

	for testName, test := range tests {
		t.Run(testName, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			state := tosca.NewMockWorldState(ctrl)
			if test.err == nil {
				state.EXPECT().GetNonce(tosca.Address{}).Return(uint64(5))
				state.EXPECT().SetNonce(tosca.Address{}, uint64(5+increment))
			}

			gas, err := stateIncNonce(state, test.sender, test.input, test.gas)
			if err != test.err {
				t.Errorf("unexpected error, want %v, got %v", test.err, err)
			}
			if gas != 0 {
				t.Errorf("unexpected gas, want %v, got %v", 0, gas)
			}
		})
	}

}

func TestPrecompiled_HandleStatePrecompiled(t *testing.T) {
	codeSize := tosca.Gas(42)
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
			StateContractAddress,
			[]byte{0xe3, 0x4, 0x43, 0xbc},
			make([]byte, 64),
			true,
			func(mock *tosca.MockWorldState) {
				mock.EXPECT().SetBalance(tosca.Address{}, tosca.Value{})
			},
		},
		"copyCode": {
			StateContractAddress,
			[]byte{0xd6, 0xa0, 0xc7, 0xaf},
			append(append(make([]byte, 12), 0x01), make([]byte, 19+32)...),
			true,
			func(mock *tosca.MockWorldState) {
				mock.EXPECT().GetCode(tosca.Address{}).Return(make([]byte, codeSize))
				mock.EXPECT().SetCode(tosca.Address{0x01}, tosca.Code(make([]byte, codeSize)))
			},
		},
		"swapCode": {
			StateContractAddress,
			[]byte{0x7, 0x69, 0xb, 0x2a},
			append(append(make([]byte, 12), 0x01), make([]byte, 19+32)...),
			true,
			func(mock *tosca.MockWorldState) {
				mock.EXPECT().GetCode(tosca.Address{0x00}).Return(make([]byte, codeSize))
				mock.EXPECT().GetCode(tosca.Address{0x01}).Return(make([]byte, codeSize))
				mock.EXPECT().SetCode(tosca.Address{0x00}, tosca.Code(make([]byte, codeSize)))
				mock.EXPECT().SetCode(tosca.Address{0x01}, tosca.Code(make([]byte, codeSize)))
			},
		},
		"setStorage": {
			StateContractAddress,
			[]byte{0x39, 0xe5, 0x3, 0xab},
			make([]byte, 96),
			true,
			func(mock *tosca.MockWorldState) {
				mock.EXPECT().SetStorage(tosca.Address{}, tosca.Key{}, tosca.Word{})
			},
		},
		"incNonce": {
			StateContractAddress,
			[]byte{0x79, 0xbe, 0xad, 0x38},
			append(append(make([]byte, 32), increment), make([]byte, 31)...),
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

			sender := DriverAddress
			gas := tosca.Gas(1000000)
			input := append(test.inputPrefix, test.input...)

			result, isStatePrecompiled := handleStatePrecompiledContract(state, sender, test.recipient, input, gas)
			if isStatePrecompiled != test.isStatePrecompiled {
				t.Errorf("wrong state precompiled address, want %v, got %v", test.isStatePrecompiled, isStatePrecompiled)
			}
			if isStatePrecompiled && result.Success != true {
				t.Errorf("execution was not successful")
			}
		})
	}
}
