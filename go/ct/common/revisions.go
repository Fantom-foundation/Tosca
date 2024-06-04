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
	"encoding/json"
	"fmt"
	"regexp"
)

type Revision int

const (
	R07_Istanbul Revision = iota
	R09_Berlin
	R10_London
	R11_Paris
	R12_Shanghai
	R13_Cancun
	R99_UnknownNextRevision
)

// Newest Revision currently supported by the CT specification
const NewestSupportedRevision = R11_Paris

const MinRevision = R07_Istanbul
const MaxRevision = R99_UnknownNextRevision

func (r Revision) String() string {
	switch r {
	case R07_Istanbul:
		return "Istanbul"
	case R09_Berlin:
		return "Berlin"
	case R10_London:
		return "London"
	case R11_Paris:
		return "Paris"
	case R12_Shanghai:
		return "Shanghai"
	case R13_Cancun:
		return "Cancun"
	case R99_UnknownNextRevision:
		return "UnknownNextRevision"
	default:
		return fmt.Sprintf("Revision(%d)", r)
	}
}

func (r Revision) MarshalJSON() ([]byte, error) {
	revString := r.String()
	reg := regexp.MustCompile(`Revision\([0-9]+\)`)
	if reg.MatchString(revString) {
		return nil, &json.UnsupportedValueError{}
	}
	return json.Marshal(revString)
}

func (r *Revision) UnmarshalJSON(data []byte) error {
	var s string
	err := json.Unmarshal(data, &s)
	if err != nil {
		return err
	}
	var revision Revision

	switch s {
	case "Istanbul":
		revision = R07_Istanbul
	case "Berlin":
		revision = R09_Berlin
	case "London":
		revision = R10_London
	case "Paris":
		revision = R11_Paris
	case "Shanghai":
		revision = R12_Shanghai
	case "Cancun":
		revision = R13_Cancun
	case "UnknownNextRevision":
		revision = R99_UnknownNextRevision
	default:
		return &json.InvalidUnmarshalError{}
	}

	*r = revision
	return nil
}

// GetForkBlock returns the first block a given revision is considered to be
// enabled for when running CT state evaluations. It is intended to provide input
// for test state generators to produce consistent block numbers and code revisions,
// as well as for adapters between the CT framework and EVM interpreters to support
// the state conversion.
func GetForkBlock(revision Revision) (uint64, error) {
	switch revision {
	case R07_Istanbul:
		return 0, nil
	case R09_Berlin:
		return 1000, nil
	case R10_London:
		return 2000, nil
	case R11_Paris:
		return 3000, nil
	case R12_Shanghai:
		return 4000, nil
	case R13_Cancun:
		return 5000, nil
	case R99_UnknownNextRevision:
		return 6000, nil
	}
	return 0, fmt.Errorf("unknown revision: %v", revision)
}

func GetForkTimestamp(revision Revision) (uint64, error) {
	switch revision {
	case R07_Istanbul:
	case R09_Berlin:
	case R10_London:
	case R11_Paris:
		return 0, fmt.Errorf("revision: %v is too low for timestamps", revision)
	case R12_Shanghai:
		return 4000, nil
	case R13_Cancun:
		return 5000, nil
	case R99_UnknownNextRevision:
		return 6000, nil
	}
	return 0, fmt.Errorf("unknown revision: %v", revision)
}

// GetBlockRangeLengthFor returns the number of block numbers between the given revision and the following
// in case of an Unknown revision, 0 is returned.
func GetBlockRangeLengthFor(revision Revision) (uint64, error) {
	revisionNumber, err := GetForkBlock(revision)
	if err != nil {
		return 0, err
	}
	revisionNumberRange := uint64(0)

	// if it's the last supported revision, the blockNumber range has no limit.
	// if it's not, we want to limit this range to the first block number of next revision.
	if revision < R99_UnknownNextRevision {
		nextRevisionNumber, err := GetForkBlock(revision + 1)
		if err != nil {
			return 0, err
		}
		// since we know both numbers are positive, and nextRevisionNumber is bigger,
		// we can safely convert them to uint64
		revisionNumberRange = nextRevisionNumber - revisionNumber
	}
	return revisionNumberRange, nil
}
