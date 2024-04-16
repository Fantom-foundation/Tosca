//
// Copyright (c) 2024 Fantom Foundation
//
// Use of this software is governed by the Business Source License included
// in the LICENSE.TXT file and at fantom.foundation/bsl11.
//
// Change Date: 2028-4-16
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by the GNU Lesser General Public Licence v3 
//

package st

import (
	"bytes"
	"encoding/json"
	"maps"
	"os"
	"slices"

	. "github.com/Fantom-foundation/Tosca/go/ct/common"
	"github.com/Fantom-foundation/Tosca/go/vm"
)

////////////////////////////////////////////////////////////
// Importing/exporting state

// ExportStateJSON exports the given state in json format to the given file path.
// If the file does not exist, it will be created.
// If the file already exists, it will be overwritten.
func ExportStateJSON(state *State, filePath string) error {
	serializableState := newStateSerializableFromState(state)
	serialized, err := serializableState.serialize()
	if err != nil {
		return err
	}
	return os.WriteFile(filePath, serialized, 0644)
}

// ImportStateJSON imports a state from the given json file.
// If the file does not exist, or is not parsable, the import fails.
func ImportStateJSON(filePath string) (*State, error) {
	serialized, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}
	serializableState, err := newStateSerializableFromSerialized(serialized)
	if err != nil {
		return nil, err
	}
	return serializableState.deserialize(), nil
}

////////////////////////////////////////////////////////////
// Serialization helpers

// stateSerializable is a serializable representation of the State struct.
// It can be used to serialize and deserialize a State struct.
type stateSerializable struct {
	Status             StatusCode
	Revision           Revision
	ReadOnly           bool
	Pc                 uint16
	Gas                vm.Gas
	GasRefund          vm.Gas
	Code               Bytes
	Stack              []U256
	Memory             Bytes
	Storage            *storageSerializable
	Accounts           *accountsSerializable
	Logs               *logsSerializable
	CallContext        CallContext
	BlockContext       BlockContext
	CallData           Bytes
	LastCallReturnData Bytes
	ReturnData         Bytes
	CallJournal        *CallJournal
}

// storageSerializable is a serializable representation of the Storage struct.
type storageSerializable struct {
	Current  map[U256]U256
	Original map[U256]U256
	Warm     map[U256]bool
}

// accountsSerializable is a serializable representation of the Accounts struct.
type accountsSerializable struct {
	Balance map[vm.Address]U256
	Code    map[vm.Address]Bytes
	Warm    map[vm.Address]bool
}

// logsSerializable is a serializable representation of the Log.
type logsSerializable struct {
	Entries []logEntrySerializable
}

type logEntrySerializable struct {
	Topics []U256
	Data   Bytes
}

func newLogsSerializable(logs *Logs) *logsSerializable {
	serializable := &logsSerializable{}
	for _, entry := range logs.Entries {
		serializable.addLog(NewBytes(entry.Data), entry.Topics...)
	}
	return serializable
}

func (l *logsSerializable) addLog(data Bytes, topics ...U256) {
	l.Entries = append(l.Entries, logEntrySerializable{
		slices.Clone(topics),
		data,
	})
}

// newStateSerializableFromState creates a new stateSerializable instance from the given State instance.
// The data of the input state is deep copied.
func newStateSerializableFromState(state *State) *stateSerializable {
	return &stateSerializable{
		Status:             state.Status,
		Revision:           state.Revision,
		ReadOnly:           state.ReadOnly,
		Pc:                 state.Pc,
		Gas:                state.Gas,
		GasRefund:          state.GasRefund,
		Code:               NewBytes(state.Code.code),
		Stack:              slices.Clone(state.Stack.stack),
		Memory:             NewBytes(state.Memory.mem),
		Storage:            newStorageSerializable(state.Storage),
		Accounts:           newAccountsSerializable(state.Accounts),
		Logs:               newLogsSerializable(state.Logs),
		CallContext:        state.CallContext,
		BlockContext:       state.BlockContext,
		CallData:           state.CallData,
		LastCallReturnData: state.LastCallReturnData,
		ReturnData:         state.ReturnData,
		CallJournal:        state.CallJournal,
	}
}

// newStateSerializableFromSerialized creates a new stateSerializable instance from the given serialized data.
func newStateSerializableFromSerialized(data []byte) (*stateSerializable, error) {
	serializableState := &stateSerializable{}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	err := decoder.Decode(serializableState)
	return serializableState, err
}

// serialize serializes the stateSerializable instance.
func (s *stateSerializable) serialize() ([]byte, error) {
	return json.MarshalIndent(s, "", "  ")
}

// deserialize converts the stateSerializable to a State instance.
// The data of the stateSerializable is deep copied.
func (s *stateSerializable) deserialize() *State {
	state := NewState(NewCode(s.Code.ToBytes()))
	state.Status = s.Status
	state.Revision = s.Revision
	state.ReadOnly = s.ReadOnly
	state.Pc = s.Pc
	state.Gas = s.Gas
	state.GasRefund = s.GasRefund
	state.Stack = NewStack(slices.Clone(s.Stack)...)
	state.Memory = NewMemory(s.Memory.ToBytes()...)

	if s.Storage != nil {
		storageBuilder := NewStorageBuilder()
		for key, val := range s.Storage.Current {
			storageBuilder.SetCurrent(key, val)
		}

		for key, val := range s.Storage.Original {
			storageBuilder.SetOriginal(key, val)
		}

		for key, val := range s.Storage.Warm {
			storageBuilder.SetWarm(key, val)
		}
		state.Storage = storageBuilder.Build()
	}

	if s.Accounts != nil {
		accountsBuilder := NewAccountsBuilder()

		for address, value := range s.Accounts.Balance {
			accountsBuilder.SetBalance(address, value)
		}

		// Code needs to be manually copied because of serializablebytes
		for address, code := range s.Accounts.Code {
			accountsBuilder.SetCode(address, code)
		}

		for key := range s.Accounts.Warm {
			accountsBuilder.SetWarm(key)
		}

		state.Accounts = accountsBuilder.Build()
	}
	if s.Logs != nil {
		state.Logs = NewLogs()
		for _, entry := range s.Logs.Entries {
			state.Logs.AddLog(entry.Data.ToBytes(), entry.Topics...)
		}
	}
	state.CallContext = s.CallContext
	state.BlockContext = s.BlockContext
	state.CallData = s.CallData
	state.LastCallReturnData = s.LastCallReturnData
	state.ReturnData = s.ReturnData
	if s.CallJournal != nil {
		state.CallJournal = s.CallJournal.Clone()
	}
	return state
}

// newStorageSerializable creates a new storageSerializable instance from the given Storage instance.
func newStorageSerializable(storage *Storage) *storageSerializable {
	return &storageSerializable{
		Current:  maps.Clone(storage.current),
		Original: maps.Clone(storage.original),
		Warm:     maps.Clone(storage.warm),
	}
}

// newAccountsSerializable creates a new balanceSerializable instance from the given Balance instance.
func newAccountsSerializable(accounts *Accounts) *accountsSerializable {
	warm := make(map[vm.Address]bool)
	for key := range accounts.warm {
		warm[key] = true
	}

	codes := make(map[vm.Address]Bytes)
	for address, code := range accounts.code {
		codes[address] = code
	}

	return &accountsSerializable{
		Balance: maps.Clone(accounts.balance),
		Code:    codes,
		Warm:    warm,
	}
}
