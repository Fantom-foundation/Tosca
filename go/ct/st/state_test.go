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
	"bytes"
	"fmt"
	"regexp"
	"slices"
	"strings"
	"testing"

	. "github.com/Fantom-foundation/Tosca/go/ct/common"
	"github.com/Fantom-foundation/Tosca/go/tosca"
	"github.com/Fantom-foundation/Tosca/go/tosca/vm"
)

////////////////////////////////////////////////////////////
// Helper functions

func getNewFilledState() *State {
	s := NewState(NewCode([]byte{byte(vm.PUSH2), 7, 4, byte(vm.ADD), byte(vm.STOP)}))
	s.Status = Running
	s.Revision = tosca.R10_London
	s.ReadOnly = true
	s.Pc = 3
	s.Gas = 42
	s.GasRefund = 63
	s.Stack.Push(NewU256(42))
	s.Memory.Write([]byte{1, 2, 3}, 31)
	s.Storage = NewStorageBuilder().
		SetCurrent(NewU256(42), NewU256(7)).
		SetOriginal(NewU256(77), NewU256(4)).
		SetWarm(NewU256(9), true).
		Build()
	s.TransientStorage = &TransientStorage{}
	s.Accounts = NewAccountsBuilder().
		SetBalance(tosca.Address{0x01}, NewU256(42)).
		SetCode(tosca.Address{0x01}, NewBytes([]byte{byte(vm.PUSH1), byte(6)})).
		Build()
	s.Accounts.MarkWarm(tosca.Address{0x02})
	s.Logs.AddLog([]byte{4, 5, 6}, NewU256(21), NewU256(22))
	s.CallContext = CallContext{AccountAddress: tosca.Address{0x01}}
	s.BlockContext = BlockContext{BlockNumber: 1}
	s.TransactionContext = &TransactionContext{BlobHashes: []tosca.Hash{{4, 3, 2, 1}}}
	s.CallData = NewBytes([]byte{1})
	s.LastCallReturnData = NewBytes([]byte{1})
	s.HasSelfDestructed = true
	s.SelfDestructedJournal = []SelfDestructEntry{{tosca.Address{1}, tosca.Address{2}}}
	s.RecentBlockHashes = NewImmutableHashArray(tosca.Hash{0x01})
	return s
}

type testStruct struct {
	change func(*State)
	want   string
}

func getTestChanges() map[string]testStruct {
	tests := map[string]testStruct{
		"status": {func(state *State) {
			state.Status = Stopped
		},
			"Different status",
		},
		"revision": {func(state *State) {
			state.Revision = tosca.R07_Istanbul
		},
			"Different revision",
		},
		"read-only": {func(state *State) {
			state.ReadOnly = false
		},
			"Different read only mode",
		},
		"pc": {func(state *State) {
			state.Pc = 1
		},
			"Different pc",
		},
		"gas": {func(state *State) {
			state.Gas = 2
		},
			"Different gas",
		},
		"gas_refund": {func(state *State) {
			state.GasRefund = 15
		},
			"Different gas refund",
		},
		"code": {func(state *State) {
			state.Code = NewCode([]byte{byte(vm.ADD)})
		},
			"Different code",
		},
		"stack": {func(state *State) {
			state.Stack.Push(NewU256(3))
		},
			"Different stack",
		},
		"memory": {func(state *State) {
			state.Memory.Write([]byte{1, 2, 3}, 1)
		},
			"Different memory value",
		},
		"storage": {func(state *State) {
			state.Storage.SetCurrent(NewU256(4), NewU256(5))
		},
			"Different current entry",
		},
		"accounts": {func(state *State) {
			state.Accounts.SetBalance(tosca.Address{0x01}, NewU256(6))
		}, "Different account entry",
		},
		"logs": {func(state *State) {
			state.Logs.AddLog([]byte{10, 11}, NewU256(21), NewU256(22))
		},
			"Different log count",
		},
		"call_context": {func(state *State) {
			state.CallContext.AccountAddress = tosca.Address{0xff}
		},
			"Different call context",
		},
		"block_context": {func(state *State) {
			state.BlockContext.BlockNumber = 251
		},
			"Different block context",
		},
		"transaction_context": {func(state *State) {
			state.TransactionContext.OriginAddress = tosca.Address{0xff}
		},
			"Different transaction context",
		},
		"call_data": {func(state *State) {
			state.CallData = NewBytes([]byte{245})
		},
			"Different call data",
		},
		"last_call_return_data": {func(state *State) {
			state.LastCallReturnData = NewBytes([]byte{244})
		},
			"Different last call return data",
		},
		"return_data": {func(state *State) {
			state.Status = Stopped // return data is only checked when not running
			state.ReturnData = NewBytes([]byte{45})
		},
			"Different return data",
		},
		"has_self_destructed": {func(state *State) {
			state.HasSelfDestructed = false
		},
			"Different has-self-destructed",
		},
		"self_destructed_journal": {func(state *State) {
			state.SelfDestructedJournal = []SelfDestructEntry{{tosca.Address{0x01}, tosca.Address{0x04}}}
		},
			"Different has-self-destructed journal entry",
		},
		"block_number_hashes": {func(state *State) {
			state.RecentBlockHashes = NewImmutableHashArray(tosca.Hash{0x02})
		},
			"Different block hash at index 0: 0200000000000000000000000000000000000000000000000000000000000000 vs 0100000000000000000000000000000000000000000000000000000000000000",
		},
	}
	return tests
}

////////////////////////////////////////////////////////////
// State tests

func TestState_Clone(t *testing.T) {
	tests := getTestChanges()
	s1 := getNewFilledState()

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			s2 := s1.Clone()
			if !s1.Eq(s2) {
				t.Fatalf("clones are not equal")
			}
			test.change(s2)
			if s2.Eq(s1) {
				t.Errorf("clones are not independent")
			}
		})
	}
}

func TestState_Diff(t *testing.T) {
	tests := getTestChanges()
	s1 := getNewFilledState()

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			s2 := getNewFilledState()
			diffs := s1.Diff(s2)
			if len(diffs) != 0 {
				t.Errorf("unexpected differences: %v", diffs)
			}

			test.change(s2)
			diffs = s2.Diff(s1)
			if !strings.Contains(diffs[len(diffs)-1], test.want) {
				t.Errorf("unexpected differences: want %s, got %s", test.want, diffs[len(diffs)-1])
			}
		})
	}
}

func TestState_EqualAndDiffAreCompatible(t *testing.T) {
	tests := getTestChanges()
	s1 := getNewFilledState()

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			s2 := getNewFilledState()
			if s1.Eq(s2) && len(s2.Diff(s1)) != 0 {
				t.Errorf("state is equal but diffs are not empty")
			}
			test.change(s2)
			if !s1.Eq(s2) && len(s2.Diff(s1)) == 0 {
				t.Errorf("state is not equal but diffs are empty")
			}
		})
	}
}

func TestState_Equal_PcBeyondCodeAreTreatedEqual(t *testing.T) {
	const N = 10
	code := NewCode(make([]byte, N))

	s1 := NewState(code)
	s2 := NewState(code)

	s1.Pc = N - 1
	s2.Pc = N - 1

	if !s1.Eq(s2) {
		t.Errorf("states should be considered equal")
	}

	s1.Pc = N
	if s1.Eq(s2) {
		t.Errorf("states should not be considered equal")
	}

	s2.Pc = N
	if !s1.Eq(s2) {
		t.Errorf("states should be considered equal")
	}

	s1.Pc = N + 1
	if !s1.Eq(s2) {
		t.Errorf("states should be considered equal")
	}
}

func TestState_EqFailureStates(t *testing.T) {
	s1 := NewState(NewCode([]byte{}))
	s2 := NewState(NewCode([]byte{}))

	s1.Status = Failed
	s1.Gas = 1

	s2.Status = Failed
	s2.Gas = 2

	if !s1.Eq(s2) {
		t.Errorf("This field should not be considered when status is Failed")
	}
}

func TestState_PrinterStatus(t *testing.T) {
	s := NewState(NewCode([]byte{}))
	s.Status = Running

	r := regexp.MustCompile("Status: ([[:alpha:]]+)")
	match := r.FindStringSubmatch(s.String())

	if len(match) != 2 {
		t.Fatal("invalid print, did not find 'Status' text")
	}

	want := s.Status.String()
	got := match[1]

	if got != want {
		t.Errorf("invalid print, wanted %s, got %s", want, got)
	}
}

func TestState_PrinterRevision(t *testing.T) {
	s := NewState(NewCode([]byte{}))
	s.Revision = tosca.R10_London

	r := regexp.MustCompile("Revision: ([[:alpha:]]+)")
	match := r.FindStringSubmatch(s.String())

	if len(match) != 2 {
		t.Fatal("invalid print, did not find 'Revision' text")
	}

	want := s.Revision.String()
	got := match[1]

	if got != want {
		t.Errorf("invalid print, wanted %s, got %s", want, got)
	}
}

func TestState_PrinterPc(t *testing.T) {
	s := NewState(NewCode([]byte{byte(vm.STOP)}))
	s.Pc = 1

	r := regexp.MustCompile(`Pc: ([[:digit:]]+) \(0x([0-9a-f]{4})\)`)
	match := r.FindStringSubmatch(s.String())

	if len(match) != 3 {
		t.Fatal("invalid print, did not find 'Pc' text")
	}

	want := fmt.Sprintf("%d", s.Pc)
	got := match[1]
	if got != want {
		t.Errorf("invalid print, wanted %s, got %s", want, got)
	}

	want = fmt.Sprintf("%04x", s.Pc)
	got = match[2]
	if got != want {
		t.Errorf("invalid print, wanted %s, got %s", want, got)
	}
}

func TestState_PrinterPcData(t *testing.T) {
	s := NewState(NewCode([]byte{byte(vm.PUSH1), 7}))
	s.Pc = 1

	r := regexp.MustCompile(`\(points to data\)`)
	match := r.MatchString(s.String())

	if !match {
		t.Error("invalid print, did not find 'points to data' text")
	}
}

func TestState_PrinterPcOperation(t *testing.T) {
	s := NewState(NewCode([]byte{byte(vm.ADD)}))
	s.Pc = 0

	r := regexp.MustCompile(`\(operation: ([[:alpha:]]+)\)`)
	match := r.FindStringSubmatch(s.String())

	if len(match) != 2 {
		t.Fatal("invalid print, did not find 'operation' text")
	}

	want := vm.OpCode(s.Code.code[s.Pc]).String()
	got := match[1]
	if want != got {
		t.Errorf("invalid print, wanted %s, got %s", want, got)
	}
}

func TestState_PrinterPcOutOfBounds(t *testing.T) {
	s := NewState(NewCode([]byte{byte(vm.STOP)}))
	s.Pc = 2

	r := regexp.MustCompile(`\(out of bounds\)`)
	match := r.MatchString(s.String())

	if !match {
		t.Error("invalid print, did not find 'out of bounds' text")
	}
}

func TestState_PrinterGas(t *testing.T) {
	s := NewState(NewCode([]byte{byte(vm.STOP)}))
	s.Gas = 42

	r := regexp.MustCompile("Gas: ([[:digit:]]+)")
	match := r.FindStringSubmatch(s.String())

	if len(match) != 2 {
		t.Fatal("invalid print, did not find 'Gas' text")
	}

	want := fmt.Sprintf("%d", s.Gas)
	got := match[1]
	if want != got {
		t.Errorf("invalid print, wanted %s, got %s", want, got)
	}
}

func TestState_PrinterCode(t *testing.T) {
	s := NewState(NewCode([]byte{byte(vm.PUSH2), 42, 42, byte(vm.ADD), byte(vm.STOP)}))

	r := regexp.MustCompile("Code: ([0-9a-f]+)")
	match := r.FindStringSubmatch(s.String())

	if len(match) != 2 {
		t.Fatal("invalid print, did not find 'Code' text")
	}

	want := s.Code.String()
	got := match[1]
	if want != got {
		t.Errorf("invalid print, wanted %s, got %s", want, got)
	}
}

func TestState_PrinterAbbreviatedCode(t *testing.T) {
	var longCode []byte
	for i := 0; i < dataCutoffLength+1; i++ {
		longCode = append(longCode, byte(vm.INVALID))
	}

	s := NewState(NewCode(longCode))

	r := regexp.MustCompile(`Code: ([0-9a-f]+)... \(size: ([[:digit:]]+)\)`)
	match := r.FindStringSubmatch(s.String())

	if len(match) != 3 {
		t.Fatal("invalid print, did not find 'Code' text")
	}

	want := fmt.Sprintf("%x", s.Code.code[:dataCutoffLength])
	got := match[1]
	if want != got {
		t.Errorf("invalid print, wanted %s, got %s", want, got)
	}

	want = fmt.Sprintf("%d", len(s.Code.code))
	got = match[2]
	if want != got {
		t.Errorf("invalid print, wanted %s, got %s", want, got)
	}
}

func TestState_PrinterStackSize(t *testing.T) {
	s := NewState(NewCode([]byte{}))
	s.Stack.Push(NewU256(1))
	s.Stack.Push(NewU256(2))
	s.Stack.Push(NewU256(3))

	r := regexp.MustCompile(`Stack size: ([[:digit:]]+)`)
	match := r.FindStringSubmatch(s.String())

	if len(match) != 2 {
		t.Fatal("invalid print, did not find stack size")
	}

	if want, got := "3", match[1]; want != got {
		t.Errorf("invalid stack size, want %v, got %v", want, got)
	}
}

func TestState_PrinterMemorySize(t *testing.T) {
	s := NewState(NewCode([]byte{}))
	s.Memory.Write([]byte{1, 2, 3}, 31)

	r := regexp.MustCompile(`Memory size: ([[:digit:]]+)`)
	match := r.FindStringSubmatch(s.String())

	if len(match) != 2 {
		t.Fatal("invalid print, did not find memory size")
	}

	if want, got := "64", match[1]; want != got {
		t.Errorf("invalid memory size, want %v, got %v", want, got)
	}
}

func TestState_PrinterRecentBlockHashes(t *testing.T) {
	s := NewState(NewCode([]byte{byte(vm.BLOCKHASH)}))
	s.Stack.Push(NewU256(0))
	s.RecentBlockHashes = NewImmutableHashArray(tosca.Hash{0x01})

	r := regexp.MustCompile(`Hash of block 0: 0x([0-9a-fA-F]+)`) // \[([0-9a-f]+)\]
	str := s.String()
	match := r.FindStringSubmatch(str)

	if len(match) != 2 {
		t.Fatal("invalid print, did not find recent block hashes")
	}

	if want, got := "0100000000000000000000000000000000000000000000000000000000000000", match[1]; want != got {
		t.Errorf("invalid recent block hashes, want %v, got %v", want, got)
	}
}

func TestState_DiffMatch(t *testing.T) {
	s1 := NewState(NewCode([]byte{byte(vm.PUSH2), 7, 4, byte(vm.ADD), byte(vm.STOP)}))
	s1.Status = Running
	s1.Revision = tosca.R10_London
	s1.Pc = 3
	s1.Gas = 42
	s1.GasRefund = 63
	s1.Stack.Push(NewU256(42))
	s1.Memory.Write([]byte{1, 2, 3}, 31)
	s1.Storage.MarkWarm(NewU256(42))
	s1.Logs.AddLog([]byte{4, 5, 6}, NewU256(21), NewU256(22))
	s1.CallContext = CallContext{AccountAddress: tosca.Address{0x01}}
	s1.BlockContext = BlockContext{BlockNumber: 1}
	s1.TransactionContext = &TransactionContext{BlobHashes: []tosca.Hash{{4, 3, 2, 1}}}
	s1.CallData = NewBytes([]byte{1})
	s1.LastCallReturnData = NewBytes([]byte{1})
	s1.HasSelfDestructed = true
	s1.SelfDestructedJournal = []SelfDestructEntry{{tosca.Address{0x01}, tosca.Address{0x01}}}
	s1.RecentBlockHashes = NewImmutableHashArray(tosca.Hash{0x01})

	s2 := NewState(NewCode([]byte{byte(vm.PUSH2), 7, 4, byte(vm.ADD), byte(vm.STOP)}))
	s2.Status = Running
	s2.Revision = tosca.R10_London
	s2.Pc = 3
	s2.Gas = 42
	s2.GasRefund = 63
	s2.Stack.Push(NewU256(42))
	s2.Memory.Write([]byte{1, 2, 3}, 31)
	s2.Storage.MarkWarm(NewU256(42))
	s2.Logs.AddLog([]byte{4, 5, 6}, NewU256(21), NewU256(22))
	s2.CallContext = CallContext{AccountAddress: tosca.Address{0x01}}
	s2.BlockContext = BlockContext{BlockNumber: 1}
	s2.TransactionContext = &TransactionContext{BlobHashes: []tosca.Hash{{4, 3, 2, 1}}}
	s2.CallData = NewBytes([]byte{1})
	s2.LastCallReturnData = NewBytes([]byte{1})
	s2.HasSelfDestructed = true
	s2.SelfDestructedJournal = []SelfDestructEntry{{tosca.Address{0x01}, tosca.Address{0x01}}}
	s2.RecentBlockHashes = NewImmutableHashArray(tosca.Hash{0x01})

	diffs := s1.Diff(s2)

	if len(diffs) != 0 {
		t.Logf("invalid diff, expected no differences, found %d:\n", len(diffs))
		for _, diff := range diffs {
			t.Logf("%s\n", diff)
		}
		t.Fail()
	}
}

func TestState_DiffMismatch(t *testing.T) {
	s1 := NewState(NewCode([]byte{byte(vm.PUSH2), 7, 4, byte(vm.ADD)}))
	s1.Status = Stopped
	s1.Revision = tosca.R09_Berlin
	s1.Pc = 0
	s1.Gas = 7
	s1.GasRefund = 8
	s1.Stack.Push(NewU256(42))
	s1.Memory.Write([]byte{1, 2, 3}, 31)
	s1.Storage.MarkWarm(NewU256(42))
	s1.Logs.AddLog([]byte{4, 5, 6}, NewU256(21), NewU256(22))
	s1.Logs.AddLog([]byte{4, 5, 6}, NewU256(21), NewU256(22))
	s1.CallContext = CallContext{AccountAddress: tosca.Address{0xff}}
	s1.BlockContext = BlockContext{BlockNumber: 1}
	s1.TransactionContext = &TransactionContext{BlobHashes: []tosca.Hash{{4, 3, 2, 1}}}
	s1.CallData = NewBytes([]byte{1})
	s1.LastCallReturnData = NewBytes([]byte{1})
	s1.HasSelfDestructed = true
	s1.SelfDestructedJournal = []SelfDestructEntry{{tosca.Address{0x01}, tosca.Address{0x01}}}
	s1.RecentBlockHashes = NewImmutableHashArray(tosca.Hash{0x01})

	s2 := NewState(NewCode([]byte{byte(vm.PUSH2), 7, 5, byte(vm.ADD)}))
	s2.Status = Running
	s2.Revision = tosca.R10_London
	s2.Pc = 3
	s2.Gas = 42
	s2.GasRefund = 9
	s2.Stack.Push(NewU256(16))
	s2.Memory.Write([]byte{1, 2, 4}, 31)
	s2.Storage.MarkCold(NewU256(42))
	s2.Logs.AddLog([]byte{4, 7, 6}, NewU256(24), NewU256(22))
	s2.CallContext = CallContext{AccountAddress: tosca.Address{0xef}}
	s2.BlockContext = BlockContext{BlockNumber: 251}
	s2.TransactionContext = &TransactionContext{BlobHashes: []tosca.Hash{{1}}}
	s2.CallData = NewBytes([]byte{250})
	s2.LastCallReturnData = NewBytes([]byte{249})
	s2.HasSelfDestructed = false
	s2.SelfDestructedJournal = []SelfDestructEntry{{tosca.Address{0xf3}, tosca.Address{0xf3}}}
	s2.RecentBlockHashes = NewImmutableHashArray(tosca.Hash{0xf2})

	diffs := s1.Diff(s2)

	expectedDiffs := []string{
		"Different status",
		"Different revision",
		"Different pc",
		"Different gas",
		"Different gas refund",
		"Different code",
		"Different stack",
		"Different memory value",
		"Different warm entry",
		"Different log count",
		"Different topics for log entry",
		"Different data for log entry",
		"Different call context",
		"Different block context",
		"Different transaction context",
		"Different call data",
		"Different last call return data",
		"Different has-self-destructed",
		"Different has-self-destructed journal entry",
		"Different block hash at index 0: 0100000000000000000000000000000000000000000000000000000000000000 vs f200000000000000000000000000000000000000000000000000000000000000",
	}

	if len(diffs) != len(expectedDiffs) {
		t.Logf("invalid diff, expected %d differences, found %d:\n", len(expectedDiffs), len(diffs))
		for _, diff := range diffs {
			t.Logf("%s\n", diff)
		}
		t.FailNow()
	}

	for i := 0; i < len(diffs); i++ {
		if !strings.Contains(diffs[i], expectedDiffs[i]) {
			t.Errorf("invalid diff, expected '%s' found '%s'", expectedDiffs[i], diffs[i])
		}
	}
}

func TestState_StatusCodeMarshal(t *testing.T) {
	tests := map[StatusCode]string{
		Running:  "\"running\"",
		Stopped:  "\"stopped\"",
		Reverted: "\"reverted\"",
		Failed:   "\"failed\"",
	}

	for input, expected := range tests {
		marshaled, err := input.MarshalJSON()
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if !bytes.Equal(marshaled, []byte(expected)) {
			t.Errorf("Unexpected marshaled status code, wanted: %v vs got: %v", expected, marshaled)
		}
	}
}

func TestState_StatusCodeMarshalError(t *testing.T) {
	statusCodes := []StatusCode{StatusCode(42), StatusCode(100)}
	for _, status := range statusCodes {
		marshaled, err := status.MarshalJSON()
		if err == nil {
			t.Errorf("Expected error but got: %v", marshaled)
		}
	}
}

func TestState_StatusCodeUnmarshal(t *testing.T) {
	tests := map[string]StatusCode{
		"\"running\"":  Running,
		"\"stopped\"":  Stopped,
		"\"reverted\"": Reverted,
		"\"failed\"":   Failed,
	}

	for input, expected := range tests {
		var status StatusCode
		err := status.UnmarshalJSON([]byte(input))
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if status != expected {
			t.Errorf("Unexpected unmarshaled status, wanted: %v vs got: %v", expected, status)
		}
	}
}

func TestState_StatusCodeUnmarshalError(t *testing.T) {
	tests := []string{"StatusCode(42)", "Error", "running"}
	var status StatusCode
	for _, input := range tests {
		err := status.UnmarshalJSON([]byte(input))
		if err == nil {
			t.Errorf("Expected error but got: %v", status)
		}
	}
}

func TestState_EqualConsidersReturnDataOnlyWhenStoppedOrReverted(t *testing.T) {
	dataValue1 := []byte{1}
	dataValue2 := []byte{2}
	tests := map[string]struct {
		status StatusCode
		wanted bool
	}{
		"stopped":  {Stopped, false},
		"reverted": {Reverted, false},
		"running":  {Running, true},
		"failed":   {Failed, true},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			s1 := getNewFilledState()
			s1.Status = test.status
			s1.ReturnData = NewBytes(dataValue1)
			s2 := s1.Clone()
			s2.ReturnData = NewBytes(dataValue2)
			if want, got := test.wanted, s1.Eq(s2); want != got {
				t.Errorf("unexpected equality result, wanted %t, got %t", want, got)
			}
		})
	}
}

func TestState_EqualityConsidersRelevantFieldsDependingOnStatus(t *testing.T) {
	allButFailed := []StatusCode{Running, Stopped, Reverted}
	allStatusCodes := append(allButFailed, Failed)
	if int(NumStatusCodes) != len(allStatusCodes) {
		t.Fatalf("Missing status codes in test, got %v", allStatusCodes)
	}

	onlyRunning := []StatusCode{Running}
	tests := map[string]struct {
		modify      func(*State)
		relevantFor []StatusCode
	}{
		"status": {
			modify:      func(s *State) { s.Status++ },
			relevantFor: allStatusCodes,
		},
		"revision": {
			modify:      func(s *State) { s.Revision++ },
			relevantFor: allButFailed,
		},
		"read_only": {
			modify:      func(s *State) { s.ReadOnly = !s.ReadOnly },
			relevantFor: allButFailed,
		},
		"gas": {
			modify:      func(s *State) { s.Gas++ },
			relevantFor: allButFailed,
		},
		"gas_refund": {
			modify:      func(s *State) { s.GasRefund++ },
			relevantFor: allButFailed,
		},
		"code": {
			modify:      func(s *State) { s.Code = NewCode([]byte{3, 2, 1}) },
			relevantFor: allButFailed,
		},
		"pc": {
			modify:      func(s *State) { s.Pc++ },
			relevantFor: onlyRunning,
		},
		"stack": {
			modify: func(s *State) {
				if s.Stack.Size() > 0 {
					s.Stack.Pop()
				} else {
					s.Stack.Push(U256{})
				}
			},
			relevantFor: onlyRunning,
		},
		"memory": {
			modify:      func(s *State) { s.Memory.Append([]byte{1}) },
			relevantFor: onlyRunning,
		},
		"storage": {
			modify:      func(s *State) { s.Storage.SetCurrent(NewU256(1), NewU256(2)) },
			relevantFor: allButFailed,
		},
		"accounts": {
			modify:      func(s *State) { s.Accounts.SetBalance(tosca.Address{}, NewU256(1)) },
			relevantFor: allButFailed,
		},
		"logs": {
			modify:      func(s *State) { s.Logs.AddLog([]byte{}) },
			relevantFor: allButFailed,
		},
		"call_context": {
			modify:      func(s *State) { s.CallContext.AccountAddress[0]++ },
			relevantFor: allButFailed,
		},
		"block_context": {
			modify:      func(s *State) { s.BlockContext.BlockNumber++ },
			relevantFor: allButFailed,
		},
		"transaction_context": {
			modify:      func(s *State) { s.TransactionContext.OriginAddress[0]++ },
			relevantFor: allButFailed,
		},
		"call_data": {
			modify:      func(s *State) { s.CallData = NewBytes([]byte{1, 2, 3}) },
			relevantFor: allButFailed,
		},
		"call_journal": {
			modify:      func(s *State) { s.CallJournal.Future = []FutureCall{{Success: false}} },
			relevantFor: allButFailed,
		},
		"last_call_return_data": {
			modify:      func(s *State) { s.LastCallReturnData = NewBytes([]byte{1, 2, 3}) },
			relevantFor: onlyRunning,
		},
		"return_data": {
			modify:      func(s *State) { s.ReturnData = NewBytes([]byte{1, 2, 3}) },
			relevantFor: []StatusCode{Stopped, Reverted},
		},
		"has_self_destructed": {
			modify:      func(s *State) { s.HasSelfDestructed = true },
			relevantFor: allButFailed,
		},
		"has_self_destructed_journal": {
			modify: func(s *State) {
				s.SelfDestructedJournal = []SelfDestructEntry{{tosca.Address{0xf3}, tosca.Address{0xf3}}}
			},
			relevantFor: allButFailed,
		},
		"block_number_hashes": {
			modify:      func(s *State) { s.RecentBlockHashes = NewImmutableHashArray(tosca.Hash{0xf2}) },
			relevantFor: allButFailed,
		},
	}

	code := NewCode([]byte{1, 2, 3})
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			for _, status := range allStatusCodes {
				t.Run(fmt.Sprintf("%v", status), func(t *testing.T) {
					s1 := NewState(code)
					s1.Status = status
					s2 := s1.Clone()
					test.modify(s2)

					wantToBeDetected := slices.Contains(test.relevantFor, status)
					detectedAsDifferent := !s1.Eq(s2)
					if wantToBeDetected != detectedAsDifferent {
						t.Errorf(
							"wanted change to be considered = %t, got %t",
							wantToBeDetected, detectedAsDifferent,
						)
					}
				})
			}
		})
	}
}

func TestState_RecycledMembers(t *testing.T) {
	state := NewState(NewCode([]byte{byte(vm.INVALID)}))
	state.Stack = NewStack()

	if state.Stack == nil {
		t.Error("No stack was returned from stack pool")
	}

	state.Release()
	if state.Stack != nil {
		t.Error("Stack should have been returned and nil")
	}
}

func BenchmarkState_CloneState(b *testing.B) {
	state := getNewFilledState()
	for i := 0; i < b.N; i++ {
		clone := state.Clone()
		_ = clone
	}
}

func BenchmarkState_EqState(b *testing.B) {
	state := getNewFilledState()
	clone := state.Clone()
	for i := 0; i < b.N; i++ {
		_ = state.Eq(clone)
	}
}
