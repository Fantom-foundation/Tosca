// Copyright (c) 2024 Fantom Foundation
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at fantom.foundation/bsl11.
//
// Change Date: 2028-4-16
//
// On the date above, in accordance with the Business Source License, use of
// this software will be governed by the GNU Lesser General Public License v3.

package floria

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"strings"

	"github.com/Fantom-foundation/Tosca/go/tosca"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	geth "github.com/ethereum/go-ethereum/core/vm"
)

func handlePrecompiledContract(revision tosca.Revision, input tosca.Data, address tosca.Address, gas tosca.Gas) (tosca.CallResult, bool) {
	contract, ok := getPrecompiledContract(address, revision)
	if !ok {
		return tosca.CallResult{}, false
	}
	gasCost := contract.RequiredGas(input)
	if gas < tosca.Gas(gasCost) {
		return tosca.CallResult{}, true
	}
	gas -= tosca.Gas(gasCost)
	output, err := contract.Run(input)

	return tosca.CallResult{
		Success: err == nil, // precompiled contracts only return errors on invalid input
		Output:  output,
		GasLeft: gas,
	}, true
}

func getPrecompiledContract(address tosca.Address, revision tosca.Revision) (geth.PrecompiledContract, bool) {
	var precompiles map[common.Address]geth.PrecompiledContract
	switch revision {
	case tosca.R13_Cancun:
		precompiles = geth.PrecompiledContractsCancun
	case tosca.R12_Shanghai, tosca.R11_Paris, tosca.R10_London, tosca.R09_Berlin:
		precompiles = geth.PrecompiledContractsBerlin
	default: // Istanbul is the oldest supported revision supported by Sonic
		precompiles = geth.PrecompiledContractsIstanbul
	}
	contract, ok := precompiles[common.Address(address)]
	return contract, ok
}

////////////////////////////////////////////////////////////////////////////////
// State precompiled contract

// DriverAddress is the NodeDriver contract address
var DriverAddress = tosca.Address(common.HexToAddress("0xd100a01e00000000000000000000000000000000"))

// StateContractAddress is the EvmWriter pre-compiled contract address
var StateContractAddress = tosca.Address(common.HexToAddress("0xd100ec0000000000000000000000000000000000"))

// stateContractABI is the input ABI used to generate the binding from
var stateContractABI string = "[{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"num\",\"type\":\"uint256\"}],\"name\":\"AdvanceEpochs\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"bytes\",\"name\":\"diff\",\"type\":\"bytes\"}],\"name\":\"UpdateNetworkRules\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"version\",\"type\":\"uint256\"}],\"name\":\"UpdateNetworkVersion\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"uint256\",\"name\":\"validatorID\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"bytes\",\"name\":\"pubkey\",\"type\":\"bytes\"}],\"name\":\"UpdateValidatorPubkey\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"uint256\",\"name\":\"validatorID\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"weight\",\"type\":\"uint256\"}],\"name\":\"UpdateValidatorWeight\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"backend\",\"type\":\"address\"}],\"name\":\"UpdatedBackend\",\"type\":\"event\"},{\"constant\":false,\"inputs\":[{\"internalType\":\"address\",\"name\":\"_backend\",\"type\":\"address\"}],\"name\":\"setBackend\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"internalType\":\"address\",\"name\":\"_backend\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"_evmWriterAddress\",\"type\":\"address\"}],\"name\":\"initialize\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"internalType\":\"address\",\"name\":\"acc\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"value\",\"type\":\"uint256\"}],\"name\":\"setBalance\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"internalType\":\"address\",\"name\":\"acc\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"from\",\"type\":\"address\"}],\"name\":\"copyCode\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"internalType\":\"address\",\"name\":\"acc\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"with\",\"type\":\"address\"}],\"name\":\"swapCode\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"internalType\":\"address\",\"name\":\"acc\",\"type\":\"address\"},{\"internalType\":\"bytes32\",\"name\":\"key\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"value\",\"type\":\"bytes32\"}],\"name\":\"setStorage\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"internalType\":\"address\",\"name\":\"acc\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"diff\",\"type\":\"uint256\"}],\"name\":\"incNonce\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"internalType\":\"bytes\",\"name\":\"diff\",\"type\":\"bytes\"}],\"name\":\"updateNetworkRules\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"version\",\"type\":\"uint256\"}],\"name\":\"updateNetworkVersion\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"num\",\"type\":\"uint256\"}],\"name\":\"advanceEpochs\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"validatorID\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"value\",\"type\":\"uint256\"}],\"name\":\"updateValidatorWeight\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"validatorID\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"pubkey\",\"type\":\"bytes\"}],\"name\":\"updateValidatorPubkey\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"internalType\":\"address\",\"name\":\"_auth\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"validatorID\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"pubkey\",\"type\":\"bytes\"},{\"internalType\":\"uint256\",\"name\":\"status\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"createdEpoch\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"createdTime\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"deactivatedEpoch\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"deactivatedTime\",\"type\":\"uint256\"}],\"name\":\"setGenesisValidator\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"internalType\":\"address\",\"name\":\"delegator\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"toValidatorID\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"stake\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"lockedStake\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"lockupFromEpoch\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"lockupEndTime\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"lockupDuration\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"earlyUnlockPenalty\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"rewards\",\"type\":\"uint256\"}],\"name\":\"setGenesisDelegation\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"validatorID\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"status\",\"type\":\"uint256\"}],\"name\":\"deactivateValidator\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"internalType\":\"uint256[]\",\"name\":\"nextValidatorIDs\",\"type\":\"uint256[]\"}],\"name\":\"sealEpochValidators\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"internalType\":\"uint256[]\",\"name\":\"offlineTimes\",\"type\":\"uint256[]\"},{\"internalType\":\"uint256[]\",\"name\":\"offlineBlocks\",\"type\":\"uint256[]\"},{\"internalType\":\"uint256[]\",\"name\":\"uptimes\",\"type\":\"uint256[]\"},{\"internalType\":\"uint256[]\",\"name\":\"originatedTxsFee\",\"type\":\"uint256[]\"}],\"name\":\"sealEpoch\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"internalType\":\"uint256[]\",\"name\":\"offlineTimes\",\"type\":\"uint256[]\"},{\"internalType\":\"uint256[]\",\"name\":\"offlineBlocks\",\"type\":\"uint256[]\"},{\"internalType\":\"uint256[]\",\"name\":\"uptimes\",\"type\":\"uint256[]\"},{\"internalType\":\"uint256[]\",\"name\":\"originatedTxsFee\",\"type\":\"uint256[]\"},{\"internalType\":\"uint256\",\"name\":\"usedGas\",\"type\":\"uint256\"}],\"name\":\"sealEpochV1\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"}]"

var (
	setBalanceMethodID []byte
	copyCodeMethodID   []byte
	swapCodeMethodID   []byte
	setStorageMethodID []byte
	incNonceMethodID   []byte
)

func init() {
	abi, err := abi.JSON(strings.NewReader(stateContractABI))
	if err != nil {
		panic(err)
	}

	for name, constID := range map[string]*[]byte{
		"setBalance": &setBalanceMethodID,
		"copyCode":   &copyCodeMethodID,
		"swapCode":   &swapCodeMethodID,
		"setStorage": &setStorageMethodID,
		"incNonce":   &incNonceMethodID,
	} {
		method, exist := abi.Methods[name]
		if !exist {
			panic("unknown EvmWriter method")
		}

		*constID = make([]byte, len(method.ID))
		copy(*constID, method.ID)
	}
}

var ErrExecutionReverted = fmt.Errorf("execution reverted")
var ErrOutOfGas = fmt.Errorf("out of gas")

func handleStatePrecompiledContract(state tosca.WorldState, sender tosca.Address, address tosca.Address, input []byte, gas tosca.Gas) (tosca.CallResult, bool) {
	if address != StateContractAddress {
		return tosca.CallResult{}, false
	}
	gas, err := statePrecompiledContract(state, sender, input, gas)
	return tosca.CallResult{
		Success: err == nil,
		Output:  nil,
		GasLeft: tosca.Gas(gas),
	}, true
}

func statePrecompiledContract(state tosca.WorldState, sender tosca.Address, input []byte, gas tosca.Gas) (tosca.Gas, error) {
	if sender != DriverAddress {
		return 0, ErrExecutionReverted
	}
	if len(input) < 4 {
		return 0, ErrExecutionReverted
	}
	selector := input[:4]
	input = input[4:]
	if bytes.Equal(selector, setBalanceMethodID) {
		return stateSetBalance(state, sender, input, gas)
	} else if bytes.Equal(selector, copyCodeMethodID) {
		return stateCopyCode(state, input, gas)
	} else if bytes.Equal(selector, swapCodeMethodID) {
		return stateSwapCode(state, input, gas)
	} else if bytes.Equal(selector, setStorageMethodID) {
		return stateSetStorage(state, input, gas)
	} else if bytes.Equal(selector, incNonceMethodID) {
		return stateIncNonce(state, sender, input, gas)
	}

	return 0, fmt.Errorf("invalid method ID")
}

func stateSetBalance(state tosca.WorldState, sender tosca.Address, input []byte, gas tosca.Gas) (tosca.Gas, error) {
	if gas < CallValueTransferGas {
		return 0, ErrOutOfGas
	}
	gas -= CallValueTransferGas
	if len(input) != 64 {
		return 0, ErrExecutionReverted
	}

	account := tosca.Address(input[12:32])
	value := tosca.Value(input[32:64])

	if account == sender {
		// Origin balance shouldn't decrease during his transaction
		return 0, ErrExecutionReverted
	}

	state.SetBalance(account, value)
	return gas, nil
}

func stateCopyCode(state tosca.WorldState, input []byte, gas tosca.Gas) (tosca.Gas, error) {
	if gas < CreateGas {
		return 0, ErrOutOfGas
	}
	gas -= CreateGas
	if len(input) != 64 {
		return 0, ErrExecutionReverted
	}

	accountTo := tosca.Address(input[12:32])
	accountFrom := tosca.Address(input[32+12 : 32+32])

	code := state.GetCode(accountFrom)
	if code == nil {
		code = []byte{}
	}
	cost := tosca.Gas(len(code)) * (CreateDataGas + MemoryGas)
	if gas < cost {
		return 0, ErrOutOfGas
	}
	gas -= cost
	if accountFrom != accountTo {
		state.SetCode(accountTo, code)
	}

	return gas, nil
}

func stateSwapCode(state tosca.WorldState, input []byte, gas tosca.Gas) (tosca.Gas, error) {
	cost := 2 * CreateGas
	if gas < cost {
		return 0, ErrOutOfGas
	}
	gas -= cost
	if len(input) != 64 {
		return 0, ErrExecutionReverted
	}

	account0 := tosca.Address(input[12:32])
	account1 := tosca.Address(input[32+12 : 32+32])
	code0 := state.GetCode(account0)
	if code0 == nil {
		code0 = []byte{}
	}
	code1 := state.GetCode(account1)
	if code1 == nil {
		code1 = []byte{}
	}
	cost0 := tosca.Gas(len(code0)) * (CreateDataGas + MemoryGas)
	cost1 := tosca.Gas(len(code1)) * (CreateDataGas + MemoryGas)
	cost = (cost0 + cost1) / 2 // 50% discount because trie size won't increase after pruning
	if gas < cost {
		return 0, ErrOutOfGas
	}
	gas -= cost
	if account0 != account1 {
		state.SetCode(account0, code1)
		state.SetCode(account1, code0)
	}

	return gas, nil
}

func stateSetStorage(state tosca.WorldState, input []byte, gas tosca.Gas) (tosca.Gas, error) {
	if gas < SstoreSetGasEIP2200 {
		return 0, ErrOutOfGas
	}
	gas -= SstoreSetGasEIP2200
	if len(input) != 96 {
		return 0, ErrExecutionReverted
	}

	account := tosca.Address(input[12:32])
	key := tosca.Key(input[32:64])
	value := tosca.Word(input[64:96])

	state.SetStorage(account, key, value)

	return gas, nil
}

func stateIncNonce(state tosca.WorldState, sender tosca.Address, input []byte, gas tosca.Gas) (tosca.Gas, error) {
	if gas < CallValueTransferGas {
		return 0, ErrOutOfGas
	}
	gas -= CallValueTransferGas
	if len(input) != 64 {
		return 0, ErrExecutionReverted
	}

	account := tosca.Address(input[12:32])
	value := tosca.Value(input[32:64])
	valueUint := binary.LittleEndian.Uint64(value[:])

	if account == sender {
		// Origin nonce shouldn't change during his transaction
		return 0, ErrExecutionReverted
	}

	if valueUint <= 0 || valueUint >= 256 {
		// Don't allow large nonce increasing to prevent a nonce overflow
		return 0, ErrExecutionReverted
	}

	state.SetNonce(account, state.GetNonce(account)+valueUint)

	return gas, nil
}
