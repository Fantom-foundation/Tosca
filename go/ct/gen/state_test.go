package gen

import (
	"errors"
	"math"
	"slices"
	"strings"
	"testing"

	"pgregory.net/rand"

	. "github.com/Fantom-foundation/Tosca/go/ct/common"
	"github.com/Fantom-foundation/Tosca/go/ct/st"
	"github.com/Fantom-foundation/Tosca/go/vm"
)

func TestStateGenerator_UnconstrainedGeneratorCanProduceState(t *testing.T) {
	rnd := rand.New(0)
	generator := NewStateGenerator()
	if _, err := generator.Generate(rnd); err != nil {
		t.Fatalf("unexpected error during build: %v", err)
	}
}

////////////////////////////////////////////////////////////
// Status

func TestStateGenerator_SetStatusIsEnforced(t *testing.T) {
	statuses := []st.StatusCode{st.Running, st.Failed, st.Reverted}

	rnd := rand.New(0)
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
	rnd := rand.New(0)
	if _, err := generator.Generate(rnd); !errors.Is(err, ErrUnsatisfiable) {
		t.Errorf("unsatisfiable constraint not detected, got %v", err)
	}
}

func TestStateGenerator_NegativeStatusesAreDetected(t *testing.T) {
	generator := NewStateGenerator()
	generator.SetStatus(st.StatusCode(-12))
	rnd := rand.New(0)
	if _, err := generator.Generate(rnd); !errors.Is(err, ErrUnsatisfiable) {
		t.Errorf("unsatisfiable constraint not detected, got %v", err)
	}
}

func TestStateGenerator_NonConflictingStatusesAreAccepted(t *testing.T) {
	generator := NewStateGenerator()
	generator.SetStatus(st.Reverted)
	generator.SetStatus(st.Reverted)
	rnd := rand.New(0)
	if _, err := generator.Generate(rnd); err != nil {
		t.Errorf("generation failed: %v", err)
	}
}

////////////////////////////////////////////////////////////
// Revision

func TestStateGenerator_SetRevisionIsEnforced(t *testing.T) {
	revisions := []Revision{R07_Istanbul, R09_Berlin, R10_London}

	rnd := rand.New(0)
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
	generator.SetRevision(R07_Istanbul)
	generator.SetRevision(R10_London)
	rnd := rand.New(0)
	if _, err := generator.Generate(rnd); !errors.Is(err, ErrUnsatisfiable) {
		t.Errorf("unsatisfiable constraint not detected, got %v", err)
	}
}

func TestStateGenerator_NegativeRevisionsAreDetected(t *testing.T) {
	generator := NewStateGenerator()
	generator.SetRevision(Revision(-12))
	rnd := rand.New(0)
	if _, err := generator.Generate(rnd); !errors.Is(err, ErrUnsatisfiable) {
		t.Errorf("unsatisfiable constraint not detected, got %v", err)
	}
}

func TestStateGenerator_NonConflictingRevisionsAreAccepted(t *testing.T) {
	generator := NewStateGenerator()
	generator.SetRevision(R10_London)
	generator.SetRevision(R10_London)
	rnd := rand.New(0)
	if _, err := generator.Generate(rnd); err != nil {
		t.Errorf("generation failed: %v", err)
	}
}

func TestStateGenerator_SetRevisionBoundsIsEnforced(t *testing.T) {
	generator := NewStateGenerator()
	generator.SetRevisionBounds(R07_Istanbul, R09_Berlin)
	generator.SetRevisionBounds(R09_Berlin, R10_London)

	state, err := generator.Generate(rand.New(0))
	if err != nil {
		t.Fatalf("unexpected error during build: %v", err)
	}
	if want, got := R09_Berlin, state.Revision; want != got {
		t.Fatalf("Revision bounds not working, want %v, got %v", want, got)
	}
}

func TestStateGenerator_ConflictingRevisionBoundsAreDetected(t *testing.T) {
	generator := NewStateGenerator()
	generator.SetRevisionBounds(R07_Istanbul, R09_Berlin)
	generator.SetRevisionBounds(R10_London, R99_UnknownNextRevision)

	if _, err := generator.Generate(rand.New(0)); !errors.Is(err, ErrUnsatisfiable) {
		t.Errorf("unsatisfiable constraint not detected, got %v", err)
	}
}

////////////////////////////////////////////////////////////
// Program Counter

func TestStateGenerator_SetPcIsEnforced(t *testing.T) {
	pcs := []uint16{0, 2, 4}

	rnd := rand.New(0)
	for _, pc := range pcs {
		generator := NewStateGenerator()
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
	generator.SetPc(0)
	generator.SetPc(1)
	rnd := rand.New(0)
	if _, err := generator.Generate(rnd); !errors.Is(err, ErrUnsatisfiable) {
		t.Errorf("unsatisfiable constraint not detected, got %v", err)
	}
}

func TestStateGenerator_NonConflictingPcesAreAccepted(t *testing.T) {
	generator := NewStateGenerator()
	generator.SetPc(1)
	generator.SetPc(1)
	rnd := rand.New(0)
	if _, err := generator.Generate(rnd); err != nil {
		t.Errorf("generation failed: %v", err)
	}
}

////////////////////////////////////////////////////////////
// Gas Counter

func TestStateGenerator_SetGasIsEnforced(t *testing.T) {
	gasCounts := []vm.Gas{0, 42, st.MaxGas}

	rnd := rand.New(0)
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
	rnd := rand.New(0)
	if _, err := generator.Generate(rnd); !errors.Is(err, ErrUnsatisfiable) {
		t.Errorf("unsatisfiable constraint not detected, got %v", err)
	}
}

func TestStateGenerator_NonConflictingGasAreAccepted(t *testing.T) {
	generator := NewStateGenerator()
	generator.SetGas(42)
	generator.SetGas(42)
	rnd := rand.New(0)
	if _, err := generator.Generate(rnd); err != nil {
		t.Errorf("generation failed: %v", err)
	}
}

////////////////////////////////////////////////////////////
// Gas Refund Counter

func TestStateGenerator_SetGasRefundIsEnforced(t *testing.T) {
	gasRefundCounts := []vm.Gas{0, 42, math.MaxInt64}

	rnd := rand.New(0)
	for _, gasRefund := range gasRefundCounts {
		generator := NewStateGenerator()
		generator.SetGasRefund(gasRefund)
		state, err := generator.Generate(rnd)
		if err != nil {
			t.Fatalf("unexpected error during build: %v", err)
		}
		if want, got := gasRefund, state.GasRefund; want != got {
			t.Errorf("unexpected amount of gas refund, wanted %d, got %d", want, got)
		}
	}
}

func TestStateGenerator_ConflictingGasRefundAreDetected(t *testing.T) {
	generator := NewStateGenerator()
	generator.SetGasRefund(0)
	generator.SetGasRefund(42)
	rnd := rand.New(0)
	if _, err := generator.Generate(rnd); !errors.Is(err, ErrUnsatisfiable) {
		t.Errorf("unsatisfiable constraint not detected, got %v", err)
	}
}

func TestStateGenerator_NonConflictingGasRefundAreAccepted(t *testing.T) {
	generator := NewStateGenerator()
	generator.SetGasRefund(42)
	generator.SetGasRefund(42)
	rnd := rand.New(0)
	if _, err := generator.Generate(rnd); err != nil {
		t.Errorf("generation failed: %v", err)
	}
}

////////////////////////////////////////////////////////////
// Clone / Restore

func TestStateGenerator_CloneCopiesBuilderState(t *testing.T) {
	original := NewStateGenerator()
	original.SetStatus(st.Reverted)
	original.SetRevision(R10_London)
	original.SetPc(4)
	original.SetGas(5)
	original.SetGasRefund(6)
	original.BindValue(Variable("x"), NewU256(12))

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
	clone1.SetRevision(R10_London)
	clone1.SetGas(5)
	clone1.SetGasRefund(6)
	clone1.SetCodeOperation(20, ADD)
	clone1.SetMinStackSize(2)
	clone1.SetMaxStackSize(200)
	clone1.BindValue(Variable("x"), NewU256(12))

	clone2 := base.Clone()
	clone2.SetStatus(st.Running)
	clone2.SetRevision(R09_Berlin)
	clone2.SetGas(7)
	clone2.SetGasRefund(8)
	clone2.SetCodeOperation(30, ADD)
	clone2.SetMinStackSize(3)
	clone2.SetMaxStackSize(300)
	clone2.BindValue(Variable("y"), NewU256(14))

	checkPrint := func(clone *StateGenerator, want []string) {
		t.Helper()
		str := clone.String()
		if len(str) < 2 {
			t.Fatalf("unexpected print format: %v", str)
		}
		str = str[1 : len(str)-1]
		if got := strings.Split(str, ","); !slices.Equal(want, got) {
			t.Errorf("invalid clone, wanted %s, got %s", want, got)
		}
	}

	checkPrint(clone1, []string{
		"$x=0000000000000000 0000000000000000 0000000000000000 000000000000000c",
		"status=reverted",
		"revision=London",
		"pc=4",
		"gas=5",
		"gasRefund=6",
		"code={op[20]=ADD}",
		"stack={2≤size≤200}",
		"memory={}",
		"storage={}",
		"accounts={}",
		"callContext={}",
		"callJournal={}",
		"blockContext={}",
	})

	checkPrint(clone2, []string{
		"$y=0000000000000000 0000000000000000 0000000000000000 000000000000000e",
		"status=running",
		"revision=Berlin",
		"pc=4",
		"gas=7",
		"gasRefund=8",
		"code={op[30]=ADD}",
		"stack={3≤size≤300}",
		"memory={}",
		"storage={}",
		"accounts={}",
		"callContext={}",
		"callJournal={}",
		"blockContext={}",
	})
}

func TestStateGenerator_CloneCanBeUsedToResetBuilder(t *testing.T) {
	gen := NewStateGenerator()
	gen.SetPc(4)

	backup := gen.Clone()

	gen.SetGas(42)
	gen.SetGasRefund(17)
	if base, modified := backup.String(), gen.String(); base == modified {
		t.Errorf("clones are not independent")
	}

	gen.Restore(backup)

	if want, got := backup.String(), gen.String(); want != got {
		t.Errorf("restore did not work, wanted %s, got %s", want, got)
	}
}

// //////////////////////////////////////////////////////////

func genRandomState(t *testing.T) *st.State {
	t.Helper()
	gen := NewStateGenerator()
	rnd := rand.New(0)
	state, err := gen.Generate(rnd)
	if err != nil {
		t.Fatalf("error generating new state: %v", err)
	}
	return state
}

func testData(data []byte, name string, t *testing.T) {
	t.Helper()
	if len(data) == 0 {
		t.Errorf("failed to generate a non-empty %v", name)
	} else {
		allzeros := true
		for b := range data {
			if b != 0 {
				allzeros = false
				break
			}
		}
		if allzeros {
			t.Errorf("failed to generate a non-zero %v", name)
		}
	}
}

// //////////////////////////////////////////////////////////
// Call data
// Last Call Return data

func TestStateGenerator_DataGeneration(t *testing.T) {
	state := genRandomState(t)
	testData(state.CallData, "call data", t)
	testData(state.LastCallReturnData, "last call return data", t)
}

// //////////////////////////////////////////////////////////
// Return data should always be empty

func TestStateGenerator_ReturnDataShouldBeEmpty(t *testing.T) {
	state := genRandomState(t)
	if want, got := 0, len(state.ReturnData); want != got {
		t.Errorf("unexpected length of generated return data, wanted %d, got %d", want, got)
	}
}
