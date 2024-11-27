use std::process;

use evmc_vm::{
    ffi::evmc_capabilities, EvmcVm, ExecutionContext, ExecutionMessage, ExecutionResult, Revision,
    StatusCode as EvmcStatusCode, StepResult, StepStatusCode as EvmcStepStatusCode,
    SteppableEvmcVm, Uint256,
};

use crate::{
    ffi::EVMC_CAPABILITY,
    interpreter::Interpreter,
    types::{LoggingObserver, Memory, NoOpObserver, ObserverType, Stack},
    u256,
};

pub struct EvmRs {
    observer_type: ObserverType,
}

impl EvmcVm for EvmRs {
    fn init() -> Self {
        EvmRs {
            observer_type: ObserverType::NoOp,
        }
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
        let interpreter = Interpreter::new(revision, message, context, code);
        match self.observer_type {
            ObserverType::NoOp => interpreter.run(&mut NoOpObserver()),
            ObserverType::Logging => interpreter.run(&mut LoggingObserver::new(std::io::stdout())),
        }
    }

    fn set_option(&mut self, key: &str, value: &str) -> Result<(), evmc_vm::SetOptionError> {
        match (key, value) {
            ("logging", "true") => self.observer_type = ObserverType::Logging,
            ("logging", "false") => self.observer_type = ObserverType::NoOp,
            _ => (),
        }
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
        step_status_code: EvmcStepStatusCode,
        pc: u64,
        gas_refund: i64,
        stack: &'a mut [Uint256],
        memory: &'a mut [u8],
        last_call_return_data: &'a mut [u8],
        steps: i32,
    ) -> StepResult {
        if step_status_code != EvmcStepStatusCode::EVMC_STEP_RUNNING {
            return StepResult::new(
                step_status_code,
                match step_status_code {
                    EvmcStepStatusCode::EVMC_STEP_RUNNING
                    | EvmcStepStatusCode::EVMC_STEP_STOPPED
                    | EvmcStepStatusCode::EVMC_STEP_RETURNED => EvmcStatusCode::EVMC_SUCCESS,
                    EvmcStepStatusCode::EVMC_STEP_REVERTED => EvmcStatusCode::EVMC_REVERT,
                    EvmcStepStatusCode::EVMC_STEP_FAILED => EvmcStatusCode::EVMC_FAILURE,
                },
                revision,
                pc,
                gas_refund,
                gas_refund,
                None,
                stack.to_owned(),
                memory.to_owned(),
                if last_call_return_data.is_empty() {
                    None
                } else {
                    Some(last_call_return_data.to_owned())
                },
            );
        }
        assert_ne!(
            EVMC_CAPABILITY,
            evmc_capabilities::EVMC_CAPABILITY_PRECOMPILES
        );
        let Some(context) = context else {
            // Since EVMC_CAPABILITY_PRECOMPILES is not supported context must be set.
            // If this is not the case it violates the EVMC spec and is an irrecoverable error.
            process::abort();
        };
        let stack = Stack::new(&stack.iter().map(|i| u256::from(*i)).collect::<Vec<_>>());
        let memory = Memory::new(memory.to_owned());
        let interpreter = Interpreter::new_steppable(
            revision,
            message,
            context,
            code,
            pc as usize,
            gas_refund,
            stack,
            memory,
            Some(last_call_return_data.to_owned()),
            Some(steps),
        );
        match self.observer_type {
            ObserverType::NoOp => interpreter.run(&mut NoOpObserver()),
            ObserverType::Logging => interpreter.run(&mut LoggingObserver::new(std::io::stdout())),
        }
    }
}
