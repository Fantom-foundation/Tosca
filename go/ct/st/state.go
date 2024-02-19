package st

import (
	"bytes"
	"encoding/json"
	"fmt"
	"regexp"
	"slices"
	"strings"

	. "github.com/Fantom-foundation/Tosca/go/ct/common"
	"github.com/Fantom-foundation/Tosca/go/vm"
)

// Upper bound for gas, this limit is required since evmc defines a signed type for gas.
// Limiting gas also solves issue 293 regarding out of memory failures,
// discussed here: https://github.com/Fantom-foundation/Tosca/issues/293
const MaxGas = vm.Gas(1 << 60)

// MaxDataSize is the maximum length of the call data vector generated for a test state. While
// the maximum size is not limited in a real-world setup, larger inputs are not expected to trigger
// additional issues in EVM implementations (with the exception of resource issues). Thus, this
// limit was chosen to avoid excessive overhead during the generation of states, their execution
// and their comparison.
const MaxDataSize = 1024

////////////////////////////////////////////////////////////

type StatusCode int

const (
	Running        StatusCode = iota // still running
	Stopped                          // stopped execution successfully
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
	case Reverted:
		return "reverted"
	case Failed:
		return "failed"
	default:
		return fmt.Sprintf("StatusCode(%d)", s)
	}
}

func (s StatusCode) MarshalJSON() ([]byte, error) {
	statusString := s.String()
	reg := regexp.MustCompile(`StatusCode\([0-9]+\)`)
	if reg.MatchString(statusString) {
		return nil, &json.UnsupportedValueError{}
	}
	return json.Marshal(statusString)
}

func (s *StatusCode) UnmarshalJSON(data []byte) error {
	var statusString string
	err := json.Unmarshal(data, &statusString)
	if err != nil {
		return err
	}
	var status StatusCode

	switch statusString {
	case "running":
		status = Running
	case "stopped":
		status = Stopped
	case "reverted":
		status = Reverted
	case "failed":
		status = Failed
	default:
		return &json.InvalidUnmarshalError{}
	}

	*s = status
	return nil
}

////////////////////////////////////////////////////////////

// State represents an EVM's execution state.
type State struct {
	Status             StatusCode
	Revision           Revision
	ReadOnly           bool
	Pc                 uint16
	Gas                vm.Gas
	GasRefund          vm.Gas
	Code               *Code
	Stack              *Stack
	Memory             *Memory
	Storage            *Storage
	Accounts           *Accounts
	Logs               *Logs
	CallContext        CallContext
	BlockContext       BlockContext
	CallData           []byte
	LastCallReturnData []byte
	ReturnData         []byte
}

// NewState creates a new State instance with the given code.
func NewState(code *Code) *State {
	return &State{
		Status:             Running,
		Revision:           R07_Istanbul,
		Code:               code,
		Stack:              NewStack(),
		Memory:             NewMemory(),
		Storage:            NewStorage(),
		Accounts:           NewAccounts(),
		Logs:               NewLogs(),
		CallData:           make([]byte, 0),
		LastCallReturnData: make([]byte, 0),
	}
}

func (s *State) Clone() *State {
	clone := NewState(s.Code.Clone())
	clone.Status = s.Status
	clone.Revision = s.Revision
	clone.ReadOnly = s.ReadOnly
	clone.Pc = s.Pc
	clone.Gas = s.Gas
	clone.GasRefund = s.GasRefund
	clone.Stack = s.Stack.Clone()
	clone.Memory = s.Memory.Clone()
	clone.Storage = s.Storage.Clone()
	clone.Accounts = s.Accounts.Clone()
	clone.Logs = s.Logs.Clone()
	clone.CallContext = s.CallContext
	clone.BlockContext = s.BlockContext
	clone.CallData = bytes.Clone(s.CallData)
	clone.LastCallReturnData = bytes.Clone(s.LastCallReturnData)
	clone.ReturnData = bytes.Clone(s.ReturnData)
	return clone
}

func (s *State) Eq(other *State) bool {
	// All failure states are considered equal.
	if s.Status == Failed && other.Status == Failed {
		return true
	}

	isHaltedState := func(s *State) bool {
		return s.Status != Running
	}
	pcIsEqual := s.Pc == other.Pc
	if isHaltedState(s) && isHaltedState(other) {
		// The program counter does not matter for halted states.
		pcIsEqual = true
	}

	equalReturnData := true
	if s.Status == Stopped || other.Status == Reverted {
		equalReturnData = bytes.Equal(s.ReturnData, other.ReturnData)
	}

	return equalReturnData &&
		s.Status == other.Status &&
		s.Revision == other.Revision &&
		s.ReadOnly == other.ReadOnly &&
		pcIsEqual &&
		s.Gas == other.Gas &&
		s.GasRefund == other.GasRefund &&
		s.Code.Eq(other.Code) &&
		s.Stack.Eq(other.Stack) &&
		s.Memory.Eq(other.Memory) &&
		s.Storage.Eq(other.Storage) &&
		s.Accounts.Eq(other.Accounts) &&
		s.Logs.Eq(other.Logs) &&
		s.CallContext == other.CallContext &&
		s.BlockContext == other.BlockContext &&
		slices.Equal(s.CallData, other.CallData) &&
		slices.Equal(s.LastCallReturnData, other.LastCallReturnData)
}

const dataCutoffLength = 20
const stackCutOffLength = 5

func (s *State) String() string {
	builder := strings.Builder{}
	builder.WriteString("{\n")
	builder.WriteString(fmt.Sprintf("\tStatus: %v\n", s.Status))
	builder.WriteString(fmt.Sprintf("\tRevision: %v\n", s.Revision))
	builder.WriteString(fmt.Sprintf("\tRead only mode: %t\n", s.ReadOnly))
	builder.WriteString(fmt.Sprintf("\tPc: %d (0x%04x)\n", s.Pc, s.Pc))
	if !s.Code.IsCode(int(s.Pc)) {
		builder.WriteString("\t    (points to data)\n")
	} else if s.Pc < uint16(len(s.Code.code)) {
		builder.WriteString(fmt.Sprintf("\t    (operation: %v)\n", OpCode(s.Code.code[s.Pc])))
	} else {
		builder.WriteString("\t    (out of bounds)\n")
	}
	builder.WriteString(fmt.Sprintf("\tGas: %d\n", s.Gas))
	builder.WriteString(fmt.Sprintf("\tGas refund: %d\n", s.GasRefund))
	if len(s.Code.code) > dataCutoffLength {
		builder.WriteString(fmt.Sprintf("\tCode: %x... (size: %d)\n", s.Code.code[:dataCutoffLength], len(s.Code.code)))
	} else {
		builder.WriteString(fmt.Sprintf("\tCode: %v\n", s.Code))
	}
	builder.WriteString(fmt.Sprintf("\tStack size: %d\n", s.Stack.Size()))
	for i := 0; i < s.Stack.Size() && i < stackCutOffLength; i++ {
		builder.WriteString(fmt.Sprintf("\t    %d: %v\n", i, s.Stack.Get(i)))
	}
	if s.Stack.Size() > stackCutOffLength {
		builder.WriteString("\t    ...\n")
	}
	builder.WriteString(fmt.Sprintf("\tMemory size: %d\n", s.Memory.Size()))
	builder.WriteString("\tStorage.Current:\n")
	for k, v := range s.Storage.Current {
		builder.WriteString(fmt.Sprintf("\t    [%v]=%v\n", k, v))
	}
	builder.WriteString("\tStorage.Original:\n")
	for k, v := range s.Storage.Original {
		builder.WriteString(fmt.Sprintf("\t    [%v]=%v\n", k, v))
	}
	builder.WriteString("\tStorage.Warm:\n")
	for k := range s.Storage.warm {
		builder.WriteString(fmt.Sprintf("\t    [%v]\n", k))
	}
	builder.WriteString("\tAccount.Balance:\n")
	for k, v := range s.Accounts.Balance {
		builder.WriteString(fmt.Sprintf("\t    [%v]=%v\n", k, v))
	}
	builder.WriteString("\tAccount.Code:\n")
	for k, v := range s.Accounts.Code {
		builder.WriteString(fmt.Sprintf("\t    [%v]=%v\n", k, v))
	}
	builder.WriteString("\tAddress.Warm:\n")
	for k := range s.Accounts.warm {
		builder.WriteString(fmt.Sprintf("\t    [%v]\n", k))
	}
	builder.WriteString("\tLogs:\n")
	for entryId, entry := range s.Logs.Entries {
		builder.WriteString(fmt.Sprintf("\t    entry %02d:\n", entryId))
		for topicId, topic := range entry.Topics {
			builder.WriteString(fmt.Sprintf("\t        topic %02d: %v\n", topicId, topic))
		}
		builder.WriteString(fmt.Sprintf("\t        data: %x\n", entry.Data))
	}
	builder.WriteString(fmt.Sprintf("\t%v", s.CallContext.String()))
	builder.WriteString(fmt.Sprintf("\t%v", s.BlockContext.String()))

	if len(s.CallData) > dataCutoffLength {
		builder.WriteString(fmt.Sprintf("\tCallData: %x... (size: %d)\n", s.CallData[:dataCutoffLength], len(s.CallData)))
	} else {
		builder.WriteString(fmt.Sprintf("\tCallData: %x\n", s.CallData))
	}

	if len(s.LastCallReturnData) > dataCutoffLength {
		builder.WriteString(fmt.Sprintf("\tLastCallReturnData: %x... (size: %d)\n", s.LastCallReturnData[:dataCutoffLength], len(s.LastCallReturnData)))
	} else {
		builder.WriteString(fmt.Sprintf("\tLastCallReturnData: %x\n", s.LastCallReturnData))
	}

	if len(s.ReturnData) > dataCutoffLength {
		builder.WriteString(fmt.Sprintf("\tReturnData: %x... (size: %d)\n", s.ReturnData[:dataCutoffLength], len(s.ReturnData)))
	} else {
		builder.WriteString(fmt.Sprintf("\tReturnData: %x\n", s.ReturnData))
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

	if s.ReadOnly != o.ReadOnly {
		res = append(res, fmt.Sprintf("Different read only mode: %t vs %t", s.ReadOnly, o.ReadOnly))
	}

	if s.Pc != o.Pc {
		res = append(res, fmt.Sprintf("Different pc: %v vs %v", s.Pc, o.Pc))
	}

	if s.Gas != o.Gas {
		res = append(res, fmt.Sprintf("Different gas: %v vs %v", s.Gas, o.Gas))
	}

	if s.GasRefund != o.GasRefund {
		res = append(res, fmt.Sprintf("Different gas refund: %v vs %v", s.GasRefund, o.GasRefund))
	}

	if !s.Code.Eq(o.Code) {
		res = append(res, s.Code.Diff(o.Code)...)
	}

	if !s.Stack.Eq(o.Stack) {
		res = append(res, s.Stack.Diff(o.Stack)...)
	}

	if !s.Memory.Eq(o.Memory) {
		res = append(res, s.Memory.Diff(o.Memory)...)
	}

	if !s.Storage.Eq(o.Storage) {
		res = append(res, s.Storage.Diff(o.Storage)...)
	}

	if !s.Accounts.Eq(o.Accounts) {
		res = append(res, s.Accounts.Diff(o.Accounts)...)
	}

	if !s.Logs.Eq(o.Logs) {
		res = append(res, s.Logs.Diff(o.Logs)...)
	}

	if s.CallContext != o.CallContext {
		res = append(res, s.CallContext.Diff(&o.CallContext)...)
	}

	if s.BlockContext != o.BlockContext {
		res = append(res, s.BlockContext.Diff(&o.BlockContext)...)
	}

	if !slices.Equal(s.CallData, o.CallData) {
		res = append(res, fmt.Sprintf("Different call data: %x vs %x", s.CallData, o.CallData))
	}

	if !slices.Equal(s.LastCallReturnData, o.LastCallReturnData) {
		res = append(res, fmt.Sprintf("Different last call return data: %x vs %x.", s.LastCallReturnData, o.LastCallReturnData))
	}

	if (s.Status == Stopped || o.Status == Reverted) && !slices.Equal(s.ReturnData, o.ReturnData) {
		res = append(res, fmt.Sprintf("Different return data: %x vs %x", s.ReturnData, o.ReturnData))
	}

	return res
}
