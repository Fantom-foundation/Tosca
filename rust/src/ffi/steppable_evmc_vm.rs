//! This module implements the functions for the [`SteppableEvmcVm`] interface which are called
//! from the host language via FFI. The functions in this module only check the provided
//! arguments for validity, map them to Rust types and then call the business logic.
//! This is in essence what evmc_declare::evmc_declare_vm generates, but for [`SteppableEvmcVm`]
//! instead of [`EvmcVm`](evmc_vm::EvmcVm).

use std::{ffi::c_void, panic, slice};

use ::evmc_vm::{
    ffi::{
        evmc_bytes32, evmc_capabilities, evmc_host_interface, evmc_message, evmc_revision,
        evmc_step_result, evmc_step_status_code, evmc_vm_steppable,
    },
    ExecutionContext, ExecutionMessage, StatusCode, StepResult, StepStatusCode,
    SteppableEvmcContainer, SteppableEvmcVm,
};

use crate::{
    ffi::evmc_vm::{self, EVMC_CAPABILITY},
    EvmRs,
};

#[no_mangle]
extern "C" fn evmc_create_steppable_evmrs() -> *const evmc_vm_steppable {
    let new_instance = evmc_vm_steppable {
        vm: evmc_vm::evmc_create_evmrs() as *mut ::evmc_vm::ffi::evmc_vm,
        step_n: Some(__evmc_step_n),
        destroy: Some(__evmc_steppable_destroy),
    };
    let container = SteppableEvmcContainer::<EvmRs>::new(new_instance);

    // Release ownership to EVMC.
    SteppableEvmcContainer::into_ffi_pointer(container)
}

extern "C" fn __evmc_steppable_destroy(instance: *mut evmc_vm_steppable) {
    if instance.is_null() {
        // This is an irrecoverable error that violates the EVMC spec.
        std::process::abort();
    }
    unsafe {
        // Acquire ownership from EVMC. This will deallocate it also at the end of the scope.
        SteppableEvmcContainer::<EvmRs>::from_ffi_pointer(instance);
    }
}

#[no_mangle]
extern "C" fn __evmc_step_n(
    instance: *mut evmc_vm_steppable,
    host: *const evmc_host_interface,
    context: *mut c_void,
    revision: evmc_revision,
    message: *const evmc_message,
    code: *const u8,
    code_size: usize,
    status: evmc_step_status_code,
    pc: u64,
    gas_refunds: i64,
    stack: *mut evmc_bytes32,
    stack_size: usize,
    memory: *mut u8,
    memory_size: usize,
    last_call_result_data: *mut u8,
    last_call_result_data_size: usize,
    steps: i32,
) -> evmc_step_result {
    if instance.is_null()
        || (host.is_null() && EVMC_CAPABILITY != evmc_capabilities::EVMC_CAPABILITY_PRECOMPILES)
        || message.is_null()
        || (code.is_null() && code_size > 0)
        || (stack.is_null() && stack_size > 0)
        || (memory.is_null() && memory_size > 0)
        || (last_call_result_data.is_null() && last_call_result_data_size > 0)
    {
        // These are irrecoverable errors that violate the EVMC spec.
        std::process::abort();
    }

    let execution_message: ExecutionMessage = unsafe {
        // SAFETY:
        // message is not null
        (&*message).into()
    };

    let code_ref = if code.is_null() {
        &[]
    } else {
        // SAFETY:
        // code is not null and code size > 0
        unsafe { slice::from_raw_parts(code, code_size) }
    };

    let container = unsafe {
        // Acquire ownership from EVMC.
        SteppableEvmcContainer::<EvmRs>::from_ffi_pointer(instance)
    };

    let result = panic::catch_unwind(|| {
        assert_ne!(
            EVMC_CAPABILITY,
            evmc_capabilities::EVMC_CAPABILITY_PRECOMPILES
        );
        let mut execution_context = unsafe {
            // SAFETY:
            // Because EVMC_CAPABILITY_PRECOMPILES is not supported host is not null.
            ExecutionContext::new(&*host, context)
        };

        let stack = if stack.is_null() {
            &mut []
        } else {
            unsafe {
                // SAFETY:
                // stack is not null and stack size > 0
                slice::from_raw_parts_mut(stack, stack_size)
            }
        };

        let memory = if memory.is_null() {
            &mut []
        } else {
            unsafe {
                // SAFETY:
                // memory is not null and memory size > 0
                slice::from_raw_parts_mut(memory, memory_size)
            }
        };

        let last_call_result_data = if last_call_result_data.is_null() {
            &mut []
        } else {
            unsafe {
                // SAFETY:
                // last call return data is not null and size > 0
                slice::from_raw_parts_mut(last_call_result_data, last_call_result_data_size)
            }
        };

        container.step_n(
            revision,
            code_ref,
            &execution_message,
            &mut execution_context,
            status,
            pc,
            gas_refunds,
            stack,
            memory,
            last_call_result_data,
            steps,
        )
    });

    // Release ownership to EVMC.
    SteppableEvmcContainer::into_ffi_pointer(container);

    result
        .unwrap_or(StepResult::new(
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
        ))
        .into()
}
