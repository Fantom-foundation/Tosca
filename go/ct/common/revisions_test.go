package common

import (
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
