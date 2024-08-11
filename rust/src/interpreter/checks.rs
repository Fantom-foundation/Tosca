use evmc_vm::{ExecutionMessage, MessageFlags, Revision, StatusCode, StepStatusCode};

#[inline(always)]
pub(super) fn check_min_revision(
    min_revision: Revision,
    revision: Revision,
) -> Result<(), (StepStatusCode, StatusCode)> {
    if revision < min_revision {
        return Err((
            StepStatusCode::EVMC_STEP_FAILED,
            StatusCode::EVMC_UNDEFINED_INSTRUCTION,
        ));
    }
    Ok(())
}

#[inline(always)]
pub(super) fn check_not_read_only(
    message: &ExecutionMessage,
    revision: Revision,
) -> Result<(), (StepStatusCode, StatusCode)> {
    if revision >= Revision::EVMC_BYZANTIUM {
        let read_only = message.flags() == MessageFlags::EVMC_STATIC as u32;
        if read_only {
            return Err((
                StepStatusCode::EVMC_STEP_FAILED,
                StatusCode::EVMC_STATIC_MODE_VIOLATION,
            ));
        }
    }
    Ok(())
}
