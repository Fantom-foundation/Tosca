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

func TestEffect_String(t *testing.T) {
	pcAdd1 := Change(func(s *st.State) {
		s.Pc += 1
	})

	if pcAdd1.String() != "change" {
		t.Errorf("effect string is wrong")
	}
}

func TestEffect_NoEffect(t *testing.T) {
	original := st.NewState(st.NewCode([]byte{}))
	original.Pc = 1

	clone := original.Clone()
	NoEffect().Apply(clone)

	if !original.Eq(clone) {
		t.Errorf("effect should have changed anythiong")
	}
}

func TestEffect_FailEffect(t *testing.T) {
	state := st.NewState(st.NewCode([]byte{}))
	state.Status = st.Running
	state.Gas = 10

	FailEffect().Apply(state)

	if state.Status != st.Failed {
		t.Errorf("effect should have failed the state")
	}
	if state.Gas != 0 {
		t.Errorf("effect should have set gas to 0")
	}
}
