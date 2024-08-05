use std::mem;

use evmc_vm::{
    EvmcVm, ExecutionContext, ExecutionMessage, ExecutionResult, Revision, StepResult,
    StepStatusCode, SteppableEvmcVm, Uint256,
};

use crate::types::u256;

mod ffi;
mod interpreter;
mod types;

//#[evmc_declare::evmc_declare_vm("evmrs", "ewasm, evm", "0.1.0")]
pub struct EvmRs;

impl EvmcVm for EvmRs {
    fn init() -> Self {
        EvmRs {}
    }

    fn execute(
        &self,
        revision: Revision,
        code: &[u8],
        message: &ExecutionMessage,
        context: Option<&mut ExecutionContext>,
    ) -> ExecutionResult {
        let step_result = interpreter::run(
            revision,
            code,
            message,
            context,
            StepStatusCode::EVMC_STEP_RUNNING,
            0,
            0,
            Vec::with_capacity(1024),
            Vec::new(),
            None,
            None,
        );

        step_result
            .map(Into::into)
            .unwrap_or_else(|(_, status_code)| ExecutionResult::new(status_code, 0, 0, None))
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
        step_status: StepStatusCode,
        pc: u64,
        gas_refund: i64,
        stack: &'a mut [Uint256],
        memory: &'a mut [u8],
        last_call_result_data: &'a mut [u8],
        steps: i32,
    ) -> StepResult {
        interpreter::run(
            revision,
            code,
            message,
            context,
            step_status,
            pc as usize,
            gas_refund,
            // SAFETY
            // u256 is a newtype of Uint256 with repr(transparent) which guarantees the same memory
            // layout.
            unsafe { mem::transmute::<Vec<Uint256>, Vec<u256>>(stack.to_owned()) },
            memory.to_owned(),
            Some(last_call_result_data.to_owned()),
            Some(steps),
        )
        .unwrap_or_else(|(step_status_code, status_code)| {
            StepResult::new(
                step_status_code,
                status_code,
                revision,
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
