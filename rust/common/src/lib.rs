#[cfg(not(feature = "custom-evmc"))]
pub extern crate evmc_vm_tosca as evmc_vm;
#[cfg(feature = "custom-evmc")]
pub extern crate evmc_vm_tosca_refactor as evmc_vm;

mod mock_execution_message;
pub mod opcode;
pub use mock_execution_message::*;
