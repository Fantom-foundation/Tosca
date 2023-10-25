package st

import "fmt"

////////////////////////////////////////////////////////////

type StatusCode int

const (
	Running        StatusCode = iota // still running
	Stopped                          // stopped execution successfully
	Returned                         // finished successfully
	Reverted                         // finished with revert signal
	Failed                           // failed (for any reason)
	NumStatusCodes                   // not an actual status
)

func (s StatusCode) String() string {
	switch s {
	case Running:
		return "running"
	case Stopped:
		return "stopped"
	case Returned:
		return "returned"
	case Reverted:
		return "reverted"
	case Failed:
		return "failed"
	default:
		return fmt.Sprintf("StatusCode(%d)", s)
	}
}

////////////////////////////////////////////////////////////

type Revision int

const (
	Istanbul Revision = iota
	Berlin
	London
	NumRevisions // not an actual revision
)

func (r Revision) String() string {
	switch r {
	case Istanbul:
		return "Istanbul"
	case Berlin:
		return "Berlin"
	case London:
		return "London"
	default:
		return fmt.Sprintf("Revision(%d)", r)
	}
}

////////////////////////////////////////////////////////////

// State represents an EVM's execution state.
type State struct {
	Status   StatusCode
	Revision Revision
	Pc       uint16
	Gas      uint64
	Code     *Code
}

// NewState creates a new State instance with the given code.
func NewState(code *Code) *State {
	return &State{
		Status:   Running,
		Revision: Istanbul,
		Code:     code,
	}
}

func (s *State) Eq(other *State) bool {
	// All failure states are considered equal.
	if s.Status == Failed && other.Status == Failed {
		return true
	}
	return s.Status == other.Status &&
		s.Revision == other.Revision &&
		s.Pc == other.Pc &&
		s.Gas == other.Gas &&
		s.Code.Eq(other.Code)
}
