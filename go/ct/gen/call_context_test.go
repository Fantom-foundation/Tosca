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
	"github.com/Fantom-foundation/Tosca/go/tosca"
	"pgregory.net/rand"
)

func TestCallContextGen_Generate(t *testing.T) {
	rnd := rand.New(0)
	callCtxGen := NewCallContextGenerator()
	accountAddress := common.RandomAddress(rnd)
	callCtx, err := callCtxGen.Generate(rnd, accountAddress)
	if err != nil {
		t.Errorf("Error generating call context: %v", err)
	}

	if callCtx.AccountAddress == (tosca.Address{}) {
		t.Errorf("Generated account address has default value.")
	}
	if callCtx.CallerAddress == (tosca.Address{}) {
		t.Errorf("Generated caller address has default value.")
	}
	if callCtx.Value.Eq(common.NewU256(0)) {
		t.Errorf("Generated call value has default value.")
	}
}
