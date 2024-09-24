#![allow(unused_crate_dependencies)]
mod evmc;
mod ffi;
mod interpreter;
mod types;
mod utils;

#[cfg(feature = "mock")]
pub use types::MockExecutionContextTrait;
pub use types::{u256, ExecutionContextTrait, MockExecutionMessage, Opcode};
