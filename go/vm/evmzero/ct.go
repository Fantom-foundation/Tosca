package evmzero

import (
	"errors"
	"fmt"
	"math"
	"math/big"

	bridge "github.com/Fantom-foundation/Tosca/go/common"
	ctcommon "github.com/Fantom-foundation/Tosca/go/ct/common"
	"github.com/Fantom-foundation/Tosca/go/ct/st"
	"github.com/Fantom-foundation/Tosca/go/ct/utils"
	"github.com/ethereum/evmc/v10/bindings/go/evmc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/params"
)

////////////////////////////////////////////////////////////
// Types

type evmzeroSteppableInterpreter struct {
	chainConfig *params.ChainConfig
	evm         *vm.EVM
	interpreter *bridge.EvmcSteppableInterpreter
}

type evaluation struct {
	// Errors that occurred during the build process of the evaluation
	issues []error

	// Interpreter
	evmzero *evmzeroSteppableInterpreter

	// Step parameters
	contract  *vm.Contract
	revision  evmc.Revision
	gasRefund uint64
	input     []byte
	status    evmc.StepStatus
	pc        uint64
	stack     []byte
	memory    []byte

	// Gas accounting for values exceeding int64
	gasReduction       uint64
	gasRefundReduction uint64
}

////////////////////////////////////////////////////////////
// Helpers

// transferFunc subtracts amount from sender and adds amount to recipient using the given Db
// Now is doing nothing as this is not changing gas computation
func transferFunc(stateDB vm.StateDB, callerAddress common.Address, to common.Address, value *big.Int) {
	// Can be something like this:
	// stateDB.SubBalance(callerAddress, value)
	// stateDB.AddBalance(to, value)
}

// canTransferFunc is the signature of a transfer function
func canTransferFunc(stateDB vm.StateDB, callerAddress common.Address, value *big.Int) bool {
	return stateDB.GetBalance(callerAddress).Cmp(value) >= 0
}

func getSteppableEvmzero(state *st.State, stateDb vm.StateDB) (*evmzeroSteppableInterpreter, error) {
	istanbulBlock, err := ctcommon.GetForkBlock(ctcommon.R07_Istanbul)
	if err != nil {
		return nil, err
	}
	berlinBlock, err := ctcommon.GetForkBlock(ctcommon.R09_Berlin)
	if err != nil {
		return nil, err
	}
	londonBlock, err := ctcommon.GetForkBlock(ctcommon.R10_London)
	if err != nil {
		return nil, err
	}

	// Set hard forks for chainconfig
	chainConfig := &params.ChainConfig{}
	chainConfig.ChainID = big.NewInt(0)
	chainConfig.IstanbulBlock = big.NewInt(int64(istanbulBlock))
	chainConfig.BerlinBlock = big.NewInt(int64(berlinBlock))
	chainConfig.LondonBlock = big.NewInt(int64(londonBlock))
	chainConfig.Ethash = new(params.EthashConfig)

	blockCtx, txCtx := convertCtBlockContextToEvmc(state.BlockContext)

	// Set interpreter variant for this VM
	config := vm.Config{
		InterpreterImpl: "evmzero-steppable",
	}

	evm := vm.NewEVM(blockCtx, txCtx, stateDb, chainConfig, config)

	evmzeroInterpreter, ok := evm.Interpreter().(*bridge.EvmcSteppableInterpreter)
	if !ok {
		return nil, fmt.Errorf("unable to get evmzero interpreter")
	}

	return &evmzeroSteppableInterpreter{
		chainConfig: chainConfig,
		evm:         evm,
		interpreter: evmzeroInterpreter,
	}, nil
}

// capGasToInt64 reduces the gas so that it can be represented by a signed 64 bit integer.
// However much gas exceeds the maximum signed 64 bit integer is returned as the second return value.
func capGasToInt64(gas uint64) (uint64, uint64) {
	if gas > math.MaxInt64 {
		reduction := gas - math.MaxInt64
		reducedGas := gas - reduction
		return reducedGas, reduction
	}
	return gas, 0
}

////////////////////////////////////////////////////////////
// ct -> evmzero/evmc

func convertCtStatusToEvmcStatus(status st.StatusCode) (evmc.StepStatus, error) {
	switch status {
	case st.Running:
		return evmc.Running, nil
	case st.Stopped:
		return evmc.Stopped, nil
	case st.Returned:
		return evmc.Returned, nil
	case st.Reverted:
		return evmc.Reverted, nil
	case st.Failed:
		return evmc.Failed, nil
	}
	return evmc.Failed, fmt.Errorf("unknown status code: %v", status)
}

func convertCtRevisionToEvmcRevision(revision ctcommon.Revision) (evmc.Revision, error) {
	switch revision {
	case ctcommon.R07_Istanbul:
		return evmc.Istanbul, nil
	case ctcommon.R09_Berlin:
		return evmc.Berlin, nil
	case ctcommon.R10_London:
		return evmc.London, nil
	case ctcommon.R99_UnknownNextRevision:
		return evmc.MaxRevision, nil
	}
	return -1, fmt.Errorf("unknown revision: %v", revision)
}

func convertCtCodeToEvmcCode(code *st.Code) []byte {
	evmcCode := make([]byte, code.Length())
	code.CopyTo(evmcCode)
	return evmcCode
}

func convertCtStackToEvmcStack(stack *st.Stack) []byte {
	stackBytes := stack.Size() * 32
	evmcStack := make([]byte, stackBytes)
	for i := stack.Size() - 1; i >= 0; i-- {
		val := stack.Get(i).Bytes32be()
		copy(evmcStack[stackBytes-(i+1)*32:], val[:])
	}
	return evmcStack
}

func convertCtMemoryToEvmcMemory(memory *st.Memory) []byte {
	return memory.Read(0, uint64(memory.Size()))
}

func convertCtBlockContextToEvmc(blockCtx st.BlockContext) (vm.BlockContext, vm.TxContext) {
	// Hashing function used in the context for BLOCKHASH instruction
	getHash := func(num uint64) common.Hash {
		return common.Hash{}
	}

	evmcBlockCtx := vm.BlockContext{
		CanTransfer: canTransferFunc,
		Transfer:    transferFunc,
		GetHash:     getHash,
		Coinbase:    (common.Address)(blockCtx.CoinBase[:]),
		GasLimit:    blockCtx.GasLimit,
		BlockNumber: big.NewInt(0).SetUint64(blockCtx.BlockNumber),
		Time:        big.NewInt(0).SetUint64(blockCtx.TimeStamp),
		Difficulty:  blockCtx.Difficulty.ToBigInt(),
		BaseFee:     big.NewInt(100),
	}

	evmcTxCtx := vm.TxContext{
		GasPrice: blockCtx.GasPrice.ToBigInt(),
	}

	return evmcBlockCtx, evmcTxCtx
}

func CreateEvaluation(state *st.State) (e *evaluation) {
	e = &evaluation{}

	// Hack: Special handling for unknown revision, because evmzero cannot represent an invalid revision.
	// So we mark the status as failed already.
	// TODO: Fix this once we add full revision support to the CT and evmzero.
	if state.Revision > ctcommon.R10_London {
		state.Revision = ctcommon.R10_London
		state.Status = st.Failed
	}

	revision, err := convertCtRevisionToEvmcRevision(state.Revision)
	if err != nil {
		e.issues = append(e.issues, err)
		return
	}

	stateDb := utils.NewConformanceTestStateDb(state.Storage, state.Logs, state.Revision)
	stateDb.AddRefund(state.GasRefund)

	evmzero, err := getSteppableEvmzero(state, stateDb)
	if err != nil {
		e.issues = append(e.issues, err)
		return
	}

	convertedGas, gasReduction := capGasToInt64(state.Gas)
	convertedGasRefund, gasRefundReduction := capGasToInt64(state.GasRefund)

	evmzero.evm.Origin = (common.Address)(state.CallContext.OriginAddress)
	objectAddress := (vm.AccountRef)(state.CallContext.AccountAddress)
	callerAddress := (vm.AccountRef)(state.CallContext.CallerAddress)
	contract := vm.NewContract(callerAddress, objectAddress, state.CallContext.Value.ToBigInt(), uint64(convertedGas))
	contract.Code = convertCtCodeToEvmcCode(state.Code)

	status, err := convertCtStatusToEvmcStatus(state.Status)
	if err != nil {
		e.issues = append(e.issues, err)
		return
	}

	e.evmzero = evmzero
	e.contract = contract
	e.revision = revision
	e.gasRefund = convertedGasRefund
	e.input = make([]byte, 0)
	e.status = status
	e.pc = uint64(state.Pc)
	e.stack = convertCtStackToEvmcStack(state.Stack)
	e.memory = convertCtMemoryToEvmcMemory(state.Memory)
	e.gasReduction = gasReduction
	e.gasRefundReduction = gasRefundReduction

	return
}

func (e *evaluation) Run(numSteps int) (*st.State, error) {
	if len(e.issues) > 0 {
		return nil, errors.Join(e.issues...)
	}
	res, err := e.evmzero.interpreter.StepN(
		e.contract,
		e.revision,
		e.gasRefund,
		e.input,
		e.status,
		e.pc,
		e.stack,
		e.memory,
		numSteps)
	if err != nil {
		return nil, err
	}

	return e.convertEvmzeroStateToCtState(res)
}

////////////////////////////////////////////////////////////
// evmzero/evmc -> ct

func convertEvmcStatusToCtStatus(stepStatus evmc.StepStatus) (st.StatusCode, error) {
	switch stepStatus {
	case evmc.Running:
		return st.Running, nil
	case evmc.Stopped:
		return st.Stopped, nil
	case evmc.Returned:
		return st.Returned, nil
	case evmc.Reverted:
		return st.Reverted, nil
	case evmc.Failed:
		return st.Failed, nil
	}
	return st.Failed, fmt.Errorf("unknown status code: %v", stepStatus)
}

func convertEvmcRevisionToCtRevision(revision evmc.Revision) (ctcommon.Revision, error) {
	switch revision {
	case evmc.Istanbul:
		return ctcommon.R07_Istanbul, nil
	case evmc.Berlin:
		return ctcommon.R09_Berlin, nil
	case evmc.London:
		return ctcommon.R10_London, nil
	case evmc.MaxRevision:
		return ctcommon.R99_UnknownNextRevision, nil
	}
	return ctcommon.R99_UnknownNextRevision, fmt.Errorf("unknown revision: %v", revision)
}

func convertEvmcStackToCtStack(stack []byte) (*st.Stack, error) {
	if len(stack)%32 != 0 {
		return nil, fmt.Errorf("stack size is not a multiple of 32")
	}
	result := st.NewStack()
	for i := len(stack) - 32; i >= 0; i -= 32 {
		val := ctcommon.NewU256FromBytes(stack[i : i+32]...)
		result.Push(val)
	}
	return result, nil
}

func convertEvmcMemoryToCtMemory(memory []byte) *st.Memory {
	result := st.NewMemory()
	result.Set(memory)
	return result
}

func convertEvmzeroStateToCallContext(e *evaluation) st.CallContext {
	return st.CallContext{
		AccountAddress: (ctcommon.Address)(e.contract.Address()),
		OriginAddress:  (ctcommon.Address)(e.evmzero.evm.Origin),
		CallerAddress:  (ctcommon.Address)(e.contract.CallerAddress),
		Value:          ctcommon.NewU256FromBigInt(e.contract.Value()),
	}
}

func convertEvmzeroStateToBlockContext(e *evaluation) st.BlockContext {
	return st.BlockContext{
		BlockNumber: e.evmzero.evm.Context.BlockNumber.Uint64(),
		CoinBase:    (ctcommon.Address)(e.evmzero.evm.Context.Coinbase),
		GasLimit:    e.evmzero.evm.Context.GasLimit,
		GasPrice:    ctcommon.NewU256FromBigInt(e.evmzero.evm.TxContext.GasPrice),
		Difficulty:  ctcommon.NewU256FromBigInt(e.evmzero.evm.Context.Difficulty),
		TimeStamp:   e.evmzero.evm.Context.Time.Uint64(),
	}
}

func (e *evaluation) convertEvmzeroStateToCtState(result evmc.StepResult) (*st.State, error) {
	status, err := convertEvmcStatusToCtStatus(result.StepStatusCode)
	if err != nil {
		return nil, err
	}

	stack, err := convertEvmcStackToCtStack(result.Stack)
	if err != nil {
		return nil, err
	}

	revision, err := convertEvmcRevisionToCtRevision(result.Revision)
	if err != nil {
		return nil, err
	}

	res := &st.State{
		Status:       status,
		Revision:     revision,
		Pc:           uint16(result.Pc),
		Gas:          uint64(result.GasLeft) + e.gasReduction,
		GasRefund:    uint64(result.GasRefund) + e.gasRefundReduction,
		Code:         st.NewCode(e.contract.Code),
		Stack:        stack,
		Memory:       convertEvmcMemoryToCtMemory(result.Memory),
		CallContext:  convertEvmzeroStateToCallContext(e),
		BlockContext: convertEvmzeroStateToBlockContext(e),
	}

	if e.evmzero.evm.StateDB != nil {
		stateDb, ok := e.evmzero.evm.StateDB.(*utils.ConformanceTestStateDb)
		if !ok {
			return nil, fmt.Errorf("unexpected StateDB type: %T", e.evmzero.evm.StateDB)
		}
		res.Storage = stateDb.Storage
		res.Logs = stateDb.Logs
	}

	return res, nil
}
