// Copyright (c) 2024 Fantom Foundation
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at fantom.foundation/bsl11.
//
// Change Date: 2028-4-16
//
// On the date above, in accordance with the Business Source License, use of
// this software will be governed by the GNU Lesser General Public License v3.

package geth

import (
	"errors"
	"fmt"
	"math/big"

	ct "github.com/Fantom-foundation/Tosca/go/ct/common"
	"github.com/Fantom-foundation/Tosca/go/tosca"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/stateless"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/types"
	geth "github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/trie/utils"
	"github.com/holiman/uint256"
)

func init() {
	tosca.MustRegisterInterpreterFactory("geth", func(any) (tosca.Interpreter, error) {
		return &gethVm{}, nil
	})
}

type gethVm struct{}

// Defines the newest supported revision for this interpreter implementation
const newestSupportedRevision = tosca.R13_Cancun

func (m *gethVm) Run(parameters tosca.Parameters) (tosca.Result, error) {
	if parameters.Revision > newestSupportedRevision {
		return tosca.Result{}, &tosca.ErrUnsupportedRevision{Revision: parameters.Revision}
	}
	evm, contract, stateDb := createGethInterpreterContext(parameters)

	output, err := evm.Interpreter().Run(contract, parameters.Input, false)

	result := tosca.Result{
		Output:    output,
		GasLeft:   tosca.Gas(contract.Gas),
		GasRefund: tosca.Gas(stateDb.refund),
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
	switch {
	case errors.Is(err, geth.ErrOutOfGas),
		errors.Is(err, geth.ErrCodeStoreOutOfGas),
		errors.Is(err, geth.ErrDepth),
		errors.Is(err, geth.ErrInsufficientBalance),
		errors.Is(err, geth.ErrContractAddressCollision),
		errors.Is(err, geth.ErrExecutionReverted),
		errors.Is(err, geth.ErrMaxCodeSizeExceeded),
		errors.Is(err, geth.ErrInvalidJump),
		errors.Is(err, geth.ErrWriteProtection),
		errors.Is(err, geth.ErrReturnDataOutOfBounds),
		errors.Is(err, geth.ErrReturnDataOutOfBounds),
		errors.Is(err, geth.ErrGasUintOverflow),
		errors.Is(err, geth.ErrInvalidCode):
		return tosca.Result{Success: false}, nil
	}

	if _, ok := err.(*geth.ErrStackOverflow); ok {
		return tosca.Result{Success: false}, nil
	}
	if _, ok := err.(*geth.ErrStackUnderflow); ok {
		return tosca.Result{Success: false}, nil
	}
	if _, ok := err.(*geth.ErrInvalidOpCode); ok {
		return tosca.Result{Success: false}, nil
	}

	// In all other cases an EVM error should be reported.
	return tosca.Result{}, fmt.Errorf("internal EVM error in geth: %v", err)
}

// MakeChainConfig returns a chain config for the given chain ID and target revision.
// The baseline config is used as a starting point, so that any prefilled configuration from go-ethereum:params/config.go can be used.
// chainId needs to be prefilled as it may be accessed with the opcode CHAINID.
// the fork-blocks and the fork-times are set to the respective values for the given revision.
func MakeChainConfig(baseline params.ChainConfig, chainId *big.Int, targetRevision tosca.Revision) params.ChainConfig {
	istanbulBlock := ct.GetForkBlock(tosca.R07_Istanbul)
	berlinBlock := ct.GetForkBlock(tosca.R09_Berlin)
	londonBlock := ct.GetForkBlock(tosca.R10_London)
	parisBlock := ct.GetForkBlock(tosca.R11_Paris)
	shanghaiTime := ct.GetForkTime(tosca.R12_Shanghai)
	cancunTime := ct.GetForkTime(tosca.R13_Cancun)

	chainConfig := baseline
	chainConfig.ChainID = chainId
	chainConfig.ByzantiumBlock = big.NewInt(0)
	chainConfig.IstanbulBlock = big.NewInt(0).SetUint64(istanbulBlock)
	chainConfig.BerlinBlock = big.NewInt(0).SetUint64(berlinBlock)
	chainConfig.LondonBlock = big.NewInt(0).SetUint64(londonBlock)

	if targetRevision >= tosca.R11_Paris {
		chainConfig.MergeNetsplitBlock = big.NewInt(0).SetUint64(parisBlock)
	}
	if targetRevision >= tosca.R12_Shanghai {
		chainConfig.ShanghaiTime = &shanghaiTime
	}
	if targetRevision >= tosca.R13_Cancun {
		chainConfig.CancunTime = &cancunTime
	}

	return chainConfig
}

func currentBlock(revision tosca.Revision) *big.Int {
	block := ct.GetForkBlock(revision)
	return big.NewInt(int64(block + 2))
}

func createGethInterpreterContext(parameters tosca.Parameters) (*geth.EVM, *geth.Contract, *stateDbAdapter) {
	// Set hard forks for chainconfig
	chainConfig :=
		MakeChainConfig(*params.AllEthashProtocolChanges,
			new(big.Int).SetBytes(parameters.ChainID[:]),
			parameters.Revision)

	// Hashing function used in the context for BLOCKHASH instruction
	getHash := func(num uint64) common.Hash {
		return common.Hash(parameters.Context.GetBlockHash(int64(num)))
	}

	// Create empty block context based on block number
	blockCtx := geth.BlockContext{
		BlockNumber: currentBlock(parameters.Revision),
		Time:        uint64(parameters.Timestamp),
		Difficulty:  big.NewInt(1),
		GasLimit:    uint64(parameters.GasLimit),
		GetHash:     getHash,
		BaseFee:     new(big.Int).SetBytes(parameters.BaseFee[:]),
		BlobBaseFee: new(big.Int).SetBytes(parameters.BlobBaseFee[:]),
		Transfer:    transferFunc,
		CanTransfer: canTransferFunc,
	}

	if parameters.Revision >= tosca.R11_Paris {
		// Setting the random signals to geth that a post-merge (Paris) revision should be utilized.
		hash := common.BytesToHash(parameters.PrevRandao[:])
		blockCtx.Random = &hash
	}

	// Create empty tx context
	txCtx := geth.TxContext{
		GasPrice:   new(big.Int).SetBytes(parameters.GasPrice[:]),
		BlobFeeCap: new(big.Int).SetBytes(parameters.BlobBaseFee[:]),
	}

	for _, hash := range parameters.BlobHashes {
		txCtx.BlobHashes = append(txCtx.BlobHashes, common.Hash(hash))
	}

	// Set interpreter variant for this VM
	config := geth.Config{}

	stateDb := &stateDbAdapter{context: parameters.Context}
	evm := geth.NewEVM(blockCtx, txCtx, stateDb, &chainConfig, config)

	evm.Origin = common.Address(parameters.Origin)
	evm.Context.BlockNumber = big.NewInt(parameters.BlockNumber)
	evm.Context.Coinbase = common.Address(parameters.Coinbase)
	evm.Context.Difficulty = new(big.Int).SetBytes(parameters.PrevRandao[:])
	evm.Context.Time = uint64(parameters.Timestamp)

	value := parameters.Value.ToUint256()
	addr := geth.AccountRef(parameters.Recipient)
	contract := geth.NewContract(addr, addr, value, uint64(parameters.Gas))
	contract.CallerAddress = common.Address(parameters.Sender)
	contract.Code = parameters.Code
	contract.CodeHash = crypto.Keccak256Hash(parameters.Code)
	contract.Input = parameters.Input

	return evm, contract, stateDb
}

// --- Adapter ---

// transferFunc subtracts amount from sender and adds amount to recipient using the given Db
// Now is doing nothing as this is not changing gas computation
func transferFunc(stateDB geth.StateDB, callerAddress common.Address, to common.Address, value *uint256.Int) {
	// Can be something like this:
	stateDB.SubBalance(callerAddress, value, tracing.BalanceChangeTransfer)
	stateDB.AddBalance(to, value, tracing.BalanceChangeTransfer)
}

// canTransferFunc is the signature of a transfer function
func canTransferFunc(stateDB geth.StateDB, callerAddress common.Address, value *uint256.Int) bool {
	return stateDB.GetBalance(callerAddress).Cmp(value) >= 0
}

// stateDbAdapter adapts the tosca.TransactionContext interface for its usage as a geth.StateDB
// in test environment setups. The main purpose is to facilitate unit, integration, and
// conformance testing for the geth interpreter.
type stateDbAdapter struct {
	context         tosca.TransactionContext
	refund          uint64
	lastBeneficiary tosca.Address
	refundBackups   map[tosca.Snapshot]uint64
}

func NewStateDbAdapter(context tosca.TransactionContext) *stateDbAdapter {
	return &stateDbAdapter{
		context: context,
	}
}

func (s *stateDbAdapter) CreateAccount(common.Address) {
	// ignored: effect not needed in test environments
}

func (s *stateDbAdapter) CreateContract(common.Address) {
	// ignored: effect not needed in test environments
}

func (s *stateDbAdapter) SubBalance(addr common.Address, diff *uint256.Int, _ tracing.BalanceChangeReason) {
	// Tests only running the interpreter would never call this function since balances are
	// handled by the EVM implementation. However, Fantom's state precompile contract may
	// conduct direct calls tho this function as part of a contract execution. Thus, it
	// is required when running tests targeting the processor.
	account := tosca.Address(addr)
	cur := s.context.GetBalance(account)
	s.context.SetBalance(account, tosca.Sub(cur, tosca.ValueFromUint256(diff)))
}

func (s *stateDbAdapter) AddBalance(addr common.Address, diff *uint256.Int, _ tracing.BalanceChangeReason) {
	// Tests only running the interpreter would never call this function since balances are
	// handled by the EVM implementation. However, Fantom's state precompile contract may
	// conduct direct calls tho this function as part of a contract execution. Thus, it
	// is required when running tests targeting the processor.
	account := tosca.Address(addr)
	cur := s.context.GetBalance(account)
	s.context.SetBalance(account, tosca.Add(cur, tosca.ValueFromUint256(diff)))

	// we save this address to be used as the beneficiary in a selfdestruct case.
	s.lastBeneficiary = tosca.Address(addr)
}

func (s *stateDbAdapter) GetBalance(addr common.Address) *uint256.Int {
	value := s.context.GetBalance(tosca.Address(addr))
	return value.ToUint256()
}

func (s *stateDbAdapter) GetNonce(addr common.Address) uint64 {
	return s.context.GetNonce(tosca.Address(addr))
}

func (s *stateDbAdapter) SetNonce(addr common.Address, nonce uint64) {
	s.context.SetNonce(tosca.Address(addr), nonce)
}

func (s *stateDbAdapter) GetCodeHash(addr common.Address) common.Hash {
	return common.Hash(s.context.GetCodeHash(tosca.Address(addr)))
}

func (s *stateDbAdapter) GetCode(addr common.Address) []byte {
	return s.context.GetCode(tosca.Address(addr))
}

func (s *stateDbAdapter) SetCode(addr common.Address, code []byte) {
	s.context.SetCode(tosca.Address(addr), code)
}

func (s *stateDbAdapter) GetCodeSize(addr common.Address) int {
	return s.context.GetCodeSize(tosca.Address(addr))
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
	//lint:ignore SA1019 deprecated functions to be migrated in #616
	return common.Hash(s.context.GetCommittedStorage(tosca.Address(addr), tosca.Key(key)))
}

func (s *stateDbAdapter) GetState(addr common.Address, key common.Hash) common.Hash {
	return common.Hash(s.context.GetStorage(tosca.Address(addr), tosca.Key(key)))
}

func (s *stateDbAdapter) SetState(addr common.Address, key common.Hash, value common.Hash) {
	s.context.SetStorage(tosca.Address(addr), tosca.Key(key), tosca.Word(value))
}

func (s *stateDbAdapter) GetStorageRoot(addr common.Address) common.Hash {
	// ignored: effect not needed in test environments
	return common.Hash{}
}

func (s *stateDbAdapter) GetTransientState(addr common.Address, key common.Hash) common.Hash {
	return common.Hash(s.context.GetTransientStorage(tosca.Address(addr), tosca.Key(key)))
}

func (s *stateDbAdapter) SetTransientState(addr common.Address, key, value common.Hash) {
	s.context.SetTransientStorage(tosca.Address(addr), tosca.Key(key), tosca.Word(value))
}

func (s *stateDbAdapter) SelfDestruct(addr common.Address) {
	s.context.SelfDestruct(tosca.Address(addr), s.lastBeneficiary)
}

func (s *stateDbAdapter) HasSelfDestructed(addr common.Address) bool {
	//lint:ignore SA1019 deprecated functions to be migrated in #616
	return s.context.HasSelfDestructed(tosca.Address(addr))
}

func (s *stateDbAdapter) Selfdestruct6780(addr common.Address) {
	s.context.SelfDestruct(tosca.Address(addr), s.lastBeneficiary)
}

func (s *stateDbAdapter) Exist(addr common.Address) bool {
	return s.context.AccountExists(tosca.Address(addr))
}

func (s *stateDbAdapter) Empty(addr common.Address) bool {
	return s.GetBalance(addr).IsZero() && s.GetNonce(addr) == 0 && s.GetCodeSize(addr) == 0
}

func (s *stateDbAdapter) PrepareAccessList(sender common.Address, dest *common.Address, precompiles []common.Address, txAccesses types.AccessList) {
	s.context.AccessAccount(tosca.Address(sender))
	if dest != nil {
		s.context.AccessAccount(tosca.Address(*dest))
	}
	for _, addr := range precompiles {
		s.context.AccessAccount(tosca.Address(addr))
	}
	for _, el := range txAccesses {
		s.context.AccessAccount(tosca.Address(el.Address))
		for _, key := range el.StorageKeys {
			s.context.AccessStorage(tosca.Address(el.Address), tosca.Key(key))
		}
	}
}

func (s *stateDbAdapter) AddressInAccessList(addr common.Address) bool {
	//lint:ignore SA1019 deprecated functions to be migrated in #616
	return s.context.IsAddressInAccessList(tosca.Address(addr))
}

func (s *stateDbAdapter) SlotInAccessList(addr common.Address, slot common.Hash) (addressOk bool, slotOk bool) {
	//lint:ignore SA1019 deprecated functions to be migrated in #616
	return s.context.IsSlotInAccessList(tosca.Address(addr), tosca.Key(slot))
}

func (s *stateDbAdapter) AddAddressToAccessList(addr common.Address) {
	s.context.AccessAccount(tosca.Address(addr))
}

func (s *stateDbAdapter) AddSlotToAccessList(addr common.Address, slot common.Hash) {
	s.context.AccessStorage(tosca.Address(addr), tosca.Key(slot))
}

func (s *stateDbAdapter) Prepare(rules params.Rules, sender, coinbase common.Address, dest *common.Address, precompiles []common.Address, txAccesses types.AccessList) {
	// ignored: effect not needed in test environments
	panic("not implemented")
}

func (s *stateDbAdapter) RevertToSnapshot(snapshot int) {
	s.context.RestoreSnapshot(tosca.Snapshot(snapshot))
	s.refund = s.refundBackups[tosca.Snapshot(snapshot)]
}

func (s *stateDbAdapter) Snapshot() int {
	id := s.context.CreateSnapshot()
	if s.refundBackups == nil {
		s.refundBackups = make(map[tosca.Snapshot]uint64)
	}
	s.refundBackups[id] = s.refund
	return int(id)
}

func (s *stateDbAdapter) AddLog(log *types.Log) {
	topics := make([]tosca.Hash, 0, len(log.Topics))
	for _, cur := range log.Topics {
		topics = append(topics, tosca.Hash(cur))
	}
	s.context.EmitLog(tosca.Log{
		Address: tosca.Address(log.Address),
		Topics:  topics,
		Data:    log.Data,
	})
}

func (s *stateDbAdapter) GetLogs() []tosca.Log {
	return s.context.GetLogs()
}

func (s *stateDbAdapter) AddPreimage(common.Hash, []byte) {
	panic("should not be needed in test environments")
}

func (s *stateDbAdapter) ForEachStorage(common.Address, func(common.Hash, common.Hash) bool) error {
	panic("should not be needed in test environments")
}

func (s *stateDbAdapter) PointCache() *utils.PointCache {
	// see https://eips.ethereum.org/EIPS/eip-4762
	panic("should not be needed by revisions up to Cancun")
}

func (s *stateDbAdapter) Witness() *stateless.Witness {
	// this should not be relevant for revisions up to Cancun
	return nil
}
