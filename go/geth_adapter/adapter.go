// Copyright (c) 2024 Fantom Foundation
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at fantom.foundation/bsl11.
//
// Change Date: 2028-4-16
//
// On the date above, in accordance with the Business Source License, use of
// this software will be governed by the GNU Lesser General Public License v3.

// This package registers Tosca Interpreters in the go-ethereum-substate
// VM registry such that they can be used in tools like Aida until the
// EVM implementation provided by go-ethereum-substate is ultimately
// replaced by Tosca's implementation.
//
// This package does not provide any public API. It provides test
// infrastructure for the Aida-based nightly integration tests and
// as such implicitly tested.
package geth_adapter

import (
	"fmt"
	"math/big"

	"github.com/Fantom-foundation/Tosca/go/tosca"
	gc "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/types"
	geth "github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/params"
	"github.com/holiman/uint256"
)

func NewGethInterpreterFactory(interpreter tosca.Interpreter) geth.InterpreterFactory {
	return func(evm *geth.EVM) geth.Interpreter {
		return &gethInterpreterAdapter{
			interpreter: interpreter,
			evm:         evm,
		}
	}
}

const adapterDebug = false

type gethInterpreterAdapter struct {
	interpreter tosca.Interpreter
	evm         *geth.EVM
}

func (a *gethInterpreterAdapter) Run(contract *geth.Contract, input []byte, readOnly bool) (ret []byte, err error) {
	var result tosca.Result
	if adapterDebug {
		debugRunStart(input, readOnly)
		defer func() { debugRunEnd(result) }()
	}

	// Tosca EVM implementations update the refund in the StateDB only at the
	// end of a contract execution. As a result, it may happen that the refund
	// becomes temporary negative, since a nested contract may trigger a
	// refund reduction of some refund earned by an enclosing, yet not finished
	// contract. However, geth can not handle negative refunds. Thus, we are
	// shifting the refund base line for a Tosca execution artificially by 2^60
	// to avoid temporary negative refunds, and eliminate this refund at the
	// end of the contract execution again.
	if a.evm.GetDepth() == 0 {
		const refundShift = 1 << 60
		a.evm.StateDB.AddRefund(refundShift)
		defer func() { undoRefundShift(a.evm.StateDB, err, refundShift) }()
	}

	// The geth EVM infrastructure does not offer means for forwarding read-only
	// state information through recursive interpreter calls. Internally, geth
	// is tracking this in a non-accessible member field of the geth interpreter.
	// This is not a desirable solution (due to its dependency on a stateful
	// interpreter). To circumvent this, this adapter encodes the read-only mode
	// into the highest bit of the gas value (see Call function below). This section
	// is eliminating this encoded information again.
	readOnly, contract.Gas = decodeReadOnlyFromGas(a.evm.GetDepth(), readOnly, contract.Gas)

	// Track the recursive call depth of this Call within a transaction.
	// A maximum limit of params.CallCreateDepth must be enforced.
	if a.evm.GetDepth() > int(params.CallCreateDepth) {
		return nil, geth.ErrDepth
	}
	a.evm.SetDepth(a.evm.GetDepth() + 1)
	defer func() { a.evm.SetDepth(a.evm.GetDepth() - 1) }()

	rules := a.evm.ChainConfig().Rules(a.evm.Context.BlockNumber, a.evm.Context.Random != nil, a.evm.Context.Time)
	revision, err := convertRevision(rules)
	if err != nil {
		return nil, fmt.Errorf("unsupported revision: %w", err)
	}
	if adapterDebug {
		fmt.Printf("Running revision %v\n", revision)
	}

	// Convert the value from big-int to tosca.Value.
	value := tosca.ValueFromUint256(contract.Value())
	// BaseFee can be assumed zero unless set.
	baseFee, err := bigIntToValue(a.evm.Context.BaseFee)
	if err != nil {
		return nil, fmt.Errorf("could not convert base fee: %v", err)
	}
	chainId, err := bigIntToWord(a.evm.ChainConfig().ChainID)
	if err != nil {
		return nil, fmt.Errorf("could not convert chain Id: %v", err)
	}
	blobBaseFee, err := bigIntToValue(a.evm.Context.BlobBaseFee)
	if err != nil {
		return nil, fmt.Errorf("could not convert blob-base fee: %v", err)
	}
	gasPrice, err := bigIntToValue(a.evm.TxContext.GasPrice)
	if err != nil {
		return nil, fmt.Errorf("could not convert gas price: %v", err)
	}
	prevRandao, err := getPrevRandao(&a.evm.Context, revision)
	if err != nil {
		return nil, err
	}

	var codeHash *tosca.Hash
	if contract.CodeHash != (gc.Hash{}) {
		codeHash = (*tosca.Hash)(&contract.CodeHash)
	}

	blockParameters := tosca.BlockParameters{
		ChainID:     chainId,
		BlockNumber: a.evm.Context.BlockNumber.Int64(),
		Timestamp:   int64(a.evm.Context.Time),
		Coinbase:    tosca.Address(a.evm.Context.Coinbase),
		GasLimit:    tosca.Gas(a.evm.Context.GasLimit),
		PrevRandao:  prevRandao,
		BaseFee:     baseFee,
		BlobBaseFee: blobBaseFee,
		Revision:    revision,
	}

	blobHashes := make([]tosca.Hash, len(a.evm.BlobHashes))
	for i, hash := range a.evm.BlobHashes {
		blobHashes[i] = tosca.Hash(hash)
	}

	transactionParameters := tosca.TransactionParameters{
		Origin:     tosca.Address(a.evm.Origin),
		GasPrice:   gasPrice,
		BlobHashes: blobHashes,
	}

	params := tosca.Parameters{
		BlockParameters:       blockParameters,
		TransactionParameters: transactionParameters,
		Context:               &runContextAdapter{a.evm, contract, readOnly},
		Kind:                  tosca.Call, // < this might be wrong, but seems to be unused
		Static:                readOnly,
		Depth:                 a.evm.GetDepth() - 1,
		Gas:                   tosca.Gas(contract.Gas),
		Recipient:             tosca.Address(contract.Address()),
		Sender:                tosca.Address(contract.Caller()),
		Input:                 input,
		Value:                 value,
		CodeHash:              codeHash,
		Code:                  contract.Code,
	}

	result, err = a.interpreter.Run(params)
	if err != nil {
		return nil, fmt.Errorf("internal interpreter error: %v", err)
	}

	// Update gas levels.
	if result.GasLeft > 0 {
		contract.Gas = uint64(result.GasLeft)
	} else {
		contract.Gas = 0
	}

	// Update refunds.
	if result.Success {
		if result.GasRefund >= 0 {
			a.evm.StateDB.AddRefund(uint64(result.GasRefund))
		} else {
			a.evm.StateDB.SubRefund(uint64(-result.GasRefund))
		}
	}

	// In geth, reverted executions are signaled through an error.
	// The only two types that need to be differentiated are revert
	// errors (in which gas is accounted for accurately) and any
	// other error.
	if (result.GasLeft > 0 || len(result.Output) > 0) && !result.Success {
		return result.Output, geth.ErrExecutionReverted
	}
	if !result.Success {
		return nil, geth.ErrOutOfGas // < they are all handled equally
	}
	return result.Output, nil
}

func getPrevRandao(context *geth.BlockContext, revision tosca.Revision) (tosca.Hash, error) {
	if revision < tosca.R11_Paris {
		prevRandao, err := bigIntToHash(context.Difficulty)
		if err != nil {
			return tosca.Hash{}, fmt.Errorf("could not convert difficulty: %v", err)
		}
		return prevRandao, nil
	}

	var prevRandao tosca.Hash
	if context.Random != nil {
		prevRandao = tosca.Hash(*context.Random)
	}
	return prevRandao, nil
}

func debugRunStart(input []byte, readOnly bool) {
	fmt.Printf("Begin of interpreter:\n")
	fmt.Printf("\tInput:  %v\n", input)
	fmt.Printf("\tStatic: %v\n", readOnly)
}

func debugRunEnd(res tosca.Result) {
	fmt.Printf("End of interpreter:\n")
	fmt.Printf("\tSuccess:  %v\n", res.Success)
	fmt.Printf("\tOutput:   %v\n", res.Output)
	fmt.Printf("\tGas Left: %v\n", res.GasLeft)
	fmt.Printf("\tRefund:   %v\n", res.GasRefund)
}

func undoRefundShift(stateDB geth.StateDB, err error, refundShift uint64) {
	if err == nil || err == geth.ErrExecutionReverted {
		// In revert cases the accumulated refund to this point may be negative,
		// which would cause the subtraction of the original refundShift to
		// underflow the refund in the StateDB. Thus, the back-shift is capped
		// by the available refund.
		shift := refundShift
		if cur := stateDB.GetRefund(); cur < shift {
			shift = cur
		}
		stateDB.SubRefund(shift)
	}
}

func convertRevision(rules params.Rules) (tosca.Revision, error) {
	if rules.IsCancun {
		return tosca.R13_Cancun, nil
	} else if rules.IsShanghai {
		return tosca.R12_Shanghai, nil
	} else if rules.IsMerge {
		return tosca.R11_Paris, nil
	} else if rules.IsLondon {
		return tosca.R10_London, nil
	} else if rules.IsBerlin {
		return tosca.R09_Berlin, nil
	} else if rules.IsIstanbul {
		return tosca.R07_Istanbul, nil
	}
	return tosca.Revision(-1), &tosca.ErrUnsupportedRevision{Revision: tosca.Revision(-1)}
}

// runContextAdapter implements the tosca.RunContext interface using geth infrastructure.
type runContextAdapter struct {
	evm      *geth.EVM
	contract *geth.Contract
	readOnly bool
}

func (a *runContextAdapter) Call(kind tosca.CallKind, parameter tosca.CallParameters) (result tosca.CallResult, reserr error) {
	if adapterDebug {
		debugCallStart(kind, parameter)
		defer func() { debugCallEnd(result, reserr) }()
	}

	gas := encodeReadOnlyInGas(uint64(parameter.Gas), parameter.CodeAddress, a.readOnly)

	// Documentation of the parameters can be found here: t.ly/yhxC
	toAddr := gc.Address(parameter.Recipient)

	var (
		err            error
		output         []byte
		returnGas      uint64
		createdAddress tosca.Address
	)
	switch kind {
	case tosca.Call:
		output, returnGas, err = a.evm.Call(a.contract, toAddr, parameter.Input, gas, parameter.Value.ToUint256())
	case tosca.StaticCall:
		output, returnGas, err = a.evm.StaticCall(a.contract, toAddr, parameter.Input, gas)
	case tosca.DelegateCall:
		toAddr = gc.Address(parameter.CodeAddress)
		output, returnGas, err = a.evm.DelegateCall(a.contract, toAddr, parameter.Input, gas)
	case tosca.CallCode:
		toAddr = gc.Address(parameter.CodeAddress)
		output, returnGas, err = a.evm.CallCode(a.contract, toAddr, parameter.Input, gas, parameter.Value.ToUint256())
	case tosca.Create:
		var newAddr gc.Address
		output, newAddr, returnGas, err = a.evm.Create(a.contract, parameter.Input, gas, parameter.Value.ToUint256())
		createdAddress = tosca.Address(newAddr)
	case tosca.Create2:
		var newAddr gc.Address
		vmSalt := &uint256.Int{}
		vmSalt.SetBytes(parameter.Salt[:])
		output, newAddr, returnGas, err = a.evm.Create2(a.contract, parameter.Input, gas, parameter.Value.ToUint256(), vmSalt)
		createdAddress = tosca.Address(newAddr)
	default:
		panic(fmt.Sprintf("unsupported call kind: %v", kind))
	}

	// revert errors are not an error in Tosca
	if err != nil && err != geth.ErrExecutionReverted {
		return gethToVMErrors(err, parameter.Gas)
	}

	return tosca.CallResult{
		Output:         output,
		GasLeft:        tosca.Gas(returnGas),
		GasRefund:      0, // refunds of nested calls are managed by the geth EVM and this adapter
		CreatedAddress: createdAddress,
		Success:        err == nil,
	}, nil
}

// The geth EVM context does not provide the needed means
// to forward an existing read-only mode through arbitrary
// nested calls, as it would be needed. Thus, this information
// is encoded into the hightest bit of the gas value, which is
// interpreted as such by the Run() function above.
// The geth implementation itself tracks the read-only state in
// an implementation specific interpreter internal flag, which
// is not accessible from this context. Also, this method depends
// on a new interpreter per transaction call (for proper) scoping
// which is not a desired trait for Tosca interpreter implementations.
// With this trick, this requirement is circumvented.
func encodeReadOnlyInGas(gas uint64, recipient tosca.Address, readOnly bool) uint64 {
	if !isPrecompiledContract(recipient) {
		if readOnly {
			gas += (1 << 63)
		}
	}
	return gas
}

func decodeReadOnlyFromGas(depth int, readOnly bool, gas uint64) (bool, uint64) {
	if depth > 0 {
		readOnly = readOnly || gas >= (1<<63)
		if gas >= (1 << 63) {
			gas -= (1 << 63)
		}
	}
	return readOnly, gas
}

func gethToVMErrors(err error, gas tosca.Gas) (tosca.CallResult, error) {
	switch err {
	case
		geth.ErrInsufficientBalance,
		geth.ErrDepth,
		geth.ErrNonceUintOverflow:
		// In these cases, the caller get its gas back.
		// TODO: this seems to be a geth implementation quirk that got
		// transferred into the LFVM implementation; this should be fixed.
		return tosca.CallResult{
			GasLeft: gas,
			Success: false,
		}, nil
	case
		geth.ErrOutOfGas,
		geth.ErrCodeStoreOutOfGas,
		geth.ErrContractAddressCollision,
		geth.ErrExecutionReverted,
		geth.ErrMaxInitCodeSizeExceeded,
		geth.ErrMaxCodeSizeExceeded,
		geth.ErrInvalidJump,
		geth.ErrWriteProtection,
		geth.ErrReturnDataOutOfBounds,
		geth.ErrGasUintOverflow,
		geth.ErrInvalidCode:
		// These errors are issues encountered during the execution of
		// EVM byte code that got correctly handled by aborting the
		// execution. In Tosca, these are not considered errors, but
		// unsuccessful executions, and thus, they are reported as such.
		return tosca.CallResult{Success: false}, nil
	}

	if _, ok := err.(*geth.ErrStackUnderflow); ok {
		return tosca.CallResult{Success: false}, nil
	}
	if _, ok := err.(*geth.ErrStackOverflow); ok {
		return tosca.CallResult{Success: false}, nil
	}
	if _, ok := err.(*geth.ErrInvalidOpCode); ok {
		return tosca.CallResult{Success: false}, nil
	}

	return tosca.CallResult{Success: false}, err
}

func debugCallStart(kind tosca.CallKind, parameter tosca.CallParameters) {
	if adapterDebug {
		fmt.Printf("Start of call:\n")
		fmt.Printf("\tType:         %v\n", kind)
		fmt.Printf("\tRecipient:    %v\n", parameter.Recipient)
		fmt.Printf("\tSender:       %v\n", parameter.Sender)
		fmt.Printf("\tGas:          %v\n", parameter.Gas)
		fmt.Printf("\tInput:        %v\n", parameter.Input)
		fmt.Printf("\tValue:        %v\n", parameter.Value)
		fmt.Printf("\tSalt:         %v\n", parameter.Salt)
		fmt.Printf("\tCode address: %v\n", parameter.CodeAddress)
	}
}

func debugCallEnd(result tosca.CallResult, reserr error) {
	if adapterDebug {
		fmt.Printf("End of call:\n")
		fmt.Printf("\tOutput:    %v\n", result.Output)
		fmt.Printf("\tGasLeft:   %v\n", result.GasLeft)
		fmt.Printf("\tGasRefund: %v\n", result.GasRefund)
		fmt.Printf("\tSuccess:   %v\n", result.Success)
		fmt.Printf("\tError:     %v\n", reserr)
	}
}

func (a *runContextAdapter) AccountExists(addr tosca.Address) bool {
	return a.evm.StateDB.Exist(gc.Address(addr))
}

func (a *runContextAdapter) CreateAccount(addr tosca.Address, code tosca.Code) bool {
	if a.AccountExists(addr) {
		return false
	}
	a.evm.StateDB.CreateAccount(gc.Address(addr))
	a.evm.StateDB.SetCode(gc.Address(addr), code)
	return true
}

func (a *runContextAdapter) GetNonce(addr tosca.Address) uint64 {
	return a.evm.StateDB.GetNonce(gc.Address(addr))
}

func (a *runContextAdapter) SetNonce(addr tosca.Address, nonce uint64) {
	a.evm.StateDB.SetNonce(gc.Address(addr), nonce)
}

func (a *runContextAdapter) GetStorage(addr tosca.Address, key tosca.Key) tosca.Word {
	return tosca.Word(a.evm.StateDB.GetState(gc.Address(addr), gc.Hash(key)))
}

func (a *runContextAdapter) SetStorage(addr tosca.Address, key tosca.Key, future tosca.Word) tosca.StorageStatus {
	current := a.GetStorage(addr, key)
	if current == future {
		return tosca.StorageAssigned
	}
	original := tosca.Word(a.evm.StateDB.GetCommittedState(gc.Address(addr), gc.Hash(key)))
	a.evm.StateDB.SetState(gc.Address(addr), gc.Hash(key), gc.Hash(future))
	return tosca.GetStorageStatus(original, current, future)
}

func (a *runContextAdapter) GetTransientStorage(addr tosca.Address, key tosca.Key) tosca.Word {
	return tosca.Word(a.evm.StateDB.GetTransientState(gc.Address(addr), gc.Hash(key)))
}

func (a *runContextAdapter) SetTransientStorage(addr tosca.Address, key tosca.Key, future tosca.Word) {
	a.evm.StateDB.SetTransientState(gc.Address(addr), gc.Hash(key), gc.Hash(future))
}

func (a *runContextAdapter) GetBalance(addr tosca.Address) tosca.Value {
	return tosca.ValueFromUint256(a.evm.StateDB.GetBalance(gc.Address(addr)))
}

func (a *runContextAdapter) SetBalance(addr tosca.Address, value tosca.Value) {
	trg := gc.Address(addr)
	balance := a.evm.StateDB.GetBalance(trg)
	have := tosca.ValueFromUint256(balance)

	order := have.Cmp(value)
	if order < 0 {
		diff := tosca.Sub(value, have)
		a.evm.StateDB.AddBalance(trg, diff.ToUint256(), tracing.BalanceChangeUnspecified)
	} else if order > 0 {
		diff := tosca.Sub(have, value)
		a.evm.StateDB.SubBalance(trg, diff.ToUint256(), tracing.BalanceChangeUnspecified)
	}
}

func (a *runContextAdapter) GetCodeSize(addr tosca.Address) int {
	return a.evm.StateDB.GetCodeSize(gc.Address(addr))
}

func (a *runContextAdapter) GetCodeHash(addr tosca.Address) tosca.Hash {
	return tosca.Hash(a.evm.StateDB.GetCodeHash(gc.Address(addr)))
}

func (a *runContextAdapter) GetCode(addr tosca.Address) tosca.Code {
	return a.evm.StateDB.GetCode(gc.Address(addr))
}

func (a *runContextAdapter) SetCode(addr tosca.Address, code tosca.Code) {
	a.evm.StateDB.SetCode(gc.Address(addr), code)
}

func (a *runContextAdapter) GetBlockHash(number int64) tosca.Hash {
	return tosca.Hash(a.evm.Context.GetHash(uint64(number)))
}

func (a *runContextAdapter) EmitLog(log tosca.Log) {
	topics_in := log.Topics
	topics := make([]gc.Hash, len(topics_in))
	for i := range topics {
		topics[i] = gc.Hash(topics_in[i])
	}

	a.evm.StateDB.AddLog(&types.Log{
		Address:     gc.Address(log.Address),
		Topics:      ([]gc.Hash)(topics),
		Data:        log.Data,
		BlockNumber: a.evm.Context.BlockNumber.Uint64(),
	})
}

func (a *runContextAdapter) GetLogs() []tosca.Log {
	panic("not implemented")
}

func (a *runContextAdapter) SelfDestruct(addr tosca.Address, beneficiary tosca.Address) bool {
	if adapterDebug {
		fmt.Printf("SelfDestruct called with %v, %v\n", addr, beneficiary)
	}

	stateDb := a.evm.StateDB
	selfdestructed := true
	if stateDb.HasSelfDestructed(gc.Address(addr)) {
		selfdestructed = false
	}
	balance := stateDb.GetBalance(a.contract.Address())
	stateDb.AddBalance(gc.Address(beneficiary), balance, tracing.BalanceDecreaseSelfdestruct)

	if a.evm.ChainConfig().IsCancun(a.evm.Context.BlockNumber, a.evm.Context.Time) {
		stateDb.SubBalance(a.contract.Address(), balance, tracing.BalanceDecreaseSelfdestruct)
		stateDb.Selfdestruct6780(gc.Address(addr))
	} else {
		stateDb.SelfDestruct(gc.Address(addr))
	}

	return selfdestructed
}

func (a *runContextAdapter) CreateSnapshot() tosca.Snapshot {
	return tosca.Snapshot(a.evm.StateDB.Snapshot())
}

func (a *runContextAdapter) RestoreSnapshot(snapshot tosca.Snapshot) {
	a.evm.StateDB.RevertToSnapshot(int(snapshot))
}

func (a *runContextAdapter) AccessAccount(addr tosca.Address) tosca.AccessStatus {
	warm := a.IsAddressInAccessList(addr)
	a.evm.StateDB.AddAddressToAccessList(gc.Address(addr))
	if warm {
		return tosca.WarmAccess
	}
	return tosca.ColdAccess
}

func (a *runContextAdapter) AccessStorage(addr tosca.Address, key tosca.Key) tosca.AccessStatus {
	_, warm := a.IsSlotInAccessList(addr, key)
	a.evm.StateDB.AddSlotToAccessList(gc.Address(addr), gc.Hash(key))
	if warm {
		return tosca.WarmAccess
	}
	return tosca.ColdAccess
}

// -- legacy API needed by LFVM and Geth, to be removed in the future ---

func (a *runContextAdapter) GetCommittedStorage(addr tosca.Address, key tosca.Key) tosca.Word {
	return tosca.Word(a.evm.StateDB.GetCommittedState(gc.Address(addr), gc.Hash(key)))
}

func (a *runContextAdapter) IsAddressInAccessList(addr tosca.Address) bool {
	return a.evm.StateDB.AddressInAccessList(gc.Address(addr))
}

func (a *runContextAdapter) IsSlotInAccessList(addr tosca.Address, key tosca.Key) (addressPresent, slotPresent bool) {
	return a.evm.StateDB.SlotInAccessList(gc.Address(addr), gc.Hash(key))
}

func (a *runContextAdapter) HasSelfDestructed(addr tosca.Address) bool {
	return a.evm.StateDB.HasSelfDestructed(gc.Address(addr))
}

// utility functions

func bigIntToValue(value *big.Int) (result tosca.Value, err error) {
	if value == nil {
		return tosca.Value{}, nil
	}
	if value.Sign() < 0 {
		return result, fmt.Errorf("cannot convert a negative number to a Hash, got %v", value)
	}
	if len(value.Bytes()) > 32 {
		return result, fmt.Errorf("value exceeds maximum value for Hash, %v of 32 bytes max", len(value.Bytes()))
	}
	value.FillBytes(result[:])
	return result, nil
}

func bigIntToHash(value *big.Int) (tosca.Hash, error) {
	res, err := bigIntToValue(value)
	return tosca.Hash(res), err
}

func bigIntToWord(value *big.Int) (tosca.Word, error) {
	res, err := bigIntToValue(value)
	return tosca.Word(res), err
}

func isPrecompiledContract(recipient tosca.Address) bool {
	// the addresses 1-9 are precompiled contracts
	for i := 0; i < 18; i++ {
		if recipient[i] != 0 {
			return false
		}
	}
	return 1 <= recipient[19] && recipient[19] <= 9
}
