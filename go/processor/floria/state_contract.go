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
	"fmt"
	"math/big"
	"strings"

	"github.com/Fantom-foundation/Tosca/go/tosca"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
)

const (
	callValueTransferGas = tosca.Gas(9000)
	createGas            = tosca.Gas(32000)
	createDataGas        = tosca.Gas(200)
	memoryGas            = tosca.Gas(3)
	sStoreSetGasEIP2200  = tosca.Gas(20000)
)

// DriverAddress is the NodeDriver contract address
// It is wrapped in a function to be immutable
func DriverAddress() tosca.Address {
	return tosca.Address(common.HexToAddress("0xd100a01e00000000000000000000000000000000"))
}

// StateContractAddress is the EvmWriter pre-compiled contract address
// It is wrapped in a function to be immutable
func StateContractAddress() tosca.Address {
	return tosca.Address(common.HexToAddress("0xd100ec0000000000000000000000000000000000"))
}

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
		panic(fmt.Errorf("failed to parse stateContractABI: %w", err))
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

func isStateContract(address tosca.Address) bool {
	return address == StateContractAddress()
}

// handleStateContract is a reworked version of the original function from the Opera client.
// It is used to handle epochs and allows to set balance, copy code, swap code, set storage, and increment nonce.
// Source: https://github.com/Fantom-foundation/Sonic/blob/main/opera/contracts/evmwriter/evm_writer.go#L24
func handleStateContract(
	state tosca.WorldState,
	sender tosca.Address,
	receiver tosca.Address,
	input []byte,
	gas tosca.Gas,
) (tosca.CallResult, bool) {
	if receiver != StateContractAddress() {
		return tosca.CallResult{}, false
	}
	if sender != DriverAddress() {
		return tosca.CallResult{}, true
	}
	if len(input) < 4 {
		return tosca.CallResult{}, true
	}

	err := fmt.Errorf("invalid method ID")
	gasLeft := tosca.Gas(0)

	selector := input[:4]
	input = input[4:]
	if bytes.Equal(selector, setBalanceMethodID) {
		gasLeft, err = executeStateSetBalance(state, sender, input, gas)
	} else if bytes.Equal(selector, copyCodeMethodID) {
		gasLeft, err = executeStateContractCopyCode(state, input, gas)
	} else if bytes.Equal(selector, swapCodeMethodID) {
		gasLeft, err = executeStateContractSwapCode(state, input, gas)
	} else if bytes.Equal(selector, setStorageMethodID) {
		gasLeft, err = executeStateContractSetStorage(state, input, gas)
	} else if bytes.Equal(selector, incNonceMethodID) {
		gasLeft, err = executeStateContractIncNonce(state, sender, input, gas)
	}

	return tosca.CallResult{
		Success: err == nil,
		Output:  nil,
		GasLeft: gasLeft,
	}, true
}

func executeStateSetBalance(state tosca.WorldState, sender tosca.Address, input []byte, gas tosca.Gas) (tosca.Gas, error) {
	if gas < callValueTransferGas {
		return 0, ErrOutOfGas
	}
	gas -= callValueTransferGas
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

func executeStateContractCopyCode(state tosca.WorldState, input []byte, gas tosca.Gas) (tosca.Gas, error) {
	if gas < createGas {
		return 0, ErrOutOfGas
	}
	gas -= createGas
	if len(input) != 64 {
		return 0, ErrExecutionReverted
	}

	accountTo := tosca.Address(input[12:32])
	accountFrom := tosca.Address(input[32+12 : 32+32])

	code := state.GetCode(accountFrom)
	cost := tosca.Gas(len(code)) * (createDataGas + memoryGas)
	if gas < cost {
		return 0, ErrOutOfGas
	}
	gas -= cost
	if accountFrom != accountTo {
		state.SetCode(accountTo, code)
	}

	return gas, nil
}

func executeStateContractSwapCode(state tosca.WorldState, input []byte, gas tosca.Gas) (tosca.Gas, error) {
	cost := 2 * createGas
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
	code1 := state.GetCode(account1)

	cost0 := tosca.Gas(len(code0)) * (createDataGas + memoryGas)
	cost1 := tosca.Gas(len(code1)) * (createDataGas + memoryGas)
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

func executeStateContractSetStorage(state tosca.WorldState, input []byte, gas tosca.Gas) (tosca.Gas, error) {
	if gas < sStoreSetGasEIP2200 {
		return 0, ErrOutOfGas
	}
	gas -= sStoreSetGasEIP2200
	if len(input) != 96 {
		return 0, ErrExecutionReverted
	}

	account := tosca.Address(input[12:32])
	key := tosca.Key(input[32:64])
	value := tosca.Word(input[64:96])

	state.SetStorage(account, key, value)

	return gas, nil
}

func executeStateContractIncNonce(state tosca.WorldState, sender tosca.Address, input []byte, gas tosca.Gas) (tosca.Gas, error) {
	if gas < callValueTransferGas {
		return 0, ErrOutOfGas
	}
	gas -= callValueTransferGas
	if len(input) != 64 {
		return 0, ErrExecutionReverted
	}

	account := tosca.Address(input[12:32])
	value := tosca.Value(input[32:64])
	valueUint := big.NewInt(0).SetBytes(value[:]).Uint64()

	if account == sender {
		// Origin nonce shouldn't change during his transaction
		return 0, ErrExecutionReverted
	}

	if valueUint <= 0 || 256 <= valueUint {
		// Don't allow large nonce increasing to prevent a nonce overflow
		return 0, ErrExecutionReverted
	}

	state.SetNonce(account, state.GetNonce(account)+valueUint)

	return gas, nil
}
