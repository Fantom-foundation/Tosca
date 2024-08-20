use evmc_vm::{AccessStatus, Address, ExecutionContext, Revision, StatusCode};

use crate::{interpreter::utils::word_size, types::u256};

#[inline(always)]
pub(super) fn consume_gas(gas_left: &mut u64, gas: u64) -> Result<(), StatusCode> {
    if *gas_left < gas {
        return Err(StatusCode::EVMC_OUT_OF_GAS);
    }
    *gas_left -= gas;
    Ok(())
}

#[inline(always)]
pub(super) fn consume_positive_value_cost(
    value: &u256,
    gas_left: &mut u64,
) -> Result<(), StatusCode> {
    if *value != u256::ZERO {
        consume_gas(gas_left, 9000)?;
    }
    Ok(())
}

#[inline(always)]
pub(super) fn consume_value_to_empty_account_cost(
    value: &u256,
    addr: &Address,
    context: &mut ExecutionContext,
    gas_left: &mut u64,
) -> Result<(), StatusCode> {
    if *value != u256::ZERO && !context.account_exists(addr) {
        consume_gas(gas_left, 25000)?;
    }
    Ok(())
}

#[inline(always)]
pub(super) fn consume_address_access_cost(
    gas_left: &mut u64,
    addr: &Address,
    context: &mut ExecutionContext,
    revision: Revision,
) -> Result<(), StatusCode> {
    let tx_context = context.get_tx_context();
    if revision < Revision::EVMC_BERLIN {
        return Ok(());
    }
    if *addr != tx_context.tx_origin
            //&& addr != tx_context.tx_to // TODO
            && !(revision >= Revision::EVMC_SHANGHAI && *addr == tx_context.block_coinbase)
            && context.access_account(addr) == AccessStatus::EVMC_ACCESS_COLD
    {
        consume_gas(gas_left, 2600)
    } else {
        consume_gas(gas_left, 100)
    }
}

/// consume 3 * minimum_word_size
#[inline(always)]
pub(super) fn consume_copy_cost(gas_left: &mut u64, len: u64) -> Result<(), StatusCode> {
    let (cost, cost_overflow) = word_size(len)?.overflowing_mul(3);
    if cost_overflow {
        return Err(StatusCode::EVMC_OUT_OF_GAS);
    }
    consume_gas(gas_left, cost)
}
