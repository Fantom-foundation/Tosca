use std::{mem, process};

use evmc_vm::{
    ffi::evmc_capabilities, EvmcVm, ExecutionContext, ExecutionMessage, ExecutionResult, Revision,
    StatusCode, StepResult, StepStatusCode, SteppableEvmcVm, Uint256,
};

use crate::{
    ffi::EVMC_CAPABILITY,
    interpreter::Interpreter,
    types::{u256, Memory, Stack},
};

pub struct EvmRs;

impl EvmcVm for EvmRs {
    fn init() -> Self {
        EvmRs {}
    }

    fn execute<'a>(
        &self,
        revision: Revision,
        code: &'a [u8],
        message: &'a ExecutionMessage,
        context: Option<&'a mut ExecutionContext<'a>>,
    ) -> ExecutionResult {
        assert_ne!(
            EVMC_CAPABILITY,
            evmc_capabilities::EVMC_CAPABILITY_PRECOMPILES
        );
        let Some(context) = context else {
            // Since EVMC_CAPABILITY_PRECOMPILES is not supported context must be set.
            // If this is not the case it violates the EVMC spec and is an irrecoverable error.
            process::abort();
        };
        Interpreter::new(revision, message, context, code)
            .run()
            .map(Into::into)
            .unwrap_or_else(|status_code| ExecutionResult::new(status_code, 0, 0, None))
    }

    fn set_option(&mut self, _: &str, _: &str) -> Result<(), evmc_vm::SetOptionError> {
        Ok(())
    }
}

impl SteppableEvmcVm for EvmRs {
    fn step_n<'a>(
        &self,
        revision: Revision,
        code: &'a [u8],
        message: &'a ExecutionMessage,
        context: Option<&'a mut ExecutionContext<'a>>,
        step_status_code: StepStatusCode,
        pc: u64,
        gas_refund: i64,
        stack: &'a mut [Uint256],
        memory: &'a mut [u8],
        last_call_return_data: &'a mut [u8],
        steps: i32,
    ) -> StepResult {
        assert_ne!(
            EVMC_CAPABILITY,
            evmc_capabilities::EVMC_CAPABILITY_PRECOMPILES
        );
        let Some(context) = context else {
            // Since EVMC_CAPABILITY_PRECOMPILES is not supported context must be set.
            // If this is not the case it violates the EVMC spec and is an irrecoverable error.
            process::abort();
        };
        let stack = Stack::new(
            // SAFETY:
            // u256 is a newtype of Uint256 with repr(transparent) which guarantees the same memory
            // layout.
            unsafe { mem::transmute::<Vec<Uint256>, Vec<u256>>(stack.to_owned()) },
        );
        let memory = Memory::new(memory.to_owned());
        Interpreter::new_steppable(
            revision,
            message,
            context,
            step_status_code,
            code,
            pc as usize,
            gas_refund,
            stack,
            memory,
            Some(last_call_return_data.to_owned()),
            Some(steps),
        )
        .run()
        .map(Into::into)
        .unwrap_or_else(|status_code| {
            let step_status_code = match status_code {
                StatusCode::EVMC_SUCCESS => StepStatusCode::EVMC_STEP_RUNNING,
                StatusCode::EVMC_REVERT => StepStatusCode::EVMC_STEP_REVERTED,
                _ => StepStatusCode::EVMC_STEP_FAILED,
            };
            StepResult::new(
                step_status_code,
                status_code,
                Revision::EVMC_FRONTIER,
                0,
                0,
                0,
                None,
                Vec::new(),
                Vec::new(),
                None,
            )
        })
    }
}
