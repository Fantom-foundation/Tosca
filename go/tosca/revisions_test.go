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
	"testing"
)

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
