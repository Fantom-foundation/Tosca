package gen

import (
	"errors"
	"math"
	"testing"

	"github.com/Fantom-foundation/Tosca/go/ct/st"
	"pgregory.net/rand"
)

func TestStateGenerator_UnconstrainedGeneratorCanProduceState(t *testing.T) {
	rnd := rand.New()
	generator := NewStateGenerator()
	if _, err := generator.Generate(rnd); err != nil {
		t.Fatalf("unexpected error during build: %v", err)
	}
}

////////////////////////////////////////////////////////////
// Status

func TestStateGenerator_SetStatusIsEnforced(t *testing.T) {
	statuses := []st.StatusCode{st.Running, st.Failed, st.Reverted}

	rnd := rand.New()
	for _, status := range statuses {
		generator := NewStateGenerator()
		generator.SetStatus(status)
		state, err := generator.Generate(rnd)
		if err != nil {
			t.Fatalf("unexpected error during build: %v", err)
		}
		if want, got := status, state.Status; want != got {
			t.Errorf("unexpected status, wanted %d, got %d", want, got)
		}
	}
}

func TestStateGenerator_ConflictingStatusesAreDetected(t *testing.T) {
	generator := NewStateGenerator()
	generator.SetStatus(st.Running)
	generator.SetStatus(st.Failed)
	rnd := rand.New()
	if _, err := generator.Generate(rnd); !errors.Is(err, ErrUnsatisfiable) {
		t.Errorf("unsatisfiable constraint not detected, got %v", err)
	}
}

func TestStateGenerator_NegativeStatusesAreDetected(t *testing.T) {
	generator := NewStateGenerator()
	generator.SetStatus(st.StatusCode(-12))
	rnd := rand.New()
	if _, err := generator.Generate(rnd); !errors.Is(err, ErrUnsatisfiable) {
		t.Errorf("unsatisfiable constraint not detected, got %v", err)
	}
}

func TestStateGenerator_NonConflictingStatusesAreAccepted(t *testing.T) {
	generator := NewStateGenerator()
	generator.SetStatus(st.Reverted)
	generator.SetStatus(st.Reverted)
	rnd := rand.New()
	if _, err := generator.Generate(rnd); err != nil {
		t.Errorf("generation failed: %v", err)
	}
}

////////////////////////////////////////////////////////////
// Revision

func TestStateGenerator_SetRevisionIsEnforced(t *testing.T) {
	revisions := []st.Revision{st.Istanbul, st.Berlin, st.London}

	rnd := rand.New()
	for _, revision := range revisions {
		generator := NewStateGenerator()
		generator.SetRevision(revision)
		state, err := generator.Generate(rnd)
		if err != nil {
			t.Fatalf("unexpected error during build: %v", err)
		}
		if want, got := revision, state.Revision; want != got {
			t.Errorf("unexpected revision, wanted %d, got %d", want, got)
		}
	}
}

func TestStateGenerator_ConflictingRevisionsAreDetected(t *testing.T) {
	generator := NewStateGenerator()
	generator.SetRevision(st.Istanbul)
	generator.SetRevision(st.London)
	rnd := rand.New()
	if _, err := generator.Generate(rnd); !errors.Is(err, ErrUnsatisfiable) {
		t.Errorf("unsatisfiable constraint not detected, got %v", err)
	}
}

func TestStateGenerator_NegativeRevisionsAreDetected(t *testing.T) {
	generator := NewStateGenerator()
	generator.SetRevision(st.Revision(-12))
	rnd := rand.New()
	if _, err := generator.Generate(rnd); !errors.Is(err, ErrUnsatisfiable) {
		t.Errorf("unsatisfiable constraint not detected, got %v", err)
	}
}

func TestStateGenerator_NonConflictingRevisionsAreAccepted(t *testing.T) {
	generator := NewStateGenerator()
	generator.SetRevision(st.London)
	generator.SetRevision(st.London)
	rnd := rand.New()
	if _, err := generator.Generate(rnd); err != nil {
		t.Errorf("generation failed: %v", err)
	}
}

////////////////////////////////////////////////////////////
// Program Counter

func TestStateGenerator_SetPcIsEnforced(t *testing.T) {
	pcs := []uint16{0, 2, 4}

	rnd := rand.New()
	for _, pc := range pcs {
		generator := NewStateGenerator()
		generator.SetCodeSize(16)
		generator.SetPc(pc)
		state, err := generator.Generate(rnd)
		if err != nil {
			t.Fatalf("unexpected error during build: %v", err)
		}
		if want, got := pc, state.Pc; want != got {
			t.Errorf("unexpected program counter, wanted %d, got %d", want, got)
		}
	}
}

func TestStateGenerator_ConflictingPcesAreDetected(t *testing.T) {
	generator := NewStateGenerator()
	generator.SetCodeSize(16)
	generator.SetPc(0)
	generator.SetPc(1)
	rnd := rand.New()
	if _, err := generator.Generate(rnd); !errors.Is(err, ErrUnsatisfiable) {
		t.Errorf("unsatisfiable constraint not detected, got %v", err)
	}
}

func TestStateGenerator_NonConflictingPcesAreAccepted(t *testing.T) {
	generator := NewStateGenerator()
	generator.SetCodeSize(16)
	generator.SetPc(1)
	generator.SetPc(1)
	rnd := rand.New()
	if _, err := generator.Generate(rnd); err != nil {
		t.Errorf("generation failed: %v", err)
	}
}

////////////////////////////////////////////////////////////
// Gas Counter

func TestStateGenerator_SetGasIsEnforced(t *testing.T) {
	gasCounts := []uint64{0, 42, math.MaxUint64}

	rnd := rand.New()
	for _, gas := range gasCounts {
		generator := NewStateGenerator()
		generator.SetGas(gas)
		state, err := generator.Generate(rnd)
		if err != nil {
			t.Fatalf("unexpected error during build: %v", err)
		}
		if want, got := gas, state.Gas; want != got {
			t.Errorf("unexpected amount of gas, wanted %d, got %d", want, got)
		}
	}
}

func TestStateGenerator_ConflictingGasAreDetected(t *testing.T) {
	generator := NewStateGenerator()
	generator.SetGas(0)
	generator.SetGas(42)
	rnd := rand.New()
	if _, err := generator.Generate(rnd); !errors.Is(err, ErrUnsatisfiable) {
		t.Errorf("unsatisfiable constraint not detected, got %v", err)
	}
}

func TestStateGenerator_NonConflictingGasAreAccepted(t *testing.T) {
	generator := NewStateGenerator()
	generator.SetGas(42)
	generator.SetGas(42)
	rnd := rand.New()
	if _, err := generator.Generate(rnd); err != nil {
		t.Errorf("generation failed: %v", err)
	}
}

////////////////////////////////////////////////////////////
// Clone / Restore

func TestStateGenerator_CloneCopiesBuilderState(t *testing.T) {
	original := NewStateGenerator()
	original.SetStatus(st.Reverted)
	original.SetRevision(st.London)
	original.SetPc(4)
	original.SetGas(5)
	original.SetCodeSize(42)

	clone := original.Clone()

	if want, got := original.String(), clone.String(); want != got {
		t.Errorf("invalid clone, wanted %s, got %s", want, got)
	}
}

func TestStateGenerator_ClonesAreIndependent(t *testing.T) {
	base := NewStateGenerator()
	base.SetPc(4)

	clone1 := base.Clone()
	clone1.SetStatus(st.Reverted)
	clone1.SetRevision(st.London)
	clone1.SetGas(5)
	clone1.SetCodeSize(20)

	clone2 := base.Clone()
	clone2.SetStatus(st.Running)
	clone2.SetRevision(st.Berlin)
	clone2.SetGas(6)
	clone2.SetCodeSize(30)

	want := "{status=reverted,revision=London,pc=4,gas=5,code={size=20}}"
	if got := clone1.String(); want != got {
		t.Errorf("invalid clone, wanted %s, got %s", want, got)
	}

	want = "{status=running,revision=Berlin,pc=4,gas=6,code={size=30}}"
	if got := clone2.String(); want != got {
		t.Errorf("invalid clone, wanted %s, got %s", want, got)
	}
}

func TestStateGenerator_CloneCanBeUsedToResetBuilder(t *testing.T) {
	gen := NewStateGenerator()
	gen.SetPc(4)

	backup := gen.Clone()

	gen.SetGas(42)
	want := "{pc=4,gas=42,code={}}"
	if got := gen.String(); want != got {
		t.Errorf("invalid clone, wanted %s, got %s", want, got)
	}

	gen.Restore(backup)

	want = "{pc=4,code={}}"
	if got := gen.String(); want != got {
		t.Errorf("invalid clone, wanted %s, got %s", want, got)
	}
}
