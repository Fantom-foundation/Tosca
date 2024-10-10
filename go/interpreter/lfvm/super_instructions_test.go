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

	"github.com/holiman/uint256"
)

func TestSI_opDup2_Lt(t *testing.T) {

	tests := map[string]struct {
		a, b   uint256.Int
		result uint256.Int
	}{
		"a<b": {
			a:      *uint256.NewInt(1),
			b:      *uint256.NewInt(2),
			result: *uint256.NewInt(0),
		},
		"a>b": {
			a:      *uint256.NewInt(1),
			b:      *uint256.NewInt(0),
			result: *uint256.NewInt(1),
		},
		"a==b": {
			a:      *uint256.NewInt(0),
			b:      *uint256.NewInt(0),
			result: *uint256.NewInt(0),
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {

			ctxt := getEmptyContext()
			ctxt.stack = fillStack(test.a, test.b)

			opDup2_Lt(&ctxt)

			if want, got := 2, ctxt.stack.stackPointer; want != got {
				t.Errorf("unexpected stack size, got %v, want %v", got, want)
			}

			if want, got := test.result, ctxt.stack.peek(); want.Cmp(got) != 0 {
				t.Errorf("unexpected result, got %v, expected %v", got, want)
			}
		})
	}
}
