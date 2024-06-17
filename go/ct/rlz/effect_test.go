// Copyright (c) 2024 Fantom Foundation
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at fantom.foundation/bsl11.
//
// Change Date: 2028-4-16
//
// On the date above, in accordance with the Business Source License, use of
// this software will be governed by the GNU Lesser General Public License v3.

package rlz

import (
	"testing"

	"github.com/Fantom-foundation/Tosca/go/ct/st"
)

func TestEffect_Change(t *testing.T) {
	pcAdd1 := Change(func(s *st.State) {
		s.Pc += 1
	})

	state := st.NewState(st.NewCode([]byte{}))
	state.Pc = 0

	pcAdd1.Apply(state)
	if state.Pc != 1 {
		t.Errorf("effect did not apply")
	}
}
