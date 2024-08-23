use evmc_vm::{AccessStatus, Address, Revision, StatusCode};

use crate::{
    interpreter::Interpreter,
    types::{u256, ExecutionContextTrait},
    utils::word_size,
};

#[inline(always)]
pub fn consume_gas(gas_left: &mut u64, gas: u64) -> Result<(), StatusCode> {
    if *gas_left < gas {
        return Err(StatusCode::EVMC_OUT_OF_GAS);
    }
    *gas_left -= gas;
    Ok(())
}

#[inline(always)]
pub fn consume_positive_value_cost(value: &u256, gas_left: &mut u64) -> Result<(), StatusCode> {
    if *value != u256::ZERO {
        consume_gas(gas_left, 9000)?;
    }
    Ok(())
}

#[inline(always)]
pub fn consume_value_to_empty_account_cost<E: ExecutionContextTrait>(
    value: &u256,
    addr: &Address,
    state: &mut Interpreter<E>,
) -> Result<(), StatusCode> {
    if *value != u256::ZERO && !state.context.account_exists(addr) {
        consume_gas(&mut state.gas_left, 25000)?;
    }
    Ok(())
}

#[inline(always)]
pub fn consume_address_access_cost<E: ExecutionContextTrait>(
    addr: &Address,
    state: &mut Interpreter<E>,
) -> Result<(), StatusCode> {
    if state.revision < Revision::EVMC_BERLIN {
        return Ok(());
    }
    if state.context.access_account(addr) == AccessStatus::EVMC_ACCESS_COLD {
        consume_gas(&mut state.gas_left, 2600)
    } else {
        consume_gas(&mut state.gas_left, 100)
    }
}

/// consume 3 * minimum_word_size
#[inline(always)]
pub fn consume_copy_cost(gas_left: &mut u64, len: u64) -> Result<(), StatusCode> {
    let (cost, cost_overflow) = word_size(len)?.overflowing_mul(3);
    if cost_overflow {
        return Err(StatusCode::EVMC_OUT_OF_GAS);
    }
    consume_gas(gas_left, cost)
}
