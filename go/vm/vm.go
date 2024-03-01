package vm

// VirtualMachine (VM) represents an instance of an EVM-byte-code execution engine
// loaded in memory. A VM instance is capable of running multiple code executions
// in parallel.
// To obtain a VM instance, client could should use the GetVirtualMachine() provided
// by the registry file in this package.
type VirtualMachine interface {
	// Run executes the code provided by the parameters in the specified context
	// and returns the processing result. The resulting error is nil whenever the
	// code was correctly executed (even if the execution was aborted due do to
	// a discovered issue). The error is not nil if some problem within the VM
	// caused the VM to fail the correct execution. In this case the result is
	// undefined.
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
	Call(kind CallKind,
		recipient Address, sender Address, value Value, input []byte, gas Gas, depth int,
		static bool, salt Hash, codeAddress Address) (output []byte, gasLeft Gas, gasRefund Gas,
		createAddr Address, reverted bool, err error)
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
type Gas uint64

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
	GasPrice   Value
	Origin     Address
	Coinbase   Address
	BlockNumber     int64
	Timestamp  int64
	GasLimit   Gas
	PrevRandao Hash
	ChainID    Word
	BaseFee    Value
}

type AccessStatus bool

const (
	ColdAccess AccessStatus = false
	WarmAccess AccessStatus = true
)

type StorageStatus int

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

type CallKind int

const (
	Call CallKind = iota
	DelegateCall
	CallCode
	Create
	Create2
)

type Revision int

// The list of revisions supported so far by Tosca.
const (
	R07_Istanbul Revision = iota
	R09_Berlin
	R10_London
)

// ProfilingVM is an optional extension to the VirtualMachine interface above which
// may be implemented by VM implementations collecting statistical data regarding
// their execution.
type ProfilingVM interface {
	VirtualMachine

	// ResetProfile resets the operation statistic collected by the underlying VM implementation.
	// Use this, for instance, at the beginning of a benchmark. It should not be called while
	// running operations on the VM implementations in parallel.
	ResetProfile()

	// DumpProfile prints a snapshot of the profiling data collected since the last reset to stdout.
	// In the future this interface will be changed to return the result instead of printing it.
	DumpProfile()
}
