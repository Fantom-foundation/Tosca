// Copyright (c) 2024 Fantom Foundation
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at fantom.foundation/bsl11.
//
// Change Date: 2028-4-16
//
// On the date above, in accordance with the Business Source License, use of
// this software will be governed by the GNU Lesser General Public License v3.

package lfvm

import (
	"github.com/Fantom-foundation/Tosca/go/tosca"
	"github.com/holiman/uint256"
)

const (
	CallNewAccountGas    tosca.Gas = 25000 // Paid for CALL when the destination address didn't exist prior.
	CallValueTransferGas tosca.Gas = 9000  // Paid for CALL when the value transfer is non-zero.
	CallStipend          tosca.Gas = 2300  // Free gas given at beginning of call.

	ColdSloadCostEIP2929         tosca.Gas = 2100 // Cost of cold SLOAD after EIP 2929
	ColdAccountAccessCostEIP2929 tosca.Gas = 2600 // Cost of cold account access after EIP 2929

	// CreateBySelfdestructGas is used when the refunded account is one that does
	// not exist. This logic is similar to call.
	// Introduced in Tangerine Whistle (Eip 150)
	CreateBySelfdestructGas tosca.Gas = 25000

	SelfdestructGasEIP150             tosca.Gas = 5000  // Gas cost of SELFDESTRUCT post EIP-150
	SelfdestructRefundGas             tosca.Gas = 24000 // Refunded following a selfdestruct operation.
	SloadGasEIP2200                   tosca.Gas = 800   // Cost of SLOAD after EIP 2200 (part of Istanbul)
	SstoreClearsScheduleRefundEIP2200 tosca.Gas = 15000 // Once per SSTORE operation for clearing an originally existing storage slot

	// SstoreClearsScheduleRefundEIP3529 is the refund for clearing a storage slot after EIP-3529.
	// In EIP-2200: SstoreResetGas was 5000.
	// In EIP-2929: SstoreResetGas was changed to '5000 - COLD_SLOAD_COST'.
	// In EIP-3529: SSTORE_CLEARS_SCHEDULE is defined as SSTORE_RESET_GAS + ACCESS_LIST_STORAGE_KEY_COST
	// Which becomes: 5000 - 2100 + 1900 = 4800
	SstoreClearsScheduleRefundEIP3529 tosca.Gas = 4800

	SstoreResetGasEIP2200      tosca.Gas = 5000  // Once per SSTORE operation from clean non-zero to something else
	SstoreSentryGasEIP2200     tosca.Gas = 2300  // Minimum gas required to be present for an SSTORE call, not consumed
	SstoreSetGasEIP2200        tosca.Gas = 20000 // Once per SSTORE operation from clean zero to non-zero
	WarmStorageReadCostEIP2929 tosca.Gas = 100   // Cost of reading warm storage after EIP 2929

	UNKNOWN_GAS_PRICE = 999999
)

var static_gas_prices = newOpCodePropertyMap(getStaticGasPriceInternal)
var static_gas_prices_berlin = newOpCodePropertyMap(getBerlinGasPriceInternal)

func getBerlinGasPriceInternal(op OpCode) tosca.Gas {
	gp := getStaticGasPriceInternal(op)

	// Changed static gas prices with EIP2929
	switch op {
	case SLOAD:
		gp = 0
	case EXTCODECOPY:
		gp = 100
	case EXTCODESIZE:
		gp = 100
	case EXTCODEHASH:
		gp = 100
	case BALANCE:
		gp = 100
	case CALL:
		gp = 0
	case CALLCODE:
		gp = 0
	case STATICCALL:
		gp = 0
	case DELEGATECALL:
		gp = 0
	case SELFDESTRUCT:
		gp = 5000
	}
	return gp
}

func getStaticGasPrices(revision tosca.Revision) *opCodePropertyMap[tosca.Gas] {
	if revision >= tosca.R09_Berlin {
		return &static_gas_prices_berlin
	}
	return &static_gas_prices
}

func getStaticGasPriceInternal(op OpCode) tosca.Gas {
	if PUSH1 <= op && op <= PUSH32 {
		return 3
	}
	if DUP1 <= op && op <= DUP16 {
		return 3
	}
	if SWAP1 <= op && op <= SWAP16 {
		return 3
	}
	// this range covers: LT, GT, SLT, SGT, EQ, ISZERO, AND, OR,
	// XOR, NOT, BYTE, SHL, SHR, SAR
	if LT <= op && op <= SAR {
		return 3
	}
	// this range covers: COINBASE, TIMESTAMP, NUMBER, DIFFICULTY/PREVRANDO,
	// 					GAS, GASLIMIT, CHAINID
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
	case BLOBHASH:
		return 3
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
	case TLOAD:
		return 100
	case TSTORE:
		return 100
	case PC:
		return 2
	case MSIZE:
		return 2
	case MCOPY:
		return 3
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
		return 700
	case CALLCODE:
		return 700
	case STATICCALL:
		return 700
	case RETURN:
		return 0
	case STOP:
		return 0
	case REVERT:
		return 0
	case INVALID:
		return 0
	case DELEGATECALL:
		return 700
	case SELFDESTRUCT:
		return 0 // should be 5000 according to evm.code
	}

	if op.isSuperInstruction() {
		var sum tosca.Gas
		for _, subOp := range op.decompose() {
			sum += getStaticGasPriceInternal(subOp)
		}
		return sum
	}

	return UNKNOWN_GAS_PRICE
}

// callGas returns the actual gas cost of the call.
//
// The cost of gas was changed during the homestead price change HF.
// As part of EIP 150 (TangerineWhistle), the returned gas is gas - base * 63 / 64.
func callGas(availableGas, base tosca.Gas, callCost *uint256.Int) tosca.Gas {
	availableGas = availableGas - base
	if availableGas < 0 {
		return base
	}
	gas := availableGas - availableGas/64
	if !callCost.IsUint64() || (gas < tosca.Gas(callCost.Uint64())) {
		return gas
	}
	return tosca.Gas(callCost.Uint64())
}

func getDynamicCostsForSstore(
	revision tosca.Revision,
	storageStatus tosca.StorageStatus,
) tosca.Gas {
	switch storageStatus {
	case tosca.StorageAdded:
		return 20000
	case tosca.StorageModified,
		tosca.StorageDeleted:
		if revision >= tosca.R09_Berlin {
			return 2900
		} else {
			return 5000
		}
	default:
		if revision >= tosca.R09_Berlin {
			return 100
		}
		return 800
	}
}

func getRefundForSstore(
	revision tosca.Revision,
	storageStatus tosca.StorageStatus,
) tosca.Gas {
	switch storageStatus {
	case tosca.StorageDeleted,
		tosca.StorageModifiedDeleted:
		if revision >= tosca.R10_London {
			return 4800
		}
		return 15000
	case tosca.StorageDeletedAdded:
		if revision >= tosca.R10_London {
			return -4800
		}
		return -15000
	case tosca.StorageDeletedRestored:
		if revision >= tosca.R10_London {
			return -4800 + 5000 - 2100 - 100
		} else if revision >= tosca.R09_Berlin {
			return -15000 + 5000 - 2100 - 100
		}
		return -15000 + 4200
	case tosca.StorageAddedDeleted:
		if revision >= tosca.R09_Berlin {
			return 19900
		}
		return 19200
	case tosca.StorageModifiedRestored:
		if revision >= tosca.R09_Berlin {
			return 5000 - 2100 - 100
		}
		return 4200
	default:
		return 0
	}
}

func gasEip2929AccountCheck(c *context, address tosca.Address) error {
	if c.isAtLeast(tosca.R09_Berlin) {
		// Charge extra for cold locations.
		//lint:ignore SA1019 deprecated functions to be migrated in #616
		if !c.context.IsAddressInAccessList(address) {
			if !c.useGas(ColdAccountAccessCostEIP2929 - WarmStorageReadCostEIP2929) {
				return errOutOfGas
			}
			c.context.AccessAccount(address)
		}
	}
	return nil
}
