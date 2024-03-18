package common

import (
	"bytes"
	"testing"
)

func TestRevisions_RangeLength(t *testing.T) {
	tests := map[string]struct {
		revision    Revision
		rangeLength uint64
	}{
		"Istanbul":    {R07_Istanbul, 10},
		"Berlin":      {R09_Berlin, 10},
		"London":      {R10_London, 10},
		"UnknownNext": {R99_UnknownNextRevision, 0},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			got, err := GetBlockRangeLengthFor(test.revision)
			if err != nil {
				t.Errorf("Error getting block range length. %v", err)
			}
			if want := test.rangeLength; want != got {
				t.Errorf("Unexpected range length for %v, got %v", name, got)
			}
		})
	}
}

func TestRevisions_InvalidRevision(t *testing.T) {
	name := "unknown revision"
	invalidRevision := R99_UnknownNextRevision + 1
	_, err := GetBlockRangeLengthFor(invalidRevision)
	if err == nil {
		t.Errorf("Error handling %v. %v", name, err)
	}

}

func TestRevisions_GetForkBlock(t *testing.T) {
	tests := map[string]struct {
		revision  Revision
		forkBlock uint64
	}{
		"Istanbul":    {R07_Istanbul, 0},
		"Berlin":      {R09_Berlin, 10},
		"London":      {R10_London, 20},
		"UnknownNext": {R99_UnknownNextRevision, 30},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			got, err := GetForkBlock(test.revision)
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
			if test.forkBlock != got {
				t.Errorf("Unexpected revision fork: %v", got)
			}
		})
	}
}

func TestRevisions_GetForkBlockInvalid(t *testing.T) {
	_, err := GetForkBlock(R99_UnknownNextRevision + 1)
	if err == nil {
		t.Errorf("Unexpected error: %v", err)
	}
}

func TestRevisions_Marshal(t *testing.T) {
	tests := map[Revision]string{
		R07_Istanbul:            "\"Istanbul\"",
		R09_Berlin:              "\"Berlin\"",
		R10_London:              "\"London\"",
		R99_UnknownNextRevision: "\"UnknownNextRevision\"",
	}

	for input, expected := range tests {
		marshaled, err := input.MarshalJSON()
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if !bytes.Equal(marshaled, []byte(expected)) {
			t.Errorf("Unexpected marshaled revision, wanted: %v vs got: %v", expected, marshaled)
		}
	}
}

func TestRevisions_MarshalError(t *testing.T) {
	revisions := []Revision{Revision(42), Revision(100)}
	for _, rev := range revisions {
		marshaled, err := rev.MarshalJSON()
		if err == nil {
			t.Errorf("Expected error but got: %v", marshaled)
		}
	}
}

func TestRevisions_Unmarshal(t *testing.T) {
	tests := map[string]Revision{
		"\"Istanbul\"":            R07_Istanbul,
		"\"Berlin\"":              R09_Berlin,
		"\"London\"":              R10_London,
		"\"UnknownNextRevision\"": R99_UnknownNextRevision,
	}

	for input, expected := range tests {
		var rev Revision
		err := rev.UnmarshalJSON([]byte(input))
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if rev != expected {
			t.Errorf("Unexpected unmarshaled revision, wanted: %v vs got: %v", expected, rev)
		}
	}
}

func TestRevisions_UnmarshalError(t *testing.T) {
	inputs := []string{"Error", "Revision(42)", "Istanbul"}
	for _, input := range inputs {
		var rev Revision
		err := rev.UnmarshalJSON([]byte(input))
		if err == nil {
			t.Errorf("Expected error but got: %v", rev)
		}
	}
}
