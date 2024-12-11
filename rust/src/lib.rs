#![allow(unused_crate_dependencies)]
mod evmc;
mod ffi;
mod interpreter;
mod types;
mod utils;

#[cfg(all(
    feature = "needs-cache",
    not(feature = "code-analysis-cache"),
    not(feature = "hash-cache"),
))]
compile_error!(
    "Feature `needs-cache` is only a helper feature and not supposed to be enabled on its own.
    Either disable it or enable one or all of `code-analysis-cache` or `hash-cache`."
);

#[cfg(all(
    feature = "needs-fn-ptr-conversion",
    not(feature = "fn-ptr-conversion-expanded-dispatch"),
    not(feature = "fn-ptr-conversion-inline-dispatch"),
))]
compile_error!(
    "Feature `needs-fn-ptr-conversion` is only a helper feature and not supposed to be enabled on its own.
    Either disable it or enable one or all of `fn-ptr-conversion-expanded-dispatch` or `fn-ptr-conversion-inline-dispatch`."
);

#[cfg(all(
    feature = "needs-jumptable",
    not(feature = "jumptable-dispatch"),
    not(feature = "fn-ptr-conversion-expanded-dispatch"),
    not(feature = "fn-ptr-conversion-inline-dispatch"),
))]
compile_error!(
    "Feature `needs-jumptable` is only a helper feature and not supposed to be enabled on its own.
    Either disable it or enable one or all of `jumptable-dispatch`, `fn-ptr-conversion-expanded-dispatch` or `fn-ptr-conversion-inline-dispatch`."
);

#[cfg(not(feature = "custom-evmc"))]
pub extern crate evmc_vm_tosca as evmc_vm;
#[cfg(feature = "custom-evmc")]
pub extern crate evmc_vm_tosca_refactor as evmc_vm;

use llvm_profile_wrappers::{
    llvm_profile_enabled, llvm_profile_reset_counters, llvm_profile_set_filename,
    llvm_profile_write_file,
};
use types::u256;
pub use types::ExecutionContextTrait;
#[cfg(feature = "mock")]
pub use types::MockExecutionContextTrait;

/// Dump coverage data when compiled with `RUSTFLAGS="-C instrument-coverage"`.
/// Otherwise this is a no-op.
#[no_mangle]
pub extern "C" fn evmrs_dump_coverage(filename: Option<&std::ffi::c_char>) {
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
