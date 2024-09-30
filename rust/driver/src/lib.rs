use std::{ffi, ptr};

use evmc_vm::{
    ffi::{
        evmc_host_interface, evmc_message, evmc_result, evmc_step_result, evmc_step_status_code,
        evmc_tx_context, evmc_vm as evmc_vm_t, evmc_vm_steppable,
    },
    Address, Revision, Uint256,
};
#[cfg(feature = "mock")]
use evmrs::MockExecutionMessage;

pub mod host_interface;

extern "C" {
    fn evmc_create_evmrs() -> *const evmc_vm_t;
    fn evmc_create_steppable_evmrs() -> *const evmc_vm_steppable;
}

pub const ZERO: Uint256 = Uint256 { bytes: [0; 32] };
pub const ZERO_ADDR: Address = Address { bytes: [0; 20] };

pub const TX_CONTEXT_ZEROED: evmc_tx_context = evmc_tx_context {
    tx_gas_price: ZERO,
    tx_origin: ZERO_ADDR,
    block_coinbase: ZERO_ADDR,
    block_number: 0,
    block_timestamp: 0,
    block_gas_limit: 0,
    block_prev_randao: ZERO,
    chain_id: ZERO,
    block_base_fee: ZERO,
    blob_base_fee: ZERO,
    blob_hashes: ptr::null(),
    blob_hashes_count: 0,
    initcodes: ptr::null(),
    initcodes_count: 0,
};

/// # Safety
///
/// The value of the pointer is not used because a constant value is returned.
pub unsafe extern "C" fn get_tx_context_zeroed(_context: *mut ffi::c_void) -> evmc_tx_context {
    TX_CONTEXT_ZEROED
}

#[cfg(feature = "mock")]
pub fn to_evmc_message(message: &MockExecutionMessage) -> evmc_message {
    evmc_message {
        kind: message.kind,
        flags: message.flags,
        depth: message.depth,
        gas: message.gas,
        recipient: message.recipient,
        sender: message.sender,
        input_data: message.input.map(<[u8]>::as_ptr).unwrap_or(ptr::null()),
        input_size: message.input.map(<[u8]>::len).unwrap_or_default(),
        value: message.value,
        create2_salt: message.create2_salt,
        code_address: message.code_address,
        code: message.code.map(<[u8]>::as_ptr).unwrap_or(ptr::null()),
        code_size: message.code.map(<[u8]>::len).unwrap_or_default(),
        code_hash: ptr::null(),
    }
}

/// # Safety
///
/// All pointers must be valid, except for `context` which can be null if the `evmc_host_interface`
/// accepts null pointers as context.
pub unsafe fn run_raw(
    host: *const evmc_host_interface,
    context: *mut ffi::c_void,
    revision: Revision,
    message: *const evmc_message,
    code: *const u8,
    code_len: usize,
) -> evmc_result {
    let instance = evmc_create_evmrs();
    if instance.is_null() {
        panic!("vm instance is null")
    }
    // SAFETY:
    // `instance is not null`. `evmc_create_evmrs` must return a valid pointer to an `evmc_vm_t`.
    let instance = unsafe { &mut *(instance as *mut evmc_vm_t) };

    let execute = instance.execute.unwrap();
    let destroy = instance.destroy.unwrap();

    let result = unsafe { execute(instance, host, context, revision, message, code, code_len) };

    unsafe {
        destroy(instance);
    }

    result
}

pub fn run<T>(
    host: &evmc_host_interface,
    context: &mut T,
    revision: Revision,
    message: &evmc_message,
    code: &[u8],
) -> evmc_result {
    // SAFETY:
    // All pointer are valid since they are created from references.
    unsafe {
        run_raw(
            host,
            context as *mut T as *mut ffi::c_void,
            revision,
            message,
            if code.is_empty() {
                ptr::null()
            } else {
                code.as_ptr()
            },
            code.len(),
        )
    }
}

pub fn run_with_null_context(
    host: &evmc_host_interface,
    revision: Revision,
    message: &evmc_message,
    code: &[u8],
) -> evmc_result {
    // SAFETY:
    // All pointer are valid since they are created from references except for `context` which is
    // allowed to be null.
    unsafe {
        run_raw(
            host,
            ptr::null_mut(),
            revision,
            message,
            if code.is_empty() {
                ptr::null()
            } else {
                code.as_ptr()
            },
            code.len(),
        )
    }
}

/// # Safety
///
/// All pointers must be valid, except for `context` which can be null if the `evmc_host_interface`
/// accepts null pointers as context.
#[allow(clippy::too_many_arguments)]
pub unsafe fn run_steppable_raw(
    host: *const evmc_host_interface,
    context: *mut ffi::c_void,
    revision: Revision,
    message: *const evmc_message,
    code: *const u8,
    code_len: usize,
    status: evmc_step_status_code,
    pc: u64,
    gas_refunds: i64,
    stack: *mut Uint256,
    stack_len: usize,
    memory: *mut u8,
    memory_len: usize,
    last_call_result_data: *mut u8,
    last_call_result_data_len: usize,
    steps: i32,
) -> evmc_step_result {
    let instance = evmc_create_steppable_evmrs();
    if instance.is_null() {
        panic!("vm instance is null")
    }
    let instance = unsafe { &mut *(instance as *mut evmc_vm_steppable) };

    let step_n = instance.step_n.unwrap();
    let destroy = instance.destroy.unwrap();

    let result = unsafe {
        step_n(
            instance,
            host,
            context,
            revision,
            message,
            code,
            code_len,
            status,
            pc,
            gas_refunds,
            stack,
            stack_len,
            memory,
            memory_len,
            last_call_result_data,
            last_call_result_data_len,
            steps,
        )
    };

    unsafe {
        destroy(instance);
    }

    result
}

#[allow(clippy::too_many_arguments)]
pub fn run_steppable<T>(
    host: &evmc_host_interface,
    context: &mut T,
    revision: Revision,
    message: &evmc_message,
    code: &[u8],
    status: evmc_step_status_code,
    pc: u64,
    gas_refunds: i64,
    stack: &mut [Uint256],
    memory: &mut [u8],
    last_call_result_data: &mut [u8],
    steps: i32,
) -> evmc_step_result {
    // SAFETY:
    // All pointer are valid since they are created from references.
    unsafe {
        run_steppable_raw(
            host,
            context as *mut T as *mut ffi::c_void,
            revision,
            message,
            if code.is_empty() {
                ptr::null()
            } else {
                code.as_ptr()
            },
            code.len(),
            status,
            pc,
            gas_refunds,
            if stack.is_empty() {
                ptr::null_mut()
            } else {
                stack.as_mut_ptr()
            },
            stack.len(),
            if memory.is_empty() {
                ptr::null_mut()
            } else {
                memory.as_mut_ptr()
            },
            memory.len(),
            if last_call_result_data.is_empty() {
                ptr::null_mut()
            } else {
                last_call_result_data.as_mut_ptr()
            },
            last_call_result_data.len(),
            steps,
        )
    }
}

#[allow(clippy::too_many_arguments)]
pub fn run_steppable_with_null_context(
    host: &evmc_host_interface,
    revision: Revision,
    message: &evmc_message,
    code: &[u8],
    status: evmc_step_status_code,
    pc: u64,
    gas_refunds: i64,
    stack: &mut [Uint256],
    memory: &mut [u8],
    last_call_result_data: &mut [u8],
    steps: i32,
) -> evmc_step_result {
    // SAFETY:
    // All pointer are valid since they are created from references except for `context` which is
    // allowed to be null.
    unsafe {
        run_steppable_raw(
            host,
            ptr::null_mut(),
            revision,
            message,
            if code.is_empty() {
                ptr::null()
            } else {
                code.as_ptr()
            },
            code.len(),
            status,
            pc,
            gas_refunds,
            if stack.is_empty() {
                ptr::null_mut()
            } else {
                stack.as_mut_ptr()
            },
            stack.len(),
            if memory.is_empty() {
                ptr::null_mut()
            } else {
                memory.as_mut_ptr()
            },
            memory.len(),
            if last_call_result_data.is_empty() {
                ptr::null_mut()
            } else {
                last_call_result_data.as_mut_ptr()
            },
            last_call_result_data.len(),
            steps,
        )
    }
}
