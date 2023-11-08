package ct

import "github.com/Fantom-foundation/Tosca/go/ct/st"

// Evm represents the interface through which the CT can test a specific EVM implementation.
type Evm interface {
	// StepN executes up to N instructions on the given state, returning the resulting state or an error.
	// The function may modify the provided state to produce the result state.
	StepN(state *st.State, numSteps int) (*st.State, error)
}
