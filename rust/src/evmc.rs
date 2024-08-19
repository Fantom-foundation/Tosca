use std::{mem, process};

use evmc_vm::{
    ffi::evmc_capabilities, EvmcVm, ExecutionContext, ExecutionMessage, ExecutionResult, Revision,
    StepResult, StepStatusCode, SteppableEvmcVm, Uint256,
};

use crate::{
    ffi::EVMC_CAPABILITY,
    interpreter,
    interpreter::{CodeState, Memory, Stack},
    types::u256,
};

//#[evmc_declare::evmc_declare_vm("evmrs", "ewasm, evm", "0.1.0")]
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
        run(
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
        )
        .into()
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
        run(
            revision,
            code,
            message,
            context,
            step_status,
            pc,
            gas_refund,
            stack.to_owned(),
            memory.to_owned(),
            Some(last_call_result_data.to_owned()),
            Some(steps),
        )
    }
}

#[allow(clippy::too_many_arguments)]
pub fn run<'a>(
    revision: Revision,
    code: &'a [u8],
    message: &'a ExecutionMessage,
    context: Option<&'a mut ExecutionContext<'a>>,
    step_status_code: StepStatusCode,
    pc: u64,
    gas_refund: i64,
    stack: Vec<Uint256>,
    memory: Vec<u8>,
    last_call_return_data: Option<Vec<u8>>,
    steps: Option<i32>,
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
    interpreter::run(
        revision,
        message,
        context,
        step_status_code,
        CodeState::new(code, pc as usize),
        gas_refund,
        // SAFETY
        // u256 is a newtype of Uint256 with repr(transparent) which guarantees the same memory
        // layout.
        Stack::new(unsafe { mem::transmute::<Vec<Uint256>, Vec<u256>>(stack.to_owned()) }),
        Memory::new(memory),
        last_call_return_data,
        steps,
    )
    .map(Into::into)
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
