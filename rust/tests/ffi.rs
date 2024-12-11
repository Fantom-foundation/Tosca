#![allow(unused_crate_dependencies)]
use common::{
    evmc_vm::{Revision, StatusCode, StepStatusCode},
    opcode::{ADD, PUSH0},
    MockExecutionContextTrait, MockExecutionMessage,
};
#[cfg(not(feature = "custom-evmc"))]
use driver::TX_CONTEXT_ZEROED;
use driver::{
    get_tx_context_zeroed,
    host_interface::{self, null_ptr_host_interface},
    Instance, SteppableInstance, ZERO,
};

#[test]
fn execute_can_be_called_with_mocked_context() {
    let mut instance = Instance::default();
    let host = host_interface::mocked_host_interface();
    let mut context = MockExecutionContextTrait::new();
    #[cfg(not(feature = "custom-evmc"))]
    context
        .expect_get_tx_context()
        .times(1)
        .return_const(TX_CONTEXT_ZEROED);
    let revision = Revision::EVMC_CANCUN;
    let message = MockExecutionMessage::default().to_evmc_message();
    let code = &[PUSH0];
    let result = instance.run(&host, &mut context, revision, &message, code);
    assert_eq!(result.status_code, StatusCode::EVMC_SUCCESS);
}

#[test]
fn execute_can_be_called_with_hardcoded_context() {
    let mut instance = Instance::default();
    let mut host = null_ptr_host_interface();
    host.get_tx_context = Some(get_tx_context_zeroed);
    let revision = Revision::EVMC_CANCUN;
    let message = MockExecutionMessage::default().to_evmc_message();
    let code = &[PUSH0];
    let result = instance.run_with_null_context(&host, revision, &message, code);
    assert_eq!(result.status_code, StatusCode::EVMC_SUCCESS);
}

#[test]
fn execute_can_be_called_with_empty_code() {
    let mut instance = Instance::default();
    let mut host = null_ptr_host_interface();
    host.get_tx_context = Some(get_tx_context_zeroed);
    let revision = Revision::EVMC_CANCUN;
    let message = MockExecutionMessage::default().to_evmc_message();
    let code = &[];
    let result = instance.run_with_null_context(&host, revision, &message, code);
    assert_eq!(result.status_code, StatusCode::EVMC_SUCCESS);
}

#[test]
fn execute_handles_error_correctly() {
    let mut instance = Instance::default();
    let mut host = null_ptr_host_interface();
    host.get_tx_context = Some(get_tx_context_zeroed);
    let revision = Revision::EVMC_CANCUN;
    let message = MockExecutionMessage::default().to_evmc_message();
    let code = &[ADD]; // this will error because the stack is empty
    let result = instance.run_with_null_context(&host, revision, &message, code);
    assert_eq!(result.status_code, StatusCode::EVMC_STACK_UNDERFLOW);
}

#[test]
fn step_n_can_be_called_with_mocked_context() {
    let mut instance = SteppableInstance::default();
    let host = host_interface::mocked_host_interface();
    let mut context = MockExecutionContextTrait::new();
    #[cfg(not(feature = "custom-evmc"))]
    context
        .expect_get_tx_context()
        .times(1)
        .return_const(TX_CONTEXT_ZEROED);
    let revision = Revision::EVMC_CANCUN;
    let message = MockExecutionMessage::default().to_evmc_message();
    let code = &[PUSH0];
    let result = instance.run(
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
    let mut instance = SteppableInstance::default();
    let mut host = null_ptr_host_interface();
    host.get_tx_context = Some(get_tx_context_zeroed);
    let revision = Revision::EVMC_CANCUN;
    let message = MockExecutionMessage::default().to_evmc_message();
    let code = &[PUSH0];
    let result = instance.run_with_null_context(
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
    let mut instance = SteppableInstance::default();
    let mut host = null_ptr_host_interface();
    host.get_tx_context = Some(get_tx_context_zeroed);
    let revision = Revision::EVMC_CANCUN;
    let message = MockExecutionMessage::default().to_evmc_message();
    let code = &[];
    let result = instance.run_with_null_context(
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
    let mut instance = SteppableInstance::default();
    let mut host = null_ptr_host_interface();
    host.get_tx_context = Some(get_tx_context_zeroed);
    let revision = Revision::EVMC_CANCUN;
    let message = MockExecutionMessage::default().to_evmc_message();
    let code = &[ADD]; // this will error because the stack is empty
    let result = instance.run_with_null_context(
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
