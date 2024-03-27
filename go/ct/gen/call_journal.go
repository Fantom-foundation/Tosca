package gen

import (
	"github.com/Fantom-foundation/Tosca/go/ct/common"
	"github.com/Fantom-foundation/Tosca/go/ct/st"
	"github.com/Fantom-foundation/Tosca/go/vm"
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
		Success:   rnd.Int31n(2) == 1,
		Output:    common.RandomBytes(rnd),
		GasCosts:  vm.Gas(rnd.Int63()),
		GasRefund: vm.Gas(rnd.Int63()),
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
