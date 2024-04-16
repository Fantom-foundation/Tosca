//
// Copyright (c) 2024 Fantom Foundation
//
// Use of this software is governed by the Business Source License included
// in the LICENSE.TXT file and at fantom.foundation/bsl11.
//
// Change Date: 2028-4-16
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by the GNU Lesser General Public Licence v3
//

package gen

import (
	"testing"

	"pgregory.net/rand"
)

func TestCallJournalGenerator_CanProduceNonEmptyJournal(t *testing.T) {
	rnd := rand.New(0)
	generator := NewCallJournalGenerator()
	journal, err := generator.Generate(rnd)
	if err != nil {
		t.Fatalf("failed to generate journal, err: %v", err)
	}
	if len(journal.Past) != 0 {
		t.Errorf("the generator should not produce past calls")
	}
	if len(journal.Future) != 1 {
		t.Errorf("expected exactly one future call")
	}
}
