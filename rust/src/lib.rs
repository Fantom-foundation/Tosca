#[evmc_declare::evmc_declare_vm("This is an example VM name", "ewasm, evm", "1.2.3-custom")]
pub struct ExampleVM;

impl evmc_vm::EvmcVm for ExampleVM {
    fn init() -> Self {
        ExampleVM {}
    }

    fn execute(
        &self,
        revision: evmc_vm::ffi::evmc_revision,
        code: &[u8],
        message: &evmc_vm::ExecutionMessage,
        context: Option<&mut evmc_vm::ExecutionContext>,
    ) -> evmc_vm::ExecutionResult {
        evmc_vm::ExecutionResult::success(1337, 0, None)
    }
}
