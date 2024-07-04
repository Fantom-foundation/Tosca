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
	"fmt"
	"math"

	"github.com/Fantom-foundation/Tosca/go/tosca"
)

// Newest tosca.Revision currently supported by the CT specification
const NewestSupportedRevision = tosca.R13_Cancun
const NewestFullySupportedRevision = tosca.R12_Shanghai

const R99_UnknownNextRevision = NewestSupportedRevision + 1
const MinRevision = tosca.R07_Istanbul
const MaxRevision = R99_UnknownNextRevision

// GetForkBlock returns the first block a given revision is considered to be
// enabled for when running CT state evaluations. It is intended to provide input
// for test state generators to produce consistent block numbers and code revisions,
// as well as for adapters between the CT framework and EVM interpreters to support
// the state conversion.
func GetForkBlock(revision tosca.Revision) (uint64, error) {
	switch revision {
	case tosca.R07_Istanbul:
		return 0, nil
	case tosca.R09_Berlin:
		return 1000, nil
	case tosca.R10_London:
		return 2000, nil
	case tosca.R11_Paris:
		return 3000, nil
	case tosca.R12_Shanghai:
		return 4000, nil
	case tosca.R13_Cancun:
		return 5000, nil
	case R99_UnknownNextRevision:
		return 6000, nil
	}
	// TODO: remove this error
	return 0, fmt.Errorf("unknown revision: %v", revision)
}

// GetForkTime returns the revision fork timestamp.
// It is intended to provide input for test state generators to produce consistent
// fork timestamps and code revisions, as well as for adapters between the CT framework
// and EVM interpreters to support the state conversion.
// This function will never fail, as it may be required to generate a timestamp for an future revision.
func GetForkTime(revision tosca.Revision) uint64 {
	switch revision {
	case tosca.R07_Istanbul:
		return 0
	case tosca.R09_Berlin:
		return 1000
	case tosca.R10_London:
		return 2000
	case tosca.R11_Paris:
		return 3000
	case tosca.R12_Shanghai:
		return 4000
	case tosca.R13_Cancun:
		return 5000
	default:
		return 6000
	}
}

// GetRevisionForBlock returns the revision that is considered to be enabled for a given block number.
func GetRevisionForBlock(block uint64) tosca.Revision {
	for rev := MinRevision; rev <= MaxRevision; rev++ {
		forkBlock, _ := GetForkBlock(rev)
		if block < forkBlock {
			return rev - 1
		}
	}
	return R99_UnknownNextRevision
}

// GetBlockRangeLengthFor returns the number of block numbers between the given revision and the following
// in case of an Unknown revision, math.MaxUint64 is returned.
func GetBlockRangeLengthFor(revision tosca.Revision) (uint64, error) {
	revisionNumber, err := GetForkBlock(revision)
	if err != nil {
		return 0, err
	}
	revisionNumberRange := uint64(math.MaxUint64)

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
