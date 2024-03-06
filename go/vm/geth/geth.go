package geth

import (
	"fmt"
	"math/big"

	"github.com/Fantom-foundation/Tosca/go/vm"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	geth "github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
)

func init() {
	vm.RegisterVirtualMachine("geth", &gethVm{})
}

type gethVm struct{}

func (m *gethVm) Run(parameters vm.Parameters) (vm.Result, error) {
	// Set hard forks for chainconfig
	chainConfig := params.AllEthashProtocolChanges
	chainConfig.ChainID = big.NewInt(0)
	chainConfig.IstanbulBlock = big.NewInt(int64(vm.R07_Istanbul) * 10)
	chainConfig.BerlinBlock = big.NewInt(int64(vm.R09_Berlin) * 10)
	chainConfig.LondonBlock = big.NewInt(int64(vm.R10_London) * 10)

	// Hashing function used in the context for BLOCKHASH instruction
	getHash := func(num uint64) common.Hash {
		return common.Hash{}
	}

	var transactionContext vm.TransactionContext
	if parameters.Context != nil {
		transactionContext = parameters.Context.GetTransactionContext()
	}

	// Create empty block context based on block number
	blockCtx := geth.BlockContext{
		BlockNumber: big.NewInt(int64(parameters.Revision)*10 + 2),
		Time:        big.NewInt(transactionContext.Timestamp),
		Difficulty:  big.NewInt(1),
		GasLimit:    uint64(transactionContext.GasLimit),
		GetHash:     getHash,
		BaseFee:     new(big.Int).SetBytes(transactionContext.BaseFee[:]),
		Transfer:    transferFunc,
		CanTransfer: canTransferFunc,
	}
	// Create empty tx context
	txCtx := geth.TxContext{
		GasPrice: new(big.Int).SetBytes(transactionContext.GasPrice[:]),
	}
	// Set interpreter variant for this VM
	config := geth.Config{
		InterpreterImpl: "geth",
	}

	stateDb := &stateDbAdapter{context: parameters.Context}
	evm := geth.NewEVM(blockCtx, txCtx, stateDb, chainConfig, config)

	addr := geth.AccountRef{}
	contract := geth.NewContract(addr, addr, big.NewInt(0), uint64(parameters.Gas))
	contract.CodeAddr = &common.Address{}
	contract.Code = parameters.Code
	contract.CodeHash = crypto.Keccak256Hash(parameters.Code)
	contract.CallerAddress = common.Address(parameters.Sender)

	output, err := evm.Interpreter().Run(contract, parameters.Input, false)

	result := vm.Result{
		Output:    output,
		GasLeft:   vm.Gas(contract.Gas),
		GasRefund: vm.Gas(stateDb.refund),
		Success:   true,
	}

	// If no error is reported, the execution ended with a STOP, RETURN, or SUICIDE.
	if err == nil {
		return result, nil
	}

	// In case of a revert the result should indicate an unsuccessful execution.
	if err == geth.ErrExecutionReverted {
		result.Success = false
		return result, nil
	}

	// In case of an issue caused by the code execution, the result should indicate
	// a failed execution but no error should be reported.
	switch err {
	case geth.ErrOutOfGas,
		geth.ErrCodeStoreOutOfGas,
		geth.ErrDepth,
		geth.ErrInsufficientBalance,
		geth.ErrContractAddressCollision,
		geth.ErrExecutionReverted,
		geth.ErrMaxCodeSizeExceeded,
		geth.ErrInvalidJump,
		geth.ErrWriteProtection,
		geth.ErrReturnDataOutOfBounds,
		geth.ErrGasUintOverflow,
		geth.ErrInvalidCode:
		return vm.Result{Success: false}, nil
	}
	if _, ok := err.(*geth.ErrStackOverflow); ok {
		return vm.Result{Success: false}, nil
	}
	if _, ok := err.(*geth.ErrStackUnderflow); ok {
		return vm.Result{Success: false}, nil
	}
	if _, ok := err.(*geth.ErrInvalidOpCode); ok {
		return vm.Result{Success: false}, nil
	}

	// In all other cases an EVM error should be reported.
	return vm.Result{}, fmt.Errorf("internal EVM error in geth: %v", err)
}

// --- Adapter ---

// transferFunc subtracts amount from sender and adds amount to recipient using the given Db
// Now is doing nothing as this is not changing gas computation
func transferFunc(stateDB geth.StateDB, callerAddress common.Address, to common.Address, value *big.Int) {
	// Can be something like this:
	// stateDB.SubBalance(callerAddress, value)
	// stateDB.AddBalance(to, value)
}

// canTransferFunc is the signature of a transfer function
func canTransferFunc(stateDB geth.StateDB, callerAddress common.Address, value *big.Int) bool {
	return stateDB.GetBalance(callerAddress).Cmp(value) >= 0
}

type stateDbAdapter struct {
	context vm.RunContext
	refund  uint64
}

func (s *stateDbAdapter) CreateAccount(common.Address) {
	panic("should not be needed for a single contract execution")
}

func (s *stateDbAdapter) SubBalance(common.Address, *big.Int) {
	panic("should not be needed for a single contract execution")
}

func (s *stateDbAdapter) AddBalance(common.Address, *big.Int) {
	panic("should not be needed for a single contract execution")
}

func (s *stateDbAdapter) GetBalance(addr common.Address) *big.Int {
	value := s.context.GetBalance(vm.Address(addr))
	return new(big.Int).SetBytes(value[:])
}

func (s *stateDbAdapter) GetNonce(common.Address) uint64 {
	panic("should not be needed for a single contract execution")
}

func (s *stateDbAdapter) SetNonce(common.Address, uint64) {
	panic("should not be needed for a single contract execution")
}

func (s *stateDbAdapter) GetCodeHash(addr common.Address) common.Hash {
	return common.Hash(s.context.GetCodeHash(vm.Address(addr)))
}

func (s *stateDbAdapter) GetCode(addr common.Address) []byte {
	return s.context.GetCode(vm.Address(addr))
}

func (s *stateDbAdapter) SetCode(common.Address, []byte) {
	panic("should not be needed for a single contract execution")
}

func (s *stateDbAdapter) GetCodeSize(addr common.Address) int {
	return s.context.GetCodeSize(vm.Address(addr))
}

func (s *stateDbAdapter) AddRefund(value uint64) {
	s.refund += value
}

func (s *stateDbAdapter) SubRefund(value uint64) {
	s.refund -= value
}

func (s *stateDbAdapter) GetRefund() uint64 {
	return s.refund
}

func (s *stateDbAdapter) GetCommittedState(addr common.Address, key common.Hash) common.Hash {
	return common.Hash(s.context.GetCommittedStorage(vm.Address(addr), vm.Key(key)))
}

func (s *stateDbAdapter) GetState(addr common.Address, key common.Hash) common.Hash {
	return common.Hash(s.context.GetStorage(vm.Address(addr), vm.Key(key)))
}

func (s *stateDbAdapter) SetState(addr common.Address, key common.Hash, value common.Hash) {
	s.context.SetStorage(vm.Address(addr), vm.Key(key), vm.Word(value))
}

func (s *stateDbAdapter) Suicide(addr common.Address) bool {
	panic("not implemented")
}

func (s *stateDbAdapter) HasSuicided(addr common.Address) bool {
	return s.context.HasSelfDestructed(vm.Address(addr))
}

func (s *stateDbAdapter) Exist(addr common.Address) bool {
	return s.context.AccountExists(vm.Address(addr))
}

func (s *stateDbAdapter) Empty(addr common.Address) bool {
	return s.context.AccountExists(vm.Address(addr))
}

func (s *stateDbAdapter) PrepareAccessList(sender common.Address, dest *common.Address, precompiles []common.Address, txAccesses types.AccessList) {
	panic("should not be needed for a single contract execution")
}

func (s *stateDbAdapter) AddressInAccessList(addr common.Address) bool {
	return s.context.IsAddressInAccessList(vm.Address(addr))
}

func (s *stateDbAdapter) SlotInAccessList(addr common.Address, slot common.Hash) (addressOk bool, slotOk bool) {
	return s.context.IsSlotInAccessList(vm.Address(addr), vm.Key(slot))
}

func (s *stateDbAdapter) AddAddressToAccessList(addr common.Address) {
	s.context.AccessAccount(vm.Address(addr))
}

func (s *stateDbAdapter) AddSlotToAccessList(addr common.Address, slot common.Hash) {
	s.context.AccessStorage(vm.Address(addr), vm.Key(slot))
}

func (s *stateDbAdapter) RevertToSnapshot(int) {
	panic("should not be needed for a single contract execution")
}

func (s *stateDbAdapter) Snapshot() int {
	panic("should not be needed for a single contract execution")
}

func (s *stateDbAdapter) AddLog(log *types.Log) {
	topics := make([]vm.Hash, 0, len(log.Topics))
	for _, cur := range log.Topics {
		topics = append(topics, vm.Hash(cur))
	}
	s.context.EmitLog(vm.Address(log.Address), topics, log.Data)
}

func (s *stateDbAdapter) AddPreimage(common.Hash, []byte) {
	panic("should not be needed for a single contract execution")
}

func (s *stateDbAdapter) ForEachStorage(common.Address, func(common.Hash, common.Hash) bool) error {
	panic("should not be needed for a single contract execution")
}
