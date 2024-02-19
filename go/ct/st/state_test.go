package st

import (
	"bytes"
	"fmt"
	"regexp"
	"strings"
	"testing"

	. "github.com/Fantom-foundation/Tosca/go/ct/common"
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
	state.Storage.Current[NewU256(4)] = NewU256(5)
	state.Storage.Original[NewU256(6)] = NewU256(7)
	state.Logs.AddLog([]byte{10, 11}, NewU256(21), NewU256(22))
	state.CallContext.AccountAddress = Address{0xff}
	state.CallContext.OriginAddress = Address{0xfe}
	state.CallContext.CallerAddress = Address{0xfd}
	state.CallContext.Value = NewU256(252)
	state.BlockContext.BlockNumber = 251
	state.BlockContext.CoinBase[0] = 0xfa
	state.BlockContext.GasLimit = 249
	state.BlockContext.GasPrice = NewU256(248)
	state.BlockContext.Difficulty = NewU256(247)
	state.BlockContext.TimeStamp = 246
	state.CallData = append(state.CallData, 245)
	state.LastCallReturnData = append(state.LastCallReturnData, 244)

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
	state.Storage.Current[NewU256(4)] = NewU256(5)
	state.Storage.Original[NewU256(6)] = NewU256(7)
	state.Logs.AddLog([]byte{10, 11}, NewU256(21), NewU256(22))
	state.CallContext.AccountAddress = Address{0xff}
	state.CallContext.OriginAddress = Address{0xfe}
	state.CallContext.CallerAddress = Address{0xfd}
	state.CallContext.Value = NewU256(252)
	state.BlockContext.BlockNumber = 251
	state.BlockContext.CoinBase[0] = 0xfa
	state.BlockContext.GasLimit = 249
	state.BlockContext.GasPrice = NewU256(248)
	state.BlockContext.Difficulty = NewU256(247)
	state.BlockContext.TimeStamp = 246
	state.CallData = append(state.CallData, 245)
	state.LastCallReturnData = append(state.LastCallReturnData, 244)

	clone := state.Clone()
	clone.Status = Running
	clone.Revision = R09_Berlin
	clone.Pc = 4
	clone.Gas = 5
	clone.GasRefund = 16
	clone.Stack.Push(NewU256(6))
	clone.Memory.Write([]byte{4, 5, 6, 7}, 64)
	clone.Storage.Current[NewU256(7)] = NewU256(16)
	clone.Storage.Original[NewU256(6)] = NewU256(6)
	clone.Storage.MarkWarm(NewU256(42))
	clone.Logs.Entries[0].Data[0] = 31
	clone.Logs.Entries[0].Topics[0] = NewU256(41)
	clone.CallContext.AccountAddress = Address{0x01}
	clone.CallContext.OriginAddress = Address{0x02}
	clone.CallContext.CallerAddress = Address{0x03}
	clone.CallContext.Value = NewU256(4)
	clone.BlockContext.BlockNumber = 5
	clone.BlockContext.CoinBase[0] = 0x06
	clone.BlockContext.GasLimit = 7
	clone.BlockContext.GasPrice = NewU256(8)
	clone.BlockContext.Difficulty = NewU256(9)
	clone.BlockContext.TimeStamp = 10
	clone.CallData[0] = 11
	clone.LastCallReturnData[0] = 12

	ok := state.Status == Stopped &&
		state.Revision == R10_London &&
		state.Pc == 1 &&
		state.Gas == 2 &&
		state.GasRefund == 15 &&
		state.Stack.Size() == 1 &&
		state.Stack.Get(0).Uint64() == 3 &&
		state.Memory.Size() == 32 &&
		state.Storage.Current[NewU256(4)].Eq(NewU256(5)) &&
		state.Storage.Current[NewU256(7)].IsZero() &&
		state.Storage.Original[NewU256(6)].Eq(NewU256(7)) &&
		!state.Storage.IsWarm(NewU256(42)) &&
		state.Logs.Entries[0].Data[0] == 10 &&
		state.Logs.Entries[0].Topics[0] == NewU256(21) &&
		state.CallContext.AccountAddress == Address{0xff} &&
		state.CallContext.OriginAddress == Address{0xfe} &&
		state.CallContext.CallerAddress == Address{0xfd} &&
		state.CallContext.Value.Eq(NewU256(252)) &&
		state.BlockContext.BlockNumber == 251 &&
		state.BlockContext.CoinBase[0] == 0xfa &&
		state.BlockContext.GasLimit == 249 &&
		state.BlockContext.GasPrice == NewU256(248) &&
		state.BlockContext.Difficulty == NewU256(247) &&
		state.BlockContext.TimeStamp == 246 &&
		state.CallData[0] == 245 &&
		state.LastCallReturnData[0] == 244
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

	s1.Storage.Current[NewU256(42)] = NewU256(32)
	if s1.Eq(s2) {
		t.Fail()
	}
	s2.Storage.Current[NewU256(42)] = NewU256(32)

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

	s1.CallContext = CallContext{AccountAddress: Address{0x00}}
	s2.CallContext = CallContext{AccountAddress: Address{0xff}}
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

	s1.CallData = append(s1.CallData, 1)
	s2.CallData = append(s2.CallData, 250)
	if s1.Eq(s2) {
		t.Fail()
	}
	s2.CallData = s1.CallData

	s1.LastCallReturnData = append(s1.LastCallReturnData, 1)
	s2.LastCallReturnData = append(s2.LastCallReturnData, 250)
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
	s1.CallContext = CallContext{AccountAddress: Address{0x01}}
	s1.BlockContext = BlockContext{BlockNumber: 1}
	s1.CallData = append(s1.CallData, 1)
	s1.LastCallReturnData = append(s1.LastCallReturnData, 1)

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
	s2.CallContext = CallContext{AccountAddress: Address{0x01}}
	s2.BlockContext = BlockContext{BlockNumber: 1}
	s2.CallData = append(s2.CallData, 1)
	s2.LastCallReturnData = append(s2.LastCallReturnData, 1)

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
	s1.CallContext = CallContext{AccountAddress: Address{0xff}}
	s1.BlockContext = BlockContext{BlockNumber: 1}
	s1.CallData = append(s1.CallData, 1)
	s1.LastCallReturnData = append(s1.LastCallReturnData, 1)

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
	s2.CallContext = CallContext{AccountAddress: Address{0xef}}
	s2.BlockContext = BlockContext{BlockNumber: 251}
	s2.CallData = append(s2.CallData, 250)
	s2.LastCallReturnData = append(s2.LastCallReturnData, 249)

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

func TestState_EqReturnDataCompare(t *testing.T) {
	dataValue := make([]byte, 1)
	dataValue[0]++
	tests := map[string]struct {
		status StatusCode
		wanted bool
	}{
		"stopped":  {Stopped, false},
		"reverted": {Reverted, false},
		"running":  {Running, true},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			s1 := getNewFilledState()
			s1.ReturnData = dataValue
			s2 := s1.Clone()
			s2.ReturnData[0]++
			s1.Status = test.status
			s2.Status = test.status
			if s1.Eq(s2) != test.wanted {
				t.Error("Compared return data when not stopped")
			}
		})
	}
}
