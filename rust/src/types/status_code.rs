use common::evmc_vm::{
    ExecutionResult, Revision, StatusCode as EvmcStatusCode, StepResult,
    StepStatusCode as EvmcStepStatusCode,
};

/// This type combines the success variants of [`EvmcStatusCode`] and [`EvmcStepStatusCode`].
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum ExecStatus {
    Running,
    Stopped,
    Returned,
    Revert,
}

/// This type combines the failure variants of [`EvmcStatusCode`] and [`EvmcStepStatusCode`].
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum FailStatus {
    Failure = EvmcStatusCode::EVMC_FAILURE as isize,
    OutOfGas = EvmcStatusCode::EVMC_OUT_OF_GAS as isize,
    InvalidInstruction = EvmcStatusCode::EVMC_INVALID_INSTRUCTION as isize,
    UndefinedInstruction = EvmcStatusCode::EVMC_UNDEFINED_INSTRUCTION as isize,
    StackOverflow = EvmcStatusCode::EVMC_STACK_OVERFLOW as isize,
    StackUnderflow = EvmcStatusCode::EVMC_STACK_UNDERFLOW as isize,
    BadJumpDestination = EvmcStatusCode::EVMC_BAD_JUMP_DESTINATION as isize,
    InvalidMemoryAccess = EvmcStatusCode::EVMC_INVALID_MEMORY_ACCESS as isize,
    CallDepthExceeded = EvmcStatusCode::EVMC_CALL_DEPTH_EXCEEDED as isize,
    StaticModeViolation = EvmcStatusCode::EVMC_STATIC_MODE_VIOLATION as isize,
    PrecompileFailure = EvmcStatusCode::EVMC_PRECOMPILE_FAILURE as isize,
    ContractValidationFailure = EvmcStatusCode::EVMC_CONTRACT_VALIDATION_FAILURE as isize,
    ArgumentOutOfRange = EvmcStatusCode::EVMC_ARGUMENT_OUT_OF_RANGE as isize,
    WasmUnreachableInstruction = EvmcStatusCode::EVMC_WASM_UNREACHABLE_INSTRUCTION as isize,
    WasmTrap = EvmcStatusCode::EVMC_WASM_TRAP as isize,
    InsufficientBalance = EvmcStatusCode::EVMC_INSUFFICIENT_BALANCE as isize,
    InternalError = EvmcStatusCode::EVMC_INTERNAL_ERROR as isize,
    Rejected = EvmcStatusCode::EVMC_REJECTED as isize,
    OutOfMemory = EvmcStatusCode::EVMC_OUT_OF_MEMORY as isize,
}

impl From<FailStatus> for EvmcStatusCode {
    fn from(value: FailStatus) -> Self {
        match value {
            FailStatus::Failure => Self::EVMC_FAILURE,
            FailStatus::OutOfGas => Self::EVMC_OUT_OF_GAS,
            FailStatus::InvalidInstruction => Self::EVMC_INVALID_INSTRUCTION,
            FailStatus::UndefinedInstruction => Self::EVMC_UNDEFINED_INSTRUCTION,
            FailStatus::StackOverflow => Self::EVMC_STACK_OVERFLOW,
            FailStatus::StackUnderflow => Self::EVMC_STACK_UNDERFLOW,
            FailStatus::BadJumpDestination => Self::EVMC_BAD_JUMP_DESTINATION,
            FailStatus::InvalidMemoryAccess => Self::EVMC_INVALID_MEMORY_ACCESS,
            FailStatus::CallDepthExceeded => Self::EVMC_CALL_DEPTH_EXCEEDED,
            FailStatus::StaticModeViolation => Self::EVMC_STATIC_MODE_VIOLATION,
            FailStatus::PrecompileFailure => Self::EVMC_PRECOMPILE_FAILURE,
            FailStatus::ContractValidationFailure => Self::EVMC_CONTRACT_VALIDATION_FAILURE,
            FailStatus::ArgumentOutOfRange => Self::EVMC_ARGUMENT_OUT_OF_RANGE,
            FailStatus::WasmUnreachableInstruction => Self::EVMC_WASM_UNREACHABLE_INSTRUCTION,
            FailStatus::WasmTrap => Self::EVMC_WASM_TRAP,
            FailStatus::InsufficientBalance => Self::EVMC_INSUFFICIENT_BALANCE,
            FailStatus::InternalError => Self::EVMC_INTERNAL_ERROR,
            FailStatus::Rejected => Self::EVMC_REJECTED,
            FailStatus::OutOfMemory => Self::EVMC_OUT_OF_MEMORY,
        }
    }
}

impl From<FailStatus> for EvmcStepStatusCode {
    fn from(_value: FailStatus) -> Self {
        Self::EVMC_STEP_FAILED
    }
}

impl From<ExecStatus> for EvmcStatusCode {
    fn from(value: ExecStatus) -> Self {
        match value {
            ExecStatus::Running | ExecStatus::Stopped | ExecStatus::Returned => Self::EVMC_SUCCESS,
            ExecStatus::Revert => Self::EVMC_REVERT,
        }
    }
}

impl From<ExecStatus> for EvmcStepStatusCode {
    fn from(value: ExecStatus) -> Self {
        match value {
            ExecStatus::Running => Self::EVMC_STEP_RUNNING,
            ExecStatus::Stopped => Self::EVMC_STEP_STOPPED,
            ExecStatus::Returned => Self::EVMC_STEP_RETURNED,
            ExecStatus::Revert => Self::EVMC_STEP_REVERTED,
        }
    }
}

impl From<FailStatus> for StepResult {
    fn from(fail_status: FailStatus) -> Self {
        Self::new(
            fail_status.into(),
            fail_status.into(),
            Revision::EVMC_ISTANBUL,
            0,
            0,
            0,
            None,
            Vec::new(),
            Vec::new(),
            None,
        )
    }
}

impl From<FailStatus> for ExecutionResult {
    fn from(fail_status: FailStatus) -> Self {
        Self::new(fail_status.into(), 0, 0, None)
    }
}
