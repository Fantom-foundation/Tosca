use evmc_vm::ffi::evmc_revision;

#[evmc_declare::evmc_declare_vm("evmrs", "ewasm, evm", "0.1.0")]
pub struct EvmRs;

impl evmc_vm::EvmcVm for EvmRs {
    fn init() -> Self {
        EvmRs {}
    }

    fn execute(
        &self,
        revision: evmc_revision,
        code: &[u8],
        message: &evmc_vm::ExecutionMessage,
        context: Option<&mut evmc_vm::ExecutionContext>,
    ) -> evmc_vm::ExecutionResult {
        evmc_vm::ExecutionResult::success(1337, 0, None)
    }

    fn set_option(&mut self, _: &str, _: &str) -> Result<(), evmc_vm::SetOptionError> {
        Ok(())
    }
}

#[no_mangle]
extern "C" fn evmc_create_steppable_evmrs() -> *const ::evmc_vm::ffi::evmc_vm_steppable {
    let new_instance = ::evmc_vm::ffi::evmc_vm_steppable {
        vm: evmc_create_evmrs() as *mut ::evmc_vm::ffi::evmc_vm,
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

#[no_mangle]
extern "C" fn __evmc_step_n(
    vm: *mut evmc_vm::ffi::evmc_vm_steppable,
    host: *const evmc_vm::ffi::evmc_host_interface,
    context: *mut std::ffi::c_void,
    revision: evmc_revision,
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
    evmc_vm::ffi::evmc_step_result {
        step_status_code: evmc_vm::ffi::evmc_step_status_code::EVMC_STEP_FAILED,
        status_code: evmc_vm::ffi::evmc_status_code::EVMC_FAILURE,
        revision: evmc_revision::EVMC_CANCUN,
        pc: 0,
        gas_left: 0,
        gas_refund: 0,
        output_data: std::ptr::null(),
        output_size: 0,
        stack: std::ptr::null(),
        stack_size: 0,
        memory: std::ptr::null(),
        memory_size: 0,
        last_call_return_data: std::ptr::null(),
        last_call_return_data_size: 0,
        release: None,
    }
}
