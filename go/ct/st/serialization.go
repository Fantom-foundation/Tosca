package st

import (
	"bytes"
	"encoding/json"
	"os"
	"slices"

	. "github.com/Fantom-foundation/Tosca/go/ct/common"
	"golang.org/x/exp/maps"
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
	Status       StatusCode
	Revision     Revision
	ReadOnly     bool
	Pc           uint16
	Gas          uint64
	GasRefund    uint64
	Code         []byte
	Stack        []U256
	Memory       []byte
	Storage      *storageSerializable
	Logs         *Logs
	CallContext  CallContext
	BlockContext BlockContext
}

// storageSerializable is a serializable representation of the Storage struct.
type storageSerializable struct {
	Current  map[U256]U256
	Original map[U256]U256
	Warm     map[U256]bool
}

// newStateSerializableFromState creates a new stateSerializable instance from the given State instance.
// The data of the input state is deep copied.
func newStateSerializableFromState(state *State) *stateSerializable {
	return &stateSerializable{
		Status:       state.Status,
		Revision:     state.Revision,
		ReadOnly:     state.ReadOnly,
		Pc:           state.Pc,
		Gas:          state.Gas,
		GasRefund:    state.GasRefund,
		Code:         bytes.Clone(state.Code.code),
		Stack:        slices.Clone(state.Stack.stack),
		Memory:       bytes.Clone(state.Memory.mem),
		Storage:      newStorageSerializable(state.Storage),
		Logs:         state.Logs.Clone(),
		CallContext:  state.CallContext,
		BlockContext: state.BlockContext,
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
	return json.Marshal(s)
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
	state.Storage = NewStorage()
	if s.Storage != nil {
		state.Storage.Current = maps.Clone(s.Storage.Current)
		state.Storage.Original = maps.Clone(s.Storage.Original)
		state.Storage.warm = maps.Clone(s.Storage.Warm)
	}
	if s.Logs != nil {
		state.Logs = s.Logs.Clone()
	}
	state.CallContext = s.CallContext
	state.BlockContext = s.BlockContext
	return state
}

// newStorageSerializable creates a new storageSerializable instance from the given Storage instance.
func newStorageSerializable(storage *Storage) *storageSerializable {
	return &storageSerializable{
		Current:  maps.Clone(storage.Current),
		Original: maps.Clone(storage.Original),
		Warm:     maps.Clone(storage.warm),
	}
}
