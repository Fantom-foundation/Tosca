#![allow(unused_crate_dependencies)]
mod evmc;
mod ffi;
mod interpreter;
mod types;
mod utils;

#[cfg(feature = "mock")]
pub use types::MockExecutionContextTrait;
pub use types::{u256, ExecutionContextTrait, MockExecutionMessage, Opcode};

#[cfg(feature = "dump-cov")]
extern "C" {
    fn __llvm_profile_write_file() -> i32;
}

/// Dump coverage data when feature `dump-cov` is enabled, no-op otherwise.
/// # Safety
/// When feature `dump-cov` is enabled, this library must be compiled with `RUSTFLAGS="-C
/// instrument-coverage"`. However failing to do so is also safe because linking will simply fail.
#[no_mangle]
pub unsafe extern "C" fn evmrs_dump_coverage() {
    #[cfg(feature = "dump-cov")]
    __llvm_profile_write_file();
}

#[no_mangle]
pub extern "C" fn evmrs_is_coverage_enabled() -> u8 {
    cfg!(feature = "dump-cov") as u8
}

#[cfg(feature = "mimalloc")]
use mimalloc::MiMalloc;

#[cfg(feature = "mimalloc")]
#[global_allocator]
static GLOBAL: MiMalloc = MiMalloc;
