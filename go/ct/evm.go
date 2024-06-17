// Copyright (c) 2024 Fantom Foundation
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at fantom.foundation/bsl11.
//
// Change Date: 2028-4-16
//
// On the date above, in accordance with the Business Source License, use of
// this software will be governed by the GNU Lesser General Public License v3.

package ct

import "github.com/Fantom-foundation/Tosca/go/ct/st"

// Evm represents the interface through which the CT can test a specific EVM implementation.
type Evm interface {
	// StepN executes up to N instructions on the given state, returning the resulting state or an error.
	// The function may modify the provided state to produce the result state.
	StepN(state *st.State, numSteps int) (*st.State, error)
}
