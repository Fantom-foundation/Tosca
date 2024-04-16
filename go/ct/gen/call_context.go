//
// Copyright (c) 2024 Fantom Foundation
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at fantom.foundation/bsl11.
//
// Change Date: 2028-4-16
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by the GNU Lesser General Public Licence v3
//

package gen

import (
	"pgregory.net/rand"

	"github.com/Fantom-foundation/Tosca/go/ct/common"
	"github.com/Fantom-foundation/Tosca/go/ct/st"
	"github.com/Fantom-foundation/Tosca/go/vm"
)

type CallContextGenerator struct {
}

func NewCallContextGenerator() *CallContextGenerator {
	return &CallContextGenerator{}
}

func (*CallContextGenerator) Generate(rnd *rand.Rand, accountAddress vm.Address) (st.CallContext, error) {
	originAddress, err := common.RandAddress(rnd)
	if err != nil {
		return st.CallContext{}, err
	}

	callerAddress, err := common.RandAddress(rnd)
	if err != nil {
		return st.CallContext{}, err
	}

	return st.CallContext{
		AccountAddress: accountAddress,
		OriginAddress:  originAddress,
		CallerAddress:  callerAddress,
		Value:          common.RandU256(rnd),
	}, nil
}

func (*CallContextGenerator) Clone() *CallContextGenerator {
	return &CallContextGenerator{}
}

func (*CallContextGenerator) Restore(*CallContextGenerator) {
}

func (*CallContextGenerator) String() string {
	return "{}"
}
