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

package vm

import "fmt"

//go:generate mockgen -source interpreter.go -destination interpreter_mock.go -package vm

// Interpreter is a component capable of executing EVM byte-code. It is the main
// part of an EVM implementation, though a full EVM adds the ability to handle
// recursive contract calls and transaction handling.
// To obtain an Interpreter instance, client code should use GetInterpreter() provided
// by the registry file in this package.
type Interpreter interface {
	// Run executes the code provided by the parameters in the specified context
	// and returns the processing result. The resulting error is nil whenever the
	// code was correctly executed (even if the execution was aborted due do to
	// a code-internal issue). The error is not nil if some problem within the
	// interpreter caused the execution to fail to correctly process the provided
	// program. In such a case the result is undefined. During a call with an
	// unsupported Revision an ErrUnsupportedRevision Error is returned.
	// Interpreters are required to be thread-safe. Thus, multiple runs may be
	// conducted in parallel.
	Run(Parameters) (Result, error)
}

// Parameters summarizes the list of input parameters required for executing code.
type Parameters struct {
	Context   RunContext
	Revision  Revision
	Kind      CallKind
	Static    bool
	Depth     int
	Gas       Gas
	Recipient Address
	Sender    Address
	Input     []byte
	Value     Value
	CodeHash  *Hash
	Code      []byte
}

// Result summarizes the result of a EVM code computation.
type Result struct {
	Success   bool // false if the execution ended in a revert, true otherwise
	Output    []byte
	GasLeft   Gas
	GasRefund Gas
}

// RunContext provides an interface to access and manipulate state and transaction
// properties as needed by individual EVM instructions.
type RunContext interface {
	AccountExists(addr Address) bool
	GetStorage(addr Address, key Key) Word
	SetStorage(addr Address, key Key, value Word) StorageStatus
	GetBalance(addr Address) Value
	GetCodeSize(addr Address) int
	GetCodeHash(addr Address) Hash
	GetCode(addr Address) []byte
	GetTransactionContext() TransactionContext
	GetBlockHash(number int64) Hash
	EmitLog(addr Address, topics []Hash, data []byte)
	Call(kind CallKind, parameter CallParameter) (CallResult, error)
	SelfDestruct(addr Address, beneficiary Address) bool
	AccessAccount(addr Address) AccessStatus
	AccessStorage(addr Address, key Key) AccessStatus

	// -- legacy API needed by LFVM and Geth, to be removed in the future ---

	// Deprecated: should not be needed when using result of SetStorage(..)
	GetCommittedStorage(addr Address, key Key) Word
	// Deprecated: should not be needed when using result of SetStorage(..)
	IsAddressInAccessList(addr Address) bool
	// Deprecated: should not be needed when using result of SetStorage(..)
	IsSlotInAccessList(addr Address, key Key) (addressPresent, slotPresent bool)
	// Deprecated: should not be needed
	HasSelfDestructed(addr Address) bool
}

// Gas represents the type used to represent the Gas values.
type Gas int64

// Address represents the 160-bit (20 bytes) address of an account.
type Address [20]byte

// Key represents the 256-bit (32 bytes) key of a storage slot.
type Key [32]byte

// Word represents an arbitrary 256-bit (32 byte) word in the EVM.
type Word [32]byte

// Value represents the 256-bit (32 bytes) value of a storage slot.
type Value [32]byte

// Hash represents the 256-bit (32 bytes) hash of a code, a block, a topic
// or similar sequence of cryptographic summary information.
type Hash [32]byte

// TransactionContext contains information about current transaction and block.
type TransactionContext struct {
	GasPrice    Value
	Origin      Address
	Coinbase    Address
	BlockNumber int64
	Timestamp   int64
	GasLimit    Gas
	PrevRandao  Hash
	ChainID     Word
	BaseFee     Value
	BlobBaseFee Value
}

// AccessStatus is an enum utilized to indicate cold and warm account or
// storage slot accesses.
type AccessStatus bool

const (
	ColdAccess AccessStatus = false
	WarmAccess AccessStatus = true
)

// StorageStatus is an enum utilized to indicate the effect of a storage
// slot update on the respective slot in the context of the current
// transaction. It is needed to perform proper gas price calculations of
// SSTORE operations.
type StorageStatus int

// See t.ly/b5HPf for the definition of these values.
const (
	StorageAssigned StorageStatus = iota
	StorageAdded
	StorageDeleted
	StorageModified
	StorageDeletedAdded
	StorageModifiedDeleted
	StorageDeletedRestored
	StorageAddedDeleted
	StorageModifiedRestored
)

// CallKind is an enum enabling the differentiation of the different types
// of recursive contract calls supported in the EVM.
type CallKind int

const (
	Call CallKind = iota
	DelegateCall
	StaticCall
	CallCode
	Create
	Create2
)

type CallParameter struct {
	Sender      Address // TODO: remove and handle implicit
	Recipient   Address // < not relevant for CREATE and CREATE2 // TODO: remove and handle implicit
	Value       Value   // < ignored by static calls, considered to be 0
	Input       []byte
	Gas         Gas
	Salt        Hash    // < only relevant for CREATE2 calls
	CodeAddress Address // < only relevant for DELEGATECALL and CALLCODE calls
}

type CallResult struct {
	Output         []byte
	GasLeft        Gas
	GasRefund      Gas
	CreatedAddress Address // < only meaningful for CREATE and CREATE2
	Success        bool    // false if the execution ended in a revert, true otherwise
}

// Revision is an enumeration for EVM specification revisions (aka. Hard-Forks).
type Revision int

// The list of revisions supported so far by Tosca.
const (
	R07_Istanbul Revision = iota
	R09_Berlin
	R10_London
	R11_Paris
	R12_Shanghai
	R13_Cancun
)

// Error for runs with unsupported Revision
type ErrUnsupportedRevision struct {
	Revision Revision
}

func (e *ErrUnsupportedRevision) Error() string {
	return fmt.Sprintf("Unsupported revision %d", e.Revision)
}

// ProfilingInterpreter is an optional extension to the Interpreter interface
// above which may be implemented by interpreters collecting statistical data
// on their executions.
type ProfilingInterpreter interface {
	Interpreter

	// ResetProfile resets the operation statistic collected by the underlying
	// Interpreter implementation. Use this, for instance, at the beginning of
	// a benchmark. It should not be called while running operations on the
	// Interpreter in parallel.
	ResetProfile()

	// DumpProfile prints a snapshot of the profiling data collected since the
	// last reset to stdout. In the future this interface will be changed to
	// return the result instead of printing it.
	// TODO: produce the result as a string
	DumpProfile()
}
