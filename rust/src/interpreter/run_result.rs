use std::mem;

use evmc_vm::{ExecutionResult, Revision, StatusCode, StepResult, StepStatusCode, Uint256};

use crate::{
    interpreter::{CodeState, Memory, Stack},
    types::u256,
};

#[derive(Debug)]
pub struct RunResult<'a> {
    pub step_status_code: StepStatusCode,
    pub status_code: StatusCode,
    pub revision: Revision,
    pub code_state: CodeState<'a>,
    pub gas_left: u64,
    pub gas_refund: i64,
    pub output: Option<Vec<u8>>,
    pub stack: Stack,
    pub memory: Memory,
    pub last_call_return_data: Option<Vec<u8>>,
}

impl<'a> From<StatusCode> for RunResult<'a> {
    fn from(status_code: StatusCode) -> Self {
        let step_status_code = match status_code {
            StatusCode::EVMC_SUCCESS => StepStatusCode::EVMC_STEP_RUNNING,
            StatusCode::EVMC_REVERT => StepStatusCode::EVMC_STEP_REVERTED,
            _ => StepStatusCode::EVMC_STEP_FAILED,
        };
        Self {
            step_status_code,
            status_code,
            revision: Revision::EVMC_FRONTIER,
            code_state: CodeState::new(&[], 0),
            gas_left: 0,
            gas_refund: 0,
            output: None,
            stack: Stack::new(Vec::new()),
            memory: Memory::new(Vec::new()),
            last_call_return_data: None,
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
        Self::new(
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

impl<'a> From<RunResult<'a>> for ExecutionResult {
    fn from(value: RunResult) -> Self {
        Self::new(
            value.status_code,
            value.gas_left as i64,
            value.gas_refund,
            value.output.as_deref(),
        )
    }
}
