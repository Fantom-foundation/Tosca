package vm

//go:generate mockgen -source processor.go -destination processor_mock.go -package vm

type Processor interface {
	Run(BlockInfo, Transaction, WorldState) (Receipt, error)
}

type BlockInfo struct {
	BlockNumber int64
	Timestamp   int64
	GasPrice    Value
	Coinbase    Address
	GasLimit    Gas
	PrevRandao  Hash
	ChainID     Word
	BaseFee     Value
	BlobBaseFee Value
	Revision    Revision
}

type Transaction struct {
	Sender     Address
	Recipient  *Address
	Nonce      uint64
	Input      []byte
	Value      Value
	GasLimit   Gas
	AccessList []AccessTuple
}

type WorldState interface {
	RunContext

	GetNonce(Address) uint64
	SetNonce(Address, uint64)
	SetBalance(Address, Value)
	SetCode(Address, []byte)

	CreateSnapshot() int
	RestoreSnapshot(int)
}

type AccessTuple struct {
	Address Address
	Keys    []Key
}

type Receipt struct {
	Success         bool     // false if the execution ended in a revert, true otherwise
	Output          []byte   // the output produced by the transaction
	ContractAddress *Address // filled if a contract was created by this transaction
	GasUsed         Gas      // gas used by contract calls
	BlobGasUsed     Gas      // gas used for blob transactions
}
