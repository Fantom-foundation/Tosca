package common

import (
	"fmt"
	"math"
	"math/big"

	"github.com/ethereum/evmc/v10/bindings/go/evmc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
)

// Instantiates an interpreter with the given evm and config. `implementation`
// is a filepath to the shared library with the wanted interpreter
// implementation (e.g. libevmone.so).
func NewEVMCInterpreter(implementation string, evm *vm.EVM, cfg vm.Config) *EVMCInterpreter {
	vm, err := evmc.Load(implementation)
	if err != nil {
		panic(fmt.Sprintf("Could not create %s instance %s", implementation, err))
	}

	return &EVMCInterpreter{
		evmc: vm,
		evm:  evm,
		cfg:  cfg,
	}
}

type EVMCInterpreter struct {
	evmc *evmc.VM
	evm  *vm.EVM
	cfg  vm.Config
}

func (e *EVMCInterpreter) Run(contract *vm.Contract, input []byte, readOnly bool) (ret []byte, err error) {
	if contract.Gas > math.MaxInt64 {
		panic(fmt.Sprintf("Gas too big %v for int64", contract.Gas))
	}

	host_ctx := HostContext{
		interpreter: e,
		contract:    contract,
	}

	value, err := bigIntToHash(contract.Value())
	if err != nil {
		panic(fmt.Sprintf("Could not convert value: %v", err))
	}

	// TODO: Not all parameters here are correct. More information needs be
	// extracted from the contract and interpreter.
	output, _, err := e.evmc.Execute(&host_ctx, evmc.London, evmc.Call, false, e.evm.Depth, int64(contract.Gas), evmc.Address(contract.Address()), evmc.Address(contract.CallerAddress), input, value, contract.Code)

	return output, err
}

// The HostContext allows a non-Go EVM implementation to access the StateDB and
// other systems external to the interpreter. This implementation leverages
// evmc's Go bindings.
type HostContext struct {
	interpreter *EVMCInterpreter
	contract    *vm.Contract
}

func (ctx *HostContext) AccountExists(addr evmc.Address) bool {
	return ctx.interpreter.evm.StateDB.Exist((common.Address)(addr))
}

func (ctx *HostContext) GetStorage(addr evmc.Address, key evmc.Hash) evmc.Hash {
	return evmc.Hash(ctx.interpreter.evm.StateDB.GetState((common.Address)(addr), (common.Hash)(key)))
}

func (ctx *HostContext) SetStorage(addr evmc.Address, key evmc.Hash, value evmc.Hash) evmc.StorageStatus {
	ctx.interpreter.evm.StateDB.SetState((common.Address)(addr), (common.Hash)(key), (common.Hash)(value))

	// TODO: SetState does not report back the storage status. Assuming
	// StorageAdded for now.
	return evmc.StorageAdded
}

func (ctx *HostContext) GetBalance(addr evmc.Address) evmc.Hash {
	balance := ctx.interpreter.evm.StateDB.GetBalance((common.Address)(addr))
	result, err := bigIntToHash(balance)
	if err != nil {
		panic(fmt.Sprintf("Could not convert balance: %v", err))
	}
	return result
}

func (ctx *HostContext) GetCodeSize(addr evmc.Address) int {
	return ctx.interpreter.evm.StateDB.GetCodeSize((common.Address)(addr))
}

func (ctx *HostContext) GetCodeHash(addr evmc.Address) evmc.Hash {
	return evmc.Hash(ctx.interpreter.evm.StateDB.GetCodeHash((common.Address)(addr)))
}

func (ctx *HostContext) GetCode(addr evmc.Address) []byte {
	return ctx.interpreter.evm.StateDB.GetCode((common.Address)(addr))
}

func (ctx *HostContext) Selfdestruct(addr evmc.Address, beneficiary evmc.Address) bool {
	return ctx.interpreter.evm.StateDB.Suicide((common.Address)(addr))
}

func (ctx *HostContext) GetTxContext() evmc.TxContext {
	gasPrice, err := bigIntToHash(ctx.interpreter.evm.TxContext.GasPrice)
	if err != nil {
		panic(fmt.Sprintf("Could not convert gas price: %v", err))
	}

	gasLimit := ctx.interpreter.evm.Context.GasLimit
	if gasLimit > math.MaxInt64 {
		panic(fmt.Sprintf("Cannot use gas limit %v, too big for int64", gasLimit))
	}

	// BaseFee can be assumed zero unless set.
	var baseFee evmc.Hash
	if ctx.interpreter.evm.Context.BaseFee != nil {
		baseFee, err = bigIntToHash(ctx.interpreter.evm.Context.BaseFee)
		if err != nil {
			panic(fmt.Sprintf("Could not convert base fee: %v", err))
		}
	}

	return evmc.TxContext{
		GasPrice:   gasPrice,
		Origin:     evmc.Address(ctx.interpreter.evm.Origin),
		Coinbase:   evmc.Address(ctx.interpreter.evm.Context.Coinbase),
		Number:     ctx.interpreter.evm.Context.BlockNumber.Int64(),
		Timestamp:  ctx.interpreter.evm.Context.Time.Int64(),
		GasLimit:   int64(gasLimit),
		PrevRandao: evmc.Hash{}, // TODO: No idea where to get this from.
		ChainID:    evmc.Hash(ctx.interpreter.evm.ChainConfig().EIP150Hash),
		BaseFee:    baseFee,
	}
}

func (ctx *HostContext) GetBlockHash(number int64) evmc.Hash {
	return evmc.Hash(ctx.interpreter.evm.Context.GetHash(uint64(number)))
}

func (ctx *HostContext) EmitLog(addr evmc.Address, topics_in []evmc.Hash, data []byte) {
	topics := make([]common.Hash, len(topics_in))
	for i := range topics {
		topics[i] = (common.Hash)(topics_in[i])
	}

	ctx.interpreter.evm.StateDB.AddLog(&types.Log{
		Address: (common.Address)(addr),
		Topics:  ([]common.Hash)(topics),
		Data:    data,
		// This is a non-consensus field, but assigned here because
		// core/state doesn't know the current block number.
		BlockNumber: ctx.interpreter.evm.Context.BlockNumber.Uint64(),
	})
}

func (ctx *HostContext) Call(kind evmc.CallKind, recipient evmc.Address, sender evmc.Address, value evmc.Hash, input []byte, gas int64, depth int, static bool, salt evmc.Hash, codeAddress evmc.Address) (output []byte, gasLeft int64, gasRefund int64, createAddr evmc.Address, err error) {
	// TODO
	return
}

func (ctx *HostContext) AccessAccount(addr evmc.Address) evmc.AccessStatus {
	// TODO
	return evmc.ColdAccess
}

func (ctx *HostContext) AccessStorage(addr evmc.Address, key evmc.Hash) evmc.AccessStatus {
	// TODO
	return evmc.ColdAccess
}

func bigIntToHash(value *big.Int) (result evmc.Hash, err error) {
	if value == nil {
		return result, fmt.Errorf("unable to convert nil to Hash")
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
