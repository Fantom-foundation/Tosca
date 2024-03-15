package st

import (
	"bytes"
	"os"
	"path"
	"testing"

	. "github.com/Fantom-foundation/Tosca/go/ct/common"
)

////////////////////////////////////////////////////////////
// Helper functions

func getNewFilledState() *State {
	s := NewState(NewCode([]byte{byte(PUSH2), 7, 4, byte(ADD), byte(STOP)}))
	s.Status = Running
	s.Revision = R10_London
	s.ReadOnly = true
	s.Pc = 3
	s.Gas = 42
	s.GasRefund = 63
	s.Stack.Push(NewU256(42))
	s.Memory.Write([]byte{1, 2, 3}, 31)
	s.Storage.Current[NewU256(42)] = NewU256(7)
	s.Storage.Original[NewU256(77)] = NewU256(4)
	s.Storage.MarkWarm(NewU256(9))
	s.Accounts = NewAccounts()
	s.Accounts.Balance[Address{0x01}] = NewU256(42)
	s.Accounts.Code[Address{0x01}] = []byte{byte(PUSH1), byte(6)}
	s.Accounts.MarkWarm(Address{0x02})
	s.Logs.AddLog([]byte{4, 5, 6}, NewU256(21), NewU256(22))
	s.CallContext = CallContext{AccountAddress: Address{0x01}}
	s.BlockContext = BlockContext{BlockNumber: 1}
	s.CallData = []byte{1}
	s.LastCallReturnData = []byte{1}
	return s
}

////////////////////////////////////////////////////////////
// Importing/exporting tests

const testFileName = "state_serialization_test.json"

func TestSerialization_ExportState(t *testing.T) {
	s := getNewFilledState()

	testFilePath := path.Join(t.TempDir(), testFileName)

	err := ExportStateJSON(s, testFilePath)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(testFilePath); err != nil {
		t.Error("file not found")
	}
}

func TestSerialization_ImportState(t *testing.T) {
	s := getNewFilledState()

	testFilePath := path.Join(t.TempDir(), testFileName)

	err := ExportStateJSON(s, testFilePath)
	if err != nil {
		t.Fatal(err)
	}

	s2, err := ImportStateJSON(testFilePath)
	if err != nil {
		t.Fatal(err)
	}

	if !s.Eq(s2) {
		diffs := s.Diff(s2)
		t.Error("invalid deserialization, differences found:")
		for _, diff := range diffs {
			t.Error(diff)
		}
	}
}

func TestSerialization_ImportStateFileNotFound(t *testing.T) {
	_, err := ImportStateJSON("nonexistent_file.json")
	if err == nil {
		t.Errorf("expected error, got nil")
	}
}

func TestSerialization_ImportStateInvalidFile(t *testing.T) {
	testFilePath := path.Join(t.TempDir(), "invalid_file.json")

	err := os.WriteFile(testFilePath, []byte("invalid json syntax"), 0644)
	if err != nil {
		t.Fatal(err)
	}

	_, err = ImportStateJSON(testFilePath)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

////////////////////////////////////////////////////////////
// Serialization tests

func TestSerialization_StateSerialization(t *testing.T) {
	s := getNewFilledState()

	serializableState := newStateSerializableFromState(s)
	_, err := serializableState.serialize()
	if err != nil {
		t.Fatal(err)
	}
}

func TestSerialization_StateDeserialization(t *testing.T) {
	s := getNewFilledState()

	serializableState := newStateSerializableFromState(s)
	serialized, err := serializableState.serialize()
	if err != nil {
		t.Fatal(err)
	}

	_, err = newStateSerializableFromSerialized(serialized)
	if err != nil {
		t.Fatal(err)
	}
}

func TestSerialization_SerializationRoundTrip(t *testing.T) {
	s := getNewFilledState()

	serializableState := newStateSerializableFromState(s)
	serialized, err := serializableState.serialize()
	if err != nil {
		t.Fatal(err)
	}

	deserializedState, err := newStateSerializableFromSerialized(serialized)
	if err != nil {
		t.Fatal(err)
	}

	state := deserializedState.deserialize()

	if !s.Eq(state) {
		diffs := s.Diff(state)
		t.Error("invalid deserialization, differences found:")
		for _, diff := range diffs {
			t.Error(diff)
		}
	}
}

func TestSerialization_NewStateSerializableIsIndependent(t *testing.T) {
	s := getNewFilledState()

	serializableState := newStateSerializableFromState(s)
	serializableState.Status = Stopped
	serializableState.Revision = R09_Berlin
	serializableState.ReadOnly = false
	serializableState.Pc = 42
	serializableState.Gas = 77
	serializableState.GasRefund = 88
	serializableState.Code[0] = byte(INVALID)
	serializableState.Stack[0] = NewU256(77)
	serializableState.Memory[0] = 42
	serializableState.Storage.Current[NewU256(42)] = NewU256(4)
	serializableState.Storage.Original[NewU256(77)] = NewU256(7)
	serializableState.Storage.Warm[NewU256(9)] = false
	serializableState.Accounts.Balance[Address{0x01}] = NewU256(77)
	serializableState.Accounts.Code[Address{0x01}] = []byte{byte(INVALID)}
	delete(serializableState.Accounts.Warm, Address{0x02})
	serializableState.Logs.Entries[0].Data[0] = 99
	serializableState.Logs.Entries[0].Topics[0] = NewU256(42)
	serializableState.CallContext.AccountAddress = Address{0x02}
	serializableState.BlockContext.BlockNumber = 42
	serializableState.CallData = []byte{4}
	serializableState.LastCallReturnData = []byte{6}

	ok := s.Status == Running &&
		s.Revision == R10_London &&
		s.ReadOnly &&
		s.Pc == 3 &&
		s.Gas == 42 &&
		s.GasRefund == 63 &&
		s.Code.Length() == 5 &&
		s.Code.code[0] == byte(PUSH2) &&
		s.Stack.Size() == 1 &&
		s.Stack.Get(0).Uint64() == 42 &&
		s.Memory.Size() == 64 &&
		s.Memory.mem[31] == 1 &&
		s.Storage.Current[NewU256(42)].Eq(NewU256(7)) &&
		s.Storage.Current[NewU256(7)].IsZero() &&
		s.Storage.Original[NewU256(77)].Eq(NewU256(4)) &&
		s.Storage.IsWarm(NewU256(9)) &&
		s.Accounts.Balance[Address{0x01}].Eq(NewU256(42)) &&
		bytes.Equal(s.Accounts.Code[Address{0x01}], []byte{byte(PUSH1), byte(6)}) &&
		s.Accounts.IsWarm(Address{0x02}) &&
		s.Logs.Entries[0].Data[0] == 4 &&
		s.Logs.Entries[0].Topics[0] == NewU256(21) &&
		s.CallContext.AccountAddress == Address{0x01} &&
		s.BlockContext.BlockNumber == 1 &&
		len(s.CallData) == 1 &&
		s.CallData[0] == 1 &&
		len(s.LastCallReturnData) == 1 &&
		s.LastCallReturnData[0] == 1
	if !ok {
		t.Errorf("new serializable state is not independent")
	}
}

func TestSerialization_DeserializedStateIsIndependent(t *testing.T) {
	s := newStateSerializableFromState(getNewFilledState())

	deserializedState := s.deserialize()
	deserializedState.Status = Stopped
	deserializedState.Revision = R09_Berlin
	deserializedState.ReadOnly = false
	deserializedState.Pc = 42
	deserializedState.Gas = 77
	deserializedState.GasRefund = 88
	deserializedState.Code.code[0] = byte(INVALID)
	deserializedState.Stack.stack[0] = NewU256(77)
	deserializedState.Memory.mem[0] = 42
	deserializedState.Storage.Current[NewU256(42)] = NewU256(4)
	deserializedState.Storage.Original[NewU256(77)] = NewU256(7)
	deserializedState.Storage.warm[NewU256(9)] = false
	deserializedState.Accounts.Balance[Address{0x01}] = NewU256(77)
	deserializedState.Accounts.Code[Address{0x01}] = []byte{byte(INVALID)}
	delete(deserializedState.Accounts.warm, Address{0x02})
	deserializedState.Logs.Entries[0].Data[0] = 99
	deserializedState.Logs.Entries[0].Topics[0] = NewU256(42)
	deserializedState.CallContext.AccountAddress = Address{0x02}
	deserializedState.BlockContext.BlockNumber = 42
	deserializedState.CallData = []byte{4}
	deserializedState.LastCallReturnData = []byte{6}

	ok := s.Status == Running &&
		s.Revision == R10_London &&
		s.ReadOnly &&
		s.Pc == 3 &&
		s.Gas == 42 &&
		s.GasRefund == 63 &&
		len(s.Code) == 5 &&
		s.Code[0] == byte(PUSH2) &&
		len(s.Stack) == 1 &&
		s.Stack[0].Uint64() == 42 &&
		len(s.Memory) == 64 &&
		s.Memory[31] == 1 &&
		s.Storage.Current[NewU256(42)].Eq(NewU256(7)) &&
		s.Storage.Current[NewU256(7)].IsZero() &&
		s.Storage.Original[NewU256(77)].Eq(NewU256(4)) &&
		s.Storage.Warm[NewU256(9)] == true &&
		s.Accounts.Balance[Address{0x01}].Eq(NewU256(42)) &&
		bytes.Equal(s.Accounts.Code[Address{0x01}], []byte{byte(PUSH1), byte(6)}) &&
		s.Accounts.Warm[Address{0x02}] == true &&
		s.Logs.Entries[0].Data[0] == 4 &&
		s.Logs.Entries[0].Topics[0] == NewU256(21) &&
		s.CallContext.AccountAddress == Address{0x01} &&
		s.BlockContext.BlockNumber == 1 &&
		len(s.CallData) == 1 &&
		s.CallData[0] == 1 &&
		len(s.LastCallReturnData) == 1 &&
		s.LastCallReturnData[0] == 1
	if !ok {
		t.Errorf("deserialized state is not independent")
	}
}

func TestSerialization_EmptyState(t *testing.T) {
	inputState := NewState(NewCode([]byte{}))

	serializableState := newStateSerializableFromState(inputState)
	_, err := serializableState.serialize()
	if err != nil {
		t.Fatal(err)
	}
}

func TestSerialization_IncompleteSerializedData(t *testing.T) {
	serialized := []byte(`
	{
		"Status": "running",
		"Revision": "London",
		"Pc": 3
	}
	`)
	serializableState, err := newStateSerializableFromSerialized(serialized)
	if err != nil {
		t.Error(err)
	}

	state := serializableState.deserialize()

	if want, got := Running, state.Status; want != got {
		t.Errorf("invalid deserialization of Status, want: %v, got: %v", want, got)
	}
	if want, got := R10_London, state.Revision; want != got {
		t.Errorf("invalid deserialization of Revision, want: %v, got: %v", want, got)
	}
	if want, got := uint16(3), state.Pc; want != got {
		t.Errorf("invalid deserialization of Pc, want: %v, got: %v", want, got)
	}
}

func TestSerialization_UnknownSerializedData(t *testing.T) {
	serialized := []byte(`
	{
		"UnknownField": 42
	}
	`)
	_, err := newStateSerializableFromSerialized(serialized)
	if err == nil {
		t.Errorf("expected error, got nil")
	}
}

func TestSerialization_InvalidData(t *testing.T) {
	serialized := []byte(`
	{
		invalid json syntax
	}
	`)
	_, err := newStateSerializableFromSerialized(serialized)
	if err == nil {
		t.Errorf("expected error, got nil")
	}
}
