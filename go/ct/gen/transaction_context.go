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
)

type TransactionContextGenerator struct {
}

func NewTransactionContextGenerator() *TransactionContextGenerator {
	return &TransactionContextGenerator{}
}

func (*TransactionContextGenerator) Generate(rnd *rand.Rand) (st.TransactionContext, error) {
	originAddress := common.RandomAddress(rnd)

	return st.TransactionContext{
		OriginAddress: originAddress,
	}, nil
}

func (*TransactionContextGenerator) Clone() *TransactionContextGenerator {
	return &TransactionContextGenerator{}
}

func (*TransactionContextGenerator) Restore(*TransactionContextGenerator) {
}

func (*TransactionContextGenerator) String() string {
	return "{}"
}
