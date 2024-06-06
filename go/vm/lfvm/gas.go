//
// Copyright (c) 2024 Fantom Foundation
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at fantom.foundation/bsl11.
//
// Change Date: 2028-4-16
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by the GNU Lesser General Public Licence v3
//

package lfvm

import (
	"errors"
	"fmt"

	"github.com/ethereum/go-ethereum/params" // < TODO: remove
	"github.com/holiman/uint256"

	"github.com/Fantom-foundation/Tosca/go/vm"
)

const UNKNOWN_GAS_PRICE = 999999

var static_gas_prices = [NUM_OPCODES]vm.Gas{}
var static_gas_prices_berlin = [NUM_OPCODES]vm.Gas{}

func init() {
	var gp vm.Gas
	for i := 0; i < int(NUM_EXECUTABLE_OPCODES); i++ {
		gp = getStaticGasPriceInternal(OpCode(i))
		static_gas_prices[i] = gp
		static_gas_prices_berlin[i] = gp
	}
	initBerlinGasPrice()

	for i := 0; i < int(NUM_EXECUTABLE_OPCODES); i++ {
		if static_gas_prices[i] == UNKNOWN_GAS_PRICE {
			panic(fmt.Sprintf("Gas price for %v is unknown", OpCode(i)))
		}
		if static_gas_prices_berlin[i] == UNKNOWN_GAS_PRICE {
			panic(fmt.Sprintf("Berlin gas price for %v is unknown", OpCode(i)))
		}
	}
}

func initBerlinGasPrice() {
	// Changed static gas prices with EIP2929
	static_gas_prices_berlin[SLOAD] = 0
	static_gas_prices_berlin[EXTCODECOPY] = 100
	static_gas_prices_berlin[EXTCODESIZE] = 100
	static_gas_prices_berlin[EXTCODEHASH] = 100
	static_gas_prices_berlin[BALANCE] = 100
	static_gas_prices_berlin[CALL] = 100
	static_gas_prices_berlin[CALLCODE] = 100
	static_gas_prices_berlin[STATICCALL] = 100
	static_gas_prices_berlin[DELEGATECALL] = 100
	static_gas_prices_berlin[SELFDESTRUCT] = 5000
}

func getStaticGasPrices(isBerlin bool) []vm.Gas {
	if isBerlin {
		return static_gas_prices_berlin[:]
	}
	return static_gas_prices[:]
}

func getStaticGasPriceInternal(op OpCode) vm.Gas {
	price := getStaticGasPriceInternal
	if PUSH1 <= op && op <= PUSH32 {
		return 3
	}
	if DUP1 <= op && op <= DUP16 {
		return 3
	}
	if SWAP1 <= op && op <= SWAP16 {
		return 3
	}
	if LT <= op && op <= SAR {
		return 3
	}
	if COINBASE <= op && op <= CHAINID {
		return 2
	}
	switch op {
	case POP:
		return 2
	case PUSH0:
		return 2
	case ADD:
		return 3
	case SUB:
		return 3
	case MUL:
		return 5
	case DIV:
		return 5
	case SDIV:
		return 5
	case MOD:
		return 5
	case SMOD:
		return 5
	case ADDMOD:
		return 8
	case MULMOD:
		return 8
	case EXP:
		return 10
	case SIGNEXTEND:
		return 5
	case SHA3:
		return 30
	case ADDRESS:
		return 2
	case BALANCE:
		return 700 // Should be 100 for warm access, 2600 for cold access
	case ORIGIN:
		return 2
	case CALLER:
		return 2
	case CALLVALUE:
		return 2
	case CALLDATALOAD:
		return 3
	case CALLDATASIZE:
		return 2
	case CALLDATACOPY:
		return 3
	case CODESIZE:
		return 2
	case CODECOPY:
		return 3
	case GASPRICE:
		return 2
	case EXTCODESIZE:
		return 700 // This seems to be different than documented on evm.codes (it should be 100)
	case EXTCODECOPY:
		return 700 // From EIP150 it is 700, was 20
	case RETURNDATASIZE:
		return 2
	case RETURNDATACOPY:
		return 3
	case EXTCODEHASH:
		return 700 // Should be 100 for warm access, 2600 for cold access
	case BLOCKHASH:
		return 20
	case SELFBALANCE:
		return 5
	case BASEFEE:
		return 2
	case BLOBBASEFEE:
		return 2
	case MLOAD:
		return 3
	case MSTORE:
		return 3
	case MSTORE8:
		return 3
	case SLOAD:
		return 800 // This is supposed to be 100 for warm and 2100 for cold accesses
	case SSTORE:
		return 0 // Costs are handled in gasSStore(..) function below
	case JUMP:
		return 8
	case JUMPI:
		return 10
	case JUMPDEST:
		return 1
	case JUMP_TO:
		return 0
	case PC:
		return 2
	case MSIZE:
		return 2
	case GAS:
		return 2
	case LOG0:
		return 375
	case LOG1:
		return 750
	case LOG2:
		return 1125
	case LOG3:
		return 1500
	case LOG4:
		return 1875
	case CREATE:
		return 32000
	case CREATE2:
		return 32000
	case CALL:
		return 700 // Should be 100 according to evm.code
	case CALLCODE:
		return 700
	case STATICCALL:
		return 700 // Should be 100 according to evm.code
	case RETURN:
		return 0
	case STOP:
		return 0
	case REVERT:
		return 0
	case INVALID:
		return 0
	case DELEGATECALL:
		return 700 // Should be 100 according to evm.code
	case SELFDESTRUCT:
		return 0 // should be 5000 according to evm.code

		// --- Super instructions ---
	case PUSH1_ADD:
		return price(PUSH1) + price(ADD)
	case PUSH1_SHL:
		return price(PUSH1) + price(SHL)
	case PUSH1_DUP1:
		return price(PUSH1) + price(DUP1)
	case PUSH2_JUMP:
		return price(PUSH2) + price(JUMP)
	case PUSH2_JUMPI:
		return price(PUSH2) + price(JUMPI)
	case SWAP1_POP:
		return price(SWAP1) + price(POP)
	case SWAP2_POP:
		return price(SWAP2) + price(POP)
	case DUP2_MSTORE:
		return price(DUP2) + price(MSTORE)
	case DUP2_LT:
		return price(DUP2) + price(LT)
	case POP_JUMP:
		return price(POP) + price(JUMP)
	case POP_POP:
		return price(POP) + price(POP)
	case SWAP2_SWAP1:
		return price(SWAP2) + price(SWAP1)
	case PUSH1_PUSH1:
		return price(PUSH1) + price(PUSH1)
	case ISZERO_PUSH2_JUMPI:
		return price(ISZERO) + price(PUSH2) + price(JUMPI)
	case PUSH1_PUSH4_DUP3:
		return price(PUSH1) + price(PUSH4) + price(DUP3)
	case SWAP2_SWAP1_POP_JUMP:
		return price(SWAP2) + price(SWAP1) + price(POP) + price(JUMP)
	case SWAP1_POP_SWAP2_SWAP1:
		return price(SWAP1) + price(POP) + price(SWAP2) + price(SWAP1)
	case POP_SWAP2_SWAP1_POP:
		return price(POP) + price(SWAP2) + price(SWAP1) + price(POP)
	case AND_SWAP1_POP_SWAP2_SWAP1:
		return price(AND) + price(SWAP1) + price(POP) + price(SWAP2) + price(SWAP1)
	case PUSH1_PUSH1_PUSH1_SHL_SUB:
		return 3*price(PUSH1) + price(SHL) + price(SUB)
	}

	return UNKNOWN_GAS_PRICE
}

// callGas returns the actual gas cost of the call.
//
// The cost of gas was changed during the homestead price change HF.
// As part of EIP 150 (TangerineWhistle), the returned gas is gas - base * 63 / 64.
func callGas(availableGas, base vm.Gas, callCost *uint256.Int) vm.Gas {
	availableGas = availableGas - base
	if availableGas < 0 {
		return base
	}
	gas := availableGas - availableGas/64
	if !callCost.IsUint64() || (gas < vm.Gas(callCost.Uint64())) {
		return gas
	}
	return vm.Gas(callCost.Uint64())
}

// Computes the costs for an SSTORE operation
func gasSStore(c *context) (vm.Gas, error) {
	return gasSStoreEIP2200(c)
}

//  0. If *gasleft* is less than or equal to 2300, fail the current call.
//  1. If current value equals new value (this is a no-op), SLOAD_GAS is deducted.
//  2. If current value does not equal new value:
//     2.1. If original value equals current value (this storage slot has not been changed by the current execution context):
//     2.1.1. If original value is 0, SSTORE_SET_GAS (20K) gas is deducted.
//     2.1.2. Otherwise, SSTORE_RESET_GAS gas is deducted. If new value is 0, add SSTORE_CLEARS_SCHEDULE to refund counter.
//     2.2. If original value does not equal current value (this storage slot is dirty), SLOAD_GAS gas is deducted. Apply both of the following clauses:
//     2.2.1. If original value is not 0:
//     2.2.1.1. If current value is 0 (also means that new value is not 0), subtract SSTORE_CLEARS_SCHEDULE gas from refund counter.
//     2.2.1.2. If new value is 0 (also means that current value is not 0), add SSTORE_CLEARS_SCHEDULE gas to refund counter.
//     2.2.2. If original value equals new value (this storage slot is reset):
//     2.2.2.1. If original value is 0, add SSTORE_SET_GAS - SLOAD_GAS to refund counter.
//     2.2.2.2. Otherwise, add SSTORE_RESET_GAS - SLOAD_GAS gas to refund counter.
func gasSStoreEIP2200(c *context) (vm.Gas, error) {
	// If we fail the minimum gas availability invariant, fail (0)
	if c.gas <= vm.Gas(params.SstoreSentryGasEIP2200) {
		c.status = OUT_OF_GAS
		return 0, errors.New("not enough gas for reentrancy sentry")
	}
	// Gas sentry honoured, do the actual gas calculation based on the stored value
	var (
		zero    = vm.Word{}
		key     = vm.Key(c.stack.Back(0).Bytes32())
		value   = vm.Word(c.stack.Back(1).Bytes32())
		current = c.context.GetStorage(c.params.Recipient, key)
	)

	if current == value { // noop (1)
		return vm.Gas(params.SloadGasEIP2200), nil
	}
	original := c.context.GetCommittedStorage(c.params.Recipient, key)
	if original == current {
		if original == zero { // create slot (2.1.1)
			return vm.Gas(params.SstoreSetGasEIP2200), nil
		}
		if value == zero { // delete slot (2.1.2b)
			c.refund += vm.Gas(params.SstoreClearsScheduleRefundEIP2200)
		}
		return vm.Gas(params.SstoreResetGasEIP2200), nil // write existing slot (2.1.2)
	}
	if original != zero {
		if current == zero { // recreate slot (2.2.1.1)
			c.refund -= vm.Gas(params.SstoreClearsScheduleRefundEIP2200)
		} else if value == zero { // delete slot (2.2.1.2)
			c.refund += vm.Gas(params.SstoreClearsScheduleRefundEIP2200)
		}
	}
	if original == value {
		if original == zero { // reset to original inexistent slot (2.2.2.1)
			c.refund += vm.Gas(params.SstoreSetGasEIP2200 - params.SloadGasEIP2200)
		} else { // reset to original existing slot (2.2.2.2)
			c.refund += vm.Gas(params.SstoreResetGasEIP2200 - params.SloadGasEIP2200)
		}
	}
	return vm.Gas(params.SloadGasEIP2200), nil // dirty update (2.2)
}

func gasSStoreEIP2929(c *context) (vm.Gas, error) {

	clearingRefund := vm.Gas(params.SstoreClearsScheduleRefundEIP2200)
	if c.isLondon() {
		clearingRefund = vm.Gas(params.SstoreClearsScheduleRefundEIP3529)
	}

	// If we fail the minimum gas availability invariant, fail (0)
	if c.gas <= vm.Gas(params.SstoreSentryGasEIP2200) {
		c.status = OUT_OF_GAS
		return 0, errors.New("not enough gas for reentrancy sentry")
	}
	// Gas sentry honoured, do the actual gas calculation based on the stored value
	var (
		zero    = vm.Word{}
		y, x    = c.stack.Back(1), c.stack.peek()
		slot    = vm.Key(x.Bytes32())
		current = c.context.GetStorage(c.params.Recipient, slot)
		cost    = vm.Gas(0)
	)
	// Check slot presence in the access list
	if addrPresent, slotPresent := c.context.IsSlotInAccessList(c.params.Recipient, slot); !slotPresent {
		if !addrPresent {
			c.status = ERROR
			return 0, errors.New("address was not present in access list during sstore op")
		}
		cost = vm.Gas(params.ColdSloadCostEIP2929)
		// If the caller cannot afford the cost, this change will be rolled back
		c.context.AccessStorage(c.params.Recipient, slot)
	}
	value := vm.Word(y.Bytes32())

	if current == value { // noop (1)
		return cost + vm.Gas(params.WarmStorageReadCostEIP2929), nil // SLOAD_GAS
	}
	original := c.context.GetCommittedStorage(c.params.Recipient, slot)
	if original == current {
		if original == zero { // create slot (2.1.1)
			return cost + vm.Gas(params.SstoreSetGasEIP2200), nil
		}
		if value == zero { // delete slot (2.1.2b)
			c.refund += clearingRefund
		}
		return cost + vm.Gas(params.SstoreResetGasEIP2200-params.ColdSloadCostEIP2929), nil // write existing slot (2.1.2)
	}
	if original != zero {
		if current == zero { // recreate slot (2.2.1.1)
			c.refund -= clearingRefund
		} else if value == zero { // delete slot (2.2.1.2)
			c.refund += clearingRefund
		}
	}
	if original == value {
		if original == zero { // reset to original inexistent slot (2.2.2.1)
			c.refund += vm.Gas(params.SstoreSetGasEIP2200 - params.WarmStorageReadCostEIP2929)
		} else { // reset to original existing slot (2.2.2.2)
			c.refund += vm.Gas((params.SstoreResetGasEIP2200 - params.ColdSloadCostEIP2929) - params.WarmStorageReadCostEIP2929)
		}
	}
	return cost + vm.Gas(params.WarmStorageReadCostEIP2929), nil // dirty update (2.2)
}

func gasEip2929AccountCheck(c *context, address vm.Address) error {
	if c.isBerlin() {
		// Charge extra for cold locations.
		if !c.context.IsAddressInAccessList(address) {
			if !c.UseGas(vm.Gas(params.ColdAccountAccessCostEIP2929 - params.WarmStorageReadCostEIP2929)) {
				return errOutOfGas
			}
			c.context.AccessAccount(address)
		}
	}
	return nil
}

func addressInAccessList(c *context) (warmAccess bool, coldCost vm.Gas, err error) {
	warmAccess = true
	if c.isBerlin() {
		addr := vm.Address(c.stack.Back(1).Bytes20())
		// Check slot presence in the access list
		warmAccess = c.context.IsAddressInAccessList(addr)
		// The WarmStorageReadCostEIP2929 (100) is already deducted in the form of a constant cost, so
		// the cost to charge for cold access, if any, is Cold - Warm
		coldCost = vm.Gas(params.ColdAccountAccessCostEIP2929 - params.WarmStorageReadCostEIP2929)
		if !warmAccess {
			c.context.AccessAccount(addr)
			// Charge the remaining difference here already, to correctly calculate available
			// gas for call
			if !c.UseGas(coldCost) {
				return false, 0, errOutOfGas
			}
		}
	}
	return warmAccess, coldCost, nil
}

func gasSelfdestruct(c *context) vm.Gas {
	gas := vm.Gas(params.SelfdestructGasEIP150)
	var address = vm.Address(c.stack.Back(0).Bytes20())

	// if beneficiary needs to be created
	if !c.context.AccountExists(address) && c.context.GetBalance(c.params.Recipient) != (vm.Value{}) {
		gas += vm.Gas(params.CreateBySelfdestructGas)
	}
	if !c.context.HasSelfDestructed(c.params.Recipient) {
		c.refund += vm.Gas(params.SelfdestructRefundGas)
	}
	return gas
}

func gasSelfdestructEIP2929(c *context) vm.Gas {
	var (
		gas     vm.Gas
		address = vm.Address(c.stack.Back(0).Bytes20())
	)
	if !c.context.IsAddressInAccessList(address) {
		// If the caller cannot afford the cost, this change will be rolled back
		c.context.AccessAccount(address)
		gas = vm.Gas(params.ColdAccountAccessCostEIP2929)
	}
	// if empty and transfers value
	if !c.context.AccountExists(address) && c.context.GetBalance(c.params.Recipient) != (vm.Value{}) {
		gas += vm.Gas(params.CreateBySelfdestructGas)
	}
	// do this only for Berlin and not after London fork
	if c.isBerlin() && !c.isLondon() {
		if !c.context.HasSelfDestructed(c.params.Recipient) {
			c.refund += vm.Gas(params.SelfdestructRefundGas)
		}
	}
	return gas
}
