// Copyright (c) 2024 Fantom Foundation
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at fantom.foundation/bsl11.
//
// Change Date: 2028-4-16
//
// On the date above, in accordance with the Business Source License, use of
// this software will be governed by the GNU Lesser General Public License v3.

package gen

import (
	"testing"

	"github.com/Fantom-foundation/Tosca/go/ct/common"
)

func TestAssignment_String(t *testing.T) {

	tests := map[string]struct {
		value common.U256
	}{
		"a": {common.NewU256(1)},
		"b": {common.NewU256(2, 2)},
		"c": {common.NewU256(3, 3, 3)},
		"d": {common.NewU256(4, 4, 4, 4)},
	}

	for variable, test := range tests {
		a := make(Assignment)
		a[Variable(variable)] = test.value

		got := a.String()
		want := "{" + variable + "->" + test.value.String() + "}"

		if got != want {
			t.Errorf("Assignment.String() = %v, want %v", got, want)
		}
	}

	if a := Assignment(nil); a.String() != "{}" {
		t.Errorf("Assignment.String() = %v, want {}", a.String())
	}

}
