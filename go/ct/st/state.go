// Copyright (c) 2024 Fantom Foundation
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at fantom.foundation/bsl11.
//
// Change Date: 2028-4-16
//
// On the date above, in accordance with the Business Source License, use of
// this software will be governed by the GNU Lesser General Public License v3.

package st

import (
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

type SelfDestructEntry struct {
	account     vm.Address
	beneficiary vm.Address
}

func NewSelfDestructEntry(account vm.Address, beneficiary vm.Address) SelfDestructEntry {
	return SelfDestructEntry{account, beneficiary}
}

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

// Releaser interface for object recycling
type Releaser interface {
	Release()
}

////////////////////////////////////////////////////////////

// State represents an EVM's execution state.
type State struct {
	Status                StatusCode
	Revision              Revision
	ReadOnly              bool
	Pc                    uint16
	Gas                   vm.Gas
	GasRefund             vm.Gas
	Code                  *Code
	Stack                 *Stack
	Memory                *Memory
	Storage               *Storage
	Accounts              *Accounts
	Logs                  *Logs
	CallContext           CallContext
	CallJournal           *CallJournal
	BlockContext          BlockContext
	CallData              Bytes
	LastCallReturnData    Bytes
	ReturnData            Bytes
	HasSelfDestructed     bool
	SelfDestructedJournal []SelfDestructEntry
	RecentBlockHashes     [256]vm.Hash
	TransactionContext    TransactionContext
}

// NewState creates a new State instance with the given code.
func NewState(code *Code) *State {
	return &State{
		Status:                Running,
		Revision:              R07_Istanbul,
		Code:                  code,
		Stack:                 &Stack{},
		Memory:                NewMemory(),
		Storage:               &Storage{},
		Accounts:              NewAccounts(),
		Logs:                  NewLogs(),
		CallJournal:           NewCallJournal(),
		CallData:              Bytes{},
		LastCallReturnData:    Bytes{},
		SelfDestructedJournal: []SelfDestructEntry{},
	}
}

// Release releases all member objects to be recycled.
func (s *State) Release() {
	s.Stack.Release()
	s.Stack = nil
}

func (s *State) Clone() *State {
	clone := &State{}
	clone.Code = s.Code.Clone()
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
	clone.CallJournal = s.CallJournal.Clone()
	clone.BlockContext = s.BlockContext
	clone.CallData = s.CallData
	clone.LastCallReturnData = s.LastCallReturnData
	clone.ReturnData = s.ReturnData
	clone.HasSelfDestructed = s.HasSelfDestructed
	clone.SelfDestructedJournal = slices.Clone(s.SelfDestructedJournal)
	clone.RecentBlockHashes = s.RecentBlockHashes
	clone.TransactionContext = s.TransactionContext
	return clone
}

func (s *State) Eq(other *State) bool {
	if s == other {
		return true
	}

	if s.Status != other.Status {
		return false
	}

	// All failure states are considered equal.
	if s.Status == Failed && other.Status == Failed {
		return true
	}

	// Check public observable state properties first.
	equivalent := true &&
		s.Code.Eq(other.Code) &&
		s.Revision == other.Revision &&
		s.ReadOnly == other.ReadOnly &&
		s.Gas == other.Gas &&
		s.GasRefund == other.GasRefund &&
		s.CallContext == other.CallContext &&
		s.CallJournal.Equal(other.CallJournal) &&
		s.BlockContext == other.BlockContext &&
		s.CallData == other.CallData &&
		s.Storage.Eq(other.Storage) &&
		s.Accounts.Eq(other.Accounts) &&
		s.Logs.Eq(other.Logs) &&
		s.HasSelfDestructed == other.HasSelfDestructed &&
		slices.Equal(s.SelfDestructedJournal, other.SelfDestructedJournal) &&
		s.RecentBlockHashes == other.RecentBlockHashes &&
		s.TransactionContext == other.TransactionContext

	// For terminal states, internal state can be ignored, but the result is important.
	if s.Status != Running {
		return equivalent &&
			s.ReturnData == other.ReturnData
	}

	// If the state is running, internal state is relevant, but the result can be ignored.
	return equivalent &&
		s.Pc == other.Pc &&
		s.Stack.Eq(other.Stack) &&
		s.Memory.Eq(other.Memory) &&
		s.LastCallReturnData == other.LastCallReturnData
}

const dataCutoffLength = 20
const stackCutOffLength = 5

func (s *State) String() string {
	builder := strings.Builder{}

	write := func(pattern string, args ...any) {
		builder.WriteString(fmt.Sprintf(pattern, args...))
	}

	write("{\n")
	write("\tStatus: %v\n", s.Status)
	write("\tRevision: %v\n", s.Revision)
	write("\tRead only mode: %t\n", s.ReadOnly)
	write("\tPc: %d (0x%04x)\n", s.Pc, s.Pc)
	if !s.Code.IsCode(int(s.Pc)) {
		write("\t    (points to data)\n")
	} else if s.Pc < uint16(len(s.Code.code)) {
		write("\t    (operation: %v)\n", OpCode(s.Code.code[s.Pc]))
	} else {
		write("\t    (out of bounds)\n")
	}
	write("\tGas: %d\n", s.Gas)
	write("\tGas refund: %d\n", s.GasRefund)
	if len(s.Code.code) > dataCutoffLength {
		write("\tCode: %x... (size: %d)\n", s.Code.code[:dataCutoffLength], len(s.Code.code))
	} else {
		write("\tCode: %v\n", s.Code)
	}
	write("\tStack size: %d\n", s.Stack.Size())
	for i := 0; i < s.Stack.Size() && i < stackCutOffLength; i++ {
		write("\t    %d: %v\n", i, s.Stack.Get(i))
	}
	if s.Stack.Size() > stackCutOffLength {
		write("\t    ...\n")
	}
	write("\tMemory size: %d\n", s.Memory.Size())
	write("\tStorage.Current:\n")
	for k, v := range s.Storage.current {
		write("\t    [%v]=%v\n", k, v)
	}
	write("\tStorage.Original:\n")
	for k, v := range s.Storage.original {
		write("\t    [%v]=%v\n", k, v)
	}
	write("\tStorage.Warm:\n")
	for k := range s.Storage.warm {
		write("\t    [%v]\n", k)
	}
	write(s.Accounts.String())
	write("\tLogs:\n")
	for entryId, entry := range s.Logs.Entries {
		write("\t    entry %02d:\n", entryId)
		for topicId, topic := range entry.Topics {
			write("\t        topic %02d: %v\n", topicId, topic)
		}
		write("\t        data: %x\n", entry.Data)
	}
	write("\t%v", s.CallContext.String())
	write("\t%v", s.BlockContext.String())
	write("\t%v", s.TransactionContext.String())

	write("\tPast Calls:\n")
	for i, cur := range s.CallJournal.Past {
		write("\t\tCall %d:\n", i)
		write("\t\t\tKind:      %v\n", cur.Kind)
		write("\t\t\tRecipient: %v\n", cur.Recipient)
		write("\t\t\tSender:    %v\n", cur.Sender)
		write("\t\t\tInput:     %v\n", cur.Input)
		write("\t\t\tValue:     %v\n", cur.Value)
		write("\t\t\tGas:       %v\n", cur.Gas)
	}

	write("\tFuture Calls:\n")
	for i, cur := range s.CallJournal.Future {
		write("\t\tCall %d:\n", i)
		write("\t\t\tSuccess:  %t\n", cur.Success)
		write("\t\t\tOutput:   %v\n", cur.Output)
		write("\t\t\tGasCosts: %v\n", cur.GasCosts)
	}

	for entryId, entry := range s.Logs.Entries {
		write("\t    entry %02d:\n", entryId)
		for topicId, topic := range entry.Topics {
			write("\t        topic %02d: %v\n", topicId, topic)
		}
		write("\t        data: %x\n", entry.Data)
	}

	if s.CallData.Length() > dataCutoffLength {
		write("\tCallData: %x... (size: %d)\n", s.CallData.Get(0, dataCutoffLength), s.CallData.Length())
	} else {
		write("\tCallData: %x\n", s.CallData)
	}

	if s.LastCallReturnData.Length() > dataCutoffLength {
		write("\tLastCallReturnData: %x... (size: %d)\n", s.LastCallReturnData.Get(0, dataCutoffLength), s.LastCallReturnData.Length())
	} else {
		write("\tLastCallReturnData: %x\n", s.LastCallReturnData)
	}

	if s.ReturnData.Length() > dataCutoffLength {
		write("\tReturnData: %x... (size: %d)\n", s.ReturnData.Get(0, dataCutoffLength), s.ReturnData.Length())
	} else {
		write("\tReturnData: %x\n", s.ReturnData)
	}

	write("\tHasSelfDestructed: %v\n", s.HasSelfDestructed)
	write("\tSelfDestructedJournal: %v\n", s.SelfDestructedJournal)

	// only print if next instruction is blockhash and the top of the stack is a valid uint64
	if s.Code != nil && s.Code.Length() > int(s.Pc) && s.Stack != nil && s.Stack.Size() > 0 {
		offset := s.Stack.stack[s.Stack.Size()-1]
		if s.Code.IsCode(int(s.Pc)) && OpCode(s.Code.code[s.Pc]) == BLOCKHASH &&
			offset.IsUint64() && offset.Uint64() < 256 {
			write("\tHash of block %d: %#x\n", s.BlockContext.BlockNumber-offset.Uint64(), s.RecentBlockHashes[offset.Uint64()])
		}
	}

	write("}")
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
		res = append(res, fmt.Sprintf("Different gas: %v vs %v (diff: %d)", s.Gas, o.Gas, o.Gas-s.Gas))
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

	if !s.CallJournal.Equal(o.CallJournal) {
		res = append(res, s.CallJournal.Diff(o.CallJournal)...)
	}

	if s.BlockContext != o.BlockContext {
		res = append(res, s.BlockContext.Diff(&o.BlockContext)...)
	}

	if s.TransactionContext != o.TransactionContext {
		res = append(res, s.TransactionContext.Diff(&o.TransactionContext)...)
	}

	if s.CallData != o.CallData {
		res = append(res, fmt.Sprintf("Different call data: %x vs %x", s.CallData, o.CallData))
	}

	if s.LastCallReturnData != o.LastCallReturnData {
		res = append(res, fmt.Sprintf("Different last call return data: %x vs %x.", s.LastCallReturnData, o.LastCallReturnData))
	}

	if (s.Status == Stopped || s.Status == Reverted) && s.ReturnData != o.ReturnData {
		res = append(res, fmt.Sprintf("Different return data: %x vs %x", s.ReturnData, o.ReturnData))
	}

	if s.HasSelfDestructed != o.HasSelfDestructed {
		res = append(res, fmt.Sprintf("Different has-self-destructed: %v vs %v ", s.HasSelfDestructed, o.HasSelfDestructed))
	}

	if !slices.Equal(s.SelfDestructedJournal, o.SelfDestructedJournal) {
		if len(s.SelfDestructedJournal) != len(o.SelfDestructedJournal) {
			res = append(res, fmt.Sprintf("Different has-self-destructed journal length: %v vs %v",
				len(s.SelfDestructedJournal), len(o.SelfDestructedJournal)))
		} else {
			for index, entry1 := range s.SelfDestructedJournal {
				entry2 := o.SelfDestructedJournal[index]
				if entry1 != entry2 {
					res = append(res, fmt.Sprintf("Different has-self-destructed journal entry:\n\t(%v, %v)\n\tvs\n\t(%v, %v)",
						entry1.account, entry1.beneficiary, entry2.account, entry2.beneficiary))
				}
			}
		}
	}

	for i, want := range s.RecentBlockHashes {
		if want != o.RecentBlockHashes[i] {
			res = append(res, fmt.Sprintf("Different block number hash at index %d: %x vs %x", i, want, o.RecentBlockHashes[i]))
		}
	}

	return res
}
