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
	"os"
	"path"
	"testing"

	. "github.com/Fantom-foundation/Tosca/go/ct/common"
	"github.com/Fantom-foundation/Tosca/go/tosca"
	"github.com/Fantom-foundation/Tosca/go/tosca/vm"
)

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
	serializableState.Revision = tosca.R09_Berlin
	serializableState.ReadOnly = false
	serializableState.Pc = 42
	serializableState.Gas = 77
	serializableState.GasRefund = 88
	serializableState.Code.ToBytes()[0] = byte(vm.INVALID)
	serializableState.Stack[0] = NewU256(77)
	serializableState.Memory = NewBytes([]byte{42})
	serializableState.Storage.Current[NewU256(42)] = NewU256(4)
	serializableState.Storage.Original[NewU256(77)] = NewU256(7)
	serializableState.Storage.Warm[NewU256(9)] = false
	serializableState.Accounts.Balance[tosca.Address{0x01}] = NewU256(77)
	serializableState.Accounts.Code[tosca.Address{0x01}] = NewBytes([]byte{byte(vm.INVALID)})
	delete(serializableState.Accounts.Warm, tosca.Address{0x02})
	serializableState.Logs.Entries[0].Data = NewBytes([]byte{99})
	serializableState.Logs.Entries[0].Topics[0] = NewU256(42)
	serializableState.CallContext.AccountAddress = tosca.Address{0x02}
	serializableState.BlockContext.BlockNumber = 42
	serializableState.CallData = NewBytes([]byte{4})
	serializableState.LastCallReturnData = NewBytes([]byte{6})
	serializableState.HasSelfDestructed = false
	serializableState.SelfDestructedJournal = newSerializableJournal([]SelfDestructEntry{})
	serializableState.RecentBlockHashes = NewImmutableHashArray(tosca.Hash{0x01})

	ok := s.Status == Running &&
		s.Revision == tosca.R10_London &&
		s.ReadOnly &&
		s.Pc == 3 &&
		s.Gas == 42 &&
		s.GasRefund == 63 &&
		s.Code.Length() == 5 &&
		s.Code.code[0] == byte(vm.PUSH2) &&
		s.Stack.Size() == 1 &&
		s.Stack.Get(0).Uint64() == 42 &&
		s.Memory.Size() == 64 &&
		s.Memory.mem[31] == 1 &&
		s.Storage.GetCurrent(NewU256(42)).Eq(NewU256(7)) &&
		s.Storage.GetCurrent(NewU256(7)).IsZero() &&
		s.Storage.GetOriginal(NewU256(77)).Eq(NewU256(4)) &&
		s.Storage.IsWarm(NewU256(9)) &&
		s.Accounts.GetBalance(tosca.Address{0x01}).Eq(NewU256(42)) &&
		s.Accounts.GetCode(tosca.Address{0x01}) == NewBytes([]byte{byte(vm.PUSH1), byte(6)}) &&
		s.Accounts.IsWarm(tosca.Address{0x02}) &&
		s.Logs.Entries[0].Data[0] == 4 &&
		s.Logs.Entries[0].Topics[0] == NewU256(21) &&
		s.CallContext.AccountAddress == tosca.Address{0x01} &&
		s.BlockContext.BlockNumber == 1 &&
		s.CallData.Length() == 1 &&
		s.CallData.Get(0, 1)[0] == 1 &&
		s.LastCallReturnData.Length() == 1 &&
		s.LastCallReturnData.Get(0, 1)[0] == 1 &&
		s.HasSelfDestructed &&
		len(s.SelfDestructedJournal) == 1 &&
		s.SelfDestructedJournal[0] == SelfDestructEntry{tosca.Address{1}, tosca.Address{2}} &&
		s.RecentBlockHashes.Equal(NewImmutableHashArray(tosca.Hash{0x01}))

	if !ok {
		t.Errorf("new serializable state is not independent")
	}
}

func TestSerialization_DeserializedStateIsIndependent(t *testing.T) {
	s := newStateSerializableFromState(getNewFilledState())

	deserializedState := s.deserialize()
	deserializedState.Status = Stopped
	deserializedState.Revision = tosca.R09_Berlin
	deserializedState.ReadOnly = false
	deserializedState.Pc = 42
	deserializedState.Gas = 77
	deserializedState.GasRefund = 88
	deserializedState.Code.code[0] = byte(vm.INVALID)
	deserializedState.Stack.stack[0] = NewU256(77)
	deserializedState.Memory.mem[0] = 42
	deserializedState.Storage.current[NewU256(42)] = NewU256(4)
	deserializedState.Storage.original[NewU256(77)] = NewU256(7)
	deserializedState.Storage.warm[NewU256(9)] = false
	deserializedState.Accounts.SetBalance(tosca.Address{0x01}, NewU256(77))
	delete(deserializedState.Accounts.warm, tosca.Address{0x02})
	deserializedState.Logs.Entries[0].Data[0] = 99
	deserializedState.Logs.Entries[0].Topics[0] = NewU256(42)
	deserializedState.CallContext.AccountAddress = tosca.Address{0x02}
	deserializedState.BlockContext.BlockNumber = 42
	deserializedState.CallData = NewBytes([]byte{4})
	deserializedState.LastCallReturnData = NewBytes([]byte{6})
	deserializedState.HasSelfDestructed = false
	deserializedState.SelfDestructedJournal = []SelfDestructEntry{}
	deserializedState.RecentBlockHashes = NewImmutableHashArray(tosca.Hash{0x02})

	ok := s.Status == Running &&
		s.Revision == tosca.R10_London &&
		s.ReadOnly &&
		s.Pc == 3 &&
		s.Gas == 42 &&
		s.GasRefund == 63 &&
		s.Code.Length() == 5 &&
		s.Code.ToBytes()[0] == byte(vm.PUSH2) &&
		len(s.Stack) == 1 &&
		s.Stack[0].Uint64() == 42 &&
		s.Memory.Length() == 64 &&
		s.Memory.ToBytes()[31] == 1 &&
		s.Storage.Current[NewU256(42)].Eq(NewU256(7)) &&
		s.Storage.Current[NewU256(7)].IsZero() &&
		s.Storage.Original[NewU256(77)].Eq(NewU256(4)) &&
		s.Storage.Warm[NewU256(9)] == true &&
		s.Accounts.Balance[tosca.Address{0x01}].Eq(NewU256(42)) &&
		s.Accounts.Code[tosca.Address{0x01}] == NewBytes([]byte{byte(vm.PUSH1), byte(6)}) &&
		s.Accounts.Warm[tosca.Address{0x02}] == true &&
		s.Logs.Entries[0].Data.ToBytes()[0] == 4 &&
		s.Logs.Entries[0].Topics[0] == NewU256(21) &&
		s.CallContext.AccountAddress == tosca.Address{0x01} &&
		s.BlockContext.BlockNumber == 1 &&
		s.CallData.Length() == 1 &&
		s.CallData.ToBytes()[0] == 1 &&
		s.LastCallReturnData.Length() == 1 &&
		s.LastCallReturnData.ToBytes()[0] == 1 &&
		s.HasSelfDestructed &&
		len(s.SelfDestructedJournal) == 1 &&
		s.SelfDestructedJournal[0] == serializableSelfDestructEntry{tosca.Address{1}, tosca.Address{2}} &&
		s.RecentBlockHashes.Equal(NewImmutableHashArray(tosca.Hash{0x01}))

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
	if want, got := tosca.R10_London, state.Revision; want != got {
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

func TestSerialization_byteSliceMarshal(t *testing.T) {
	tests := []struct {
		slice    Bytes
		expected string
	}{
		{NewBytes([]byte{byte(0x01)}), "\"01\""},
		{NewBytes([]byte{byte(0xff)}), "\"ff\""},
		{NewBytes([]byte{byte(0x01), byte(0x02), byte(0x03), byte(0x04), byte(0x05), byte(0x06)}), "\"010203040506\""},
		{NewBytes([]byte{byte(0xfa), byte(0xfb), byte(0xfc), byte(0xfd), byte(0xfe), byte(0xff)}), "\"fafbfcfdfeff\""},
		{NewBytes([]byte{byte(0x01), byte(0x23), byte(0x45), byte(0x67), byte(0x89), byte(0xab)}), "\"0123456789ab\""},
	}

	for _, test := range tests {
		marshaled, err := test.slice.MarshalJSON()
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if !bytes.Equal(marshaled, []byte(test.expected)) {
			t.Errorf("Unexpected marshaled byte slice, wanted: %v vs got: %v", test.expected, marshaled)
		}
	}
}

func TestSerialization_byteSliceUnmarshal(t *testing.T) {
	tests := []struct {
		input    string
		expected Bytes
	}{
		{"\"01\"", NewBytes([]byte{byte(0x01)})},
		{"\"ff\"", NewBytes([]byte{byte(0xff)})},
		{"\"010203040506\"", NewBytes([]byte{byte(0x01), byte(0x02), byte(0x03), byte(0x04), byte(0x05), byte(0x06)})},
		{"\"fafbfcfdfeff\"", NewBytes([]byte{byte(0xfa), byte(0xfb), byte(0xfc), byte(0xfd), byte(0xfe), byte(0xff)})},
		{"\"0123456789ab\"", NewBytes([]byte{byte(0x01), byte(0x23), byte(0x45), byte(0x67), byte(0x89), byte(0xab)})},
	}

	for _, test := range tests {
		var slice Bytes
		err := slice.UnmarshalJSON([]byte(test.input))
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if slice != test.expected {
			t.Errorf("Unexpected unmarshaled byte slice, wanted: %v vs got: %v", test.expected, slice)
		}
	}
}

func TestSerialization_byteSliceUnmarshalError(t *testing.T) {
	inputs := []string{"error", "-1", "ABCDEFG"}
	for _, input := range inputs {
		var slice Bytes
		err := slice.UnmarshalJSON([]byte(input))
		if err == nil {
			t.Errorf("Expected error but got: %v", slice)
		}
	}
}
