package geth

import (
	"bytes"
	"fmt"
	"math/big"

	ct "github.com/Fantom-foundation/Tosca/go/ct/common"
	"github.com/Fantom-foundation/Tosca/go/ct/st"
	"github.com/Fantom-foundation/Tosca/go/ct/utils"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/params"
)

////////////////////////////////////////////////////////////
// geth -> ct : helper functions

func convertGethStatusToCtStatus(state *vm.GethState) (st.StatusCode, error) {
	if !state.Halted && state.Err == nil {
		return st.Running, nil
	}

	if state.Err == vm.ErrExecutionReverted {
		return st.Reverted, nil
	}

	if state.Err != nil {
		return st.Failed, nil
	}

	if state.Result != nil {
		return st.Returned, nil
	}

	if state.Halted {
		return st.Stopped, nil
	}

	return st.Failed, fmt.Errorf("unable to convert geth status to ct status")
}

func convertGethRevisionToCtRevision(geth *gethInterpreter) ct.Revision {
	if geth.isLondon() {
		return ct.R10_London
	} else if geth.isBerlin() {
		return ct.R09_Berlin
	} else if geth.isIstanbul() {
		return ct.R07_Istanbul
	}
	return ct.R99_UnknownNextRevision
}

func convertGethStackToCtStack(state *vm.GethState) *st.Stack {
	stack := st.NewStack()

	for i := 0; i < state.Stack.Len(); i++ {
		val := state.Stack.Data()[i]
		stack.Push(ct.NewU256(val[3], val[2], val[1], val[0]))
	}
	return stack
}

func convertGethMemoryToCtMemory(state *vm.GethState) *st.Memory {
	memory := st.NewMemory()
	memory.Set(state.Memory.Data())
	return memory
}

////////////////////////////////////////////////////////////
// geth -> ct

func convertGethToCtCallContext(geth *gethInterpreter, state *vm.GethState) st.CallContext {
	return st.CallContext{
		AccountAddress: (ct.Address)(state.Contract.Address()),
		OriginAddress:  (ct.Address)(geth.evm.Origin),
		CallerAddress:  (ct.Address)(state.Contract.CallerAddress),
		Value:          ct.NewU256FromBigInt(state.Contract.Value()),
	}
}

func convertGethToCtBlockContext(geth *gethInterpreter) st.BlockContext {
	return st.BlockContext{
		BaseFee:     ct.NewU256FromBigInt(geth.evm.Context.BaseFee),
		BlockNumber: geth.evm.Context.BlockNumber.Uint64(),
		ChainID:     ct.NewU256FromBigInt(geth.evm.ChainConfig().ChainID),
		CoinBase:    (ct.Address)(geth.evm.Context.Coinbase),
		GasLimit:    geth.evm.Context.GasLimit,
		GasPrice:    ct.NewU256FromBigInt(geth.evm.GasPrice),
		Difficulty:  ct.NewU256FromBigInt(geth.evm.Context.Difficulty),
		TimeStamp:   geth.evm.Context.Time.Uint64(),
	}
}

func ConvertGethToCtState(geth *gethInterpreter, state *vm.GethState) (*st.State, error) {
	status, err := convertGethStatusToCtStatus(state)
	if err != nil {
		return nil, err
	}

	revision := convertGethRevisionToCtRevision(geth)
	if revision == ct.R99_UnknownNextRevision {
		return nil, fmt.Errorf("unknown revision: %v", revision)
	}

	ctState := st.NewState(st.NewCode(state.Contract.Code))
	ctState.Status = status
	ctState.Revision = revision
	ctState.ReadOnly = state.ReadOnly
	ctState.Pc = uint16(state.Pc)
	ctState.Gas = state.Contract.Gas
	if geth.evm.StateDB != nil {
		ctState.GasRefund = geth.evm.StateDB.GetRefund()
	}
	ctState.Stack = convertGethStackToCtStack(state)
	ctState.Memory = convertGethMemoryToCtMemory(state)
	if geth.evm.StateDB != nil {
		ctState.Storage = geth.evm.StateDB.(*utils.ConformanceTestStateDb).Storage
		ctState.Balance = geth.evm.StateDB.(*utils.ConformanceTestStateDb).Balance
		ctState.Logs = geth.evm.StateDB.(*utils.ConformanceTestStateDb).Logs
	}

	ctState.CallContext = convertGethToCtCallContext(geth, state)
	ctState.BlockContext = convertGethToCtBlockContext(geth)

	ctState.CallData = bytes.Clone(state.Contract.Input)

	return ctState, nil
}

////////////////////////////////////////////////////////////
// ct -> geth : helper functions

func convertCtCodeToGethCode(state *st.State) []byte {
	code := make([]byte, state.Code.Length())
	state.Code.CopyTo(code)
	return code
}

func convertCtStatusToGethStatus(ctState *st.State, geth *gethInterpreter, state *vm.GethState) error {
	switch ctState.Status {
	case st.Running:
		state.Halted = false
		state.Err = nil
	case st.Stopped:
		state.Halted = true
		state.Err = nil
	case st.Returned:
		state.Halted = true
		state.Err = nil
		state.Result = make([]byte, 0)
	case st.Reverted:
		state.Halted = true
		state.Err = vm.ErrExecutionReverted
	case st.Failed:
		state.Halted = true
		state.Err = vm.ErrInvalidCode
	default:
		return fmt.Errorf("unable to convert ct status %v to geth status", ctState.Status)
	}
	return nil
}

func convertCtStackToGethStack(state *st.State) *vm.Stack {
	stack := vm.NewStack()
	for i := state.Stack.Size() - 1; i >= 0; i-- {
		val := state.Stack.Get(i).Uint256()
		stack.Push(&val)
	}
	return stack
}

func convertCtMemoryToGethMemory(state *st.State) *vm.Memory {
	data := state.Memory.Read(0, uint64(state.Memory.Size()))
	memory := vm.NewMemory()
	// Set internal memory gas cost state so future grow operations compute the correct cost.
	vm.MemoryGasCost(memory, uint64(len(data)))
	memory.Resize(uint64(len(data)))
	memory.Set(0, uint64(len(data)), data)
	return memory
}

////////////////////////////////////////////////////////////
// ct -> geth

type gethInterpreter struct {
	chainConfig *params.ChainConfig
	evm         *vm.EVM
	interpreter *vm.GethEVMInterpreter
}

func (g *gethInterpreter) isIstanbul() bool {
	blockNr := g.evm.Context.BlockNumber
	return g.chainConfig.IsIstanbul(blockNr)
}

func (g *gethInterpreter) isBerlin() bool {
	blockNr := g.evm.Context.BlockNumber
	return g.chainConfig.IsBerlin(blockNr)
}

func (g *gethInterpreter) isLondon() bool {
	blockNr := g.evm.Context.BlockNumber
	return g.chainConfig.IsLondon(blockNr)
}

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

func convertCtBlockContextToGeth(ctBlock st.BlockContext) (vm.BlockContext, vm.TxContext) {
	// Hashing function used in the context for BLOCKHASH instruction
	getHash := func(num uint64) common.Hash {
		return common.Hash{}
	}

	gethBlock := vm.BlockContext{
		CanTransfer: canTransferFunc,
		Transfer:    transferFunc,
		GetHash:     getHash,
		Coinbase:    (common.Address)(ctBlock.CoinBase[:]),
		GasLimit:    ctBlock.GasLimit,
		BlockNumber: big.NewInt(0).SetUint64(ctBlock.BlockNumber),
		Time:        big.NewInt(0).SetUint64(ctBlock.TimeStamp),
		Difficulty:  ctBlock.Difficulty.ToBigInt(),
		BaseFee:     ctBlock.BaseFee.ToBigInt(),
	}
	gethTx := vm.TxContext{GasPrice: ctBlock.GasPrice.ToBigInt()}
	return gethBlock, gethTx
}

func getGethEvm(state *st.State) (*gethInterpreter, error) {
	chainConfig := convertCtChainConfigToGeth(state)

	blockCtx, txCtx := convertCtBlockContextToGeth(state.BlockContext)
	txCtx.Origin = (common.Address)(state.CallContext.OriginAddress)

	// Set interpreter variant for this VM
	config := vm.Config{
		InterpreterImpl: "geth",
	}

	stateDb := utils.NewConformanceTestStateDb(state.Storage, state.Balance, state.Logs, state.Revision)
	stateDb.AddRefund(state.GasRefund)

	evm := vm.NewEVM(blockCtx, txCtx, stateDb, chainConfig, config)

	gethInt, ok := evm.Interpreter().(*vm.GethEVMInterpreter)
	if !ok {
		return nil, fmt.Errorf("unable to get geth interpreter")
	}

	return &gethInterpreter{
		chainConfig: chainConfig,
		evm:         evm,
		interpreter: gethInt,
	}, nil
}

func ConvertCtStateToGeth(state *st.State) (*gethInterpreter, *vm.GethState, error) {
	// Special handling for unknown revision, because geth cannot represent an invalid revision.
	// So we mark the status as failed already.
	if state.Revision > ct.R10_London {
		state.Revision = ct.R10_London
		state.Status = st.Failed
	}

	geth, err := getGethEvm(state)
	if err != nil {
		return nil, nil, err
	}

	objectAddress := (vm.AccountRef)(state.CallContext.AccountAddress)
	callerAddress := (vm.AccountRef)(state.CallContext.CallerAddress)
	contract := vm.NewContract(callerAddress, objectAddress, state.CallContext.Value.ToBigInt(), state.Gas)
	contract.Code = convertCtCodeToGethCode(state)
	contract.Input = bytes.Clone(state.CallData)

	interpreterState := vm.NewGethState(
		contract,
		convertCtMemoryToGethMemory(state),
		convertCtStackToGethStack(state),
		uint64(state.Pc))

	interpreterState.ReadOnly = state.ReadOnly

	if err = convertCtStatusToGethStatus(state, geth, interpreterState); err != nil {
		return nil, nil, err
	}

	return geth, interpreterState, nil
}

func convertCtChainConfigToGeth(state *st.State) *params.ChainConfig {
	return ct.GetChainConfig(state.BlockContext.ChainID.ToBigInt())
}
