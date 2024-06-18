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
	"fmt"

	"github.com/Fantom-foundation/Tosca/go/ct/st"
)

type Effect interface {
	// Apply modifies the given state with this effect.
	Apply(*st.State)

	fmt.Stringer
}

////////////////////////////////////////////////////////////
// Change

type change struct {
	fun func(*st.State)
}

func Change(fun func(*st.State)) Effect {
	return &change{fun}
}

func (c *change) Apply(state *st.State) {
	c.fun(state)
}

func (c *change) String() string {
	return "change"
}

////////////////////////////////////////////////////////////

func NoEffect() Effect {
	return Change(func(*st.State) {})
}

func FailEffect() Effect {
	return Change(func(s *st.State) {
		s.Status = st.Failed
		s.Gas = 0
	})
}
