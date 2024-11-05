#![allow(unused_crate_dependencies)]
mod evmc;
mod ffi;
mod interpreter;
mod types;
mod utils;

#[cfg(not(feature = "custom-evmc"))]
pub extern crate evmc_vm_tosca as evmc_vm;
#[cfg(feature = "custom-evmc")]
pub extern crate evmc_vm_tosca_refactor as evmc_vm;

use llvm_profile_wrappers::{
    llvm_profile_enabled, llvm_profile_reset_counters, llvm_profile_set_filename,
    llvm_profile_write_file,
};
#[cfg(feature = "mock")]
pub use types::MockExecutionContextTrait;
pub use types::{u256, ExecutionContextTrait, MockExecutionMessage, Opcode};

/// Dump coverage data when compiled with `RUSTFLAGS="-C instrument-coverage"`.
/// Otherwise this is a no-op.
/// # Safety
/// The provided filename can be a C string to set a new name or null to reset to the default
/// behavior.
#[no_mangle]
pub unsafe extern "C" fn evmrs_dump_coverage(filename: *const std::ffi::c_char) {
    if llvm_profile_enabled() != 0 {
        llvm_profile_set_filename(filename);
        llvm_profile_write_file();
        llvm_profile_reset_counters();
    }
}

#[no_mangle]
pub extern "C" fn evmrs_is_coverage_enabled() -> u8 {
    llvm_profile_enabled()
}

#[cfg(feature = "mimalloc")]
use mimalloc::MiMalloc;

#[cfg(feature = "mimalloc")]
#[global_allocator]
static GLOBAL: MiMalloc = MiMalloc;
