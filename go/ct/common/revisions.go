package common

import "fmt"

type Revision int

const (
	R07_Istanbul Revision = iota
	R09_Berlin
	R10_London
	R99_UnknownNextRevision
)

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

func GetForkBlock(revision Revision) (int64, error) {
	switch revision {
	case R07_Istanbul:
		return 0, nil
	case R09_Berlin:
		return 10, nil
	case R10_London:
		return 20, nil
	}
	return -1, fmt.Errorf("unknown revision: %v", revision)
}
