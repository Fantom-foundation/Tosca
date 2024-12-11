use std::{
    ffi,
    ops::{Deref, DerefMut},
    ptr,
};

use common::evmc_vm::{
    ffi::{
        evmc_host_interface, evmc_message, evmc_step_status_code, evmc_tx_context,
        evmc_vm as evmc_vm_t, evmc_vm_steppable,
    },
    Address, ExecutionResult, Revision, StepResult, Uint256,
};
// This is needed in order for driver to link against evmrs.
#[allow(unused_imports, clippy::single_component_path_imports)]
use evmrs;

pub mod host_interface;

unsafe extern "C" {
    safe fn evmc_create_evmrs() -> *mut evmc_vm_t;
    safe fn evmc_create_steppable_evmrs() -> *mut evmc_vm_steppable;
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

pub extern "C" fn get_tx_context_zeroed(_context: *mut ffi::c_void) -> evmc_tx_context {
    TX_CONTEXT_ZEROED
}

pub struct Instance(&'static mut evmc_vm_t);

impl Deref for Instance {
    type Target = evmc_vm_t;
    fn deref(&self) -> &Self::Target {
        self.0
    }
}

impl DerefMut for Instance {
    fn deref_mut(&mut self) -> &mut Self::Target {
        self.0
    }
}

impl Default for Instance {
    fn default() -> Self {
        let instance = evmc_create_evmrs();
        if instance.is_null() {
            panic!("failed to construct evmrs instance")
        }
        // SAFETY:
        // `instance is not null`. `evmc_create_evmrs` must return a valid pointer to an
        // `evmc_vm_t`.
        let instance = unsafe { &mut *instance };
        Self(instance)
    }
}

impl Drop for Instance {
    fn drop(&mut self) {
        let destroy = self.0.destroy.unwrap();
        // SAFETY:
        // The supplied pointer to `evmc_vm_t` is valid because it is created from a reference;
        unsafe { destroy(self.0) };
    }
}

impl Instance {
    /// Run the interpreter (the `execute` function) with the supplied values. This function is
    /// unsafe because it takes raw pointers. It intended to be used to verify that the checks in
    /// the ffi module work as intended.
    ///
    /// # Safety
    ///
    /// All pointers must be valid, except for `context` which can be null if the
    /// `evmc_host_interface` accepts null pointers as context.
    pub unsafe fn run_raw(
        &mut self,
        host: *const evmc_host_interface,
        context: *mut ffi::c_void,
        revision: Revision,
        message: *const evmc_message,
        code: *const u8,
        code_len: usize,
    ) -> ExecutionResult {
        let execute = self.execute.unwrap();

        execute(self.0, host, context, revision, message, code, code_len).into()
    }

    /// Run the interpreter (the `execute` function) with the supplied values. This is a safe
    /// wrapper around `Instance::run_raw` which takes references and therefore does not allow null
    /// pointers to be passed.
    pub fn run<T>(
        &mut self,
        host: &evmc_host_interface,
        context: &mut T,
        revision: Revision,
        message: &evmc_message,
        code: &[u8],
    ) -> ExecutionResult {
        // SAFETY:
        // All pointer are valid since they are created from references.
        unsafe {
            self.run_raw(
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

    /// Run the interpreter (the `execute` function) with the supplied values. This is a safe
    /// wrapper around `Instance::run_raw` which takes references and therefore does not allow null
    /// pointers to be passed. Unlike `Instance::run` this function uses a null pointer as context.
    pub fn run_with_null_context(
        &mut self,
        host: &evmc_host_interface,
        revision: Revision,
        message: &evmc_message,
        code: &[u8],
    ) -> ExecutionResult {
        // SAFETY:
        // All pointer are valid since they are created from references except for `context` which
        // is allowed to be null.
        unsafe {
            self.run_raw(
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
}

pub struct SteppableInstance(&'static mut evmc_vm_steppable);

impl Deref for SteppableInstance {
    type Target = evmc_vm_steppable;
    fn deref(&self) -> &Self::Target {
        self.0
    }
}

impl DerefMut for SteppableInstance {
    fn deref_mut(&mut self) -> &mut Self::Target {
        self.0
    }
}

impl Default for SteppableInstance {
    fn default() -> Self {
        let instance = evmc_create_steppable_evmrs();
        if instance.is_null() {
            panic!("vm instance is null")
        }
        // SAFETY:
        // `instance is not null`. `evmc_create_steppable_evmrs` must return a valid pointer to an
        // `evmc_vm_steppable`.
        let instance = unsafe { &mut *instance };
        Self(instance)
    }
}

impl Drop for SteppableInstance {
    fn drop(&mut self) {
        let destroy = self.0.destroy.unwrap();
        // SAFETY:
        // The supplied pointer to `evmc_vm_steppable` is valid because it is created from a
        // reference;
        unsafe { destroy(self.0) };
    }
}

impl SteppableInstance {
    /// Run the interpreter (the `step_n` function) with the supplied values. This function is
    /// unsafe because it takes raw pointers. It intended to be used to verify that the checks in
    /// the ffi module work as intended.
    ///
    /// # Safety
    ///
    /// All pointers must be valid, except for `context` which can be null if the
    /// `evmc_host_interface` accepts null pointers as context.
    #[allow(clippy::too_many_arguments)]
    pub unsafe fn run_raw(
        &mut self,
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
    ) -> StepResult {
        let step_n = self.step_n.unwrap();

        step_n(
            self.0,
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
        .into()
    }

    /// Run the interpreter (the `step_n` function) with the supplied values. This is a safe
    /// wrapper around `SteppableInstance::run_raw` which takes references and therefore
    /// does not allow null pointers to be passed.
    #[allow(clippy::too_many_arguments)]
    pub fn run<T>(
        &mut self,
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
    ) -> StepResult {
        // SAFETY:
        // All pointer are valid since they are created from references.
        unsafe {
            self.run_raw(
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

    /// Run the interpreter (the `step_n` function) with the supplied values. This is a safe
    /// wrapper around `SteppableInstance::run_raw` which takes references and therefore
    /// does not allow null pointers to be passed. Unlike `SteppableInstance::run` this function
    /// uses a null pointer as context.
    #[allow(clippy::too_many_arguments)]
    pub fn run_with_null_context(
        &mut self,
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
    ) -> StepResult {
        // SAFETY:
        // All pointer are valid since they are created from references except for `context` which
        // is allowed to be null.
        unsafe {
            self.run_raw(
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
}
