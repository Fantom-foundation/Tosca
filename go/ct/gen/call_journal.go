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
	"github.com/Fantom-foundation/Tosca/go/ct/common"
	"github.com/Fantom-foundation/Tosca/go/ct/st"
	"github.com/Fantom-foundation/Tosca/go/tosca"
	"pgregory.net/rand"
)

type CallJournalGenerator struct {
}

func NewCallJournalGenerator() *CallJournalGenerator {
	return &CallJournalGenerator{}
}

func (*CallJournalGenerator) Generate(rnd *rand.Rand) (*st.CallJournal, error) {
	journal := st.NewCallJournal()

	// One future call is enough for any single instruction.
	journal.Future = append(journal.Future, st.FutureCall{
		Success:        rnd.Int31n(2) == 1,
		Output:         common.RandomBytes(rnd, 2000),
		GasCosts:       tosca.Gas(rnd.Int63()),
		GasRefund:      tosca.Gas(rnd.Int63()),
		CreatedAccount: common.RandomAddress(rnd),
	})

	return journal, nil
}

func (*CallJournalGenerator) Clone() *CallJournalGenerator {
	return &CallJournalGenerator{}
}

func (*CallJournalGenerator) Restore(*CallJournalGenerator) {
}

func (*CallJournalGenerator) String() string {
	return "{}"
}
