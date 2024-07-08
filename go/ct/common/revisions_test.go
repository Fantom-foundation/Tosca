// Copyright (c) 2024 Fantom Foundation
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at fantom.foundation/bsl11.
//
// Change Date: 2028-4-16
//
// On the date above, in accordance with the Business Source License, use of
// this software will be governed by the GNU Lesser General Public License v3.

package common

import (
	"math"
	"testing"

	"github.com/Fantom-foundation/Tosca/go/tosca"
)

func TestRevisions_RangeLength(t *testing.T) {
	tests := map[string]struct {
		revision    tosca.Revision
		rangeLength uint64
	}{
		"Istanbul":    {tosca.R07_Istanbul, 1000},
		"Berlin":      {tosca.R09_Berlin, 1000},
		"London":      {tosca.R10_London, 1000},
		"Paris":       {tosca.R11_Paris, 1000},
		"Shanghai":    {tosca.R12_Shanghai, 1000},
		"Cancun":      {tosca.R13_Cancun, 1000},
		"UnknownNext": {tosca.R99_UnknownNextRevision, math.MaxUint64},
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
	invalidRevision := tosca.R99_UnknownNextRevision + 1
	_, err := GetBlockRangeLengthFor(invalidRevision)
	if err == nil {
		t.Errorf("Error handling %v. %v", name, err)
	}

}

func TestRevisions_GetForkBlock(t *testing.T) {
	tests := map[string]struct {
		revision  tosca.Revision
		forkBlock uint64
	}{
		"Istanbul":    {tosca.R07_Istanbul, 0},
		"Berlin":      {tosca.R09_Berlin, 1000},
		"London":      {tosca.R10_London, 2000},
		"Paris":       {tosca.R11_Paris, 3000},
		"Shanghai":    {tosca.R12_Shanghai, 4000},
		"Cancun":      {tosca.R13_Cancun, 5000},
		"UnknownNext": {tosca.R99_UnknownNextRevision, 6000},
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
	_, err := GetForkBlock(tosca.R99_UnknownNextRevision + 1)
	if err == nil {
		t.Errorf("Unexpected error: %v", err)
	}
}

func TestRevision_GetRevisionForBlock(t *testing.T) {

	revisions := map[tosca.Revision]uint64{}

	for i := tosca.R07_Istanbul; i <= NewestSupportedRevision; i++ {
		revisionBlockNumber, _ := GetForkBlock(i)
		revisions[i] = revisionBlockNumber
	}

	for revision, revisionBlockNumber := range revisions {
		t.Run(revision.String(), func(t *testing.T) {
			got := GetRevisionForBlock(revisionBlockNumber)
			if revision != got {
				t.Errorf("Unexpected revision for block number: %v", got)
			}
		})
	}

}
