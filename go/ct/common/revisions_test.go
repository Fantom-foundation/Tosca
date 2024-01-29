package common

import (
	"strings"
	"testing"
)

func TestRevisions_RangeLength(t *testing.T) {
	tests := map[string]struct {
		revision    Revision
		rangeLength uint64
	}{
		"Istanbul": {R07_Istanbul, 10},
		"Berlin":   {R09_Berlin, 10},
		"London":   {R10_London, 10},
		"Future":   {R99_UnknownNextRevision, 0},
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
	want := uint64(0)
	got, err := GetBlockRangeLengthFor(invalidRevision)
	if !strings.Contains(err.Error(), name) {
		t.Errorf("Error handling %v. %v", name, err)
	}
	if want != got {
		t.Errorf("Unexpected range length for %v, got %v", name, got)
	}

}

func TestRevisions_GetForkBlock(t *testing.T) {
	tests := map[string]struct {
		revision  Revision
		forkBlock uint64
	}{
		"Istanbul":   {R07_Istanbul, 0},
		"Berlin":     {R09_Berlin, 10},
		"London":     {R10_London, 20},
		"UknownNext": {R99_UnknownNextRevision, 30},
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
	name := "unknown revision"
	got, err := GetForkBlock(R99_UnknownNextRevision + 1)
	if !strings.Contains(err.Error(), name) {
		t.Errorf("Unexpected error: %v", err)
	}
	if got != 0 {
		t.Errorf("Unexpected revision fork: %v", got)
	}
}
