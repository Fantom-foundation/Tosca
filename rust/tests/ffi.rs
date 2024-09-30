#![allow(clippy::undocumented_unsafe_blocks, unused_crate_dependencies)]
use driver::{
    get_tx_context_zeroed,
    host_interface::{self, null_ptr_host_interface},
    TX_CONTEXT_ZEROED, ZERO,
};
use evmc_vm::{Revision, StatusCode, StepStatusCode};
use evmrs::{MockExecutionContextTrait, MockExecutionMessage, Opcode};

#[test]
fn execute_can_be_called_with_mocked_context() {
    let host = host_interface::mocked_host_interface();
    let mut context = MockExecutionContextTrait::new();
    context
        .expect_get_tx_context()
        .times(1)
        .return_const(TX_CONTEXT_ZEROED);
    let revision = Revision::EVMC_CANCUN;
    let message = driver::to_evmc_message(&MockExecutionMessage::default());
    let code = &[Opcode::Push0 as u8];
    let result = driver::run(&host, &mut context, revision, &message, code);
    assert_eq!(result.status_code, StatusCode::EVMC_SUCCESS);
}

#[test]
fn execute_can_be_called_with_hardcoded_context() {
    let mut host = null_ptr_host_interface();
    host.get_tx_context = Some(get_tx_context_zeroed);
    let revision = Revision::EVMC_CANCUN;
    let message = driver::to_evmc_message(&MockExecutionMessage::default());
    let code = &[Opcode::Push0 as u8];
    let result = driver::run_with_null_context(&host, revision, &message, code);
    assert_eq!(result.status_code, StatusCode::EVMC_SUCCESS);
}

#[test]
fn execute_can_be_called_with_empty_code() {
    let mut host = null_ptr_host_interface();
    host.get_tx_context = Some(get_tx_context_zeroed);
    let revision = Revision::EVMC_CANCUN;
    let message = driver::to_evmc_message(&MockExecutionMessage::default());
    let code = &[];
    let result = driver::run_with_null_context(&host, revision, &message, code);
    assert_eq!(result.status_code, StatusCode::EVMC_SUCCESS);
}

#[test]
fn execute_handles_error_correctly() {
    let mut host = null_ptr_host_interface();
    host.get_tx_context = Some(get_tx_context_zeroed);
    let revision = Revision::EVMC_CANCUN;
    let message = driver::to_evmc_message(&MockExecutionMessage::default());
    let code = &[Opcode::Add as u8]; // this will error because the stack is empty
    let result = driver::run_with_null_context(&host, revision, &message, code);
    assert_eq!(result.status_code, StatusCode::EVMC_STACK_UNDERFLOW);
}

#[test]
fn step_n_can_be_called_with_mocked_context() {
    let host = host_interface::mocked_host_interface();
    let mut context = MockExecutionContextTrait::new();
    context
        .expect_get_tx_context()
        .times(1)
        .return_const(TX_CONTEXT_ZEROED);
    let revision = Revision::EVMC_CANCUN;
    let message = driver::to_evmc_message(&MockExecutionMessage::default());
    let code = &[Opcode::Push0 as u8];
    let result = driver::run_steppable(
        &host,
        &mut context,
        revision,
        &message,
        code,
        StepStatusCode::EVMC_STEP_RUNNING,
        0,
        0,
        &mut [ZERO],
        &mut [0],
        &mut [0],
        1,
    );
    assert_eq!(result.status_code, StatusCode::EVMC_SUCCESS);
    assert_eq!(result.step_status_code, StepStatusCode::EVMC_STEP_RUNNING);
}

#[test]
fn step_n_can_be_called_with_hardcoded_context() {
    let mut host = null_ptr_host_interface();
    host.get_tx_context = Some(get_tx_context_zeroed);
    let revision = Revision::EVMC_CANCUN;
    let message = driver::to_evmc_message(&MockExecutionMessage::default());
    let code = &[Opcode::Push0 as u8];
    let result = driver::run_steppable_with_null_context(
        &host,
        revision,
        &message,
        code,
        StepStatusCode::EVMC_STEP_RUNNING,
        0,
        0,
        &mut [ZERO],
        &mut [0],
        &mut [0],
        1,
    );
    assert_eq!(result.status_code, StatusCode::EVMC_SUCCESS);
    assert_eq!(result.step_status_code, StepStatusCode::EVMC_STEP_RUNNING);
}

#[test]
fn step_n_can_be_called_with_empty_code_and_stack_and_memory_and_last_call_result_data() {
    let mut host = null_ptr_host_interface();
    host.get_tx_context = Some(get_tx_context_zeroed);
    let revision = Revision::EVMC_CANCUN;
    let message = driver::to_evmc_message(&MockExecutionMessage::default());
    let code = &[];
    let result = driver::run_steppable_with_null_context(
        &host,
        revision,
        &message,
        code,
        StepStatusCode::EVMC_STEP_RUNNING,
        0,
        0,
        &mut [],
        &mut [],
        &mut [],
        1,
    );
    assert_eq!(result.status_code, StatusCode::EVMC_SUCCESS);
    assert_eq!(result.step_status_code, StepStatusCode::EVMC_STEP_STOPPED);
}

#[test]
fn step_n_handles_error_correctly() {
    let mut host = null_ptr_host_interface();
    host.get_tx_context = Some(get_tx_context_zeroed);
    let revision = Revision::EVMC_CANCUN;
    let message = driver::to_evmc_message(&MockExecutionMessage::default());
    let code = &[Opcode::Add as u8]; // this will error because the stack is empty
    let result = driver::run_steppable_with_null_context(
        &host,
        revision,
        &message,
        code,
        StepStatusCode::EVMC_STEP_RUNNING,
        0,
        0,
        &mut [ZERO],
        &mut [0],
        &mut [0],
        1,
    );
    assert_eq!(result.status_code, StatusCode::EVMC_STACK_UNDERFLOW);
    assert_eq!(result.step_status_code, StepStatusCode::EVMC_STEP_FAILED);
}
