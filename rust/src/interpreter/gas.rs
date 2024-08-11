use evmc_vm::{AccessStatus, Address, ExecutionContext, Revision, StatusCode, StepStatusCode};

use crate::{interpreter::word_size, types::u256};

#[inline(always)]
pub fn consume_gas<const GAS: u64>(gas_left: &mut u64) -> Result<(), (StepStatusCode, StatusCode)> {
    if *gas_left < GAS {
        return Err((
            StepStatusCode::EVMC_STEP_FAILED,
            StatusCode::EVMC_OUT_OF_GAS,
        ));
    }
    *gas_left -= GAS;
    Ok(())
}

#[inline(always)]
pub fn consume_dyn_gas(gas_left: &mut u64, gas: u64) -> Result<(), (StepStatusCode, StatusCode)> {
    if *gas_left < gas {
        return Err((
            StepStatusCode::EVMC_STEP_FAILED,
            StatusCode::EVMC_OUT_OF_GAS,
        ));
    }
    *gas_left -= gas;
    Ok(())
}

#[inline(always)]
pub fn consume_positive_value_cost(
    value: &u256,
    gas_left: &mut u64,
) -> Result<(), (StepStatusCode, StatusCode)> {
    if *value != u256::ZERO {
        consume_gas::<9000>(gas_left)?;
    }
    Ok(())
}

#[inline(always)]
pub fn consume_value_to_empty_account_cost(
    value: &u256,
    addr: &Address,
    context: &mut ExecutionContext,
    gas_left: &mut u64,
) -> Result<(), (StepStatusCode, StatusCode)> {
    if *value != u256::ZERO && !context.account_exists(&addr) {
        consume_gas::<25000>(gas_left)?;
    }
    Ok(())
}

#[inline(always)]
pub fn consume_address_access_cost(
    gas_left: &mut u64,
    addr: &Address,
    context: &mut ExecutionContext,
    revision: Revision,
) -> Result<(), (StepStatusCode, StatusCode)> {
    let tx_context = context.get_tx_context();
    if revision >= Revision::EVMC_BERLIN {
        if *addr != tx_context.tx_origin
            //&& addr != tx_context.tx_to // TODO
            && !(revision >= Revision::EVMC_SHANGHAI && *addr == tx_context.block_coinbase)
            && context.access_account(addr) == AccessStatus::EVMC_ACCESS_COLD
        {
            consume_gas::<2600>(gas_left)?;
        } else {
            consume_gas::<100>(gas_left)?;
        }
    }
    Ok(())
}

/// consume 3 * minimum_word_size
#[inline(always)]
pub fn consume_copy_cost(gas_left: &mut u64, len: u64) -> Result<(), (StepStatusCode, StatusCode)> {
    consume_dyn_gas(gas_left, 3 * word_size(len))?;
    Ok(())
}