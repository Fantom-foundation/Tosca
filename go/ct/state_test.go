package ct

import (
	"testing"
)

func TestStateBuilder_CanBuildRandomState(t *testing.T) {
	GetRandomState()
}

func TestStateBuilder_SameSeedProducesSameState(t *testing.T) {
	s1 := GetRandomStateWithSeed(12)
	s2 := GetRandomStateWithSeed(12)
	if !s1.Equal(&s2) {
		t.Errorf("expected same states, got %v and %v", s1, s2)
	}
}

func TestStateBuilder_DifferentSeedsProduceDifferentState(t *testing.T) {
	s1 := GetRandomStateWithSeed(12)
	s2 := GetRandomStateWithSeed(14)
	if s1.Equal(&s2) {
		t.Errorf("expected different states, got %v and %v", s1, s2)
	}
}

func TestStateBuilder_StateWithFixedCodeLength(t *testing.T) {
	b := NewStateBuilder()
	b.SetCodeLength(12)
	state := b.Build()
	if got, want := len(state.Code), 12; got != want {
		t.Errorf("invalid length of resulting code, wanted %d, got %d", want, got)
	}
}

/*
func TestStateBuilder_GenerateRandomStates(t *testing.T) {
	for i := 0; i < 5; i++ {
		s := GetRandomState()
		fmt.Printf("%v\n", s)
	}
	t.Fail()
}
*/
