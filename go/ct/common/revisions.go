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
