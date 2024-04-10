package st

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
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
	Code               byteSliceSerializable
	Stack              []U256
	Memory             byteSliceSerializable
	Storage            *storageSerializable
	Accounts           *accountsSerializable
	Logs               *logsSerializable
	CallContext        CallContext
	BlockContext       BlockContext
	CallData           byteSliceSerializable
	LastCallReturnData byteSliceSerializable
	ReturnData         byteSliceSerializable
	CallJournal        *CallJournal
}

// byteSliceSerializable is a wrapper to achieve hex code output
type byteSliceSerializable []byte

// storageSerializable is a serializable representation of the Storage struct.
type storageSerializable struct {
	Current  map[U256]U256
	Original map[U256]U256
	Warm     map[U256]bool
}

// accountsSerializable is a serializable representation of the Accounts struct.
type accountsSerializable struct {
	Balance map[vm.Address]U256
	Code    map[vm.Address]byteSliceSerializable
	Warm    map[vm.Address]bool
}

// logsSerializable is a serializable representation of the Log.
type logsSerializable struct {
	Entries []logEntrySerializable
}

type logEntrySerializable struct {
	Topics []U256
	Data   byteSliceSerializable
}

func newLogsSerializable(logs *Logs) *logsSerializable {
	serializable := &logsSerializable{}
	for _, entry := range logs.Entries {
		serializable.addLog(entry.Data, entry.Topics...)
	}
	return serializable
}

func (l *logsSerializable) addLog(data byteSliceSerializable, topics ...U256) {
	l.Entries = append(l.Entries, logEntrySerializable{
		slices.Clone(topics),
		slices.Clone(data),
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
		Code:               bytes.Clone(state.Code.code),
		Stack:              slices.Clone(state.Stack.stack),
		Memory:             bytes.Clone(state.Memory.mem),
		Storage:            newStorageSerializable(state.Storage),
		Accounts:           newAccountsSerializable(state.Accounts),
		Logs:               newLogsSerializable(state.Logs),
		CallContext:        state.CallContext,
		BlockContext:       state.BlockContext,
		CallData:           state.CallData.ToBytes(),
		LastCallReturnData: state.LastCallReturnData.ToBytes(),
		ReturnData:         state.ReturnData.ToBytes(),
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
	state := NewState(NewCode(s.Code))
	state.Status = s.Status
	state.Revision = s.Revision
	state.ReadOnly = s.ReadOnly
	state.Pc = s.Pc
	state.Gas = s.Gas
	state.GasRefund = s.GasRefund
	state.Stack = NewStack(slices.Clone(s.Stack)...)
	state.Memory = NewMemory(slices.Clone(s.Memory)...)

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
			accountsBuilder.SetCode(address, NewBytes(code))
		}

		for key := range s.Accounts.Warm {
			accountsBuilder.SetWarm(key)
		}

		state.Accounts = accountsBuilder.Build()
	}
	if s.Logs != nil {
		state.Logs = NewLogs()
		for _, entry := range s.Logs.Entries {
			state.Logs.AddLog(entry.Data, entry.Topics...)
		}
	}
	state.CallContext = s.CallContext
	state.BlockContext = s.BlockContext
	state.CallData = NewBytes(s.CallData)
	state.LastCallReturnData = NewBytes(s.LastCallReturnData)
	state.ReturnData = NewBytes(s.ReturnData)
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

	codes := make(map[vm.Address]byteSliceSerializable)
	for address, code := range accounts.code {
		codes[address] = code.ToBytes()
	}

	return &accountsSerializable{
		Balance: maps.Clone(accounts.balance),
		Code:    codes,
		Warm:    warm,
	}
}

func (c byteSliceSerializable) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf("\"%x\"", c)), nil
}

func (c *byteSliceSerializable) UnmarshalJSON(data []byte) error {
	var s string
	err := json.Unmarshal(data, &s)
	if err != nil {
		return err
	}

	code, err := hex.DecodeString(s)
	if err != nil {
		return err
	}

	*c = code
	return nil
}
