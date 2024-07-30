//! This module implements the functions for the [`SteppableEvmcVm`] interface which are called
//! from the host language via FFI. The functions in this module only check the provided
//! arguments for validity, map them to Rust types and then call the business logic.
//! This is in essence what evmc_declare::evmc_declare_vm generates, but for [`SteppableEvmcVm`]
//! instead of [`EvmcVm`](evmc_vm::EvmcVm).

use std::slice;

use evmc_vm::{ExecutionContext, StatusCode, StepResult, StepStatusCode, SteppableEvmcVm};

use crate::EvmRs;

#[no_mangle]
extern "C" fn evmc_create_steppable_evmrs() -> *const ::evmc_vm::ffi::evmc_vm_steppable {
    let new_instance = ::evmc_vm::ffi::evmc_vm_steppable {
        vm: crate::evmc_create_evmrs() as *mut ::evmc_vm::ffi::evmc_vm,
        step_n: Some(__evmc_step_n),
        destroy: Some(__evmc_steppable_destroy),
    };
    let container = ::evmc_vm::SteppableEvmcContainer::<EvmRs>::new(new_instance);
    ::evmc_vm::SteppableEvmcContainer::into_ffi_pointer(container)
}

extern "C" fn __evmc_steppable_destroy(instance: *mut ::evmc_vm::ffi::evmc_vm_steppable) {
    if instance.is_null() {
        std::process::abort();
    }
    unsafe {
        ::evmc_vm::SteppableEvmcContainer::<EvmRs>::from_ffi_pointer(instance);
    }
}

// must be defined in evmc_declare_vm
const EVMC_CAPABILITY_PRECOMPILES: bool = false;

#[no_mangle]
extern "C" fn __evmc_step_n(
    instance: *mut evmc_vm::ffi::evmc_vm_steppable,
    host: *const evmc_vm::ffi::evmc_host_interface,
    context: *mut std::ffi::c_void,
    revision: evmc_vm::ffi::evmc_revision,
    message: *const evmc_vm::ffi::evmc_message,
    code: *const u8,
    code_size: usize,
    status: evmc_vm::ffi::evmc_step_status_code,
    pc: u64,
    gas_refunds: i64,
    stack: *mut evmc_vm::ffi::evmc_bytes32,
    stack_size: usize,
    memory: *mut u8,
    memory_size: usize,
    last_call_result_data: *mut u8,
    last_call_result_data_size: usize,
    steps: i32,
) -> evmc_vm::ffi::evmc_step_result {
    if instance.is_null()
        || (host.is_null() && !EVMC_CAPABILITY_PRECOMPILES)
        || message.is_null()
        || (code.is_null() && code_size > 0)
        || (stack.is_null() && stack_size > 0)
        || (memory.is_null() && memory_size > 0)
        || (last_call_result_data.is_null() && last_call_result_data_size > 0)
    {
        std::process::abort();
    }
    let execution_message: ::evmc_vm::ExecutionMessage =
        //unsafe { message.as_ref().expect("EVMC message is null").into() };
        unsafe { (&*message).into() };
    let code_ref: &[u8] = if code.is_null() {
        &[]
    } else {
        // SAFETY:
        // code is not null and code size > 0
        unsafe { ::std::slice::from_raw_parts(code, code_size) }
    };
    let container =
        unsafe { ::evmc_vm::SteppableEvmcContainer::<EvmRs>::from_ffi_pointer(instance) };

    let result = ::std::panic::catch_unwind(|| {
        let mut execution_context = if host.is_null() {
            None
        } else {
            Some(unsafe {
                ExecutionContext::new(host.as_ref().expect("EVMC host is null"), context)
            })
        };
        let stack = if stack.is_null() {
            &mut []
        } else {
            unsafe { slice::from_raw_parts_mut(stack, stack_size) }
        };
        let memory = if memory.is_null() {
            &mut []
        } else {
            unsafe { slice::from_raw_parts_mut(memory, memory_size) }
        };
        let last_call_result_data = if last_call_result_data.is_null() {
            &mut []
        } else {
            unsafe { slice::from_raw_parts_mut(last_call_result_data, last_call_result_data_size) }
        };
        container.step_n(
            revision,
            code_ref,
            &execution_message,
            execution_context.as_mut(),
            status,
            pc,
            gas_refunds,
            stack,
            memory,
            last_call_result_data,
            steps,
        )
    });
    ::evmc_vm::SteppableEvmcContainer::into_ffi_pointer(container);

    result
        .unwrap_or_else(|_| {
            StepResult::new(
                StepStatusCode::EVMC_STEP_FAILED,
                StatusCode::EVMC_INTERNAL_ERROR,
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
        .into()
}
