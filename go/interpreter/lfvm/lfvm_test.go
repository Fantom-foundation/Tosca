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
	"testing"

	"github.com/Fantom-foundation/Tosca/go/tosca"
)

func TestVm_Run(t *testing.T) {

	tests := map[string]struct {
		code           []byte
		revision       tosca.Revision
		expectedResult tosca.Result
		expectedError  *tosca.ErrUnsupportedRevision
	}{
		"empty code": {
			revision: tosca.R13_Cancun,
			expectedResult: tosca.Result{
				Success:   true,
				GasLeft:   1000000,
				GasRefund: 0,
				Output:    []byte{}},
		},
		"invalid code": {
			code:     []byte{0x0C},
			revision: tosca.R13_Cancun,
			expectedResult: tosca.Result{
				Success:   false,
				GasLeft:   0,
				GasRefund: 0,
				Output:    []byte{}},
		},
		"newer unsupported revision": {
			revision: newestSupportedRevision + 1,
			expectedError: &tosca.ErrUnsupportedRevision{
				Revision: newestSupportedRevision + 1},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {

			params := tosca.Parameters{
				Gas:      1000000,
				Code:     test.code,
				CodeHash: &tosca.Hash{},
				BlockParameters: tosca.BlockParameters{
					Revision: test.revision,
				},
			}

			vm, err := NewVm(Config{
				ConversionConfig: ConversionConfig{
					WithSuperInstructions: false,
				},
				NoShaCache: true,
			})
			if err != nil {
				t.Fatalf("failed to create vm: %v", err)
			}

			result, err := vm.Run(params)
			// TODO: simplify error checking when ErrUnsoportedRevision is a tosca.ConstErr
			if err != nil || test.expectedError != nil {
				if !errors.As(err, &test.expectedError) {
					t.Errorf("unexpected error: got %v (type %T), want %v (type %T)", err, err, test.expectedError, test.expectedError)
				} else {
					// if err is not nil and is as expected, we can return
					return
				}
			}

			if result.Success != test.expectedResult.Success {
				t.Errorf("unexpected result, want %v but got %v",
					test.expectedResult.Success, result.Success)
			}
			if result.GasLeft != test.expectedResult.GasLeft {
				t.Errorf("unexpected GasLeft, want %v but got %v",
					test.expectedResult.GasLeft, result.GasLeft)
			}
			if result.GasRefund != test.expectedResult.GasRefund {
				t.Errorf("unexpected GasRefund, want %v but got %v",
					test.expectedResult.GasRefund, result.GasRefund)
			}
			if !bytes.Equal(result.Output, test.expectedResult.Output) {
				t.Errorf("unexpected Output, want %v but got %v",
					test.expectedResult.Output, result.Output)
			}

		})
	}

}
