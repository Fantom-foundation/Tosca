package common

import (
	"fmt"
)

type Revision int

const (
	R07_Istanbul Revision = iota
	R09_Berlin
	R10_London
	R99_UnknownNextRevision
)

var validRevisions = map[Revision]bool{
	R07_Istanbul:            true,
	R09_Berlin:              true,
	R10_London:              true,
	R99_UnknownNextRevision: true,
}

func (r Revision) String() string {
	switch r {
	case R07_Istanbul:
		return "Istanbul"
	case R09_Berlin:
		return "Berlin"
	case R10_London:
		return "London"
	case R99_UnknownNextRevision:
		return "UnknownNextRevision"
	default:
		return fmt.Sprintf("Revision(%d)", r)
	}
}

func IsValidRevision(revision Revision) bool {
	return validRevisions[revision]
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
		return 10, nil
	case R10_London:
		return 20, nil
	case R99_UnknownNextRevision:
		return 30, nil
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
	// if it's not, we want to limit this range to the firt block number of next revision.
	if revision < R99_UnknownNextRevision {
		nextRevisionNumber, err := GetForkBlock(revision + 1)
		if err != nil {
			return 0, err
		}
		// since we know both numbers are positive, and nextRevisionNumber is bigger,
		// we can safely converet them to uint64
		revisionNumberRange = nextRevisionNumber - revisionNumber
	}
	return revisionNumberRange, nil
}
