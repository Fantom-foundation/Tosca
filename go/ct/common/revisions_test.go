//
// Copyright (c) 2024 Fantom Foundation
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at fantom.foundation/bsl11.
//
// Change Date: 2028-4-16
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by the GNU Lesser General Public Licence v3
//

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
		"Istanbul":    {R07_Istanbul, 1000},
		"Berlin":      {R09_Berlin, 1000},
		"London":      {R10_London, 1000},
		"Paris":       {R11_Paris, 1000},
		"Shanghai":    {R12_Shanghai, 1000},
		"Cancun":      {R13_Cancun, 1000},
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
		"Berlin":      {R09_Berlin, 1000},
		"London":      {R10_London, 2000},
		"Paris":       {R11_Paris, 3000},
		"Shanghai":    {R12_Shanghai, 4000},
		"Cancun":      {R13_Cancun, 5000},
		"UnknownNext": {R99_UnknownNextRevision, 6000},
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
		R11_Paris:               "\"Paris\"",
		R12_Shanghai:            "\"Shanghai\"",
		R13_Cancun:              "\"Cancun\"",
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
		"\"Paris\"":               R11_Paris,
		"\"Shanghai\"":            R12_Shanghai,
		"\"Cancun\"":              R13_Cancun,
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

func TestRevisions_GetRevisionForBlock(t *testing.T) {
	for i := uint64(0); i < 10000; i++ {
		rev := GetRevisionForBlock(i)
		forkBlock, _ := GetForkBlock(rev)
		if i < forkBlock {
			t.Fatalf("wrong revision for block %v, got %v", i, rev)
		}
		if rev != R99_UnknownNextRevision {
			if length, _ := GetBlockRangeLengthFor(rev); i >= forkBlock+length {
				t.Fatalf("wrong revision for block %v, got %v", i, rev)
			}
		}
	}
}
