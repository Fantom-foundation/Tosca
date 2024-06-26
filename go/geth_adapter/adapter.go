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

	"github.com/Fantom-foundation/Tosca/go/vm"
	_ "github.com/Fantom-foundation/Tosca/go/vm/evmone"
	_ "github.com/Fantom-foundation/Tosca/go/vm/evmzero"
	_ "github.com/Fantom-foundation/Tosca/go/vm/lfvm"
	gc "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/types"
	geth "github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/params"
	"github.com/holiman/uint256"
)

const adapterDebug = false

func init() {
	for name, interpreter := range vm.GetAllRegisteredInterpreters() {
		interpreter := interpreter
		geth.RegisterInterpreterFactory(name, func(evm *geth.EVM, cfg geth.Config) geth.Interpreter {
			return &gethInterpreterAdapter{
				interpreter: interpreter,
				evm:         evm,
				cfg:         cfg,
			}
		})
	}
}

type gethInterpreterAdapter struct {
	interpreter vm.Interpreter
	evm         *geth.EVM
	cfg         geth.Config
}

func (a *gethInterpreterAdapter) Run(contract *geth.Contract, input []byte, readOnly bool) (ret []byte, err error) {

	if adapterDebug {
		fmt.Printf("Begin of interpreter:\n")
		fmt.Printf("\tInput:  %v\n", input)
		fmt.Printf("\tStatic: %v\n", readOnly)
	}

	if a.evm.GetDepth() == 0 {
		// Tosca EVM implementations update the refund in the StateDB only at the
		// end of a contract execution. As a result, it may happen that the refund
		// becomes temporary negative, since a nested contract may trigger a
		// refund reduction of some refund earned by an enclosing, yet not finished
		// contract. However, geth can not handle negative refunds. Thus, we are
		// shifting the refund base line for a Tosca execution artificially by 2^60
		// to avoid temporary negative refunds, and eliminate this refund at the
		// end of the contract execution again.
		const refundShift = uint64(1 << 60)
		a.evm.StateDB.AddRefund(refundShift)
		defer func() {
			if err == nil || err == geth.ErrExecutionReverted {
				// In revert cases the accumulated refund to this point may be negative,
				// which would cause the subtraction of the original refundShift to
				// underflow the refund in the StateDB. Thus, the back-shift is capped
				// by the available refund.
				shift := refundShift
				if cur := a.evm.StateDB.GetRefund(); cur < shift {
					shift = cur
				}
				a.evm.StateDB.SubRefund(shift)
			}
		}()
	}

	// The geth EVM infrastructure does not offer means for forwarding read-only
	// state information through recursive interpreter calls. Internally, geth
	// is tracking this in a non-accessible member field of the geth interpreter.
	// This is not a desirable solution (due to its dependency on a stateful
	// interpreter). To circumvent this, this adapter encodes the read-only mode
	// into the highest bit of the gas value (see Call function below). This section
	// is eliminating this encoded information again.
	if a.evm.GetDepth() > 0 {
		readOnly = readOnly || contract.Gas >= (1<<63)
		if contract.Gas >= (1 << 63) {
			contract.Gas -= (1 << 63)
		}
	}

	// Track the recursive call depth of this Call within a transaction.
	// A maximum limit of params.CallCreateDepth must be enforced.
	if a.evm.GetDepth() > int(params.CallCreateDepth) {
		return nil, geth.ErrDepth
	}
	a.evm.SetDepth(a.evm.GetDepth() + 1)
	defer func() { a.evm.SetDepth(a.evm.GetDepth() - 1) }()

	// Pick proper Tosca revision based on block height.
	revision := vm.R07_Istanbul
	if chainConfig := a.evm.ChainConfig(); chainConfig != nil {
		// Note: configurations need to be checked in reverse order since
		// later revisions implicitly include earlier revisions.
		if chainConfig.IsLondon(a.evm.Context.BlockNumber) {
			revision = vm.R10_London
		} else if chainConfig.IsBerlin(a.evm.Context.BlockNumber) {
			revision = vm.R09_Berlin
		}
	}

	if adapterDebug {
		fmt.Printf("Running revision %v\n", revision)
	}

	// Convert the value from big-int to vm.Value.
	value := vm.Uint256ToValue(contract.Value())

	var codeHash *vm.Hash
	if contract.CodeHash != (gc.Hash{}) {
		hash := vm.Hash(contract.CodeHash)
		codeHash = &hash
	}

	chainId, err := bigIntToWord(a.evm.ChainConfig().ChainID)
	if err != nil {
		return nil, fmt.Errorf("could not convert chain Id: %v", err)
	}

	// BaseFee can be assumed zero unless set.
	baseFee, err := bigIntToValue(a.evm.Context.BaseFee)
	if err != nil {
		return nil, fmt.Errorf("could not convert base fee: %v", err)
	}

	difficulty, err := bigIntToHash(a.evm.Context.Difficulty)
	if err != nil {
		return nil, fmt.Errorf("could not convert difficulty: %v", err)
	}

	blobBaseFee, err := bigIntToValue(a.evm.Context.BlobBaseFee)
	if err != nil {
		return nil, fmt.Errorf("could not convert blob-base fee: %v", err)
	}

	blockParameters := vm.BlockParameters{
		ChainID:     chainId,
		BlockNumber: a.evm.Context.BlockNumber.Int64(),
		Timestamp:   int64(a.evm.Context.Time),
		Coinbase:    vm.Address(a.evm.Context.Coinbase),
		GasLimit:    vm.Gas(a.evm.Context.GasLimit),
		PrevRandao:  difficulty,
		BaseFee:     baseFee,
		BlobBaseFee: blobBaseFee,
		Revision:    revision,
	}

	gasPrice, err := bigIntToValue(a.evm.TxContext.GasPrice)
	if err != nil {
		return nil, fmt.Errorf("could not convert gas price: %v", err)
	}

	transactionParameters := vm.TransactionParameters{
		Origin:     vm.Address(a.evm.Origin),
		GasPrice:   gasPrice,
		BlobHashes: nil, // TODO: add
	}

	params := vm.Parameters{
		BlockParameters:       blockParameters,
		TransactionParameters: transactionParameters,
		Context:               &runContextAdapter{a.evm, &a.cfg, contract, readOnly},
		Kind:                  vm.Call, // < this might be wrong, but seems to be unused
		Static:                readOnly,
		Depth:                 a.evm.GetDepth() - 1,
		Gas:                   vm.Gas(contract.Gas),
		Recipient:             vm.Address(contract.Address()),
		Sender:                vm.Address(contract.Caller()),
		Input:                 input,
		Value:                 value,
		CodeHash:              codeHash,
		Code:                  contract.Code,
	}

	res, err := a.interpreter.Run(params)
	if err != nil {
		return nil, fmt.Errorf("internal interpreter error: %v", err)
	}

	if adapterDebug {
		fmt.Printf("End of interpreter:\n")
		fmt.Printf("\tSuccess:  %v\n", res.Success)
		fmt.Printf("\tOutput:   %v\n", res.Output)
		fmt.Printf("\tGas Left: %v\n", res.GasLeft)
		fmt.Printf("\tRefund:   %v\n", res.GasRefund)
	}

	// Update gas levels.
	if res.GasLeft > 0 {
		contract.Gas = uint64(res.GasLeft)
	} else {
		contract.Gas = 0
	}

	// Update refunds.
	if res.Success {
		if res.GasRefund >= 0 {
			a.evm.StateDB.AddRefund(uint64(res.GasRefund))
		} else {
			a.evm.StateDB.SubRefund(uint64(-res.GasRefund))
		}
	}

	// In geth, reverted executions are signaled through an error.
	// The only two types that need to be differentiated are revert
	// errors (in which gas is accounted for accurately) and any
	// other error.
	if (res.GasLeft > 0 || len(res.Output) > 0) && !res.Success {
		return res.Output, geth.ErrExecutionReverted
	}
	if !res.Success {
		return nil, geth.ErrOutOfGas // < they are all handled equally
	}
	return res.Output, nil
}

// runContextAdapter implements the vm.RunContext interface using geth infrastructure.
type runContextAdapter struct {
	evm      *geth.EVM
	cfg      *geth.Config
	contract *geth.Contract
	readOnly bool
}

func (a *runContextAdapter) AccountExists(addr vm.Address) bool {
	return a.evm.StateDB.Exist(gc.Address(addr))
}

func (a *runContextAdapter) CreateAccount(addr vm.Address, code vm.Code) bool {
	if a.AccountExists(addr) {
		return false
	}
	a.evm.StateDB.CreateAccount(gc.Address(addr))
	a.evm.StateDB.SetCode(gc.Address(addr), code)
	return true
}

func (a *runContextAdapter) GetNonce(addr vm.Address) uint64 {
	return a.evm.StateDB.GetNonce(gc.Address(addr))
}

func (a *runContextAdapter) SetNonce(addr vm.Address, nonce uint64) {
	a.evm.StateDB.SetNonce(gc.Address(addr), nonce)
}

func (a *runContextAdapter) GetStorage(addr vm.Address, key vm.Key) vm.Word {
	return vm.Word(a.evm.StateDB.GetState(gc.Address(addr), gc.Hash(key)))
}

func (a *runContextAdapter) SetStorage(addr vm.Address, key vm.Key, future vm.Word) vm.StorageStatus {
	current := a.GetStorage(addr, key)
	if current == future {
		return vm.StorageAssigned
	}
	original := vm.Word(a.evm.StateDB.GetCommittedState(gc.Address(addr), gc.Hash(key)))
	a.evm.StateDB.SetState(gc.Address(addr), gc.Hash(key), gc.Hash(future))
	return vm.GetStorageStatus(original, current, future)
}

func (a *runContextAdapter) GetTransientStorage(addr vm.Address, key vm.Key) vm.Word {
	return vm.Word(a.evm.StateDB.GetTransientState(gc.Address(addr), gc.Hash(key)))
}

func (a *runContextAdapter) SetTransientStorage(addr vm.Address, key vm.Key, future vm.Word) {
	a.evm.StateDB.SetTransientState(gc.Address(addr), gc.Hash(key), gc.Hash(future))
}

func (a *runContextAdapter) GetBalance(addr vm.Address) vm.Value {
	return vm.Uint256ToValue(a.evm.StateDB.GetBalance(gc.Address(addr)))
}

func (a *runContextAdapter) SetBalance(addr vm.Address, value vm.Value) {
	panic("not implemented - should not be needed")
}

func (a *runContextAdapter) GetCodeSize(addr vm.Address) int {
	return a.evm.StateDB.GetCodeSize(gc.Address(addr))
}

func (a *runContextAdapter) GetCodeHash(addr vm.Address) vm.Hash {
	return vm.Hash(a.evm.StateDB.GetCodeHash(gc.Address(addr)))
}

func (a *runContextAdapter) GetCode(addr vm.Address) vm.Code {
	return a.evm.StateDB.GetCode(gc.Address(addr))
}

func (a *runContextAdapter) GetBlockHash(number int64) vm.Hash {
	return vm.Hash(a.evm.Context.GetHash(uint64(number)))
}

func (a *runContextAdapter) EmitLog(log vm.Log) {
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

func (a *runContextAdapter) GetLogs() []vm.Log {
	panic("not implemented")
}

func (a *runContextAdapter) Call(kind vm.CallKind, parameter vm.CallParameters) (result vm.CallResult, reserr error) {
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

		defer func() {
			fmt.Printf("End of call:\n")
			fmt.Printf("\tOutput:    %v\n", result.Output)
			fmt.Printf("\tGasLeft:   %v\n", result.GasLeft)
			fmt.Printf("\tGasRefund: %v\n", result.GasRefund)
			fmt.Printf("\tSuccess:   %v\n", result.Success)
			fmt.Printf("\tError:     %v\n", reserr)
		}()
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
	gas := uint64(parameter.Gas)
	if !isPrecompiledContract(parameter.Recipient) {
		if a.readOnly {
			gas += (1 << 63)
		}
	}

	// Documentation of the parameters can be found here: t.ly/yhxC
	toAddr := gc.Address(parameter.Recipient)

	var err error
	var output []byte
	var returnGas uint64
	var createdAddress vm.Address
	switch kind {
	case vm.Call:
		output, returnGas, err = a.evm.Call(a.contract, toAddr, parameter.Input, gas, vm.ValueToUint256(parameter.Value))
	case vm.StaticCall:
		output, returnGas, err = a.evm.StaticCall(a.contract, toAddr, parameter.Input, gas)
	case vm.DelegateCall:
		toAddr = gc.Address(parameter.CodeAddress)
		output, returnGas, err = a.evm.DelegateCall(a.contract, toAddr, parameter.Input, gas)
	case vm.CallCode:
		toAddr = gc.Address(parameter.CodeAddress)
		output, returnGas, err = a.evm.CallCode(a.contract, toAddr, parameter.Input, gas, vm.ValueToUint256(parameter.Value))
	case vm.Create:
		var newAddr gc.Address
		output, newAddr, returnGas, err = a.evm.Create(a.contract, parameter.Input, gas, vm.ValueToUint256(parameter.Value))
		createdAddress = vm.Address(newAddr)
	case vm.Create2:
		var newAddr gc.Address
		vmSalt := &uint256.Int{}
		vmSalt.SetBytes(parameter.Salt[:])
		output, newAddr, returnGas, err = a.evm.Create2(a.contract, parameter.Input, gas, vm.ValueToUint256(parameter.Value), vmSalt)
		createdAddress = vm.Address(newAddr)
	default:
		panic(fmt.Sprintf("unsupported call kind: %v", kind))
	}

	if adapterDebug {
		fmt.Printf("Result:\n\t%v\n\t%v\n\t%v\n\t%v\n", output, createdAddress, returnGas, err)
	}

	if err != nil {
		// translate geth errors to vm errors
		switch err {
		case geth.ErrExecutionReverted:
			// revert errors are not an error in Tosca
		case
			geth.ErrOutOfGas,
			geth.ErrCodeStoreOutOfGas,
			geth.ErrDepth,
			geth.ErrContractAddressCollision,
			geth.ErrExecutionReverted,
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
			return vm.CallResult{Success: false}, nil
		case geth.ErrInsufficientBalance:
			// In this case, the caller get its gas back.
			// TODO: this seems to be a geth implementation quirk that got
			// transferred into the LFVM implementation; this should be fixed.
			return vm.CallResult{
				GasLeft: parameter.Gas,
				Success: false,
			}, nil
		default:
			if _, ok := err.(*geth.ErrStackUnderflow); ok {
				return vm.CallResult{Success: false}, nil
			}
			if _, ok := err.(*geth.ErrStackOverflow); ok {
				return vm.CallResult{Success: false}, nil
			}
			if _, ok := err.(*geth.ErrInvalidOpCode); ok {
				return vm.CallResult{Success: false}, nil
			}
			return vm.CallResult{Success: false}, err
		}
	}

	return vm.CallResult{
		Output:         output,
		GasLeft:        vm.Gas(returnGas),
		GasRefund:      0, // refunds of nested calls are managed by the geth EVM and this adapter
		CreatedAddress: createdAddress,
		Success:        err == nil,
	}, nil
}

func (a *runContextAdapter) SelfDestruct(addr vm.Address, beneficiary vm.Address) bool {

	if adapterDebug {
		fmt.Printf("SelfDestruct called with %v, %v\n", addr, beneficiary)
	}

	stateDb := a.evm.StateDB
	if stateDb.HasSelfDestructed(gc.Address(addr)) {
		return false
	}
	balance := stateDb.GetBalance(a.contract.Address())
	stateDb.AddBalance(gc.Address(beneficiary), balance, tracing.BalanceDecreaseSelfdestruct)
	stateDb.SelfDestruct(gc.Address(addr))
	return true
}

func (a *runContextAdapter) SelfDestruct6780(addr vm.Address, beneficiary vm.Address) bool {
	if adapterDebug {
		fmt.Printf("SelfDestruct called with %v, %v\n", addr, beneficiary)
	}

	stateDb := a.evm.StateDB
	if stateDb.HasSelfDestructed(gc.Address(addr)) {
		return false
	}
	balance := stateDb.GetBalance(a.contract.Address())
	stateDb.AddBalance(gc.Address(beneficiary), balance, tracing.BalanceDecreaseSelfdestruct)
	stateDb.SubBalance(a.contract.Address(), balance, tracing.BalanceDecreaseSelfdestruct)
	stateDb.SelfDestruct(gc.Address(addr))
	return true
}

func (a *runContextAdapter) CreateSnapshot() vm.Snapshot {
	return vm.Snapshot(a.evm.StateDB.Snapshot())
}

func (a *runContextAdapter) RestoreSnapshot(snapshot vm.Snapshot) {
	a.evm.StateDB.RevertToSnapshot(int(snapshot))
}

func (a *runContextAdapter) AccessAccount(addr vm.Address) vm.AccessStatus {
	warm := a.IsAddressInAccessList(addr)
	a.evm.StateDB.AddAddressToAccessList(gc.Address(addr))
	if warm {
		return vm.WarmAccess
	}
	return vm.ColdAccess
}

func (a *runContextAdapter) AccessStorage(addr vm.Address, key vm.Key) vm.AccessStatus {
	_, warm := a.IsSlotInAccessList(addr, key)
	a.evm.StateDB.AddSlotToAccessList(gc.Address(addr), gc.Hash(key))
	if warm {
		return vm.WarmAccess
	}
	return vm.ColdAccess
}

// -- legacy API needed by LFVM and Geth, to be removed in the future ---

func (a *runContextAdapter) GetCommittedStorage(addr vm.Address, key vm.Key) vm.Word {
	return vm.Word(a.evm.StateDB.GetCommittedState(gc.Address(addr), gc.Hash(key)))
}

func (a *runContextAdapter) IsAddressInAccessList(addr vm.Address) bool {
	return a.evm.StateDB.AddressInAccessList(gc.Address(addr))
}

func (a *runContextAdapter) IsSlotInAccessList(addr vm.Address, key vm.Key) (addressPresent, slotPresent bool) {
	return a.evm.StateDB.SlotInAccessList(gc.Address(addr), gc.Hash(key))
}

func (a *runContextAdapter) HasSelfDestructed(addr vm.Address) bool {
	return a.evm.StateDB.HasSelfDestructed(gc.Address(addr))
}

func bigIntToValue(value *big.Int) (result vm.Value, err error) {
	if value == nil {
		return vm.Value{}, nil
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

func bigIntToHash(value *big.Int) (vm.Hash, error) {
	res, err := bigIntToValue(value)
	return vm.Hash(res), err
}

func bigIntToWord(value *big.Int) (vm.Word, error) {
	res, err := bigIntToValue(value)
	return vm.Word(res), err
}

func valueToBigInt(value vm.Value) *big.Int {
	return new(big.Int).SetBytes(value[:])
}

func isPrecompiledContract(recipient vm.Address) bool {
	// the addresses 1-9 are precompiled contracts
	for i := 0; i < 18; i++ {
		if recipient[i] != 0 {
			return false
		}
	}
	return 1 <= recipient[19] && recipient[19] <= 9
}
