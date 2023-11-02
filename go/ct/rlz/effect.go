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
