package st

import (
	"fmt"
	"strings"
)

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

const codeCutoffLength = 20

func (s *State) String() string {
	builder := strings.Builder{}
	builder.WriteString("{\n")
	builder.WriteString(fmt.Sprintf("\tStatus: %v\n", s.Status))
	builder.WriteString(fmt.Sprintf("\tRevision: %v\n", s.Revision))
	builder.WriteString(fmt.Sprintf("\tPc: %d (0x%04x)\n", s.Pc, s.Pc))
	if !s.Code.IsCode(int(s.Pc)) {
		builder.WriteString("\t    (points to data)\n")
	} else if s.Pc < uint16(len(s.Code.code)) {
		builder.WriteString(fmt.Sprintf("\t    (operation: %v)\n", OpCode(s.Code.code[s.Pc])))
	} else {
		builder.WriteString("\t    (out of bounds)\n")
	}
	builder.WriteString(fmt.Sprintf("\tGas: %d\n", s.Gas))
	if len(s.Code.code) > codeCutoffLength {
		builder.WriteString(fmt.Sprintf("\tCode: %x... (size: %d)\n", s.Code.code[:codeCutoffLength], len(s.Code.code)))
	} else {
		builder.WriteString(fmt.Sprintf("\tCode: %v\n", s.Code))
	}
	builder.WriteString("}")
	return builder.String()
}

func (s *State) Diff(o *State) []string {
	res := []string{}

	if s.Status != o.Status {
		res = append(res, fmt.Sprintf("Different status: %v vs %v", s.Status, o.Status))
	}

	if s.Revision != o.Revision {
		res = append(res, fmt.Sprintf("Different revision: %v vs %v", s.Revision, o.Revision))
	}

	if s.Pc != o.Pc {
		res = append(res, fmt.Sprintf("Different pc: %v vs %v", s.Pc, o.Pc))
	}

	if s.Gas != o.Gas {
		res = append(res, fmt.Sprintf("Different gas: %v vs %v", s.Gas, o.Gas))
	}

	if !s.Code.Eq(o.Code) {
		res = append(res, fmt.Sprintf("Different code: size %d vs %d", len(s.Code.code), len(o.Code.code)))
	}

	return res
}
