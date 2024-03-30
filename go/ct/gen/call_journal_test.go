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
