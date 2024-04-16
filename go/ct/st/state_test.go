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

package st

import (
	"bytes"
	"fmt"
	"regexp"
	"slices"
	"strings"
	"testing"

	. "github.com/Fantom-foundation/Tosca/go/ct/common"
	"github.com/Fantom-foundation/Tosca/go/vm"
)

func TestState_CloneCreatesEqualState(t *testing.T) {
	state := NewState(NewCode([]byte{byte(ADD)}))
	state.Status = Stopped
	state.Revision = R10_London
	state.Pc = 1
	state.Gas = 2
	state.GasRefund = 15
	state.Stack.Push(NewU256(3))
	state.Memory.Write([]byte{1, 2, 3}, 1)
	state.Storage.SetCurrent(NewU256(4), NewU256(5))
	state.Storage.SetOriginal(NewU256(6), NewU256(7))
	state.Logs.AddLog([]byte{10, 11}, NewU256(21), NewU256(22))
	state.CallContext.AccountAddress = vm.Address{0xff}
	state.CallContext.OriginAddress = vm.Address{0xfe}
	state.CallContext.CallerAddress = vm.Address{0xfd}
	state.CallContext.Value = NewU256(252)
	state.BlockContext.BlockNumber = 251
	state.BlockContext.CoinBase[0] = 0xfa
	state.BlockContext.GasLimit = 249
	state.BlockContext.GasPrice = NewU256(248)
	state.BlockContext.Difficulty = NewU256(247)
	state.BlockContext.TimeStamp = 246
	state.CallData = NewBytes([]byte{245})
	state.LastCallReturnData = NewBytes([]byte{244})

	clone := state.Clone()
	if !state.Eq(clone) {
		t.Errorf("clone failed to copy. %v", state.Diff(clone))
	}
}

func TestState_CloneIsIndependent(t *testing.T) {
	state := NewState(NewCode([]byte{byte(ADD)}))
	state.Status = Stopped
	state.Revision = R10_London
	state.Pc = 1
	state.Gas = 2
	state.GasRefund = 15
	state.Stack.Push(NewU256(3))
	state.Memory.Write([]byte{1, 2, 3}, 1)
	state.Storage.SetCurrent(NewU256(4), NewU256(5))
	state.Storage.SetOriginal(NewU256(6), NewU256(7))
	state.Logs.AddLog([]byte{10, 11}, NewU256(21), NewU256(22))
	state.CallContext.AccountAddress = vm.Address{0xff}
	state.CallContext.OriginAddress = vm.Address{0xfe}
	state.CallContext.CallerAddress = vm.Address{0xfd}
	state.CallContext.Value = NewU256(252)
	state.BlockContext.BlockNumber = 251
	state.BlockContext.CoinBase[0] = 0xfa
	state.BlockContext.GasLimit = 249
	state.BlockContext.GasPrice = NewU256(248)
	state.BlockContext.Difficulty = NewU256(247)
	state.BlockContext.TimeStamp = 246
	state.CallData = NewBytes([]byte{245})
	state.LastCallReturnData = NewBytes([]byte{244})

	clone := state.Clone()
	clone.Status = Running
	clone.Revision = R09_Berlin
	clone.Pc = 4
	clone.Gas = 5
	clone.GasRefund = 16
	clone.Stack.Push(NewU256(6))
	clone.Memory.Write([]byte{4, 5, 6, 7}, 64)
	clone.Storage.SetCurrent(NewU256(7), NewU256(16))
	clone.Storage.SetOriginal(NewU256(6), NewU256(6))
	clone.Storage.MarkWarm(NewU256(42))
	clone.Logs.Entries[0].Data[0] = 31
	clone.Logs.Entries[0].Topics[0] = NewU256(41)
	clone.CallContext.AccountAddress = vm.Address{0x01}
	clone.CallContext.OriginAddress = vm.Address{0x02}
	clone.CallContext.CallerAddress = vm.Address{0x03}
	clone.CallContext.Value = NewU256(4)
	clone.BlockContext.BlockNumber = 5
	clone.BlockContext.CoinBase[0] = 0x06
	clone.BlockContext.GasLimit = 7
	clone.BlockContext.GasPrice = NewU256(8)
	clone.BlockContext.Difficulty = NewU256(9)
	clone.BlockContext.TimeStamp = 10
	clone.CallData = NewBytes([]byte{11})
	clone.LastCallReturnData = NewBytes([]byte{12})

	ok := state.Status == Stopped &&
		state.Revision == R10_London &&
		state.Pc == 1 &&
		state.Gas == 2 &&
		state.GasRefund == 15 &&
		state.Stack.Size() == 1 &&
		state.Stack.Get(0).Uint64() == 3 &&
		state.Memory.Size() == 32 &&
		state.Storage.GetCurrent(NewU256(4)).Eq(NewU256(5)) &&
		state.Storage.GetCurrent(NewU256(7)).IsZero() &&
		state.Storage.GetOriginal(NewU256(6)).Eq(NewU256(7)) &&
		!state.Storage.IsWarm(NewU256(42)) &&
		state.Logs.Entries[0].Data[0] == 10 &&
		state.Logs.Entries[0].Topics[0] == NewU256(21) &&
		state.CallContext.AccountAddress == vm.Address{0xff} &&
		state.CallContext.OriginAddress == vm.Address{0xfe} &&
		state.CallContext.CallerAddress == vm.Address{0xfd} &&
		state.CallContext.Value.Eq(NewU256(252)) &&
		state.BlockContext.BlockNumber == 251 &&
		state.BlockContext.CoinBase[0] == 0xfa &&
		state.BlockContext.GasLimit == 249 &&
		state.BlockContext.GasPrice == NewU256(248) &&
		state.BlockContext.Difficulty == NewU256(247) &&
		state.BlockContext.TimeStamp == 246 &&
		state.CallData.Get(0, 1)[0] == 245 &&
		state.LastCallReturnData.Get(0, 1)[0] == 244
	if !ok {
		t.Errorf("clone is not independent")
	}
}

func TestState_Eq(t *testing.T) {
	s1 := NewState(NewCode([]byte{}))
	s2 := NewState(NewCode([]byte{}))
	if !s1.Eq(s2) {
		t.Fail()
	}

	s1.Status = Running
	s2.Status = Stopped
	if s1.Eq(s2) {
		t.Fail()
	}
	s2.Status = Running

	s1.Revision = R07_Istanbul
	s2.Revision = R10_London
	if s1.Eq(s2) {
		t.Fail()
	}
	s2.Revision = R07_Istanbul

	s1.Pc = 1
	s2.Pc = 2
	if s1.Eq(s2) {
		t.Fail()
	}
	s2.Pc = 1

	s1.Gas = 1
	s2.Gas = 2
	if s1.Eq(s2) {
		t.Fail()
	}
	s2.Gas = 1

	s1.GasRefund = 1
	s2.GasRefund = 2
	if s1.Eq(s2) {
		t.Fail()
	}
	s2.GasRefund = 1

	s1.Stack.Push(NewU256(1))
	if s1.Eq(s2) {
		t.Fail()
	}
	s2.Stack.Push(NewU256(1))

	if !s1.Eq(s2) {
		t.Fail()
	}

	s1.Memory.Write([]byte{1, 2, 3}, 1)
	if s1.Eq(s2) {
		t.Fail()
	}
	s2.Memory.Write([]byte{1, 2, 3}, 1)

	s1.Storage.SetCurrent(NewU256(42), NewU256(32))
	if s1.Eq(s2) {
		t.Fail()
	}
	s2.Storage.SetCurrent(NewU256(42), NewU256(32))

	s1.Logs.AddLog([]byte{4, 5, 6}, NewU256(21), NewU256(22))
	if s1.Eq(s2) {
		t.Fail()
	}
	s2.Logs.AddLog([]byte{4, 5, 6}, NewU256(21), NewU256(22))

	s1 = NewState(NewCode([]byte{byte(ADD), byte(STOP)}))
	s2 = NewState(NewCode([]byte{byte(ADD), byte(ADD)}))
	if s1.Eq(s2) {
		t.Fail()
	}
	s2 = NewState(NewCode([]byte{byte(ADD), byte(STOP)}))

	s1.CallContext = CallContext{AccountAddress: vm.Address{0x00}}
	s2.CallContext = CallContext{AccountAddress: vm.Address{0xff}}
	if s1.Eq(s2) {
		t.Fail()
	}
	s2.CallContext = s1.CallContext

	s1.BlockContext = BlockContext{BlockNumber: 0}
	s2.BlockContext = BlockContext{BlockNumber: 251}
	if s1.Eq(s2) {
		t.Fail()
	}
	s2.BlockContext = s1.BlockContext

	s1.CallData = NewBytes([]byte{1})
	s2.CallData = NewBytes([]byte{250})
	if s1.Eq(s2) {
		t.Fail()
	}
	s2.CallData = s1.CallData

	s1.LastCallReturnData = NewBytes([]byte{1})
	s2.LastCallReturnData = NewBytes([]byte{249})
	if s1.Eq(s2) {
		t.Fail()
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
		t.Fail()
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
	s.Revision = R10_London

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
	s := NewState(NewCode([]byte{byte(STOP)}))
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
	s := NewState(NewCode([]byte{byte(PUSH1), 7}))
	s.Pc = 1

	r := regexp.MustCompile(`\(points to data\)`)
	match := r.MatchString(s.String())

	if !match {
		t.Error("invalid print, did not find 'points to data' text")
	}
}

func TestState_PrinterPcOperation(t *testing.T) {
	s := NewState(NewCode([]byte{byte(ADD)}))
	s.Pc = 0

	r := regexp.MustCompile(`\(operation: ([[:alpha:]]+)\)`)
	match := r.FindStringSubmatch(s.String())

	if len(match) != 2 {
		t.Fatal("invalid print, did not find 'operation' text")
	}

	want := OpCode(s.Code.code[s.Pc]).String()
	got := match[1]
	if want != got {
		t.Errorf("invalid print, wanted %s, got %s", want, got)
	}
}

func TestState_PrinterPcOutOfBounds(t *testing.T) {
	s := NewState(NewCode([]byte{byte(STOP)}))
	s.Pc = 2

	r := regexp.MustCompile(`\(out of bounds\)`)
	match := r.MatchString(s.String())

	if !match {
		t.Error("invalid print, did not find 'out of bounds' text")
	}
}

func TestState_PrinterGas(t *testing.T) {
	s := NewState(NewCode([]byte{byte(STOP)}))
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
	s := NewState(NewCode([]byte{byte(PUSH2), 42, 42, byte(ADD), byte(STOP)}))

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
		longCode = append(longCode, byte(INVALID))
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

func TestState_DiffMatch(t *testing.T) {
	s1 := NewState(NewCode([]byte{byte(PUSH2), 7, 4, byte(ADD), byte(STOP)}))
	s1.Status = Running
	s1.Revision = R10_London
	s1.Pc = 3
	s1.Gas = 42
	s1.GasRefund = 63
	s1.Stack.Push(NewU256(42))
	s1.Memory.Write([]byte{1, 2, 3}, 31)
	s1.Storage.MarkWarm(NewU256(42))
	s1.Logs.AddLog([]byte{4, 5, 6}, NewU256(21), NewU256(22))
	s1.CallContext = CallContext{AccountAddress: vm.Address{0x01}}
	s1.BlockContext = BlockContext{BlockNumber: 1}
	s1.CallData = NewBytes([]byte{1})
	s1.LastCallReturnData = NewBytes([]byte{1})

	s2 := NewState(NewCode([]byte{byte(PUSH2), 7, 4, byte(ADD), byte(STOP)}))
	s2.Status = Running
	s2.Revision = R10_London
	s2.Pc = 3
	s2.Gas = 42
	s2.GasRefund = 63
	s2.Stack.Push(NewU256(42))
	s2.Memory.Write([]byte{1, 2, 3}, 31)
	s2.Storage.MarkWarm(NewU256(42))
	s2.Logs.AddLog([]byte{4, 5, 6}, NewU256(21), NewU256(22))
	s2.CallContext = CallContext{AccountAddress: vm.Address{0x01}}
	s2.BlockContext = BlockContext{BlockNumber: 1}
	s2.CallData = NewBytes([]byte{1})
	s2.LastCallReturnData = NewBytes([]byte{1})

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
	s1 := NewState(NewCode([]byte{byte(PUSH2), 7, 4, byte(ADD)}))
	s1.Status = Stopped
	s1.Revision = R09_Berlin
	s1.Pc = 0
	s1.Gas = 7
	s1.GasRefund = 8
	s1.Stack.Push(NewU256(42))
	s1.Memory.Write([]byte{1, 2, 3}, 31)
	s1.Storage.MarkWarm(NewU256(42))
	s1.Logs.AddLog([]byte{4, 5, 6}, NewU256(21), NewU256(22))
	s1.Logs.AddLog([]byte{4, 5, 6}, NewU256(21), NewU256(22))
	s1.CallContext = CallContext{AccountAddress: vm.Address{0xff}}
	s1.BlockContext = BlockContext{BlockNumber: 1}
	s1.CallData = NewBytes([]byte{1})
	s1.LastCallReturnData = NewBytes([]byte{1})

	s2 := NewState(NewCode([]byte{byte(PUSH2), 7, 5, byte(ADD)}))
	s2.Status = Running
	s2.Revision = R10_London
	s2.Pc = 3
	s2.Gas = 42
	s2.GasRefund = 9
	s2.Stack.Push(NewU256(16))
	s2.Memory.Write([]byte{1, 2, 4}, 31)
	s2.Storage.MarkCold(NewU256(42))
	s2.Logs.AddLog([]byte{4, 7, 6}, NewU256(24), NewU256(22))
	s2.CallContext = CallContext{AccountAddress: vm.Address{0xef}}
	s2.BlockContext = BlockContext{BlockNumber: 251}
	s2.CallData = NewBytes([]byte{250})
	s2.LastCallReturnData = NewBytes([]byte{249})

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
		"Different call data",
		"Different last call return data",
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
			modify:      func(s *State) { s.Accounts.SetBalance(vm.Address{}, NewU256(1)) },
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
	state := NewState(NewCode([]byte{byte(INVALID)}))

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
