package common

import (
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
			if want, got := test.rangeLength, GetBlockRangeLengthFor(test.revision); want != got {
				t.Errorf("Unexpected range length for %v, got %v", name, got)
			}
		})
	}
}
