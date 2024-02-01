package gen

import (
	"testing"

	"pgregory.net/rand"
)

func TestReturnDataGenerator_SetSize(t *testing.T) {
	sizes := []int{0, 1, 2, 50}
	rnd := rand.New(0)

	for _, size := range sizes {
		gen := NewReturnDateGenerator()
		gen.SetSize(size)
		returnData, err := gen.Generate(rnd)
		if err != nil {
			t.Errorf("unexpected error generating return data: %v", err)
		}
		if want, got := size, len(returnData); want != got {
			t.Errorf("unexpected return data size, wanted: %v, got %v", want, got)
		}
	}
}

func TestReturnDataGenerator_SetSizeInvalid(t *testing.T) {
	size := -1
	rnd := rand.New(0)
	gen := NewReturnDateGenerator()
	gen.SetSize(size)
	_, err := gen.Generate(rnd)
	if err == nil {
		t.Error("expected error for negative size")
	}
}

func TestReturnDataGenerator_Generate(t *testing.T) {
	rnd := rand.New(0)
	gen := NewReturnDateGenerator()
	returnData, err := gen.Generate(rnd)
	if err != nil {
		t.Errorf("unexpected error generating return data: %v", err)
	}
	if len(returnData) > 50 {
		t.Errorf("unexpected return data size ourside of default range, size %v.", len(returnData))
	}
}

func TestReturnDataGenerator_String(t *testing.T) {
	sizes := []int{0, 1, 2, 50}
	gen := NewReturnDateGenerator()
	for _, size := range sizes {
		gen.SetSize(size)
	}
	if want, got := "{sizes: 0, 1, 2, 50}", gen.String(); want != got {
		t.Errorf("unexpected string, wanted: %v, got: %v", want, got)
	}
}

func TestReturnDataGenerator_Clone(t *testing.T) {
	gen1 := NewReturnDateGenerator()
	gen1.SetSize(1)
	gen2 := gen1.Clone()
	gen2.SetSize(2)

	want := "{sizes: 1}"
	if got := gen1.String(); want != got {
		t.Errorf("clones are not independent.")
	}
}
