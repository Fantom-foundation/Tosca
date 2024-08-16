use std::mem;

use evmc_vm::{Revision, StatusCode, StepResult, StepStatusCode, Uint256};

use crate::{
    interpreter::{CodeState, Memory, Stack},
    types::u256,
};

#[derive(Debug)]
pub struct RunResult<'a> {
    step_status_code: StepStatusCode,
    status_code: StatusCode,
    revision: Revision,
    code_state: CodeState<'a>,
    gas_left: u64,
    gas_refund: i64,
    output: Option<Vec<u8>>,
    stack: Stack,
    memory: Memory,
    last_call_return_data: Option<Vec<u8>>,
}

impl<'a> RunResult<'a> {
    #[allow(clippy::too_many_arguments)]
    pub fn new(
        step_status_code: StepStatusCode,
        status_code: StatusCode,
        revision: Revision,
        code_state: CodeState<'a>,
        gas_left: u64,
        gas_refund: i64,
        output: Option<Vec<u8>>,
        stack: Stack,
        memory: Memory,
        last_call_return_data: Option<Vec<u8>>,
    ) -> Self {
        Self {
            step_status_code,
            status_code,
            revision,
            code_state,
            gas_left,
            gas_refund,
            output,
            stack,
            memory,
            last_call_return_data,
        }
    }
}

impl<'a> From<RunResult<'a>> for StepResult {
    fn from(value: RunResult) -> Self {
        let stack = value.stack.into_inner();
        let stack = unsafe {
            // SAFETY
            // u256 is a newtype of Uint256 with repr(transparent) which guarantees the same memory
            // layout.
            mem::transmute::<Vec<u256>, Vec<Uint256>>(stack)
        };
        StepResult::new(
            value.step_status_code,
            value.status_code,
            value.revision,
            value.code_state.pc() as u64,
            value.gas_left as i64,
            value.gas_refund,
            value.output,
            stack,
            value.memory.into_inner(),
            value.last_call_return_data,
        )
    }
}
