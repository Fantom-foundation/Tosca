package gen

import (
	"errors"
	"testing"

	"pgregory.net/rand"

	. "github.com/Fantom-foundation/Tosca/go/ct/common"
)

func TestStackGenerator_UnconstrainedGeneratorCanProduceStack(t *testing.T) {
	rnd := rand.New(0)
	generator := NewStackGenerator()
	if _, err := generator.Generate(rnd); err != nil {
		t.Fatalf("unexpected error during build: %v", err)
	}
}

func TestStackGenerator_SetSizeIsEnforced(t *testing.T) {
	sizes := []int{0, 1, 2, 42}

	rnd := rand.New(0)
	for _, size := range sizes {
		generator := NewStackGenerator()
		generator.SetSize(size)
		stack, err := generator.Generate(rnd)
		if err != nil {
			t.Fatalf("unexpected error during build: %v", err)
		}
		if want, got := size, stack.Size(); want != got {
			t.Errorf("unexpected stack size, wanted %d, got %d", want, got)
		}
	}
}

func TestStackGenerator_NonConflictingSizesAreAccepted(t *testing.T) {
	generator := NewStackGenerator()
	generator.SetSize(12)
	generator.SetSize(12)
	rnd := rand.New(0)
	if _, err := generator.Generate(rnd); err != nil {
		t.Errorf("generation failed: %v", err)
	}
}

func TestStackGenerator_ConflictingSizesAreDetected(t *testing.T) {
	generator := NewStackGenerator()
	generator.SetSize(12)
	generator.SetSize(14)
	rnd := rand.New(0)
	if _, err := generator.Generate(rnd); !errors.Is(err, ErrUnsatisfiable) {
		t.Errorf("unsatisfiable constraint not detected, got %v", err)
	}
}

func TestStackGenerator_SetValueIsEnforced(t *testing.T) {
	type value struct {
		pos   int
		value U256
	}
	tests := [][]value{
		{{0, NewU256(1)}},
		{{3, NewU256(6)}},
		{{42, NewU256(21)}},
	}

	rnd := rand.New(0)
	for _, test := range tests {
		generator := NewStackGenerator()

		for _, v := range test {
			generator.SetValue(v.pos, v.value)
		}

		stack, err := generator.Generate(rnd)
		if err != nil {
			t.Fatalf("unexpected error during build: %v", err)
		}

		for _, v := range test {
			if want, got := v.value, stack.Get(v.pos); want != got {
				t.Errorf("invalid value at %d, wanted %s, got %s", v.pos, want, got)
			}
		}
	}
}

func TestStackGenerator_NegativeValuePositionsAreDetected(t *testing.T) {
	generator := NewStackGenerator()
	generator.SetValue(-1, NewU256(42))
	rnd := rand.New(0)
	if _, err := generator.Generate(rnd); !errors.Is(err, ErrUnsatisfiable) {
		t.Errorf("unsatisfiable constraint not detected, got %v", err)
	}
}

func TestStackGenerator_NonConflictingValuesAreAccepted(t *testing.T) {
	generator := NewStackGenerator()
	generator.SetValue(0, NewU256(42))
	generator.SetValue(0, NewU256(42))
	rnd := rand.New(0)
	if _, err := generator.Generate(rnd); err != nil {
		t.Errorf("generation failed: %v", err)
	}
}

func TestStackGenerator_ConflictingValuesAreDetected(t *testing.T) {
	generator := NewStackGenerator()
	generator.SetValue(0, NewU256(42))
	generator.SetValue(0, NewU256(21))
	rnd := rand.New(0)
	if _, err := generator.Generate(rnd); !errors.Is(err, ErrUnsatisfiable) {
		t.Errorf("unsatisfiable constraint not detected, got %v", err)
	}
}

func TestStackGenerator_ConflictingValuePositionsWithSizesAreDetected(t *testing.T) {
	generator := NewStackGenerator()
	generator.SetSize(10)
	generator.SetValue(10, NewU256(21))
	rnd := rand.New(0)
	if _, err := generator.Generate(rnd); !errors.Is(err, ErrUnsatisfiable) {
		t.Errorf("unsatisfiable constraint not detected, got %v", err)
	}
}

func TestStackGenerator_CloneCopiesGeneratorState(t *testing.T) {
	original := NewStackGenerator()
	original.SetSize(5)
	original.SetValue(0, NewU256(42))
	original.SetValue(0, NewU256(43))

	clone := original.Clone()

	if want, got := original.String(), clone.String(); want != got {
		t.Errorf("invalid clone, wanted %s, got %s", want, got)
	}
}

func TestStackGenerator_ClonesAreIndependent(t *testing.T) {
	base := NewStackGenerator()
	base.SetSize(5)

	clone1 := base.Clone()
	clone1.SetValue(0, NewU256(16))

	clone2 := base.Clone()
	clone2.SetValue(0, NewU256(17))

	want := "{size=5,value[0]=0000000000000000 0000000000000000 0000000000000000 0000000000000010}"
	if got := clone1.String(); got != want {
		t.Errorf("invalid clone, wanted %s, got %s", want, got)
	}

	want = "{size=5,value[0]=0000000000000000 0000000000000000 0000000000000000 0000000000000011}"
	if got := clone2.String(); got != want {
		t.Errorf("invalid clone, wanted %s, got %s", want, got)
	}
}

func TestStackGenerator_ClonesCanBeUsedToResetGenerator(t *testing.T) {
	generator := NewStackGenerator()
	generator.SetValue(0, NewU256(42))

	backup := generator.Clone()

	generator.SetSize(5)
	generator.SetValue(1, NewU256(16))

	want := "{size=5,value[0]=0000000000000000 0000000000000000 0000000000000000 000000000000002a,value[1]=0000000000000000 0000000000000000 0000000000000000 0000000000000010}"
	if got := generator.String(); got != want {
		t.Errorf("unexpected generator state, wanted %s, got %s", want, got)
	}

	generator.Restore(backup)
	want = "{value[0]=0000000000000000 0000000000000000 0000000000000000 000000000000002a}"
	if got := generator.String(); got != want {
		t.Errorf("unexpected generator state, wanted %s, got %s", want, got)
	}
}
