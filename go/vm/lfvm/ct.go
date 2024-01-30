package lfvm

import (
	"fmt"
	"math/big"

	ct "github.com/Fantom-foundation/Tosca/go/ct/common"
	"github.com/Fantom-foundation/Tosca/go/ct/st"
	"github.com/Fantom-foundation/Tosca/go/ct/utils"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/params"
	"github.com/holiman/uint256"
)

////////////////////////////////////////////////////////////
// lfvm -> ct : helper functions

func convertLfvmStatusToCtStatus(status Status) (st.StatusCode, error) {
	switch status {
	case RUNNING:
		return st.Running, nil
	case STOPPED:
		return st.Stopped, nil
	case REVERTED:
		return st.Reverted, nil
	case RETURNED:
		return st.Returned, nil
	case SUICIDED:
		// Suicide is not yet modeled by the CT, and for now it just maps to the STOPPED status.
		return st.Stopped, nil
	case INVALID_INSTRUCTION:
		return st.Failed, nil
	case OUT_OF_GAS:
		return st.Failed, nil
	case SEGMENTATION_FAULT:
		return st.Failed, nil
	case ERROR:
		return st.Failed, nil
	default:
		return st.Failed, fmt.Errorf("unable to convert lfvm status %v to ct status", status)
	}
}

func convertLfvmRevisionToCtRevision(ctx *context) ct.Revision {
	if ctx.isLondon {
		return ct.R10_London
	} else if ctx.isBerlin {
		return ct.R09_Berlin
	} else {
		return ct.R07_Istanbul
	}
}

func convertLfvmStackToCtStack(ctx *context) *st.Stack {
	stack := st.NewStack()

	for i := 0; i < ctx.stack.len(); i++ {
		val := ctx.stack.Data()[i]
		stack.Push(ct.NewU256(val[3], val[2], val[1], val[0]))
	}
	return stack
}

func convertLfvmMemoryToCtMemory(ctx *context) *st.Memory {
	memory := st.NewMemory()
	memory.Set(ctx.memory.GetSlice(0, ctx.memory.Len()))
	return memory
}

////////////////////////////////////////////////////////////
// lfvm -> ct

// getBigIntIfNotNil returns the input value if it is initialized
// or a default initialized new big.Int if not
func getBigIntIfNotNil(b *big.Int) *big.Int {
	if b == nil {
		return big.NewInt(0)
	}
	return b
}

// getChainConfigIfNotNil returns the input value if it is initialized
// or a default initialized new params.ChainConfig if not
func getChainConfigIfNotNil(c *params.ChainConfig) *params.ChainConfig {
	if c == nil {
		return &params.ChainConfig{}
	}
	return c
}

func convertLfvmContextToCtCallContext(ctx *context) st.CallContext {
	return st.CallContext{
		AccountAddress: (ct.Address)(ctx.contract.Address()),
		OriginAddress:  (ct.Address)(ctx.evm.Origin),
		CallerAddress:  (ct.Address)(ctx.contract.CallerAddress),
		Value:          ct.NewU256FromBigInt(getBigIntIfNotNil(ctx.contract.Value())),
	}
}

func convertLfvmContextToCtBlockContext(ctx *context) st.BlockContext {
	return st.BlockContext{
		BaseFee:     ct.NewU256FromBigInt(getBigIntIfNotNil(ctx.evm.Context.BaseFee)),
		BlockNumber: getBigIntIfNotNil(ctx.evm.Context.BlockNumber).Uint64(),
		ChainID:     ct.NewU256FromBigInt(getBigIntIfNotNil(getChainConfigIfNotNil(ctx.evm.ChainConfig()).ChainID)),
		CoinBase:    (ct.Address)(ctx.evm.Context.Coinbase),
		GasLimit:    ctx.evm.Context.GasLimit,
		GasPrice:    ct.NewU256FromBigInt(getBigIntIfNotNil(ctx.evm.GasPrice)),
		Difficulty:  ct.NewU256FromBigInt(getBigIntIfNotNil(ctx.evm.Context.Difficulty)),
		TimeStamp:   getBigIntIfNotNil(ctx.evm.Context.Time).Uint64(),
	}
}

func ConvertLfvmContextToCtState(ctx *context, originalCode *st.Code, pcMap *PcMap) (*st.State, error) {
	status, err := convertLfvmStatusToCtStatus(ctx.status)
	if err != nil {
		return nil, err
	}

	pc, ok := pcMap.lfvmToEvm[uint16(ctx.pc)]

	// Since two failed states are considered equal, the PC conversion may fail when the status is failed.
	if !ok && status != st.Failed {
		return nil, fmt.Errorf("unable to convert lfvm pc %d to evm pc", ctx.pc)
	}

	state := st.NewState(originalCode)
	state.Status = status
	state.Revision = convertLfvmRevisionToCtRevision(ctx)
	state.Pc = pc
	state.Gas = ctx.contract.Gas
	if ctx.stateDB != nil {
		state.GasRefund = ctx.stateDB.GetRefund()
	}
	state.ReadOnly = ctx.readOnly
	state.Code = originalCode
	state.Stack = convertLfvmStackToCtStack(ctx)
	state.Memory = convertLfvmMemoryToCtMemory(ctx)
	if ctx.stateDB != nil {
		state.Storage = ctx.stateDB.(*utils.ConformanceTestStateDb).Storage
		state.Logs = ctx.stateDB.(*utils.ConformanceTestStateDb).Logs
	}

	state.CallContext = convertLfvmContextToCtCallContext(ctx)

	state.BlockContext = convertLfvmContextToCtBlockContext(ctx)

	state.CallData = make([]byte, len(ctx.data))
	copy(state.CallData, ctx.data)

	return state, nil
}

////////////////////////////////////////////////////////////
// ct -> lfvm : helper functions

func convertCtCodeToLfvmCode(state *st.State) (Code, error) {
	const maxConversionCacheLength = 1024
	if getConversionCacheLength() > maxConversionCacheLength {
		clearConversionCache()
	}

	code := make([]byte, state.Code.Length())
	state.Code.CopyTo(code)
	address := common.Address{}
	return Convert(address, code, false, 0, false, false, state.Code.Hash())
}

func convertCtStatusToLfvmStatus(state *st.State) (Status, error) {
	switch state.Status {
	case st.Running:
		return RUNNING, nil
	case st.Stopped:
		return STOPPED, nil
	case st.Returned:
		return RETURNED, nil
	case st.Reverted:
		return REVERTED, nil
	case st.Failed:
		return ERROR, nil
	default:
		return ERROR, fmt.Errorf("unable to convert ct status %v to lfvm status", state.Status)
	}
}

func convertCtStackToLfvmStack(state *st.State) *Stack {
	stack := NewStack()
	for i := state.Stack.Size() - 1; i >= 0; i-- {
		val := state.Stack.Get(i).Uint256()
		stack.push(&val)
	}
	return stack
}

func convertCtMemoryToLfvmMemory(state *st.State) (*Memory, error) {
	data := state.Memory.Read(0, uint64(state.Memory.Size()))

	memory := NewMemory()
	memory.EnsureCapacityWithoutGas(uint64(len(data)), nil)
	err := memory.Set(0, uint64(len(data)), data)
	return memory, err
}

func convertCtRevisionToLfvmRevision(revision ct.Revision, ctx *context) error {
	switch revision {
	case ct.R07_Istanbul:
		// True by default in context.
	case ct.R09_Berlin:
		ctx.isBerlin = true
	case ct.R10_London:
		// London implies Berlin.
		ctx.isBerlin = true
		ctx.isLondon = true
	default:
		return fmt.Errorf("failed to convert revision: %v", revision)
	}
	return nil
}

////////////////////////////////////////////////////////////
// ct -> lfvm

func ConvertCtStateToLfvmContext(state *st.State, pcMap *PcMap) (*context, error) {
	// Special handling for unknown revision, because lfvm cannot represent an invalid revision.
	// So we mark the status as failed already.
	if state.Revision > ct.R10_London {
		state.Revision = ct.R10_London
		state.Status = st.Failed
	}

	// Create a dummy contract.
	objectAddress := (vm.AccountRef)(state.CallContext.AccountAddress[:])
	callerAddress := (vm.AccountRef)(state.CallContext.CallerAddress[:])
	contract := vm.NewContract(callerAddress, objectAddress, state.CallContext.Value.ToBigInt(), state.Gas)
	codeInBytes := make([]byte, state.Code.Length())
	state.Code.CopyTo(codeInBytes)
	contract.Code = codeInBytes

	pc, ok := pcMap.evmToLfvm[state.Pc]
	if !ok {
		return nil, fmt.Errorf("unable to convert evm pc %d to lfvm pc", state.Pc)
	}

	status, err := convertCtStatusToLfvmStatus(state)
	if err != nil {
		return nil, err
	}

	code, err := convertCtCodeToLfvmCode(state)
	if err != nil {
		return nil, err
	}

	memory, err := convertCtMemoryToLfvmMemory(state)
	if err != nil {
		return nil, err
	}

	data := make([]byte, len(state.CallData))
	copy(data, state.CallData)

	stateDb := utils.NewConformanceTestStateDb(state.Storage, state.Logs, state.Revision)

	stateDb.AddRefund(state.GasRefund)

	evm, err := getLfvmEvm(state, stateDb)
	if err != nil {
		return nil, err
	}

	// Create execution context.
	ctx := context{
		evm:      evm,
		pc:       int32(pc),
		stack:    convertCtStackToLfvmStack(state),
		memory:   memory,
		stateDB:  stateDb,
		status:   status,
		contract: contract,
		code:     code,
		data:     data,
		callsize: *uint256.NewInt(uint64(len(data))),
		readOnly: state.ReadOnly,
	}

	ctx.evm.Origin = (common.Address)(state.CallContext.OriginAddress[:])

	err = convertCtRevisionToLfvmRevision(state.Revision, &ctx)
	if err != nil {
		return nil, err
	}

	return &ctx, nil
}

func convertCtChainConfigToLfvm(state *st.State) *params.ChainConfig {
	return ct.GetChainConfig(state.BlockContext.ChainID.ToBigInt())
}

func convertCtBlockContextToLfvm(ctBlock st.BlockContext) (vm.BlockContext, vm.TxContext) {
	getHash := func(num uint64) common.Hash {
		return common.Hash{}
	}

	lfvmBlock := vm.BlockContext{
		BaseFee:     ctBlock.BaseFee.ToBigInt(),
		BlockNumber: big.NewInt(0).SetUint64(ctBlock.BlockNumber),
		Coinbase:    (common.Address)(ctBlock.CoinBase[:]),
		Difficulty:  ctBlock.Difficulty.ToBigInt(),
		GasLimit:    ctBlock.GasLimit,
		GetHash:     getHash,
		Time:        big.NewInt(0).SetUint64(ctBlock.TimeStamp),
	}
	lfvmTx := vm.TxContext{GasPrice: ctBlock.GasPrice.ToBigInt()}
	return lfvmBlock, lfvmTx
}

func getLfvmEvm(state *st.State, stateDB vm.StateDB) (*vm.EVM, error) {
	chainConfig := convertCtChainConfigToLfvm(state)

	blockCtx, txCtx := convertCtBlockContextToLfvm(state.BlockContext)
	txCtx.Origin = (common.Address)(state.CallContext.OriginAddress)

	return vm.NewEVM(blockCtx, txCtx, stateDB, chainConfig, vm.Config{}), nil
}
