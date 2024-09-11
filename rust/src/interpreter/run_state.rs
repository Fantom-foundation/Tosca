#![allow(dead_code)] // TODO remove once all is merged
use evmc_vm::{ExecutionContext, ExecutionMessage, Revision, StatusCode, StepStatusCode};

use crate::interpreter::{CodeState, Memory, Stack};

pub struct RunState<'a> {
    pub step_status_code: StepStatusCode,
    pub status_code: StatusCode,
    pub message: &'a ExecutionMessage,
    pub context: &'a mut ExecutionContext<'a>,
    pub revision: Revision,
    pub code_state: CodeState<'a>,
    pub gas_left: u64,
    pub gas_refund: i64,
    pub output: Option<Vec<u8>>,
    pub stack: Stack,
    pub memory: Memory,
    pub last_call_return_data: Option<Vec<u8>>,
    pub steps: Option<i32>,
}

impl<'a> RunState<'a> {
    pub fn new(
        revision: Revision,
        message: &'a ExecutionMessage,
        context: &'a mut ExecutionContext<'a>,
        step_status_code: StepStatusCode,
        code: &'a [u8],
        gas_refund: i64,
    ) -> Self {
        Self {
            step_status_code,
            status_code: StatusCode::EVMC_SUCCESS,
            message,
            context,
            revision,
            code_state: CodeState::new(code, 0),
            gas_left: message.gas() as u64,
            gas_refund,
            output: None,
            stack: Stack::new(Vec::new()),
            memory: Memory::new(Vec::new()),
            last_call_return_data: None,
            steps: None,
        }
    }

    #[allow(clippy::too_many_arguments)]
    pub fn new_steppable(
        revision: Revision,
        message: &'a ExecutionMessage,
        context: &'a mut ExecutionContext<'a>,
        step_status_code: StepStatusCode,
        code: &'a [u8],
        pc: usize,
        gas_refund: i64,
        stack: Stack,
        memory: Memory,
        last_call_return_data: Option<Vec<u8>>,
        steps: Option<i32>,
    ) -> Self {
        Self {
            step_status_code,
            status_code: StatusCode::EVMC_SUCCESS,
            message,
            context,
            revision,
            code_state: CodeState::new(code, pc),
            gas_left: message.gas() as u64,
            gas_refund,
            output: None,
            stack,
            memory,
            last_call_return_data,
            steps,
        }
    }
}
