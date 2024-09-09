// Copyright (c) 2024 Fantom Foundation
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at fantom.foundation/bsl11.
//
// Change Date: 2028-4-16
//
// On the date above, in accordance with the Business Source License, use of
// this software will be governed by the GNU Lesser General Public License v3.

package tosca

import (
	"bytes"
	"slices"
	"strings"
	"testing"
)

func TestRevisions_Marshal(t *testing.T) {
	tests := map[Revision]string{
		R07_Istanbul: "\"Istanbul\"",
		R09_Berlin:   "\"Berlin\"",
		R10_London:   "\"London\"",
		R11_Paris:    "\"Paris\"",
		R12_Shanghai: "\"Shanghai\"",
		R13_Cancun:   "\"Cancun\"",
		Revision(42): "\"Revision(42)\"",
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

func TestRevisions_Unmarshal(t *testing.T) {
	tests := map[string]Revision{
		"\"Istanbul\"":     R07_Istanbul,
		"\"Berlin\"":       R09_Berlin,
		"\"London\"":       R10_London,
		"\"Paris\"":        R11_Paris,
		"\"Shanghai\"":     R12_Shanghai,
		"\"Cancun\"":       R13_Cancun,
		"\"Revision(42)\"": Revision(42),
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
	inputs := []string{"Error", "Istanbul"}
	for _, input := range inputs {
		var rev Revision
		err := rev.UnmarshalJSON([]byte(input))
		if err == nil {
			t.Errorf("Expected error but got: %v", rev)
		}
	}
}

func TestAllKnownRevisions(t *testing.T) {
	existing := []Revision{}
	for r := Revision(0); ; r++ {
		if strings.HasPrefix(r.String(), "Revision") {
			break
		}
		existing = append(existing, r)
	}
	all := GetAllKnownRevisions()
	slices.Sort(existing)
	slices.Sort(all)
	if !slices.Equal(existing, all) {
		t.Errorf("Unexpected revisions, wanted: %v vs got: %v", existing, all)
	}
}
