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
extern "C" fn evmc_create_steppable_evmrs() -> *const ::evmc_vm::ffi::evmc_vm {
    panic!("steppable not implemented");
    unsafe { 0 as *const ::evmc_vm::ffi::evmc_vm }
}
