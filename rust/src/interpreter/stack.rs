use evmc_vm::{StatusCode, StepStatusCode};

use crate::types::u256;

#[inline(always)]
pub fn check_stack_overflow<const N: usize>(
    stack: &[u256],
) -> Result<(), (StepStatusCode, StatusCode)> {
    if stack.len() + N > 1024 {
        return Err((
            StepStatusCode::EVMC_STEP_FAILED,
            StatusCode::EVMC_STACK_OVERFLOW,
        ));
    }
    Ok(())
}

#[inline(always)]
pub fn pop_from_stack<const N: usize>(
    stack: &mut Vec<u256>,
) -> Result<[u256; N], (StepStatusCode, StatusCode)> {
    if stack.len() < N {
        return Err((
            StepStatusCode::EVMC_STEP_FAILED,
            StatusCode::EVMC_STACK_UNDERFLOW,
        ));
    }
    let mut array = [u256::ZERO; N];
    for element in &mut array {
        *element = stack.pop().unwrap();
    }

    Ok(array)
}

#[inline(always)]
pub fn nth_ref_from_stack<const N: usize>(
    stack: &[u256],
) -> Result<&u256, (StepStatusCode, StatusCode)> {
    if stack.len() < N {
        return Err((
            StepStatusCode::EVMC_STEP_FAILED,
            StatusCode::EVMC_STACK_UNDERFLOW,
        ));
    }

    Ok(&stack[stack.len() - N])
}
