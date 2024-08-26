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
	"strings"
	"testing"

	"github.com/Fantom-foundation/Tosca/go/tosca"
)

func TestVm_Run(t *testing.T) {

	tests := map[string]struct {
		code                  []byte
		codeHash              tosca.Hash
		revision              tosca.Revision
		withSuperInstructions bool
		withCodeCache         bool
		expectedResult        tosca.Result
		expectedError         error
	}{
		"empty code": {
			revision: tosca.R13_Cancun,
			expectedResult: tosca.Result{Success: true, GasLeft: 1000000,
				GasRefund: 0, Output: []byte{}},
		},
		"invalid code": {
			code:     []byte{0x0C},
			revision: tosca.R13_Cancun,
			expectedResult: tosca.Result{Success: false, GasLeft: 0,
				GasRefund: 0, Output: []byte{}},
		},
		"newer unsupported revision": {
			revision:      newestSupportedRevision + 1,
			expectedError: &tosca.ErrUnsupportedRevision{Revision: newestSupportedRevision + 1},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {

			params := tosca.Parameters{
				Gas:      1000000,
				Code:     test.code,
				CodeHash: &test.codeHash,
				BlockParameters: tosca.BlockParameters{
					Revision: test.revision,
				},
			}

			vm := &VM{
				with_super_instructions: test.withSuperInstructions,
				no_code_cache:           !test.withCodeCache,
			}

			result, err := vm.Run(params)
			if err != test.expectedError && strings.Compare(err.Error(), test.expectedError.Error()) != 0 {
				t.Fatalf("unexpected error: want %v but got %v", test.expectedError, err)
			}
			if test.expectedError != nil {
				return
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
